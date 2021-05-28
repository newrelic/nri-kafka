// +build integration

package integration

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-kafka/tests/integration/helpers"
	"github.com/newrelic/nri-kafka/tests/integration/jsonschema"
	"github.com/stretchr/testify/assert"
)

var (
	iName = "kafka"

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
		log.Debug("Integration command Standard Error: ", stderr)
	}

	return stdout, stderr, err
}

func ensureKafkaReady() {
	time.Sleep(5 * time.Second)
}

func TestMain(m *testing.M) {
	flag.Parse()
	ensureKafkaReady()
	result := m.Run()
	os.Exit(result)
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
		"--collect_broker_topic_data",
		"--topic_mode", "all",
	)
}

func zookeeperDiscoverConfig(command []string) []string {
	return append(
		command,
		"--cluster_name", "kfk-cluster-zookeeper",
		"--zookeeper_hosts", `[{"host": "zookeeper", "port": 2181}]`,
		"--autodiscover_strategy", "zookeeper",
		"--collect_broker_topic_data", "true",
		"--topic_mode", "all",
		"--topic_bucket", "3/3",
	)
}

func TestKafkaIntegration_bootstrap(t *testing.T) {
	stdout, stderr, err := runIntegration(t, bootstrapDiscoverConfig)

	assert.NotNil(t, stderr, "unexpected stderr")
	assert.NoError(t, err, "Unexpected error")

	schemaPath := filepath.Join("json-schema-files", "kafka-schema.json")
	err = jsonschema.Validate(schemaPath, stdout)
	assert.NoError(t, err, "The output of kafka integration doesn't have expected format.")
}

func TestKafkaIntegration_zookeeper(t *testing.T) {
	stdout, stderr, err := runIntegration(t, zookeeperDiscoverConfig)

	assert.NotNil(t, stderr, "unexpected stderr")
	assert.NoError(t, err, "Unexpected error")

	schemaPath := filepath.Join("json-schema-files", "kafka-schema.json")
	err = jsonschema.Validate(schemaPath, stdout)
	assert.NoError(t, err, "The output of kafka integration doesn't have expected format.")
}
