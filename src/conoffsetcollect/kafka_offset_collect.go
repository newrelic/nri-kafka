package conoffsetcollect

import (
	"strconv"

	"github.com/Shopify/sarama"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-kafka/src/args"
)

// caches used to prevent multiple API calls
var (
	allBrokers []*sarama.Broker
	allTopics  []string
)

func fillKafkaCaches(client sarama.Client) {
	var err error

	// collect cache for topics and brokers
	allBrokers = client.Brokers()
	allTopics, err = client.Topics()
	if err != nil {
		log.Warn("Unable to get list of topics from Kafka: %s", err.Error())
		// fill out a blank list to avoid nil checks everywhere this is used
		allTopics = make([]string, 0)
	}
}

// getKafkaConsumerOffsets collects consumer offsets from Kafka brokers rather than Zookeeper
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

	// Fill out any missing
	topicPartitions = fillOutTopicPartitionsFromKafka(client, topicPartitions)

	return getConsumerOffsetsFromBroker(groupName, topicPartitions, brokers)
}

// getConsumerOffsetsFromBroker collects a consumer groups offsets from the given brokers
func getConsumerOffsetsFromBroker(groupName string, topicPartitions args.TopicPartitions, brokers []*sarama.Broker) []*partitionOffsets {
	request := &sarama.OffsetFetchRequest{
		ConsumerGroup: groupName,
		Version:       int16(1),
	}

	offsets := make([]*partitionOffsets, 0)
	for _, broker := range brokers {
		resp, err := broker.FetchOffset(request)
		if err != nil {
			log.Debug("Error fetching offset requests for group '%s' from broker with id '%d': %s", groupName, broker.ID(), err.Error())
			continue
		}

		if len(resp.Blocks) == 0 {
			log.Debug("No offset data found for consumer gorup '%s'", groupName)
			return offsets
		}

		for topic, partitions := range topicPartitions {
			// case if partitions could not be collected from Kafka
			if partitions == nil {
				continue
			}

			for _, partition := range partitions {
				block := resp.GetBlock(topic, partition)
				if block != nil && block.Err == sarama.ErrNoError {
					offsetData := &partitionOffsets{
						Topic:          topic,
						Partition:      strconv.Itoa(int(partition)),
						ConsumerOffset: block.Offset,
					}

					offsets = append(offsets, offsetData)
				}
			}
		}
	}

	return offsets
}

// fillOutTopicPartitionsFromKafka checks all topics for the consumer group if no topics are list then all topics
// will be added for the consumer group. If a topic has no partition then all partitions of a topic will be added.
// all calls will query Kafka rather than Zookeeper
func fillOutTopicPartitionsFromKafka(client sarama.Client, topicPartitions args.TopicPartitions) args.TopicPartitions {
	if len(topicPartitions) == 0 {
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
