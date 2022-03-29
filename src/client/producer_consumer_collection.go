// Package client handles collection of Consumer and Producer client metric data
package client

import (
	"sync"

	"github.com/newrelic/nri-kafka/src/connection"

	"github.com/newrelic/infra-integrations-sdk/data/attribute"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/metrics"
)

// StartWorkerPool starts a pool of workers to handle collecting data for wither Consumer or producer entities.
// The channel returned is to be closed by the user (or by feedWorkerPool)
func StartWorkerPool(
	poolSize int,
	wg *sync.WaitGroup,
	integration *integration.Integration,
	worker func(<-chan *args.JMXHost, *sync.WaitGroup, *integration.Integration, connection.JMXProvider),
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

// ConsumerWorker collects information for consumers sent down the consumerChan
func ConsumerWorker(consumerChan <-chan *args.JMXHost, wg *sync.WaitGroup, i *integration.Integration, jmxConnProvider connection.JMXProvider) {
	defer wg.Done()

	for {
		jmxInfo, ok := <-consumerChan
		if !ok {
			return // Stop worker if consumerChan is closed
		}

		// Create an entity for the consumer
		clusterIDAttr := integration.NewIDAttribute("clusterName", args.GlobalArgs.ClusterName)
		hostIDAttr := integration.NewIDAttribute("host", jmxInfo.Host)
		consumerEntity, err := i.Entity(jmxInfo.Name, "ka-consumer", clusterIDAttr, hostIDAttr)
		if err != nil {
			log.Error("Unable to create entity for Consumer %s: %s", jmxInfo.Name, err.Error())
			continue
		}

		// Gather Metrics for consumer
		if args.GlobalArgs.All() || args.GlobalArgs.Metrics {
			log.Debug("Collecting metrics for consumer %s", consumerEntity.Metadata.Name)

			jmxConfig := connection.NewConfigBuilder().
				FromArgs().
				WithHostname(jmxInfo.Host).WithPort(jmxInfo.Port).
				WithUsername(jmxInfo.User).WithPassword(jmxInfo.Password).
				Build()

			conn, err := jmxConnProvider.NewConnection(jmxConfig)
			if err != nil {
				log.Error("Unable to make JMX connection for Consumer '%s': %v", consumerEntity.Metadata.Name, err)
				continue
			}

			// Create a sample for consumer metrics
			sample := consumerEntity.NewMetricSet("KafkaConsumerSample",
				attribute.Attribute{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
				attribute.Attribute{Key: "displayName", Value: jmxInfo.Name},
				attribute.Attribute{Key: "entityName", Value: "consumer:" + jmxInfo.Name},
				attribute.Attribute{Key: "host", Value: jmxInfo.Host},
			)

			// Collect the consumer metrics and populate the sample with them
			log.Debug("Collecting metrics for Consumer '%s'", consumerEntity.Metadata.Name)
			metrics.GetConsumerMetrics(consumerEntity.Metadata.Name, sample, conn)

			// Collect metrics that are topic-specific per Consumer
			metrics.CollectTopicSubMetrics(consumerEntity, metrics.ConsumerTopicMetricDefs, metrics.ApplyConsumerTopicName, conn)

			log.Debug("Collecting metrics for consumer %s", consumerEntity.Metadata.Name)

			// Close connection and release lock so another process can make JMX Connections
			conn.Close()
		}
	}
}

// ProducerWorker collect information for Producers sent down the producerChan
func ProducerWorker(producerChan <-chan *args.JMXHost, wg *sync.WaitGroup, i *integration.Integration, jmxConnProvider connection.JMXProvider) {
	defer wg.Done()
	for {
		jmxInfo, ok := <-producerChan
		if !ok {
			return // Stop the worker if the channel is closed
		}

		// Create the producer entity
		clusterIDAttr := integration.NewIDAttribute("clusterName", args.GlobalArgs.ClusterName)
		hostIDAttr := integration.NewIDAttribute("host", jmxInfo.Host)
		producerEntity, err := i.Entity(jmxInfo.Name, "ka-producer", clusterIDAttr, hostIDAttr)
		if err != nil {
			log.Error("Unable to create entity for Producer %s: %s", jmxInfo.Name, err.Error())
			continue
		}

		// Gather Metrics for producer
		if args.GlobalArgs.All() || args.GlobalArgs.Metrics {
			log.Debug("Collecting metrics for producer %s", producerEntity.Metadata.Name)

			// Open a JMX connection to the producer
			jmxConfig := connection.NewConfigBuilder().
				FromArgs().
				WithHostname(jmxInfo.Host).WithPort(jmxInfo.Port).
				WithUsername(jmxInfo.User).WithPassword(jmxInfo.Password).
				Build()

			conn, err := jmxConnProvider.NewConnection(jmxConfig)
			if err != nil {
				log.Error("Unable to make JMX connection for Producer '%s': %v", producerEntity.Metadata.Name, err)
				continue
			}

			// Create a metric set for the producer
			sample := producerEntity.NewMetricSet("KafkaProducerSample",
				attribute.Attribute{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
				attribute.Attribute{Key: "displayName", Value: jmxInfo.Name},
				attribute.Attribute{Key: "entityName", Value: "producer:" + jmxInfo.Name},
				attribute.Attribute{Key: "host", Value: jmxInfo.Host},
			)

			// Collect producer metrics and populate the metric set with them
			log.Debug("Collecting metrics for Producer '%s'", producerEntity.Metadata.Name)
			metrics.GetProducerMetrics(producerEntity.Metadata.Name, sample, conn)

			// Collect metrics that are topic specific per Producer
			metrics.CollectTopicSubMetrics(producerEntity, metrics.ProducerTopicMetricDefs, metrics.ApplyProducerTopicName, conn)

			log.Debug("Done Collecting metrics for producer %s", producerEntity.Metadata.Name)

			// Close connection and release lock so another process can make JMX Connections
			conn.Close()
		}
	}
}
