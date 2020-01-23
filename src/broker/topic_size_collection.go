package broker

import (
	"fmt"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/connection"
	"github.com/newrelic/nri-kafka/src/jmxwrapper"
	"github.com/newrelic/nri-kafka/src/metrics"
)

func gatherTopicSizes(b *connection.Broker, topicSampleLookup map[string]*metric.Set, i *integration.Integration) {
	entity, err := b.Entity(i)
	if err != nil {
		log.Error("Failed to get broker entity: %s", err)
		return
	}

	for topicName, sample := range topicSampleLookup {
		beanModifier := metrics.ApplyTopicName(topicName)

		beanName := beanModifier(metrics.TopicSizeMetricDef.MBean)
		results, err := jmxwrapper.JMXQuery(beanName, args.GlobalArgs.Timeout)
		if err != nil {
			log.Error("Broker '%s' failed to make JMX Query: %s", b.Host, err.Error())
			continue
		} else if len(results) == 0 {
			continue
		}

		topicSize, err := aggregateTopicSize(results)
		if err != nil {
			log.Error("Unable to calculate size for Topic %s: %s", topicName, err.Error())
			continue
		}

		if err := sample.SetMetric("topic.diskSize", topicSize, metric.GAUGE); err != nil {
			log.Error("Unable to collect topic size for Topic %s on Broker %s: %s", topicName, entity.Metadata.Name, err.Error())
		}
	}
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
