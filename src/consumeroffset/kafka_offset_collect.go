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
	topicOffsetGetter TopicOffsetGetter,
) {
	log.Debug("Collecting offsets for consumer group '%s'", consumerGroup)
	defer log.Debug("Finished collecting offsets for consumer group '%s'", consumerGroup)

	var (
		clientPartitionWg        sync.WaitGroup
		consumerGroupPartitionWg sync.WaitGroup
	)
	// topics to be excluded from InactiveConsumerGroupsOffset calculation
	topicExclussions := map[string]struct{}{kfkConsumerOffsetsTopic: {}, kfkSchemaTopic: {}}
	clientPartitionLagChan := make(chan partitionLagResult, 1000)
	cGroupPartitionLagChan := make(chan partitionLagResult, 1000)

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
						cGroupPartitionLagChan,
						kafkaIntegration,
					)
				}(topic, partition, description, block)
			}
		}
	}

	if args.GlobalArgs.InactiveConsumerGroupOffset {
		collectInactiveConsumerGroupOffsets(clusterAdmin, consumerGroup, topicExclussions, &consumerGroupPartitionWg, topicOffsetGetter, cGroupPartitionLagChan)
	}

	calculateClientLagTotals(clientPartitionLagChan, &clientPartitionWg, kafkaIntegration, consumerGroup)
	calculateConsumerGroupLagTotals(cGroupPartitionLagChan, &consumerGroupPartitionWg, kafkaIntegration, consumerGroup)
}

func collectClientPartitionOffsetMetrics(
	topicOffsetGetter TopicOffsetGetter,
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

func collectInactiveConsumerGroupOffsets(
	clusterAdmin sarama.ClusterAdmin,
	consumerGroup string,
	topicExclussions map[string]struct{},
	consumerGroupPartitionWg *sync.WaitGroup,
	topicOffsetGetter TopicOffsetGetter,
	cGroupPartitionLagChan chan partitionLagResult,
) {
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

					cGroupPartitionLagChan <- partitionLagResult{
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
