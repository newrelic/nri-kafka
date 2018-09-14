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

type SaramaClient struct {
	Inner sarama.Client
}

func (c SaramaClient) Brokers() []Broker {
	saramaBrokers := c.Inner.Brokers()
	brokers := make([]Broker, len(saramaBrokers))
	for i, saramaBroker := range saramaBrokers {
		brokers[i] = saramaBroker
	}

	return brokers
}

func (c SaramaClient) Topics() ([]string, error) {
	return c.Inner.Topics()
}

func (c SaramaClient) Partitions(topic string) ([]int32, error) {
	return c.Inner.Partitions(topic)
}

func (c SaramaClient) RefreshCoordinator(groupID string) error {
	return c.Inner.RefreshCoordinator(groupID)
}

func (c SaramaClient) Coordinator(groupID string) (Broker, error) {
	return c.Inner.Coordinator(groupID)
}

func (c SaramaClient) Leader(topic string, partition int32) (Broker, error) {
	return c.Inner.Leader(topic, partition)
}

func (c SaramaClient) Close() error {
	return c.Inner.Close()
}

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

type MockClient struct {
	mock.Mock
}

func (m MockClient) Brokers() []Broker {
	args := m.Called()
	return args.Get(0).([]Broker)
}

func (m MockClient) Topics() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

func (m MockClient) Partitions(topic string) ([]int32, error) {
	args := m.Called(topic)
	return args.Get(0).([]int32), args.Error(1)
}

func (m MockClient) RefreshCoordinator(groupID string) error {
	args := m.Called(groupID)
	return args.Error(0)
}

func (m MockClient) Coordinator(groupID string) (Broker, error) {
	args := m.Called(groupID)
	return args.Get(0).(Broker), args.Error(1)
}

func (m MockClient) Leader(topic string, partition int32) (Broker, error) {
	args := m.Called(topic, partition)
	return args.Get(0).(Broker), args.Error(1)
}

func (m MockClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m MockClient) GetOffset(topic string, partition int32, offset int64) (int64, error) {
	args := m.Called(topic, partition, offset)
	return args.Get(0).(int64), args.Error(1)
}

type MockBroker struct {
	mock.Mock
}

func (b MockBroker) Connected() (bool, error) {
	args := b.Called()
	return args.Bool(0), args.Error(1)
}

func (b MockBroker) FetchOffset(request *sarama.OffsetFetchRequest) (*sarama.OffsetFetchResponse, error) {
	args := b.Called(request)
	return args.Get(0).(*sarama.OffsetFetchResponse), args.Error(1)
}

func (b MockBroker) Fetch(request *sarama.FetchRequest) (*sarama.FetchResponse, error) {
	args := b.Called(request)
	return args.Get(0).(*sarama.FetchResponse), args.Error(1)
}

func (b MockBroker) Open(config *sarama.Config) error {
	args := b.Called(config)
	return args.Error(0)
}

func (b MockBroker) DescribeGroups(request *sarama.DescribeGroupsRequest) (*sarama.DescribeGroupsResponse, error) {
	args := b.Called(request)
	return args.Get(0).(*sarama.DescribeGroupsResponse), args.Error(1)
}

func (b MockBroker) ListGroups(request *sarama.ListGroupsRequest) (*sarama.ListGroupsResponse, error) {
	args := b.Called(request)
	return args.Get(0).(*sarama.ListGroupsResponse), args.Error(1)
}

func (b MockBroker) Close() error {
	args := b.Called()
	return args.Error(0)
}
