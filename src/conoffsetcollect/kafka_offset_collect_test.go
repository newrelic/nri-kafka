package conoffsetcollect

import (
	"errors"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/newrelic/nri-kafka/src/connection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_getConsumerOffsets(t *testing.T) {
	groupName := "testGroup"
	topicPartitions := TopicPartitions{"testTopic": {0}}
	fakeClient := new(connection.MockClient)
	fakeBroker := new(connection.MockBroker)
	fetchOffsetResponse := new(sarama.OffsetFetchResponse)
	offsetResponseBlock := new(sarama.OffsetFetchResponseBlock)
	offsetResponseBlock.Offset = 10
	offsetResponseBlock.Err = sarama.ErrNoError
	fetchOffsetResponse.Blocks = map[string]map[int32]*sarama.OffsetFetchResponseBlock{
		"testTopic": {0: offsetResponseBlock},
	}

	fakeClient.On("RefreshCoordinator", mock.Anything).Return(nil)
	fakeClient.On("Coordinator", groupName).Return(fakeBroker, nil)
	fakeBroker.On("Connected").Return(true, nil)
	fakeBroker.On("FetchOffset", mock.Anything).Return(fetchOffsetResponse, nil)
	fakeBroker.On("Close").Return(nil)
	fakeBroker.On("Open", mock.Anything).Return(nil)

	offsets, err := getConsumerOffsets(groupName, topicPartitions, fakeClient)

	assert.Nil(t, err)

	assert.Equal(t, int64(10), offsets["testTopic"][0])
}

func Test_getHighWaterMarks(t *testing.T) {
	topicPartitions := TopicPartitions{"testTopic": {0}}
	fakeClient := new(connection.MockClient)
	fakeBroker := new(connection.MockBroker)
	fakeFetchResponse := &sarama.FetchResponse{}
	fakeFetchResponse.Blocks = map[string]map[int32]*sarama.FetchResponseBlock{
		"testTopic": {0: {HighWaterMarkOffset: 20}},
	}

	fakeClient.On("Leader", "testTopic", int32(0)).Return(fakeBroker, nil)
	fakeBroker.On("Connected").Return(true, nil)
	fakeClient.On("GetOffset", "testTopic", int32(0), int64(-2)).Return(int64(20), nil)
	fakeBroker.On("Fetch", mock.Anything).Return(fakeFetchResponse, nil)
	fakeBroker.On("Close").Return(nil)
	fakeBroker.On("Open", mock.Anything).Return(nil)

	hwms, err := getHighWaterMarks(topicPartitions, fakeClient)

	assert.Nil(t, err)
	assert.Equal(t, int64(20), hwms["testTopic"][0])
}

func Test_getHighWaterMarks_FetchErr(t *testing.T) {
	topicPartitions := TopicPartitions{"testTopic": {0}}
	fakeClient := new(connection.MockClient)
	fakeBroker := new(connection.MockBroker)
	fakeFetchResponse := &sarama.FetchResponse{}
	fakeFetchResponse.Blocks = map[string]map[int32]*sarama.FetchResponseBlock{
		"testTopic": {0: {HighWaterMarkOffset: 20}},
	}

	fakeClient.On("Leader", "testTopic", int32(0)).Return(fakeBroker, nil)
	fakeBroker.On("Connected").Return(true, nil)
	fakeClient.On("GetOffset", "testTopic", int32(0), int64(-2)).Return(int64(20), nil)
	fakeBroker.On("Fetch", mock.Anything).Return(&sarama.FetchResponse{}, errors.New("this is a test error"))
	fakeBroker.On("Close").Return(nil)
	fakeBroker.On("Open", mock.Anything).Return(nil)

	hwms, err := getHighWaterMarks(topicPartitions, fakeClient)

	assert.Nil(t, err)
	assert.Equal(t, 0, len(hwms))
}

func Test_getHighWaterMarks_ClosedErr(t *testing.T) {
	topicPartitions := TopicPartitions{"testTopic": {0}}
	fakeClient := new(connection.MockClient)
	fakeBroker := new(connection.MockBroker)
	fakeFetchResponse := &sarama.FetchResponse{}
	fakeFetchResponse.Blocks = map[string]map[int32]*sarama.FetchResponseBlock{
		"testTopic": {0: {HighWaterMarkOffset: 20}},
	}

	fakeClient.On("Leader", "testTopic", int32(0)).Return(fakeBroker, nil)
	fakeBroker.On("Connected").Return(false, nil)
	fakeBroker.On("Open", mock.Anything).Return(errors.New("this is a test error"))

	fakeClient.On("GetOffset", "testTopic", int32(0), int64(-2)).Return(int64(20), nil)
	fakeBroker.On("Fetch", mock.Anything).Return(fakeFetchResponse, nil)

	_, err := getHighWaterMarks(topicPartitions, fakeClient)

	assert.Nil(t, err, "Expected an error, but it was nil")
}

func Test_fillTopicPartitions(t *testing.T) {
	groupID := "testGroup"
	topicPartitions := map[string][]int32{}
	fakeClient := new(connection.MockClient)

	newTopicPartitions := fillTopicPartitions(groupID, topicPartitions, fakeClient)

	assert.Equal(t, 0, len(newTopicPartitions["testTopic"]))
}

func Test_populateOffsetStructs(t *testing.T) {
	inputOffsets := groupOffsets{"testTopic": {0: 12}}
	inputHwms := groupOffsets{"testTopic": {0: 13}}

	partitionOffsets := populateOffsetStructs(inputOffsets, inputHwms)
	assert.Equal(t, 1, len(partitionOffsets))
	assert.Equal(t, int64(12), *partitionOffsets[0].ConsumerOffset)
	assert.Equal(t, int64(13), *partitionOffsets[0].HighWaterMark)
	assert.Equal(t, int64(1), *partitionOffsets[0].ConsumerLag)

}
