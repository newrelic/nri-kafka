package conoffsetcollect

import (
	"fmt"
	"strconv"

	"github.com/Shopify/sarama"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/connection"
)

// caches used to prevent multiple API calls
var (
	allBrokers []connection.Broker
	allTopics  []string
)

// fillKafkaCaches pre-retrieves the brokers and topics for quicker lookup in future requests
func fillKafkaCaches(client connection.Client) {
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

// getConsumerOffsets collects consumer offsets from Kafka brokers rather than Zookeeper
func getConsumerOffsets(groupName string, topicPartitions TopicPartitions, client connection.Client) (groupOffsets, error) {

	// refresh coordinator cache (suggested by sarama to do so)
	if err := client.RefreshCoordinator(groupName); err != nil {
		log.Debug("Unable to refresh coordinator for group '%s': %v", groupName, err)
	}

	brokers := make([]connection.Broker, 0)

	// get coordinator broker if possible, if not look through all brokers
	coordinator, err := client.Coordinator(groupName)
	if err != nil {
		log.Debug("Unable to retrieve coordinator for group '%s'", groupName)
		brokers = allBrokers
	} else {
		brokers = append(brokers, coordinator)
	}

	return getConsumerOffsetsFromBroker(groupName, topicPartitions, brokers)
}

// getConsumerOffsetsFromBroker collects a consumer groups offsets from the given brokers
func getConsumerOffsetsFromBroker(groupName string, topicPartitions TopicPartitions, brokers []connection.Broker) (groupOffsets, error) {
	offsetRequest := createOffsetFetchRequest(groupName, topicPartitions)

	offsets := make(groupOffsets)
	for _, broker := range brokers {
		err := resetBrokerConnection(broker)
		if err != nil {
			return nil, err
		}

		resp, err := broker.FetchOffset(offsetRequest)
		if err != nil {
			log.Debug("Error fetching offset requests for group '%s': %s", groupName, err.Error())
			continue
		}

		if len(resp.Blocks) == 0 {
			log.Debug("No offset data found for consumer group '%s'. There may not be any active consumers.", groupName)
			continue
		}

		for topic, partitions := range topicPartitions {
			offsets[topic] = make(topicOffsets)
			// case if partitions could not be collected from Kafka
			if partitions == nil {
				continue
			}

			for _, partition := range partitions {
				if block := resp.GetBlock(topic, partition); block != nil && block.Err == sarama.ErrNoError {
					offsets[topic][partition] = block.Offset
				} else {
					return nil, fmt.Errorf("no offset found for topic '%s', partition %v", topic, partition)
				}
			}
		}
	}

	return offsets, nil
}

func resetBrokerConnection(broker connection.Broker) error {
	if yes, _ := broker.Connected(); !yes {
		if err := broker.Open(sarama.NewConfig()); err != nil {
			return err
		}
	}

	return nil
}

type groupOffsets map[string]topicOffsets

type topicOffsets map[int32]int64

// getHighWaterMarkFromBrokers retrieves the high water mark for every partition in topicPartitions.
// To do this, it first must determine which broker is the leader for a partition because a request
// can only be made to the leader of a partition.
// Next, for each broker, it creates a fetch request that fetches a block per partition that that broker
// is the leader for. This is more complicated than it sounds, see createFetchRequest for details.
// Finally, it makes the request, and for each partition in the request, it inserts the highWaterMarkOffset
// from the partition-associated block into the hwms map to be returned.
func getHighWaterMarks(topicPartitions TopicPartitions, client connection.Client) (groupOffsets, error) {
	// Determine which broker is the leader for each partition
	brokerLeaderMap, err := getBrokerLeaderMap(topicPartitions, client)
	if err != nil {
		return nil, err
	}

	hwms := make(groupOffsets)
	for broker, tps := range brokerLeaderMap {

		resp, err := fetchHighWaterMarkResponse(broker, tps, client)
		if err != nil {
			log.Error("Failed to collect high water marks for topics %v", tps)
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
					log.Error("Failed to collect hwm for partition %v", partition)
				}
			}
		}
	}

	return hwms, nil

}

func fetchHighWaterMarkResponse(broker connection.Broker, tps TopicPartitions, client connection.Client) (*sarama.FetchResponse, error) {
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

	return resp, err

}

func getBrokerLeaderMap(topicPartitions TopicPartitions, client connection.Client) (map[connection.Broker]TopicPartitions, error) {
	brokerLeaderMap := make(map[connection.Broker]TopicPartitions)
	for topic, partitions := range topicPartitions {
		for _, partition := range partitions {
			leader, err := client.Leader(topic, partition)
			if err != nil {
				return nil, fmt.Errorf("cannot determine leader for partition %v", partition)
			}

			if _, ok := brokerLeaderMap[leader]; !ok {
				brokerLeaderMap[leader] = make(TopicPartitions)
			}
			brokerLeaderMap[leader][topic] = append(brokerLeaderMap[leader][topic], partition)
		}
	}

	return brokerLeaderMap, nil
}

// fillOutTopicPartitionsFromKafka checks all topics for the consumer group if no topics are listed then all topics
// will be added for the consumer group. If a topic has no partition then all partitions of a topic will be added.
// All calls will query Kafka rather than Zookeeper
func fillTopicPartitions(groupID string, topicPartitions TopicPartitions, client connection.Client) TopicPartitions {

	// If no topics, request the list of topics for a group
	if len(topicPartitions) == 0 {
		err := fillTopics(groupID, topicPartitions, client)
		if err != nil {
			log.Error("Unable to collect all topic partitions: %v", err)
			return nil
		}
	}

	// For each topic, if it has no partitions, collect all partitions
	for topic, partitions := range topicPartitions {
		if len(partitions) == 0 {
			var err error
			if partitions, err = client.Partitions(topic); err != nil {
				log.Warn("Unable to gather partitions for topic '%s': %s", topic, err.Error())
				continue
			}

			topicPartitions[topic] = partitions
		}
	}

	return topicPartitions
}

func fillTopics(groupID string, topicPartitions TopicPartitions, client connection.Client) error {
	groupDescribeReq := sarama.DescribeGroupsRequest{}
	groupDescribeReq.AddGroup(groupID)
	broker := client.Brokers()[0]
	config := sarama.NewConfig()
	config.Version = sarama.V0_9_0_0
	err := broker.Open(config)
	if err != nil {
		log.Error("Failed to open broker connection")
		return nil
	}

	groupDescribeResp, err := client.Brokers()[0].DescribeGroups(&groupDescribeReq)
	if err != nil {
		log.Error("Failed to collect list of topics from groups: %v", err)
		return nil
	}

	if len(groupDescribeResp.Groups) != 1 {
		log.Error("Failed to collect topics from groups")
		return nil
	}

	for _, memberDescription := range groupDescribeResp.Groups[0].Members {
		metadata, err := memberDescription.GetMemberMetadata()
		if err != nil {
			continue
		}

		for _, topic := range metadata.Topics {
			if _, ok := topicPartitions[topic]; !ok {
				topicPartitions[topic] = make([]int32, 0)
			}
		}
	}

	return nil
}

// createOffsetFetchRequest creates an offsetFetchRequest for the partitions in topicPartitions
func createOffsetFetchRequest(groupName string, topicPartitions TopicPartitions) *sarama.OffsetFetchRequest {
	request := &sarama.OffsetFetchRequest{
		ConsumerGroup: groupName,
		Version:       int16(1),
	}

	// Add partitions to request
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

func createFetchRequest(topicPartitions TopicPartitions, client connection.Client) *sarama.FetchRequest {
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
				log.Error("Failed to get offset for partition %d", partition)
			}
			request.AddBlock(topic, partition, offset, 10000)
		}
	}

	return request
}

// getAllConsumerGroupsFromKafka gets a list of all consumer groups, and populates
// it with a list of all topics belonging to those consumer groups.
func getAllConsumerGroupsFromKafka(client connection.Client) (args.ConsumerGroups, error) {

	// Create a broker with an API v9.0 connection
	broker := client.Brokers()[0]
	conf := sarama.NewConfig()
	conf.Version = sarama.V0_9_0_0
	if err := broker.Close(); err != nil {
		return nil, err
	}
	if err := broker.Open(conf); err != nil {
		return nil, err
	}

	// Get a list of all groups
	groupListResp, err := broker.ListGroups(&sarama.ListGroupsRequest{})
	if err != nil {
		return nil, err
	}

	// Add each group to the request
	groupDescribeReq := sarama.DescribeGroupsRequest{}
	for group := range groupListResp.Groups {
		groupDescribeReq.AddGroup(group)
	}
	groupDescribeResp, err := broker.DescribeGroups(&groupDescribeReq)
	if err != nil {
		return nil, err
	}

	// For each group, get its topics
	consumerGroups := make(args.ConsumerGroups)
	for _, groupDescription := range groupDescribeResp.Groups {
		consumerGroups[groupDescription.GroupId] = make(map[string][]int32)
		for _, memberDescription := range groupDescription.Members {
			metadata, err := memberDescription.GetMemberMetadata()
			if err != nil {
				continue
			}
			for _, topic := range metadata.Topics {
				consumerGroups[groupDescription.GroupId][topic] = make([]int32, 0)
			}
		}
	}

	return consumerGroups, nil

}

// populateOffsetStructs takes a map of offsets and high water marks and
// populates an array of partitionOffsets which can then be marshalled into metric
// sets
func populateOffsetStructs(offsets, hwms groupOffsets) []*partitionOffsets {

	var poffsets []*partitionOffsets
	for topic, partitions := range hwms {
		for partition, hwm := range partitions {
			offsetPointer := func() *int64 {
				topicOffsets, ok := offsets[topic]
				if !ok || len(topicOffsets) == 0 {
					log.Error("Offset not collected for topic %s, partition %d", topic, partition)
					return nil
				}

				offset, ok := topicOffsets[partition]
				if !ok || offset == -1 {
					log.Error("Offset not collected for topic %s, partition %d", topic, partition)
					return nil
				}

				return &offset
			}()

			lag := func() *int64 {
				if offsetPointer == nil {
					return nil
				}

				returnLag := hwm - *offsetPointer
				return &returnLag
			}()

			poffset := &partitionOffsets{
				Topic:          topic,
				Partition:      strconv.Itoa(int(partition)),
				ConsumerOffset: offsetPointer,
				HighWaterMark:  &hwm,
				ConsumerLag:    lag,
			}

			poffsets = append(poffsets, poffset)
		}
	}

	return poffsets

}
