package consumeroffset

import (
	"sync"

	"github.com/newrelic/infra-integrations-sdk/data/attribute"
	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-kafka/src/args"
)

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

		// Add ClientID to number of active consumers
		if result.ClientID != "" {
			cGroupActiveClientsRollup[result.ClientID] = struct{}{}
		}

		// Add aggregation by topic
		if args.GlobalArgs.ConsumerGroupOffsetByTopic {
			topicRollup[result.Topic] = topicRollup[result.Topic] + result.Lag

			// Calculate the max lag for the consumer group in this topic
			if result.Lag > topicMaxLagRollup[result.Topic] {
				topicMaxLagRollup[result.Topic] = result.Lag
			}

			// Add ClientID to number of active consumers for this topic
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
			log.Error("Failed to get entity for partition consumer: %s in topic %s", err, topic)
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
