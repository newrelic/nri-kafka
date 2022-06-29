package consumeroffset

import (
	"github.com/Shopify/sarama"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	consumerGroupOne = "consumer-one"
	topicOne         = "one"
)

type ConsumerGroupTopicListerMock struct{}

func (cm *ConsumerGroupTopicListerMock) ListTopics() (map[string]sarama.TopicDetail, error) {
	return nil, nil
}

func (cm *ConsumerGroupTopicListerMock) ListConsumerGroupOffsets(group string, _ map[string][]int32) (*sarama.OffsetFetchResponse, error) {
	if group == consumerGroupOne {
		return &sarama.OffsetFetchResponse{
			Blocks: map[string]map[int32]*sarama.OffsetFetchResponseBlock{
				"one": {
					0: {Offset: 10},
					1: {Offset: 10},
				},
			},
		}, nil
	}

	return nil, nil
}

type TopicOffsetGetterMock struct{}

func (tm *TopicOffsetGetterMock) GetFromTopicPartition(topicName string, partition int32) (int64, error) {
	// data for "consumer-1"
	if topicName == topicOne {
		switch partition {
		case 0:
			return 25, nil
		case 1:
			return 30, nil
		}
	}
	return 0, nil
}

func TestCollectOffsetsForConsumerGroup(t *testing.T) {
	// MemberAssignment mock created as in sarama's consumer_group_member_test.go
	members := map[string]*sarama.GroupMemberDescription{
		"consumer-1": {
			ClientId:       "consumer-1",
			ClientHost:     "a-host",
			MemberMetadata: nil,
			MemberAssignment: []byte{
				0, 0, // Version
				0, 0, 0, 1, // Topic array length
				0, 3, 'o', 'n', 'e', // Topic one
				0, 0, 0, 3, // Topic one, partition array length
				0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 4, // 0, 2, 4
				0, 0, 0, 3, 0x01, 0x02, 0x03, // Userdata
			},
		},
	}

	args.GlobalArgs = &args.ParsedArguments{}
	args.GlobalArgs.InactiveConsumerGroupOffset = false

	kafkaIntegration, _ := integration.New("test", "test")

	collectOffsetsForConsumerGroup(
		&ConsumerGroupTopicListerMock{},
		consumerGroupOne,
		members,
		kafkaIntegration,
		&TopicOffsetGetterMock{},
	)

	assert.Equal(t, 4, len(kafkaIntegration.Entities))
	for _, entity := range kafkaIntegration.Entities {
		switch entity.Metadata.Namespace {
		case nrConsumerGroupEntity:
			// comes from: 25 - 10 + 30 - 10
			assert.Equal(t, float64(35), entity.Metrics[0].Metrics["consumerGroup.totalLag"])
			// comes from max: 30 - 10
			assert.Equal(t, float64(20), entity.Metrics[0].Metrics["consumerGroup.maxLag"])
			assert.Equal(t, float64(1), entity.Metrics[0].Metrics["consumerGroup.activeConsumers"])
		case nrConsumerGroupTopicEntity:
		case nrConsumerEntity:
		case nrPartitionConsumerEntity:
		}
		assert.NotEmpty(t, entity)
	}
}
