// Package consumeroffset handles collection of consumer offsets for consumer groups
package consumeroffset

import (
	"errors"
	"fmt"
	"sync"

	"github.com/IBM/sarama"

	"github.com/newrelic/infra-integrations-sdk/data/attribute"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/connection"
)

const (
	nrConsumerGroupEntity      = "ka-consumer-group"
	nrConsumerGroupTopicEntity = "ka-consumer-group-topic"
	nrConsumerEntity           = "ka-consumer"
	nrPartitionConsumerEntity  = "ka-partition-consumer"
)

var ErrNoConsumerGroupRegex = errors.New("if consumer_offset is set, consumer_group_regex must also be set")

type partitionOffsets struct {
	Topic          string `metric_name:"topic" source_type:"attribute"`
	Partition      string `metric_name:"partition" source_type:"attribute"`
	ConsumerOffset *int64 `metric_name:"kafka.consumerOffset" source_type:"gauge"`
	HighWaterMark  *int64 `metric_name:"kafka.highWaterMark" source_type:"gauge"`
	ConsumerLag    *int64 `metric_name:"kafka.consumerLag" source_type:"gauge"`
}

// TopicPartitions is the substructure within the consumer group structure
type TopicPartitions map[string][]int32

// Collect collects offset data per consumer group specified in the arguments
func Collect(client connection.Client, kafkaIntegration *integration.Integration) error {
	clusterAdmin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		return fmt.Errorf("failed to create cluster admin: %w", err)
	}

	// Use the more modern collection method if the configuration exists
	if args.GlobalArgs.ConsumerGroupRegex == nil {
		return ErrNoConsumerGroupRegex
	}

	consumerGroupMap, err := clusterAdmin.ListConsumerGroups()
	if err != nil {
		return fmt.Errorf("failed to get list of consumer groups: %w", err)
	}
	consumerGroupList := make([]string, 0, len(consumerGroupMap))
	for consumerGroup := range consumerGroupMap {
		consumerGroupList = append(consumerGroupList, consumerGroup)
	}
	log.Debug("Retrieved the list of consumer groups: %v", consumerGroupList)

	consumerGroups, err := clusterAdmin.DescribeConsumerGroups(consumerGroupList)
	if err != nil {
		return fmt.Errorf("failed to get consumer group descriptions: %w", err)
	}
	log.Debug("Retrieved the descriptions of all consumer groups")

	topicOffsetGetter := NewSaramaTopicOffsetGetter(client)
	cAdminConsumerGroupTopicLister := NewCAdminConsumerGroupTopicLister(clusterAdmin)

	var unmatchedConsumerGroups []string
	var wg sync.WaitGroup
	for _, consumerGroup := range consumerGroups {
		if args.GlobalArgs.ConsumerGroupRegex.MatchString(consumerGroup.GroupId) {
			wg.Add(1)
			go func(consumerGroup *sarama.GroupDescription) {
				collectOffsetsForConsumerGroup(
					cAdminConsumerGroupTopicLister,
					consumerGroup.GroupId,
					consumerGroup.Members,
					kafkaIntegration,
					topicOffsetGetter,
				)
				wg.Done()
			}(consumerGroup)
		} else {
			unmatchedConsumerGroups = append(unmatchedConsumerGroups, consumerGroup.GroupId)
		}
	}

	if len(unmatchedConsumerGroups) > 0 {
		log.Debug("Skipped collecting consumer offsets for unmatched consumer groups %v", unmatchedConsumerGroups)
	}

	wg.Wait()

	return nil
}

// setMetrics adds the metrics from an array of partitionOffsets to the integration
func setMetrics(consumerGroup string, offsetData []*partitionOffsets, kafkaIntegration *integration.Integration) error {
	clusterIDAttr := integration.NewIDAttribute("clusterName", args.GlobalArgs.ClusterName)
	groupEntity, err := kafkaIntegration.Entity(consumerGroup, "ka-consumerGroup", clusterIDAttr)
	if err != nil {
		return err
	}

	for _, offsetData := range offsetData {
		metricSet := groupEntity.NewMetricSet("KafkaOffsetSample",
			attribute.Attribute{Key: "displayName", Value: groupEntity.Metadata.Name},
			attribute.Attribute{Key: "entityName", Value: "consumerGroup:" + groupEntity.Metadata.Name},
			attribute.Attribute{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
		)

		if err := metricSet.MarshalMetrics(offsetData); err != nil {
			log.Error("Error Marshaling offset metrics for consumer group '%s': %s", consumerGroup, err.Error())
			continue
		}
	}

	return nil
}
