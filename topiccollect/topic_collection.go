// Package topiccollect handles collection of Topic inventory and metric data
package topiccollect

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Shopify/sarama"
	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	bc "github.com/newrelic/nri-kafka/brokercollect"
	"github.com/newrelic/nri-kafka/logger"
	"github.com/newrelic/nri-kafka/utils"
	"github.com/newrelic/nri-kafka/zookeeper"
)

// Topic is a storage struct for information about topics
type Topic struct {
	Entity            *integration.Entity
	Name              string
	PartitionCount    int
	ReplicationFactor int
	Configs           map[string]string
	Partitions        []*partition
}

// StartTopicPool Starts a pool of topicWorkers to handle collecting data for Topic entities.
// The channel returned is to be closed by the user.
func StartTopicPool(poolSize int, wg *sync.WaitGroup, zkConn zookeeper.Connection) chan *Topic {
	topicChan := make(chan *Topic)

	if utils.KafkaArgs.CollectBrokerTopicData {
		for i := 0; i < poolSize; i++ {
			go topicWorker(topicChan, wg, zkConn)
		}
	}

	return topicChan
}

// GetTopics retrieves the list of topics to collect based on the user-provided configuration
func GetTopics(zkConn zookeeper.Connection) ([]string, error) {
	switch utils.KafkaArgs.TopicMode {
	case "None":
		return []string{}, nil
	case "Specific":
		return utils.KafkaArgs.TopicList, nil
	case "All":
		// If they want all topics, ask Zookeeper for the list of topics
		collectedTopics, _, err := zkConn.Children("/brokers/topics")
		if err != nil {
			logger.Errorf("Unable to get list of topics from Zookeeper with error: %s", err)
			return nil, err
		}
		return collectedTopics, nil
	default:
		return nil, fmt.Errorf("Bad topic_mode '%s'", utils.KafkaArgs.TopicMode)
	}
}

// FeedTopicPool sends Topic structs down the topicChan for workers to collect and build Topic structs
func FeedTopicPool(topicChan chan<- *Topic, integration *integration.Integration, collectedTopics []string) {
	defer close(topicChan)

	if utils.KafkaArgs.CollectBrokerTopicData {
		for _, topicName := range collectedTopics {
			// create topic entity
			topicEntity, err := integration.Entity(topicName, "topic")
			if err != nil {
				logger.Errorf("Unable to create an entity for topic %s", topicName)
			}

			topicChan <- &Topic{
				Name:   topicName,
				Entity: topicEntity,
			}
		}
	}
}

// Collect inventory and metrics for topics sent down topicChan
func topicWorker(topicChan <-chan *Topic, wg *sync.WaitGroup, zkConn zookeeper.Connection) {
	wg.Add(1)
	defer wg.Done()

	for {
		topic, ok := <-topicChan
		if !ok {
			return // Stop if topicChan is closed
		}

		// Finish populating topic struct
		if err := setTopicInfo(topic, zkConn); err != nil {
			logger.Errorf("Unable to set topic data for topic %s with error: %s", topic.Name, err)
			continue
		}

		// Collect and populate inventory with topic configuration
		if utils.KafkaArgs.All() || utils.KafkaArgs.Inventory {
			errors := populateTopicInventory(topic)
			if len(errors) != 0 {
				logger.Errorf("Failed to populate inventory with %d errors", len(errors))
			}

		}

		// Collect topic metrics
		if utils.KafkaArgs.All() || utils.KafkaArgs.Metrics {
			// Create metric set for topic
			sample := topic.Entity.NewMetricSet("KafkaTopicSample",
				metric.Attribute{Key: "displayName", Value: topic.Name},
				metric.Attribute{Key: "entityName", Value: "topic:" + topic.Name},
			)

			// Collect metrics and populate metric set with them
			if err := populateTopicMetrics(topic, sample, zkConn); err != nil {
				logger.Errorf("Error collecting metrics from Topic '%s': %s", topic.Name, err.Error())
			}
		}
	}
}

// Calculate topic metrics and populate metric set with them
func populateTopicMetrics(t *Topic, sample *metric.Set, zkConn zookeeper.Connection) error {

	if err := calculateTopicRetention(t.Configs, sample); err != nil {
		return err
	}

	if err := calculateNonPreferredLeader(t.Partitions, sample); err != nil {
		return err
	}

	if err := calculateUnderReplicatedCount(t.Partitions, sample); err != nil {
		return err
	}

	responds := topicRespondsToMetadata(t, zkConn)
	return sample.SetMetric("topic.respondsToMetadataRequests", responds, metric.GAUGE)
}

func calculateTopicRetention(configs map[string]string, sample *metric.Set) error {
	var topicRetention int
	if _, ok := configs["retention.bytes"]; ok {
		topicRetention = 1
	} else {
		topicRetention = 0
	}

	return sample.SetMetric("topic.retentionBytesOrTime", topicRetention, metric.GAUGE)
}

func calculateNonPreferredLeader(partitions []*partition, sample *metric.Set) error {
	numberNonPreferredLeader := 0
	for _, p := range partitions {
		if p.Leader != p.Replicas[0] {
			numberNonPreferredLeader++
		}
	}

	return sample.SetMetric("topic.partitionsWithNonPreferredLeader", numberNonPreferredLeader, metric.GAUGE)
}

func calculateUnderReplicatedCount(partitions []*partition, sample *metric.Set) error {
	numberUnderReplicated := 0
	for _, p := range partitions {
		if len(p.InSyncReplicas) < len(p.Replicas) {
			numberUnderReplicated++
		}
	}

	return sample.SetMetric("topic.underReplicatedPartitions", numberUnderReplicated, metric.GAUGE)
}

// Makes a metadata request to determine whether a topic is able to respond
func topicRespondsToMetadata(t *Topic, zkConn zookeeper.Connection) int {

	// Get connection information for a broker
	host, _, port, err := bc.GetBrokerConnectionInfo(0, zkConn)
	if err != nil {
		return 0
	}

	// Create a broker connection object and open the connection
	broker := sarama.NewBroker(fmt.Sprintf("%s:%d", host, port))
	config := sarama.NewConfig()
	err = broker.Open(config)
	if err != nil {
		return 0
	}

	// Attempt to collect metadata and determine whether it errors out
	_, err = broker.GetMetadata(&sarama.MetadataRequest{Version: 0, Topics: []string{t.Name}, AllowAutoTopicCreation: false})
	if err != nil {
		return 0
	}

	return 1
}

// Collect and populate the remainder of the topic struct fields
func setTopicInfo(t *Topic, zkConn zookeeper.Connection) error {

	// Collect topic configuration from Zookeeper
	config, _, err := zkConn.Get("/config/topics/" + t.Name)
	if err != nil {
		return err
	}
	type topicConfigDecoder struct {
		Config map[string]string `json:"config"`
	}
	var decodedTopicConfig topicConfigDecoder
	if err = json.Unmarshal([]byte(config), &decodedTopicConfig); err != nil {
		return err
	}

	// Collect partition information asynchronously
	var wg sync.WaitGroup
	partitionInChan, partitionOutChans := startPartitionPool(50, &wg, zkConn)
	go feedPartitionPool(partitionInChan, t.Name, zkConn)
	partitions := collectPartitions(partitionOutChans)

	// Populate topic struct fields
	t.Partitions = partitions
	t.Configs = decodedTopicConfig.Config
	t.PartitionCount = len(partitions)
	if len(partitions) > 0 {
		t.ReplicationFactor = len(partitions[0].Replicas)
	}

	return nil
}

// Populate inventory with topic configuration
func populateTopicInventory(t *Topic) []error {

	// Add partition scheme to inventory
	var errors []error
	if err := t.Entity.SetInventoryItem("Partition Scheme", "Number of Partitions", t.PartitionCount); err != nil {
		errors = append(errors, err)
	}
	if err := t.Entity.SetInventoryItem("Partition Scheme", "Replication Factor", t.ReplicationFactor); err != nil {
		errors = append(errors, err)
	}

	// Add topic configs to inventory
	for key, value := range t.Configs {
		if err := t.Entity.SetInventoryItem("Config", key, value); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}
