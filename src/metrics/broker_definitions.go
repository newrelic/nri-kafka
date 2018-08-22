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
				Name:       "request.avgTimeOffset",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=Mean",
			},
			{
				Name:       "request.avgTimeOffset99Percentile",
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
				Name:       "request.avgTimeProduceRequest",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=Mean",
			},
			{
				Name:       "request.produceTime99Percentile",
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
				SourceType: metric.GAUGE,
				JMXAttr:    "name=IsrExpandsPerSec,attr=Count",
			},
			{
				Name:       "replication.isrShrinksPerSecond",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=IsrShrinksPerSec,attr=Count",
			},
			{
				Name:       "replication.unreplicatedPartitions",
				SourceType: metric.GAUGE,
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
				SourceType: metric.GAUGE,
				JMXAttr:    "name=LeaderElectionRateAndTimeMs,attr=Count",
			},
			{
				Name:       "replication.uncleanLeaderElectionPerSecond",
				SourceType: metric.GAUGE,
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
				SourceType: metric.GAUGE,
				JMXAttr:    "name=BytesInPerSec,attr=Count",
			},
			{
				Name:       "broker.IOOutPerSecond",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=BytesOutPerSec,attr=Count",
			},
			{
				Name:       "broker.messagesInPerSecond",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=MessagesInPerSec,attr=Count",
			},
			{
				Name:       "net.bytesRejectedPerSecond",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=BytesRejectedPerSec,attr=Count",
			},
			{
				Name:       "request.clientFetchesFailedPerSecond",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=FailedFetchRequestsPerSec,attr=Count",
			},
			{
				Name:       "request.produceRequestsFailedPerSecond",
				SourceType: metric.GAUGE,
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
				Name:       "broker.logFlushPerSecond",
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
		MBean:        "kafka.server:type=DelayedFetchMetrics,name=ExpiresPerSec,fetcherType=*",
		MetricPrefix: "kafka.server:type=DelayedFetchMetrics,name=ExpiresPerSec,",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "consumer.requestsExpiredPerSecond",
				SourceType: metric.GAUGE,
				JMXAttr:    "fetcherType=consumer,attr=Count",
			},
			{
				Name:       "follower.requestExpirationPerSecond",
				SourceType: metric.GAUGE,
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
func ApplyTopicName(topicName string) BeanModifier {
	return func(beanName string) string {
		return strings.Replace(beanName, topicHolder, topicName, -1)
	}
}
