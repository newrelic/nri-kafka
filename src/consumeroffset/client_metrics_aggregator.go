package consumeroffset

import "github.com/newrelic/infra-integrations-sdk/v3/log"

type (
	clientID           string
	ClientAggregations map[clientID]int
)

type ClientMetricsAggregator struct {
	clientMetrics ClientAggregations
	consumerGroup string
}

func NewClientMetricsAggregator(consumerGroup string) *ClientMetricsAggregator {
	return &ClientMetricsAggregator{
		consumerGroup: consumerGroup,
		clientMetrics: make(map[clientID]int),
	}
}

// WaitAndAggregateMetrics waits for data from partitionLagChan and aggregates it, returning it when the channel closes
func (cma *ClientMetricsAggregator) WaitAndAggregateMetrics(partitionLagChan chan partitionLagResult) {
	log.Debug("Calculating consumer lag rollup metrics for consumer group '%s'", cma.consumerGroup)
	defer log.Debug("Finished calculating consumer lag rollup metrics for consumer group '%s'", cma.consumerGroup)

	for {
		result, ok := <-partitionLagChan
		if !ok {
			break // channel has been closed
		}

		// Add lag to the total lag for the client
		cma.clientMetrics[clientID(result.ClientID)] += result.Lag
	}
}

func (cma *ClientMetricsAggregator) GetAggregatedMetrics() ClientAggregations {
	return cma.clientMetrics
}
