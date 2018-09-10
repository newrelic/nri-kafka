package conoffsetcollect

import (
	"errors"
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

	// Collect the high water marks for each partition
	hwms, err := getHighWaterMarkFromBrokers(topicPartitions, allBrokers, client)
	if err != nil {
		log.Error("Unable to collect high water marks: %s", err)
		return nil
	}
	/*
	 *hwms, err := getHighWaterMarkFromClient(topicPartitions, client)
	 *if err != nil {
	 *  log.Error("Unable to collect high water marks: %s", err)
	 *  return nil
	 *}
	 */

	return getConsumerOffsetsFromBroker(groupName, hwms, topicPartitions, brokers)
}

// getConsumerOffsetsFromBroker collects a consumer groups offsets from the given brokers
func getConsumerOffsetsFromBroker(groupName string, hwms highWaterMarks, topicPartitions args.TopicPartitions, brokers []*sarama.Broker) []*partitionOffsets {
	offsetRequest := createOffsetFetchRequest(groupName, topicPartitions)

	offsets := make([]*partitionOffsets, 0)
	for _, broker := range brokers {
		if yes, _ := broker.Connected(); !yes {
			err := broker.Open(sarama.NewConfig())
			if err != nil {
				return nil
			}
		}
		resp, err := broker.FetchOffset(offsetRequest)
		if err != nil {
			log.Debug("Error fetching offset requests for group '%s' from broker with id '%d': %s", groupName, broker.ID(), err.Error())
			continue
		}

		if len(resp.Blocks) == 0 {
			log.Debug("No offset data found for consumer group '%s'", groupName)
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
						ConsumerLag:    hwms[topic][partition] - block.Offset,
					}

					offsets = append(offsets, offsetData)
				}
			}
		}
	}

	return offsets
}

type highWaterMarks map[string]map[int32]int64

func getHighWaterMarkFromBrokers(topicPartitions args.TopicPartitions, brokers []*sarama.Broker, client sarama.Client) (highWaterMarks, error) {
	// Explanation of this function:
	// getHighWaterMarkFromBrokers retrieves the high water mark for every partition in topicPartitions.
	// To do this, it first must determine which broker is the leader for a partition because a request
	// can only be made to the leader of a partition.
	// Next, for each broker, it creates a fetch request that fetches a block per partition that that broker
	// is the leader for. This is more complicated than it sounds, see createFetchRequest for details.
	// Finally, it makes the request, and for each partition in the request, it inserts the highWaterMarkOffset
	// from the partition-associated block into the hwms map to be returned.

	// Determine which broker is the leader for each partition
	brokerLeaderMap := make(map[*sarama.Broker]args.TopicPartitions) // TODO don't use args to define this
	for topic, partitions := range topicPartitions {
		for _, partition := range partitions {
			leader, err := client.Leader(topic, partition)
			if err != nil {
				log.Error("Can't determine leader")
			}

			if _, ok := brokerLeaderMap[leader]; !ok {
				brokerLeaderMap[leader] = make(args.TopicPartitions)
			}
			brokerLeaderMap[leader][topic] = append(brokerLeaderMap[leader][topic], partition)
		}
	}

	hwms := make(highWaterMarks)
	for broker, tps := range brokerLeaderMap {

		// Open the connection if necessary
		if connected, _ := broker.Connected(); !connected {
			err := broker.Open(sarama.NewConfig())
			if err != nil {
				return nil, err
			}
		}

		// Create the fetch request for the correct partitions
		fetchRequest := createFetchRequest(tps, client)

		// Run the request
		resp, err := broker.Fetch(fetchRequest)
		if err != nil {
			log.Debug("Error fetching high water mark from broker with id '%d': %s", broker.ID(), err.Error())
			continue
		}

		// Insert the high water mark into the map
		for topic, partitions := range tps {

			if _, ok := hwms[topic]; !ok {
				hwms[topic] = make(map[int32]int64)
			}
			// case if partitions could not be collected from Kafka
			if partitions == nil {
				continue
			}

			for _, partition := range partitions {
				block := resp.GetBlock(topic, partition)
				if block != nil && block.Err == sarama.ErrNoError {
					hwms[topic][partition] = block.HighWaterMarkOffset
				} else {
					log.Error("Failed to collect hwm: %v", block.Err)
				}
			}
		}
	}

	if len(hwms) == 0 {
		return nil, errors.New("failed to fetch high water marks from any broker")
	}
	return hwms, nil

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

func createOffsetFetchRequest(groupName string, topicPartitions args.TopicPartitions) *sarama.OffsetFetchRequest {
	request := &sarama.OffsetFetchRequest{
		ConsumerGroup: groupName,
		Version:       int16(1),
	}

	// add partitions to request
	for topic, partitions := range topicPartitions {
		// case if partitions could not be collected from Kafka
		if partitions == nil {
			continue
		}

		for _, partition := range partitions {
			request.AddPartition(topic, partition)
		}
	}

	return request
}

func createFetchRequest(topicPartitions args.TopicPartitions, client sarama.Client) *sarama.FetchRequest {
	// Explanation:
	// To add a block, you have to specify an offset to start reading from. Unfortunately, unlike
	// other parts of the library, this offset cannot use the standard enums OffsetOldest or OffsetNewest.
	// Thus, in order to make a fetch request, we first have to get the oldest offset with client.GetOffset.
	// This significantly slows things down, and we will likely need to run this concurrently for decent performance

	request := &sarama.FetchRequest{
		MaxWaitTime: int32(1000),
		MinBytes:    int32(0),
		MaxBytes:    int32(10000),
		Version:     int16(0),
		Isolation:   sarama.ReadUncommitted,
	}

	for topic, partitions := range topicPartitions {
		for _, partition := range partitions {
			offset, err := client.GetOffset(topic, partition, sarama.OffsetOldest)
			if err != nil {
				log.Info("Failed to get offset for partition")
			}
			request.AddBlock(topic, partition, offset, 10000)
		}
	}

	return request
}
