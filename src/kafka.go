package main

import (
	"sync"

	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/jmx"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-kafka/args"
	"github.com/newrelic/nri-kafka/zookeeper"
)

const (
	integrationName    = "com.newrelic.kafka"
	integrationVersion = "0.1.0"
)

var (
	logger    log.Logger
	kafkaArgs *args.KafkaArguments

	jmxLock sync.Mutex

	// Saved functions to allow easier mocking of jmx functions in tests
	queryFunc    = jmx.Query
	jmxOpenFunc  = jmx.Open
	jmxCloseFunc = jmx.Close
)

func main() {
	var argList args.ArgumentList
	// Create Integration
	kafkaIntegration, err := integration.New(integrationName, integrationVersion, integration.Args(&argList))
	panicOnErr(err)

	// Needs to be after integration creation for args to be set
	logger = kafkaIntegration.Logger()

	// Parse args into structs
	// This has to be after integration creation for defaults to be populated
	kafkaArgs, err = args.ParseArgs(argList)
	panicOnErr(err)

	zkConn, err := zookeeper.NewConnection(kafkaArgs)
	panicOnErr(err)

	// Get topic list
	collectedTopics, err := getTopics(zkConn)
	panicOnErr(err)

	// Setup wait group
	var wg sync.WaitGroup

	// Start all worker pools
	brokerChan := startBrokerPool(3, &wg, zkConn, kafkaIntegration, collectedTopics)
	topicChan := startTopicPool(5, &wg, zkConn)
	consumerChan := startWorkerPool(3, &wg, kafkaIntegration, collectedTopics, consumerWorker)
	producerChan := startWorkerPool(3, &wg, kafkaIntegration, collectedTopics, producerWorker)

	// After all worker pools are created start feeding them.
	// It is important to not start feeding any pool until all are created
	// so that a race condition does not exist between creating all pools and waiting.
	// Run all of theses in their own Go Routine to maximize concurrency
	go feedBrokerPool(zkConn, brokerChan)
	go feedTopicPool(topicChan, kafkaIntegration, collectedTopics)
	go feedWorkerPool(consumerChan, kafkaArgs.Consumers)
	go feedWorkerPool(producerChan, kafkaArgs.Producers)

	wg.Wait()

	panicOnErr(kafkaIntegration.Publish())
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}
