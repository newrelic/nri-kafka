package main

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/kr/pretty"
	"github.com/newrelic/infra-integrations-sdk/data/inventory"
	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/nri-kafka/args"
	"github.com/newrelic/nri-kafka/zookeeper"
)

func TestGetTopics(t *testing.T) {
	testCases := []struct {
		topicMode     string
		topicNames    []string
		expectedNames []string
		expectedErr   bool
	}{
		{"None", []string{}, []string{}, false},
		{"All", []string{}, []string{"test1", "test2", "test3"}, false},
		{"Specific", []string{"test1", "test2"}, []string{"test1", "test2"}, false},
		{"FakeMode", []string{"test1", "test2"}, nil, true},
	}
	zkConn := &zookeeper.MockConnection{}

	for _, tc := range testCases {
		kafkaArgs = &args.KafkaArguments{
			TopicMode: tc.topicMode,
			TopicList: tc.topicNames,
		}

		topicNames, err := getTopics(zkConn)
		if (err != nil) != tc.expectedErr {
			t.Error("Incorrect error state returned.")
		}
		if !reflect.DeepEqual(topicNames, tc.expectedNames) && err != nil {
			t.Errorf("For topicMode %s, expected topic names %s, got %s", tc.topicMode, tc.expectedNames, topicNames)
		}
	}
}
func TestGetTopics_zkErr(t *testing.T) {
	testCases := []struct {
		topicMode     string
		topicNames    []string
		expectedNames []string
		expectedErr   bool
	}{
		{"None", []string{}, []string{}, false},
		{"All", []string{}, []string{"test1", "test2", "test3"}, true},
		{"Specific", []string{"test1", "test2"}, []string{"test1", "test2"}, false},
		{"FakeMode", []string{"test1", "test2"}, nil, true},
	}
	zkConn := &zookeeper.MockConnection{ReturnChildrenError: true}
	i, _ := integration.New("kafka", "1.0.0")
	logger = i.Logger()

	for _, tc := range testCases {
		kafkaArgs = &args.KafkaArguments{
			TopicMode: tc.topicMode,
			TopicList: tc.topicNames,
		}

		topicNames, err := getTopics(zkConn)
		if (err != nil) != tc.expectedErr {
			t.Error("Incorrect error state returned.")
		}
		if !reflect.DeepEqual(topicNames, tc.expectedNames) && (err != nil) != tc.expectedErr {
			t.Errorf("For topicMode %s, expected topic names %s, got %s", tc.topicMode, tc.expectedNames, topicNames)
		}
	}
}

func TestStartTopicPool(t *testing.T) {
	setupTestArgs()
	var wg sync.WaitGroup
	zkConn := zookeeper.MockConnection{}

	topicChan := startTopicPool(3, &wg, &zkConn)
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
	setupTestArgs()
	kafkaArgs.TopicMode = "All"

	i, err := integration.New("kafka", "1.0.0")
	if err != nil {
		t.Error("Failed to create integration")
	}
	zkConn := zookeeper.MockConnection{}
	topicChan := make(chan *topic, 10)

	collectedTopics, err := getTopics(zkConn)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		t.FailNow()
	}

	feedTopicPool(topicChan, i, collectedTopics)

	var topics []*topic
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

}

func TestTopicWorker(t *testing.T) {
	topicChan := make(chan *topic)
	var wg sync.WaitGroup
	zkConn := zookeeper.MockConnection{}

	setupTestArgs()
	kafkaArgs.Metrics = false

	i, err := integration.New("kafka", "1.0.0")
	if err != nil {
		t.Error(err)
	}
	e, err := i.Entity("testtopic", "topic")
	if err != nil {
		t.Error(err)
	}

	go topicWorker(topicChan, &wg, &zkConn)

	myTopic := &topic{
		Name:   "test",
		Entity: e,
	}

	topicChan <- myTopic
	close(topicChan)

	wg.Wait()

	expectedTopic := &topic{
		Name:              "test",
		Partitions:        nil,
		PartitionCount:    3,
		ReplicationFactor: 3,
		Configs:           map[string]string{"flush.messages": "12345"},
	}

	myTopic.Partitions = nil
	myTopic.Entity = nil

	if !reflect.DeepEqual(expectedTopic, myTopic) {
		t.Error("Created topic doesn't match expected")
	}

}

func TestPopulateTopicInventory(t *testing.T) {
	setupTestArgs()

	i, err := integration.New("kafka", "1.0.0")
	if err != nil {
		t.Error(err)
	}
	e, err := i.Entity("testtopic", "topic")
	if err != nil {
		t.Error(err)
	}

	myTopic := &topic{
		Entity:            e,
		Name:              "test",
		PartitionCount:    1,
		ReplicationFactor: 1,
		Configs:           map[string]string{"flush.messages": "12345"},
		Partitions: []*partition{
			{
				ID:             0,
				Leader:         0,
				Replicas:       []int{0},
				InSyncReplicas: []int{0},
			},
		},
	}

	expectedInventoryItems := inventory.Items{
		"Partition Scheme": {
			"Number of Partitions": 1,
			"Replication Factor":   1,
		},
		"Config": {
			"flush.messages": "12345",
		},
	}

	populateTopicInventory(myTopic)

	if !reflect.DeepEqual(myTopic.Entity.Inventory.Items(), expectedInventoryItems) {
		t.Error("Inventory not created correctly")
	}

	return

}

func TestPopulateTopicMetrics(t *testing.T) {
	zkConn := &zookeeper.MockConnection{}

	testTopic := &topic{
		Name: "test",
	}

	i, err := integration.New("kafka", "1.0.0")
	if err != nil {
		t.Error(err)
	}
	testTopic.Entity, err = i.Entity(testTopic.Name, "topics")
	if err != nil {
		t.Error(err)
	}

	sample := testTopic.Entity.NewMetricSet("KafkaTopicSample",
		metric.Attribute{Key: "type", Value: "topic"},
		metric.Attribute{Key: "name", Value: testTopic.Name},
	)

	populateTopicMetrics(testTopic, sample, zkConn)

	expectedMetrics := map[string]interface{}{
		"event_type": "KafkaTopicSample",
		"type":       "topic",
		"name":       "test",
		"topic.retentionBytesOrTime":             0.0,
		"topic.partitionsWithNonPreferredLeader": 0.0,
		"topic.underReplicatedPartitions":        0.0,
		"topic.respondsToMetadataRequests":       0.0,
	}

	if !reflect.DeepEqual(expectedMetrics, sample.Metrics) {
		fmt.Println(pretty.Diff(expectedMetrics, sample.Metrics))
		t.Error("Got unexpected metrics")
	}

	return
}
