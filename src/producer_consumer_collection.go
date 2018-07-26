package main

import (
	"strconv"
	"sync"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
)

// Starts a pool of workers to handle collecting data for wither Consumer or producer entities.
// The channel returned is to be closed by the user (or by feedWorkerPool)
func startWorkerPool(poolSize int, wg *sync.WaitGroup, integration *integration.Integration, collectedTopics []string,
	worker func(<-chan *jmxHost, *sync.WaitGroup, *integration.Integration, []string)) chan *jmxHost {

	jmxHostChan := make(chan *jmxHost)

	for i := 0; i < poolSize; i++ {
		wg.Add(1)
		go worker(jmxHostChan, wg, integration, collectedTopics)
	}

	return jmxHostChan
}

// Feeds the worker pool with jmxHost objects, which contain connection information
// for each producer/consumer to be collected
func feedWorkerPool(jmxHostChan chan<- *jmxHost, jmxHosts []*jmxHost) {
	defer close(jmxHostChan)

	for _, jmxHost := range jmxHosts {
		jmxHostChan <- jmxHost
	}
}

// Collect information for consumers sent down the consumerChan
func consumerWorker(consumerChan <-chan *jmxHost, wg *sync.WaitGroup, i *integration.Integration, collectedTopics []string) {
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
		if kafkaArgs.All() || kafkaArgs.Metrics {
			// Lock since we can only make a single JMX connection at a time.
			jmxLock.Lock()

			// Open a JMX connection to the consumer
			if err := jmxOpenFunc(jmxInfo.Host, strconv.Itoa(jmxInfo.Port), jmxInfo.User, jmxInfo.Password); err != nil {
				logger.Errorf("Unable to make JMX connection for Consumer '%s': %s", consumerEntity.Metadata.Name, err.Error())
				jmxCloseFunc() // Close needs to be called even on a failed open to clear out any set variables
				jmxLock.Unlock()
				continue
			}

			// Create a sample for consumer metrics
			sample := consumerEntity.NewMetricSet("KafkaConsumerSample",
				metric.Attribute{Key: "displayName", Value: jmxInfo.Name},
				metric.Attribute{Key: "entityName", Value: "consumer:" + jmxInfo.Name},
				metric.Attribute{Key: "host", Value: jmxInfo.Host},
			)

			// Collect the consumer metrics and populate the sample with them
			if err := getConsumerMetrics(consumerEntity.Metadata.Name, sample); err != nil {
				logger.Errorf("Error collecting metrics from Consumer '%s': %s", consumerEntity.Metadata.Name, err.Error())
				jmxCloseFunc()
				jmxLock.Unlock()
				continue
			}

			// Collect metrics that are topic-specific per Consumer
			collectTopicSubMetrics(consumerEntity, "consumer", consumerTopicMetricDefs,
				collectedTopics, applyconsumerTopicName)

			// Close connection and release lock so another process can make JMX Connections
			jmxCloseFunc()
			jmxLock.Unlock()
		}
	}
}

// Collect information for producers sent down the producerChan
func producerWorker(producerChan <-chan *jmxHost, wg *sync.WaitGroup, i *integration.Integration, collectedTopics []string) {
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
		if kafkaArgs.All() || kafkaArgs.Metrics {
			// Lock since we can only make a single JMX connection at a time.
			jmxLock.Lock()

			// Open a JMX connection to the producer
			if err := jmxOpenFunc(jmxInfo.Host, strconv.Itoa(jmxInfo.Port), jmxInfo.User, jmxInfo.Password); err != nil {
				logger.Errorf("Unable to make JMX connection for Producer '%s': %s", producerEntity.Metadata.Name, err.Error())
				jmxCloseFunc() // Close needs to be called even on a failed open to clear out any set variables
				jmxLock.Unlock()
				continue
			}

			// Create a metric set for the producer
			sample := producerEntity.NewMetricSet("KafkaProducerSample",
				metric.Attribute{Key: "displayName", Value: jmxInfo.Name},
				metric.Attribute{Key: "entityName", Value: "producer:" + jmxInfo.Name},
				metric.Attribute{Key: "host", Value: jmxInfo.Host},
			)

			// Collect producer metrics and populate the metric set with them
			if err := getProducerMetrics(producerEntity.Metadata.Name, sample); err != nil {
				logger.Errorf("Error collecting metrics from Producer '%s': %s", producerEntity.Metadata.Name, err.Error())
				jmxCloseFunc()
				jmxLock.Unlock()
				continue
			}

			// Collect metrics that are topic specific per Producer
			collectTopicSubMetrics(producerEntity, "producer", producerTopicMetricDefs,
				collectedTopics, applyProducerTopicName)

			// Close connection and release lock so another process can make JMX Connections
			jmxCloseFunc()
			jmxLock.Unlock()
		}
	}
}
