// Package conoffsetcollect handles collection of consumer offsets for consumer groups
package conoffsetcollect

import (
	"fmt"
	"strconv"

	"github.com/Shopify/sarama"
	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-kafka/src/args"
	bc "github.com/newrelic/nri-kafka/src/brokercollect"
	"github.com/newrelic/nri-kafka/src/zookeeper"
)

var (
	allBrokers []*sarama.Broker
	allTopics  []string
)

type partitionOffsets struct {
	Topic          string `metric_name:"topic" source_type:"attribute"`
	Partition      string `metric_name:"partition" source_type:"attribute"`
	ConsumerOffset int64  `metric_name:"kafka.consumerOffset" source_type:"gauge"`
}

// Collect collects offset data per consumer group specified in the arguments
func Collect(zkConn zookeeper.Connection, kafkaIntegration *integration.Integration) error {
	client, err := createClient(zkConn)
	if err != nil {
		return err
	}

	defer func() {
		if err := client.Close(); err != nil {
			log.Debug("Error closing client connection: %s", err.Error())
		}
	}()

	// collect cache for topics and brokers
	allBrokers = client.Brokers()
	allTopics, err = client.Topics()
	if err != nil {
		log.Warn("Unable to get list of topics from Kafka: %s", err.Error())
		// fill out a blank list to avoid nil checks everywhere this is used
		allTopics = make([]string, 0)
	}

	for consumerGroup, topicPartitions := range args.GlobalArgs.ConsumerGroups {
		offsetData := getKafkaConsumerOffsets(client, consumerGroup, topicPartitions)
		if err := setMetrics(consumerGroup, offsetData, kafkaIntegration); err != nil {
			log.Error("Error setting metrics for consumer group '%s': %s", consumerGroup, err.Error())
		}
	}

	return nil
}

func createClient(zkConn zookeeper.Connection) (sarama.Client, error) {
	brokerIDs, err := bc.GetBrokerIDs(zkConn)
	if err != nil {
		return nil, err
	}

	brokers := make([]string, 0, len(brokerIDs))
	for _, brokerID := range brokerIDs {
		// convert to int id
		intID, err := strconv.Atoi(brokerID)
		if err != nil {
			log.Warn("Unable to parse integer broker ID from %s", brokerID)
			continue
		}

		// get broker connection info
		host, _, port, err := bc.GetBrokerConnectionInfo(intID, zkConn)
		if err != nil {
			return nil, err
		}

		brokers = append(brokers, fmt.Sprintf("%s:%d", host, port))
	}

	return sarama.NewClient(brokers, sarama.NewConfig())
}

func getKafkaConsumerOffsets(client sarama.Client, groupName string, topicPartitions args.TopicPartitions) []*partitionOffsets {
	// refresh coordinator cache (suggested by sarama to do so)
	if err := client.RefreshCoordinator(groupName); err != nil {
		log.Debug("Unable to refresh coordinator for group '%s'", groupName)
	}

	brokers := make([]*sarama.Broker, 0)

	// get coordinator broker if possible, if not look through all brokers
	coordinator, err := client.Coordinator(groupName)
	if err != nil {
		log.Debug("Unable to retrieve coordinator for group '%s'", groupName)
		brokers = allBrokers
	} else {
		brokers = append(brokers, coordinator)
	}

	// Fille out any missing
	topicPartitions = fillOutTopicPartitionsFromKafka(client, topicPartitions)

	return getConsumerOffsets(groupName, topicPartitions, brokers)
}

func getConsumerOffsets(groupName string, topicPartitions args.TopicPartitions, brokers []*sarama.Broker) []*partitionOffsets {
	request := &sarama.OffsetFetchRequest{
		ConsumerGroup: groupName,
		Version:       int16(1),
	}

	consumerOffsets := make([]*partitionOffsets, 0)
	for _, broker := range brokers {
		resp, err := broker.FetchOffset(request)
		if err != nil {
			log.Debug("Error fetching offset requests for group '%s' from broker with id '%d': %s", groupName, broker.ID(), err.Error())
			continue
		}

		for topic, partitions := range topicPartitions {
			// case if partitions could not be collected from Kafka
			if partitions == nil {
				continue
			}

			for _, partition := range partitions {
				if block := resp.GetBlock(topic, partition); block != nil && block.Err == sarama.ErrNoError {
					offsetData := &partitionOffsets{
						Topic:          topic,
						Partition:      strconv.Itoa(int(partition)),
						ConsumerOffset: block.Offset,
					}

					consumerOffsets = append(consumerOffsets, offsetData)
				}
			}
		}
	}

	return consumerOffsets
}

// fillOutTopicPartitionsFromKafka checks all topics for the consumer group if no topics are list then all topics
// will be added for the consumer group. If a topic has no partition then all partitions of a topic will be added.
// all calls will query Kafka rather than Zookeeper
func fillOutTopicPartitionsFromKafka(client sarama.Client, topicPartitions args.TopicPartitions) args.TopicPartitions {
	if len(topicPartitions) == 0 {
		topicPartitions = make(args.TopicPartitions)
		for _, topic := range allTopics {
			topicPartitions[topic] = make([]int32, 0)
		}
	}

	for topic, partitions := range topicPartitions {
		if partitions == nil || len(partitions) == 0 {
			var err error
			partitions, err = client.Partitions(topic)
			if err != nil {
				log.Warn("Unable to gather partitions for topic '%s': %s", topic, err.Error())
				continue
			}
		}

		topicPartitions[topic] = partitions
	}

	return topicPartitions
}

func setMetrics(consumerGroup string, offsetData []*partitionOffsets, kafkaIntegration *integration.Integration) error {
	groupEntity, err := kafkaIntegration.Entity(consumerGroup, "consumerGroup")
	if err != nil {
		return err
	}

	for _, offsetData := range offsetData {
		metricSet := groupEntity.NewMetricSet("ConsumerGroupOffsetSample",
			metric.Attribute{Key: "displayName", Value: groupEntity.Metadata.Name},
			metric.Attribute{Key: "entityName", Value: "consumerGroup:" + groupEntity.Metadata.Name})

		if err := metricSet.MarshalMetrics(offsetData); err != nil {
			log.Error("Error Marshaling offset metrics for consumer group '%s': %s", consumerGroup, err.Error())
			continue
		}
	}

	return nil
}
