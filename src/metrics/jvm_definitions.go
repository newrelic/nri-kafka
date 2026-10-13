package metrics

import (
	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
)

// jvmMetricDefs collects standard JVM heap-memory metrics over the same JMX connection already open to the broker.
var jvmMetricDefs = []*JMXMetricSet{
	// Heap memory - nrjmx flattens the composite HeapMemoryUsage attribute with capitalized
	// sub-field names (HeapMemoryUsage.Used, not .used), verified against a live broker.
	{
		MBean:        "java.lang:type=Memory",
		MetricPrefix: "java.lang:type=Memory,",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "jvm.heapMemoryUsedBytes",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=HeapMemoryUsage.Used",
			},
			{
				Name:       "jvm.heapMemoryMaxBytes",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=HeapMemoryUsage.Max",
			},
		},
	},
}

const (
	jvmGCMBean              = "java.lang:type=GarbageCollector,name=*"
	jvmGCCollectionTimeAttr = ",attr=CollectionTime"
)
