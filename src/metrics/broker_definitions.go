package metrics

import (
	"strings"

	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
)

// Broker metrics
var brokerRequestMetricDefs = []*JMXMetricSet{
	{
		MBean:        "kafka.network:type=RequestMetrics,name=RequestsPerSec,request=*,*",
		MetricPrefix: "kafka.network:type=RequestMetrics,name=RequestsPerSec,",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "request.produceRequestsPerSecond",
				SourceType: metric.GAUGE,
				JMXAttr:    "request=Produce",
			},
			{
				Name:       "request.fetchConsumerRequestsPerSecond",
				SourceType: metric.GAUGE,
				JMXAttr:    "request=FetchConsumer",
			},
			{
				Name:       "request.fetchFollowerRequestsPerSecond",
				SourceType: metric.GAUGE,
				JMXAttr:    "request=FetchFollower",
			},
			{
				Name:       "request.metadataRequestsPerSecond",
				SourceType: metric.GAUGE,
				JMXAttr:    "request=Metadata",
			},
			{
				Name:       "request.offsetCommitRequestsPerSecond",
				SourceType: metric.GAUGE,
				JMXAttr:    "request=OffsetCommit",
			},
			{
				Name:       "request.listGroupsRequestsPerSecond",
				SourceType: metric.GAUGE,
				JMXAttr:    "request=ListGroups",
			},
		},
	},
}

// Broker metrics
var brokerMetricDefs = []*JMXMetricSet{
	// Controller status - unlike GlobalPartitionCount (only accurate from the active
	// controller's own tracked state, see ClusterMetricDefs), ActiveControllerCount is a
	// strict 0/1 gauge that's valid to read from any broker's own JMX: it just reports
	// whether that specific broker is currently the elected controller.
	{
		MBean:        "kafka.controller:type=KafkaController,name=ActiveControllerCount",
		MetricPrefix: "kafka.controller:type=KafkaController,name=ActiveControllerCount,",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "broker.isActiveController",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=Value",
			},
		},
	},
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
				SourceType: metric.GAUGE,
				JMXAttr:    "name=UnderReplicatedPartitions,attr=Value",
			},
			{
				Name:       "broker.leaderCount",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=LeaderCount,attr=Value",
			},
			{
				Name:       "broker.underMinIsrPartitionCount",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=UnderMinIsrPartitionCount,attr=Value",
			},
			{
				// One failure away from violating min.insync.replicas - the early-warning
				// counterpart to UnderMinIsrPartitionCount, which only fires after the fact.
				Name:       "broker.atMinIsrPartitionCount",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=AtMinIsrPartitionCount,attr=Value",
			},
		},
	},
	// Replica fetcher lag - how far this broker's follower replicas are behind their
	// leaders, distinct from consumer lag. No equivalent anywhere else in this file.
	{
		MBean:        "kafka.server:type=ReplicaFetcherManager,name=MaxLag,clientId=Replica",
		MetricPrefix: "kafka.server:type=ReplicaFetcherManager,name=MaxLag,clientId=Replica,",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "replication.maxLag",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=Value",
			},
		},
	},
	// Request queue depth - fills up ahead of handler-thread starvation, a leading
	// indicator broker.* throughput/latency metrics don't otherwise surface.
	{
		MBean:        "kafka.network:type=RequestChannel,name=RequestQueueSize",
		MetricPrefix: "kafka.network:type=RequestChannel,name=RequestQueueSize,",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "request.queueSize",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=Value",
			},
		},
	},
	// Purgatory depth - requests currently parked waiting on acks/fetch data, not just
	// the rate at which they eventually expire (consumer.requestsExpiredPerSecond above).
	{
		MBean:        "kafka.server:type=DelayedOperationPurgatory,name=PurgatorySize,delayedOperation=*",
		MetricPrefix: "kafka.server:type=DelayedOperationPurgatory,name=PurgatorySize,",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "broker.produceRequestPurgatorySize",
				SourceType: metric.GAUGE,
				JMXAttr:    "delayedOperation=Produce,attr=Value",
			},
			{
				Name:       "broker.fetchRequestPurgatorySize",
				SourceType: metric.GAUGE,
				JMXAttr:    "delayedOperation=Fetch,attr=Value",
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
				Name:       "broker.messagesInPerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "name=MessagesInPerSec,attr=Count",
			},
			{
				Name:       "net.bytesRejectedPerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "name=BytesRejectedPerSec,attr=Count",
			},
			{
				Name:       "request.clientFetchesFailedPerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "name=FailedFetchRequestsPerSec,attr=Count",
			},
			{
				Name:       "request.produceRequestsFailedPerSecond",
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
				Name:       "broker.logFlushPerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "name=LogFlushRateAndTimeMs,attr=Count",
			},
			{
				Name:       "broker.logFlushTimeMsMean",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=LogFlushRateAndTimeMs,attr=Mean",
			},
			{
				Name:       "broker.logFlushTime99Percentile",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=LogFlushRateAndTimeMs,attr=99thPercentile",
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
				Name:       "broker.bytesWrittenToTopicPerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "attr=Count",
			},
		},
	},
}

// BrokerTopicMetricDefs metric definitions for topic metrics that are specific to a Broker
var BrokerTopicV2MetricDefs = []*JMXMetricSet{
	{
		MBean:        "kafka.server:type=BrokerTopicMetrics,name=BytesOutPerSec,topic=" + topicHolder,
		MetricPrefix: "kafka.server:type=BrokerTopicMetrics,name=BytesOutPerSec,topic=" + topicHolder + ",",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "broker.bytesReadFromTopicPerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "attr=Count",
			},
		},
	},
	{
		MBean:        "kafka.server:type=BrokerTopicMetrics,name=MessagesInPerSec,topic=" + topicHolder,
		MetricPrefix: "kafka.server:type=BrokerTopicMetrics,name=MessagesInPerSec,topic=" + topicHolder + ",",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "broker.messagesProducedToTopicPerSecond",
				SourceType: metric.RATE,
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

// TopicOffsetMetricDef metric definition for calculating the roll up for a Topic's
// offset for a given Broker
var TopicOffsetMetricDef = &JMXMetricSet{
	MBean: "kafka.log:type=Log,name=LogEndOffset,topic=" + topicHolder + ",partition=*",
}

// ApplyTopicName to modified bean name for Topic
func ApplyTopicName(topicName string) BeanModifier {
	return func(beanName string) string {
		return strings.Replace(beanName, topicHolder, topicName, -1)
	}
}
