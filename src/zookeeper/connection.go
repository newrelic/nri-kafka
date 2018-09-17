// Package zookeeper has a common interface and mock objects to implement and test Zookeeper connections.
package zookeeper

import (
	"fmt"
	"strconv"
	"time"

	"encoding/json"

	"github.com/Shopify/sarama"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/connection"
	"github.com/samuel/go-zookeeper/zk"
)

// Connection interface to allow easy mocking of a Zookeeper connection
type Connection interface {
	Get(string) ([]byte, *zk.Stat, error)
	Children(string) ([]string, *zk.Stat, error)
	CreateClient() (connection.Client, error)
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

func (z zookeeperConnection) CreateClient() (connection.Client, error) {
	brokerIDs, _, err := z.Children(Path("/brokers/ids"))
	if err != nil {
		return nil, err
	}

	brokers := make([]string, 0, len(brokerIDs))
	for _, brokerID := range brokerIDs {
		// convert to int id
		intID, err := strconv.Atoi(brokerID)
		if err != nil {
			log.Warn("Unable to parse integer broker ID from %s", brokerID)
			continue
		}

		// get broker connection info
		host, _, port, err := GetBrokerConnectionInfo(intID, z)
		if err != nil {
			log.Warn("Unable to get connection information for broker with ID '%d'. Will not collect offset data for consumer groups on this broker.", intID)
			continue
		}

		brokers = append(brokers, fmt.Sprintf("%s:%d", host, port))
	}

	c, err := sarama.NewClient(brokers, sarama.NewConfig())
	if err != nil {
		return nil, err
	}

	newClient := connection.SaramaClient{c}

	return newClient, nil
}

// NewConnection creates a new Connection with the given arguments.
// If not hosts are specified then a nil Connection and error will be returned
//
// Waiting on issue https://github.com/samuel/go-zookeeper/issues/108 so we can change this function
// and allow us to mock out the zk.Connect function
func NewConnection(kafkaArgs *args.KafkaArguments) (Connection, error) {
	// No Zookeeper hosts so can't make a connection
	if len(kafkaArgs.ZookeeperHosts) == 0 {
		return nil, nil
	}

	// Create array of host:port strings for connecting
	zkHosts := make([]string, 0, len(kafkaArgs.ZookeeperHosts))
	for _, zkHost := range kafkaArgs.ZookeeperHosts {
		zkHosts = append(zkHosts, fmt.Sprintf("%s:%d", zkHost.Host, zkHost.Port))
	}

	// Create connection and add authentication if provided
	zkConn, _, err := zk.Connect(zkHosts, time.Second)
	if err != nil {
		log.Error("Failed to connect to Zookeeper: %s", err.Error())
		return nil, err
	}

	if kafkaArgs.ZookeeperAuthScheme != "" {
		if err = zkConn.AddAuth(kafkaArgs.ZookeeperAuthScheme, []byte(kafkaArgs.ZookeeperAuthSecret)); err != nil {
			log.Error("Failed to Authenticate to Zookeeper: %s", err.Error())
			return nil, err
		}
	}

	return zookeeperConnection{zkConn}, nil
}

// GetBrokerIDs retrieves the broker ids from Zookeeper
func GetBrokerIDs(zkConn Connection) ([]string, error) {
	brokerIDs, _, err := zkConn.Children(Path("/brokers/ids"))
	if err != nil {
		log.Info(Path("/brokers/ids"))
		return nil, fmt.Errorf("unable to get broker ID from Zookeeper: %s", err.Error())
	}

	return brokerIDs, nil
}

// GetBrokerConnectionInfo Collects Broker connection info from Zookeeper
func GetBrokerConnectionInfo(brokerID int, zkConn Connection) (brokerHost string, jmxPort int, brokerPort int, err error) {

	// Query Zookeeper for broker information
	path := Path("/brokers/ids/" + strconv.Itoa(brokerID))
	rawBrokerJSON, _, err := zkConn.Get(path)
	if err != nil {
		return "", 0, 0, err
	}

	// Parse the JSON returned by Zookeeper
	type brokerJSONDecoder struct {
		Host    string `json:"host"`
		JmxPort int    `json:"jmx_port"`
		Port    int    `json:"port"`
	}
	var brokerDecoded brokerJSONDecoder
	err = json.Unmarshal(rawBrokerJSON, &brokerDecoded)
	if err != nil {
		return "", 0, 0, err
	}

	return brokerDecoded.Host, brokerDecoded.JmxPort, brokerDecoded.Port, nil
}
