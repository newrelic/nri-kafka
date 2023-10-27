package consumeroffset

import (
	"sync"

	"github.com/IBM/sarama"
)

type ConsumerGroupTopicLister interface {
	ListTopics() (map[string]sarama.TopicDetail, error)
	ListConsumerGroupOffsets(group string, topicPartitions map[string][]int32) (*sarama.OffsetFetchResponse, error)
}

type CAdminConsumerGroupTopicLister struct {
	clusterAdmin sarama.ClusterAdmin
	topicCache   map[string]sarama.TopicDetail
	mux          sync.Mutex
}

func NewCAdminConsumerGroupTopicLister(clusterAdmin sarama.ClusterAdmin) *CAdminConsumerGroupTopicLister {
	return &CAdminConsumerGroupTopicLister{clusterAdmin: clusterAdmin}
}

func (c *CAdminConsumerGroupTopicLister) ListTopics() (map[string]sarama.TopicDetail, error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if c.topicCache != nil {
		return c.topicCache, nil
	}

	var err error
	c.topicCache, err = c.clusterAdmin.ListTopics()
	if err != nil {
		return nil, err
	}

	return c.topicCache, nil
}

func (c *CAdminConsumerGroupTopicLister) ListConsumerGroupOffsets(group string, topicPartitions map[string][]int32) (*sarama.OffsetFetchResponse, error) {
	return c.clusterAdmin.ListConsumerGroupOffsets(group, topicPartitions)
}
