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
				Name:         "java.lang:type=Memory,attr=HeapMemoryUsage.used",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     1000,
			},
			{
				Name:         "java.lang:type=Threading,attr=ThreadCount",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     120,
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
	if _, ok := m.Metrics["jvm.gcCollectionsPerSecond"]; !ok {
		t.Error("expected jvm.gcCollectionsPerSecond to be collected when EnableBrokerJVMMetrics is true")
	}
}

func TestCollectGarbageCollectorMetrics_SumsAcrossCollectors(t *testing.T) {
	testutils.SetupTestArgs()

	mockResponse := &mocks.MockJMXResponse{
		Result: []*gojmx.AttributeResponse{
			{
				Name:         "java.lang:type=GarbageCollector,name=G1 Young Generation,attr=CollectionCount",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     10,
			},
			{
				Name:         "java.lang:type=GarbageCollector,name=G1 Old Generation,attr=CollectionCount",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     2,
			},
			{
				Name:         "java.lang:type=GarbageCollector,name=G1 Young Generation,attr=CollectionTime",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     500,
			},
			{
				Name:         "java.lang:type=GarbageCollector,name=G1 Old Generation,attr=CollectionTime",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     300,
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

	expected := map[string]interface{}{
		"event_type":                 "testMetrics",
		"jvm.gcCollectionsPerSecond": float64(0),
		"jvm.gcTimePerSecond":        float64(0),
	}

	if !reflect.DeepEqual(expected, m.Metrics) {
		t.Errorf("Expected %+v got %+v", expected, m.Metrics)
	}
}
