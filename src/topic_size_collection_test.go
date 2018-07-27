package main

import (
	"errors"
	"reflect"
	"testing"

	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/nri-kafka/utils"
)

func TestGatherTopicSize_Single(t *testing.T) {
	utils.SetupJmxTesting()
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

	utils.JMXQuery = func(query string, timeout int) (map[string]interface{}, error) {
		return map[string]interface{}{
			"one":   float64(1),
			"two":   float64(2),
			"three": float64(3),
			"four":  float64(4),
		}, nil
	}

	collectedTopics := []string{
		"topic",
	}

	broker := &broker{
		Host:    "localhost",
		JMXPort: 9999,
		Entity:  e,
	}

	gatherTopicSizes(broker, collectedTopics)

	if len(broker.Entity.Metrics) != 1 {
		t.Error("No metric set created")
		t.FailNow()
	}

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
	utils.SetupJmxTesting()
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

	utils.JMXQuery = func(query string, timeout int) (map[string]interface{}, error) { return nil, errors.New("error") }

	collectedTopics := []string{
		"topic",
	}

	broker := &broker{
		Host:    "localhost",
		JMXPort: 9999,
		Entity:  e,
	}

	gatherTopicSizes(broker, collectedTopics)

	if len(broker.Entity.Metrics) != 0 {
		t.Error("Metric set was created")
	}
}

func TestGatherTopicSize_QueryBlank(t *testing.T) {
	utils.SetupJmxTesting()
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

	utils.JMXQuery = func(query string, timeout int) (map[string]interface{}, error) {
		return make(map[string]interface{}), nil
	}

	collectedTopics := []string{
		"topic",
	}

	broker := &broker{
		Host:    "localhost",
		JMXPort: 9999,
		Entity:  e,
	}

	gatherTopicSizes(broker, collectedTopics)

	if len(broker.Entity.Metrics) != 0 {
		t.Error("Metric set was created")
	}
}

func TestGatherTopicSize_JMXOpenFail(t *testing.T) {
	utils.SetupJmxTesting()
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

	utils.JMXOpen = func(hostname, port, username, password string) error { return errors.New("error") }

	collectedTopics := []string{
		"topic",
	}

	broker := &broker{
		Host:    "localhost",
		JMXPort: 9999,
		Entity:  e,
	}

	gatherTopicSizes(broker, collectedTopics)

	if len(broker.Entity.Metrics) != 0 {
		t.Error("Metric set was created")
	}
}

func TestGatherTopicSize_AggregateError(t *testing.T) {
	utils.SetupJmxTesting()
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

	utils.JMXQuery = func(query string, timeout int) (map[string]interface{}, error) {
		return map[string]interface{}{
			"one":  "nope",
			"four": float64(4),
		}, nil
	}

	collectedTopics := []string{
		"topic",
	}

	broker := &broker{
		Host:    "localhost",
		JMXPort: 9999,
		Entity:  e,
	}

	gatherTopicSizes(broker, collectedTopics)

	if len(broker.Entity.Metrics) != 0 {
		t.Error("Metric set was created")
	}
}
