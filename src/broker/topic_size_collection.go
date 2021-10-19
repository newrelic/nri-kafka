package broker

import (
	"fmt"

	"github.com/newrelic/infra-integrations-sdk/data/metric"

	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/jmxwrapper"
	"github.com/newrelic/nri-kafka/src/metrics"
)

// gatherTopicSizes Obtains the topic sizes for all topics using the currently open JMX connection
func gatherTopicSizes(topicSampleLookup map[string]*metric.Set) (errors []error) {
	for topicName, sample := range topicSampleLookup {
		beanModifier := metrics.ApplyTopicName(topicName)

		beanName := beanModifier(metrics.TopicSizeMetricDef.MBean)
		results, err := jmxwrapper.JMXQuery(beanName, args.GlobalArgs.Timeout)
		if err != nil {
			errors = append(errors, fmt.Errorf("running query %q: %w", beanName, err))
			continue
		} else if len(results) == 0 {
			continue
		}

		topicSize, err := aggregateTopicSize(results)
		if err != nil {
			errors = append(errors, fmt.Errorf("aggregating topic size: %w", err))
			continue
		}

		if err := sample.SetMetric("topic.diskSize", topicSize, metric.GAUGE); err != nil {
			errors = append(errors, fmt.Errorf("registering topic.diskSize metric: %w", err))
		}
	}

	return
}

func aggregateTopicSize(jmxResult map[string]interface{}) (size float64, err error) {
	for key, value := range jmxResult {
		partitionSize, ok := value.(float64)
		if !ok {
			size = float64(-1)
			err = fmt.Errorf("unable to cast bean '%s' value '%v' as float64", key, value)
			return
		}

		size += partitionSize
	}

	return
}
