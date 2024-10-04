package broker

import (
	"fmt"
	"os"

	"github.com/newrelic/nrjmx/gojmx"

	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/jmx"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-kafka/src/connection"
	"github.com/newrelic/nri-kafka/src/metrics"
)

func gatherTopicOffset(b *connection.Broker, topicSampleLookup map[string]*metric.Set, i *integration.Integration, conn connection.JMXConnection) {
	entity, err := b.Entity(i)
	if err != nil {
		log.Error("Failed to get broker entity: %s", err)
		return
	}

	for topicName, sample := range topicSampleLookup {
		beanModifier := metrics.ApplyTopicName(topicName)

		beanName := beanModifier(metrics.TopicOffsetMetricDef.MBean)
		results, err := conn.QueryMBeanAttributes(beanName)
		if err != nil {
			if jmxConnErr, ok := gojmx.IsJMXConnectionError(err); ok {
				log.Error("Connection error for %s:%s : %s", jmx.HostName(), jmx.Port(), jmxConnErr)
				os.Exit(1)
			}
			log.Error("Broker '%s' failed to make JMX Query: %v", b.Host, err)
			continue
		} else if len(results) == 0 {
			continue
		}

		topicOffset, err := aggregateTopicOffset(results)
		if err != nil {
			log.Error("Unable to calculate offset for Topic %s: %s", topicName, err.Error())
			continue
		}

		if err := sample.SetMetric("topic.offset", topicOffset, metric.GAUGE); err != nil {
			log.Error("Unable to collect topic offset for Topic %s on Broker %s: %s", topicName, entity.Metadata.Name, err.Error())
		}
	}
}

func aggregateTopicOffset(jmxResult []*gojmx.AttributeResponse) (offset float64, err error) {
	for _, attr := range jmxResult {
		if attr.ResponseType == gojmx.ResponseTypeErr {
			log.Warn("Unable to process attribute for query: %s status: %s, while aggregating TopicOffset", attr.Name, attr.StatusMsg)
			continue
		}

		partitionOffset, err := attr.GetValueAsFloat()
		if err != nil {
			offset = float64(-1)
			err = fmt.Errorf("bean '%s' error getting value: %w", attr.Name, err)
			return offset, err
		}

		offset += partitionOffset
	}

	return
}
