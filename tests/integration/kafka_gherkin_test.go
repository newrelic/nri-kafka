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

	"github.com/cucumber/godog"
	"github.com/newrelic/nri-kafka/tests/integration/jsonschema"
)

const (
	BROKER_CONN_MAX_RETRIES   = 10
	ENSURE_TOPICS_MAX_RETRIES = 20
	KAFKA1_PORT               = "19092"
	BROKERS_IN_CLUSTER        = 3
	NUMBER_OF_TOPICS          = 3
)

var (
	iName = "kafka"

	topicNames = []string{"topicA", "topicB", "topicC"}

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

/*func TestMain(m *testing.M) {
	flag.Parse()
	ensureBrokerClusterReady(0)
	ensureTopicsCreated(0)
	result := m.Run()
	os.Exit(result)
}*/

func zookeeperDiscoverConfig(command []string) []string {
	return append(
		command,
		"--cluster_name", "kfk-cluster-zookeeper",
		"--zookeeper_hosts", `[{"host": "zookeeper", "port": 2181}]`,
		"--autodiscover_strategy", "zookeeper",
		"--collect_broker_topic_data",
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
		"--collect_broker_topic_data",
		"--topic_mode", "all",
	)
}

func TestMain(m *testing.M) {
	flag.Parse()

	status := godog.TestSuite{
		Name:                 "all",
		TestSuiteInitializer: InitializeTestSuite,
		ScenarioInitializer:  InitializeScenario,
	}.Run()

	os.Exit(status)
}

type integrationFeature struct {
	stdout string
	stderr string
}

func (i *integrationFeature) setToAutodiscoverStrategyAndExecuted(strategy string) error {
	command := make([]string, 0)
	command = append(command, *binPath)

	if strategy == "zookeeper" {
		command = zookeeperDiscoverConfig(command)
	} else {
		command = bootstrapDiscoverConfig(command)
	}

	var err error
	i.stdout, i.stderr, err = helpers.ExecInContainer(*container, command)
	if err != nil {
		return fmt.Errorf("unexpected error %w", err)
	}
	return nil
}

func (i *integrationFeature) setToAndExecuted(mode string) error {
	command := make([]string, 0)
	command = append(command, *binPath)

	var err error
	i.stdout, i.stderr, err = helpers.ExecInContainer(*container, append(bootstrapDiscoverConfig(command), fmt.Sprintf("--%s", mode)))
	if err != nil {
		return fmt.Errorf("unexpected error %w", err)
	}
	return nil
}

func (i *integrationFeature) theResponseShouldMatchJSONSchema(schema string) error {
	schemaPath := filepath.Join("json-schema-files", fmt.Sprintf("kafka-schema-%s.json", schema))
	err := jsonschema.Validate(schemaPath, i.stdout)
	if err != nil {
		return fmt.Errorf("the output of kafka integration doesn't have expected format: %w", err)
	}
	return nil
}

func (i *integrationFeature) theResponseShouldHaveTheTopicsMetrics() error {
	for _, topic := range topicNames {
		if !strings.Contains(i.stdout, topic) {
			return fmt.Errorf("the output doesn't have the topic %s", topic)
		}
	}
	return nil
}

func InitializeTestSuite(ctx *godog.TestSuiteContext) {
	ctx.BeforeSuite(func() {
		ensureBrokerClusterReady(0)
		ensureTopicsCreated(0)
	})
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	i := &integrationFeature{}
	ctx.Step(`^a (\d+) node kafka is running and has (\d+) topics$`, func() error { return nil })
	ctx.Step(`set to (zookeeper|bootstrap) autodiscover strategy and executed$`, i.setToAutodiscoverStrategyAndExecuted)
	ctx.Step(`set to (inventory) and executed$`, i.setToAndExecuted)
	ctx.Step(`the response should match the (all|metrics|inventory) json schema$`, i.theResponseShouldMatchJSONSchema)
	ctx.Step(`^the response should have the (\d+) topics metrics$`, i.theResponseShouldHaveTheTopicsMetrics)
}
