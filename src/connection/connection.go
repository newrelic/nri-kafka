package connection

import (
	"github.com/Shopify/sarama"
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
	sarama.Client
}

// Brokers wraps the sarama Brokers function
func (c SaramaClient) Brokers() []Broker {
	saramaBrokers := c.Client.Brokers()
	brokers := make([]Broker, len(saramaBrokers))
	for i, broker := range saramaBrokers {
		saramaBrokers[i] = broker
	}

	return brokers
}

// Coordinator wraps the sarama.Client.Coordinator() function
func (c SaramaClient) Coordinator(groupID string) (Broker, error) {
	return c.Client.Coordinator(groupID)
}

// Leader wraps the sarama.Client.Leader() function
func (c SaramaClient) Leader(topic string, partition int32) (Broker, error) {
	return c.Client.Leader(topic, partition)
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
