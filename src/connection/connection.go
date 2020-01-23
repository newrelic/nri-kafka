package connection

import (
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/Shopify/sarama"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/nri-kafka/src/args"
)

// Broker TODO
type Broker struct {
	JMXPort     int
	JMXUser     string
	JMXPassword string
	Host        string
	*sarama.Broker
}

func (b *Broker) Entity(i *integration.Integration) (*integration.Entity, error) {
	clusterIDAttr := integration.NewIDAttribute("clusterName", args.GlobalArgs.ClusterName)
	return i.Entity(b.Addr(), "ka-broker", clusterIDAttr)
}

func NewBroker(host string, port int, protocol string) (*sarama.Broker, error) {
	address := fmt.Sprintf("%s:%d", host, port)

	switch protocol {
	case "PLAINTEXT":
		broker := sarama.NewBroker(address)
		err := broker.Open(newPlaintextConfig())
		if err != nil {
			return nil, fmt.Errorf("failed opening connection: %w", err)
		}
		connected, err := broker.Connected()
		if err != nil {
			return nil, fmt.Errorf("failed checking if connection opened successfully: %w", err)
		}
		if !connected {
			return nil, errors.New("broker is not connected")
		}
		return broker, nil
	case "SSL":
		broker := sarama.NewBroker(address)
		err := broker.Open(newSSLConfig())
		if err != nil {
			return nil, fmt.Errorf("failed opening connection: %w", err)
		}
		connected, err := broker.Connected()
		if err != nil {
			return nil, fmt.Errorf("failed checking if connection opened successfully: %w", err)
		}
		if !connected {
			return nil, errors.New("broker is not connected")
		}
		return broker, nil
	case "SASL_PLAINTEXT":
		return nil, fmt.Errorf("skipping %s://%s:%d because it uses unsupported protocol %s", protocol, host, port, protocol)
	case "SASL_SSL":
		return nil, fmt.Errorf("skipping %s://%s:%d because it uses unsupported protocol %s", protocol, host, port, protocol)
	default:
		return nil, fmt.Errorf("skipping %s://%s:%d because it uses unknown protocol %s", protocol, host, port, protocol)
	}

}

func NewClient(host string, port int, protocol string) (sarama.Client, error) {
	address := fmt.Sprintf("%s:%d", host, port)

	switch protocol {
	case "PLAINTEXT":
		return sarama.NewClient([]string{address}, newPlaintextConfig())
	case "SSL":
		return sarama.NewClient([]string{address}, newSSLConfig())
	case "SASL_PLAINTEXT":
		return nil, fmt.Errorf("skipping %s://%s:%d because it uses unsupported protocol %s", protocol, host, port, protocol)
	case "SASL_SSL":
		return nil, fmt.Errorf("skipping %s://%s:%d because it uses unsupported protocol %s", protocol, host, port, protocol)
	default:
		return nil, fmt.Errorf("skipping %s://%s:%d because it uses unknown protocol %s", protocol, host, port, protocol)
	}

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
