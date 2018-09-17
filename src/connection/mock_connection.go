package connection

import (
	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/mock"
)

// MockClient is a mock implementation of the Client interface
type MockClient struct {
	mock.Mock
}

// Brokers is a mocked implementation of the sarama.Client.Brokers() method
func (m MockClient) Brokers() []Broker {
	args := m.Called()
	return args.Get(0).([]Broker)
}

// Topics is a mocked implementation of the sarama.Client.Topics() method
func (m MockClient) Topics() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

// Partitions is a mocked implementation of the sarama.Client.Partitions() method
func (m MockClient) Partitions(topic string) ([]int32, error) {
	args := m.Called(topic)
	return args.Get(0).([]int32), args.Error(1)
}

// RefreshCoordinator is a mocked implementation of the sarama.Client.RefreshCoordinator() method
func (m MockClient) RefreshCoordinator(groupID string) error {
	args := m.Called(groupID)
	return args.Error(0)
}

// Coordinator is a mocked implementation of the sarama.Client.Coordinator() method
func (m MockClient) Coordinator(groupID string) (Broker, error) {
	args := m.Called(groupID)
	return args.Get(0).(Broker), args.Error(1)
}

// Leader is a mocked implementation of the sarama.Client.Leader() method
func (m MockClient) Leader(topic string, partition int32) (Broker, error) {
	args := m.Called(topic, partition)
	return args.Get(0).(Broker), args.Error(1)
}

// Close is a mocked implementation of the sarama.Client.Close() method
func (m MockClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

// GetOffset is a mocked implementation of the sarama.Client.GetOffset() method
func (m MockClient) GetOffset(topic string, partition int32, offset int64) (int64, error) {
	args := m.Called(topic, partition, offset)
	return args.Get(0).(int64), args.Error(1)
}

// MockBroker is a mock implementation of the Broker interface
type MockBroker struct {
	mock.Mock
}

// Connected is a mocked implementation of the sarama.Broker.Connected() method
func (b MockBroker) Connected() (bool, error) {
	args := b.Called()
	return args.Bool(0), args.Error(1)
}

// FetchOffset is a mocked implementation of the sarama.Broker.FetchOffset() method
func (b MockBroker) FetchOffset(request *sarama.OffsetFetchRequest) (*sarama.OffsetFetchResponse, error) {
	args := b.Called(request)
	return args.Get(0).(*sarama.OffsetFetchResponse), args.Error(1)
}

// Fetch is a mocked implementation of the sarama.Broker.Fetch() method
func (b MockBroker) Fetch(request *sarama.FetchRequest) (*sarama.FetchResponse, error) {
	args := b.Called(request)
	return args.Get(0).(*sarama.FetchResponse), args.Error(1)
}

// Open is a mocked implementation of the sarama.Broker.Open() method
func (b MockBroker) Open(config *sarama.Config) error {
	args := b.Called(config)
	return args.Error(0)
}

// DescribeGroups is a mocked implementation of the sarama.Broker.DescribeGroups() method
func (b MockBroker) DescribeGroups(request *sarama.DescribeGroupsRequest) (*sarama.DescribeGroupsResponse, error) {
	args := b.Called(request)
	return args.Get(0).(*sarama.DescribeGroupsResponse), args.Error(1)
}

// ListGroups is a mocked implementation of the sarama.Broker.ListGroups() method
func (b MockBroker) ListGroups(request *sarama.ListGroupsRequest) (*sarama.ListGroupsResponse, error) {
	args := b.Called(request)
	return args.Get(0).(*sarama.ListGroupsResponse), args.Error(1)
}

// Close is a mocked implementation of the sarama.Broker.Close() method
func (b MockBroker) Close() error {
	args := b.Called()
	return args.Error(0)
}
