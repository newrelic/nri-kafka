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
	kfkNoOffset                = -1
	partitionLagChannelsBuffer = 1000
	kfkConsumerOffsetsTopic    = "__consumer_offsets"
	kfkSchemaTopic             = "_schema"
)

type partitionLagResult struct {
	ClientID      string
	ConsumerGroup string
	Topic         string
	PartitionID   string
	Lag           int
}

type MetricsAggregator interface {
	WaitAndAggregateMetrics(partitionLagChan chan partitionLagResult)
}

func collectOffsetsForConsumerGroup(
	cGroupTopicLister ConsumerGroupTopicLister,
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
		lagResultsWg             sync.WaitGroup
	)
	// topics to be excluded from InactiveConsumerGroupsOffset calculation
	topicExclusions := map[string]struct{}{kfkConsumerOffsetsTopic: {}, kfkSchemaTopic: {}}
	clientPartitionLagChan := make(chan partitionLagResult, partitionLagChannelsBuffer)
	cGroupPartitionLagChan := make(chan partitionLagResult, partitionLagChannelsBuffer)

	for memberName, description := range members {
		assignment, err := description.GetMemberAssignment()
		if err != nil {
			log.Error("Failed to get group member assignment for member %s: %s", memberName, err)
			continue
		}
		log.Debug("Retrieved assignment for consumer group '%s' member '%s': %#v", consumerGroup, memberName, assignment)

		listGroupsResponse, err := cGroupTopicLister.ListConsumerGroupOffsets(consumerGroup, assignment.Topics)
		if err != nil {
			log.Error("Failed to get consumer group offsets for member %s: %s", memberName, err)
			continue
		}

		for topic, partitionMap := range listGroupsResponse.Blocks {
			// we add it to topic exclusions to not be recalculated in InactiveConsumerGroupOffset
			topicExclusions[topic] = struct{}{}
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
					var partitionLag partitionLagResult
					collectClientPartitionOffsetMetrics(
						&partitionLag,
						topicOffsetGetter,
						consumerGroup,
						description,
						topic,
						partition,
						block,
						kafkaIntegration,
					)
					if partitionLag.Topic == topic {
						clientPartitionLagChan <- partitionLag
						cGroupPartitionLagChan <- partitionLag
					}
				}(topic, partition, description, block)
			}
		}
	}

	if args.GlobalArgs.InactiveConsumerGroupOffset {
		collectInactiveConsumerGroupOffsets(cGroupTopicLister, consumerGroup, topicExclusions, &consumerGroupPartitionWg, topicOffsetGetter, cGroupPartitionLagChan)
	}

	clientMetricsAggregator := NewClientMetricsAggregator(consumerGroup)
	lagResultsWg.Add(1)
	go aggregateLagData(&clientPartitionWg, consumerGroup, clientPartitionLagChan, clientMetricsAggregator, &lagResultsWg)

	cGroupMetricsAggregator := NewCGroupMetricsAggregator(consumerGroup, args.GlobalArgs.ConsumerGroupOffsetByTopic)
	lagResultsWg.Add(1)
	go aggregateLagData(&consumerGroupPartitionWg, consumerGroup, cGroupPartitionLagChan, cGroupMetricsAggregator, &lagResultsWg)

	lagResultsWg.Wait()

	generateConsumerNRMetrics(kafkaIntegration, clientMetricsAggregator.GetAggregatedMetrics())
	generateConsumerGroupNRMetrics(kafkaIntegration, cGroupMetricsAggregator.GetAggregatedMetrics(), consumerGroup)
}

func aggregateLagData(
	partitionWg *sync.WaitGroup,
	consumerGroup string,
	partitionLagChan chan partitionLagResult,
	metricsAggregator MetricsAggregator,
	lagResultsWg *sync.WaitGroup,
) {
	go func() {
		partitionWg.Wait()
		log.Debug("Finished retrieving offsets for all member partitions in consumer group '%s'", consumerGroup)
		close(partitionLagChan)
	}()
	metricsAggregator.WaitAndAggregateMetrics(partitionLagChan)
	lagResultsWg.Done()
}

func collectClientPartitionOffsetMetrics(
	partitionLagResult *partitionLagResult,
	topicOffsetGetter TopicOffsetGetter,
	consumerGroup string,
	memberDescription *sarama.GroupMemberDescription,
	topic string,
	partition int32,
	block *sarama.OffsetFetchResponseBlock,
	kafkaIntegration *integration.Integration,
) {
	log.Debug("Collecting offsets for consumerGroup '%s', member '%s', topic '%s', partition '%d'", consumerGroup, memberDescription.ClientId, topic, partition)
	defer log.Debug("Finished collecting offsets for consumerGroup '%s', member '%s', topic '%s', partition '%d'", consumerGroup, memberDescription.ClientId, topic, partition)

	// high watermark last partition offset
	hwm, err := topicOffsetGetter.GetFromTopicPartition(topic, partition)
	if err != nil {
		log.Error("Failed to get hwm for topic %s, partition %d: %s", topic, partition, err)
		return
	}

	lag := hwm - block.Offset

	clusterIDAttr := integration.NewIDAttribute("clusterName", args.GlobalArgs.ClusterName)
	consumerGroupIDAttr := integration.NewIDAttribute("consumerGroup", consumerGroup)
	topicIDAttr := integration.NewIDAttribute("topic", topic)
	partitionIDAttr := integration.NewIDAttribute("partition", strconv.Itoa(int(partition)))

	partitionConsumerEntity, err := kafkaIntegration.Entity(strconv.Itoa(int(partition)), nrPartitionConsumerEntity, clusterIDAttr, consumerGroupIDAttr, topicIDAttr, partitionIDAttr)
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

		partitionLagResult.ConsumerGroup = consumerGroup
		partitionLagResult.Topic = topic
		partitionLagResult.PartitionID = strconv.Itoa(int(partition))
		partitionLagResult.ClientID = memberDescription.ClientId
		partitionLagResult.Lag = int(lag)
	}

	err = ms.SetMetric("consumer.hwm", hwm, metric.GAUGE)
	if err != nil {
		log.Error("Failed to set metric consumer.hwm: %s", err)
	}
}

func collectInactiveConsumerGroupOffsets(
	cGroupTopicLister ConsumerGroupTopicLister,
	consumerGroup string,
	topicExclusions map[string]struct{},
	consumerGroupPartitionWg *sync.WaitGroup,
	topicOffsetGetter TopicOffsetGetter,
	cGroupPartitionLagChan chan partitionLagResult,
) {
	topicMap, err := cGroupTopicLister.ListTopics()
	if err != nil {
		log.Error("Failed to list topics for consumerGroup: %s", consumerGroup, err)
		return
	}

	for topicName, topic := range topicMap {
		if _, ok := topicExclusions[topicName]; ok {
			continue
		}

		topicPartitions := map[string][]int32{}
		for i := int32(0); i < topic.NumPartitions; i++ {
			topicPartitions[topicName] = append(topicPartitions[topicName], i)
		}

		listGroupsResponse, _ := cGroupTopicLister.ListConsumerGroupOffsets(consumerGroup, topicPartitions)

		for _, partitionMap := range listGroupsResponse.Blocks {
			for partition, block := range partitionMap {
				consumerGroupPartitionWg.Add(1)
				go func(partition int32, block *sarama.OffsetFetchResponseBlock, topicName string) {
					defer consumerGroupPartitionWg.Done()

					if block.Offset == kfkNoOffset {
						return
					}

					offset, err := topicOffsetGetter.GetFromTopicPartition(topicName, partition)
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

func generateConsumerNRMetrics(kafkaIntegration *integration.Integration, consumerClientRollup map[clientID]int) {
	for clientID, totalLag := range consumerClientRollup {
		clusterIDAttr := integration.NewIDAttribute("clusterName", args.GlobalArgs.ClusterName)

		clientEntity, err := kafkaIntegration.Entity(string(clientID), nrConsumerEntity, clusterIDAttr)
		if err != nil {
			log.Error("Failed to get entity for client: %v", err)
			continue
		}

		ms := clientEntity.NewMetricSet("KafkaOffsetSample",
			attribute.Attribute{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
			attribute.Attribute{Key: "clientID", Value: string(clientID)},
			attribute.Attribute{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
		)

		err = ms.SetMetric("consumer.totalLag", totalLag, metric.GAUGE)
		if err != nil {
			log.Error("Failed to set metric consumer.totalLag: %v", err)
		}
	}
}

func generateConsumerGroupNRMetrics(kafkaIntegration *integration.Integration, cGroupAggregations CGroupAggregations, consumerGroup string) {
	consumerGroupMetrics(cGroupAggregations.consumerGroupRollup, kafkaIntegration, cGroupAggregations.consumerGroupMaxLagRollup, cGroupAggregations.cGroupActiveClientsRollup)
	consumerGroupByTopicMetrics(cGroupAggregations.topicRollup, consumerGroup, kafkaIntegration, cGroupAggregations.topicMaxLagRollup, cGroupAggregations.topicActiveClientsRollup)
}

func consumerGroupMetrics(
	consumerGroupRollup map[consumerGroupID]int,
	kafkaIntegration *integration.Integration,
	consumerGroupMaxLagRollup map[consumerGroupID]int,
	cGroupActiveClientsRollup map[clientID]struct{},
) {
	for consumerGroup, totalLag := range consumerGroupRollup {
		clusterIDAttr := integration.NewIDAttribute("clusterName", args.GlobalArgs.ClusterName)

		consumerGroupEntity, err := kafkaIntegration.Entity(string(consumerGroup), nrConsumerGroupEntity, clusterIDAttr)
		if err != nil {
			log.Error("Failed to get entity for consumer group: %s", err)
			continue
		}

		ms := consumerGroupEntity.NewMetricSet("KafkaOffsetSample",
			attribute.Attribute{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
			attribute.Attribute{Key: "consumerGroup", Value: string(consumerGroup)},
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
}

func consumerGroupByTopicMetrics(
	topicRollup map[topic]int,
	consumerGroup string,
	kafkaIntegration *integration.Integration,
	topicMaxLagRollup map[topic]int,
	topicActiveClientsRollup map[topic]map[clientID]struct{},
) {
	for topic, totalLag := range topicRollup {
		clusterIDAttr := integration.NewIDAttribute("clusterName", args.GlobalArgs.ClusterName)
		consumerGroupIDAttr := integration.NewIDAttribute("consumerGroup", consumerGroup)
		topicIDAttr := integration.NewIDAttribute("topic", string(topic))

		partitionConsumerEntity, err := kafkaIntegration.Entity(string(topic), nrConsumerGroupTopicEntity, clusterIDAttr, consumerGroupIDAttr, topicIDAttr)
		if err != nil {
			log.Error("Failed to get entity for partition consumer: %s in topic %s", err, topic)
			return
		}

		ms := partitionConsumerEntity.NewMetricSet("KafkaOffsetSample",
			attribute.Attribute{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
			attribute.Attribute{Key: "consumerGroup", Value: consumerGroup},
			attribute.Attribute{Key: "topic", Value: string(topic)},
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
