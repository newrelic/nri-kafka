package main

import (
	"strings"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
)

const (
	consumerHolder = "%CONSUMER%"
	producerHolder = "%PRODUCER%"
	topicHolder    = "%TOPIC%"
)

type metricDefinition struct {
	Name       string
	SourceType metric.SourceType
	JMXAttr    string
}

type jmxMetricSet struct {
	MBean        string
	MetricPrefix string
	MetricDefs   []*metricDefinition
}

// Broker metrics
var brokerMetricDefs = []*jmxMetricSet{
	// Request Metrics
	{
		MBean:        "kafka.network:type=RequestMetrics,name=TotalTimeMs,request=*",
		MetricPrefix: "kafka.network:type=RequestMetrics,name=TotalTimeMs,",
		MetricDefs: []*metricDefinition{
			{
				Name:       "request.avgTimeFetch",
				SourceType: metric.GAUGE,
				JMXAttr:    "request=Fetch,attr=Mean",
			},
			{
				Name:       "request.avgTimeMetadata",
				SourceType: metric.GAUGE,
				JMXAttr:    "request=Metadata,attr=Mean",
			},
			{
				Name:       "request.avgTimeMetadata99Percentile",
				SourceType: metric.GAUGE,
				JMXAttr:    "request=Metadata,attr=99thPercentile",
			},
			{
				Name:       "request.avgTimeOffset",
				SourceType: metric.GAUGE,
				JMXAttr:    "request=Offsets,attr=Mean",
			},
			{
				Name:       "request.avgTimeOffset99Percentile",
				SourceType: metric.GAUGE,
				JMXAttr:    "request=Offsets,attr=99thPercentile",
			},
			{
				Name:       "request.avgTimeUpdateMetadata",
				SourceType: metric.GAUGE,
				JMXAttr:    "request=UpdateMetadata,attr=Mean",
			},
			{
				Name:       "request.avgTimeUpdateMetadata99Percentile",
				SourceType: metric.GAUGE,
				JMXAttr:    "request=UpdateMetadata,attr=99thPercentile",
			},
			{
				Name:       "request.fetchTime99Percentile",
				SourceType: metric.GAUGE,
				JMXAttr:    "request=Fetch,attr=99thPercentile",
			},
			{
				Name:       "requests.avgTimeProduceRequest",
				SourceType: metric.GAUGE,
				JMXAttr:    "request=Produce,attr=Mean",
			},
			{
				Name:       "requests.produceTime99Percentile",
				SourceType: metric.GAUGE,
				JMXAttr:    "request=Produce,attr=99thPercentile",
			},
		},
	},
	// ReplicaManager Metrics
	{
		MBean:        "kafka.server:type=ReplicaManager,name=*",
		MetricPrefix: "kafka.server:type=ReplicaManager,",
		MetricDefs: []*metricDefinition{
			{
				Name:       "replication.isrExpandsPerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "name=IsrExpandsPerSec,attr=Count",
			},
			{
				Name:       "replication.isrShrinksPerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "name=IsrShrinksPerSec,attr=Count",
			},
		},
	},
	// Leader Metrics
	{
		MBean:        "kafka.controller:type=ControllerStats,name=*",
		MetricPrefix: "kafka.controller:type=ControllerStats,",
		MetricDefs: []*metricDefinition{
			{
				Name:       "replication.leaderElectionPerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "name=LeaderElectionRateAndTimeMs,attr=Count",
			},
			{
				Name:       "replication.uncleanLeaderElectionPerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "name=UncleanLeaderElectionsPerSec,attr=Count",
			},
		},
	},
	// Broker Topic Metrics
	{
		MBean:        "kafka.server:type=BrokerTopicMetrics,name=*",
		MetricPrefix: "kafka.server:type=BrokerTopicMetrics,",
		MetricDefs: []*metricDefinition{
			{
				Name:       "broker.IOInPerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "name=BytesInPerSec,attr=Count",
			},
			{
				Name:       "broker.IOOutPerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "name=BytesOutPerSec,attr=Count",
			},
			{
				Name:       "messagesInPerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "name=MessagesInPerSec,attr=Count",
			},
			{
				Name:       "net.bytesRejectedPerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "name=BytesRejectedPerSec,attr=Count",
			},
			{
				Name:       "requests.clientFetchesFailedPerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "name=FailedFetchRequestsPerSec,attr=Count",
			},
			{
				Name:       "requests.produceRequestsFailedPerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "name=FailedProduceRequestsPerSec,attr=Count",
			},
		},
	},
	// Log Flush Stats
	{
		MBean:        "kafka.log:type=LogFlushStats,name=*",
		MetricPrefix: "kafka.log:type=LogFlushStats,",
		MetricDefs: []*metricDefinition{
			{
				Name:       "logFlushPerSecond",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=LogFlushRateAndTimeMs,attr=Count",
			},
		},
	},
	// Idle Handler
	{
		MBean:        "kafka.server:type=KafkaRequestHandlerPool,name=RequestHandlerAvgIdlePercent",
		MetricPrefix: "kafka.server:type=KafkaRequestHandlerPool,name=RequestHandlerAvgIdlePercent,",
		MetricDefs: []*metricDefinition{
			{
				Name:       "request.handlerIdle",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=OneMinuteRate",
			},
		},
	},
	// Delayed Fetch Metrics
	{
		MBean:        "kafak.server:type=DelayedFetchMetrics,name=ExpiresPerSec,fetcherType=*",
		MetricPrefix: "kafak.server:type=DelayedFetchMetrics,name=ExpiresPerSec,",
		MetricDefs: []*metricDefinition{
			{
				Name:       "consumer.requestsExpiredPerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "fetcherType=consumer,attr=Count",
			},
			{
				Name:       "follower.requestExpirationPerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "fetcherType=follower,attr=Count",
			},
		},
	},
}

// Consumer Metrics
var consumerMetricDefs = []*jmxMetricSet{
	{
		MBean:        "kafka.consumer:type=consumer-fetch-manager-metrics,client-id=" + consumerHolder,
		MetricPrefix: "kafka.consumer:type=consumer-fetch-manager-metrics,client-id=" + consumerHolder,
		MetricDefs: []*metricDefinition{
			{
				Name:       "consumer.bytesInPerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "attr=consumed-rate",
			},
			{
				Name:       "consumer.fetchPerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "attr=fetch-rate",
			},
			{
				Name:       "consumer.maxLag",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=records-lag-max",
			},
			{
				Name:       "consumer.MessageConsumptionPerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "attr=records-consumed-rate",
			},
		},
	},
	{
		MBean:        "kafka.consumer:type=ZookeeperConsumerConnector,name=*,clientId=" + consumerHolder,
		MetricPrefix: "kafka.consumer:type=ZookeeperConsumerConnector,",
		MetricDefs: []*metricDefinition{
			{
				Name:       "consumer.offsetKafkaCommitsPerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "name=KafkaCommitsPerSec,clientId=" + consumerHolder + "attr=Count",
			},
			{
				Name:       "consumer.offsetZooKeeperCommitsPerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "name=ZooKeeperCommitsPerSec,clientId=" + consumerHolder + "attr=Count",
			},
		},
	},
}

// Function to be used when passed to collectMetricDefinitions to modified bean name for Consumer
func applyConsumerName(consumerName string) func(string) string {
	return func(beanName string) string {
		return strings.Replace(beanName, consumerHolder, consumerName, -1)
	}
}

// Consumer Topic specific metrics
var consumerTopicMetricDefs = []*jmxMetricSet{
	{
		MBean:        "kafka.consumer:type=consumer-fetch-manager-metrics,client-id=" + consumerHolder + ",topic=*",
		MetricPrefix: "kafka.consumer:type=consumer-fetch-manager-metrics,client-id=" + consumerHolder + ",topic=" + topicHolder + ",",
		MetricDefs: []*metricDefinition{
			{
				Name:       "consumer.avgFetchSizeInBytes",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=fetch-size-avg",
			},
			{
				Name:       "consumer.maxFetchSizeInBytes",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=fetch-size-max",
			},
			{
				Name:       "consumer.avgRecordConsumedPerTopicPerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "attr=records-consumed-rate",
			},
			{
				Name:       "consumer.avgRecordConsumedPerTopic",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=records-per-request-avg",
			},
		},
	},
}

// Function to be used when passed to collectMetricDefinitions to modified bean name
// for Consumer and Topic
func applyconsumerTopicName(consumerName, topicName string) func(string) string {
	return func(beanName string) string {
		modifiedBeanName := strings.Replace(beanName, consumerHolder, consumerName, -1)
		return strings.Replace(modifiedBeanName, topicHolder, topicName, -1)
	}
}

// Producer Metrics
var producerMetricDefs = []*jmxMetricSet{
	{
		MBean:        "kafka.producer:type=producer-metrics,client-id=" + producerHolder,
		MetricPrefix: "kafka.producer:type=producer-metrics,",
		MetricDefs: []*metricDefinition{
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
				SourceType: metric.RATE,
				JMXAttr:    "client-id=" + producerHolder + ",attr=record-send-rate",
			},
			{
				Name:       "producer.avgRequestLatencyPerSecond",
				SourceType: metric.RATE,
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
				JMXAttr:    "client-id=" + producerHolder + ",attr=bufferpool-wait-time",
			},
			{
				Name:       "producer.bytesOutPerSecond",
				SourceType: metric.RATE,
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
				SourceType: metric.RATE,
				JMXAttr:    "client-id=" + producerHolder + ",attr=request-rate",
			},
			{
				Name:       "producer.requestsWaitingResponse",
				SourceType: metric.GAUGE,
				JMXAttr:    "client-id=" + producerHolder + ",attr=requests-in-flight",
			},
			{
				Name:       "producer.responsePerSecond",
				SourceType: metric.RATE,
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
		MetricDefs: []*metricDefinition{
			{
				Name:       "producer.messageRatePerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "client-id=" + producerHolder + ",attr=Count",
			},
		},
	},
}

// Function to be used when passed to collectMetricDefinitions to modified bean name for Consumer
func applyProducerName(producerName string) func(string) string {
	return func(beanName string) string {
		return strings.Replace(beanName, producerHolder, producerName, -1)
	}
}

// Producer Topic specific metrics
var producerTopicMetricDefs = []*jmxMetricSet{
	{
		MBean:        "kafka.producer:type=producer-topic-metrics,client-id=" + producerHolder + ",topic=*",
		MetricPrefix: "kafka.producer:type=producer-topic-metrics,client-id=" + producerHolder + ",topic=" + topicHolder + ",",
		MetricDefs: []*metricDefinition{
			{
				Name:       "producer.avgRecordsSentPerTopicPerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "attr=record-send-rate",
			},
		},
	},
}

// Function to be used when passed to collectMetricDefinitions to modified bean name
// for Producer and Topic
func applyProducerTopicName(producerName, topicName string) func(string) string {
	return func(beanName string) string {
		modifiedBeanName := strings.Replace(beanName, producerHolder, producerName, -1)
		return strings.Replace(modifiedBeanName, topicHolder, topicName, -1)
	}
}

var topicSizeMetricDef = &jmxMetricSet{
	MBean: "kafka.log:type=Log,name=Size,topic=" + topicHolder + ",partition=*",
}

// Function to modified bean name for Topic
func applyTopicName(beanName, topicName string) string {
	return strings.Replace(beanName, topicHolder, topicName, -1)
}
