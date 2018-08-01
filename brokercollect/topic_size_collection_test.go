package brokercollect

import (
	"errors"
	"reflect"
	"testing"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/nri-kafka/testutils"
	"github.com/newrelic/nri-kafka/utils"
)

func TestGatherTopicSize_Single(t *testing.T) {
	testutils.SetupJmxTesting()
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

	utils.JMXQuery = func(query string, timeout int) (map[string]interface{}, error) {
		return map[string]interface{}{
			"one":   float64(1),
			"two":   float64(2),
			"three": float64(3),
			"four":  float64(4),
		}, nil
	}

	collectedTopics := map[string]*metric.Set{
		"topic": e.NewMetricSet("KafkaBrokerSample",
			metric.Attribute{Key: "displayName", Value: "testEntity"},
			metric.Attribute{Key: "entityName", Value: "broker:testEntity"},
			metric.Attribute{Key: "topic", Value: "topic"},
		),
	}

	broker := &broker{
		Host:    "localhost",
		JMXPort: 9999,
		Entity:  e,
	}

	gatherTopicSizes(broker, collectedTopics)

	expected := map[string]interface{}{
		"topic.diskSize": float64(10),
		"event_type":     "KafkaBrokerSample",
		"entityName":     "broker:testEntity",
		"displayName":    "testEntity",
		"topic":          "topic",
	}

	m := broker.Entity.Metrics[0]
	if !reflect.DeepEqual(m.Metrics, expected) {
		t.Errorf("Expected %+v got %+v", expected, m.Metrics)
	}
}

func TestGatherTopicSize_QueryError(t *testing.T) {
	testutils.SetupJmxTesting()
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

	utils.JMXQuery = func(query string, timeout int) (map[string]interface{}, error) { return nil, errors.New("error") }

	collectedTopics := map[string]*metric.Set{
		"topic": e.NewMetricSet("KafkaBrokerSample",
			metric.Attribute{Key: "displayName", Value: "testEntity"},
			metric.Attribute{Key: "entityName", Value: "broker:testEntity"},
			metric.Attribute{Key: "topic", Value: "topic"},
		),
	}

	broker := &broker{
		Host:    "localhost",
		JMXPort: 9999,
		Entity:  e,
	}

	gatherTopicSizes(broker, collectedTopics)

	if _, ok := broker.Entity.Metrics[0].Metrics["topic.diskSize"]; ok {
		t.Error("topic.diskSize metric set was created")
	}
}

func TestGatherTopicSize_QueryBlank(t *testing.T) {
	testutils.SetupJmxTesting()
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

	utils.JMXQuery = func(query string, timeout int) (map[string]interface{}, error) {
		return make(map[string]interface{}), nil
	}

	collectedTopics := map[string]*metric.Set{
		"topic": e.NewMetricSet("KafkaBrokerSample",
			metric.Attribute{Key: "displayName", Value: "testEntity"},
			metric.Attribute{Key: "entityName", Value: "broker:testEntity"},
			metric.Attribute{Key: "topic", Value: "topic"},
		),
	}

	broker := &broker{
		Host:    "localhost",
		JMXPort: 9999,
		Entity:  e,
	}

	gatherTopicSizes(broker, collectedTopics)

	if _, ok := broker.Entity.Metrics[0].Metrics["topic.diskSize"]; ok {
		t.Error("topic.diskSize metric set was created")
	}
}

func TestGatherTopicSize_AggregateError(t *testing.T) {
	testutils.SetupJmxTesting()
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

	utils.JMXQuery = func(query string, timeout int) (map[string]interface{}, error) {
		return map[string]interface{}{
			"one":  "nope",
			"four": float64(4),
		}, nil
	}

	collectedTopics := map[string]*metric.Set{
		"topic": e.NewMetricSet("KafkaBrokerSample",
			metric.Attribute{Key: "displayName", Value: "testEntity"},
			metric.Attribute{Key: "entityName", Value: "broker:testEntity"},
			metric.Attribute{Key: "topic", Value: "topic"},
		),
	}

	broker := &broker{
		Host:    "localhost",
		JMXPort: 9999,
		Entity:  e,
	}

	gatherTopicSizes(broker, collectedTopics)

	if _, ok := broker.Entity.Metrics[0].Metrics["topic.diskSize"]; ok {
		t.Error("topic.diskSize metric set was created")
	}
}
