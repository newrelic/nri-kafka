// Package metrics contains definitions for all JMX collected Metrics, and core collection
// methods for Brokers, Consumers, and Producers.
package metrics

import (
	"fmt"
	"strings"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/jmxwrapper"
)

// GetBrokerMetrics collects all Broker JMX metrics and stores them in sample
func GetBrokerMetrics(sample *metric.Set) {
	CollectMetricDefinitions(sample, brokerMetricDefs, nil)
	CollectBrokerRequestMetrics(sample, brokerRequestMetricDefs)
}

// GetConsumerMetrics collects all Consumer metrics for the given
// consumerName and stores them in sample.
func GetConsumerMetrics(consumerName string, sample *metric.Set) {
	CollectMetricDefinitions(sample, consumerMetricDefs, applyConsumerName(consumerName))
}

// GetProducerMetrics collects all Producer and Producer metrics for the given
// producerName and stores them in sample.
func GetProducerMetrics(producerName string, sample *metric.Set) {
	CollectMetricDefinitions(sample, producerMetricDefs, applyProducerName(producerName))
}

// CollectTopicSubMetrics collects Topic metrics that are related to either a Producer or Consumer
//
// beanModifier is a function that is used to replace place holder with actual Consumer/Producer
// and Topic names for a given MBean
func CollectTopicSubMetrics(entity *integration.Entity, entityType string,
	metricSets []*JMXMetricSet, topicList []string,
	beanModifier func(string, string) BeanModifier) {

	// need to title case the type so it matches the metric set of the parent entity
	titleEntityType := strings.Title(strings.TrimPrefix(entity.Metadata.Namespace, "ka-"))

	for _, topicName := range topicList {
		topicSample := entity.NewMetricSet("Kafka"+titleEntityType+"Sample",
			metric.Attribute{Key: "displayName", Value: entity.Metadata.Name},
			metric.Attribute{Key: "entityName", Value: fmt.Sprintf("%s:%s", strings.TrimPrefix(entity.Metadata.Namespace, "ka-"), entity.Metadata.Name)},
			metric.Attribute{Key: "topic", Value: topicName},
		)

		CollectMetricDefinitions(topicSample, metricSets, beanModifier(entity.Metadata.Name, topicName))
	}
}

// CollectBrokerRequestMetrics collects request metrics from brokers
func CollectBrokerRequestMetrics(sample *metric.Set, metricSets []*JMXMetricSet) {
	notFoundMetrics := make([]string, 0)

	for _, metricSet := range metricSets {
		beanName := metricSet.MBean

		// Return all the results under a specific mBean
		results, err := jmxwrapper.JMXQuery(beanName, args.GlobalArgs.Timeout)
		// If we fail we don't want a total failure as other metrics can be collected even if a single failure/timout occurs
		if err != nil {
			log.Error("Unable to execute JMX query for MBean '%s': %s", beanName, err.Error())
			continue
		}

		// For each metric to collect, populate the sample if it is
		// found in results, otherwise save the mBeanKey as not found
		for _, metricDef := range metricSet.MetricDefs {
			versionRollup := 0.0
			found := false
			// Newer versions of Kafka have nest the request metrics under a version, so we have to roll these up
			for metric, value := range results {
				if strings.HasPrefix(metric, metricSet.MetricPrefix+metricDef.JMXAttr) && strings.HasSuffix(metric, "attr=OneMinuteRate") {
					found = true
					rate, ok := value.(float64)
					if !ok {
						log.Warn("Got non-float64 value for a rate")
						continue
					}
					versionRollup += rate
				}
			}

			if !found {
				notFoundMetrics = append(notFoundMetrics, metricDef.Name)
				continue
			}

			if err := sample.SetMetric(metricDef.Name, versionRollup, metricDef.SourceType); err != nil {
				log.Error("Error setting value: %s", err)
			}
		}
	}

	if len(notFoundMetrics) > 0 {
		log.Warn("Can't find raw metrics in results for keys: %v", notFoundMetrics)
	}
}

// CollectMetricDefinitions collects the set of metrics from the current open JMX connection and add them to the sample
func CollectMetricDefinitions(sample *metric.Set, metricSets []*JMXMetricSet, beanModifier BeanModifier) {
	notFoundMetrics := make([]string, 0)

	for _, metricSet := range metricSets {
		beanName := metricSet.MBean

		if beanModifier != nil {
			beanName = beanModifier(beanName)
		}

		// Return all the results under a specific mBean
		results, err := jmxwrapper.JMXQuery(beanName, args.GlobalArgs.Timeout)
		// If we fail we don't want a total failure as other metrics can be collected even if a single failure/timout occurs
		if err != nil {
			log.Error("Unable to execute JMX query for MBean '%s': %s", beanName, err.Error())
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
					log.Error("Error setting value: %s", err)
				}
			}
		}
	}

	if len(notFoundMetrics) > 0 {
		log.Warn("Can't find raw metrics in results for keys: %v", notFoundMetrics)
	}
}
