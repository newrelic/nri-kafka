package topiccollect

import (
	"encoding/json"
	"strconv"
	"sync"

	"github.com/newrelic/nri-kafka/logger"
	"github.com/newrelic/nri-kafka/zookeeper"
)

// partition is a storage struct for information about partitions
type partition struct {
	ID             int
	Leader         int
	Replicas       []int
	InSyncReplicas []int
}

// partitionSender is a storage struct to send through the partition creation channel
type partitionSender struct {
	ID               int
	TopicName        string
	TopicReplication map[string][]int
}

// Starts a pool of partitionWorkers to handle collection partition information
// Returns a channel for inbound partitionSenders and outbound partition/error
func startPartitionPool(poolSize int, wg *sync.WaitGroup, zkConn zookeeper.Connection) (chan *partitionSender, []chan interface{}) {
	partitionInChan := make(chan *partitionSender, 20)
	var partitionOutChans []chan interface{}
	for i := 0; i < poolSize; i++ {
		partitionOutChan := make(chan interface{}, 20)
		partitionOutChans = append(partitionOutChans, partitionOutChan)
		wg.Add(1)
		go partitionWorker(partitionInChan, partitionOutChan, wg, zkConn)
	}

	return partitionInChan, partitionOutChans
}

// Collects the list of partition IDs from zookeeper, then feeds the partition
// workers with partitionSender structs which contain all the information needed
// for the partitionWorker to populate a partition struct
func feedPartitionPool(partitionInChan chan<- *partitionSender, topicName string, zkConn zookeeper.Connection) {
	defer close(partitionInChan)

	// Collect the partition replication info from the topic configuration in Zookeeper
	partitionInfo, _, err := zkConn.Get("/brokers/topics/" + topicName)
	if err != nil {
		logger.Errorf("Unable to collect partition info: %s", err)
		return
	}

	// Parse the JSON returned by Zookeeper
	type partitionInfoDecoder struct {
		Partitions map[string][]int `json:"partitions"`
	}
	var decodedPartitionInfo partitionInfoDecoder
	if err := json.Unmarshal([]byte(partitionInfo), &decodedPartitionInfo); err != nil {
		logger.Errorf("Unable to collect topic partition configuration: %s", err)
		return
	}

	// Feed partition info to the partitionWorkers
	for partitionID := range decodedPartitionInfo.Partitions {
		intID, err := strconv.Atoi(partitionID)
		if err != nil {
			logger.Errorf("Unable to parse id %s to int", partitionID)
			continue
		}
		newSender := &partitionSender{
			ID:               intID,
			TopicName:        topicName,
			TopicReplication: decodedPartitionInfo.Partitions,
		}

		partitionInChan <- newSender
	}

	return
}

// Collect created partitions from the list of channels into a single
// array of partitions
func collectPartitions(partitionOutChans []chan interface{}) []*partition {

	// Create a channel to combine all the worker channels into
	partitionCollectChan := make(chan *partition, 1)

	// Create a wait group that exits when all partitions in a partitionWorker
	// channel have been pushed onto the partitionCollectChan and the partitionWorker
	// channel has been closed
	var wg sync.WaitGroup

	// For each of the partitionWorker channels, collect all the partitions sent down
	// and push them onto partitionCollectChan
	for _, channel := range partitionOutChans {

		wg.Add(1)
		go func(channel chan interface{}) {
			defer wg.Done()

			for {
				p, ok := <-channel
				if !ok {
					return // If the channel has been closed, return (and decrement wait group)
				}

				// If it's an error log it, if it's a partition, push it onto partitionCollectChan
				switch p.(type) {
				case error:
					logger.Errorf("Unable to create partition: %s", p.(error))
				case *partition:
					partitionCollectChan <- p.(*partition)
				}
			}
		}(channel)
	}

	// Have to close the channel at some point, but it has to be after all the input channels have been
	// collected. So, we wait for all the goroutines with input channels and then close, but do it
	// asynchronously so that partitionCollectChan doesn't fill up
	go func() {
		wg.Wait()
		close(partitionCollectChan)
	}()

	// Collect the partitions pushed into partitionCollectChan into an array
	var outPartitions []*partition
	for {
		p, ok := <-partitionCollectChan
		if !ok {
			break
		}
		outPartitions = append(outPartitions, p)
	}

	return outPartitions
}

// Reads partitionSenders from a channel and pushes a completed partition struct onto its unique partitionOutChan
func partitionWorker(partitionInChan chan *partitionSender, partitionOutChan chan interface{}, wg *sync.WaitGroup, c zookeeper.Connection) {
	defer wg.Done()
	defer close(partitionOutChan)

	for {
		in, ok := <-partitionInChan
		if !ok {
			return // If channel is closed, stop worker
		}

		// Collect partition information from Zookeeper
		partitionJSON, _, err := c.Get("/brokers/topics/" + in.TopicName + "/partitions/" + strconv.Itoa(in.ID) + "/state")
		if err != nil {
			partitionOutChan <- err
		}

		// Parse JSON returned from Zookeeper
		type partitionJSONDecoder struct {
			Leader         int   `json:"leader"`
			InSyncReplicas []int `json:"isr"`
		}
		var decodedPartition partitionJSONDecoder
		if err = json.Unmarshal([]byte(partitionJSON), &decodedPartition); err != nil {
			partitionOutChan <- err // If parsing fails, send an error down the partition channel
		}

		newPartition := &partition{
			ID:             in.ID,
			Leader:         decodedPartition.Leader,
			InSyncReplicas: decodedPartition.InSyncReplicas,
			Replicas:       in.TopicReplication[strconv.Itoa(in.ID)],
		}

		partitionOutChan <- newPartition
	}
}
