package consumeroffset

import (
	"github.com/newrelic/infra-integrations-sdk/log"
)

type (
	consumerGroupID string
	topic           string
)

type cGroupAggregations struct {
	consumerGroupRollup       map[consumerGroupID]int
	consumerGroupMaxLagRollup map[consumerGroupID]int
	cGroupActiveClientsRollup map[clientID]struct{}

	topicRollup              map[topic]int
	topicMaxLagRollup        map[topic]int
	topicActiveClientsRollup map[topic]map[clientID]struct{}
}

type CGroupMetricsAggregator struct {
	cgroupMetrics              cGroupAggregations
	consumerGroup              string
	consumerGroupOffsetByTopic bool
}

func NewCGroupMetricsAggregator(consumerGroup string, consumerGroupOffsetByTopic bool) *CGroupMetricsAggregator {
	return &CGroupMetricsAggregator{
		consumerGroup:              consumerGroup,
		consumerGroupOffsetByTopic: consumerGroupOffsetByTopic,
		cgroupMetrics: cGroupAggregations{
			consumerGroupRollup:       make(map[consumerGroupID]int),
			consumerGroupMaxLagRollup: make(map[consumerGroupID]int),
			cGroupActiveClientsRollup: make(map[clientID]struct{}),
			topicRollup:               make(map[topic]int),
			topicMaxLagRollup:         make(map[topic]int),
			topicActiveClientsRollup:  make(map[topic]map[clientID]struct{}),
		},
	}
}

// getConsumerClientRollup waits for data from clientPartitionLagChan and aggregates it, returning it when the waitGroup finishes
func (cma *CGroupMetricsAggregator) waitAndAggregateMetrics(cGroupPartitionLagChan chan partitionLagResult) {
	log.Debug("Calculating consumer group lag rollup metrics for consumer group '%s'", cma.consumerGroup)
	defer log.Debug("Finished calculating consumer group lag rollup metrics for consumer group '%s'", cma.consumerGroup)

	for {
		result, ok := <-cGroupPartitionLagChan
		if !ok {
			break // channel has been closed
		}

		cGroupID := consumerGroupID(result.ConsumerGroup)
		topicName := topic(result.Topic)
		cID := clientID(result.ClientID)

		// Add lag to the total lag for the consumer group
		cma.cgroupMetrics.consumerGroupRollup[cGroupID] += result.Lag

		// Calculate the max lag for the consumer group
		if result.Lag > cma.cgroupMetrics.consumerGroupMaxLagRollup[cGroupID] {
			cma.cgroupMetrics.consumerGroupMaxLagRollup[cGroupID] = result.Lag
		}

		// Add ClientID to number of active consumers
		if cID != "" {
			cma.cgroupMetrics.cGroupActiveClientsRollup[cID] = struct{}{}
		}

		// Add aggregation by topic
		if cma.consumerGroupOffsetByTopic {
			cma.cgroupMetrics.topicRollup[topicName] += result.Lag

			// Calculate the max lag for the consumer group in this topic
			if result.Lag > cma.cgroupMetrics.topicMaxLagRollup[topicName] {
				cma.cgroupMetrics.topicMaxLagRollup[topicName] = result.Lag
			}

			// Add ClientID to number of active consumers for this topic
			if cID != "" {
				if _, ok := cma.cgroupMetrics.topicActiveClientsRollup[topicName]; !ok {
					cma.cgroupMetrics.topicActiveClientsRollup[topicName] = map[clientID]struct{}{}
				}
				cma.cgroupMetrics.topicActiveClientsRollup[topicName][cID] = struct{}{}
			}
		}
	}
}

func (cma *CGroupMetricsAggregator) getAggregatedMetrics() cGroupAggregations {
	return cma.cgroupMetrics
}
