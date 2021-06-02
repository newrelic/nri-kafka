package broker

import (
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/newrelic/nri-kafka/src/metrics"

	"github.com/Shopify/sarama"
	"github.com/newrelic/infra-integrations-sdk/data/attribute"
	"github.com/newrelic/infra-integrations-sdk/data/inventory"
	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/jmx"
	"github.com/newrelic/nri-kafka/src/connection"
	"github.com/newrelic/nri-kafka/src/connection/mocks"
	"github.com/newrelic/nri-kafka/src/jmxwrapper"
	"github.com/newrelic/nri-kafka/src/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStartBrokerPool(t *testing.T) {
	testutils.SetupTestArgs()

	var wg sync.WaitGroup
	collectedTopics := make([]string, 0)
	i, err := integration.New("kafka", "1.0.0")
	if err != nil {
		t.Error(err)
	}

	brokerChan := StartBrokerPool(3, &wg, i, collectedTopics)
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

func TestBrokerWorker_Exits(t *testing.T) {
	var wg sync.WaitGroup
	brokerChan := make(chan *connection.Broker, 1)
	i, _ := integration.New("kafka", "1.0.0")
	testutils.SetupJmxTesting()
	testutils.SetupTestArgs()

	wg.Add(1)
	close(brokerChan)
	brokerWorker(brokerChan, []string{}, &wg, i)

	finished := make(chan *connection.Broker)
	go func() {
		wg.Wait()
		close(finished)
	}()

	select {
	case <-finished:
	case <-time.After(100 * time.Millisecond):
		assert.Fail(t, "Broker worker did not exit before timeout")
	}

}

func TestPopulateBrokerInventory(t *testing.T) {
	mockBroker := &mocks.SaramaBroker{}
	mockBroker.On("Addr").Return("kafkabroker:9090")
	mockBroker.On("DescribeConfigs", mock.Anything).Return(&sarama.DescribeConfigsResponse{
		Resources: []*sarama.ResourceResponse{
			{
				Type: sarama.BrokerResource,
				Name: "0",
				Configs: []*sarama.ConfigEntry{
					{
						Name:  "leader.replication.throttled.replicas",
						Value: "10000",
					},
				},
			},
		},
	}, nil)

	testBroker := &connection.Broker{
		Host:         "kafkabroker",
		JMXPort:      9999,
		ID:           "0",
		SaramaBroker: mockBroker,
	}
	i, _ := integration.New("kafka", "1.0.0")

	populateBrokerInventory(testBroker, i)

	expectedInventoryItems := map[string]inventory.Item{
		"broker.hostname": {
			"value": testBroker.Host,
		},
		"broker.jmxPort": {
			"value": testBroker.JMXPort,
		},
		"broker.leader.replication.throttled.replicas": {
			"value": "10000",
		},
	}

	for key, item := range expectedInventoryItems {
		entity, _ := testBroker.Entity(i)
		if value, ok := entity.Inventory.Item(key); !ok {
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
	testBroker := &connection.Broker{
		Host:    "kafkabroker",
		JMXPort: 9999,
		ID:      "0",
	}
	i, _ := integration.New("kafka", "1.0.0")

	err := collectBrokerMetrics(testBroker, []string{}, i)
	assert.Equal(t, "jmx error", err.Error())
}

func TestPopulateBrokerMetrics_Normal(t *testing.T) {
	testutils.SetupTestArgs()
	testutils.SetupJmxTesting()

	mockBroker := &mocks.SaramaBroker{}
	mockBroker.On("Addr").Return("kafkabroker:9090")

	testBroker := &connection.Broker{
		Host:         "kafkabroker",
		JMXPort:      9999,
		ID:           "0",
		SaramaBroker: mockBroker,
	}
	i, _ := integration.New("kafka", "1.0.0")

	populateBrokerMetrics(testBroker, i)

	entity, _ := testBroker.Entity(i)
	assert.Len(t, entity.Metrics, 1, "Unexpected number of metrics")

	sample := entity.Metrics[0]

	expected := map[string]interface{}{
		"event_type":  "KafkaBrokerSample",
		"displayName": "kafkabroker:9090",
		"entityName":  "broker:" + "kafkabroker:9090",
		"clusterName": "",
	}

	assert.Equal(t, expected, sample.Metrics)
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

	i, _ := integration.New("test", "1.0.0")
	e, _ := i.Entity(
		"kafkabroker:9090",
		"ka-broker",
		integration.IDAttribute{Key: "clusterName", Value: ""},
		integration.IDAttribute{Key: "brokerID", Value: "0"},
	)

	mockBroker := &mocks.SaramaBroker{}
	mockBroker.On("Addr").Return("kafkabroker:9090")

	testBroker := &connection.Broker{
		Host:         "kafkabroker",
		JMXPort:      9999,
		ID:           "0",
		SaramaBroker: mockBroker,
	}

	sample := e.NewMetricSet("KafkaBrokerSample",
		attribute.Attribute{Key: "clusterName", Value: ""},
		attribute.Attribute{Key: "displayName", Value: "kafkabroker:9090"},
		attribute.Attribute{Key: "entityName", Value: "broker:kafkabroker:9090"},
		attribute.Attribute{Key: "topic", Value: "topic"},
	)

	err := sample.SetMetric("broker.bytesWrittenToTopicPerSecond", float64(0), metric.GAUGE)
	assert.NoError(t, err)

	expected := map[string]*metric.Set{
		"topic": sample,
	}

	out := collectBrokerTopicMetrics(testBroker, []string{"topic"}, i)

	metrics.CollectMetricDefinitions(sample, metrics.BrokerTopicMetricDefs, metrics.ApplyTopicName("topic"))

	assert.Equal(t, expected, out)
}
