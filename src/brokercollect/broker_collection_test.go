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
	"github.com/newrelic/infra-integrations-sdk/jmx"
	"github.com/newrelic/nri-kafka/src/jmxwrapper"
	"github.com/newrelic/nri-kafka/src/testutils"
	"github.com/newrelic/nri-kafka/src/zookeeper"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/stretchr/testify/assert"
)

var (
	brokerConnectionBytes = []byte(`{"listener_security_protocol_map":{"PLAINTEXT":"PLAINTEXT", "SSL":"SLL"},"endpoints":["PLAINTEXT://kafkabroker:9092", "SSL://kafkabroker:9093"],"jmx_port":9999,"host":"kafkabroker","timestamp":"1530886155628","port":9092,"version":4}`)
	brokerConfigBytes     = []byte(`{"version":1,"config":{"flush.messages":"12345"}}`)
	brokerConfigBytes2    = []byte(`{"version":1,"config":{"leader.replication.throttled.replicas":"10000"}}`)
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
	zkConn.On("Get", "/brokers/ids/0").Return(brokerConnectionBytes, new(zk.Stat), nil)
	zkConn.On("Get", "/config/brokers/0").Return(brokerConfigBytes, new(zk.Stat), nil)

	var wg sync.WaitGroup
	brokerChan := make(chan int, 10)
	i, _ := integration.New("kafka", "1.0.0")
	testutils.SetupJmxTesting()
	testutils.SetupTestArgs()

	wg.Add(1)
	brokerChan <- 0
	close(brokerChan)
	brokerWorker(brokerChan, []string{}, &wg, zkConn, i)

	wg.Wait()
}

func TestCreateBroker_ZKError(t *testing.T) {
	brokerID, zkConn := 0, &zookeeper.MockConnection{}
	zkConn.On("Get", "/brokers/ids/0").Return([]byte{}, new(zk.Stat), errors.New("this is a test error"))
	i, _ := integration.New("kafka", "1.0.0")

	_, err := createBrokers(brokerID, zkConn, i)
	if err == nil {
		t.Error("Expected error")
	}
}

func TestCreateBroker_Normal(t *testing.T) {
	brokerID, zkConn := 0, &zookeeper.MockConnection{}
	zkConn.On("Get", "/brokers/ids/0").Return(brokerConnectionBytes, new(zk.Stat), nil)
	zkConn.On("Get", "/config/brokers/0").Return(brokerConfigBytes, new(zk.Stat), nil)
	i, _ := integration.New("kafka", "1.0.0")

	brokers, err := createBrokers(brokerID, zkConn, i)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}

	expectedBrokers := []broker{
		{
			Host:      "kafkabroker",
			KafkaPort: 9092,
			JMXPort:   9999,
			ID:        0,
		},
		{
			Host:      "kafkabroker",
			KafkaPort: 9093,
			JMXPort:   9999,
			ID:        0,
		},
	}

	for i, broker := range brokers {
		if expectedBrokers[i].Host != broker.Host {
			t.Errorf("Expected JMX host '%s' got '%s'", expectedBrokers[i].Host, broker.Host)
		}
		if expectedBrokers[i].JMXPort != broker.JMXPort {
			t.Errorf("Expected JMX Port '%d' got '%d'", expectedBrokers[i].JMXPort, broker.JMXPort)
		}
		if expectedBrokers[i].KafkaPort != broker.KafkaPort {
			t.Errorf("Expected Kafka Port '%d' got '%d'", expectedBrokers[i].KafkaPort, broker.KafkaPort)
		}
		metadataName := fmt.Sprintf("%s:%d", expectedBrokers[i].Host, expectedBrokers[i].KafkaPort)
		if broker.Entity.Metadata.Name != metadataName {
			t.Errorf("Expected entity name '%s' got '%s'", metadataName, broker.Entity.Metadata.Name)
		}
		if broker.Entity.Metadata.Namespace != "ka-broker" {
			t.Errorf("Expected entity name '%s' got '%s'", "ka-broker", broker.Entity.Metadata.Namespace)
		}
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

	testBroker.Entity, _ = i.Entity("brokerHost", "ka-broker")

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

	jmxwrapper.JMXOpen = func(hostname, port, username, password string, options ...jmx.Option) error {
		return errors.New(errorText)
	}
	testBroker := &broker{
		Host:      "kafkabroker",
		JMXPort:   9999,
		KafkaPort: 9092,
		ID:        0,
	}
	i, _ := integration.New("kafka", "1.0.0")

	testBroker.Entity, _ = i.Entity(testBroker.Host, "ka-broker")

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

	testBroker.Entity, _ = i.Entity(testBroker.Host, "ka-broker")

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
	testutils.SetupTestArgs()

	brokerID := 0
	zkConn := zookeeper.MockConnection{}
	zkConn.On("Get", "/brokers/ids/0").Return(brokerConnectionBytes, new(zk.Stat), nil)
	expectedBrokers := []zookeeper.BrokerConnection{
		{
			Scheme:     "http",
			BrokerHost: "kafkabroker",
			BrokerPort: 9092,
			JmxPort:    9999,
		},
		{
			Scheme:     "https",
			BrokerHost: "kafkabroker",
			BrokerPort: 9093,
			JmxPort:    9999,
		},
	}

	brokerConnections, err := zookeeper.GetBrokerConnectionInfo(brokerID, &zkConn)
	if err != nil {
		t.Error(err)
	}

	for i, brokerConnection := range brokerConnections {
		if brokerConnection.Scheme != expectedBrokers[i].Scheme {
			t.Errorf("Expected scheme %s, got %s", expectedBrokers[i].Scheme, brokerConnection.Scheme)
		}
		if brokerConnection.BrokerHost != expectedBrokers[i].BrokerHost {
			t.Errorf("Expected host %s, got %s", expectedBrokers[i].BrokerHost, brokerConnection.BrokerHost)
		}
		if brokerConnection.BrokerPort != expectedBrokers[i].BrokerPort {
			t.Errorf("Expected kafka port %d, got %d", expectedBrokers[i].BrokerPort, brokerConnection.BrokerPort)
		}
		if brokerConnection.JmxPort != expectedBrokers[i].JmxPort {
			t.Errorf("Expected jmx port %d, got %d", expectedBrokers[i].JmxPort, brokerConnection.JmxPort)
		}
	}
}

func TestGetBrokerConfig(t *testing.T) {

	testCases := []struct {
		brokerID       int
		expectedConfig map[string]string
		expectedError  bool
	}{
		{0, map[string]string{"leader.replication.throttled.replicas": "10000"}, false},
		{1, nil, true},
	}

	for _, tc := range testCases {
		zkConn := zookeeper.MockConnection{}
		zkConn.On("Get", "/config/brokers/0").Return(brokerConfigBytes2, new(zk.Stat), nil)
		zkConn.On("Get", "/config/brokers/1").Return([]byte{}, new(zk.Stat), errors.New("this is a test error"))

		brokerConfig, err := getBrokerConfig(tc.brokerID, &zkConn)
		if (err != nil) != tc.expectedError {
			t.Error(err)
			t.FailNow()
		}

		assert.Equal(t, tc.expectedConfig, brokerConfig)
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

	sample.SetMetric("broker.bytesWrittenToTopicPerSecond", float64(0), metric.GAUGE)

	expected := map[string]*metric.Set{
		"topic": sample,
	}

	out := collectBrokerTopicMetrics(testBroker, []string{"topic"})

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("Expected %+v got %+v", expected, out)
	}
}
