// Package client handles collection of Consumer and Producer client metric data
package client

import (
	"sync"

	"github.com/newrelic/nri-kafka/src/connection"

	"github.com/newrelic/infra-integrations-sdk/v3/data/attribute"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/metrics"
)

type CollectionWorker func(<-chan *args.JMXHost, *sync.WaitGroup, *integration.Integration, connection.JMXProvider)

type CollectionFn func(*integration.Integration, *args.JMXHost, connection.JMXProvider)

// StartWorkerPool starts a pool of workers to handle collecting data for wither Consumer or producer entities.
// The channel returned is to be closed by the user (or by feedWorkerPool)
func StartWorkerPool(
	poolSize int,
	wg *sync.WaitGroup,
	integration *integration.Integration,
	worker CollectionWorker,
	jmxConnProvider connection.JMXProvider,
) chan *args.JMXHost {

	jmxHostChan := make(chan *args.JMXHost)

	for i := 0; i < poolSize; i++ {
		wg.Add(1)
		go worker(jmxHostChan, wg, integration, jmxConnProvider)
	}

	return jmxHostChan
}

// FeedWorkerPool feeds the worker pool with jmxHost objects, which contain connection information
// for each producer/consumer to be collected
func FeedWorkerPool(jmxHostChan chan<- *args.JMXHost, jmxHosts []*args.JMXHost) {
	defer close(jmxHostChan)

	for _, jmxHost := range jmxHosts {
		jmxHostChan <- jmxHost
	}
}

// Worker collects information as set in `collectionFn` for consumers/producers sent down to the chan.
func Worker(collectionFn CollectionFn) CollectionWorker {
	return func(c <-chan *args.JMXHost, wg *sync.WaitGroup, i *integration.Integration, jmxConnProvider connection.JMXProvider) {
		defer wg.Done()
		for {
			jmxInfo, ok := <-c
			if !ok {
				return // Stop worker if chan is closed
			}
			collectionFn(i, jmxInfo, jmxConnProvider)
		}
	}
}

func CollectConsumerMetrics(i *integration.Integration, jmxInfo *args.JMXHost, jmxConnProvider connection.JMXProvider) {
	jmxConfig := connection.NewConfigBuilder().FromArgs().WithJMXHostSettings(jmxInfo).Build()
	conn, err := jmxConnProvider.NewConnection(jmxConfig)
	if err != nil {
		log.Error("Unable to make JMX connection for '%s:%s': %v", jmxInfo.Host, jmxInfo.Port, err)
		return
	}
	defer conn.Close()
	// Get client identifiers for all the consumers
	clientIDs, err := detectConsumerIDs(jmxInfo, conn)
	if err != nil {
		log.Error("Unable to detect consumer/producers for '%s:%s': %s", jmxInfo.Host, jmxInfo.Port, err)
		return
	}
	for _, clientID := range clientIDs {
		// Create an entity for the consumer
		clusterIDAttr := integration.NewIDAttribute("clusterName", args.GlobalArgs.ClusterName)
		hostIDAttr := integration.NewIDAttribute("host", jmxInfo.Host)
		consumerEntity, err := i.Entity(clientID, "ka-consumer", clusterIDAttr, hostIDAttr)
		if err != nil {
			log.Error("Unable to create entity for Consumer %s: %s", clientID, err.Error())
			continue
		}

		if !(args.GlobalArgs.All() || args.GlobalArgs.Metrics) {
			continue
		}
		// Gather Metrics for consumer
		log.Debug("Collecting metrics for consumer %s", consumerEntity.Metadata.Name)
		// Create a sample for consumer metrics
		sample := consumerEntity.NewMetricSet("KafkaConsumerSample",
			attribute.Attribute{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
			attribute.Attribute{Key: "displayName", Value: clientID},
			attribute.Attribute{Key: "entityName", Value: "consumer:" + clientID},
			attribute.Attribute{Key: "host", Value: jmxInfo.Host},
		)
		// Collect the consumer metrics and populate the sample with them
		log.Debug("Collecting metrics for Consumer '%s'", consumerEntity.Metadata.Name)
		metrics.GetConsumerMetrics(consumerEntity.Metadata.Name, sample, conn)
		// Collect metrics that are topic-specific per Consumer
		metrics.CollectTopicSubMetrics(consumerEntity, metrics.ConsumerTopicMetricDefs, metrics.ApplyConsumerTopicName, conn)
		log.Debug("Done collecting metrics for consumer %s", consumerEntity.Metadata.Name)
	}
}

func CollectProducerMetrics(i *integration.Integration, jmxInfo *args.JMXHost, jmxConnProvider connection.JMXProvider) {
	jmxConfig := connection.NewConfigBuilder().FromArgs().WithJMXHostSettings(jmxInfo).Build()
	conn, err := jmxConnProvider.NewConnection(jmxConfig)
	if err != nil {
		log.Error("Unable to make JMX connection for '%s:%s': %v", jmxInfo.Host, jmxInfo.Port, err)
		return
	}
	defer conn.Close()
	// Get client identifiers for all the producers
	clientIDs, err := detectProducerIDs(jmxInfo, conn)
	if err != nil {
		log.Error("Unable to detect producers for '%s:%s': %s", jmxInfo.Host, jmxInfo.Port, err)
		return
	}
	for _, clientID := range clientIDs {
		// Create the producer entity
		clusterIDAttr := integration.NewIDAttribute("clusterName", args.GlobalArgs.ClusterName)
		hostIDAttr := integration.NewIDAttribute("host", jmxInfo.Host)
		producerEntity, err := i.Entity(clientID, "ka-producer", clusterIDAttr, hostIDAttr)
		if err != nil {
			log.Error("Unable to create entity for Producer %s: %s", clientID, err.Error())
			continue
		}
		if !(args.GlobalArgs.All() || args.GlobalArgs.Metrics) {
			continue
		}
		sample := producerEntity.NewMetricSet("KafkaProducerSample",
			attribute.Attribute{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
			attribute.Attribute{Key: "displayName", Value: clientID},
			attribute.Attribute{Key: "entityName", Value: "producer:" + clientID},
			attribute.Attribute{Key: "host", Value: jmxInfo.Host},
		)
		// Collect producer metrics and populate the metric set with them
		log.Debug("Collecting metrics for Producer '%s'", producerEntity.Metadata.Name)
		metrics.GetProducerMetrics(producerEntity.Metadata.Name, sample, conn)
		// Collect metrics that are topic specific per Producer
		metrics.CollectTopicSubMetrics(producerEntity, metrics.ProducerTopicMetricDefs, metrics.ApplyProducerTopicName, conn)
		log.Debug("Done Collecting metrics for producer %s", producerEntity.Metadata.Name)
	}
}
