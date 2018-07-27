// This file contains most code related to gather topic size adn it involves communication between brokerWorkers and topicWorkers
package main

import (
	"fmt"
	"strconv"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/nri-kafka/logger"
	"github.com/newrelic/nri-kafka/metrics"
	"github.com/newrelic/nri-kafka/utils"
)

func gatherTopicSizes(b *broker, collectedTopics []string) {
	utils.JMXLock.Lock()
	if err := utils.JMXOpen(b.Host, strconv.Itoa(b.JMXPort), utils.KafkaArgs.DefaultJMXUser, utils.KafkaArgs.DefaultJMXPassword); err != nil {
		logger.Errorf("Broker '%s' failed to open JMX connection for Topic Size collection: %s", b.Host, err.Error())
		utils.JMXClose()
		utils.JMXLock.Unlock()
		return
	}

	for _, topicName := range collectedTopics {
		beanName := metrics.ApplyTopicName(metrics.TopicSizeMetricDef.MBean, topicName)
		results, err := utils.JMXQuery(beanName, utils.KafkaArgs.Timeout)
		if err != nil {
			logger.Errorf("Broker '%s' failed to make JMX Query: %s", b.Host, err.Error())
			// Close channel to signal early exit for waiting topic worker
			continue
		} else if len(results) == 0 {
			continue
		}

		topicSize, err := aggregateTopicSize(results)
		if err != nil {
			logger.Errorf("Unable to calculate size for Topic %s: %s", topicName, err.Error())
			continue
		}

		sample := b.Entity.NewMetricSet("KafkaBrokerSample",
			metric.Attribute{Key: "displayName", Value: b.Entity.Metadata.Name},
			metric.Attribute{Key: "entityName", Value: "broker:" + b.Entity.Metadata.Name},
			metric.Attribute{Key: "topic", Value: topicName},
		)

		if err := sample.SetMetric("topic.diskSize", topicSize, metric.GAUGE); err != nil {
			logger.Errorf("Unable to collect topic size for Topic %s on Broker %s: %s", topicName, b.Entity.Metadata.Name, err.Error())
		}
	}

	utils.JMXClose()
	utils.JMXLock.Unlock()
	return
}

func aggregateTopicSize(jmxResult map[string]interface{}) (size float64, err error) {
	for key, value := range jmxResult {
		partitionSize, ok := value.(float64)
		if !ok {
			size = float64(-1)
			err = fmt.Errorf("unable to cast bean '%s' value '%v' as float64", key, value)
			return
		}

		size += partitionSize
	}

	return
}
