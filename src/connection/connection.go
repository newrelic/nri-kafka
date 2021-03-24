//go:generate mockery -name=Client -name=SaramaBroker

// Package connection implements connection code
package connection

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"

	"github.com/Shopify/sarama"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/zookeeper"
)

// Client is a wrapper around sarama.Client so that we can generate mocks
// See sarama.Client for documentation
type Client interface {
	Config() *sarama.Config
	Controller() (*sarama.Broker, error)
	Brokers() []*sarama.Broker
	Topics() ([]string, error)
	Partitions(topic string) ([]int32, error)
	WritablePartitions(topic string) ([]int32, error)
	Leader(topic string, partitionID int32) (*sarama.Broker, error)
	Replicas(topic string, partitionID int32) ([]int32, error)
	InSyncReplicas(topic string, partitionID int32) ([]int32, error)
	OfflineReplicas(topic string, partitionID int32) ([]int32, error)
	RefreshMetadata(topics ...string) error
	GetOffset(topic string, partitionID int32, time int64) (int64, error)
	Coordinator(consumerGroup string) (*sarama.Broker, error)
	RefreshCoordinator(consumerGroup string) error
	RefreshController() (*sarama.Broker, error)
	InitProducerID() (*sarama.InitProducerIDResponse, error)
	Close() error
	Closed() bool
}

// SaramaBroker is an interface over sarama.Broker for mocking
type SaramaBroker interface {
	AddOffsetsToTxn(request *sarama.AddOffsetsToTxnRequest) (*sarama.AddOffsetsToTxnResponse, error)
	AddPartitionsToTxn(request *sarama.AddPartitionsToTxnRequest) (*sarama.AddPartitionsToTxnResponse, error)
	Addr() string
	AlterConfigs(request *sarama.AlterConfigsRequest) (*sarama.AlterConfigsResponse, error)
	ApiVersions(request *sarama.ApiVersionsRequest) (*sarama.ApiVersionsResponse, error)
	Close() error
	CommitOffset(request *sarama.OffsetCommitRequest) (*sarama.OffsetCommitResponse, error)
	Connected() (bool, error)
	CreateAcls(request *sarama.CreateAclsRequest) (*sarama.CreateAclsResponse, error)
	CreatePartitions(request *sarama.CreatePartitionsRequest) (*sarama.CreatePartitionsResponse, error)
	CreateTopics(request *sarama.CreateTopicsRequest) (*sarama.CreateTopicsResponse, error)
	DeleteAcls(request *sarama.DeleteAclsRequest) (*sarama.DeleteAclsResponse, error)
	DeleteGroups(request *sarama.DeleteGroupsRequest) (*sarama.DeleteGroupsResponse, error)
	DeleteRecords(request *sarama.DeleteRecordsRequest) (*sarama.DeleteRecordsResponse, error)
	DeleteTopics(request *sarama.DeleteTopicsRequest) (*sarama.DeleteTopicsResponse, error)
	DescribeAcls(request *sarama.DescribeAclsRequest) (*sarama.DescribeAclsResponse, error)
	DescribeConfigs(request *sarama.DescribeConfigsRequest) (*sarama.DescribeConfigsResponse, error)
	DescribeGroups(request *sarama.DescribeGroupsRequest) (*sarama.DescribeGroupsResponse, error)
	EndTxn(request *sarama.EndTxnRequest) (*sarama.EndTxnResponse, error)
	Fetch(request *sarama.FetchRequest) (*sarama.FetchResponse, error)
	FetchOffset(request *sarama.OffsetFetchRequest) (*sarama.OffsetFetchResponse, error)
	FindCoordinator(request *sarama.FindCoordinatorRequest) (*sarama.FindCoordinatorResponse, error)
	GetAvailableOffsets(request *sarama.OffsetRequest) (*sarama.OffsetResponse, error)
	GetConsumerMetadata(request *sarama.ConsumerMetadataRequest) (*sarama.ConsumerMetadataResponse, error)
	GetMetadata(request *sarama.MetadataRequest) (*sarama.MetadataResponse, error)
	Heartbeat(request *sarama.HeartbeatRequest) (*sarama.HeartbeatResponse, error)
	ID() int32
	InitProducerID(request *sarama.InitProducerIDRequest) (*sarama.InitProducerIDResponse, error)
	JoinGroup(request *sarama.JoinGroupRequest) (*sarama.JoinGroupResponse, error)
	LeaveGroup(request *sarama.LeaveGroupRequest) (*sarama.LeaveGroupResponse, error)
	ListGroups(request *sarama.ListGroupsRequest) (*sarama.ListGroupsResponse, error)
	Open(conf *sarama.Config) error
	Produce(request *sarama.ProduceRequest) (*sarama.ProduceResponse, error)
	Rack() string
	SyncGroup(request *sarama.SyncGroupRequest) (*sarama.SyncGroupResponse, error)
	TxnOffsetCommit(request *sarama.TxnOffsetCommitRequest) (*sarama.TxnOffsetCommitResponse, error)
}

// Broker is a struct containing all the information to collect from both Kafka and from JMX
type Broker struct {
	Config      *sarama.Config
	JMXPort     int
	JMXUser     string
	JMXPassword string
	Host        string
	ID          string
	SaramaBroker
}

// Entity gets the entity object for the broker
func (b *Broker) Entity(i *integration.Integration) (*integration.Entity, error) {
	clusterIDAttr := integration.NewIDAttribute("clusterName", args.GlobalArgs.ClusterName)
	brokerIDAttr := integration.NewIDAttribute("brokerID", string(b.ID))
	return i.Entity(b.Addr(), "ka-broker", clusterIDAttr, brokerIDAttr)
}

// NewBroker creates a new broker
func NewBroker(brokerArgs *args.BrokerHost) (*Broker, error) {
	address := fmt.Sprintf("%s:%d", brokerArgs.Host, brokerArgs.KafkaPort)
	protocol := brokerArgs.KafkaProtocol

	config := sarama.NewConfig()
	config.Version = args.GlobalArgs.KafkaVersion
	config.ClientID = "nri-kafka"

	switch protocol {
	case "PLAINTEXT":
		saramaBroker := sarama.NewBroker(address)
		err := saramaBroker.Open(config)
		if err != nil {
			return nil, fmt.Errorf("failed opening connection: %s", err)
		}
		connected, err := saramaBroker.Connected()
		if err != nil {
			return nil, fmt.Errorf("failed checking if connection opened successfully: %s", err)
		}
		if !connected {
			return nil, errors.New("broker is not connected")
		}

		// TODO figure out how to get the ID from the broker. ID() returns -1
		newBroker := &Broker{
			SaramaBroker: saramaBroker,
			Host:         brokerArgs.Host,
			JMXPort:      brokerArgs.JMXPort,
			JMXUser:      brokerArgs.JMXUser,
			JMXPassword:  brokerArgs.JMXPassword,
			ID:           fmt.Sprintf("%d", saramaBroker.ID()),
			Config:       config,
		}
		return newBroker, nil
	case "SSL":
		saramaBroker := sarama.NewBroker(address)

		err := configureTLS(config)
		if err != nil {
			return nil, fmt.Errorf("build TLS config: %v", err)
		}

		err = saramaBroker.Open(config)
		if err != nil {
			return nil, fmt.Errorf("failed opening connection: %s", err)
		}
		connected, err := saramaBroker.Connected()
		if err != nil {
			return nil, fmt.Errorf("failed checking if connection opened successfully: %s", err)
		}
		if !connected {
			return nil, errors.New("broker is not connected")
		}
		newBroker := &Broker{
			SaramaBroker: saramaBroker,
			Host:         brokerArgs.Host,
			JMXPort:      brokerArgs.JMXPort,
			JMXUser:      brokerArgs.JMXUser,
			JMXPassword:  brokerArgs.JMXPassword,
			ID:           fmt.Sprintf("%d", saramaBroker.ID()),
			Config:       config,
		}
		return newBroker, nil
	case "SASL_PLAINTEXT", "SASL_SSL":
		var err error
		saramaBroker := sarama.NewBroker(address)

		err = configureSASL(config)
		if err != nil {
			return nil, fmt.Errorf("create %s config: %v", protocol, err)
		}

		if protocol == "SASL_SSL" {
			err = configureTLS(config)
			if err != nil {
				return nil, fmt.Errorf("build TLS config: %v", err)
			}
		}

		err = saramaBroker.Open(config)
		if err != nil {
			return nil, fmt.Errorf("failed opening connection: %s", err)
		}
		connected, err := saramaBroker.Connected()
		if err != nil {
			return nil, fmt.Errorf("failed checking if connection opened successfully: %s", err)
		}
		if !connected {
			return nil, errors.New("broker is not connected")
		}

		// TODO figure out how to get the ID from the broker. ID() returns -1
		newBroker := &Broker{
			SaramaBroker: saramaBroker,
			Host:         brokerArgs.Host,
			JMXPort:      brokerArgs.JMXPort,
			JMXUser:      brokerArgs.JMXUser,
			JMXPassword:  brokerArgs.JMXPassword,
			ID:           fmt.Sprintf("%d", saramaBroker.ID()),
			Config:       config,
		}
		return newBroker, nil
	default:
		return nil, fmt.Errorf("skipping %s://%s:%d because it uses unknown protocol '%s'", brokerArgs.KafkaProtocol, brokerArgs.Host, brokerArgs.KafkaPort, brokerArgs.KafkaProtocol)
	}

}

// NewSaramaClientFromBrokerList creates a new Client from a list of brokers
func NewSaramaClientFromBrokerList(brokers []*Broker) (Client, error) {
	if len(brokers) == 0 {
		return nil, errors.New("cannot create sarama client with no brokers")
	}

	brokerAddresses := make([]string, 0, len(brokers))
	for _, broker := range brokers {
		brokerAddresses = append(brokerAddresses, broker.Addr())
	}

	log.Debug("Creating a new client to brokers: %v", brokerAddresses)
	client, err := sarama.NewClient(brokerAddresses, brokers[0].Config)
	if err != nil {
		return nil, err
	}

	return client.(Client), nil
}

func configureSASL(config *sarama.Config) error {
	ga := args.GlobalArgs
	config.Net.SASL.Enable = true

	if ga.SaslMechanism == sarama.SASLTypePlaintext {
		config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		config.Net.SASL.User = ga.SaslUsername
		config.Net.SASL.Password = ga.SaslPassword
	} else if ga.SaslMechanism == "GSSAPI" {
		if ga.SaslGssapiKerberosConfigPath == "" || ga.SaslGssapiKeyTabPath == "" || ga.SaslGssapiRealm == "" || ga.SaslGssapiServiceName == "" || ga.SaslGssapiUsername == "" {
			return fmt.Errorf("all sasl_gssapi_* arguments must be set for GSSAPI auth")
		}

		config.Net.SASL.Mechanism = sarama.SASLTypeGSSAPI
		config.Net.SASL.GSSAPI = sarama.GSSAPIConfig{
			AuthType:           sarama.KRB5_KEYTAB_AUTH,
			Realm:              args.GlobalArgs.SaslGssapiRealm,
			ServiceName:        args.GlobalArgs.SaslGssapiServiceName,
			Username:           args.GlobalArgs.SaslGssapiUsername,
			KeyTabPath:         args.GlobalArgs.SaslGssapiKeyTabPath,
			KerberosConfigPath: args.GlobalArgs.SaslGssapiKerberosConfigPath,
			// enable/disable FAST negotiation that can cause issues with Active Directory
			DisablePAFXFAST: args.GlobalArgs.SaslGssapiDisableFASTNegotiation,
		}
	} else if ga.SaslMechanism == sarama.SASLTypeSCRAMSHA256 {
		config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
		config.Net.SASL.User = ga.SaslUsername
		config.Net.SASL.Password = ga.SaslPassword
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA256} }
	} else if ga.SaslMechanism == sarama.SASLTypeSCRAMSHA512 {
		config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
		config.Net.SASL.User = ga.SaslUsername
		config.Net.SASL.Password = ga.SaslPassword
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA512} }
	} else {
		return fmt.Errorf("Unexpected SASL mechanism '%s'", ga.SaslMechanism)
	}

	return nil
}

func configureTLS(config *sarama.Config) error {
	config.Net.TLS.Enable = true
	config.Net.TLS.Config = &tls.Config{
		InsecureSkipVerify: args.GlobalArgs.TLSInsecureSkipVerify,
	}

	if args.GlobalArgs.TLSCaFile != "" {
		certPool := x509.NewCertPool()
		ca, err := ioutil.ReadFile(args.GlobalArgs.TLSCaFile)
		if err != nil {
			return err
		}
		certPool.AppendCertsFromPEM(ca)
		config.Net.TLS.Config.RootCAs = certPool
	}

	if args.GlobalArgs.TLSCertFile != "" && args.GlobalArgs.TLSKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(args.GlobalArgs.TLSCertFile, args.GlobalArgs.TLSKeyFile)
		if err != nil {
			return err
		}
		config.Net.TLS.Config.Certificates = []tls.Certificate{cert}
	}

	return nil
}

// GetBrokerListFromZookeeper gets a list of brokers from zookeeper
func GetBrokerListFromZookeeper(zkConn zookeeper.Connection, preferredListener string) ([]*Broker, error) {
	// Get a list of brokers
	brokerIDs, _, err := zkConn.Children(zookeeper.Path("/brokers/ids"))
	if err != nil {
		return nil, fmt.Errorf("unable to get broker ID from Zookeeper path %s: %s", zookeeper.Path("/brokers/ids"), err)
	}

	brokers := make([]*Broker, 0, len(brokerIDs))
	for _, id := range brokerIDs {
		broker, err := GetBrokerFromZookeeper(zkConn, id, preferredListener)
		if err != nil {
			log.Error("Failed to get JMX connection info from broker id %s: %s", id, err)
			continue
		}
		brokers = append(brokers, broker)
	}

	return brokers, nil
}

// GetBrokerFromZookeeper gets a broker with given ID from zookeeper
func GetBrokerFromZookeeper(zkConn zookeeper.Connection, id, preferredListener string) (*Broker, error) {
	// Query Zookeeper for broker information
	rawBrokerJSON, _, err := zkConn.Get(zookeeper.Path(fmt.Sprintf("/brokers/ids/%s", id)))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve broker information: %s", err)
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
		return nil, fmt.Errorf("failed to unmarshal broker information from zookeeper: %s", err)
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
			log.Debug("Skipping endpoint '%s' because it doesn't match the preferredListener configured", endpoint)
			continue
		}

		// Check that the protocol map
		protocol, ok := brokerDecoded.ProtocolMap[listener]
		if !ok {
			log.Error("Listener '%s' was not found in the protocol map")
			continue
		}

		brokerConfig := &args.BrokerHost{
			Host:          host,
			KafkaPort:     port,
			KafkaProtocol: protocol,
			JMXPort:       brokerDecoded.JMXPort,
			JMXUser:       args.GlobalArgs.DefaultJMXUser,
			JMXPassword:   args.GlobalArgs.DefaultJMXPassword,
		}
		newBroker, err := NewBroker(brokerConfig)
		if err != nil {
			log.Warn("Failed creating client: %s", err)
			continue
		}
		newBroker.ID = id

		return newBroker, nil
	}

	return nil, fmt.Errorf("found no supported endpoint that successfully connected to broker with host %s", brokerDecoded.Host)
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
