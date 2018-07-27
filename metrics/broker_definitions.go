package metrics

import (
	"strings"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
)

// Broker metrics
var brokerMetricDefs = []*JMXMetricSet{
	// Request Metrics
	{
		MBean:        "kafka.network:type=RequestMetrics,name=TotalTimeMs,request=*",
		MetricPrefix: "kafka.network:type=RequestMetrics,name=TotalTimeMs,",
		MetricDefs: []*MetricDefinition{
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
		MetricDefs: []*MetricDefinition{
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
		MetricDefs: []*MetricDefinition{
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
		MetricDefs: []*MetricDefinition{
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
		MetricDefs: []*MetricDefinition{
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
		MetricDefs: []*MetricDefinition{
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
		MetricDefs: []*MetricDefinition{
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

// TopicSizeMetricDef metric definition for calculating the roll up for a Topic's
// on disk size for a given Broker
var TopicSizeMetricDef = &JMXMetricSet{
	MBean: "kafka.log:type=Log,name=Size,topic=" + topicHolder + ",partition=*",
}

// ApplyTopicName to modified bean name for Topic
func ApplyTopicName(beanName, topicName string) string {
	return strings.Replace(beanName, topicHolder, topicName, -1)
}
