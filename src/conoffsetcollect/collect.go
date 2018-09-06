// Package conoffsetcollect handles collection of consumer offsets for consumer groups
package conoffsetcollect

import (
	"fmt"
	"strconv"

	"github.com/Shopify/sarama"
	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-kafka/src/args"
	bc "github.com/newrelic/nri-kafka/src/brokercollect"
	"github.com/newrelic/nri-kafka/src/zookeeper"
)

type partitionOffsets struct {
	Topic          string `metric_name:"topic" source_type:"attribute"`
	Partition      string `metric_name:"partition" source_type:"attribute"`
	ConsumerOffset int64  `metric_name:"kafka.consumerOffset" source_type:"gauge"`
}

// Collect collects offset data per consumer group specified in the arguments
func Collect(zkConn zookeeper.Connection, kafkaIntegration *integration.Integration) error {
	client, err := createClient(zkConn)
	if err != nil {
		return err
	}

	defer func() {
		if err := client.Close(); err != nil {
			log.Debug("Error closing client connection: %s", err.Error())
		}
	}()

	// this step may only be needed if collecting from kafka rather than Zookeeper
	fillKafkaCaches(client)

	for consumerGroup, topicPartitions := range args.GlobalArgs.ConsumerGroups {
		offsetData := getKafkaConsumerOffsets(client, consumerGroup, topicPartitions)

		if err := setMetrics(consumerGroup, offsetData, kafkaIntegration); err != nil {
			log.Error("Error setting metrics for consumer group '%s': %s", consumerGroup, err.Error())
		}
	}

	return nil
}

func createClient(zkConn zookeeper.Connection) (sarama.Client, error) {
	brokerIDs, err := bc.GetBrokerIDs(zkConn)
	if err != nil {
		return nil, err
	}

	brokers := make([]string, 0, len(brokerIDs))
	for _, brokerID := range brokerIDs {
		// convert to int id
		intID, err := strconv.Atoi(brokerID)
		if err != nil {
			log.Warn("Unable to parse integer broker ID from %s", brokerID)
			continue
		}

		// get broker connection info
		host, _, port, err := bc.GetBrokerConnectionInfo(intID, zkConn)
		if err != nil {
			log.Warn("Unable to get connection information for broker with ID '%d'. Will not collect offest data for consumer groups on this broker.", intID)
			continue
		}

		brokers = append(brokers, fmt.Sprintf("%s:%d", host, port))
	}

	return sarama.NewClient(brokers, sarama.NewConfig())
}

func setMetrics(consumerGroup string, offsetData []*partitionOffsets, kafkaIntegration *integration.Integration) error {
	groupEntity, err := kafkaIntegration.Entity(consumerGroup, "consumerGroup")
	if err != nil {
		return err
	}

	for _, offsetData := range offsetData {
		metricSet := groupEntity.NewMetricSet("ConsumerGroupOffsetSample",
			metric.Attribute{Key: "displayName", Value: groupEntity.Metadata.Name},
			metric.Attribute{Key: "entityName", Value: "consumerGroup:" + groupEntity.Metadata.Name})

		if err := metricSet.MarshalMetrics(offsetData); err != nil {
			log.Error("Error Marshaling offset metrics for consumer group '%s': %s", consumerGroup, err.Error())
			continue
		}
	}

	return nil
}
