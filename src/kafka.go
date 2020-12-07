//go:generate goversioninfo
package main

import (
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"strings"
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
	integrationVersion = "2.13.8"
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

	sarama.Logger = saramaLogger{}

	if !args.GlobalArgs.ConsumerOffset {
		coreCollection(kafkaIntegration)
	} else {
		brokers, err := getBrokerList(args.GlobalArgs)
		ExitOnErr(err)
		client, err := connection.NewSaramaClientFromBrokerList(brokers)
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

func getBrokerList(arguments *args.ParsedArguments) ([]*connection.Broker, error) {
	switch arguments.AutodiscoverStrategy {
	case "bootstrap":
		bootstrapBroker, err := connection.NewBroker(arguments.BootstrapBroker)
		if err != nil {
			return nil, fmt.Errorf("failed to create boostrap broker: %s", err)
		}

		metadata, err := bootstrapBroker.GetMetadata(&sarama.MetadataRequest{})
		if err != nil {
			return nil, fmt.Errorf("failed to get metadata from broker: %s", err)
		}

		brokers := make([]*connection.Broker, 0, len(metadata.Brokers))
		for _, broker := range metadata.Brokers {
			if arguments.LocalOnlyCollection {
				// TODO figure out a way to get ID on this broker without trying to match addresses
				// This is really hacky, but there doesn't appear to be any way to get the ID off of
				// a broker that we create with NewBroker.
				if broker.Addr() == bootstrapBroker.Addr() {
					bootstrapBroker.ID = fmt.Sprintf("%d", broker.ID())
					return []*connection.Broker{bootstrapBroker}, nil
				}
			} else {
				err := broker.Open(bootstrapBroker.Config)
				if err != nil {
					return nil, fmt.Errorf("failed opening connection: %s", err)
				}
				connected, err := broker.Connected()
				if err != nil {
					return nil, fmt.Errorf("failed checking if connection opened successfully: %s", err)
				}
				if !connected {
					return nil, errors.New("broker is not connected")
				}

				hostPort := strings.Split(broker.Addr(), ":")
				if len(hostPort) != 2 {
					return nil, fmt.Errorf("failed to get host from broker address: %s", broker.Addr())
				}

				newBroker := &connection.Broker{
					SaramaBroker: broker,
					Host:         hostPort[0],
					JMXPort:      arguments.BootstrapBroker.JMXPort,
					JMXUser:      arguments.BootstrapBroker.JMXUser,
					JMXPassword:  arguments.BootstrapBroker.JMXPassword,
					ID:           fmt.Sprintf("%d", broker.ID()),
					Config:       bootstrapBroker.Config,
				}
				brokers = append(brokers, newBroker)

			}
		}

		if len(brokers) == 0 {
			return nil, errors.New("failed to match any brokers with configured address")
		}

		return brokers, nil
	case "zookeeper":
		zkConn, err := zookeeper.NewConnection(arguments)
		if err != nil {
			return nil, fmt.Errorf("failed to create zookeeper connection: %s", err)
		}
		defer func(conn zookeeper.Connection) {
			conn.Close()
		}(zkConn)

		return connection.GetBrokerListFromZookeeper(zkConn, arguments.PreferredListener)
	default:
		return nil, fmt.Errorf("invalid autodiscovery strategy %s", arguments.AutodiscoverStrategy)
	}
}

// coreCollection is the main integration collection. Does not handle consumerOffset collection
func coreCollection(kafkaIntegration *integration.Integration) {
	var wg sync.WaitGroup
	if args.GlobalArgs.CollectBrokers() {
		log.Info("Running core collection")
		brokers, err := getBrokerList(args.GlobalArgs)
		if err != nil {
			log.Error("Failed to get list of brokers: %s", err)
			return
		}

		clusterClient, err := connection.NewSaramaClientFromBrokerList(brokers)
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
		brokerChan := broker.StartBrokerPool(3, &wg, kafkaIntegration, collectedTopics)

		if !args.GlobalArgs.LocalOnlyCollection {
			topicChan := topic.StartTopicPool(5, &wg, clusterClient)
			go topic.FeedTopicPool(topicChan, kafkaIntegration, collectedTopics)
		}

		go broker.FeedBrokerPool(brokers, brokerChan)
	}

	consumerChan := client.StartWorkerPool(3, &wg, kafkaIntegration, client.ConsumerWorker)
	producerChan := client.StartWorkerPool(3, &wg, kafkaIntegration, client.ProducerWorker)

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
const maxTopics = 10000

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
