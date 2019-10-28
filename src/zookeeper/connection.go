// Package zookeeper has a common interface and mock objects to implement and test Zookeeper connections.
package zookeeper

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

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
	CreateClusterAdmin() (sarama.ClusterAdmin, error)
}

// BrokerConnection struct to allow for multiple connection setups.
type BrokerConnection struct {
	Scheme     string
	BrokerHost string
	JmxPort    int
	BrokerPort int
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

	connections := make(map[string][]string, 0)
	for _, brokerID := range brokerIDs {
		// convert to int id
		intID, err := strconv.Atoi(brokerID)
		if err != nil {
			log.Warn("Unable to parse integer broker ID from %s", brokerID)
			continue
		}

		// get broker connection info
		brokerConnections, err := GetBrokerConnectionInfo(intID, z)
		if err != nil {
			log.Warn("Unable to get connection information for broker with ID '%d'. Will not collect offset data for consumer groups on this broker: %s", intID, err)
			continue
		}

		for _, brokerConnection := range brokerConnections {
			connections[brokerConnection.Scheme] = append(connections[brokerConnection.Scheme],
				fmt.Sprintf("%s:%d", brokerConnection.BrokerHost, brokerConnection.BrokerPort))
		}
	}

	var client sarama.Client
	for scheme, connection := range connections {
		client, err = sarama.NewClient(connection, createConfig(scheme == "https"))
		if err != nil {
			continue
		} else { // make sure that we break when we have a working connection.
			break
		}
	}
	if err != nil {
		return nil, err
	}
	return connection.SaramaClient{client}, nil
}

func (z zookeeperConnection) CreateClusterAdmin() (sarama.ClusterAdmin, error) {
	brokerIDs, _, err := z.Children(Path("/brokers/ids"))
	if err != nil {
		return nil, err
	}

	connections := make(map[string][]string, 0)
	for _, brokerID := range brokerIDs {
		// convert to int id
		intID, err := strconv.Atoi(brokerID)
		if err != nil {
			log.Warn("Unable to parse integer broker ID from %s", brokerID)
			continue
		}

		// get broker connection info
		brokerConnections, err := GetBrokerConnectionInfo(intID, z)
		if err != nil {
			log.Warn("Unable to get connection information for broker with ID '%d'. Will not collect offset data for consumer groups on this broker: %s", intID, err)
			continue
		}

		for _, brokerConnection := range brokerConnections {
			connections[brokerConnection.Scheme] = append(connections[brokerConnection.Scheme],
				fmt.Sprintf("%s:%d", brokerConnection.BrokerHost, brokerConnection.BrokerPort))
		}
	}

	var client sarama.ClusterAdmin
	for scheme, connection := range connections {
		client, err = sarama.NewClusterAdmin(connection, createConfig(scheme == "https"))
		if err != nil {
			continue
		} else { // make sure that we break when we have a working connection.
			break
		}
	}
	if err != nil {
		return nil, err
	}

	return client, nil
}

func createConfig(isTLS bool) *sarama.Config {
	config := sarama.NewConfig()
	if isTLS {
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	config.Version = sarama.V2_0_0_0

	return config
}

// NewConnection creates a new Connection with the given arguments.
// If not hosts are specified then a nil Connection and error will be returned
//
// Waiting on issue https://github.com/samuel/go-zookeeper/issues/108 so we can change this function
// and allow us to mock out the zk.Connect function
func NewConnection(kafkaArgs *args.KafkaArguments) (Connection, error) {
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

func getURLStringAndSchemeFromEndpoints(endpoints []string, protocolMap map[string]string) (schemes []string, brokerHosts []*url.URL, err error) {
	schemes = make([]string, 0)
	brokerHosts = make([]*url.URL, 0)
	for _, urlString := range endpoints {
		scheme, brokerHost, err := getURLStringAndSchemeFromEndpoint(urlString, protocolMap)
		if err != nil {
			continue
		}
		schemes = append(schemes, scheme)
		brokerHosts = append(brokerHosts, brokerHost)
	}
	if len(schemes) == 0 && len(brokerHosts) == 0 {
		return nil, nil, errors.New("host could not be found for broker")
	}
	return schemes, brokerHosts, nil
}

func getURLStringAndSchemeFromEndpoint(urlString string, protocolMap map[string]string) (scheme string, brokerHost *url.URL, err error) {
	prefix := strings.Split(urlString, "://")[0]
	brokerHost, err = url.Parse(urlString)
	if err != nil {
		return
	}
	urlStringProtocol, hasProtocol := protocolMap[prefix]
	if strings.HasPrefix(urlString, "SSL") || (hasProtocol && strings.HasPrefix(urlStringProtocol, "SSL")) {
		scheme = "https"
		return
	} else if strings.HasPrefix(urlString, "PLAINTEXT") || (hasProtocol && strings.HasPrefix(urlStringProtocol, "PLAINTEXT")) {
		scheme = "http"
		return
	}
	return "", nil, errors.New("Protocol not found")
}

// GetBrokerConnectionInfo Collects Broker connection info from Zookeeper
func GetBrokerConnectionInfo(brokerID int, zkConn Connection) (brokerConnetions []BrokerConnection, err error) {

	// Query Zookeeper for broker information
	path := Path("/brokers/ids/" + strconv.Itoa(brokerID))
	rawBrokerJSON, _, err := zkConn.Get(path)
	if err != nil {
		return
	}

	// Parse the JSON returned by Zookeeper
	type brokerJSONDecoder struct {
		ProtocolMap map[string]string `json:"listener_security_protocol_map"`
		JmxPort     int               `json:"jmx_port"`
		Endpoints   []string          `json:"endpoints"`
	}
	var brokerDecoded brokerJSONDecoder
	err = json.Unmarshal(rawBrokerJSON, &brokerDecoded)
	if err != nil {
		return
	}

	// We only want the URL if it's SSL or PLAINTEXT
	schemes, brokerURLs, err := getURLStringAndSchemeFromEndpoints(brokerDecoded.Endpoints, brokerDecoded.ProtocolMap)

	if err != nil {
		return
	}

	connections := make([]BrokerConnection, 0)
	for i, scheme := range schemes {
		host, portString := brokerURLs[i].Hostname(), brokerURLs[i].Port()

		port, err := strconv.Atoi(portString)
		if err != nil {
			return nil, nil
		}
		connections = append(connections, BrokerConnection{Scheme: scheme, BrokerHost: host, JmxPort: brokerDecoded.JmxPort, BrokerPort: port})
	}
	return connections, nil
}
