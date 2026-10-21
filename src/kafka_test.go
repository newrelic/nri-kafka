package main

import (
	"reflect"
	"testing"

	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/connection"
	"github.com/newrelic/nri-kafka/src/connection/mocks"
	"github.com/newrelic/nri-kafka/src/testutils"
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

	count, controller, known := countActiveControllers(brokers, mockJMXProvider)

	assert.Equal(t, 3, count)
	assert.True(t, known)
	assert.Equal(t, brokers[0], controller)
}

func TestCountActiveControllers_SkipsFailedConnections(t *testing.T) {
	testutils.SetupTestArgs()

	mockJMXProvider := &mocks.MockJMXProvider{
		Response: &mocks.MockJMXResponse{Err: mocks.ErrQuery},
	}

	brokers := []*connection.Broker{
		{Host: "broker1", JMXPort: 9999},
	}

	count, controller, known := countActiveControllers(brokers, mockJMXProvider)

	assert.Equal(t, 0, count)
	assert.Nil(t, controller)
	assert.False(t, known, "a count of 0 from unreadable brokers is unknown, not zero")
}

func TestCountActiveControllers_ErrorResponseIsNotZero(t *testing.T) {
	testutils.SetupTestArgs()

	mockJMXProvider := &mocks.MockJMXProvider{
		Response: &mocks.MockJMXResponse{Result: []*gojmx.AttributeResponse{
			{Name: "kafka.controller:type=KafkaController,name=ActiveControllerCount,attr=Value", ResponseType: gojmx.ResponseTypeErr},
		}},
	}

	_, _, known := countActiveControllers([]*connection.Broker{{Host: "broker1", JMXPort: 9999}}, mockJMXProvider)

	assert.False(t, known)
}

func TestCountActiveControllers_ConfirmedZero(t *testing.T) {
	testutils.SetupTestArgs()

	mockJMXProvider := &mocks.MockJMXProvider{
		Response: &mocks.MockJMXResponse{Result: []*gojmx.AttributeResponse{
			{Name: "kafka.controller:type=KafkaController,name=ActiveControllerCount,attr=Value", ResponseType: gojmx.ResponseTypeInt, IntValue: 0},
		}},
	}

	count, controller, known := countActiveControllers([]*connection.Broker{{Host: "broker1", JMXPort: 9999}}, mockJMXProvider)

	assert.Equal(t, 0, count)
	assert.Nil(t, controller)
	assert.True(t, known, "every broker answered 0: no active controller")
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
