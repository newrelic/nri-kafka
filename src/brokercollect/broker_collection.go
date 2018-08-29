// Package brokercollect handles collection of Broker inventory and metric data
package brokercollect

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/samuel/go-zookeeper/zk"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/jmxwrapper"
	"github.com/newrelic/nri-kafka/src/metrics"
	"github.com/newrelic/nri-kafka/src/zookeeper"
)

// broker is a storage struct for information about brokers
type broker struct {
	ID        int
	Entity    *integration.Entity
	Host      string
	JMXPort   int
	KafkaPort int
	Config    map[string]string
}

// StartBrokerPool starts a pool of brokerWorkers to handle collecting data for Broker entities.
// The returned channel can be fed brokerIDs to collect, and is to be closed by the user
// (or closed by feedBrokerPool)
func StartBrokerPool(poolSize int, wg *sync.WaitGroup, zkConn zookeeper.Connection, integration *integration.Integration, collectedTopics []string) chan int {
	brokerChan := make(chan int)

	// Only spin off brokerWorkers if signaled
	if args.GlobalArgs.CollectBrokerTopicData && zkConn != nil {
		for i := 0; i < poolSize; i++ {
			wg.Add(1)
			go brokerWorker(brokerChan, collectedTopics, wg, zkConn, integration)
		}
	}

	return brokerChan
}

// FeedBrokerPool collects a list of brokerIDs from ZooKeeper and feeds them into a
// channel to be read by a broker worker pool.
func FeedBrokerPool(zkConn zookeeper.Connection, brokerChan chan<- int) error {
	defer close(brokerChan) // close the broker channel when done feeding

	// Don't make API calls or feed down channel if we don't want to collect brokers
	if args.GlobalArgs.CollectBrokerTopicData && zkConn != nil {
		brokerIDs, _, err := zkConn.Children(zookeeper.Path("/brokers/ids"))
		if err != nil {
			return fmt.Errorf("unable to get broker ID from Zookeeper: %s", err.Error())
		}

		for _, id := range brokerIDs {
			intID, err := strconv.Atoi(id)
			if err != nil {
				log.Error("Unable to parse integer broker ID from %s", id)
				continue
			}
			brokerChan <- intID
		}
	}

	return nil
}

// Reads brokerIDs from a channel, creates an entity for each broker, and collects
// inventory and metrics data for that broker. Exits when it determines the channel has
// been closed
func brokerWorker(brokerChan <-chan int, collectedTopics []string, wg *sync.WaitGroup, zkConn zookeeper.Connection, i *integration.Integration) {
	defer wg.Done()

	for {
		// Collect broker ID from channel.
		// Exit worker if channel has been closed and no more brokerIDs can be collected.
		brokerID, ok := <-brokerChan
		if !ok {
			return
		}

		// Create Broker
		b, err := createBroker(brokerID, zkConn, i)
		if err != nil {
			continue
		}

		// Populate inventory for broker
		if args.GlobalArgs.All() || args.GlobalArgs.Inventory {
			if err := populateBrokerInventory(b); err != nil {
				continue
			}
		}

		// Populate metrics for broker
		if args.GlobalArgs.All() || args.GlobalArgs.Metrics {
			if err := collectBrokerMetrics(b, collectedTopics); err != nil {
				continue
			}
		}
	}
}

// Creates and populates a broker struct with all the information needed to
// populate inventory and metrics.
func createBroker(brokerID int, zkConn zookeeper.Connection, i *integration.Integration) (*broker, error) {

	// Collect broker connection information from ZooKeeper
	host, jmxPort, kafkaPort, err := GetBrokerConnectionInfo(brokerID, zkConn)
	if err != nil {
		log.Error("Unable to get broker JMX information for broker id %s: %s", host, err)
		return nil, err
	}

	// Create broker entity
	brokerEntity, err := i.Entity(host, "broker")
	if err != nil {
		log.Error("Unable to create entity for broker ID %d: %s", brokerID, err)
		return nil, err
	}

	// Gather broker configuration from ZooKeeper
	brokerConfig, err := getBrokerConfig(brokerID, zkConn)
	if err != nil {
		log.Error("Unable to get broker configuration information for broker id %d: %s", brokerID, err)
	}

	newBroker := &broker{
		Host:      host,
		JMXPort:   jmxPort,
		KafkaPort: kafkaPort,
		Entity:    brokerEntity,
		ID:        brokerID,
		Config:    brokerConfig,
	}

	return newBroker, nil
}

// For a given broker struct, populate the inventory of its entity with the information gathered
func populateBrokerInventory(b *broker) error {
	// Populate connection information
	if err := b.Entity.SetInventoryItem("broker.hostname", "value", b.Host); err != nil {
		log.Error("Unable to set Hostinventory item for broker %d: %s", b.ID, err)
	}
	if err := b.Entity.SetInventoryItem("broker.jmxPort", "value", b.JMXPort); err != nil {
		log.Error("Unable to set JMX Port inventory item for broker %d: %s", b.ID, err)
	}
	if err := b.Entity.SetInventoryItem("broker.kafkaPort", "value", b.KafkaPort); err != nil {
		log.Error("Unable to set Kafka Port inventory item for broker %d: %s", b.ID, err)
	}

	// Populate configuration information
	for key, value := range b.Config {
		if err := b.Entity.SetInventoryItem("broker."+key, "value", value); err != nil {
			log.Error("Unable to set inventory item for broker %d: %s", b.ID, err)
		}
	}

	return nil
}

func collectBrokerMetrics(b *broker, collectedTopics []string) error {
	// Lock since we can only make a single JMX connection at a time.
	jmxwrapper.JMXLock.Lock()

	// Open JMX connection
	if err := jmxwrapper.JMXOpen(b.Host, strconv.Itoa(b.JMXPort), args.GlobalArgs.DefaultJMXUser, args.GlobalArgs.DefaultJMXPassword); err != nil {
		log.Error("Unable to make JMX connection for Broker '%s': %s", b.Host, err.Error())
		jmxwrapper.JMXClose() // Close needs to be called even on a failed open to clear out any set variables
		jmxwrapper.JMXLock.Unlock()
		return err
	}

	// Collect broker metrics
	populateBrokerMetrics(b)

	// Gather Broker specific Topic metrics
	topicSampleLookup := collectBrokerTopicMetrics(b, collectedTopics)

	// If enabled collect topic sizes
	if args.GlobalArgs.CollectTopicSize {
		gatherTopicSizes(b, topicSampleLookup)
	}

	// Close connection and release lock so another process can make JMX Connections
	jmxwrapper.JMXClose()
	jmxwrapper.JMXLock.Unlock()
	return nil
}

// For a given broker struct, collect and populate its entity with broker metrics
func populateBrokerMetrics(b *broker) {
	// Create a metric set on the broker entity
	sample := b.Entity.NewMetricSet("KafkaBrokerSample",
		metric.Attribute{Key: "displayName", Value: b.Entity.Metadata.Name},
		metric.Attribute{Key: "entityName", Value: "broker:" + b.Entity.Metadata.Name},
	)

	// Populate metrics set with broker metrics
	log.Debug("Collecting metrics for Broker '%s'", b.Entity.Metadata.Name)
	metrics.GetBrokerMetrics(sample)
}

// collectBrokerTopicMetrics gathers Broker specific Topic metrics.
// Returns a map of Topic names to the corresponding entity *metric.Set
func collectBrokerTopicMetrics(b *broker, collectedTopics []string) map[string]*metric.Set {
	topicSampleLookup := make(map[string]*metric.Set)

	for _, topicName := range collectedTopics {
		sample := b.Entity.NewMetricSet("KafkaBrokerSample",
			metric.Attribute{Key: "displayName", Value: b.Entity.Metadata.Name},
			metric.Attribute{Key: "entityName", Value: "broker:" + b.Entity.Metadata.Name},
			metric.Attribute{Key: "topic", Value: topicName},
		)

		// Insert into map
		topicSampleLookup[topicName] = sample

		metrics.CollectMetricDefintions(sample, metrics.BrokerTopicMetricDefs, metrics.ApplyTopicName(topicName))
	}

	return topicSampleLookup
}

// GetBrokerConnectionInfo Collects Broker connection info from Zookeeper
func GetBrokerConnectionInfo(brokerID int, zkConn zookeeper.Connection) (brokerHost string, jmxPort int, brokerPort int, err error) {

	// Query Zookeeper for broker information
	rawBrokerJSON, _, err := zkConn.Get(zookeeper.Path("/brokers/ids/" + strconv.Itoa(brokerID)))
	if err != nil {
		return "", 0, 0, err
	}

	// Parse the JSON returned by Zookeeper
	type brokerJSONDecoder struct {
		Host    string `json:"host"`
		JmxPort int    `json:"jmx_port"`
		Port    int    `json:"port"`
	}
	var brokerDecoded brokerJSONDecoder
	err = json.Unmarshal(rawBrokerJSON, &brokerDecoded)
	if err != nil {
		return "", 0, 0, err
	}

	return brokerDecoded.Host, brokerDecoded.JmxPort, brokerDecoded.Port, nil
}

// Collect broker configuration from Zookeeper
func getBrokerConfig(brokerID int, zkConn zookeeper.Connection) (map[string]string, error) {

	// Query Zookeeper for broker configuration
	rawBrokerConfig, _, err := zkConn.Get(zookeeper.Path("/config/brokers/" + strconv.Itoa(brokerID)))
	if err != nil {
		if err == zk.ErrNoNode {
			return map[string]string{}, nil
		}
		return nil, err
	}

	// Parse the JSON returned by Zookeeper
	type brokerConfigDecoder struct {
		Config map[string]string `json:"config"`
	}
	var brokerConfigDecoded brokerConfigDecoder
	err = json.Unmarshal(rawBrokerConfig, &brokerConfigDecoded)
	if err != nil {
		return nil, err
	}

	return brokerConfigDecoded.Config, nil
}
