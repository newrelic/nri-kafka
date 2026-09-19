package metrics

import "github.com/newrelic/infra-integrations-sdk/v3/data/metric"

// jvmMetricDefs collects standard Java platform MBean metrics (heap memory, threads,
// OS-level CPU load/file descriptors, loaded classes). These aren't Kafka-specific MBeans -
// they're generic JVM platform MBeans exposed on the same JMX connection nri-kafka already
// holds open to the broker, since it's the same JVM process.
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
	// Threads
	{
		MBean:        "java.lang:type=Threading",
		MetricPrefix: "java.lang:type=Threading,",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "jvm.threadCount",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=ThreadCount",
			},
			{
				Name:       "jvm.daemonThreadCount",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=DaemonThreadCount",
			},
		},
	},
	// OS-level: CPU load, file descriptors, processor count
	{
		MBean:        "java.lang:type=OperatingSystem",
		MetricPrefix: "java.lang:type=OperatingSystem,",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "jvm.systemLoadAverage",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=SystemLoadAverage",
			},
			{
				Name:       "jvm.availableProcessors",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=AvailableProcessors",
			},
			{
				// Only present on Unix JVMs (com.sun.management.UnixOperatingSystemMXBean);
				// absent on Windows, handled like any other not-found JMX attribute.
				Name:       "jvm.openFileDescriptorCount",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=OpenFileDescriptorCount",
			},
		},
	},
	// Loaded classes
	{
		MBean:        "java.lang:type=ClassLoading",
		MetricPrefix: "java.lang:type=ClassLoading,",
		MetricDefs: []*MetricDefinition{
			{
				Name:       "jvm.classLoadedCount",
				SourceType: metric.GAUGE,
				JMXAttr:    "attr=LoadedClassCount",
			},
		},
	},
}

const (
	jvmGCMBean               = "java.lang:type=GarbageCollector,name=*"
	jvmGCCollectionCountAttr = ",attr=CollectionCount"
	jvmGCCollectionTimeAttr  = ",attr=CollectionTime"
)
