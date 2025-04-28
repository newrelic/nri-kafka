package metrics

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/newrelic/nri-kafka/src/connection/mocks"
	"github.com/newrelic/nrjmx/gojmx"

	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/nri-kafka/src/testutils"
)

var (
	errTest = errors.New("this is an error")
)

func TestGetBrokerMetrics(t *testing.T) {
	expected := map[string]interface{}{
		"request.avgTimeFetch": float64(24),
		"event_type":           "testMetrics",
	}

	mockResponse := &mocks.MockJMXResponse{
		Err: nil,
		Result: []*gojmx.AttributeResponse{
			{
				Name:         "kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Fetch,attr=Mean",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     24,
			},
		},
	}

	mockJMXProvider := &mocks.MockJMXProvider{
		Response: mockResponse,
	}

	testutils.SetupTestArgs()

	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	e, err := i.Entity("testEntity", "testNamespace")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	m := e.NewMetricSet("testMetrics")

	GetBrokerMetrics(m, mockJMXProvider)

	if !reflect.DeepEqual(expected, m.Metrics) {
		t.Errorf("Expected %+v got %+v", expected, m.Metrics)
	}
}

func TestGetConsumerMetrics(t *testing.T) {
	expected := map[string]interface{}{
		"consumer.maxLag": float64(24),
		"event_type":      "testMetrics",
	}

	consumerName := "consumer"

	mockResponse := &mocks.MockJMXResponse{
		Err: nil,
		Result: []*gojmx.AttributeResponse{
			{
				Name:         "kafka.consumer:type=consumer-fetch-manager-metrics,client-id=" + consumerName + ",attr=records-lag-max",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     24,
			},
		},
	}

	mockJMXProvider := &mocks.MockJMXProvider{
		Response: mockResponse,
	}

	testutils.SetupTestArgs()

	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	e, err := i.Entity("testEntity", "testNamespace")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	m := e.NewMetricSet("testMetrics")

	GetConsumerMetrics(consumerName, m, mockJMXProvider)

	if !reflect.DeepEqual(expected, m.Metrics) {
		t.Errorf("Expected %+v got %+v", expected, m.Metrics)
	}
}

func TestGetProducerMetrics(t *testing.T) {
	expected := map[string]interface{}{
		"producer.ageMetadataUsedInMilliseconds": float64(24),
		"event_type":                             "testMetrics",
	}

	producerName := "producer"

	mockResponse := &mocks.MockJMXResponse{
		Err: nil,
		Result: []*gojmx.AttributeResponse{
			{
				Name:         "kafka.producer:type=producer-metrics,client-id=" + producerName + ",attr=metadata-age",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     24,
			},
		},
	}

	mockJMXProvider := &mocks.MockJMXProvider{
		Response: mockResponse,
	}

	testutils.SetupTestArgs()

	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	e, err := i.Entity("testEntity", "testNamespace")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	m := e.NewMetricSet("testMetrics")

	GetProducerMetrics(producerName, m, mockJMXProvider)

	if !reflect.DeepEqual(expected, m.Metrics) {
		t.Errorf("Expected %+v got %+v", expected, m.Metrics)
	}
}

func TestCollectMetricDefinitions_QueryError(t *testing.T) {
	testutils.SetupTestArgs()

	mockResponse := &mocks.MockJMXResponse{
		Err: errTest,
	}

	mockJMXProvider := &mocks.MockJMXProvider{
		Response: mockResponse,
	}

	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	e, err := i.Entity("testEntity", "testNamespace")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	m := e.NewMetricSet("testMetrics")

	CollectMetricDefinitions(m, brokerMetricDefs, nil, mockJMXProvider)

	if len(m.Metrics) != 1 {
		t.Error("Metrics where inserted even with a bad query")
	}
}

func TestCollectMetricDefinitions_MetricError(t *testing.T) {
	testutils.SetupTestArgs()
	expected := map[string]interface{}{
		"event_type": "testMetrics",
	}

	mockResponse := &mocks.MockJMXResponse{
		Err: nil,
		Result: []*gojmx.AttributeResponse{
			{
				Name:         "kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Fetch,attr=Mean",
				ResponseType: gojmx.ResponseTypeString,
				StringValue:  "stuff",
			},
		},
	}

	mockJMXProvider := &mocks.MockJMXProvider{
		Response: mockResponse,
	}

	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	e, err := i.Entity("testEntity", "testNamespace")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	m := e.NewMetricSet("testMetrics")

	CollectMetricDefinitions(m, brokerMetricDefs, nil, mockJMXProvider)

	if !reflect.DeepEqual(expected, m.Metrics) {
		t.Errorf("Expected %+v got %+v", expected, m.Metrics)
	}
}

func TestCollectMetricDefinitions_BeanModifier(t *testing.T) {
	testutils.SetupTestArgs()
	testMetricSet := []*JMXMetricSet{
		{
			MBean:        "kafka.network:replace=%REPLACE_ME%",
			MetricPrefix: "kafka.network:replace=%REPLACE_ME%,",
			MetricDefs: []*MetricDefinition{
				{
					Name:       "my.metric",
					SourceType: metric.GAUGE,
					JMXAttr:    "attr=Metric",
				},
			},
		},
	}

	expectedBean := "kafka.network:replace=Replaced"

	mockResponse := &mocks.MockJMXResponse{
		Err: nil,
		Result: []*gojmx.AttributeResponse{
			{
				Name:         "kafka.network:replace=Replaced,attr=Metric",
				ResponseType: gojmx.ResponseTypeDouble,
				DoubleValue:  float64(24),
			},
		},
	}

	mockJMXProvider := &mocks.MockJMXProvider{
		Response:         mockResponse,
		MBeanNamePattern: expectedBean,
	}

	expected := map[string]interface{}{
		"my.metric":  float64(24),
		"event_type": "testMetrics",
	}

	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	e, err := i.Entity("testEntity", "testNamespace")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	m := e.NewMetricSet("testMetrics")

	renameFunc := func(replaceName string) func(string) string {
		return func(bean string) string {
			return strings.Replace(bean, "%REPLACE_ME%", replaceName, -1)
		}
	}

	CollectMetricDefinitions(m, testMetricSet, renameFunc("Replaced"), mockJMXProvider)

	if !reflect.DeepEqual(expected, m.Metrics) {
		t.Errorf("Expected %+v got %+v", expected, m.Metrics)
	}
}
