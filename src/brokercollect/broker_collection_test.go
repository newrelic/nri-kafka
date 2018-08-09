package brokercollect

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/kr/pretty"
	"github.com/newrelic/infra-integrations-sdk/data/inventory"
	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/nri-kafka/src/jmxwrapper"
	"github.com/newrelic/nri-kafka/src/testutils"
	"github.com/newrelic/nri-kafka/src/zookeeper"
)

func TestStartBrokerPool(t *testing.T) {
	testutils.SetupTestArgs()

	var wg sync.WaitGroup
	zkConn := zookeeper.MockConnection{}
	collectedTopics := make([]string, 0)
	i, err := integration.New("kafka", "1.0.0")
	if err != nil {
		t.Error(err)
	}

	brokerChan := StartBrokerPool(3, &wg, &zkConn, i, collectedTopics)
	close(brokerChan)

	c := make(chan int)
	go func() {
		wg.Wait()
		c <- 1
	}()

	select {
	case <-c:
	case <-time.After(10 * time.Millisecond):
		t.Error("Wait group did not exit in a reasonable amount of time")
	}
}

func TestBrokerWorker(t *testing.T) {
	zkConn := &zookeeper.MockConnection{}
	var wg sync.WaitGroup
	brokerChan := make(chan int, 10)
	i, _ := integration.New("kafka", "1.0.0")
	testutils.SetupTestArgs()

	wg.Add(1)
	go brokerWorker(brokerChan, []string{}, &wg, zkConn, i)

	brokerChan <- 0
	close(brokerChan)

	wg.Wait()
}

func TestCreateBroker_ZKError(t *testing.T) {
	brokerID, zkConn := 0, &zookeeper.MockConnection{ReturnGetError: true}
	i, _ := integration.New("kafka", "1.0.0")

	_, err := createBroker(brokerID, zkConn, i)
	if err == nil {
		t.Error("Expected error")
	}
}

func TestCreateBroker_Normal(t *testing.T) {
	brokerID, zkConn := 0, &zookeeper.MockConnection{}
	i, _ := integration.New("kafka", "1.0.0")

	b, err := createBroker(brokerID, zkConn, i)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}

	expectedBroker := &broker{
		Host:      "kafkabroker",
		KafkaPort: 9092,
		JMXPort:   9999,
		ID:        0,
	}

	if expectedBroker.Host != b.Host {
		t.Errorf("Expected JMX host '%s' got '%s'", expectedBroker.Host, b.Host)
	}
	if expectedBroker.JMXPort != b.JMXPort {
		t.Errorf("Expected JMX Port '%d' got '%d'", expectedBroker.JMXPort, b.JMXPort)
	}
	if expectedBroker.KafkaPort != b.KafkaPort {
		t.Errorf("Expected JMX Port '%d' got '%d'", expectedBroker.KafkaPort, b.KafkaPort)
	}
	if b.Entity.Metadata.Name != b.Host {
		t.Errorf("Expected entity name '%s' got '%s'", expectedBroker.Host, expectedBroker.Entity.Metadata.Name)
	}
	if b.Entity.Metadata.Namespace != "broker" {
		t.Errorf("Expected entity name '%s' got '%s'", "broker", expectedBroker.Entity.Metadata.Name)
	}
}

func TestPopulateBrokerInventory(t *testing.T) {
	testBroker := &broker{
		Host:      "kafkabroker",
		JMXPort:   9999,
		KafkaPort: 9092,
		ID:        0,
		Config:    map[string]string{"leader.replication.throttled.replicas": "10000"},
	}
	i, _ := integration.New("kafka", "1.0.0")

	testBroker.Entity, _ = i.Entity("brokerHost", "broker")

	if err := populateBrokerInventory(testBroker); err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}

	expectedInventoryItems := map[string]inventory.Item{
		"broker.hostname": {
			"value": testBroker.Host,
		},
		"broker.jmxPort": {
			"value": testBroker.JMXPort,
		},
		"broker.kafkaPort": {
			"value": testBroker.KafkaPort,
		},
		"broker.leader.replication.throttled.replicas": {
			"value": "10000",
		},
	}

	for key, item := range expectedInventoryItems {
		if value, ok := testBroker.Entity.Inventory.Item(key); !ok {
			t.Errorf("Entity missing Inventory Key: %s", key)
		} else if !reflect.DeepEqual(item, value) {
			t.Errorf("Expected Item %+v got %+v", item, value)
		}
	}
}

func TestPopulateBrokerMetrics_JMXOpenError(t *testing.T) {
	testutils.SetupTestArgs()
	testutils.SetupJmxTesting()
	errorText := "jmx error"

	jmxwrapper.JMXOpen = func(hostname, port, username, password string) error { return errors.New(errorText) }
	testBroker := &broker{
		Host:      "kafkabroker",
		JMXPort:   9999,
		KafkaPort: 9092,
		ID:        0,
	}
	i, _ := integration.New("kafka", "1.0.0")

	testBroker.Entity, _ = i.Entity(testBroker.Host, "broker")

	err := collectBrokerMetrics(testBroker, []string{})
	if err == nil {
		t.Error("Did not get expected error")
	} else if err.Error() != errorText {
		t.Errorf("Expected error '%s' got '%s'", errorText, err.Error())
	}
}

func TestPopulateBrokerMetrics_Normal(t *testing.T) {
	testutils.SetupTestArgs()
	testutils.SetupJmxTesting()

	testBroker := &broker{
		Host:      "kafkabroker",
		JMXPort:   9999,
		KafkaPort: 9092,
		ID:        0,
	}
	i, _ := integration.New("kafka", "1.0.0")

	testBroker.Entity, _ = i.Entity(testBroker.Host, "broker")

	populateBrokerMetrics(testBroker)

	// MetricSet should still be created during a failed query.
	if len(testBroker.Entity.Metrics) != 1 {
		t.Errorf("Expected one metric set got %d", len(testBroker.Entity.Metrics))
	}

	sample := testBroker.Entity.Metrics[0]

	expected := map[string]interface{}{
		"event_type":  "KafkaBrokerSample",
		"displayName": testBroker.Host,
		"entityName":  "broker:" + testBroker.Host,
	}

	if !reflect.DeepEqual(sample.Metrics, expected) {
		fmt.Println(pretty.Diff(sample.Metrics, expected))
		t.Errorf("Expected Item %+v got %+v", expected, sample.Metrics)
	}
}

func TestGetBrokerJMX(t *testing.T) {
	brokerID := 0
	zkConn := zookeeper.MockConnection{}

	host, jmxPort, kafkaPort, err := GetBrokerConnectionInfo(brokerID, &zkConn)
	if err != nil {
		t.Error(err)
	}

	if host != "kafkabroker" {
		t.Errorf("Expected host kafkabroker, got %s", host)
	}
	if kafkaPort != 9092 {
		t.Errorf("Expected kafka port 9092, got %d", kafkaPort)
	}
	if jmxPort != 9999 {
		t.Errorf("Expected jmx port 9999, got %d", jmxPort)
	}
}

func TestGetBrokerConfig(t *testing.T) {
	zkConn := zookeeper.MockConnection{}

	testCases := []struct {
		brokerID       int
		expectedConfig map[string]string
	}{
		{0, map[string]string{"leader.replication.throttled.replicas": "10000"}},
		{10, map[string]string{}},
	}

	for _, tc := range testCases {
		brokerConfig, err := getBrokerConfig(tc.brokerID, &zkConn)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}

		if !reflect.DeepEqual(tc.expectedConfig, brokerConfig) {
			t.Errorf("Expected broker config %s, got %s", tc.expectedConfig, brokerConfig)
		}
	}

	return
}

func TestCollectBrokerTopicMetrics(t *testing.T) {
	testutils.SetupTestArgs()
	testutils.SetupJmxTesting()

	jmxwrapper.JMXQuery = func(query string, timeout int) (map[string]interface{}, error) {
		result := map[string]interface{}{
			"kafka.server:type=BrokerTopicMetrics,name=BytesInPerSec,topic=topic,attr=Count": 24,
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

	testBroker := &broker{
		Host:      "kafkabroker",
		JMXPort:   9999,
		KafkaPort: 9092,
		ID:        0,
		Entity:    e,
	}

	sample := e.NewMetricSet("KafkaBrokerSample",
		metric.Attribute{Key: "displayName", Value: "testEntity"},
		metric.Attribute{Key: "entityName", Value: "broker:testEntity"},
		metric.Attribute{Key: "topic", Value: "topic"})

	sample.SetMetric("topic.bytesWritten", float64(24), metric.GAUGE)

	expected := map[string]*metric.Set{
		"topic": sample,
	}

	out := collectBrokerTopicMetrics(testBroker, []string{"topic"})

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("Expected %+v got %+v", expected, out)
	}
}
