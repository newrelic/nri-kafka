//go:generate goversioninfo
package main

import (
	"hash/fnv"
	"os"
	"strings"
	"sync"

	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-kafka/src/args"
	bc "github.com/newrelic/nri-kafka/src/brokercollect"
	offc "github.com/newrelic/nri-kafka/src/conoffsetcollect"
	pcc "github.com/newrelic/nri-kafka/src/prodconcollect"
	tc "github.com/newrelic/nri-kafka/src/topiccollect"
	"github.com/newrelic/nri-kafka/src/zookeeper"
)

const (
	integrationName    = "com.newrelic.kafka"
	integrationVersion = "2.8.2"
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

	if !args.GlobalArgs.ConsumerOffset {
		coreCollection(zkConn, kafkaIntegration)
	} else {
		if err := offc.Collect(zkConn, kafkaIntegration); err != nil {
			log.Error("Failed collecting consumer offset data: %s", err.Error())
			os.Exit(1)
		}
	}

	if err := kafkaIntegration.Publish(); err != nil {
		log.Error("Failed to publish data: %s", err.Error())
		os.Exit(1)
	}
}

// coreCollection is the main integration collection. Does not handle consumerOffset collection
func coreCollection(zkConn zookeeper.Connection, kafkaIntegration *integration.Integration) {
	// Get topic list
	collectedTopics, err := tc.GetTopics(zkConn)
	ExitOnErr(err)
	log.Debug("Collecting metrics for the following topics: %s", strings.Join(collectedTopics, ","))

	// Enforce hard limits on Topics
	collectedTopics = filterTopicsByBucket(collectedTopics, args.GlobalArgs.TopicBucket)
	collectedTopics = enforceTopicLimit(collectedTopics)

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
	go func() {
		if err := bc.FeedBrokerPool(zkConn, brokerChan); err != nil {
			log.Error("Unable to collect Brokers: %s", err.Error())
		}
	}()

	go tc.FeedTopicPool(topicChan, kafkaIntegration, collectedTopics)
	go pcc.FeedWorkerPool(consumerChan, args.GlobalArgs.Consumers)
	go pcc.FeedWorkerPool(producerChan, args.GlobalArgs.Producers)

	wg.Wait()
}

// ExitOnErr will exit with a 1 if the error is non-nil
// All errors should be logged before calling this.
func ExitOnErr(err error) {
	if err != nil {
		log.Error("Integration failed: %s", err)
		os.Exit(1)
	}
}

// maxTopics is the maximum amount of Topics that can be collect.
// If there are more than this number of Topics then collection of
// Topics will fail.
const maxTopics = 300

func enforceTopicLimit(collectedTopics []string) []string {
	if length := len(collectedTopics); length > maxTopics {
		log.Error("There are %d topics in collection, the maximum amount of topics to collect is %d. Use the topic whitelist configuration parameter to limit collection size.", length, maxTopics)
		return make([]string, 0)
	}

	return collectedTopics
}

// filterTopicsByBucket
func filterTopicsByBucket(topicList []string, topicBucket args.TopicBucket) []string {
	filteredTopics := make([]string, 0, len(topicList))
	for _, topic := range topicList {
		h := fnv.New32()
		h.Write([]byte(topic))
		hashedTopic := int(h.Sum32())
		if hashedTopic%topicBucket.NumBuckets+1 == topicBucket.BucketNumber {
			filteredTopics = append(filteredTopics, topic)
		}
	}

	return filteredTopics
}
