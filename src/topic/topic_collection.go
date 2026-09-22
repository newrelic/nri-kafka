// Package topic handles collection of Topic inventory and metric data
package topic

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/IBM/sarama"
	"github.com/newrelic/infra-integrations-sdk/v3/data/attribute"
	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/connection"
)

// Topic is a storage struct for information about topics
type Topic struct {
	Entity            *integration.Entity
	Name              string
	PartitionCount    int
	ReplicationFactor int
	Configs           []*sarama.ConfigEntry
	Partitions        []*partition
	// ByteRates is nil when not pre-computed by the caller (e.g. CollectTopicConfigMetrics is
	// off, or the broker-side MBean for this topic hasn't been created yet).
	ByteRates *ByteRates
}

// ByteRates is bytes-in/bytes-out for a topic, summed across every broker's own JMX -
// see kafka.go's collectTopicByteRates. Kafka has no single MBean that reports a
// cluster-wide total for a topic; each broker only knows about the partitions it hosts.
type ByteRates struct {
	BytesInPerSecond  float64
	BytesOutPerSecond float64
}

type Getter interface {
	Topics() ([]string, error)
}

// StartTopicPool Starts a pool of topicWorkers to handle collecting data for Topic entities.
// The channel returned is to be closed by the user.
func StartTopicPool(poolSize int, wg *sync.WaitGroup, client connection.Client) chan *Topic {
	topicChan := make(chan *Topic)

	for i := 0; i < poolSize; i++ {
		wg.Add(1)
		go topicWorker(topicChan, wg, client)
	}

	return topicChan
}

// GetTopics retrieves the list of topics to collect based on the user-provided configuration
func GetTopics(topicGetter Getter) ([]string, error) {
	switch strings.ToLower(args.GlobalArgs.TopicMode) {
	case "none":
		return []string{}, nil
	case "list":
		return args.GlobalArgs.TopicList, nil
	case "regex":
		if args.GlobalArgs.TopicRegex == "" {
			return nil, errors.New("regex topic mode requires the topic_regex argument to be set")
		}

		pattern, err := regexp.Compile(args.GlobalArgs.TopicRegex)
		if err != nil {
			return nil, fmt.Errorf("failed to compile topic regex: %s", err)
		}

		allTopics, err := topicGetter.Topics()
		if err != nil {
			return nil, fmt.Errorf("failed to get topics from client: %w", err)
		}

		filteredTopics := make([]string, 0, len(allTopics))
		for _, topic := range allTopics {
			if pattern.MatchString(topic) {
				filteredTopics = append(filteredTopics, topic)
			}
		}

		return filteredTopics, nil
	case "all":
		allTopics, err := topicGetter.Topics()
		if err != nil {
			return nil, fmt.Errorf("failed to get topics from client: %w", err)
		}
		return allTopics, nil
	default:
		log.Error("Invalid topic mode %s", args.GlobalArgs.TopicMode)
		return nil, fmt.Errorf("invalid topic_mode '%s'", args.GlobalArgs.TopicMode)
	}
}

// FeedTopicPool sends Topic structs down the topicChan for workers to collect and build Topic structs.
// byteRates is looked up by topic name; topics with no entry (not pre-computed, or no traffic
// yet) simply get a nil Topic.ByteRates.
func FeedTopicPool(topicChan chan<- *Topic, i *integration.Integration, collectedTopics []string, byteRates map[string]ByteRates) {
	defer close(topicChan)

	for _, topicName := range collectedTopics {
		// clusterId is deliberately NOT an ID attribute - see connection.Broker.Entity.
		clusterIDAttr := integration.NewIDAttribute("clusterName", args.GlobalArgs.ClusterName)
		topicEntity, err := i.Entity(topicName, "ka-topic", clusterIDAttr)
		if err != nil {
			log.Error("Unable to create an entity for topic %s", topicName)
		}

		newTopic := &Topic{
			Name:   topicName,
			Entity: topicEntity,
		}
		if rates, ok := byteRates[topicName]; ok {
			newTopic.ByteRates = &rates
		}

		topicChan <- newTopic
	}
}

// Collect inventory and metrics for topics sent down topicChan
func topicWorker(topicChan <-chan *Topic, wg *sync.WaitGroup, client connection.Client) {
	defer wg.Done()

	for {
		topic, ok := <-topicChan
		if !ok {
			return // Stop if topicChan is closed
		}

		// Finish populating topic struct
		if err := setTopicInfo(topic, client); err != nil {
			log.Error("Unable to set topic data for topic %s with error: %s", topic.Name, err)
			continue
		}

		// Collect and populate inventory with topic configuration
		if args.GlobalArgs.HasInventory() {
			log.Debug("Collecting inventory for topic %q", topic.Name)
			errors := populateTopicInventory(topic)
			if len(errors) != 0 {
				log.Error("Failed to populate inventory with %d errors", len(errors))
			}
			log.Debug("Done collecting inventory for topic %q", topic.Name)
		}

		// Collect topic metrics
		if args.GlobalArgs.HasMetrics() {
			log.Debug("Collecting metrics for topic %s", topic.Name)
			// Create metric set for topic
			sample := topic.Entity.NewMetricSet("KafkaTopicSample",
				attribute.Attribute{Key: "displayName", Value: topic.Name},
				attribute.Attribute{Key: "entityName", Value: "topic:" + topic.Name},
				attribute.Attribute{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
				attribute.Attribute{Key: "clusterId", Value: args.GlobalArgs.ClusterID},
			)

			// Collect metrics and populate metric set with them
			if err := populateTopicMetrics(topic, sample, client); err != nil {
				log.Error("Error collecting metrics from Topic %q: %s", topic.Name, err.Error())
			}

			log.Debug("Done collecting metrics for topic %q", topic.Name)
		}
	}
}

// Calculate topic metrics and populate metric set with them
func populateTopicMetrics(t *Topic, sample *metric.Set, client connection.Client) error {

	if err := calculateNonPreferredLeader(t.Partitions, sample); err != nil {
		return err
	}

	if err := calculateUnderReplicatedCount(t.Partitions, sample); err != nil {
		return err
	}

	if args.GlobalArgs.CollectTopicConfigMetrics {
		if err := populateTopicConfigMetrics(t, sample); err != nil {
			return err
		}
	}

	responds := topicRespondsToMetadata(t, client)
	return sample.SetMetric("topic.respondsToMetadataRequests", responds, metric.GAUGE)
}

// populateTopicConfigMetrics adds metrics that are already available from the topic's
// existing DescribeConfigs/partition data - no extra network calls beyond what setTopicInfo
// already does.
func populateTopicConfigMetrics(t *Topic, sample *metric.Set) error {
	if err := sample.SetMetric("topic.partitionCount", t.PartitionCount, metric.GAUGE); err != nil {
		return err
	}

	if err := sample.SetMetric("topic.replicationFactor", t.ReplicationFactor, metric.GAUGE); err != nil {
		return err
	}

	if minISR, ok := minInSyncReplicas(t.Configs); ok {
		if err := sample.SetMetric("topic.minInSyncReplicas", minISR, metric.GAUGE); err != nil {
			return err
		}
	}

	if retention, ok := retentionMs(t.Configs); ok {
		if err := sample.SetMetric("topic.retentionMs", retention, metric.GAUGE); err != nil {
			return err
		}
	}

	if t.ByteRates != nil {
		if err := sample.SetMetric("topic.bytesInPerSecond", t.ByteRates.BytesInPerSecond, metric.RATE); err != nil {
			return err
		}
		if err := sample.SetMetric("topic.bytesOutPerSecond", t.ByteRates.BytesOutPerSecond, metric.RATE); err != nil {
			return err
		}
	}

	return nil
}

// minInSyncReplicas finds and parses the min.insync.replicas topic config, if set.
func minInSyncReplicas(configs []*sarama.ConfigEntry) (int, bool) {
	for _, config := range configs {
		if config.Name != "min.insync.replicas" {
			continue
		}

		value, err := strconv.Atoi(config.Value)
		if err != nil {
			log.Error("Failed to parse min.insync.replicas value %q: %s", config.Value, err)
			return 0, false
		}

		return value, true
	}

	return 0, false
}

// retentionMs finds and parses the retention.ms topic config, if set. -1 means infinite
// retention, a valid value, not an error.
func retentionMs(configs []*sarama.ConfigEntry) (int, bool) {
	for _, config := range configs {
		if config.Name != "retention.ms" {
			continue
		}

		value, err := strconv.Atoi(config.Value)
		if err != nil {
			log.Error("Failed to parse retention.ms value %q: %s", config.Value, err)
			return 0, false
		}

		return value, true
	}

	return 0, false
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
func topicRespondsToMetadata(t *Topic, client connection.Client) int {
	controller, err := client.Controller()
	if err != nil {
		log.Error("Failed to get controller from client: %s", err)
	}

	// Attempt to collect metadata and determine whether it errors out
	_, err = controller.GetMetadata(&sarama.MetadataRequest{Version: 0, Topics: []string{t.Name}, AllowAutoTopicCreation: false})
	if err != nil {
		return 0
	}

	return 1
}

// Collect and populate the remainder of the topic struct fields
func setTopicInfo(t *Topic, client connection.Client) error {
	// DescribeConfigs v0 was removed in Kafka 4.0 (the broker's minimum supported
	// version is now v1). Derive the request version from the negotiated Kafka
	// version so we remain compatible with both older brokers and Kafka 4.x.
	// sarama v1.43.x supports DescribeConfigs up to v2.
	descVer := int16(0)
	switch {
	case args.GlobalArgs.KafkaVersion.IsAtLeast(sarama.V2_0_0_0):
		descVer = 2
	case args.GlobalArgs.KafkaVersion.IsAtLeast(sarama.V1_1_0_0):
		descVer = 1
	}
	configRequest := &sarama.DescribeConfigsRequest{
		Version:         descVer,
		IncludeSynonyms: descVer >= 1,
		Resources: []*sarama.ConfigResource{
			{
				Type:        sarama.TopicResource,
				Name:        string(t.Name),
				ConfigNames: nil,
			},
		},
	}

	controller, err := client.Controller()
	if err != nil {
		return fmt.Errorf("failed to get controller from client: %s", err)
	}

	configResponse, err := controller.DescribeConfigs(configRequest)
	if err != nil {
		return fmt.Errorf("failed to describe configs for topic %s: %s", t.Name, err)
	}

	if len(configResponse.Resources) != 1 {
		return fmt.Errorf("received an unexpected number of config resources (%d)", len(configResponse.Resources))
	}

	// Collect partition information asynchronously
	var wg sync.WaitGroup
	partitionInChan, partitionOutChans := startPartitionPool(50, &wg, client)
	go feedPartitionPool(partitionInChan, t.Name, client)
	partitions := collectPartitions(partitionOutChans)

	// Populate topic struct fields
	t.Partitions = partitions
	t.Configs = configResponse.Resources[0].Configs
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
	if err := t.Entity.SetInventoryItem("topic.partitionScheme", "Number of Partitions", t.PartitionCount); err != nil {
		errors = append(errors, err)
	}
	if err := t.Entity.SetInventoryItem("topic.partitionScheme", "Replication Factor", t.ReplicationFactor); err != nil {
		errors = append(errors, err)
	}

	var topicKey, _ = t.Entity.Metadata.Key()
	if err := t.Entity.SetInventoryItem("topic.name", "value", topicKey.String()); err != nil {
		errors = append(errors, err)
	}

	// Add topic configs to inventory
	for _, config := range t.Configs {
		if err := t.Entity.SetInventoryItem("topic."+config.Name, "value", config.Value); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}
