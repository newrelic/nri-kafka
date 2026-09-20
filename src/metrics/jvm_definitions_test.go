package metrics

import (
	"reflect"
	"testing"

	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/connection/mocks"
	"github.com/newrelic/nri-kafka/src/testutils"
	"github.com/newrelic/nrjmx/gojmx"
)

func jvmMockResponse() *mocks.MockJMXResponse {
	return &mocks.MockJMXResponse{
		Result: []*gojmx.AttributeResponse{
			{
				Name:         "java.lang:type=Memory,attr=HeapMemoryUsage.Used",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     1000,
			},
			{
				Name:         "java.lang:type=Threading,attr=ThreadCount",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     120,
			},
			{
				Name:         "java.lang:type=Runtime,attr=Uptime",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     3798168,
			},
			{
				Name:         "java.lang:type=OperatingSystem,attr=ProcessCpuLoad",
				ResponseType: gojmx.ResponseTypeDouble,
				DoubleValue:  0.125,
			},
			{
				Name:         "java.lang:type=OperatingSystem,attr=CpuLoad",
				ResponseType: gojmx.ResponseTypeDouble,
				DoubleValue:  0.36,
			},
			{
				Name:         "java.lang:type=Memory,attr=NonHeapMemoryUsage.Used",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     500,
			},
		},
	}
}

func TestGetBrokerMetrics_JVMMetricsDisabledByDefault(t *testing.T) {
	testutils.SetupTestArgs()

	mockJMXProvider := &mocks.MockJMXProvider{Response: jvmMockResponse()}

	i, _ := integration.New("test", "1.0.0")
	e, _ := i.Entity("jvmDisabledEntity", "jvmDisabledNamespace")
	m := e.NewMetricSet("testMetrics")

	GetBrokerMetrics(m, mockJMXProvider)

	if _, ok := m.Metrics["jvm.heapMemoryUsedBytes"]; ok {
		t.Error("jvm.heapMemoryUsedBytes should not be collected when EnableBrokerJVMMetrics is false")
	}
	if _, ok := m.Metrics["jvm.threadCount"]; ok {
		t.Error("jvm.threadCount should not be collected when EnableBrokerJVMMetrics is false")
	}
	if _, ok := m.Metrics["jvm.systemCpuLoad"]; ok {
		t.Error("jvm.systemCpuLoad should not be collected when EnableBrokerJVMMetrics is false")
	}
	if _, ok := m.Metrics["jvm.heapEdenUsedBytes"]; ok {
		t.Error("jvm.heapEdenUsedBytes should not be collected when EnableBrokerJVMMetrics is false")
	}
}

func TestGetBrokerMetrics_JVMMetricsEnabled(t *testing.T) {
	testutils.SetupTestArgs()
	args.GlobalArgs.EnableBrokerJVMMetrics = true

	mockJMXProvider := &mocks.MockJMXProvider{Response: jvmMockResponse()}

	i, _ := integration.New("test", "1.0.0")
	e, _ := i.Entity("jvmEnabledEntity", "jvmEnabledNamespace")
	m := e.NewMetricSet("testMetrics")

	GetBrokerMetrics(m, mockJMXProvider)

	if got := m.Metrics["jvm.heapMemoryUsedBytes"]; got != float64(1000) {
		t.Errorf("expected jvm.heapMemoryUsedBytes = 1000, got %v", got)
	}
	if got := m.Metrics["jvm.threadCount"]; got != float64(120) {
		t.Errorf("expected jvm.threadCount = 120, got %v", got)
	}
	if got := m.Metrics["jvm.uptimeMs"]; got != float64(3798168) {
		t.Errorf("expected jvm.uptimeMs = 3798168, got %v", got)
	}
	if got := m.Metrics["jvm.processCpuLoad"]; got != float64(0.125) {
		t.Errorf("expected jvm.processCpuLoad = 0.125, got %v", got)
	}
	if got := m.Metrics["jvm.systemCpuLoad"]; got != float64(0.36) {
		t.Errorf("expected jvm.systemCpuLoad = 0.36, got %v", got)
	}
	if got := m.Metrics["jvm.nonHeapMemoryUsedBytes"]; got != float64(500) {
		t.Errorf("expected jvm.nonHeapMemoryUsedBytes = 500, got %v", got)
	}
	if _, ok := m.Metrics["jvm.gcCollectionsPerSecond"]; !ok {
		t.Error("expected jvm.gcCollectionsPerSecond to be collected when EnableBrokerJVMMetrics is true")
	}
	if _, ok := m.Metrics["jvm.heapEdenUsedBytes"]; !ok {
		t.Error("expected jvm.heapEdenUsedBytes to be collected when EnableBrokerJVMMetrics is true")
	}
}

func TestCollectGarbageCollectorMetrics_SumsAcrossCollectors(t *testing.T) {
	testutils.SetupTestArgs()

	// Key order here is "name=...,type=...,attr=..." (name before type), matching what a
	// real broker's java.lang:type=GarbageCollector MBean actually returns - verified live.
	// Kafka's own custom MBeans return "type=...,name=...,attr=...", the opposite order; see
	// jvmMBeanNameRegex for why extraction can't assume either ordering.
	mockResponse := &mocks.MockJMXResponse{
		Result: []*gojmx.AttributeResponse{
			{
				Name:         "java.lang:name=G1 Young Generation,type=GarbageCollector,attr=CollectionCount",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     10,
			},
			{
				Name:         "java.lang:name=G1 Old Generation,type=GarbageCollector,attr=CollectionCount",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     2,
			},
			{
				Name:         "java.lang:name=G1 Young Generation,type=GarbageCollector,attr=CollectionTime",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     500,
			},
			{
				Name:         "java.lang:name=G1 Old Generation,type=GarbageCollector,attr=CollectionTime",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     300,
			},
			{
				// G1's concurrent marking cycle - matches neither bucket in
				// classifyGCGeneration, should only count toward the flat totals.
				Name:         "java.lang:name=G1 Concurrent GC,type=GarbageCollector,attr=CollectionCount",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     1,
			},
		},
	}

	mockJMXProvider := &mocks.MockJMXProvider{Response: mockResponse}

	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Fatalf("Unexpected error %s", err.Error())
	}

	e, err := i.Entity("gcTestEntity", "gcTestNamespace")
	if err != nil {
		t.Fatalf("Unexpected error %s", err.Error())
	}

	m := e.NewMetricSet("testMetrics")

	CollectGarbageCollectorMetrics(m, mockJMXProvider)

	// All RATE-type: the SDK needs two samples over time to compute a delta, so a fresh
	// entity's first observation is always 0 - this only proves the metrics are set, not
	// that bucketing sums correctly. TestCollectMemoryPoolMetrics below proves that, since
	// its GAUGE metrics reflect real values on the first call and share the exact same
	// jvmMBeanNameRegex extraction path.
	expected := map[string]interface{}{
		"event_type":                         "testMetrics",
		"jvm.gcCollectionsPerSecond":         float64(0),
		"jvm.gcTimePerSecond":                float64(0),
		"jvm.gcYoungGenCollectionsPerSecond": float64(0),
		"jvm.gcYoungGenTimePerSecond":        float64(0),
		"jvm.gcOldGenCollectionsPerSecond":   float64(0),
		"jvm.gcOldGenTimePerSecond":          float64(0),
	}

	if !reflect.DeepEqual(expected, m.Metrics) {
		t.Errorf("Expected %+v got %+v", expected, m.Metrics)
	}
}

func TestCollectMemoryPoolMetrics_BucketsByPool(t *testing.T) {
	testutils.SetupTestArgs()

	// Key order and pool names verified live against a real broker JVM (JDK 21, G1GC).
	// Non-heap pools (Metaspace, Code Cache, ...) are included here specifically to prove
	// they're safely ignored rather than misclassified into a heap bucket.
	mockResponse := &mocks.MockJMXResponse{
		Result: []*gojmx.AttributeResponse{
			{
				Name:         "java.lang:name=G1 Eden Space,type=MemoryPool,attr=Usage.Used",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     341835776,
			},
			{
				Name:         "java.lang:name=G1 Eden Space,type=MemoryPool,attr=Usage.Max",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     -1,
			},
			{
				Name:         "java.lang:name=G1 Survivor Space,type=MemoryPool,attr=Usage.Used",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     39845888,
			},
			{
				Name:         "java.lang:name=G1 Old Gen,type=MemoryPool,attr=Usage.Used",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     175214120,
			},
			{
				Name:         "java.lang:name=G1 Old Gen,type=MemoryPool,attr=Usage.Max",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     1073741824,
			},
			{
				Name:         "java.lang:name=Metaspace,type=MemoryPool,attr=Usage.Used",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     13882472,
			},
			{
				Name:         "java.lang:name=Compressed Class Space,type=MemoryPool,attr=Usage.Used",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     1332480,
			},
		},
	}

	mockJMXProvider := &mocks.MockJMXProvider{Response: mockResponse}

	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Fatalf("Unexpected error %s", err.Error())
	}

	e, err := i.Entity("poolTestEntity", "poolTestNamespace")
	if err != nil {
		t.Fatalf("Unexpected error %s", err.Error())
	}

	m := e.NewMetricSet("testMetrics")

	CollectMemoryPoolMetrics(m, mockJMXProvider)

	expected := map[string]interface{}{
		"event_type":                "testMetrics",
		"jvm.heapEdenUsedBytes":     float64(341835776),
		"jvm.heapEdenMaxBytes":      float64(-1),
		"jvm.heapSurvivorUsedBytes": float64(39845888),
		"jvm.heapSurvivorMaxBytes":  float64(0),
		"jvm.heapOldGenUsedBytes":   float64(175214120),
		"jvm.heapOldGenMaxBytes":    float64(1073741824),
	}

	if !reflect.DeepEqual(expected, m.Metrics) {
		t.Errorf("Expected %+v got %+v", expected, m.Metrics)
	}
}

func TestClassifyGCGeneration(t *testing.T) {
	tests := []struct {
		name      string
		wantYoung bool
		wantOld   bool
	}{
		{"G1 Young Generation", true, false},
		{"G1 Old Generation", false, true},
		{"PS Scavenge", true, false},
		{"PS MarkSweep", false, true},
		{"ParNew", true, false},
		{"ConcurrentMarkSweep", false, true},
		{"Copy", true, false},
		{"MarkSweepCompact", false, true},
		{"G1 Concurrent GC", false, false},
		{"ZGC Cycles", false, false},
	}

	for _, tt := range tests {
		young, old := classifyGCGeneration(tt.name)
		if young != tt.wantYoung || old != tt.wantOld {
			t.Errorf("classifyGCGeneration(%q) = (young=%v, old=%v), want (young=%v, old=%v)",
				tt.name, young, old, tt.wantYoung, tt.wantOld)
		}
	}
}

func TestClassifyMemoryPool(t *testing.T) {
	tests := []struct {
		name         string
		wantEden     bool
		wantSurvivor bool
		wantOld      bool
	}{
		{"G1 Eden Space", true, false, false},
		{"PS Eden Space", true, false, false},
		{"G1 Survivor Space", false, true, false},
		{"G1 Old Gen", false, false, true},
		{"CMS Old Gen", false, false, true},
		{"Tenured Gen", false, false, true},
		{"Metaspace", false, false, false},
		{"Compressed Class Space", false, false, false},
		{"CodeHeap 'non-nmethods'", false, false, false},
	}

	for _, tt := range tests {
		eden, survivor, old := classifyMemoryPool(tt.name)
		if eden != tt.wantEden || survivor != tt.wantSurvivor || old != tt.wantOld {
			t.Errorf("classifyMemoryPool(%q) = (eden=%v, survivor=%v, old=%v), want (eden=%v, survivor=%v, old=%v)",
				tt.name, eden, survivor, old, tt.wantEden, tt.wantSurvivor, tt.wantOld)
		}
	}
}
