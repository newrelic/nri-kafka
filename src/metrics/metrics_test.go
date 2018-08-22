package metrics

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/nri-kafka/src/jmxwrapper"
	"github.com/newrelic/nri-kafka/src/testutils"
)

func TestGetBrokerMetrics(t *testing.T) {
	expected := map[string]interface{}{
		"request.avgTimeFetch": float64(24),
		"event_type":           "testMetrics",
	}

	jmxwrapper.JMXQuery = func(query string, timeout int) (map[string]interface{}, error) {
		result := map[string]interface{}{
			"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Fetch,attr=Mean": 24,
		}

		return result, nil
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

	GetBrokerMetrics(m)

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

	jmxwrapper.JMXQuery = func(query string, timeout int) (map[string]interface{}, error) {
		result := map[string]interface{}{
			"kafka.consumer:type=consumer-fetch-manager-metrics,client-id=" + consumerName + ",attr=records-lag-max": 24,
		}

		return result, nil
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

	GetConsumerMetrics(consumerName, m)

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

	jmxwrapper.JMXQuery = func(query string, timeout int) (map[string]interface{}, error) {
		result := map[string]interface{}{
			"kafka.producer:type=producer-metrics,client-id=" + producerName + ",attr=metadata-age": 24,
		}

		return result, nil
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

	GetProducerMetrics(producerName, m)

	if !reflect.DeepEqual(expected, m.Metrics) {
		t.Errorf("Expected %+v got %+v", expected, m.Metrics)
	}
}

func TestCollectMetricDefinitions_QueryError(t *testing.T) {
	testutils.SetupTestArgs()
	errString := "this is an error"

	jmxwrapper.JMXQuery = func(query string, timeout int) (map[string]interface{}, error) {
		return nil, errors.New(errString)
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

	CollectMetricDefintions(m, brokerMetricDefs, nil)

	if len(m.Metrics) != 1 {
		t.Error("Metrics where inserted even with a bad query")
	}
}

func TestCollectMetricDefinitions_MetricError(t *testing.T) {
	testutils.SetupTestArgs()
	expected := map[string]interface{}{
		"event_type": "testMetrics",
	}

	jmxwrapper.JMXQuery = func(query string, timeout int) (map[string]interface{}, error) {
		result := map[string]interface{}{
			"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Fetch,attr=Mean": "stuff",
		}

		return result, nil
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

	CollectMetricDefintions(m, brokerMetricDefs, nil)

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

	jmxwrapper.JMXQuery = func(query string, timeout int) (map[string]interface{}, error) {
		if query != expectedBean {
			return nil, fmt.Errorf("expected bean '%s' got '%s'", expectedBean, query)
		}

		result := map[string]interface{}{
			"kafka.network:replace=Replaced,attr=Metric": 24,
		}

		return result, nil
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

	CollectMetricDefintions(m, testMetricSet, renameFunc("Replaced"))

	if !reflect.DeepEqual(expected, m.Metrics) {
		t.Errorf("Expected %+v got %+v", expected, m.Metrics)
	}
}
