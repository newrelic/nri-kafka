//go:build integration
// +build integration

package integration

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	)
}

func TestKafkaIntegration_zookeeper(t *testing.T) {
	stdout, stderr, err := runIntegration(t, zookeeperDiscoverConfig)

	assert.NotNil(t, stderr, "unexpected stderr")
	assert.NoError(t, err, "Unexpected error")

	schemaPath := filepath.Join("json-schema-files", "kafka-schema.json")
	err = jsonschema.Validate(schemaPath, stdout)
	assert.NoError(t, err, "The output of kafka integration doesn't have expected format.")
	for _, topic := range topicNames {
		assert.Contains(t, stdout, topic, fmt.Sprintf("The output doesn't have the topic %s", topic))
	}
}

func TestKafkaIntegration_zookeeper_with_topicSourceZookeeper(t *testing.T) {
	zookeeperDiscoverConfigTopicSourceZookeeper := func(command []string) []string {
		return append(zookeeperDiscoverConfig(command), "--topic_source", "zookeeper")
	}

	stdout, stderr, err := runIntegration(t, zookeeperDiscoverConfigTopicSourceZookeeper)

	assert.NotNil(t, stderr, "unexpected stderr")
	assert.NoError(t, err, "Unexpected error")

	schemaPath := filepath.Join("json-schema-files", "kafka-schema.json")
	err = jsonschema.Validate(schemaPath, stdout)
	assert.NoError(t, err, "The output of kafka integration doesn't have expected format.")
	for _, topic := range topicNames {
		assert.Contains(t, stdout, topic, fmt.Sprintf("The output doesn't have the topic %s", topic))
	}
}

func TestKafkaIntegration_bootstrap(t *testing.T) {
	stdout, stderr, err := runIntegration(t, bootstrapDiscoverConfig)

	assert.NotNil(t, stderr, "unexpected stderr")
	assert.NoError(t, err, "Unexpected error")

	schemaPath := filepath.Join("json-schema-files", "kafka-schema.json")
	err = jsonschema.Validate(schemaPath, stdout)
	assert.NoError(t, err, "The output of kafka integration doesn't have expected format.")
	for _, topic := range topicNames {
		assert.Contains(t, stdout, topic, fmt.Sprintf("The output doesn't have the topic %s", topic))
	}
}

func TestKafkaIntegration_bootstrap_with_topicSourceZookeeper(t *testing.T) {
	bootstrapDiscoverConfigConfigTopicSourceZookeeper := func(command []string) []string {
		return append(bootstrapDiscoverConfig(command), "--topic_source", "zookeeper")
	}

	stdout, stderr, err := runIntegration(t, bootstrapDiscoverConfigConfigTopicSourceZookeeper)

	assert.NotNil(t, stderr, "unexpected stderr")
	assert.NoError(t, err, "Unexpected error")

	schemaPath := filepath.Join("json-schema-files", "kafka-schema.json")
	err = jsonschema.Validate(schemaPath, stdout)
	assert.NoError(t, err, "The output of kafka integration doesn't have expected format.")
	for _, topic := range topicNames {
		assert.Contains(t, stdout, topic, fmt.Sprintf("The output doesn't have the topic %s", topic))
	}
}

func TestKafkaIntegration_bootstrap_topicBucket(t *testing.T) {
	bootstrapDiscoverConfigConfigTopicBucket := func(command []string) []string {
		return append(bootstrapDiscoverConfig(command), "--topic_bucket", "3/3")
	}

	stdout, stderr, err := runIntegration(t, bootstrapDiscoverConfigConfigTopicBucket)

	assert.NotNil(t, stderr, "unexpected stderr")
	assert.NoError(t, err, "Unexpected error")

	schemaPath := filepath.Join("json-schema-files", "kafka-schema.json")
	err = jsonschema.Validate(schemaPath, stdout)
	assert.NoError(t, err, "The output of kafka integration doesn't have expected format.")

	var topicsCount int
	for _, topic := range topicNames {
		if strings.Contains(stdout, topic) {
			topicsCount++
		}
	}
	assert.Equal(t, 1, topicsCount)
}

func TestKafkaIntegration_bootstrap_localOnlyCollection(t *testing.T) {
	bootstrapDiscoverConfigLocalOnlyCollection := func(command []string) []string {
		return append(bootstrapDiscoverConfig(command), "--local_only_collection")
	}

	stdout, stderr, err := runIntegration(t, bootstrapDiscoverConfigLocalOnlyCollection)

	assert.NotNil(t, stderr, "unexpected stderr")
	assert.NoError(t, err, "Unexpected error")

	schemaPath := filepath.Join("json-schema-files", "kafka-schema-only-local.json")
	err = jsonschema.Validate(schemaPath, stdout)
	assert.NoError(t, err, "The output of kafka integration doesn't have expected format.")
	for _, topic := range topicNames {
		assert.Contains(t, stdout, topic, fmt.Sprintf("The output doesn't have the topic %s", topic))
	}
}

func TestKafkaIntegration_bootstrap_metrics(t *testing.T) {
	bootstrapDiscoverConfigMetrics := func(command []string) []string {
		return append(bootstrapDiscoverConfig(command), "--metrics")
	}

	stdout, stderr, err := runIntegration(t, bootstrapDiscoverConfigMetrics)

	assert.NotNil(t, stderr, "unexpected stderr")
	assert.NoError(t, err, "Unexpected error")

	schemaPath := filepath.Join("json-schema-files", "kafka-schema-metrics.json")
	err = jsonschema.Validate(schemaPath, stdout)
	assert.NoError(t, err, "The output of kafka integration doesn't have expected format.")
	for _, topic := range topicNames {
		assert.Contains(t, stdout, topic, fmt.Sprintf("The output doesn't have the topic %s", topic))
	}
}

func TestKafkaIntegration_bootstrap_inventory(t *testing.T) {
	bootstrapDiscoverConfigInventory := func(command []string) []string {
		return append(bootstrapDiscoverConfig(command), "--inventory")
	}

	stdout, stderr, err := runIntegration(t, bootstrapDiscoverConfigInventory)

	assert.NotNil(t, stderr, "unexpected stderr")
	assert.NoError(t, err, "Unexpected error")

	schemaPath := filepath.Join("json-schema-files", "kafka-schema-inventory.json")
	err = jsonschema.Validate(schemaPath, stdout)
	assert.NoError(t, err, "The output of kafka integration doesn't have expected format.")
	for _, topic := range topicNames {
		assert.Contains(t, stdout, topic, fmt.Sprintf("The output doesn't have the topic %s", topic))
	}
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

func TestKafkaIntegration_producer_test(t *testing.T) {
	consumerConfig := func(command []string) []string {
		return append(
			command,
			"--producers", "[{\"name\": \"kafka_dummy_producer\", \"host\": \"kafka_dummy_producer\", \"port\": 1089}]",
		)
	}

	stdout, stderr, err := runIntegration(t, consumerConfig)
	assert.NotNil(t, stderr, "unexpected stderr")
	assert.NoError(t, err, "Unexpected error")

	schemaPath := filepath.Join("json-schema-files", "kafka-schema-producer.json")
	err = jsonschema.Validate(schemaPath, stdout)
	assert.NoError(t, err, "The output of kafka integration doesn't have expected format.")
}

func TestKafkaIntegration_consumer_test(t *testing.T) {
	consumerConfig := func(command []string) []string {
		return append(
			command,
			"--consumers", "[{\"host\": \"kafka_dummy_consumer\", \"port\": 1087},{\"name\": \"kafka_dummy_consumer2\", \"host\": \"kafka_dummy_consumer2\", \"port\": 1088}]",
		)
	}

	stdout, stderr, err := runIntegration(t, consumerConfig)
	assert.NotNil(t, stderr, "unexpected stderr")
	assert.NoError(t, err, "Unexpected error")

	schemaPath := filepath.Join("json-schema-files", "kafka-schema-consumer.json")
	err = jsonschema.Validate(schemaPath, stdout)
	assert.NoError(t, err, "The output of kafka integration doesn't have expected format.")
}
