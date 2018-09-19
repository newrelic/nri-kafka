package conoffsetcollect

import (
	"testing"

	"github.com/Shopify/sarama"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/connection"
	"github.com/newrelic/nri-kafka/src/testutils"
	"github.com/newrelic/nri-kafka/src/zookeeper"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	brokerConnectionBytes = []byte(`{"listener_security_protocol_map":{"PLAINTEXT":"PLAINTEXT"},"endpoints":["PLAINTEXT://kafkabroker:9092"],"jmx_port":9999,"host":"kafkabroker","timestamp":"1530886155628","port":9092,"version":4}`)
	brokerConfigBytes     = []byte(`{"version":1,"config":{"flush.messages":"12345"}}`)
)

func TestCollect(t *testing.T) {
	mockZk := zookeeper.MockConnection{}
	i, _ := integration.New("test", "test")
	testutils.SetupTestArgs()
	mockClient := connection.MockClient{}
	mockBroker := connection.MockBroker{}

	args.GlobalArgs = &args.KafkaArguments{}
	args.GlobalArgs.ConsumerGroups = map[string]map[string][]int32{
		"testGroup": {
			"testTopic": {
				0,
			},
		},
	}

	mockZk.On("Children", "/brokers/ids").Return([]string{"0"}, new(zk.Stat), nil)
	mockZk.On("Get", "/brokers/ids/0").Return(brokerConnectionBytes, new(zk.Stat), nil)
	mockZk.On("CreateClient").Return(&mockClient, nil)
	mockClient.On("Close").Return(nil)
	mockClient.On("Brokers").Return([]connection.Broker{&mockBroker})
	mockClient.On("Topics").Return([]string{"testtopic"}, nil)
	mockBroker.On("Close").Return(nil)
	mockBroker.On("Open", mock.Anything).Return(nil)
	mockBroker.On("ListGroups", mock.Anything).Return(&sarama.ListGroupsResponse{}, nil)
	mockBroker.On("DescribeGroups", mock.Anything).Return(&sarama.DescribeGroupsResponse{}, nil)
	mockClient.On("RefreshCoordinator", "testGroup").Return(nil)
	mockClient.On("Coordinator", "testGroup").Return(&mockBroker, nil)
	mockBroker.On("Connected").Return(true, nil)
	mockBroker.On("FetchOffset", mock.Anything).Return(&sarama.OffsetFetchResponse{}, nil)
	mockClient.On("Leader", "testTopic", int32(0)).Return(&mockBroker, nil)
	mockClient.On("GetOffset", "testTopic", int32(0), int64(-2)).Return(int64(123), nil)
	mockBroker.On("Fetch", mock.Anything).Return(&sarama.FetchResponse{}, nil)

	err := Collect(mockZk, i)
	assert.Nil(t, err)

}

func Test_setMetrics(t *testing.T) {
	i, _ := integration.New("test", "test")
	offsetData := []*partitionOffsets{
		{
			Topic:          "testTopic",
			Partition:      "0",
			ConsumerOffset: func() *int64 { i := int64(123); return &i }(),
			HighWaterMark:  func() *int64 { i := int64(125); return &i }(),
			ConsumerLag:    func() *int64 { i := int64(2); return &i }(),
		},
	}

	err := setMetrics("testGroup", offsetData, i)

	assert.Nil(t, err)
	resultEntity, err := i.Entity("testGroup", "consumerGroup")
	assert.Nil(t, err)
	assert.Equal(t, 8, len(resultEntity.Metrics[0].Metrics))

}
