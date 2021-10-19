// Package broker handles collection of Broker inventory and metric data
package broker

import (
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/Shopify/sarama"

	"github.com/newrelic/infra-integrations-sdk/data/attribute"
	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/jmx"
	"github.com/newrelic/infra-integrations-sdk/log"

	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/connection"
	"github.com/newrelic/nri-kafka/src/jmxwrapper"
	"github.com/newrelic/nri-kafka/src/metrics"
)

// StartBrokerPool starts a pool of brokerWorkers to handle collecting data for Broker entities.
// The returned channel can be fed brokerIDs to collect, and is to be closed by the user
// (or closed by feedBrokerPool)
func StartBrokerPool(poolSize int, wg *sync.WaitGroup, integration *integration.Integration, collectedTopics []string) chan *connection.Broker {
	brokerChan := make(chan *connection.Broker)

	// Only spin off brokerWorkers if signaled
	for i := 0; i < poolSize; i++ {
		wg.Add(1)
		go brokerWorker(brokerChan, collectedTopics, wg, integration)
	}

	return brokerChan
}

// FeedBrokerPool collects a list of brokerIDs from ZooKeeper and feeds them into a
// channel to be read by a broker worker pool.
func FeedBrokerPool(brokers []*connection.Broker, brokerChan chan<- *connection.Broker) {
	defer close(brokerChan) // close the broker channel when done feeding

	for _, broker := range brokers {
		brokerChan <- broker
	}
}

// Reads brokerIDs from a channel, creates an entity for each broker, and collects
// inventory and metrics data for that broker. Exits when it determines the channel has
// been closed
func brokerWorker(brokerChan <-chan *connection.Broker, collectedTopics []string, wg *sync.WaitGroup, i *integration.Integration) {
	defer wg.Done()

	for {
		broker, ok := <-brokerChan
		if !ok {
			return
		}

		log.Debug("Starting collection for broker id %v", broker.ID)
		if args.GlobalArgs.HasInventory() {
			if err := populateBrokerInventory(broker, i); err != nil {
				log.Error("Could not populate inventory for broker %q: %v", broker, err)
			}
		}

		if args.GlobalArgs.HasMetrics() {
			for _, err := range collectBrokerMetrics(broker, collectedTopics, i) {
				log.Error("Collecting broker %s: %s", broker, err)
			}
		}
	}
}

// For a given broker struct, populate the inventory of its entity with the information gathered
func populateBrokerInventory(b *connection.Broker, integration *integration.Integration) error {
	// Populate connection information
	entity, err := b.Entity(integration)
	if err != nil {
		return fmt.Errorf("creating entity: %w", err)
	}

	if err := entity.SetInventoryItem("broker.hostname", "value", b.Host); err != nil {
		return fmt.Errorf("seting broker.hostname inventory entry: %w", err)
	}
	if err := entity.SetInventoryItem("broker.jmxPort", "value", b.JMXPort); err != nil {
		return fmt.Errorf("seting broker.jmxPort inventory entry: %w", err)
	}
	_, port, err := net.SplitHostPort(b.Addr())
	if err != nil {
		return fmt.Errorf("parsing address %q: %w", b.Addr(), err)
	}

	if err := entity.SetInventoryItem("broker.kafkaPort", "value", port); err != nil {
		return fmt.Errorf("seting broker.kafkaPort inventory entry: %w", err)
	}

	// Populate configuration information
	brokerConfigs, err := getBrokerConfig(b)
	if err != nil {
		return fmt.Errorf("getting broker configuration: %w", err)
	}

	for _, config := range brokerConfigs {
		if err := entity.SetInventoryItem("broker."+config.Name, "value", config.Value); err != nil {
			// Log error rather than return to avoid failing early.
			log.Error("unable to set inventory item for broker %q: %s", b, err)
		}
	}

	return nil
}

func collectBrokerMetrics(b *connection.Broker, collectedTopics []string, i *integration.Integration) (errors []error) {
	// Open JMX connection
	options := make([]jmx.Option, 0)
	if args.GlobalArgs.KeyStore != "" && args.GlobalArgs.KeyStorePassword != "" && args.GlobalArgs.TrustStore != "" && args.GlobalArgs.TrustStorePassword != "" {
		ssl := jmx.WithSSL(args.GlobalArgs.KeyStore, args.GlobalArgs.KeyStorePassword, args.GlobalArgs.TrustStore, args.GlobalArgs.TrustStorePassword)
		options = append(options, ssl)
	}
	options = append(options, jmx.WithNrJmxTool(args.GlobalArgs.NrJmx))

	// Lock since we can only make a single JMX connection at a time.
	jmxwrapper.JMXLock.Lock()
	err := jmxwrapper.JMXOpen(b.Host, strconv.Itoa(b.JMXPort), args.GlobalArgs.DefaultJMXUser, args.GlobalArgs.DefaultJMXPassword, options...)
	defer func() {
		jmxwrapper.JMXClose() // Close needs to be called even on a failed open to clear out any set variables
		jmxwrapper.JMXLock.Unlock()
	}()

	if err != nil {
		return []error{fmt.Errorf("connecting to JMX broker: %w", err)}
	}

	// Collect broker metrics
	for _, err := range populateBrokerMetrics(b, i) {
		errors = append(errors, fmt.Errorf("populating broker metrics: %w", err))
	}

	// Gather Broker specific Topic metrics
	topicSampleLookup, errs := collectBrokerTopicMetrics(b, collectedTopics, i)
	for _, err := range errs {
		errors = append(errors, fmt.Errorf("populating broker-topic metrics: %w", err))
	}

	if len(topicSampleLookup) == 0 {
		// Errors above were fatal and we were not able to collect any topic, we stop here since we need that list
		// to collect sizes and offsets.
		return
	}

	// If enabled collect topic sizes
	if args.GlobalArgs.CollectTopicSize {
		// Will use the already open JMX connection
		for _, err := range gatherTopicSizes(topicSampleLookup) {
			errors = append(errors, fmt.Errorf("gathering topic sizes: %w", err))
		}
	}

	// If enabled collect topic offset
	if args.GlobalArgs.CollectTopicOffset {
		// Will use the already open JMX connection
		for _, err := range gatherTopicOffset(topicSampleLookup) {
			errors = append(errors, fmt.Errorf("gathering topic offsets: %w", err))
		}
	}

	return
}

// For a given broker struct, collect and populate its entity with broker metrics
func populateBrokerMetrics(b *connection.Broker, i *integration.Integration) []error {
	// Create a metric set on the broker entity
	entity, err := b.Entity(i)
	if err != nil {
		return []error{fmt.Errorf("getting entity for broker: %w", err)}
	}

	sample := entity.NewMetricSet("KafkaBrokerSample",
		attribute.Attribute{Key: "displayName", Value: entity.Metadata.Name},
		attribute.Attribute{Key: "entityName", Value: "broker:" + entity.Metadata.Name},
		attribute.Attribute{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
	)

	// Populate metrics set with broker metrics
	return metrics.GetBrokerMetrics(sample)
}

// collectBrokerTopicMetrics gathers Broker specific Topic metrics.
// Returns a map of Topic names to the corresponding entity *metric.Set
func collectBrokerTopicMetrics(b *connection.Broker, collectedTopics []string, i *integration.Integration) (map[string]*metric.Set, []error) {
	topicSampleLookup := make(map[string]*metric.Set)
	entity, err := b.Entity(i)
	if err != nil {
		return nil, []error{fmt.Errorf("creating entity for broker %q: %w", b, err)}
	}

	var errors []error
	for _, topicName := range collectedTopics {
		sample := entity.NewMetricSet("KafkaBrokerSample",
			attribute.Attribute{Key: "displayName", Value: entity.Metadata.Name},
			attribute.Attribute{Key: "entityName", Value: "broker:" + entity.Metadata.Name},
			attribute.Attribute{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
			attribute.Attribute{Key: "topic", Value: topicName},
		)

		// Insert into map
		topicSampleLookup[topicName] = sample

		for _, err := range metrics.CollectMetricDefinitions(sample, metrics.BrokerTopicMetricDefs, metrics.ApplyTopicName(topicName)) {
			errors = append(errors, fmt.Errorf("collecting metrics for topic %q: %w", topicName, err))
		}
	}

	return topicSampleLookup, errors
}

// Collect broker configuration from Zookeeper
func getBrokerConfig(broker *connection.Broker) ([]*sarama.ConfigEntry, error) {

	configRequest := &sarama.DescribeConfigsRequest{
		Version:         0,
		IncludeSynonyms: true,
		Resources: []*sarama.ConfigResource{
			{
				Type:        sarama.BrokerResource,
				Name:        string(broker.ID),
				ConfigNames: nil,
			},
		},
	}

	configResponse, err := broker.DescribeConfigs(configRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to describe configs: %s", err)
	}

	if len(configResponse.Resources) != 1 {
		return nil, fmt.Errorf("expected 1 config resource, got (%d)", len(configResponse.Resources))
	}

	return configResponse.Resources[0].Configs, nil
}
