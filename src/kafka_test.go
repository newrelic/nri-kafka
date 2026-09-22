package main

import (
	"reflect"
	"testing"

	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/connection"
	"github.com/newrelic/nri-kafka/src/connection/mocks"
	"github.com/newrelic/nri-kafka/src/testutils"
	"github.com/newrelic/nri-kafka/src/topic"
	"github.com/newrelic/nrjmx/gojmx"
	"github.com/stretchr/testify/assert"
)

func TestCountActiveControllers_SumsAcrossBrokers(t *testing.T) {
	testutils.SetupTestArgs()

	mockResponse := &mocks.MockJMXResponse{
		Result: []*gojmx.AttributeResponse{
			{
				Name:         "kafka.controller:type=KafkaController,name=ActiveControllerCount,attr=Value",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     1,
			},
		},
	}
	mockJMXProvider := &mocks.MockJMXProvider{Response: mockResponse}

	brokers := []*connection.Broker{
		{Host: "broker1", JMXPort: 9999},
		{Host: "broker2", JMXPort: 9999},
		{Host: "broker3", JMXPort: 9999},
	}

	count := countActiveControllers(brokers, mockJMXProvider)

	assert.Equal(t, 3, count)
}

func TestCollectTopicByteRates_SumsAcrossBrokersIncludingInternalTopics(t *testing.T) {
	testutils.SetupTestArgs()

	mockResponse := &mocks.MockJMXResponse{
		Result: []*gojmx.AttributeResponse{
			{
				Name:         "kafka.server:type=BrokerTopicMetrics,name=BytesInPerSec,topic=orders,attr=Count",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     100,
			},
			{
				Name:         "kafka.server:type=BrokerTopicMetrics,name=BytesInPerSec,topic=orders,attr=OneMinuteRate",
				ResponseType: gojmx.ResponseTypeDouble,
				DoubleValue:  1.5,
			},
			{
				Name:         "kafka.server:type=BrokerTopicMetrics,name=BytesOutPerSec,topic=orders,attr=Count",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     50,
			},
			{
				Name:         "kafka.server:type=BrokerTopicMetrics,name=BytesInPerSec,topic=__consumer_offsets,attr=Count",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     999999,
			},
		},
	}
	mockJMXProvider := &mocks.MockJMXProvider{Response: mockResponse}

	brokers := []*connection.Broker{
		{Host: "broker1", JMXPort: 9999},
		{Host: "broker2", JMXPort: 9999},
	}

	rates := collectTopicByteRates(brokers, mockJMXProvider)

	assert.Equal(t, topic.ByteRates{BytesInPerSecond: 200, BytesOutPerSecond: 100}, rates["orders"])
	assert.Equal(t, topic.ByteRates{BytesInPerSecond: 1999998}, rates["__consumer_offsets"], "internal topics get collected like any other topic - every other topic metric already does")
}

func TestCountActiveControllers_SkipsFailedConnections(t *testing.T) {
	testutils.SetupTestArgs()

	mockJMXProvider := &mocks.MockJMXProvider{
		Response: &mocks.MockJMXResponse{Err: mocks.ErrQuery},
	}

	brokers := []*connection.Broker{
		{Host: "broker1", JMXPort: 9999},
	}

	count := countActiveControllers(brokers, mockJMXProvider)

	assert.Equal(t, 0, count)
}

func Test_enforceTopicLimit(t *testing.T) {
	overLimit := make([]string, maxTopics+1)

	for i := 0; i < maxTopics+1; i++ {
		overLimit[i] = "topic"
	}

	testCases := []struct {
		name   string
		topics []string
		want   []string
	}{
		{
			"Over Limit",
			overLimit,
			[]string{},
		},
		{
			"Under Limit",
			[]string{"topic"},
			[]string{"topic"},
		},
	}

	for _, tc := range testCases {
		out := enforceTopicLimit(tc.topics)
		if !reflect.DeepEqual(out, tc.want) {
			t.Errorf("Test Case %s Failed: Expected '%+v' got '%+v'", tc.name, tc.want, out)
		}
	}
}

func Test_filterTopicsByBucket(t *testing.T) {
	initialTopics := []string{"a", "b", "c", "d", "e", "f", "g", "h"}

	expectedTopics1 := []string{"c", "f"}
	filteredTopics1 := filterTopicsByBucket(initialTopics, args.TopicBucket{BucketNumber: 1, NumBuckets: 3})
	assert.Equal(t, expectedTopics1, filteredTopics1)

	expectedTopics2 := []string{"b", "e", "h"}
	filteredTopics2 := filterTopicsByBucket(initialTopics, args.TopicBucket{BucketNumber: 2, NumBuckets: 3})
	assert.Equal(t, expectedTopics2, filteredTopics2)

	expectedTopics3 := []string{"a", "d", "g"}
	filteredTopics3 := filterTopicsByBucket(initialTopics, args.TopicBucket{BucketNumber: 3, NumBuckets: 3})
	assert.Equal(t, expectedTopics3, filteredTopics3)
}
