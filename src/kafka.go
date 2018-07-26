package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/jmx"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/samuel/go-zookeeper/zk"
)

const (
	integrationName    = "com.newrelic.kafka"
	integrationVersion = "0.1.0"
)

var (
	logger    log.Logger
	kafkaArgs *kafkaArguments

	jmxLock sync.Mutex

	// Saved functions to allow easier mocking of jmx functions in tests
	queryFunc    = jmx.Query
	jmxOpenFunc  = jmx.Open
	jmxCloseFunc = jmx.Close
)

func main() {
	var args argumentList
	// Create Integration
	kafkaIntegration, err := integration.New(integrationName, integrationVersion, integration.Args(&args))
	panicOnErr(err)

	// Needs to be after integration creation for args to be set
	logger = kafkaIntegration.Logger()

	// Parse args into structs
	// This has to be after integration creation for defaults to be populated
	kafkaArgs, err = parseArgs(args)
	panicOnErr(err)

	zkConn, err := makeZookeeperConnection()
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

// Waiting on issue https://github.com/samuel/go-zookeeper/issues/108 so we can change this function
// and allow us to mock out the zk.Connect function
func makeZookeeperConnection() (zookeeperConn, error) {

	// Create array of host:port strings for connecting
	zkHosts := make([]string, 0, len(kafkaArgs.ZookeeperHosts))
	for _, zkHost := range kafkaArgs.ZookeeperHosts {
		zkHosts = append(zkHosts, fmt.Sprintf("%s:%d", zkHost.Host, zkHost.Port))
	}

	// Create connection and add authentication if provided
	zkConn, _, err := zk.Connect(zkHosts, time.Second)
	if kafkaArgs.ZookeeperAuthScheme != "" {
		if err = zkConn.AddAuth(kafkaArgs.ZookeeperAuthScheme, []byte(kafkaArgs.ZookeeperAuthSecret)); err != nil {
			return nil, err
		}
	}

	return zkConn, nil
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}
