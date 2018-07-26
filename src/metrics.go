package main

import (
	"fmt"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
)

func getBrokerMetrics(sample *metric.Set) error {
	return collectMetricDefintions(sample, brokerMetricDefs, nil)
}

func getConsumerMetrics(consumerName string, sample *metric.Set) error {
	return collectMetricDefintions(sample, consumerMetricDefs, applyConsumerName(consumerName))
}

func getProducerMetrics(producerName string, sample *metric.Set) error {
	return collectMetricDefintions(sample, producerMetricDefs, applyProducerName(producerName))
}

// Collects topic-related metrics from a producer or consumer
func collectTopicSubMetrics(entity *integration.Entity, entityType string,
	metricSets []*jmxMetricSet, topicList []string,
	beanModifier func(string, string) func(string) string) {
	for _, topicName := range topicList {
		topicSample := entity.NewMetricSet("Kafka"+entity.Metadata.Namespace+"Sample",
			metric.Attribute{Key: "displayName", Value: entity.Metadata.Name},
			metric.Attribute{Key: "entityName", Value: fmt.Sprintf("%s:%s", entity.Metadata.Namespace, entity.Metadata.Name)},
			metric.Attribute{Key: "topic", Value: topicName},
		)

		if err := collectMetricDefintions(topicSample, metricSets, beanModifier(entity.Metadata.Name, topicName)); err != nil {
			logger.Errorf("Unable to collect Topic %s metrics for entity %s: %s", topicName, entity.Metadata.Name, err.Error())
		}
	}
}

// Collect the set of metrics from the current open JMX connection and add them to the sample
func collectMetricDefintions(sample *metric.Set, metricSets []*jmxMetricSet, beanModifier func(string) string) error {
	notFoundMetrics := make([]string, 0)

	for _, metricSet := range metricSets {
		beanName := metricSet.MBean

		if beanModifier != nil {
			beanName = beanModifier(beanName)
		}

		// Return all the results under a specific mBean
		results, err := queryFunc(beanName, kafkaArgs.Timeout)
		if err != nil {
			return err
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

	return nil
}
