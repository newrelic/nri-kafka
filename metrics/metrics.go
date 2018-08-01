// Package metrics contains definitions for all JMX collected Metrics, and core collection
// methods for Brokers, Consumers, and Producers.
package metrics

import (
	"fmt"
	"strings"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/nri-kafka/logger"
	"github.com/newrelic/nri-kafka/utils"
)

// GetBrokerMetrics collects all Broker JMX metrics and stores them in sample
func GetBrokerMetrics(sample *metric.Set) {
	CollectMetricDefintions(sample, brokerMetricDefs, nil)
}

// GetConsumerMetrics collects all Consumer metrics for the given
// consumerName and stores them in sample.
func GetConsumerMetrics(consumerName string, sample *metric.Set) {
	CollectMetricDefintions(sample, consumerMetricDefs, applyConsumerName(consumerName))
}

// GetProducerMetrics collects all Producer and Producer metrics for the given
// producerName and stores them in sample.
func GetProducerMetrics(producerName string, sample *metric.Set) {
	CollectMetricDefintions(sample, producerMetricDefs, applyProducerName(producerName))
}

// CollectTopicSubMetrics collects Topic metrics that are related to either a Producer or Consumer
//
// beanModifier is a function that is used to replace place holder with actual Consumer/Producer
// and Topic names for a given MBean
func CollectTopicSubMetrics(entity *integration.Entity, entityType string,
	metricSets []*JMXMetricSet, topicList []string,
	beanModifier func(string, string) func(string) string) {

	// need to title case the type so it matches the metric set of the parent entity
	titleEntityType := strings.Title(entity.Metadata.Namespace)

	for _, topicName := range topicList {
		topicSample := entity.NewMetricSet("Kafka"+titleEntityType+"Sample",
			metric.Attribute{Key: "displayName", Value: entity.Metadata.Name},
			metric.Attribute{Key: "entityName", Value: fmt.Sprintf("%s:%s", entity.Metadata.Namespace, entity.Metadata.Name)},
			metric.Attribute{Key: "topic", Value: topicName},
		)

		CollectMetricDefintions(topicSample, metricSets, beanModifier(entity.Metadata.Name, topicName))
	}
}

// CollectMetricDefintions collects the set of metrics from the current open JMX connection and add them to the sample
func CollectMetricDefintions(sample *metric.Set, metricSets []*JMXMetricSet, beanModifier func(string) string) {
	notFoundMetrics := make([]string, 0)

	for _, metricSet := range metricSets {
		beanName := metricSet.MBean

		if beanModifier != nil {
			beanName = beanModifier(beanName)
		}

		// Return all the results under a specific mBean
		results, err := utils.JMXQuery(beanName, utils.KafkaArgs.Timeout)
		// If we fail we don't want a total failure as other metrics can be collected even if a single failure/timout occurs
		if err != nil {
			logger.Errorf("Unable to execute JMX query for MBean '%s': %s", beanName, err.Error())
			continue
		}

		// For each metric to collect, populate the sample if it is
		// found in results, otherwise save the mBeanKey as not found
		for _, metricDef := range metricSet.MetricDefs {
			mBeanKey := metricSet.MetricPrefix + metricDef.JMXAttr
			if beanModifier != nil {
				mBeanKey = beanModifier(mBeanKey)
			}
			if value, ok := results[mBeanKey]; !ok {
				notFoundMetrics = append(notFoundMetrics, metricDef.Name)
			} else {
				if err := sample.SetMetric(metricDef.Name, value, metricDef.SourceType); err != nil {
					logger.Errorf("Error setting value: %s", err)
				}
			}
		}
	}

	if len(notFoundMetrics) > 0 {
		logger.Debugf("Can't find raw metrics in results for keys: %v", notFoundMetrics)
	}
}
