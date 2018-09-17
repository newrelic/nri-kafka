package connection

import (
	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/mock"
)

// Client is an interface for mocking
type Client interface {
	Brokers() []Broker
	Topics() ([]string, error)
	Partitions(string) ([]int32, error)
	RefreshCoordinator(string) error
	Coordinator(string) (Broker, error)
	Leader(string, int32) (Broker, error)
	Close() error
	GetOffset(string, int32, int64) (int64, error)
}

// SaramaClient is a wrapper struct for sarama.Client
type SaramaClient struct {
	Inner sarama.Client
}

// Brokers wraps the sarama Brokers function
func (c SaramaClient) Brokers() []Broker {
	saramaBrokers := c.Inner.Brokers()
	brokers := make([]Broker, len(saramaBrokers))
	for i, saramaBroker := range saramaBrokers {
		brokers[i] = saramaBroker
	}

	return brokers
}

// Topics wraps the sarama.Client.Topics() function
func (c SaramaClient) Topics() ([]string, error) {
	return c.Inner.Topics()
}

// Partitions wraps the sarama.Client.Partitions() function
func (c SaramaClient) Partitions(topic string) ([]int32, error) {
	return c.Inner.Partitions(topic)
}

// RefreshCoordinator wraps the sarama.Client.RefreshCoordinator() function
func (c SaramaClient) RefreshCoordinator(groupID string) error {
	return c.Inner.RefreshCoordinator(groupID)
}

// Coordinator wraps the sarama.Client.Coordinator() function
func (c SaramaClient) Coordinator(groupID string) (Broker, error) {
	return c.Inner.Coordinator(groupID)
}

// Leader wraps the sarama.Client.Leader() function
func (c SaramaClient) Leader(topic string, partition int32) (Broker, error) {
	return c.Inner.Leader(topic, partition)
}

// Close wraps the sarama.Client.Close() function
func (c SaramaClient) Close() error {
	return c.Inner.Close()
}

// GetOffset wraps the sarama.Client.GetOffset() function
func (c SaramaClient) GetOffset(topic string, partition int32, offset int64) (int64, error) {
	return c.Inner.GetOffset(topic, partition, offset)
}

// Broker is an interface for mocking
type Broker interface {
	Connected() (bool, error)
	FetchOffset(*sarama.OffsetFetchRequest) (*sarama.OffsetFetchResponse, error)
	Fetch(*sarama.FetchRequest) (*sarama.FetchResponse, error)
	Open(*sarama.Config) error
	DescribeGroups(*sarama.DescribeGroupsRequest) (*sarama.DescribeGroupsResponse, error)
	ListGroups(*sarama.ListGroupsRequest) (*sarama.ListGroupsResponse, error)
	Close() error
}

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
