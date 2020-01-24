// Package zookeeper has a common interface and mock objects to implement and test Zookeeper connections.
package zookeeper

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
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
	CreateClusterAdmin() (sarama.ClusterAdmin, error)
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

func (z zookeeperConnection) CreateClusterAdmin() (sarama.ClusterAdmin, error) {
	client, err := GetClientFromZookeeper(z, args.GlobalArgs.PreferredListener)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %s", err)
	}

	return sarama.NewClusterAdminFromClient(client)
}

type zookeeperLogger struct{}

func (z zookeeperLogger) Printf(format string, args ...interface{}) {
	log.Debug(format, args...)
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
			return nil, err
		}
	}

	return zookeeperConnection{zkConn}, nil
}

func GetBrokerList(zkConn Connection, preferredListener string) ([]*connection.Broker, error) {
	// Get a list of brokers
	brokerIDs, _, err := zkConn.Children(Path("/brokers/ids"))
	if err != nil {
		return nil, fmt.Errorf("unable to get broker ID from Zookeeper path %s: %s", Path("/brokers/ids"), err)
	}

	brokers := make([]*connection.Broker, 0, len(brokerIDs))
	for _, id := range brokerIDs {
		broker, err := GetBroker(zkConn, id, preferredListener)
		if err != nil {
			log.Error("Failed to get JMX connection info from broker id %s: %s", id, err)
			continue
		}
		brokers = append(brokers, broker)
	}

	return brokers, nil
}

func GetBroker(zkConn Connection, id, preferredListener string) (*connection.Broker, error) {
	// Query Zookeeper for broker information
	rawBrokerJSON, _, err := zkConn.Get(Path(fmt.Sprintf("/brokers/ids/%s", id)))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve broker information: %w", err)
	}

	// Parse the JSON returned by Zookeeper
	type brokerJSONDecoder struct {
		Host        string
		JMXPort     int               `json:"jmx_port"`
		ProtocolMap map[string]string `json:"listener_security_protocol_map"`
		Endpoints   []string          `json:"endpoints"`
	}
	var brokerDecoded brokerJSONDecoder
	err = json.Unmarshal(rawBrokerJSON, &brokerDecoded)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal broker information from zookeeper: %w", err)
	}

	// Go through the list of brokers until we find one that uses a protocol we know how to handle
	var saramaBroker *sarama.Broker
	for _, endpoint := range brokerDecoded.Endpoints {
		listener, host, port, err := parseEndpoint(endpoint)
		if err != nil {
			log.Error("Failed to parse endpoint '%s' from zookeeper: %s", endpoint, err)
			continue
		}

		// Skip this endpoint if it doesn't match the configured listener
		if preferredListener != "" && preferredListener != listener {
			log.Debug("Skipping endpoint '%s' because it doesn't match the preferredListener configured")
			continue
		}

		// Check that the protocol map
		protocol, ok := brokerDecoded.ProtocolMap[listener]
		if !ok {
			log.Error("Listener '%s' was not found in the protocol map")
			continue
		}

		newBroker, err := connection.NewBroker(host, port, protocol)
		if err != nil {
			log.Warn("Failed creating client: %s")
			continue
		}

		saramaBroker = newBroker
	}

	if saramaBroker == nil {
		log.Error("Found no supported endpoint that successfully connected to broker with host %s", brokerDecoded.Host)
	}

	newBroker := &connection.Broker{
		Broker:      saramaBroker,
		JMXPort:     brokerDecoded.JMXPort,
		Host:        brokerDecoded.Host,
		JMXUser:     args.GlobalArgs.DefaultJMXUser,
		JMXPassword: args.GlobalArgs.DefaultJMXPassword,
		ID:          id,
	}

	return newBroker, nil
}

func GetClientFromZookeeper(zkConn Connection, preferredListener string) (sarama.Client, error) {
	// Get a list of brokers
	brokerIDs, _, err := zkConn.Children(Path("/brokers/ids"))
	if err != nil {
		log.Info(Path("/brokers/ids"))
		return nil, fmt.Errorf("unable to get broker ID from Zookeeper: %s", err.Error())
	}

	errors := make([]error, 0)
	for _, id := range brokerIDs {
		client, err := GetClientToBrokerFromZookeeper(zkConn, preferredListener, id)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		return client, nil
	}

	return nil, fmt.Errorf("could not connect to any broker: errors: %v", errors)
}

func GetClientToBrokerFromZookeeper(zkConn Connection, preferredListener string, brokerID string) (sarama.Client, error) {
	// Query Zookeeper for broker information
	rawBrokerJSON, _, err := zkConn.Get(Path("/brokers/ids/" + brokerID))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve broker information: %w", err)
	}

	// Parse the JSON returned by Zookeeper
	type brokerJSONDecoder struct {
		ProtocolMap map[string]string `json:"listener_security_protocol_map"`
		Endpoints   []string          `json:"endpoints"`
	}
	var brokerDecoded brokerJSONDecoder
	err = json.Unmarshal(rawBrokerJSON, &brokerDecoded)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal broker information from zookeeper: %w", err)
	}

	// Go through the list of brokers until we find one that uses a protocol we know how to handle
	for _, endpoint := range brokerDecoded.Endpoints {
		listener, host, port, err := parseEndpoint(endpoint)
		if err != nil {
			log.Error("Failed to parse endpoint '%s' from zookeeper: %s", endpoint, err)
			continue
		}

		// Skip this endpoint if it doesn't match the configured listener
		if preferredListener != "" && preferredListener != listener {
			log.Debug("Skipping endpoint '%s' because it doesn't match the preferredListener configured")
			continue
		}

		// Check that the protocol map
		protocol, ok := brokerDecoded.ProtocolMap[listener]
		if !ok {
			log.Error("Listener '%s' was not found in the protocol map")
			continue
		}

		client, err := connection.NewClient(host, port, protocol)
		if err != nil {
			log.Warn("Failed creating client: %s")
			continue
		}

		return client, nil
	}

	return nil, errors.New("no brokers found that I know how to connect to")
}

// parseEndpoint takes a broker endpoint from zookeeper and parses it into its listener, host, and port components
func parseEndpoint(endpoint string) (listener, host string, port int, err error) {
	re := regexp.MustCompile(`([A-Za-z_]+)://([^:]+):(\d+)`)
	matches := re.FindStringSubmatch(endpoint)
	if matches == nil {
		return "", "", 0, errors.New("regex pattern did not match endpoint")
	}

	port, err = strconv.Atoi(matches[3])
	if err != nil {
		return "", "", 0, fmt.Errorf("failed parsing port as int: %s", err)
	}

	return matches[1], matches[2], port, nil
}
