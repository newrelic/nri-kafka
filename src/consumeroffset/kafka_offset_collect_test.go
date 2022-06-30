package consumeroffset

import (
	"fmt"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"

	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/nri-kafka/src/args"
)

const (
	consumerGroupOne = "consumer-one"
	topicOne         = "one"
	topicTwo         = "two"
	testClientID     = "consumer-1"
)

type ConsumerGroupTopicListerMock struct{}

func (cm *ConsumerGroupTopicListerMock) ListTopics() (map[string]sarama.TopicDetail, error) {
	return map[string]sarama.TopicDetail{
		topicOne: {
			NumPartitions:     2,
			ReplicationFactor: 0,
			ReplicaAssignment: nil,
			ConfigEntries:     nil,
		},
		topicTwo: {
			NumPartitions:     2,
			ReplicationFactor: 0,
			ReplicaAssignment: nil,
			ConfigEntries:     nil,
		},
	}, nil
}

func (cm *ConsumerGroupTopicListerMock) ListConsumerGroupOffsets(group string, topicPartitions map[string][]int32) (*sarama.OffsetFetchResponse, error) {
	offsetFetchResponse := &sarama.OffsetFetchResponse{
		Blocks: map[string]map[int32]*sarama.OffsetFetchResponseBlock{},
	}
	for topic := range topicPartitions {
		offsetFetchResponse.Blocks[topic] = map[int32]*sarama.OffsetFetchResponseBlock{
			0: {Offset: 10},
			1: {Offset: 10},
		}
	}

	return offsetFetchResponse, nil
}

type TopicOffsetGetterMock struct{}

func (tm *TopicOffsetGetterMock) GetFromTopicPartition(topicName string, partition int32) (int64, error) {
	switch partition {
	case 0:
		return 25, nil
	case 1:
		return 30, nil
	}
	return 0, nil
}

func TestCollectOffsetsForConsumerGroup(t *testing.T) { // nolint: funlen
	// MemberAssignment mock created as in sarama's consumer_group_member_test.go
	members := map[string]*sarama.GroupMemberDescription{
		testClientID: {
			ClientId:       testClientID,
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

	testCases := []struct {
		name                        string
		inactiveConsumerGroupOffset bool
		consumerGroupOffsetByTopic  bool
		consumerGroup               string
		cGroupEntities              map[string]map[string]float64
		topicEntities               map[string]map[string]float64
		numEntities                 int
	}{
		{
			name:                        "Only active consumers",
			inactiveConsumerGroupOffset: false,
			consumerGroupOffsetByTopic:  false,
			consumerGroup:               consumerGroupOne,
			cGroupEntities: map[string]map[string]float64{
				consumerGroupOne: {
					// comes from: 25 - 10 + 30 - 10
					"totalLag": 35,
					// comes from max: 30 - 10
					"maxLag":          20,
					"activeConsumers": 1,
				},
			},
			topicEntities: nil,
			numEntities:   4,
		},
		{
			name:                        "Only active consumers with topic Aggregation",
			inactiveConsumerGroupOffset: false,
			consumerGroupOffsetByTopic:  true,
			consumerGroup:               consumerGroupOne,
			cGroupEntities: map[string]map[string]float64{
				consumerGroupOne: {
					// comes from: 25 - 10 + 30 - 10
					"totalLag": 35,
					// comes from max: 30 - 10
					"maxLag":          20,
					"activeConsumers": 1,
				},
			},
			topicEntities: map[string]map[string]float64{
				topicOne: {
					// comes from: 25 - 10 + 30 - 10
					"totalLag": 35,
					// comes from max: 30 - 10
					"maxLag":          20,
					"activeConsumers": 1,
				},
			},
			numEntities: 5,
		},
		{
			name:                        "With inactive consumers",
			inactiveConsumerGroupOffset: true,
			consumerGroupOffsetByTopic:  false,
			consumerGroup:               consumerGroupOne,
			cGroupEntities: map[string]map[string]float64{
				consumerGroupOne: {
					// comes from: (25 - 10 + 30 - 10) + (25 - 10 + 30 - 10)
					"totalLag": 70,
					// comes from max: 30 - 10
					"maxLag":          20,
					"activeConsumers": 1,
				},
			},
			topicEntities: nil,
			numEntities:   4,
		},
		{
			name:                        "With inactive consumers and topic aggregation",
			inactiveConsumerGroupOffset: true,
			consumerGroupOffsetByTopic:  true,
			consumerGroup:               consumerGroupOne,
			cGroupEntities: map[string]map[string]float64{
				consumerGroupOne: {
					// comes from: (25 - 10 + 30 - 10) + (25 - 10 + 30 - 10)
					"totalLag": 70,
					// comes from max: 30 - 10
					"maxLag":          20,
					"activeConsumers": 1,
				},
			},
			topicEntities: map[string]map[string]float64{
				topicOne: {
					// comes from: 25 - 10 + 30 - 10
					"totalLag": 35,
					// comes from max: 30 - 10
					"maxLag":          20,
					"activeConsumers": 1,
				},
				topicTwo: {
					// comes from: 25 - 10 + 30 - 10
					"totalLag": 35,
					// comes from max: 30 - 10
					"maxLag":          20,
					"activeConsumers": 0,
				},
			},
			numEntities: 6,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args.GlobalArgs.InactiveConsumerGroupOffset = tc.inactiveConsumerGroupOffset
			args.GlobalArgs.ConsumerGroupOffsetByTopic = tc.consumerGroupOffsetByTopic

			kafkaIntegration, _ := integration.New("test", "test")

			collectOffsetsForConsumerGroup(
				&ConsumerGroupTopicListerMock{},
				tc.consumerGroup,
				members,
				kafkaIntegration,
				&TopicOffsetGetterMock{},
			)

			assert.Equal(t, tc.numEntities, len(kafkaIntegration.Entities))
			for _, entity := range kafkaIntegration.Entities {
				switch entity.Metadata.Namespace {
				case nrConsumerGroupEntity:
					if entity.Metrics[0].Metrics["consumerGroup"] == tc.consumerGroup {
						assert.Equal(t, tc.cGroupEntities[tc.consumerGroup]["totalLag"], entity.Metrics[0].Metrics["consumerGroup.totalLag"])
						assert.Equal(t, tc.cGroupEntities[tc.consumerGroup]["maxLag"], entity.Metrics[0].Metrics["consumerGroup.maxLag"])
						assert.Equal(t, tc.cGroupEntities[tc.consumerGroup]["activeConsumers"], entity.Metrics[0].Metrics["consumerGroup.activeConsumers"])
					}
				case nrConsumerGroupTopicEntity:
					topicName := fmt.Sprintf("%v", entity.Metrics[0].Metrics["topic"])
					assert.Equal(t, tc.topicEntities[topicName]["totalLag"], entity.Metrics[0].Metrics["consumerGroup.totalLag"])
					assert.Equal(t, tc.topicEntities[topicName]["maxLag"], entity.Metrics[0].Metrics["consumerGroup.maxLag"])
					assert.Equal(t, tc.topicEntities[topicName]["activeConsumers"], entity.Metrics[0].Metrics["consumerGroup.activeConsumers"])
				case nrConsumerEntity:
					assert.Equal(t, testClientID, entity.Metrics[0].Metrics["clientID"])
				case nrPartitionConsumerEntity:
					// this entity only for topicOne that has member clientID
					assert.Equal(t, testClientID, entity.Metrics[0].Metrics["clientID"])
					assert.Equal(t, topicOne, entity.Metrics[0].Metrics["topic"])
				}
				assert.NotEmpty(t, entity)
			}
		})
	}
}
