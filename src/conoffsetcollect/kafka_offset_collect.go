package conoffsetcollect

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/Shopify/sarama"
	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/connection"
)

// getConsumerOffsets collects consumer offsets from Kafka brokers rather than Zookeeper
func getConsumerOffsets(groupName string, topicPartitions TopicPartitions, client connection.Client) (groupOffsets, error) {

	// refresh coordinator cache (suggested by sarama to do so)
	if err := client.RefreshCoordinator(groupName); err != nil {
		log.Debug("Unable to refresh coordinator for group '%s': %v", groupName, err)
	}

	coordinator, err := client.Coordinator(groupName)
	if err != nil {
		return nil, fmt.Errorf("unable to get the coordinator broker for group %s", groupName)
	}

	return getConsumerOffsetsFromBroker(groupName, topicPartitions, []connection.Broker{coordinator})
}

// getConsumerOffsetsFromBroker collects a consumer groups offsets from the given brokers
func getConsumerOffsetsFromBroker(groupName string, topicPartitions TopicPartitions, brokers []connection.Broker) (groupOffsets, error) {
	offsetRequest := createOffsetFetchRequest(groupName, topicPartitions)

	offsets := make(groupOffsets)
	for _, broker := range brokers {
		err := resetBrokerConnection(broker, sarama.NewConfig())
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

func resetBrokerConnection(broker connection.Broker, config *sarama.Config) error {
	if yes, _ := broker.Connected(); yes {
		if err := broker.Close(); err != nil {
			return err
		}
	}

	if err := broker.Open(config); err != nil {
		return err
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
			log.Error("Failed to collect high water marks for topics %v: %s", tps, err.Error())
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
				if block == nil {
					log.Error("Failed to collect hwm for partition %v: no blocks returned for topic %s", partition, topic)
				} else if block.Err != sarama.ErrNoError {
					log.Error("Failed to collect hwm for partition %v: %s", partition, block.Err.Error())
				} else {
					hwms[topic][partition] = block.HighWaterMarkOffset
				}
			}
		}
	}

	return hwms, nil

}

func fetchHighWaterMarkResponse(broker connection.Broker, tps TopicPartitions, client connection.Client) (*sarama.FetchResponse, error) {
	// Open the connection if necessary
	if err := resetBrokerConnection(broker, sarama.NewConfig()); err != nil {
		return nil, err
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

// fillOutTopicPartitionsFromKafka checks all topics for the consumer group.
// If a topic has no partition then all partitions of a topic will be added.
// All calls will query Kafka rather than Zookeeper
func fillTopicPartitions(groupID string, topicPartitions TopicPartitions, client connection.Client) TopicPartitions {

	// If no topics return error
	if len(topicPartitions) == 0 {
		return nil
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
				log.Error("Failed to get offset for partition %d: %s", partition, err.Error())
			}
			request.AddBlock(topic, partition, offset, 10000)
		}
	}

	return request
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
					log.Error("Offset not collected for topic %s", topic, partition)
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

func collectOffsetsForConsumerGroup(client connection.Client, clusterAdmin sarama.ClusterAdmin, consumerGroup string, members map[string]*sarama.GroupMemberDescription, kafkaIntegration *integration.Integration, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Debug("Collecting offsets for consumer group '%s'", consumerGroup)
	defer log.Debug("Finished collecting offsets for consumer group '%s'", consumerGroup)

	var partitionWg sync.WaitGroup
	partitionLagChan := make(chan partitionLagResult, 1000)
	for memberName, description := range members {
		assignment, err := description.GetMemberAssignment()
		if err != nil {
			log.Error("Failed to get group member assignment for member %s: %s", memberName, err)
			continue
		}
		log.Debug("Retrieved assignment for consumer group '%s' member '%s': %#v", consumerGroup, memberName, assignment)

		listGroupsResponse, err := clusterAdmin.ListConsumerGroupOffsets(consumerGroup, assignment.Topics)
		if err != nil {
			log.Error("Failed to get consumer group offsets for member %s: %s", memberName, err)
			continue
		}

		for topic, partitionMap := range listGroupsResponse.Blocks {
			for partition, block := range partitionMap {
				if block.Err != sarama.ErrNoError {
					log.Error("Error in consumer group offset reponse for topic %s, partition %d: %s", block.Err.Error())
				}
				partitionWg.Add(1)
				go collectPartitionOffsetMetrics(client, consumerGroup, description, topic, partition, block, partitionLagChan, &partitionWg, kafkaIntegration)
			}
		}
	}

	calculateLagTotals(partitionLagChan, &partitionWg, kafkaIntegration, consumerGroup)
}

type partitionLagResult struct {
	ConsumerGroup string
	Topic         string
	PartitionID   string
	ClientID      string
	Lag           int
}

func calculateLagTotals(partitionLagChan chan partitionLagResult, wg *sync.WaitGroup, kafkaIntegration *integration.Integration, consumerGroup string) {
	consumerGroupRollup := make(map[string]int)
	consumerGroupMaxLagRollup := make(map[string]int)
	consumerClientRollup := make(map[string]int)
	log.Debug("Calculating consumer lag rollup metrics for consumer group '%s'", consumerGroup)
	defer log.Debug("Finished calculating consumer lag rollup metrics for consumer group '%s'", consumerGroup)

	go func() {
		wg.Wait()
		log.Debug("Finished retrieving offsets for all partitions in consumer group '%s'", consumerGroup)
		close(partitionLagChan)
	}()

	for {
		result, ok := <-partitionLagChan
		if !ok {
			break // channel has been closed
		}

		// Add lag to the total lag for the consumer group
		consumerGroupRollup[result.ConsumerGroup] = consumerGroupRollup[result.ConsumerGroup] + result.Lag

		// Calculate the max lag for the consumer group
		if result.Lag > consumerGroupMaxLagRollup[result.ConsumerGroup] {
			consumerGroupMaxLagRollup[result.ConsumerGroup] = result.Lag
		}

		// Add lag to the total lag for the client
		consumerClientRollup[result.ClientID] = consumerClientRollup[result.ClientID] + result.Lag
	}

	// Submit consumer group rollup metrics
	for consumerGroup, totalLag := range consumerGroupRollup {
		clusterIDAttr := integration.NewIDAttribute("clusterName", args.GlobalArgs.ClusterName)

		consumerGroupEntity, err := kafkaIntegration.Entity(consumerGroup, "ka-consumer-group", clusterIDAttr)
		if err != nil {
			log.Error("Failed to get entity for consumer group: %s", err)
			continue
		}

		ms := consumerGroupEntity.NewMetricSet("KafkaOffsetSample",
			metric.Attribute{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
			metric.Attribute{Key: "consumerGroup", Value: consumerGroup},
		)

		err = ms.SetMetric("consumerGroup.totalLag", totalLag, metric.GAUGE)
		if err != nil {
			log.Error("Failed to set metric consumerGroup.totalLag: %s", err)
		}

		maxLag := consumerGroupMaxLagRollup[consumerGroup]
		err = ms.SetMetric("consumerGroup.maxLag", maxLag, metric.GAUGE)
		if err != nil {
			log.Error("Failed to set metric consumerGroup.maxLag: %s", err)
		}
	}

	// Submit client rollup metrics
	for clientID, totalLag := range consumerClientRollup {
		clusterIDAttr := integration.NewIDAttribute("clusterName", args.GlobalArgs.ClusterName)

		clientEntity, err := kafkaIntegration.Entity(clientID, "ka-consumer", clusterIDAttr)
		if err != nil {
			log.Error("Failed to get entity for client: %v", err)
			continue
		}

		ms := clientEntity.NewMetricSet("KafkaOffsetSample",
			metric.Attribute{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
			metric.Attribute{Key: "clientID", Value: clientID},
		)

		err = ms.SetMetric("consumer.totalLag", totalLag, metric.GAUGE)
		if err != nil {
			log.Error("Failed to set metric consumer.totalLag: %v", err)
		}
	}
}

func collectPartitionOffsetMetrics(client connection.Client, consumerGroup string, memberDescription *sarama.GroupMemberDescription, topic string, partition int32, block *sarama.OffsetFetchResponseBlock, partitionLagChan chan partitionLagResult, wg *sync.WaitGroup, kafkaIntegration *integration.Integration) {
	defer wg.Done()
	log.Debug("Collecting offsets for consumerGroup '%s', member '%s', topic '%s', partition '%d'", consumerGroup, memberDescription.ClientId, topic, partition)
	defer log.Debug("Finished collecting offsets for consumerGroup '%s', member '%s', topic '%s', partition '%d'", consumerGroup, memberDescription.ClientId, topic, partition)

	hwm, err := client.GetOffset(topic, partition, sarama.OffsetNewest)
	if err != nil {
		log.Error("Failed to get hwm for topic %s, partition %d: %s", topic, partition, err)
		return
	}

	lag := hwm - block.Offset

	clusterIDAttr := integration.NewIDAttribute("clusterName", args.GlobalArgs.ClusterName)
	consumerGroupIDAttr := integration.NewIDAttribute("consumerGroup", consumerGroup)
	topicIDAttr := integration.NewIDAttribute("topic", topic)
	partitionIDAttr := integration.NewIDAttribute("partition", strconv.Itoa(int(partition)))

	partitionConsumerEntity, err := kafkaIntegration.Entity(strconv.Itoa(int(partition)), "ka-partition-consumer", clusterIDAttr, consumerGroupIDAttr, topicIDAttr, partitionIDAttr)
	if err != nil {
		log.Error("Failed to get entity for partition consumer: %s", err)
		return
	}

	ms := partitionConsumerEntity.NewMetricSet("KafkaOffsetSample",
		metric.Attribute{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
		metric.Attribute{Key: "consumerGroup", Value: consumerGroup},
		metric.Attribute{Key: "topic", Value: topic},
		metric.Attribute{Key: "partition", Value: strconv.Itoa(int(partition))},
		metric.Attribute{Key: "clientID", Value: memberDescription.ClientId},
		metric.Attribute{Key: "clientHost", Value: memberDescription.ClientHost},
	)

	if block.Offset == -1 {
		log.Warn("Offset for topic %s, partition %d has expired (past retention period). Skipping offset and lag metrics", topic, partition)
	} else {
		err = ms.SetMetric("consumer.offset", block.Offset, metric.GAUGE)
		if err != nil {
			log.Error("Failed to set metric consumer.offset: %s", err)
		}

		err = ms.SetMetric("consumer.lag", lag, metric.GAUGE)
		if err != nil {
			log.Error("Failed to set metric consumer.lag: %s", err)
		}

		partitionLagChan <- partitionLagResult{
			ConsumerGroup: consumerGroup,
			Topic:         topic,
			PartitionID:   strconv.Itoa(int(partition)),
			ClientID:      memberDescription.ClientId,
			Lag:           int(lag),
		}
	}

	err = ms.SetMetric("consumer.hwm", hwm, metric.GAUGE)
	if err != nil {
		log.Error("Failed to set metric consumer.hwm: %s", err)
	}
}
