package consumeroffset

import (
	"sync"

	"github.com/IBM/sarama"

	"github.com/newrelic/nri-kafka/src/connection"
)

type TopicOffsetGetter interface {
	GetFromTopicPartition(topicName string, partition int32) (int64, error)
	GetEarliestFromTopicPartition(topicName string, partition int32) (int64, error)
}

type SaramaTopicOffsetGetter struct {
	client                       connection.Client
	topicPartitionOffset         map[string]map[int32]int64
	topicPartitionEarliestOffset map[string]map[int32]int64
	mux                          sync.Mutex
}

func NewSaramaTopicOffsetGetter(client connection.Client) *SaramaTopicOffsetGetter {
	return &SaramaTopicOffsetGetter{
		client:                       client,
		topicPartitionOffset:         map[string]map[int32]int64{},
		topicPartitionEarliestOffset: map[string]map[int32]int64{},
	}
}

func (h *SaramaTopicOffsetGetter) GetFromTopicPartition(topicName string, partition int32) (int64, error) {
	var err error
	h.mux.Lock()
	defer h.mux.Unlock()

	if _, ok := h.topicPartitionOffset[topicName]; !ok {
		h.topicPartitionOffset[topicName] = map[int32]int64{}
	}

	if _, ok := h.topicPartitionOffset[topicName][partition]; !ok {
		h.topicPartitionOffset[topicName][partition], err = h.client.GetOffset(topicName, partition, sarama.OffsetNewest)
	}

	return h.topicPartitionOffset[topicName][partition], err
}

// GetEarliestFromTopicPartition returns the log-start offset (earliest message the broker
// still retains) - the same ListOffsets request as GetFromTopicPartition, just OffsetOldest
// instead of OffsetNewest, so a consumer's committed offset can be checked against it to
// detect retention loss (data deleted before the consumer ever read it).
func (h *SaramaTopicOffsetGetter) GetEarliestFromTopicPartition(topicName string, partition int32) (int64, error) {
	var err error
	h.mux.Lock()
	defer h.mux.Unlock()

	if _, ok := h.topicPartitionEarliestOffset[topicName]; !ok {
		h.topicPartitionEarliestOffset[topicName] = map[int32]int64{}
	}

	if _, ok := h.topicPartitionEarliestOffset[topicName][partition]; !ok {
		h.topicPartitionEarliestOffset[topicName][partition], err = h.client.GetOffset(topicName, partition, sarama.OffsetOldest)
	}

	return h.topicPartitionEarliestOffset[topicName][partition], err
}
