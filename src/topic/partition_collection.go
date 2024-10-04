package topic

import (
	"sync"

	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-kafka/src/connection"
)

// partition is a storage struct for information about partitions
type partition struct {
	ID              int
	Leader          int32
	Replicas        []int32
	InSyncReplicas  []int32
	OfflineReplicas []int32
}

// partitionSender is a storage struct to send through the partition creation channel
type partitionSender struct {
	ID        int
	TopicName string
}

// Starts a pool of partitionWorkers to handle collection partition information
// Returns a channel for inbound partitionSenders and outbound partition/error
func startPartitionPool(poolSize int, wg *sync.WaitGroup, client connection.Client) (chan *partitionSender, []chan *partition) {
	partitionInChan := make(chan *partitionSender, 20)
	var partitionOutChans []chan *partition
	for i := 0; i < poolSize; i++ {
		partitionOutChan := make(chan *partition, 20)
		partitionOutChans = append(partitionOutChans, partitionOutChan)
		wg.Add(1)
		go partitionWorker(partitionInChan, partitionOutChan, wg, client)
	}

	return partitionInChan, partitionOutChans
}

// Collects the list of partition IDs from zookeeper, then feeds the partition
// workers with partitionSender structs which contain all the information needed
// for the partitionWorker to populate a partition struct
func feedPartitionPool(partitionInChan chan<- *partitionSender, topic string, client connection.Client) {
	defer close(partitionInChan)

	partitionIDs, err := client.Partitions(topic)
	if err != nil {
		log.Error("Failed to get partitions for topic %s: %s", topic, err)
	}

	// Feed partition info to the partitionWorkers
	for partitionID := range partitionIDs {
		newSender := &partitionSender{
			ID:        partitionID,
			TopicName: topic,
		}

		partitionInChan <- newSender
	}
}

// Collect created partitions from the list of channels into a single
// array of partitions
func collectPartitions(partitionOutChans []chan *partition) []*partition {

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
		go func(channel chan *partition) {
			defer wg.Done()

			for {
				p, ok := <-channel
				if !ok {
					return // If the channel has been closed, return (and decrement wait group)
				}

				partitionCollectChan <- p
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
func partitionWorker(partitionInChan chan *partitionSender, partitionOutChan chan *partition, wg *sync.WaitGroup, client connection.Client) {
	defer wg.Done()
	defer close(partitionOutChan)

	for {
		p, ok := <-partitionInChan
		if !ok {
			return // If channel is closed, stop worker
		}

		leader, err := client.Leader(p.TopicName, int32(p.ID))
		if err != nil {
			log.Error("Failed to get leader for topic %s, partition %d: %s", p.TopicName, p.ID, err)
			continue
		}

		inSyncReplicas, err := client.InSyncReplicas(p.TopicName, int32(p.ID))
		if err != nil {
			log.Error("Failed to get InSyncReplicas for topic %s, partition %d: %s", p.TopicName, p.ID, err)
			continue
		}

		offlineReplicas, err := client.OfflineReplicas(p.TopicName, int32(p.ID))
		if err != nil {
			log.Error("Failed to get InSyncReplicas for topic %s, partition %d: %s", p.TopicName, p.ID, err)
			continue
		}

		replicas, err := client.Replicas(p.TopicName, int32(p.ID))
		if err != nil {
			log.Error("Failed to get InSyncReplicas for topic %s, partition %d: %s", p.TopicName, p.ID, err)
			continue
		}

		newPartition := &partition{
			ID:              p.ID,
			Leader:          leader.ID(),
			InSyncReplicas:  inSyncReplicas,
			OfflineReplicas: offlineReplicas,
			Replicas:        replicas,
		}

		partitionOutChan <- newPartition
	}
}
