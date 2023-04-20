//go:build integration
// +build integration

package integration

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-kafka/tests/integration/helpers"
	"github.com/newrelic/nri-kafka/tests/integration/jsonschema"
	"github.com/stretchr/testify/assert"
)

const (
	BROKER_CONN_MAX_RETRIES   = 60
	ENSURE_TOPICS_MAX_RETRIES = 60
	KAFKA1_PORT               = "19092"
	BROKERS_IN_CLUSTER        = 3
	NUMBER_OF_TOPICS          = 4
)

var (
	iName = "kafka"

	topicNames = []string{"topicA", "topicB", "topicC", "__consumer_offsets"}

	defaultContainer = "integration_nri_kafka_1"
	defaultBinPath   = "/nri-kafka"

	// cli flags
	container = flag.String("container", defaultContainer, "container where the integration is installed")
	binPath   = flag.String("bin", defaultBinPath, "Integration binary path")
)

// Returns the standard output, or fails testing if the command returned an error
func runIntegration(t *testing.T, config func([]string) []string) (string, string, error) {
	t.Helper()

	command := make([]string, 0)
	command = append(command, *binPath)
	command = config(command)

	stdout, stderr, err := helpers.ExecInContainer(*container, command)

	if stderr != "" {
		log.Debug("Integration command Standard Error: %s", stderr)
	}

	return stdout, stderr, err
}

func ensureBrokerClusterReady(tries int) {
	config := sarama.NewConfig()
	config.Version = sarama.V1_0_0_0
	config.ClientID = "nri-kafka"

	saramaBroker := sarama.NewBroker("localhost:" + KAFKA1_PORT)
	err := saramaBroker.Open(config)
	if err != nil {
		tries += 1
		if tries > BROKER_CONN_MAX_RETRIES {
			log.Error("Failed opening connection: %s", err)
			os.Exit(1)
		}
		saramaBroker.Close()
		time.Sleep(500 * time.Millisecond)
		ensureBrokerClusterReady(tries)
	}

	connected, err := saramaBroker.Connected()
	if err != nil {
		tries += 1
		if tries > BROKER_CONN_MAX_RETRIES {
			log.Error("failed checking if connection opened successfully: %s", err)
			os.Exit(1)
		}
		saramaBroker.Close()
		time.Sleep(500 * time.Millisecond)
		ensureBrokerClusterReady(tries)
	}
	if !connected {
		tries += 1
		if tries > BROKER_CONN_MAX_RETRIES {
			log.Error("Broker is not connected: %s", err)
			os.Exit(1)
		}
		saramaBroker.Close()
		time.Sleep(500 * time.Millisecond)
		ensureBrokerClusterReady(tries)
	}

	metadata, err := saramaBroker.GetMetadata(&sarama.MetadataRequest{})
	if err != nil {
		tries += 1
		if tries > BROKER_CONN_MAX_RETRIES {
			log.Error("failed to get metadata from broker: %s", err)
			os.Exit(1)
		}
		saramaBroker.Close()
		time.Sleep(500 * time.Millisecond)
		ensureBrokerClusterReady(tries)
	}

	if metadata == nil ||
		metadata.Brokers == nil ||
		(metadata != nil && metadata.Brokers != nil && len(metadata.Brokers) < BROKERS_IN_CLUSTER) {
		tries += 1
		if tries > BROKER_CONN_MAX_RETRIES {
			log.Error("failed to start all brokers")
			os.Exit(1)
		}
		saramaBroker.Close()
		time.Sleep(500 * time.Millisecond)
		ensureBrokerClusterReady(tries)
	}
	saramaBroker.Close()
}

func ensureTopicsCreated(tries int) {
	config := sarama.NewConfig()
	config.Version = sarama.V1_0_0_0
	config.ClientID = "nri-kafka"

	client, err := sarama.NewClient([]string{"localhost:" + KAFKA1_PORT}, config)
	if err != nil {
		tries += 1
		if tries > ENSURE_TOPICS_MAX_RETRIES {
			log.Error("failed to start client: %s", err)
			os.Exit(1)
		}
		client.Close()
		time.Sleep(500 * time.Millisecond)
		ensureTopicsCreated(tries)
	}

	topics, err := client.Topics()
	if err != nil || len(topics) < NUMBER_OF_TOPICS {
		tries += 1
		if tries > ENSURE_TOPICS_MAX_RETRIES {
			log.Error("failed to get topics list")
			os.Exit(1)
		}
		client.Close()
		time.Sleep(500 * time.Millisecond)
		ensureTopicsCreated(tries)
	}
	client.Close()
}

func TestMain(m *testing.M) {
	flag.Parse()
	ensureBrokerClusterReady(0)
	ensureTopicsCreated(0)
	result := m.Run()
	os.Exit(result)
}

func zookeeperDiscoverConfig(command []string) []string {
	return append(
		command,
		"--cluster_name", "kfk-cluster-zookeeper",
		"--zookeeper_hosts", `[{"host": "zookeeper", "port": 2181}]`,
		"--autodiscover_strategy", "zookeeper",
		"--topic_mode", "all",
	)
}

func bootstrapDiscoverConfig(command []string) []string {
	return append(
		command,
		"--cluster_name", "kfk-cluster-bootstrap",
		"--autodiscover_strategy", "bootstrap",
		"--bootstrap_broker_host", "kafka1",
		"--bootstrap_broker_kafka_port", "9092",
		"--bootstrap_broker_kafka_protocol", "PLAINTEXT",
		"--bootstrap_broker_jmx_port", "1099",
		"--bootstrap_broker_jmx_user", "admin",
		"--bootstrap_broker_jmx_password", "nrone",
		"--topic_mode", "all",
		"--collect_topic_size",
		"--collect_topic_offset",
	)
}

func TestKafkaIntegration_consumer_offset(t *testing.T) {
	bootstrapDiscoverConfigInventory := func(command []string) []string {
		return append(
			bootstrapDiscoverConfig(command),
			"--consumer_offset",
			"--consumer_group_regex", ".*",
		)
	}

	stdout, stderr, err := runIntegration(t, bootstrapDiscoverConfigInventory)

	assert.NotNil(t, stderr, "unexpected stderr")
	assert.NoError(t, err, "Unexpected error")

	schemaPath := filepath.Join("json-schema-files", "kafka-schema-consumer-offset.json")
	err = jsonschema.Validate(schemaPath, stdout)
	assert.NoError(t, err, "The output of kafka integration doesn't have expected format.")
}
