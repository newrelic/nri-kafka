package broker

import (
	"testing"

	"github.com/newrelic/nrjmx/gojmx"

	"github.com/newrelic/infra-integrations-sdk/v3/data/attribute"
	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/nri-kafka/src/connection"
	"github.com/newrelic/nri-kafka/src/connection/mocks"
	"github.com/newrelic/nri-kafka/src/testutils"
	"github.com/stretchr/testify/assert"
)

func TestGatherTopicSize_Single(t *testing.T) {
	testutils.SetupTestArgs()

	i, _ := integration.New("test", "1.0.0")

	mockResponse := &mocks.MockJMXResponse{
		Result: []*gojmx.AttributeResponse{
			{
				Name:         "one",
				ResponseType: gojmx.ResponseTypeString,
				StringValue:  "1",
			},
			{
				Name:         "two",
				ResponseType: gojmx.ResponseTypeDouble,
				DoubleValue:  2,
			},
			{
				Name:         "three",
				ResponseType: gojmx.ResponseTypeString,
				StringValue:  "3",
			},
			{
				Name:         "four",
				ResponseType: gojmx.ResponseTypeDouble,
				DoubleValue:  4,
			},
		},
	}

	mockJMXProvider := &mocks.MockJMXProvider{
		Response: mockResponse,
	}

	mockBroker := &mocks.SaramaBroker{}
	mockBroker.On("Addr").Return("kafkabroker:9090")

	broker := &connection.Broker{
		Host:         "localhost",
		JMXPort:      9999,
		SaramaBroker: mockBroker,
	}

	e, _ := broker.Entity(i)
	collectedTopics := map[string]*metric.Set{
		"topic": e.NewMetricSet("KafkaBrokerSample",
			attribute.Attribute{Key: "displayName", Value: "testEntity"},
			attribute.Attribute{Key: "entityName", Value: "broker:testEntity"},
			attribute.Attribute{Key: "topic", Value: "topic"},
		),
	}

	gatherTopicSizes(broker, collectedTopics, i, mockJMXProvider)

	expected := map[string]interface{}{
		"topic.diskSize": float64(10),
		"event_type":     "KafkaBrokerSample",
		"entityName":     "broker:testEntity",
		"displayName":    "testEntity",
		"topic":          "topic",
	}

	entity, err := broker.Entity(i)
	assert.NoError(t, err)
	assert.Len(t, entity.Metrics, 1)
	assert.Equal(t, expected, entity.Metrics[0].Metrics)
}

func TestGatherTopicSize_QueryError(t *testing.T) {
	testutils.SetupTestArgs()

	i, _ := integration.New("test", "1.0.0")

	mockResponse := &mocks.MockJMXResponse{
		Err: errJMX,
	}

	mockJMXProvider := &mocks.MockJMXProvider{
		Response: mockResponse,
	}

	mockBroker := &mocks.SaramaBroker{}
	mockBroker.On("Addr").Return("kafkabroker:9090")

	broker := &connection.Broker{
		Host:         "localhost",
		JMXPort:      9999,
		SaramaBroker: mockBroker,
	}

	e, _ := broker.Entity(i)

	collectedTopics := map[string]*metric.Set{
		"topic": e.NewMetricSet("KafkaBrokerSample",
			attribute.Attribute{Key: "displayName", Value: "testEntity"},
			attribute.Attribute{Key: "entityName", Value: "broker:testEntity"},
			attribute.Attribute{Key: "topic", Value: "topic"},
		),
	}

	gatherTopicSizes(broker, collectedTopics, i, mockJMXProvider)

	assert.Len(t, e.Metrics, 1)
	assert.NotContains(t, e.Metrics[0].Metrics, "topic.diskSize", "Metric was unexpectedly set after query error")
}

func TestGatherTopicSize_QueryBlank(t *testing.T) {
	testutils.SetupTestArgs()

	i, _ := integration.New("test", "1.0.0")

	mockBroker := &mocks.SaramaBroker{}
	mockBroker.On("Addr").Return("kafkabroker:9090")

	broker := &connection.Broker{
		Host:         "localhost",
		JMXPort:      9999,
		SaramaBroker: mockBroker,
	}

	e, _ := broker.Entity(i)

	collectedTopics := map[string]*metric.Set{
		"topic": e.NewMetricSet("KafkaBrokerSample",
			attribute.Attribute{Key: "displayName", Value: "testEntity"},
			attribute.Attribute{Key: "entityName", Value: "broker:testEntity"},
			attribute.Attribute{Key: "topic", Value: "topic"},
		),
	}
	conn, _ := mocks.NewEmptyMockJMXProvider().NewConnection(nil)
	gatherTopicSizes(broker, collectedTopics, i, conn)

	assert.Len(t, e.Metrics, 1)
	assert.NotContains(t, e.Metrics[0].Metrics, "topic.diskSize", "Metric was unexpectedly set after empty query result")
}

func TestGatherTopicSize_AggregateError(t *testing.T) {
	testutils.SetupTestArgs()

	i, _ := integration.New("test", "1.0.0")

	mockResponse := &mocks.MockJMXResponse{
		Result: []*gojmx.AttributeResponse{
			{
				Name:         "one",
				ResponseType: gojmx.ResponseTypeString,
				StringValue:  "nope",
			},
			{
				Name:         "four",
				ResponseType: gojmx.ResponseTypeDouble,
				DoubleValue:  float64(4),
			},
		},
	}

	mockJMXProvider := &mocks.MockJMXProvider{
		Response: mockResponse,
	}

	mockBroker := &mocks.SaramaBroker{}
	mockBroker.On("Addr").Return("kafkabroker:9090")

	broker := &connection.Broker{
		Host:         "localhost",
		JMXPort:      9999,
		SaramaBroker: mockBroker,
	}

	e, _ := broker.Entity(i)

	collectedTopics := map[string]*metric.Set{
		"topic": e.NewMetricSet("KafkaBrokerSample",
			attribute.Attribute{Key: "displayName", Value: "testEntity"},
			attribute.Attribute{Key: "entityName", Value: "broker:testEntity"},
			attribute.Attribute{Key: "topic", Value: "topic"},
		),
	}

	gatherTopicSizes(broker, collectedTopics, i, mockJMXProvider)

	assert.Len(t, e.Metrics, 1)
	assert.NotContains(t, e.Metrics[0].Metrics, "topic.diskSize", "Metric was unexpectedly set after aggregate error")
}
