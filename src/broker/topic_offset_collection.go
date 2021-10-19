package broker

import (
	"fmt"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/jmx"

	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/jmxwrapper"
	"github.com/newrelic/nri-kafka/src/metrics"
)

func gatherTopicOffset(topicSampleLookup map[string]*metric.Set) (errors []error) {
	for topicName, sample := range topicSampleLookup {
		beanModifier := metrics.ApplyTopicName(topicName)

		beanName := beanModifier(metrics.TopicOffsetMetricDef.MBean)
		results, err := jmxwrapper.JMXQuery(beanName, args.GlobalArgs.Timeout)
		if err != nil && err == jmx.ErrConnection {
			return []error{fmt.Errorf("opening JMX connection: %w", err)}
		} else if err != nil {
			errors = append(errors, fmt.Errorf("querying %q for topic %q: %w", beanName, topicName, err))
			continue
		} else if len(results) == 0 {
			continue
		}

		topicOffset, err := aggregateTopicOffset(results)
		if err != nil {
			errors = append(errors, fmt.Errorf("aggregating offset for topic %q: %w", topicName, err))
			continue
		}

		if err := sample.SetMetric("topic.offset", topicOffset, metric.GAUGE); err != nil {
			errors = append(errors, fmt.Errorf("registering topic.offset metric: %w", err))
		}
	}

	return
}

func aggregateTopicOffset(jmxResult map[string]interface{}) (offset float64, err error) {
	for key, value := range jmxResult {
		partitionOffset, ok := value.(float64)
		if !ok {
			offset = float64(-1)
			err = fmt.Errorf("unable to cast bean '%s' value '%v' as float64", key, value)
			return
		}

		offset += partitionOffset
	}

	return
}
