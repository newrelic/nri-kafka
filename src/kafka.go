package main

import (
	"os"
	"sync"

	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-kafka/src/args"
	bc "github.com/newrelic/nri-kafka/src/brokercollect"
	pcc "github.com/newrelic/nri-kafka/src/prodconcollect"
	tc "github.com/newrelic/nri-kafka/src/topiccollect"
	"github.com/newrelic/nri-kafka/src/zookeeper"
)

const (
	integrationName    = "com.newrelic.kafka"
	integrationVersion = "0.1.0"
)

func main() {
	var argList args.ArgumentList
	// Create Integration
	kafkaIntegration, err := integration.New(integrationName, integrationVersion, integration.Args(&argList))
	ExitOnErr(err)

	// Setup logging with verbose
	log.SetupLogging(argList.Verbose)

	// Parse args into structs
	// This has to be after integration creation for defaults to be populated
	args.GlobalArgs, err = args.ParseArgs(argList)
	ExitOnErr(err)

	zkConn, err := zookeeper.NewConnection(args.GlobalArgs)
	ExitOnErr(err)

	// Get topic list
	collectedTopics, err := tc.GetTopics(zkConn)
	ExitOnErr(err)

	// Setup wait group
	var wg sync.WaitGroup

	// Start all worker pools
	brokerChan := bc.StartBrokerPool(3, &wg, zkConn, kafkaIntegration, collectedTopics)
	topicChan := tc.StartTopicPool(5, &wg, zkConn)
	consumerChan := pcc.StartWorkerPool(3, &wg, kafkaIntegration, collectedTopics, pcc.ConsumerWorker)
	producerChan := pcc.StartWorkerPool(3, &wg, kafkaIntegration, collectedTopics, pcc.ProducerWorker)

	// After all worker pools are created start feeding them.
	// It is important to not start feeding any pool until all are created
	// so that a race condition does not exist between creating all pools and waiting.
	// Run all of theses in their own Go Routine to maximize concurrency
	go bc.FeedBrokerPool(zkConn, brokerChan)
	go tc.FeedTopicPool(topicChan, kafkaIntegration, collectedTopics)
	go pcc.FeedWorkerPool(consumerChan, args.GlobalArgs.Consumers)
	go pcc.FeedWorkerPool(producerChan, args.GlobalArgs.Producers)

	wg.Wait()

	if err := kafkaIntegration.Publish(); err != nil {
		log.Error("Failed to publish data: %s", err.Error())
		os.Exit(1)
	}
}

// ExitOnErr will exit with a -1 if the error is non-nil
func ExitOnErr(err error) {
	if err != nil {
		os.Exit(1)
	}
}
