//go:generate goversioninfo
package main

import (
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"sync"

	"github.com/Shopify/sarama"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/broker"
	"github.com/newrelic/nri-kafka/src/client"
	"github.com/newrelic/nri-kafka/src/connection"
	"github.com/newrelic/nri-kafka/src/consumeroffset"
	"github.com/newrelic/nri-kafka/src/topic"
	"github.com/newrelic/nri-kafka/src/zookeeper"
)

const (
	integrationName    = "com.newrelic.kafka"
	integrationVersion = "2.8.2"
)

func main() {
	var argList args.ArgumentList
	kafkaIntegration, err := integration.New(integrationName, integrationVersion, integration.Args(&argList))
	ExitOnErr(err)

	// Setup logging with verbose
	log.SetupLogging(argList.Verbose)

	// Parsing args must be done after integration creation for defaults to be populated
	args.GlobalArgs, err = args.ParseArgs(argList)
	ExitOnErr(err)

	if !args.GlobalArgs.ConsumerOffset {
		coreCollection(kafkaIntegration)
	} else {
		client, err := getClient(args.GlobalArgs)
		ExitOnErr(err)
		if err := consumeroffset.Collect(client, kafkaIntegration); err != nil {
			log.Error("Failed collecting consumer offset data: %s", err.Error())
			os.Exit(1)
		}
	}

	if err := kafkaIntegration.Publish(); err != nil {
		log.Error("Failed to publish data: %s", err.Error())
		os.Exit(1)
	}
}

func getClient(args *args.ParsedArguments) (sarama.Client, error) {
	switch args.AutodiscoverStrategy {
	case "manual":
		for _, broker := range args.Brokers {
			newBroker, err := connection.NewClient(broker.Host, broker.KafkaPort, broker.KafkaProtocol)
			if err != nil {
				log.Error("Failed creating client to manually configured broker %s: %s", broker.Host, err)
				continue
			}
			return newBroker, nil
		}

		return nil, errors.New("failed to connect to any of the configured brokers")
	case "zookeeper":
		zkConn, err := zookeeper.NewConnection(args)
		if err != nil {
			return nil, fmt.Errorf("failed to create zookeeper connection: %s", err)
		}
		client, err := zookeeper.GetClientFromZookeeper(zkConn, args.PreferredListener)
		if err != nil {
			return nil, fmt.Errorf("failed to get client from zookeeper: %s", err)
		}
		return client, nil
	default:
		return nil, fmt.Errorf("invalid autodiscovery strategy %s", args.AutodiscoverStrategy)
	}
}

func getBrokerList(arguments *args.ParsedArguments) ([]*connection.Broker, error) {
	switch arguments.AutodiscoverStrategy {
	case "manual":
		brokers := make([]*connection.Broker, 0, len(arguments.Brokers))
		for _, broker := range arguments.Brokers {
			newSaramaBroker, err := connection.NewBroker(broker.Host, broker.KafkaPort, broker.KafkaProtocol)
			if err != nil {
				log.Error("Failed creating Kafka client to manually configured broker %s: %s", broker.Host, err)
				continue
			}

			newBroker := &connection.Broker{
				Broker:      newSaramaBroker,
				Host:        broker.Host,
				JMXPort:     broker.JMXPort,
				JMXUser:     broker.JMXUser,
				JMXPassword: broker.JMXPassword,
				ID:          fmt.Sprintf("%d", newSaramaBroker.ID()),
			}
			brokers = append(brokers, newBroker)
		}

		return brokers, nil
	case "zookeeper":
		zkConn, err := zookeeper.NewConnection(arguments)
		if err != nil {
			return nil, fmt.Errorf("failed to create zookeeper connection: %s", err)
		}
		return zookeeper.GetBrokerList(zkConn, arguments.PreferredListener)
	default:
		return nil, fmt.Errorf("invalid autodiscovery strategy %s", arguments.AutodiscoverStrategy)
	}
}

// coreCollection is the main integration collection. Does not handle consumerOffset collection
func coreCollection(kafkaIntegration *integration.Integration) {
	log.Info("Running core collection")
	brokers, err := getBrokerList(args.GlobalArgs)
	if err != nil {
		log.Error("Failed to get list of brokers: %s", err)
		return
	}

	clusterClient, err := getClient(args.GlobalArgs)
	if err != nil {
		log.Error("Failed to get a kafka client: %s", err)
		return
	}

	topics, err := topic.GetTopics(clusterClient)
	if err != nil {
		log.Error("Failed to get a list of topics. Continuing with broker collection: %s", err)
	}

	// Enforce hard limits on topics
	bucketedTopics := filterTopicsByBucket(topics, args.GlobalArgs.TopicBucket)
	collectedTopics := enforceTopicLimit(bucketedTopics)

	// Start and feed all worker pools
	var wg sync.WaitGroup
	brokerChan := broker.StartBrokerPool(3, &wg, kafkaIntegration, collectedTopics)
	consumerChan := client.StartWorkerPool(3, &wg, kafkaIntegration, collectedTopics, client.ConsumerWorker)
	producerChan := client.StartWorkerPool(3, &wg, kafkaIntegration, collectedTopics, client.ProducerWorker)

	if args.GlobalArgs.CollectClusterMetrics {
		topicChan := topic.StartTopicPool(5, &wg, clusterClient)
		go topic.FeedTopicPool(topicChan, kafkaIntegration, collectedTopics)
	}

	go broker.FeedBrokerPool(brokers, brokerChan)
	go client.FeedWorkerPool(consumerChan, args.GlobalArgs.Consumers)
	go client.FeedWorkerPool(producerChan, args.GlobalArgs.Producers)

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
		_, _ = h.Write([]byte(topic))
		hashedTopic := int(h.Sum32())
		if hashedTopic%topicBucket.NumBuckets+1 == topicBucket.BucketNumber {
			filteredTopics = append(filteredTopics, topic)
		}
	}

	return filteredTopics
}
