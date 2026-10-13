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
		},
	},
}
