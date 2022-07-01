//go:generate goversioninfo
package main

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

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
	integrationName      = "com.newrelic.kafka"
	discoverBootstrap    = "bootstrap"
	discoverZookeeper    = "zookeeper"
	topicSourceZookeeper = "zookeeper"

	numberOfConsumerWorkers = 3
	numberOfProducerWorkers = 3
)

var (
	integrationVersion = "0.0.0"
	gitCommit          = ""
	buildDate          = ""
)

func main() {
	var argList args.ArgumentList
	kafkaIntegration, err := integration.New(integrationName, integrationVersion, integration.Args(&argList))
	ExitOnErr(err)

	if argList.ShowVersion {
		fmt.Printf(
			"New Relic %s integration Version: %s, Platform: %s, GoVersion: %s, GitCommit: %s, BuildDate: %s\n",
			strings.Title(strings.Replace(integrationName, "com.newrelic.", "", 1)),
			integrationVersion,
			fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			runtime.Version(),
			gitCommit,
			buildDate)
		os.Exit(0)
	}

	// Setup logging with verbose
	log.SetupLogging(argList.Verbose)

	// Parsing args must be done after integration creation for defaults to be populated
	args.GlobalArgs, err = args.ParseArgs(argList)
	ExitOnErr(err)

	sarama.Logger = saramaLogger{}

	jmxConnProvider := connection.NewJMXProviderWithLimit(context.Background(), args.GlobalArgs.MaxJMXConnections)

	if !args.GlobalArgs.ConsumerOffset {
		coreCollection(kafkaIntegration, jmxConnProvider)
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
	case discoverBootstrap:
		bootstrapBroker, err := connection.NewBroker(arguments.BootstrapBroker)
		if err != nil {
			return nil, fmt.Errorf("failed to create boostrap broker: %s", err)
		}

		metadata, err := bootstrapBroker.GetMetadata(&sarama.MetadataRequest{})
		if err != nil {
			return nil, fmt.Errorf("failed to get metadata from broker: %s", err)
		}

		brokers := make([]*connection.Broker, 0, len(metadata.Brokers))
		log.Debug("Found %d brokers in the metadata", len(metadata.Brokers))

		for _, broker := range metadata.Brokers {
			log.Debug("Connecting to broker %d got from bootstrapBroker metadata", broker.ID())
			if arguments.LocalOnlyCollection {
				// TODO figure out a way to get ID on this broker without trying to match addresses
				// This is really hacky, but there doesn't appear to be any way to get the ID off of
				// a broker that we create with NewBroker.
				if broker.Addr() == bootstrapBroker.Addr() {
					log.Debug("BootstrapBroker's ID detected: %d", broker.ID())
					bootstrapBroker.ID = fmt.Sprintf("%d", broker.ID())
					return []*connection.Broker{bootstrapBroker}, nil
				}
				log.Debug("Broker address %s not matching BootstrapBroker's address %s", broker.Addr(), bootstrapBroker.Addr())
				log.Debug("Skipping for local-only collection")
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
				log.Debug("Broker %d with address %s connected", broker.ID(), broker.Addr())
			}
		}

		if len(brokers) == 0 {
			return nil, errors.New("failed to match any brokers with configured address")
		}

		return brokers, nil
	case discoverZookeeper:
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
func coreCollection(kafkaIntegration *integration.Integration, jmxConnProvider connection.JMXProvider) {
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

		errs := checkJMXConnection(brokers, time.Duration(args.GlobalArgs.Timeout)*time.Millisecond)
		if len(errs) > 0 {
			for _, e := range errs {
				log.Error("%v", e)
			}
			log.Error(
				"Errors were detected while probing JMX port of one or more kafka brokers. " +
					"Please ensure that JMX is enabled in all of them, and that the port is open. " +
					"https://docs.newrelic.com/docs/integrations/host-integrations/host-integrations-list/kafka-monitoring-integration",
			)
			os.Exit(2)
		}

		var topics []string
		if args.GlobalArgs.TopicSource == topicSourceZookeeper && args.GlobalArgs.AutodiscoverStrategy == discoverZookeeper {
			var zkConn zookeeper.ZkConnection
			zkConn, err = zookeeper.NewConnection(args.GlobalArgs)
			if err != nil {
				log.Error("failed to create zookeeper connection. Continuing with broker collection: %s", err)
			}
			topics, err = topic.GetTopics(zkConn)
			if err != nil {
				log.Error("Failed to get a list of topics. Continuing with broker collection: %s", err)
			}

			defer zkConn.Close()
		} else {
			topics, err = topic.GetTopics(clusterClient)
			if err != nil {
				log.Error("Failed to get a list of topics. Continuing with broker collection: %s", err)
			}
		}

		// Enforce hard limits on topics
		bucketedTopics := filterTopicsByBucket(topics, args.GlobalArgs.TopicBucket)
		collectedTopics := enforceTopicLimit(bucketedTopics)

		// Start and feed all worker pools
		brokerChan := broker.StartBrokerPool(3, &wg, kafkaIntegration, collectedTopics, jmxConnProvider)

		if !args.GlobalArgs.LocalOnlyCollection {
			topicChan := topic.StartTopicPool(5, &wg, clusterClient)
			go topic.FeedTopicPool(topicChan, kafkaIntegration, collectedTopics)
		}

		go broker.FeedBrokerPool(brokers, brokerChan)
	}

	consumerChan := client.StartWorkerPool(numberOfConsumerWorkers, &wg, kafkaIntegration, client.Worker(client.CollectConsumerMetrics), jmxConnProvider)
	producerChan := client.StartWorkerPool(numberOfProducerWorkers, &wg, kafkaIntegration, client.Worker(client.CollectProducerMetrics), jmxConnProvider)

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

// checkJMXConnection performs a basic connection check to all the supplied brokers, returning any error
func checkJMXConnection(brokers []*connection.Broker, timeout time.Duration) []error {
	var errs []error
	for _, brk := range brokers {
		addr := net.JoinHostPort(brk.Host, fmt.Sprint(brk.JMXPort))
		log.Debug("Testing reachability of JMX port for broker %s", addr)

		conn, err := net.DialTimeout("tcp", addr, timeout)
		if err != nil {
			errs = append(errs, fmt.Errorf("error connecting to JMX port on %s: %v", addr, err))
			continue
		}
		_ = conn.Close()
	}

	return errs
}
