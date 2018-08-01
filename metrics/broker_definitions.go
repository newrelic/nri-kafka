package metrics

import (
	"strings"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
)

// Broker metrics
var brokerMetricDefs = []*JMXMetricSet{
	// Metadata request Metrics
	{
		MBean:        "kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Metadata",
		MetricPrefix: "kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Metadata,",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "request.avgTimeMetadata",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=Mean",
			},
			{
				Name:       "request.avgTimeMetadata99Percentile",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=99thPercentile",
			},
		},
	},
	// Fetch request metrics
	{
		MBean:        "kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Fetch",
		MetricPrefix: "kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Fetch,",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "request.avgTimeFetch",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=Mean",
			},
			{
				Name:       "request.fetchTime99Percentile",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=99thPercentile",
			},
		},
	},
	// Offset request metrics
	{
		MBean:        "kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Offsets",
		MetricPrefix: "kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Offsets,",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "request.avgTimeUpdateMetadata",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=Mean",
			},
			{
				Name:       "request.avgTimeUpdateMetadata99Percentile",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=99thPercentile",
			},
		},
	},
	// UpdateMetadata request metrics
	{
		MBean:        "kafka.network:type=RequestMetrics,name=TotalTimeMs,request=UpdateMetadata",
		MetricPrefix: "kafka.network:type=RequestMetrics,name=TotalTimeMs,request=UpdateMetadata,",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "request.avgTimeUpdateMetadata",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=Mean",
			},
			{
				Name:       "request.avgTimeUpdateMetadata99Percentile",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=99thPercentile",
			},
		},
	},
	// Produce request metrics
	{
		MBean:        "kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Produce",
		MetricPrefix: "kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Produce,",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "requests.avgTimeProduceRequest",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=Mean",
			},
			{
				Name:       "requests.produceTime99Percentile",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=99thPercentile",
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
			{
				Name:       "replication.unreplicatedPartitions",
				SourceType: metric.RATE,
				JMXAttr:    "name=UnderReplicatedPartitions,attr=Value",
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

// BrokerTopicMetricDefs metric definitions for topic metrics that are specific to a Broker
var BrokerTopicMetricDefs = []*JMXMetricSet{
	{
		MBean:        "kafka.server:type=BrokerTopicMetrics,name=BytesInPerSec,topic=" + topicHolder,
		MetricPrefix: "kafka.server:type=BrokerTopicMetrics,name=BytesInPerSec,topic=" + topicHolder + ",",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "topic.bytesWritten",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=Count",
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
func ApplyTopicName(topicName string) func(string) string {
	return func(beanName string) string {
		return strings.Replace(beanName, topicHolder, topicName, -1)
	}
}
