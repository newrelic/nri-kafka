// Package metrics contains definitions for all JMX collected Metrics, and core collection
// methods for Brokers, Consumers, and Producers.
package metrics

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/newrelic/infra-integrations-sdk/data/attribute"
	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/jmx"
	"github.com/newrelic/infra-integrations-sdk/log"

	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/jmxwrapper"
)

var (
	ErrMetricNotFound   = errors.New("metric not found")
	ErrInvalidTopicMode = errors.New("invalid topic mode")
	ErrTopicRegexNotSet = errors.New("topic regex is not set")
)

// GetBrokerMetrics collects all Broker JMX metrics and stores them in sample
func GetBrokerMetrics(sample *metric.Set) (errors []error) {
	for _, err := range CollectMetricDefinitions(sample, brokerMetricDefs, nil) {
		errors = append(errors, fmt.Errorf("collecting metric definitions: %w", err))
	}
	for _, err := range CollectBrokerRequestMetrics(sample, brokerRequestMetricDefs) {
		errors = append(errors, fmt.Errorf("collecting broker request metrics: %w", err))
	}

	return
}

// GetConsumerMetrics collects all Consumer metrics for the given
// consumerName and stores them in sample.
func GetConsumerMetrics(consumerName string, sample *metric.Set) (errors []error) {
	for _, err := range CollectMetricDefinitions(sample, consumerMetricDefs, applyConsumerName(consumerName)) {
		errors = append(errors, fmt.Errorf("collecting consumer metrics: %w", err))
	}

	return
}

// GetProducerMetrics collects all Producer and Producer metrics for the given
// producerName and stores them in sample.
func GetProducerMetrics(producerName string, sample *metric.Set) (errors []error) {
	for _, err := range CollectMetricDefinitions(sample, producerMetricDefs, applyProducerName(producerName)) {
		errors = append(errors, fmt.Errorf("collecting producer metrics: %w", err))
	}

	return
}

// CollectTopicSubMetrics collects Topic metrics that are related to either a Producer or Consumer
//
// beanModifier is a function that is used to replace place holder with actual Consumer/Producer
// and Topic names for a given MBean
func CollectTopicSubMetrics(
	entity *integration.Entity,
	entityType string,
	metricSets []*JMXMetricSet,
	beanModifier func(string, string) BeanModifier,
) (errors []error) {
	// need to title case the type so it matches the metric set of the parent entity
	titleEntityType := strings.Title(strings.TrimPrefix(entity.Metadata.Namespace, "ka-"))

	topicList, err := getTopicListFromJMX(entity.Metadata.Name)
	if err != nil {
		return []error{
			fmt.Errorf("getting topic list: %w", err),
		}
	}

	for _, topicName := range topicList {
		topicSample := entity.NewMetricSet("Kafka"+titleEntityType+"Sample",
			attribute.Attribute{Key: "displayName", Value: entity.Metadata.Name},
			attribute.Attribute{Key: "entityName", Value: fmt.Sprintf("%s:%s", strings.TrimPrefix(entity.Metadata.Namespace, "ka-"), entity.Metadata.Name)},
			attribute.Attribute{Key: "topic", Value: topicName},
		)

		metricDefErrs := CollectMetricDefinitions(topicSample, metricSets, beanModifier(entity.Metadata.Name, topicName))
		for _, err := range metricDefErrs {
			errors = append(errors,
				fmt.Errorf("collecting topic sub metrics for %q: %w", topicName, err),
			)
		}
	}

	return errors
}

// CollectBrokerRequestMetrics collects request metrics from brokers
func CollectBrokerRequestMetrics(sample *metric.Set, metricSets []*JMXMetricSet) (errors []error) {
	for _, metricSet := range metricSets {
		beanName := metricSet.MBean

		// Return all the results under a specific mBean
		results, err := jmxwrapper.JMXQuery(beanName, args.GlobalArgs.Timeout)
		// If we fail we don't want a total failure as other metrics can be collected even if a single failure/timout occurs
		if err != nil && err == jmx.ErrConnection {
			errors = append(errors, fmt.Errorf("connecting to jmx endpoint %s: %w", net.JoinHostPort(jmx.HostName(), jmx.Port()), err))
			return
		} else if err != nil {
			errors = append(errors, fmt.Errorf("executing JMX query %q: %w", beanName, err))
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
				errors = append(errors, fmt.Errorf("%q: %w", metricDef.Name, ErrMetricNotFound))
				continue
			}

			if err := sample.SetMetric(metricDef.Name, versionRollup, metricDef.SourceType); err != nil {
				errors = append(errors, fmt.Errorf("setting value for metric %q: %w", metricDef.Name, err))
			}
		}
	}

	return
}

// CollectMetricDefinitions collects the set of metrics from the current open JMX connection and add them to the sample
func CollectMetricDefinitions(sample *metric.Set, metricSets []*JMXMetricSet, beanModifier BeanModifier) (errors []error) {
	for _, metricSet := range metricSets {
		beanName := metricSet.MBean

		if beanModifier != nil {
			beanName = beanModifier(beanName)
		}

		// Return all the results under a specific mBean
		results, err := jmxwrapper.JMXQuery(beanName, args.GlobalArgs.Timeout)
		// If we fail we don't want a total failure as other metrics can be collected even if a single failure/timout occurs
		if err != nil && err == jmx.ErrConnection {
			errors = append(errors, fmt.Errorf("connecting to jmx endpoint %s: %w", net.JoinHostPort(jmx.HostName(), jmx.Port()), err))
			return
		} else if err != nil {
			errors = append(errors, fmt.Errorf("executing JMX query %q: %w", beanName, err))
			continue
		}

		// For each metric to collect, populate the sample if it is
		// found in results, otherwise save the mBeanKey as not found
		for _, metricDef := range metricSet.MetricDefs {
			mBeanKey := metricSet.MetricPrefix + metricDef.JMXAttr
			if beanModifier != nil {
				mBeanKey = beanModifier(mBeanKey)
			}

			value, ok := results[mBeanKey]
			if !ok {
				errors = append(errors, fmt.Errorf("%q: %w", metricDef.Name, ErrMetricNotFound))
				continue
			}

			if err := sample.SetMetric(metricDef.Name, value, metricDef.SourceType); err != nil {
				errors = append(errors, fmt.Errorf("setting for metric %q: %w", metricDef.Name, err))
			}
		}
	}

	return
}

func getTopicListFromJMX(producer string) ([]string, error) {
	switch strings.ToLower(args.GlobalArgs.TopicMode) {
	case "none":
		return []string{}, nil
	case "list":
		return args.GlobalArgs.TopicList, nil
	case "regex":
		if args.GlobalArgs.TopicRegex == "" {
			return nil, ErrTopicRegexNotSet
		}

		pattern, err := regexp.Compile(args.GlobalArgs.TopicRegex)
		if err != nil {
			return nil, fmt.Errorf("failed to compile topic regex: %w", err)
		}

		allTopics, err := getAllTopicsFromJMX(producer)
		if err != nil {
			return nil, fmt.Errorf("failed to get topics from client: %w", err)
		}

		filteredTopics := make([]string, 0, len(allTopics))
		for _, topic := range allTopics {
			if pattern.MatchString(topic) {
				filteredTopics = append(filteredTopics, topic)
			}
		}

		return filteredTopics, nil
	case "all":
		allTopics, err := getAllTopicsFromJMX(producer)
		if err != nil {
			return nil, fmt.Errorf("failed to get topics from client: %w", err)
		}
		return allTopics, nil
	default:
		return nil, fmt.Errorf("%q: %w", args.GlobalArgs.TopicMode, ErrInvalidTopicMode)
	}

}

func getAllTopicsFromJMX(producer string) ([]string, error) {
	result, err := jmxwrapper.JMXQuery(fmt.Sprintf("kafka.producer:type=producer-topic-metrics,client-id=%s,topic=*", producer), args.GlobalArgs.Timeout)
	if err != nil && err == jmx.ErrConnection {
		return nil, fmt.Errorf("connecting to jmx endpoint %s: %w", net.JoinHostPort(jmx.HostName(), jmx.Port()), err)
	} else if err != nil {
		return nil, fmt.Errorf("collecting topics from JMX: %w", err)
	}

	r := regexp.MustCompile(`topic="?([^,"]+)"?`)
	uniqueTopics := make(map[string]struct{})
	for key := range result {
		match := r.FindStringSubmatch(key)
		if match == nil {
			continue
		}

		uniqueTopics[match[1]] = struct{}{}
	}

	topics := make([]string, 0, len(uniqueTopics))
	for topic := range uniqueTopics {
		topics = append(topics, topic)
	}

	return topics, nil
}
