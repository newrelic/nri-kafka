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
	return h.cachedOffset(h.topicPartitionOffset, topicName, partition, sarama.OffsetNewest)
}

// GetEarliestFromTopicPartition returns the log-start offset (earliest message the broker still retains).
func (h *SaramaTopicOffsetGetter) GetEarliestFromTopicPartition(topicName string, partition int32) (int64, error) {
	return h.cachedOffset(h.topicPartitionEarliestOffset, topicName, partition, sarama.OffsetOldest)
}

func (h *SaramaTopicOffsetGetter) cachedOffset(cache map[string]map[int32]int64, topicName string, partition int32, offsetTime int64) (int64, error) {
	var err error
	h.mux.Lock()
	defer h.mux.Unlock()

	if _, ok := cache[topicName]; !ok {
		cache[topicName] = map[int32]int64{}
	}

	if _, ok := cache[topicName][partition]; !ok {
		cache[topicName][partition], err = h.client.GetOffset(topicName, partition, offsetTime)
	}

	return cache[topicName][partition], err
}
