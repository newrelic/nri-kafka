// Package conoffsetcollect handles collection of consumer offsets for consumer groups
package conoffsetcollect

import (
	"fmt"
	"strconv"

	"github.com/Shopify/sarama"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-kafka/src/args"
	bc "github.com/newrelic/nri-kafka/src/brokercollect"
	"github.com/newrelic/nri-kafka/src/zookeeper"
)

type consumerOffset struct {
	Topic          string `metric_name:"topic" source_type:"attribute"`
	Partition      string `metric_name:"partition" source_type:"attribute"`
	ConsumerOffset int64  `metric_name:"kafka.consumerOffset" source_type:"gauge"`
}

// Collect collects offset data per consumer group specified in the arguments
func Collect(zkConn zookeeper.Connection, kafakIntegration *integration.Integration) error {
	client, err := createClient(zkConn)
	if err != nil {
		return err
	}

	defer func() {
		if err := client.Close(); err != nil {
			log.Debug("Error closing client connection: %s", err.Error())
		}
	}()

	// cache this so we don't have to look it up everytime
	allBrokers := client.Brokers()

	for consumerGroup, topicPartitions := range args.GlobalArgs.ConsumerGroups {
		// TODO use offsets
		processConsumerGroup(client, consumerGroup, topicPartitions, allBrokers)
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
			return nil, err
		}

		brokers = append(brokers, fmt.Sprintf("%s:%d", host, port))
	}

	return sarama.NewClient(brokers, sarama.NewConfig())
}

func processConsumerGroup(client sarama.Client, groupName string, topicPartitions args.TopicPartitions, allBrokers []*sarama.Broker) []*consumerOffset {
	// refresh coordinator cache (suggested by sarama to do so)
	if err := client.RefreshCoordinator(groupName); err != nil {
		log.Debug("Unable to refresh coordinator for group '%s'", groupName)
	}

	brokers := make([]*sarama.Broker, 0)

	// get coordinator broker if possible, if not look through all brokers
	coordinator, err := client.Coordinator(groupName)
	if err != nil {
		log.Debug("Unable to retrieve coordinator for group '%s'", groupName)
		brokers = allBrokers
	} else {
		brokers = append(brokers, coordinator)
	}

	return getConsumerOffsets(groupName, topicPartitions, brokers)
}

func getConsumerOffsets(groupName string, topicPartitions args.TopicPartitions, brokers []*sarama.Broker) []*consumerOffset {
	request := &sarama.OffsetFetchRequest{
		ConsumerGroup: groupName,
		Version:       int16(1),
	}

	consumerOffsets := make([]*consumerOffset, 0)
	for _, broker := range brokers {
		resp, err := broker.FetchOffset(request)
		if err != nil {
			log.Debug("Error fetching offset requests for group '%s' from broker with id '%d': %s", groupName, broker.ID(), err.Error())
			continue
		}

		for topic, partitions := range topicPartitions {
			for _, partition := range partitions {
				if block := resp.GetBlock(topic, partition); block != nil && block.Err == sarama.ErrNoError {
					offsetData := &consumerOffset{
						Topic:          topic,
						Partition:      strconv.Itoa(int(partition)),
						ConsumerOffset: block.Offset,
					}

					consumerOffsets = append(consumerOffsets, offsetData)
				}
			}
		}
	}

	return consumerOffsets
}
