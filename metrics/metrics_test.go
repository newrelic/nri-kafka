package metrics

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/nri-kafka/utils"
)

func TestGetBrokerMetrics(t *testing.T) {
	expected := map[string]interface{}{
		"request.avgTimeFetch": float64(24),
		"event_type":           "testMetrics",
	}

	utils.JMXQuery = func(query string, timeout int) (map[string]interface{}, error) {
		result := map[string]interface{}{
			"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Fetch,attr=Mean": 24,
		}

		return result, nil
	}

	utils.SetupTestArgs()

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

	if err := GetBrokerMetrics(m); err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	if !reflect.DeepEqual(expected, m.Metrics) {
		t.Errorf("Expected %+v got %+v", expected, m.Metrics)
	}
}

func TestCollectMetricDefinitions_QueryError(t *testing.T) {
	errString := "this is an error"

	utils.JMXQuery = func(query string, timeout int) (map[string]interface{}, error) {
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

	if err := collectMetricDefintions(m, brokerMetricDefs, nil); err == nil {
		t.Error("Did not get error when expected")
	} else if err.Error() != errString {
		t.Errorf("Unexpected error %s", err.Error())
	}
}

func TestCollectMetricDefinitions_MetricError(t *testing.T) {
	expected := map[string]interface{}{
		"event_type": "testMetrics",
	}

	utils.JMXQuery = func(query string, timeout int) (map[string]interface{}, error) {
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

	if err := collectMetricDefintions(m, brokerMetricDefs, nil); err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	if !reflect.DeepEqual(expected, m.Metrics) {
		t.Errorf("Expected %+v got %+v", expected, m.Metrics)
	}
}

func TestCollectMetricDefinitions_BeanModifier(t *testing.T) {
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

	utils.JMXQuery = func(query string, timeout int) (map[string]interface{}, error) {
		if query != expectedBean {
			return nil, fmt.Errorf("Expected bean '%s' got '%s'", expectedBean, query)
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

	if err := collectMetricDefintions(m, testMetricSet, renameFunc("Replaced")); err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	if !reflect.DeepEqual(expected, m.Metrics) {
		t.Errorf("Expected %+v got %+v", expected, m.Metrics)
	}
}
