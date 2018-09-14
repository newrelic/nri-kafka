package topiccollect

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/newrelic/nri-kafka/src/testutils"
	"github.com/newrelic/nri-kafka/src/zookeeper"
	"github.com/samuel/go-zookeeper/zk"
)

var (
	partitionState = []byte(`{"controller_epoch":9,"leader":2,"version":1,"leader_epoch":0,"isr":[2,0,1]}`)
	topicState     = []byte(`{"version":1,"partitions":{"0":[1,2,0]}}`)
)

func TestCollectPartitions(t *testing.T) {
	var partitionOutChans []chan interface{}
	partitionOutChans = append(partitionOutChans, make(chan interface{}))
	partitionOutChans = append(partitionOutChans, make(chan interface{}))

	for _, ch := range partitionOutChans {
		go func(ch chan interface{}) {
			ch <- &partition{
				ID:             0,
				Leader:         0,
				Replicas:       []int{0, 1, 2},
				InSyncReplicas: []int{0, 1, 2},
			}
			close(ch)
		}(ch)
	}

	expectedPartitions := []*partition{
		{
			ID:             0,
			Leader:         0,
			Replicas:       []int{0, 1, 2},
			InSyncReplicas: []int{0, 1, 2},
		},
		{
			ID:             0,
			Leader:         0,
			Replicas:       []int{0, 1, 2},
			InSyncReplicas: []int{0, 1, 2},
		},
	}

	partitions := collectPartitions(partitionOutChans)
	if !reflect.DeepEqual(partitions, expectedPartitions) {
		t.Error("Failed to produce partitions")
	}

}

func TestPartitionWorker(t *testing.T) {
	testutils.SetupTestArgs()
	partitionInChan := make(chan *partitionSender, 10)
	partitionOutChan := make(chan interface{}, 10)
	var wg sync.WaitGroup
	zkConn := zookeeper.MockConnection{}
	zkConn.On("Get", "/brokers/topics/test/partitions/0/state").Return(partitionState, new(zk.Stat), nil)

	expectedPartition := &partition{
		ID:             0,
		Leader:         2,
		Replicas:       []int{0, 1, 2},
		InSyncReplicas: []int{2, 0, 1},
	}

	topicReplication := map[string][]int{"0": {0, 1, 2}}

	wg.Add(1)
	go partitionWorker(partitionInChan, partitionOutChan, &wg, &zkConn)

	partitionInChan <- &partitionSender{
		ID:               0,
		TopicName:        "test",
		TopicReplication: topicReplication,
	}
	close(partitionInChan)

	wg.Wait()

	partition := <-partitionOutChan

	if !reflect.DeepEqual(partition, expectedPartition) {
		t.Error("Incorrect output partition")
	}

}

func TestFeedPartitionPool(t *testing.T) {

	topicReplication := map[string][]int{"0": {1, 2, 0}}

	expectedPartitionSenders := []*partitionSender{
		{
			ID:               0,
			TopicName:        "test1",
			TopicReplication: topicReplication,
		},
	}

	zkConn := zookeeper.MockConnection{}
	zkConn.On("Get", "/brokers/topics/test1").Return(topicState, new(zk.Stat), nil)
	zkConn.On("Get", "/brokers/topics/test/partitions/0/state").Return(partitionState, new(zk.Stat), nil)

	partitionInChan := make(chan *partitionSender)
	go feedPartitionPool(partitionInChan, "test1", &zkConn)

	var partitionSenders []*partitionSender
	for {
		in, ok := <-partitionInChan
		if !ok {
			break
		}

		partitionSenders = append(partitionSenders, in)
	}

	if !reflect.DeepEqual(expectedPartitionSenders, partitionSenders) {
		t.Error("Expected senders don't match actual")
	}

}

func TestStartPartitionPool(t *testing.T) {
	var wg sync.WaitGroup
	zkConn := zookeeper.MockConnection{}

	partitionInChan, partitionOutChan := startPartitionPool(3, &wg, &zkConn)
	if len(partitionOutChan) != 3 {
		t.Errorf("Expected 3 channels, got %d", len(partitionOutChan))
	}
	close(partitionInChan)

	c := make(chan int, 1)
	go func() {
		wg.Wait()
		c <- 1
	}()

	select {
	case <-c:
	case <-time.After(10 * time.Millisecond):
		t.Error("Wait did not close in a reasonable amount of time.")
	}

}
