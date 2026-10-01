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
		t.Error("jvm.heapMemoryUsedBytes should not be collected when CollectBrokerExtendedMetrics is false")
	}
	if _, ok := m.Metrics["jvm.gcCollectionsPerSecond"]; ok {
		t.Error("jvm.gcCollectionsPerSecond should not be collected when CollectBrokerExtendedMetrics is false")
	}
}

func TestGetBrokerMetrics_JVMMetricsEnabled(t *testing.T) {
	testutils.SetupTestArgs()
	args.GlobalArgs.CollectBrokerExtendedMetrics = true

	mockJMXProvider := &mocks.MockJMXProvider{Response: jvmMockResponse()}

	i, _ := integration.New("test", "1.0.0")
	e, _ := i.Entity("jvmEnabledEntity", "jvmEnabledNamespace")
	m := e.NewMetricSet("testMetrics")

	GetBrokerMetrics(m, mockJMXProvider)

	if got := m.Metrics["jvm.heapMemoryUsedBytes"]; got != float64(1000) {
		t.Errorf("expected jvm.heapMemoryUsedBytes = 1000, got %v", got)
	}
	if _, ok := m.Metrics["jvm.gcCollectionsPerSecond"]; !ok {
		t.Error("expected jvm.gcCollectionsPerSecond to be collected when CollectBrokerExtendedMetrics is true")
	}
}

func TestCollectGarbageCollectorMetrics_SumsAcrossCollectors(t *testing.T) {
	testutils.SetupTestArgs()

	// Name-before-type ordering, matching a real GarbageCollector MBean - see jvmMBeanNameRegex.
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
				// Matches neither bucket in classifyGCGeneration; only the flat totals count it.
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

	// RATE metrics report 0 on a fresh entity's first sample.
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
