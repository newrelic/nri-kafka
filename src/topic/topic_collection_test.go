package topic

import (
	"sync"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/newrelic/infra-integrations-sdk/v3/data/inventory"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/connection/mocks"
	"github.com/newrelic/nri-kafka/src/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetTopics(t *testing.T) {
	testutils.SetupTestArgs()

	testCases := []struct {
		topicMode     string
		topicNames    []string
		topicRegex    string
		expectedNames []string
		expectedErr   bool
	}{
		{"none", []string{}, "", []string{}, false},
		{"all", []string{"test1", "test2", "test3"}, "", []string{"test1", "test2", "test3"}, false},
		{"regex", []string{"test1a", "test2a", "test1b"}, `test.a`, []string{"test1a", "test2a"}, false},
		{"list", []string{"test1", "test2"}, "", []string{"test1", "test2"}, false},
		{"fakemode", []string{"test1", "test2"}, "", nil, true},
	}

	for _, tc := range testCases {
		args.GlobalArgs.TopicMode = tc.topicMode
		args.GlobalArgs.TopicList = tc.topicNames
		args.GlobalArgs.TopicRegex = tc.topicRegex

		mockClient := &MockGetter{}
		mockClient.On("Topics", mock.Anything).Return(tc.topicNames, nil)

		topicNames, err := GetTopics(mockClient)
		if tc.expectedErr {
			assert.Error(t, err, "Expected error for topicMode %s", tc.topicMode)
		} else {
			assert.NoError(t, err, "Unexpected error for topicMode %s", tc.topicMode)
		}

		assert.Equal(t, tc.expectedNames, topicNames, "Unexpected result for topicMode %s", tc.topicMode)
	}
}

func TestStartTopicPool(t *testing.T) {
	testutils.SetupTestArgs()
	var wg sync.WaitGroup
	mockClient := &mocks.Client{}

	topicChan := StartTopicPool(3, &wg, mockClient)
	close(topicChan)

	c := make(chan int)
	go func() {
		wg.Wait()
		c <- 1
	}()

	select {
	case <-c:
	case <-time.After(10 * time.Millisecond):
		t.Error("Failed to close waitgroup in reasonable amount of time")
	}
}

func TestFeedTopicPool(t *testing.T) {
	testutils.SetupTestArgs()
	args.GlobalArgs.TopicMode = "All"
	args.GlobalArgs.ClusterID = "lkc-abc123"

	i, err := integration.New("kafka", "1.0.0")
	if err != nil {
		t.Error("Failed to create integration")
	}
	mockClient := &MockGetter{}
	mockClient.On("Topics").Return([]string{"test1", "test2", "test3"}, nil)

	topicChan := make(chan *Topic, 10)

	collectedTopics, err := GetTopics(mockClient)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		t.FailNow()
	}

	FeedTopicPool(topicChan, i, collectedTopics)

	var topics []*Topic
	for {
		topic, ok := <-topicChan
		if !ok {
			break
		}

		topics = append(topics, topic)
	}

	for index, name := range []string{"test1", "test2", "test3"} {
		if topics[index].Name != name {
			t.Errorf("Expected topic name %s, got %s", name, topics[index].Name)
		}
	}

	assert.Contains(t, topics[0].Entity.Metadata.IDAttrs, integration.NewIDAttribute("clusterId", "lkc-abc123"))
}

func TestPopulateTopicInventory(t *testing.T) {
	testutils.SetupTestArgs()

	i, _ := integration.New("kafka", "1.0.0")
	e, _ := i.Entity("testtopic", "topic")
	var key, _ = e.Metadata.Key()

	myTopic := &Topic{
		Entity:            e,
		Name:              "test",
		PartitionCount:    1,
		ReplicationFactor: 1,
		Configs: []*sarama.ConfigEntry{
			{
				Name:  "flush.messages",
				Value: "12345",
			},
		},
	}

	expectedInventoryItems := inventory.Items{
		"topic.partitionScheme": {
			"Number of Partitions": 1,
			"Replication Factor":   1,
		},
		"topic.flush.messages": {
			"value": "12345",
		},
		"topic.name": {
			"value": key.String(),
		},
	}

	errors := populateTopicInventory(myTopic)
	assert.Empty(t, errors)

	assert.Equal(t, expectedInventoryItems, myTopic.Entity.Inventory.Items())

}

func TestMinInSyncReplicas_Present(t *testing.T) {
	configs := []*sarama.ConfigEntry{
		{Name: "flush.messages", Value: "12345"},
		{Name: "min.insync.replicas", Value: "2"},
	}

	value, ok := minInSyncReplicas(configs)
	assert.True(t, ok)
	assert.Equal(t, 2, value)
}

func TestMinInSyncReplicas_Absent(t *testing.T) {
	configs := []*sarama.ConfigEntry{
		{Name: "flush.messages", Value: "12345"},
	}

	_, ok := minInSyncReplicas(configs)
	assert.False(t, ok)
}

func TestMinInSyncReplicas_Unparseable(t *testing.T) {
	configs := []*sarama.ConfigEntry{
		{Name: "min.insync.replicas", Value: "not-a-number"},
	}

	_, ok := minInSyncReplicas(configs)
	assert.False(t, ok)
}

func TestRetentionMs_Present(t *testing.T) {
	configs := []*sarama.ConfigEntry{
		{Name: "flush.messages", Value: "12345"},
		{Name: "retention.ms", Value: "604800000"},
	}

	value, ok := retentionMs(configs)
	assert.True(t, ok)
	assert.Equal(t, 604800000, value)
}

func TestRetentionMs_Infinite(t *testing.T) {
	configs := []*sarama.ConfigEntry{
		{Name: "retention.ms", Value: "-1"},
	}

	value, ok := retentionMs(configs)
	assert.True(t, ok)
	assert.Equal(t, -1, value)
}

func TestRetentionMs_Absent(t *testing.T) {
	configs := []*sarama.ConfigEntry{
		{Name: "flush.messages", Value: "12345"},
	}

	_, ok := retentionMs(configs)
	assert.False(t, ok)
}

func TestPopulateTopicConfigMetrics(t *testing.T) {
	testutils.SetupTestArgs()

	i, _ := integration.New("kafka", "1.0.0")
	e, _ := i.Entity("testtopic", "topic")
	sample := e.NewMetricSet("KafkaTopicSample")

	myTopic := &Topic{
		Name:              "test",
		PartitionCount:    3,
		ReplicationFactor: 2,
		Configs: []*sarama.ConfigEntry{
			{Name: "min.insync.replicas", Value: "2"},
			{Name: "retention.ms", Value: "604800000"},
		},
	}

	err := populateTopicConfigMetrics(myTopic, sample)
	assert.NoError(t, err)

	expected := map[string]interface{}{
		"event_type":              "KafkaTopicSample",
		"topic.partitionCount":    float64(3),
		"topic.replicationFactor": float64(2),
		"topic.minInSyncReplicas": float64(2),
		"topic.retentionMs":       float64(604800000),
	}
	assert.Equal(t, expected, sample.Metrics)
}

func TestPopulateTopicConfigMetrics_NoMinInSyncReplicasConfig(t *testing.T) {
	testutils.SetupTestArgs()

	i, _ := integration.New("kafka", "1.0.0")
	e, _ := i.Entity("testtopic", "topic")
	sample := e.NewMetricSet("KafkaTopicSample")

	myTopic := &Topic{
		Name:              "test",
		PartitionCount:    3,
		ReplicationFactor: 2,
	}

	err := populateTopicConfigMetrics(myTopic, sample)
	assert.NoError(t, err)

	expected := map[string]interface{}{
		"event_type":              "KafkaTopicSample",
		"topic.partitionCount":    float64(3),
		"topic.replicationFactor": float64(2),
	}
	assert.Equal(t, expected, sample.Metrics)
}
