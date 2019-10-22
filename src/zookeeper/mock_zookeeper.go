package zookeeper

import (
	"github.com/samuel/go-zookeeper/zk"
	"github.com/stretchr/testify/mock"
	"github.com/newrelic/nri-kafka/src/connection"
	"github.com/Shopify/sarama"
)

// MockConnection implements Connection to facilitate testing.
type MockConnection struct {
	mock.Mock
}

// Get mocks the Get method
func (m MockConnection) Get(s string) ([]byte, *zk.Stat, error) {
	args := m.Called(s)
	return args.Get(0).([]byte), args.Get(1).(*zk.Stat), args.Error(2)
}

// Children mocks the Children method
func (m MockConnection) Children(s string) ([]string, *zk.Stat, error) {
	args := m.Called(s)
	return args.Get(0).([]string), args.Get(1).(*zk.Stat), args.Error(2)
}

// CreateClient mocks the CreateClient method
func (m MockConnection) CreateClient() (connection.Client, error) {
	args := m.Called()
	return args.Get(0).(connection.Client), args.Error(1)
}

// CreateClient mocks the CreateClient method
func (m MockConnection) CreateClusterAdmin() (sarama.ClusterAdmin, error) {
	args := m.Called()
	return args.Get(0).(sarama.ClusterAdmin), args.Error(1)
}
