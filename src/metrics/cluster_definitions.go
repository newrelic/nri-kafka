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
				Name:       "cluster.activeControllerCount",
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
				Name:       "cluster.numGroups",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=NumGroups,attr=Value",
			},
			{
				Name:       "cluster.numOffsets",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=NumOffsets,attr=Value",
			},
			{
				Name:       "cluster.numGroupsStable",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=NumGroupsStable,attr=Value",
			},
			{
				Name:       "cluster.numGroupsPreparingRebalance",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=NumGroupsPreparingRebalance,attr=Value",
			},
			{
				Name:       "cluster.numGroupsDead",
				SourceType: metric.GAUGE,
				JMXAttr:    "name=NumGroupsDead,attr=Value",
			},
		},
	},
	// KafkaServer for ClusterId
	{
		MBean:        "kafka.server:type=KafkaServer,name=ClusterId",
		MetricPrefix: "kafka.server:type=KafkaServer,",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "cluster.id",
				SourceType: metric.ATTRIBUTE,
				JMXAttr:    "name=ClusterId,attr=Value",
			},
		},
	},
}
