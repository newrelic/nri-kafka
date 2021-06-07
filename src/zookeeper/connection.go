package zookeeper

import (
	"errors"
	"fmt"
	"time"

	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/samuel/go-zookeeper/zk"
)

var errNoZookeeperHostSpecified = errors.New("no Zookeeper hosts specified")

// Connection interface to allow easy mocking of a Zookeeper connection
type Connection interface {
	Get(string) ([]byte, *zk.Stat, error)
	Children(string) ([]string, *zk.Stat, error)
	Close()
	Server() string
}

type ZkConnection struct {
	inner *zk.Conn
}

func (z ZkConnection) Children(s string) ([]string, *zk.Stat, error) {
	return z.inner.Children(s)
}

func (z ZkConnection) Get(s string) ([]byte, *zk.Stat, error) {
	return z.inner.Get(s)
}

func (z ZkConnection) Server() string {
	return z.inner.Server()
}

type zookeeperLogger struct{}

func (z zookeeperLogger) Printf(format string, args ...interface{}) {
	log.Debug(format, args...)
}

// Close closes the zookeeper connection. You will need to create a new connection after you close this one.
func (z ZkConnection) Close() {
	z.inner.Close()
	z.inner = nil
}

func (z ZkConnection) Topics() ([]string, error) {
	topics, _, err := z.Children(Path("/brokers/topics"))
	return topics, err
}

// NewConnection creates a new ZookeeperConnection with the given arguments.
// If not hosts are specified then an empty ZookeeperConnection and an error will be returned
//
// Waiting on issue https://github.com/samuel/go-zookeeper/issues/108 so we can change this function
// and allow us to mock out the zk.Connect function
func NewConnection(kafkaArgs *args.ParsedArguments) (ZkConnection, error) {
	// No Zookeeper hosts so can't make a connection
	if len(kafkaArgs.ZookeeperHosts) == 0 {
		return ZkConnection{}, errNoZookeeperHostSpecified
	}

	// Create array of host:port strings for connecting
	zkHosts := make([]string, 0, len(kafkaArgs.ZookeeperHosts))
	for _, zkHost := range kafkaArgs.ZookeeperHosts {
		zkHosts = append(zkHosts, fmt.Sprintf("%s:%d", zkHost.Host, zkHost.Port))
	}

	// Create array of host:port strings for connecting
	// Create connection and add authentication if provided

	zkConn, _, err := zk.Connect(zkHosts, time.Second, zk.WithLogger(zookeeperLogger{}))
	if err != nil {
		log.Error("Failed to connect to Zookeeper: %s", err.Error())
		return ZkConnection{}, err
	}

	if kafkaArgs.ZookeeperAuthScheme != "" {
		if err = zkConn.AddAuth(kafkaArgs.ZookeeperAuthScheme, []byte(kafkaArgs.ZookeeperAuthSecret)); err != nil {
			log.Error("Failed to Authenticate to Zookeeper: %s", err.Error())
			zkConn.Close()
			return ZkConnection{}, err
		}
	}

	return ZkConnection{zkConn}, nil
}
