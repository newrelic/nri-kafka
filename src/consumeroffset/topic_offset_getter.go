package consumeroffset

import (
	"github.com/Shopify/sarama"
	"github.com/newrelic/nri-kafka/src/connection"
	"sync"
)

type topicOffsetGetter struct {
	client               connection.Client
	topicPartitionOffset map[string]map[int32]int64
	mux                  sync.Mutex
}

func NewTopicOffsetGetter(client connection.Client) *topicOffsetGetter {
	return &topicOffsetGetter{client: client, topicPartitionOffset: map[string]map[int32]int64{}}
}

func (h *topicOffsetGetter) getFromTopicPartition(topicName string, partition int32) (int64, error) {
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
