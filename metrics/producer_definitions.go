package metrics

import (
	"strings"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
)

const producerHolder = "%PRODUCER%"

// Producer Metrics
var producerMetricDefs = []*JMXMetricSet{
	{
		MBean:        "kafka.producer:type=producer-metrics,client-id=" + producerHolder,
		MetricPrefix: "kafka.producer:type=producer-metrics,",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "producer.ageMetadataUsedInMilliseconds",
				SourceType: metric.GAUGE,
				JMXAttr:    "client-id=" + producerHolder + ",attr=metadata-age",
			},
			{
				Name:       "producer.availableBufferInBytes",
				SourceType: metric.GAUGE,
				JMXAttr:    "client-id=" + producerHolder + ",attr=buffer-available-bytes",
			},
			{
				Name:       "producer.avgBytesSentPerRequestInBytes",
				SourceType: metric.GAUGE,
				JMXAttr:    "client-id=" + producerHolder + ",attr=incoming-byte-rate",
			},
			{
				Name:       "producer.avgRecordSizeInBytes",
				SourceType: metric.GAUGE,
				JMXAttr:    "client-id=" + producerHolder + ",attr=record-size-avg",
			},
			{
				Name:       "producer.avgRecordsSentPerSecond",
				SourceType: metric.GAUGE,
				JMXAttr:    "client-id=" + producerHolder + ",attr=record-send-rate",
			},
			{
				Name:       "producer.avgRequestLatencyPerSecond",
				SourceType: metric.GAUGE,
				JMXAttr:    "client-id=" + producerHolder + ",attr=request-latency-avg",
			},
			{
				Name:       "producer.avgThrottleTime",
				SourceType: metric.GAUGE,
				JMXAttr:    "client-id=" + producerHolder + ",attr=produce-throttle-time-avg",
			},
			{
				Name:       "producer.bufferpoolWaitTime",
				SourceType: metric.GAUGE,
				JMXAttr:    "client-id=" + producerHolder + ",attr=bufferpool-wait-time-total",
			},
			{
				Name:       "producer.bytesOutPerSecond",
				SourceType: metric.GAUGE,
				JMXAttr:    "client-id=" + producerHolder + ",attr=outgoing-byte-rate",
			},
			{
				Name:       "producer.compressionRateRecordBatches",
				SourceType: metric.GAUGE,
				JMXAttr:    "client-id=" + producerHolder + ",attr=compression-rate-avg",
			},
			{
				Name:       "producer.ioWaitTime",
				SourceType: metric.GAUGE,
				JMXAttr:    "client-id=" + producerHolder + ",attr=io-wait-time-ns-avg",
			},
			{
				Name:       "producer.maxRecordSizeInBytes",
				SourceType: metric.GAUGE,
				JMXAttr:    "client-id=" + producerHolder + ",attr=record-size-max",
			},
			{
				Name:       "producer.maxRequestLatencyInMilliseconds",
				SourceType: metric.GAUGE,
				JMXAttr:    "client-id=" + producerHolder + ",attr=request-latency-max",
			},
			{
				Name:       "producer.maxThrottleTime",
				SourceType: metric.GAUGE,
				JMXAttr:    "client-id=" + producerHolder + ",attr=produce-throttle-time-max",
			},
			{
				Name:       "producer.requestPerSecond",
				SourceType: metric.GAUGE,
				JMXAttr:    "client-id=" + producerHolder + ",attr=request-rate",
			},
			{
				Name:       "producer.requestsWaitingResponse",
				SourceType: metric.GAUGE,
				JMXAttr:    "client-id=" + producerHolder + ",attr=requests-in-flight",
			},
			{
				Name:       "producer.responsePerSecond",
				SourceType: metric.GAUGE,
				JMXAttr:    "client-id=" + producerHolder + ",attr=response-rate",
			},
			{
				Name:       "producer.threadsWaiting",
				SourceType: metric.GAUGE,
				JMXAttr:    "client-id=" + producerHolder + ",attr=waiting-threads",
			},
			{
				Name:       "producer.bufferMemoryAvailableInBytes",
				SourceType: metric.GAUGE,
				JMXAttr:    "client-id=" + producerHolder + ",attr=buffer-total-bytes",
			},
			{
				Name:       "producer.maxBytesSentPerRequestInBytes",
				SourceType: metric.GAUGE,
				JMXAttr:    "client-id=" + producerHolder + ",attr=request-size-max",
			},
		},
	},
	{
		MBean:        "kafka.producer:type=ProducerTopicMetrics,name=MessagesPerSec,clientId=" + producerHolder,
		MetricPrefix: "kafka.producer:type=ProducerTopicMetrics,name=MessagesPerSec,",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "producer.messageRatePerSecond",
				SourceType: metric.GAUGE,
				JMXAttr:    "client-id=" + producerHolder + ",attr=Count",
			},
		},
	},
}

// applyProducerName to be used when passed to CollectMetricDefinitions to modified bean name for Producer
func applyProducerName(producerName string) func(string) string {
	return func(beanName string) string {
		return strings.Replace(beanName, producerHolder, producerName, -1)
	}
}

// ProducerTopicMetricDefs metric definitions for topic metrics that are specific to a Producer
var ProducerTopicMetricDefs = []*JMXMetricSet{
	{
		MBean:        "kafka.producer:type=producer-topic-metrics,client-id=" + producerHolder + ",topic=*",
		MetricPrefix: "kafka.producer:type=producer-topic-metrics,client-id=" + producerHolder + ",topic=" + topicHolder + ",",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "producer.avgRecordsSentPerTopicPerSecond",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=record-send-rate",
			},
		},
	},
}

// ApplyProducerTopicName to be used when passed to collectMetricDefinitions to modified bean name
// for Producer and Topic
func ApplyProducerTopicName(producerName, topicName string) func(string) string {
	return func(beanName string) string {
		modifiedBeanName := strings.Replace(beanName, producerHolder, producerName, -1)
		return strings.Replace(modifiedBeanName, topicHolder, topicName, -1)
	}
}
