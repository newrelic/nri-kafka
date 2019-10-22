package connection

import (
	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/mock"
)

type MockClient struct {
  mock.Mock
  sarama.Client
}

func (m MockClient) Brokers() []Broker {
  args := m.Called()
  return args.Get(0).([]Broker)
}

func (m MockClient) Coordinator(topic string) (Broker, error) {
  args := m.Called(topic)
  return args.Get(0).(Broker), args.Error(1)
}

func (m MockClient) RefreshCoordinator(topic string) (error) {
  args := m.Called(topic)
  return args.Error(0)
}

func (m MockClient) Leader(topic string, partition int32) (Broker, error) {
	args := m.Called(topic, partition)
	return args.Get(0).(Broker), args.Error(1)
}

func (m MockClient) Close() error {
	args := m.Called()
  return args.Error(0)
}

func (m MockClient) GetOffset(topic string, partitionID int32, time int64) (int64, error) {
	args := m.Called(topic, partitionID, time)
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

type MockClusterAdmin struct {
  mock.Mock
}

func (c MockClusterAdmin) CreateTopic(topic string, detail *sarama.TopicDetail, validateOnly bool) error {
  args := c.Called(topic, detail, validateOnly)
  return args.Error(0)
}

func (c MockClusterAdmin) ListTopics() (map[string]sarama.TopicDetail, error) {
  args := c.Called()
  return args.Get(0).(map[string]sarama.TopicDetail), args.Error(1)
}

func (c MockClusterAdmin) DescribeTopics(topics []string) (metadata []*sarama.TopicMetadata, err error) {
  args := c.Called(topics)
  return args.Get(0).([]*sarama.TopicMetadata), args.Error(1)
}

func (c MockClusterAdmin) DeleteTopic(topic string) error {
  args := c.Called(topic)
  return args.Error(0)
}

func (c MockClusterAdmin) CreatePartitions(topic string, count int32, assignment [][]int32, validateOnly bool) error {
  args := c.Called(topic, count, assignment, validateOnly)
  return args.Error(0)
}

func (c MockClusterAdmin) DeleteRecords(topic string, partitionOffsets map[int32]int64) error {
  args := c.Called(topic, partitionOffsets)
  return args.Error(0)
}

func (c MockClusterAdmin) DescribeConfig(resource sarama.ConfigResource) ([]sarama.ConfigEntry, error) {
  args := c.Called(resource)
  return args.Get(0).([]sarama.ConfigEntry), args.Error(1)
}

func (c MockClusterAdmin) AlterConfig(resourceType sarama.ConfigResourceType, name string, entries map[string]*string, validateOnly bool) error {
  args := c.Called(resourceType, name, entries, validateOnly)
  return args.Error(0)
}

func (c MockClusterAdmin) CreateACL(resource sarama.Resource, acl sarama.Acl) error {
  args := c.Called(resource, acl)
  return args.Error(0)
}

func (c MockClusterAdmin) ListAcls(filter sarama.AclFilter) ([]sarama.ResourceAcls, error) {
  args := c.Called(filter)
  return args.Get(0).([]sarama.ResourceAcls), args.Error(1)
}

func (c MockClusterAdmin) DeleteACL(filter sarama.AclFilter, validateOnly bool) ([]sarama.MatchingAcl, error) {
  args := c.Called(filter, validateOnly)
  return args.Get(0).([]sarama.MatchingAcl), args.Error(1)
}

func (c MockClusterAdmin) ListConsumerGroups() (map[string]string, error) {
  args := c.Called()
  return args.Get(0).(map[string]string), args.Error(1)
}

func (c MockClusterAdmin) DescribeConsumerGroups(groups []string) ([]*sarama.GroupDescription, error) {
  args := c.Called(groups)
  return args.Get(0).([]*sarama.GroupDescription), args.Error(1)
}

func (c MockClusterAdmin) ListConsumerGroupOffsets(group string, topicPartitions map[string][]int32) (*sarama.OffsetFetchResponse, error) {
  args := c.Called(group, topicPartitions)
  return args.Get(0).(*sarama.OffsetFetchResponse), args.Error(1)
}

func (c MockClusterAdmin) DeleteConsumerGroup(group string) error {
  args := c.Called(group)
  return args.Error(0)
}

func (c MockClusterAdmin) DescribeCluster() (brokers []*sarama.Broker, controllerID int32, err error) {
  args := c.Called()
  return args.Get(0).([]*sarama.Broker), args.Get(1).(int32), args.Error(2)
}

func (c MockClusterAdmin) Close() error {
  args := c.Called()
  return args.Error(0)
}
