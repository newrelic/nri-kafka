package zookeeper

import (
	"errors"
	"fmt"
	"time"

	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/samuel/go-zookeeper/zk"
)

// Connection interface to allow easy mocking of a Zookeeper connection
type Connection interface {
	Get(string) ([]byte, *zk.Stat, error)
	Children(string) ([]string, *zk.Stat, error)
	Close()
	Server() string
}

type zookeeperConnection struct {
	inner *zk.Conn
}

func (z zookeeperConnection) Children(s string) ([]string, *zk.Stat, error) {
	return z.inner.Children(s)
}

func (z zookeeperConnection) Get(s string) ([]byte, *zk.Stat, error) {
	return z.inner.Get(s)
}

func (z zookeeperConnection) Server() string {
	return z.inner.Server()
}

type zookeeperLogger struct{}

func (z zookeeperLogger) Printf(format string, args ...interface{}) {
	log.Debug(format, args...)
}

// Close closes the zookeeper connection. You will need to create a new connection after you close this one.
func (z zookeeperConnection) Close() {
	z.inner.Close()
	z.inner = nil
}

// NewConnection creates a new Connection with the given arguments.
// If not hosts are specified then a nil Connection and error will be returned
//
// Waiting on issue https://github.com/samuel/go-zookeeper/issues/108 so we can change this function
// and allow us to mock out the zk.Connect function
func NewConnection(kafkaArgs *args.ParsedArguments) (Connection, error) {
	// No Zookeeper hosts so can't make a connection
	if len(kafkaArgs.ZookeeperHosts) == 0 {
		return nil, errors.New("no Zookeeper hosts specified")
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
		return nil, err
	}

	if kafkaArgs.ZookeeperAuthScheme != "" {
		if err = zkConn.AddAuth(kafkaArgs.ZookeeperAuthScheme, []byte(kafkaArgs.ZookeeperAuthSecret)); err != nil {
			log.Error("Failed to Authenticate to Zookeeper: %s", err.Error())
			zkConn.Close()
			return nil, err
		}
	}

	return zookeeperConnection{zkConn}, nil
}
