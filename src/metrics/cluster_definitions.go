package metrics

import (
	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
)

// ClusterMetricDefs contains definitions for cluster-wide metrics from the controller
var ClusterMetricDefs = []*JMXMetricSet{
	// KafkaController metrics
	{
		MBean:        "kafka.controller:type=KafkaController,name=*",
		MetricPrefix: "kafka.controller:type=KafkaController,",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "cluster.activeBrokerCount",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=ActiveBrokerCount,attr=Value",
			},
			{
				Name:       "cluster.offlinePartitionsCount",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=OfflinePartitionsCount,attr=Value",
			},
			{
				Name:       "cluster.preferredReplicaImbalanceCount",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=PreferredReplicaImbalanceCount,attr=Value",
			},
			{
				Name:       "cluster.globalTopicCount",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=GlobalTopicCount,attr=Value",
			},
			{
				Name:       "cluster.globalPartitionCount",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=GlobalPartitionCount,attr=Value",
			},
			{
				// ActiveControllerCount is a strict 0/1 gauge in both ZK and KRaft mode
				// (kafka.controller.KafkaController / QuorumControllerMetrics both define it
				// as `if (isActive) 1 else 0`) - never a count - hence the boolean name here.
				Name:       "cluster.isActiveController",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=ActiveControllerCount,attr=Value",
			},
			{
				Name:       "cluster.fencedBrokerCount",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=FencedBrokerCount,attr=Value",
			},
		},
	},
	// ControllerStats metrics
	{
		MBean:        "kafka.controller:type=ControllerStats,name=UncleanLeaderElectionsPerSec",
		MetricPrefix: "kafka.controller:type=ControllerStats,",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "cluster.uncleanLeaderElectionsPerSecond",
				SourceType: metric.RATE,
				JMXAttr:    "name=UncleanLeaderElectionsPerSec,attr=Count",
			},
		},
	},
	// GroupMetadataManager metrics
	{
		MBean:        "kafka.coordinator.group:type=GroupMetadataManager,name=*",
		MetricPrefix: "kafka.coordinator.group:type=GroupMetadataManager,",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "cluster.groupCount",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=NumGroups,attr=Value",
			},
			{
				Name:       "cluster.offsetCount",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=NumOffsets,attr=Value",
			},
			{
				Name:       "cluster.stableGroupCount",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=NumGroupsStable,attr=Value",
			},
			{
				Name:       "cluster.rebalancingGroupCount",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=NumGroupsPreparingRebalance,attr=Value",
			},
			{
				Name:       "cluster.deadGroupCount",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=NumGroupsDead,attr=Value",
			},
		},
	},
}
