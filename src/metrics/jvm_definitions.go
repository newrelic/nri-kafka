package metrics

import (
	"regexp"
	"strings"

	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
)

// jvmMetricDefs collects standard Java platform heap-memory MBean metrics. This isn't a
// Kafka-specific MBean - it's a generic JVM platform MBean exposed on the same JMX connection
// nri-kafka already holds open to the broker, since it's the same JVM process.
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
			{
				Name:       "jvm.heapMemoryCommittedBytes",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=HeapMemoryUsage.Committed",
			},
		},
	},
}

const (
	jvmGCMBean               = "java.lang:type=GarbageCollector,name=*"
	jvmGCCollectionCountAttr = ",attr=CollectionCount"
	jvmGCCollectionTimeAttr  = ",attr=CollectionTime"
)

// jvmMBeanNameRegex extracts a wildcarded "name=" value without anchoring on what follows it:
// key order varies by MBean (JDK platform beans put name before type, Kafka's own beans do
// the opposite).
var jvmMBeanNameRegex = regexp.MustCompile(`name="?([^,"]+)"?`)

// classifyGCGeneration buckets a GC collector name into young/old generation. Unmatched
// collectors (e.g. G1's concurrent marking cycle) count toward neither bucket, only the
// flat jvm.gcCollectionsPerSecond/gcTimePerSecond totals.
func classifyGCGeneration(collectorName string) (young, old bool) {
	lower := strings.ToLower(collectorName)
	switch {
	case strings.Contains(lower, "young"), strings.Contains(lower, "scavenge"), strings.Contains(lower, "parnew"), lower == "copy":
		return true, false
	case strings.Contains(lower, "old"), strings.Contains(lower, "marksweep"), strings.Contains(lower, "tenured"):
		return false, true
	default:
		return false, false
	}
}
