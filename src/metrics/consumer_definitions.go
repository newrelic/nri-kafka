package metrics

import (
	"strings"

	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
)

const consumerHolder = "%CONSUMER%"

// Consumer Metrics
var consumerMetricDefs = []*JMXMetricSet{
	{
		MBean:        "kafka.consumer:type=consumer-fetch-manager-metrics,client-id=" + consumerHolder,
		MetricPrefix: "kafka.consumer:type=consumer-fetch-manager-metrics,client-id=" + consumerHolder + ",",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "consumer.bytesInPerSecond",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=bytes-consumed-rate",
			},
			{
				Name:       "consumer.fetchPerSecond",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=fetch-rate",
			},
			{
				Name:       "consumer.maxLag",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=records-lag-max",
			},
			{
				Name:       "consumer.messageConsumptionPerSecond",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=records-consumed-rate",
			},
			{
				// No request-latency visibility existed on the consumer side at all before
				// this - producer had it (avgRequestLatencyPerSecond), consumer didn't.
				Name:       "consumer.fetchLatencyAvg",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=fetch-latency-avg",
			},
			{
				Name:       "consumer.fetchLatencyMax",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=fetch-latency-max",
			},
			{
				Name:       "consumer.fetchThrottleTimeAvg",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=fetch-throttle-time-avg",
			},
			{
				Name:       "consumer.fetchThrottleTimeMax",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=fetch-throttle-time-max",
			},
		},
	},
	{
		// Rebalance churn - frequent or failing rebalances indicate crash-looping consumers
		// or a misconfigured session.timeout.ms, not visible from any broker-side metric.
		MBean:        "kafka.consumer:type=consumer-coordinator-metrics,client-id=" + consumerHolder,
		MetricPrefix: "kafka.consumer:type=consumer-coordinator-metrics,client-id=" + consumerHolder + ",",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "consumer.rebalanceTotal",
				SourceType: metric.RATE,
				JMXAttr:    "attr=rebalance-total",
			},
			{
				Name:       "consumer.failedRebalanceTotal",
				SourceType: metric.RATE,
				JMXAttr:    "attr=failed-rebalance-total",
			},
			{
				// Session/heartbeat health - the metric that tells you a consumer is about
				// to be evicted from its group, before rebalanceTotal above ever fires.
				Name:       "consumer.heartbeatRate",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=heartbeat-rate",
			},
			{
				Name:       "consumer.lastHeartbeatSecondsAgo",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=last-heartbeat-seconds-ago",
			},
			{
				Name:       "consumer.heartbeatResponseTimeMax",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=heartbeat-response-time-max",
			},
			{
				// Detects a consumer that's silently lost all its partitions (e.g. stuck
				// mid-rebalance) - invisible from rebalanceTotal alone.
				Name:       "consumer.assignedPartitions",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=assigned-partitions",
			},
			{
				// Modern commit-health metrics. Unlike consumer.offsetKafkaCommitsPerSecond
				// below (kafka.consumer:type=ZookeeperConsumerConnector, dead on any client
				// 0.9+), these read from the actual coordinator MBean every real consumer uses.
				Name:       "consumer.commitRate",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=commit-rate",
			},
			{
				Name:       "consumer.commitLatencyAvg",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=commit-latency-avg",
			},
			{
				Name:       "consumer.commitLatencyMax",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=commit-latency-max",
			},
		},
	},
	{
		MBean:        "kafka.consumer:type=ZookeeperConsumerConnector,name=*,clientId=" + consumerHolder,
		MetricPrefix: "kafka.consumer:type=ZookeeperConsumerConnector,",
		MetricDefs: []*MetricDefinition{
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

// ConsumerTopicMetricDefs metric definitions for topic metrics that are specific to a Consumer
var ConsumerTopicMetricDefs = []*JMXMetricSet{
	{
		MBean:        "kafka.consumer:type=consumer-fetch-manager-metrics,client-id=" + consumerHolder + ",topic=*",
		MetricPrefix: "kafka.consumer:type=consumer-fetch-manager-metrics,client-id=" + consumerHolder + ",topic=" + topicHolder + ",",
		MetricDefs: []*MetricDefinition{
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
				SourceType: metric.GAUGE,
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

// applyConsumerName to be used when passed to CollectMetricDefinitions to modified bean name for Consumer
func applyConsumerName(consumerName string) BeanModifier {
	return func(beanName string) string {
		return strings.Replace(beanName, consumerHolder, consumerName, -1)
	}
}

// ApplyConsumerTopicName to be used when passed to CollectMetricDefinitions to modified bean name
// for Consumer and Topic
func ApplyConsumerTopicName(consumerName, topicName string) BeanModifier {
	return func(beanName string) string {
		modifiedBeanName := strings.Replace(beanName, consumerHolder, consumerName, -1)
		return strings.Replace(modifiedBeanName, topicHolder, topicName, -1)
	}
}
