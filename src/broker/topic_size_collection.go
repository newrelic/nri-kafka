package broker

import (
	"fmt"

	"github.com/newrelic/nrjmx/gojmx"

	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-kafka/src/connection"
	"github.com/newrelic/nri-kafka/src/metrics"
)

func gatherTopicSizes(b *connection.Broker, topicSampleLookup map[string]*metric.Set, i *integration.Integration, conn connection.JMXConnection) {
	entity, err := b.Entity(i)
	if err != nil {
		log.Error("Failed to get broker entity: %s", err)
		return
	}

	for topicName, sample := range topicSampleLookup {
		beanModifier := metrics.ApplyTopicName(topicName)

		beanName := beanModifier(metrics.TopicSizeMetricDef.MBean)
		results, err := conn.QueryMBeanAttributes(beanName)
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

func aggregateTopicSize(jmxResult []*gojmx.AttributeResponse) (size float64, err error) {
	for _, attr := range jmxResult {
		if attr.ResponseType == gojmx.ResponseTypeErr {
			log.Warn("Unable to process attribute for query: %s status: %s, while aggregating TopicSize", attr.Name, attr.StatusMsg)
			continue
		}
		partitionSize, err := attr.GetValueAsFloat()
		if err != nil {
			size = float64(-1)
			err = fmt.Errorf("bean '%s' error getting value: %w", attr.Name, err)
			return size, err
		}

		size += partitionSize
	}

	return
}
