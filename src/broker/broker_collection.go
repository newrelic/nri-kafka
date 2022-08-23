// Package broker handles collection of Broker inventory and metric data
package broker

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Shopify/sarama"

	"github.com/newrelic/infra-integrations-sdk/data/attribute"
	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/connection"
	"github.com/newrelic/nri-kafka/src/metrics"
)

// StartBrokerPool starts a pool of brokerWorkers to handle collecting data for Broker entities.
// The returned channel can be fed brokerIDs to collect, and is to be closed by the user
// (or closed by feedBrokerPool)
func StartBrokerPool(
	poolSize int,
	wg *sync.WaitGroup,
	integration *integration.Integration,
	collectedTopics []string,
	jmxConnProvider connection.JMXProvider,
) chan *connection.Broker {
	brokerChan := make(chan *connection.Broker)

	// Only spin off brokerWorkers if signaled
	for i := 0; i < poolSize; i++ {
		wg.Add(1)
		go brokerWorker(brokerChan, collectedTopics, wg, integration, jmxConnProvider)
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
func brokerWorker(brokerChan <-chan *connection.Broker, collectedTopics []string, wg *sync.WaitGroup, i *integration.Integration, jmxConnProvider connection.JMXProvider) {
	defer wg.Done()

	for {
		broker, ok := <-brokerChan
		if !ok {
			return
		}

		log.Debug("Starting collection for broker id %v", broker.ID)
		if args.GlobalArgs.HasInventory() {
			populateBrokerInventory(broker, i)
		}

		if args.GlobalArgs.HasMetrics() {
			jmxConfig := connection.NewConfigBuilder().
				FromArgs().
				WithHostname(broker.Host).WithPort(broker.JMXPort).
				WithUsername(broker.JMXUser).WithPassword(broker.JMXPassword).
				Build()

			jmxConn, err := jmxConnProvider.NewConnection(jmxConfig)
			if err != nil {
				log.Error("Failed to collect broker metrics for broker: '%s', error: %v", broker.Host, err)
				continue
			}

			collectBrokerMetrics(broker, collectedTopics, i, jmxConn)

			if err := jmxConn.Close(); err != nil {
				log.Error("Unable to close JMX connection for broker: '%s', error: %v", broker.Host, err)
			}
		}
	}
}

// For a given broker struct, populate the inventory of its entity with the information gathered
func populateBrokerInventory(b *connection.Broker, integration *integration.Integration) {
	// Populate connection information
	entity, err := b.Entity(integration)
	if err != nil {
		log.Error("Failed to get entity for broker %s: %s", b.Addr(), err)
		return
	}

	if err := entity.SetInventoryItem("broker.hostname", "value", b.Host); err != nil {
		log.Error("Unable to set Hostinventory item for broker %d: %s", b.ID, err)
	}
	if err := entity.SetInventoryItem("broker.jmxPort", "value", b.JMXPort); err != nil {
		log.Error("Unable to set JMX Port inventory item for broker %d: %s", b.ID, err)
	}
	hostPort := strings.Split(b.Addr(), ":")
	if len(hostPort) == 2 {
		if err := entity.SetInventoryItem("broker.kafkaPort", "value", hostPort[1]); err != nil {
			log.Error("Unable to set Kafka Port inventory item for broker %d: %s", b.ID, err)
		}
	} else {
		log.Error("Failed to parse port from address. Skipping setting port inventory item")
	}

	// Populate configuration information
	brokerConfigs, err := getBrokerConfig(b)
	if err != nil {
		log.Error("Failed to get broker configs: %s", err)
		return
	}

	for _, config := range brokerConfigs {
		if err := entity.SetInventoryItem("broker."+config.Name, "value", config.Value); err != nil {
			log.Error("Unable to set inventory item for broker %d: %s", b.ID, err)
		}
	}
}

func collectBrokerMetrics(b *connection.Broker, collectedTopics []string, i *integration.Integration, conn connection.JMXConnection) {
	// Collect broker metrics
	populateBrokerMetrics(b, i, conn)

	// Gather Broker specific Topic metrics
	topicSampleLookup := collectBrokerTopicMetrics(b, collectedTopics, i, conn)

	// If enabled collect topic sizes
	if args.GlobalArgs.CollectTopicSize {
		gatherTopicSizes(b, topicSampleLookup, i, conn)
	}

	// If enabled collect topic offset
	if args.GlobalArgs.CollectTopicOffset {
		gatherTopicOffset(b, topicSampleLookup, i, conn)
	}
}

// For a given broker struct, collect and populate its entity with broker metrics
func populateBrokerMetrics(b *connection.Broker, i *integration.Integration, conn connection.JMXConnection) {
	// Create a metric set on the broker entity
	entity, err := b.Entity(i)
	if err != nil {
		log.Error("Failed to get entity for broker: %s", err)
		return
	}
	sample := entity.NewMetricSet("KafkaBrokerSample",
		attribute.Attribute{Key: "displayName", Value: entity.Metadata.Name},
		attribute.Attribute{Key: "entityName", Value: "broker:" + entity.Metadata.Name},
		attribute.Attribute{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
	)

	// Populate metrics set with broker metrics
	metrics.GetBrokerMetrics(sample, conn)
}

// collectBrokerTopicMetrics gathers Broker specific Topic metrics.
// Returns a map of Topic names to the corresponding entity *metric.Set
func collectBrokerTopicMetrics(b *connection.Broker, collectedTopics []string, i *integration.Integration, conn connection.JMXConnection) map[string]*metric.Set {
	topicSampleLookup := make(map[string]*metric.Set)
	entity, err := b.Entity(i)
	if err != nil {
		log.Error("Failed to create entity for broker: %s", err)
		return nil
	}

	for _, topicName := range collectedTopics {
		sample := entity.NewMetricSet("KafkaBrokerSample",
			attribute.Attribute{Key: "displayName", Value: entity.Metadata.Name},
			attribute.Attribute{Key: "entityName", Value: "broker:" + entity.Metadata.Name},
			attribute.Attribute{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
			attribute.Attribute{Key: "topic", Value: topicName},
		)

		// Insert into map
		topicSampleLookup[topicName] = sample

		metrics.CollectMetricDefinitions(sample, metrics.BrokerTopicMetricDefs, metrics.ApplyTopicName(topicName), conn)
	}

	return topicSampleLookup
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
		return nil, fmt.Errorf("got an unexpected number (%d) of config resources back", len(configResponse.Resources))
	}

	return configResponse.Resources[0].Configs, nil
}
