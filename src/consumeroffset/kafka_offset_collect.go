package consumeroffset

import (
	"strconv"
	"sync"

	"github.com/Shopify/sarama"
	"github.com/newrelic/infra-integrations-sdk/data/attribute"
	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-kafka/src/args"
)

const (
	kfkNoOffset             = -1
	kfkConsumerOffsetsTopic = "__consumer_offsets"
	kfkSchemaTopic          = "_schema"
)

type partitionLagResult struct {
	ClientID      string
	ConsumerGroup string
	Topic         string
	PartitionID   string
	Lag           int
}

func collectOffsetsForConsumerGroup(
	clusterAdmin sarama.ClusterAdmin,
	consumerGroup string,
	members map[string]*sarama.GroupMemberDescription,
	kafkaIntegration *integration.Integration,
	topicOffsetGetter *topicOffsetGetter,
) {
	log.Debug("Collecting offsets for consumer group '%s'", consumerGroup)
	defer log.Debug("Finished collecting offsets for consumer group '%s'", consumerGroup)

	var (
		clientPartitionWg        sync.WaitGroup
		consumerGroupPartitionWg sync.WaitGroup
	)
	// topics to be excluded from ConsumeGroupOffset calculation
	topicExclussions := map[string]struct{}{kfkConsumerOffsetsTopic: {}, kfkSchemaTopic: {}}
	clientPartitionLagChan := make(chan partitionLagResult, 1000)
	consumerGroupPartitionLagChan := make(chan partitionLagResult, 1000)

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
			// we add it to topic exclusions to not be recalculated in InactiveConsumerGroupOffset
			topicExclussions[topic] = struct{}{}
			for partition, block := range partitionMap {
				if block.Err != sarama.ErrNoError {
					log.Error("Error in consumer group offset response for topic %s, partition %d: %s", block.Err.Error())
				}
				clientPartitionWg.Add(1)
				consumerGroupPartitionWg.Add(1)
				go func(topic string, partition int32, description *sarama.GroupMemberDescription, block *sarama.OffsetFetchResponseBlock) {
					defer func() {
						clientPartitionWg.Done()
						consumerGroupPartitionWg.Done()
					}()
					collectClientPartitionOffsetMetrics(
						topicOffsetGetter,
						consumerGroup,
						description,
						topic,
						partition,
						block,
						clientPartitionLagChan,
						consumerGroupPartitionLagChan,
						kafkaIntegration,
					)
				}(topic, partition, description, block)
			}
		}
	}

	if args.GlobalArgs.InactiveConsumerGroupOffset {
		topicMap, err := clusterAdmin.ListTopics()
		if err != nil {
			log.Error("Failed to list topics for consumerGroup: %s", consumerGroup, err)
			return
		}

		for topicName, topic := range topicMap {
			if _, ok := topicExclussions[topicName]; ok {
				continue
			}

			topicPartitions := map[string][]int32{}
			for i := int32(0); i < topic.NumPartitions; i++ {
				topicPartitions[topicName] = append(topicPartitions[topicName], i)
			}

			listGroupsResponse, _ := clusterAdmin.ListConsumerGroupOffsets(consumerGroup, topicPartitions)

			for _, partitionMap := range listGroupsResponse.Blocks {
				for partition, block := range partitionMap {
					consumerGroupPartitionWg.Add(1)
					go func(partition int32, block *sarama.OffsetFetchResponseBlock, topicName string) {
						defer consumerGroupPartitionWg.Done()

						if block.Offset == kfkNoOffset {
							return
						}

						offset, err := topicOffsetGetter.getFromTopicPartition(topicName, partition)
						if err != nil {
							log.Error("Failed to get hwm for topic %s, partition %d: %s", topicName, partition, err)
							return
						}

						consumerGroupPartitionLagChan <- partitionLagResult{
							ConsumerGroup: consumerGroup,
							Topic:         topicName,
							PartitionID:   strconv.Itoa(int(partition)),
							Lag:           int(offset - block.Offset),
							ClientID:      "",
						}
					}(partition, block, topicName)
				}
			}
		}
	}

	calculateClientLagTotals(clientPartitionLagChan, &clientPartitionWg, kafkaIntegration, consumerGroup)
	calculateConsumerGroupLagTotals(consumerGroupPartitionLagChan, &consumerGroupPartitionWg, kafkaIntegration, consumerGroup)
}

func calculateClientLagTotals(
	partitionLagChan chan partitionLagResult,
	wg *sync.WaitGroup,
	kafkaIntegration *integration.Integration,
	consumerGroup string,
) {
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

		// Add lag to the total lag for the client
		consumerClientRollup[result.ClientID] = consumerClientRollup[result.ClientID] + result.Lag
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
			attribute.Attribute{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
			attribute.Attribute{Key: "clientID", Value: clientID},
			attribute.Attribute{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
		)

		err = ms.SetMetric("consumer.totalLag", totalLag, metric.GAUGE)
		if err != nil {
			log.Error("Failed to set metric consumer.totalLag: %v", err)
		}
	}
}

func calculateConsumerGroupLagTotals(partitionLagChan chan partitionLagResult, wg *sync.WaitGroup, kafkaIntegration *integration.Integration, consumerGroup string) {
	consumerGroupRollup := make(map[string]int)
	consumerGroupMaxLagRollup := make(map[string]int)
	cGroupActiveClientsRollup := make(map[string]struct{})

	topicRollup := make(map[string]int)
	topicMaxLagRollup := make(map[string]int)
	topicActiveClientsRollup := make(map[string]map[string]struct{})

	log.Debug("Calculating consumer group lag rollup metrics for consumer group '%s'", consumerGroup)
	defer log.Debug("Finished calculating consumer group lag rollup metrics for consumer group '%s'", consumerGroup)

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

		if result.ClientID != "" {
			cGroupActiveClientsRollup[result.ClientID] = struct{}{}
		}

		if args.GlobalArgs.ConsumerGroupOffsetByTopic {
			topicRollup[result.Topic] = topicRollup[result.Topic] + result.Lag

			// Calculate the max lag for the consumer group in this topic
			if result.Lag > topicMaxLagRollup[result.Topic] {
				topicMaxLagRollup[result.Topic] = result.Lag
			}

			if result.ClientID != "" {
				if _, ok := topicActiveClientsRollup[result.Topic]; !ok {
					topicActiveClientsRollup[result.Topic] = map[string]struct{}{}
				}

				topicActiveClientsRollup[result.Topic][result.ClientID] = struct{}{}
			}
		}
	}

	for consumerGroup, totalLag := range consumerGroupRollup {
		clusterIDAttr := integration.NewIDAttribute("clusterName", args.GlobalArgs.ClusterName)

		consumerGroupEntity, err := kafkaIntegration.Entity(consumerGroup, "ka-consumer-group", clusterIDAttr)
		if err != nil {
			log.Error("Failed to get entity for consumer group: %s", err)
			continue
		}

		ms := consumerGroupEntity.NewMetricSet("KafkaOffsetSample",
			attribute.Attribute{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
			attribute.Attribute{Key: "consumerGroup", Value: consumerGroup},
			attribute.Attribute{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
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

		err = ms.SetMetric("consumerGroup.activeConsumers", len(cGroupActiveClientsRollup), metric.GAUGE)
		if err != nil {
			log.Error("Failed to set metric consumerGroup.activeConsumers: %s", err)
		}
	}

	for topic, totalLag := range topicRollup {
		clusterIDAttr := integration.NewIDAttribute("clusterName", args.GlobalArgs.ClusterName)
		consumerGroupIDAttr := integration.NewIDAttribute("consumerGroup", consumerGroup)
		topicIDAttr := integration.NewIDAttribute("topic", topic)

		partitionConsumerEntity, err := kafkaIntegration.Entity(topic, "ka-consumer-group-topic", clusterIDAttr, consumerGroupIDAttr, topicIDAttr)
		if err != nil {
			log.Error("Failed to get entity for partition consumer: %s", err)
			return
		}

		ms := partitionConsumerEntity.NewMetricSet("KafkaOffsetSample",
			attribute.Attribute{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
			attribute.Attribute{Key: "consumerGroup", Value: consumerGroup},
			attribute.Attribute{Key: "topic", Value: topic},
		)

		err = ms.SetMetric("consumerGroup.totalLag", totalLag, metric.GAUGE)
		if err != nil {
			log.Error("Failed to set metric consumerGroup.totalLag: %s for topic: %s", err, topic)
		}

		maxLag := topicMaxLagRollup[topic]
		err = ms.SetMetric("consumerGroup.maxLag", maxLag, metric.GAUGE)
		if err != nil {
			log.Error("Failed to set metric consumerGroup.maxLag: %s for topic: %s", err, topic)
		}

		err = ms.SetMetric("consumerGroup.activeConsumers", len(topicActiveClientsRollup[topic]), metric.GAUGE)
		if err != nil {
			log.Error("Failed to set metric consumerGroup.activeConsumers: %s for topic: %s", err, topic)
		}
	}
}

func collectClientPartitionOffsetMetrics(
	topicOffsetGetter *topicOffsetGetter,
	consumerGroup string,
	memberDescription *sarama.GroupMemberDescription,
	topic string,
	partition int32,
	block *sarama.OffsetFetchResponseBlock,
	clientPartitionLagChan chan partitionLagResult,
	consumerPartitionLagChan chan partitionLagResult,
	kafkaIntegration *integration.Integration,
) {
	log.Debug("Collecting offsets for consumerGroup '%s', member '%s', topic '%s', partition '%d'", consumerGroup, memberDescription.ClientId, topic, partition)
	defer log.Debug("Finished collecting offsets for consumerGroup '%s', member '%s', topic '%s', partition '%d'", consumerGroup, memberDescription.ClientId, topic, partition)

	// high watermark last partition offset
	hwm, err := topicOffsetGetter.getFromTopicPartition(topic, partition)
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
		attribute.Attribute{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
		attribute.Attribute{Key: "consumerGroup", Value: consumerGroup},
		attribute.Attribute{Key: "topic", Value: topic},
		attribute.Attribute{Key: "partition", Value: strconv.Itoa(int(partition))},
		attribute.Attribute{Key: "clientID", Value: memberDescription.ClientId},
		attribute.Attribute{Key: "clientHost", Value: memberDescription.ClientHost},
	)

	if block.Offset == kfkNoOffset {
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

		clientPartitionLagChan <- partitionLagResult{
			ConsumerGroup: consumerGroup,
			Topic:         topic,
			PartitionID:   strconv.Itoa(int(partition)),
			ClientID:      memberDescription.ClientId,
			Lag:           int(lag),
		}

		consumerPartitionLagChan <- partitionLagResult{
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
