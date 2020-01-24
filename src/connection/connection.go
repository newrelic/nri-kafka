package connection

import (
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/Shopify/sarama"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/nri-kafka/src/args"
)

// Broker is a struct containing all the information to collect from both Kafka and from JMX
type Broker struct {
	Config      *sarama.Config
	JMXPort     int
	JMXUser     string
	JMXPassword string
	Host        string
	ID          string
	*sarama.Broker
}

func (b *Broker) Entity(i *integration.Integration) (*integration.Entity, error) {
	clusterIDAttr := integration.NewIDAttribute("clusterName", args.GlobalArgs.ClusterName)
	brokerIDAttr := integration.NewIDAttribute("brokerID", string(b.ID))
	return i.Entity(b.Addr(), "ka-broker", clusterIDAttr, brokerIDAttr)
}

func NewBroker(brokerArgs *args.BrokerHost) (*Broker, error) {
	address := fmt.Sprintf("%s:%d", brokerArgs.Host, brokerArgs.KafkaPort)

	switch brokerArgs.KafkaProtocol {
	case "PLAINTEXT":
		saramaBroker := sarama.NewBroker(address)
		config := newPlaintextConfig()
		err := saramaBroker.Open(newPlaintextConfig())
		if err != nil {
			return nil, fmt.Errorf("failed opening connection: %w", err)
		}
		connected, err := saramaBroker.Connected()
		if err != nil {
			return nil, fmt.Errorf("failed checking if connection opened successfully: %w", err)
		}
		if !connected {
			return nil, errors.New("broker is not connected")
		}

		// TODO figure out how to get the ID from the broker. ID() returns -1
		newBroker := &Broker{
			Broker:      saramaBroker,
			Host:        brokerArgs.Host,
			JMXPort:     brokerArgs.JMXPort,
			JMXUser:     brokerArgs.JMXUser,
			JMXPassword: brokerArgs.JMXPassword,
			ID:          fmt.Sprintf("%d", saramaBroker.ID()),
			Config:      config,
		}
		return newBroker, nil
	case "SSL":
		saramaBroker := sarama.NewBroker(address)
		config := newSSLConfig()
		err := saramaBroker.Open(newSSLConfig())
		if err != nil {
			return nil, fmt.Errorf("failed opening connection: %w", err)
		}
		connected, err := saramaBroker.Connected()
		if err != nil {
			return nil, fmt.Errorf("failed checking if connection opened successfully: %w", err)
		}
		if !connected {
			return nil, errors.New("broker is not connected")
		}
		newBroker := &Broker{
			Broker:      saramaBroker,
			Host:        brokerArgs.Host,
			JMXPort:     brokerArgs.JMXPort,
			JMXUser:     brokerArgs.JMXUser,
			JMXPassword: brokerArgs.JMXPassword,
			ID:          fmt.Sprintf("%d", saramaBroker.ID()),
			Config:      config,
		}
		return newBroker, nil
	case "SASL_PLAINTEXT":
		return nil, fmt.Errorf("skipping %s://%s:%d because it uses unsupported protocol '%s'", brokerArgs.KafkaProtocol, brokerArgs.Host, brokerArgs.KafkaPort, brokerArgs.KafkaProtocol)
	case "SASL_SSL":
		return nil, fmt.Errorf("skipping %s://%s:%d because it uses unsupported protocol '%s'", brokerArgs.KafkaProtocol, brokerArgs.Host, brokerArgs.KafkaPort, brokerArgs.KafkaProtocol)
	default:
		return nil, fmt.Errorf("skipping %s://%s:%d because it uses unknown protocol '%s'", brokerArgs.KafkaProtocol, brokerArgs.Host, brokerArgs.KafkaPort, brokerArgs.KafkaProtocol)
	}

}

func NewSaramaClientFromBrokerList(brokers []*Broker) (sarama.Client, error) {
	if len(brokers) == 0 {
		return nil, errors.New("cannot create sarama client with no brokers")
	}

	brokerAddresses := make([]string, 0, len(brokers))
	for _, broker := range brokers {
		brokerAddresses = append(brokerAddresses, broker.Addr())
	}

	return sarama.NewClient(brokerAddresses, brokers[0].Config)
}

func newPlaintextConfig() *sarama.Config {
	config := sarama.NewConfig()
	config.Version = sarama.V2_0_0_0

	return config
}

func newSSLConfig() *sarama.Config {
	config := sarama.NewConfig()
	config.Net.TLS.Enable = true
	config.Net.TLS.Config = &tls.Config{
		InsecureSkipVerify: true,
	}
	config.Version = sarama.V2_0_0_0

	return config
}
