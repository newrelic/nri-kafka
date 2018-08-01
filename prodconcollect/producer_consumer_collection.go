// Package prodconcollect handles collection of Consumer and Producer metric data
package prodconcollect

import (
	"strconv"
	"sync"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/nri-kafka/args"
	"github.com/newrelic/nri-kafka/logger"
	"github.com/newrelic/nri-kafka/metrics"
	"github.com/newrelic/nri-kafka/utils"
)

// StartWorkerPool starts a pool of workers to handle collecting data for wither Consumer or producer entities.
// The channel returned is to be closed by the user (or by feedWorkerPool)
func StartWorkerPool(poolSize int, wg *sync.WaitGroup, integration *integration.Integration, collectedTopics []string,
	worker func(<-chan *args.JMXHost, *sync.WaitGroup, *integration.Integration, []string)) chan *args.JMXHost {

	jmxHostChan := make(chan *args.JMXHost)

	for i := 0; i < poolSize; i++ {
		wg.Add(1)
		go worker(jmxHostChan, wg, integration, collectedTopics)
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
func ConsumerWorker(consumerChan <-chan *args.JMXHost, wg *sync.WaitGroup, i *integration.Integration, collectedTopics []string) {
	defer wg.Done()

	for {
		jmxInfo, ok := <-consumerChan
		if !ok {
			return // Stop worker if consumerChan is closed
		}

		// Create an entity for the consumer
		consumerEntity, err := i.Entity(jmxInfo.Name, "consumer")
		if err != nil {
			logger.Errorf("Unable to create entity for Consumer %s: %s", jmxInfo.Name, err.Error())
			continue
		}

		// Gather Metrics for consumer
		if utils.KafkaArgs.All() || utils.KafkaArgs.Metrics {
			// Lock since we can only make a single JMX connection at a time.
			utils.JMXLock.Lock()

			// Open a JMX connection to the consumer
			if err := utils.JMXOpen(jmxInfo.Host, strconv.Itoa(jmxInfo.Port), jmxInfo.User, jmxInfo.Password); err != nil {
				logger.Errorf("Unable to make JMX connection for Consumer '%s': %s", consumerEntity.Metadata.Name, err.Error())
				utils.JMXClose() // Close needs to be called even on a failed open to clear out any set variables
				utils.JMXLock.Unlock()
				continue
			}

			// Create a sample for consumer metrics
			sample := consumerEntity.NewMetricSet("KafkaConsumerSample",
				metric.Attribute{Key: "displayName", Value: jmxInfo.Name},
				metric.Attribute{Key: "entityName", Value: "consumer:" + jmxInfo.Name},
				metric.Attribute{Key: "host", Value: jmxInfo.Host},
			)

			// Collect the consumer metrics and populate the sample with them
			metrics.GetConsumerMetrics(consumerEntity.Metadata.Name, sample)

			// Collect metrics that are topic-specific per Consumer
			metrics.CollectTopicSubMetrics(consumerEntity, "consumer", metrics.ConsumerTopicMetricDefs,
				collectedTopics, metrics.ApplyconsumerTopicName)

			// Close connection and release lock so another process can make JMX Connections
			utils.JMXClose()
			utils.JMXLock.Unlock()
		}
	}
}

// ProducerWorker collect information for Producers sent down the producerChan
func ProducerWorker(producerChan <-chan *args.JMXHost, wg *sync.WaitGroup, i *integration.Integration, collectedTopics []string) {
	defer wg.Done()
	for {
		jmxInfo, ok := <-producerChan
		if !ok {
			return // Stop the worker if the channel is closed
		}

		// Create the producer entity
		producerEntity, err := i.Entity(jmxInfo.Name, "producer")
		if err != nil {
			logger.Errorf("Unable to create entity for Producer %s: %s", jmxInfo.Name, err.Error())
			continue
		}

		// Gather Metrics for producer
		if utils.KafkaArgs.All() || utils.KafkaArgs.Metrics {
			// Lock since we can only make a single JMX connection at a time.
			utils.JMXLock.Lock()

			// Open a JMX connection to the producer
			if err := utils.JMXOpen(jmxInfo.Host, strconv.Itoa(jmxInfo.Port), jmxInfo.User, jmxInfo.Password); err != nil {
				logger.Errorf("Unable to make JMX connection for Producer '%s': %s", producerEntity.Metadata.Name, err.Error())
				utils.JMXClose() // Close needs to be called even on a failed open to clear out any set variables
				utils.JMXLock.Unlock()
				continue
			}

			// Create a metric set for the producer
			sample := producerEntity.NewMetricSet("KafkaProducerSample",
				metric.Attribute{Key: "displayName", Value: jmxInfo.Name},
				metric.Attribute{Key: "entityName", Value: "producer:" + jmxInfo.Name},
				metric.Attribute{Key: "host", Value: jmxInfo.Host},
			)

			// Collect producer metrics and populate the metric set with them
			metrics.GetProducerMetrics(producerEntity.Metadata.Name, sample)

			// Collect metrics that are topic specific per Producer
			metrics.CollectTopicSubMetrics(producerEntity, "producer", metrics.ProducerTopicMetricDefs,
				collectedTopics, metrics.ApplyProducerTopicName)

			// Close connection and release lock so another process can make JMX Connections
			utils.JMXClose()
			utils.JMXLock.Unlock()
		}
	}
}
