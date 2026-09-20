// Package metrics contains definitions for all JMX collected Metrics, and core collection
// methods for Brokers, Consumers, and Producers.
package metrics

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/newrelic/nri-kafka/src/connection"
	"github.com/newrelic/nrjmx/gojmx"

	"github.com/newrelic/infra-integrations-sdk/v3/data/attribute"
	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/jmx"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-kafka/src/args"
)

// GetBrokerMetrics collects all Broker JMX metrics and stores them in sample
func GetBrokerMetrics(sample *metric.Set, conn connection.JMXConnection) {
	// ActiveControllerCount and GlobalPartitionCount are only meaningful read from the
	// controller broker specifically - see ClusterMetricDefs, which collects them correctly
	// via connection.FindControllerBroker instead of every broker's own JMX connection.
	CollectMetricDefinitions(sample, brokerMetricDefs, nil, conn)
	CollectBrokerRequestMetrics(sample, brokerRequestMetricDefs, conn)

	if args.GlobalArgs.EnableBrokerJVMMetrics {
		CollectMetricDefinitions(sample, jvmMetricDefs, nil, conn)
		CollectGarbageCollectorMetrics(sample, conn)
		CollectMemoryPoolMetrics(sample, conn)
	}
}

func GetFinalMetricSets(metricSets []*JMXMetricSet, v2MetricSets []*JMXMetricSet) []*JMXMetricSet {
	log.Info("Initial metric sets: %v , %v", metricSets, v2MetricSets)
	finalMetricSets := make([]*JMXMetricSet, 0, len(metricSets)+len(v2MetricSets))
	finalMetricSets = append(finalMetricSets, metricSets...)

	if args.GlobalArgs.EnableBrokerTopicMetricsV2 {
		finalMetricSets = append(finalMetricSets, v2MetricSets...)
	}

	log.Info("Final metric sets: %v", finalMetricSets)
	return finalMetricSets
}

// GetConsumerMetrics collects all Consumer metrics for the given
// consumerName and stores them in sample.
func GetConsumerMetrics(consumerName string, sample *metric.Set, conn connection.JMXConnection) {
	CollectMetricDefinitions(sample, consumerMetricDefs, applyConsumerName(consumerName), conn)
}

// GetProducerMetrics collects all Producer and Producer metrics for the given
// producerName and stores them in sample.
func GetProducerMetrics(producerName string, sample *metric.Set, conn connection.JMXConnection) {
	CollectMetricDefinitions(sample, producerMetricDefs, applyProducerName(producerName), conn)
}

// CollectTopicSubMetrics collects Topic metrics that are related to either a Producer or Consumer
//
// beanModifier is a function that is used to replace place holder with actual Consumer/Producer
// and Topic names for a given MBean
func CollectTopicSubMetrics(
	entity *integration.Entity,
	metricSets []*JMXMetricSet,
	beanModifier func(string, string) BeanModifier,
	conn connection.JMXConnection,
) {

	// need to title case the type so it matches the metric set of the parent entity
	titleEntityType := strings.Title(strings.TrimPrefix(entity.Metadata.Namespace, "ka-"))

	topicList, err := getTopicListFromJMX(entity.Metadata.Name, titleEntityType, conn)
	if err != nil {
		log.Error("Failed to collect topic list for producer or consumer: %s", err)
		return
	}

	for _, topicName := range topicList {
		topicSample := entity.NewMetricSet("Kafka"+titleEntityType+"Sample",
			attribute.Attribute{Key: "displayName", Value: entity.Metadata.Name},
			attribute.Attribute{Key: "entityName", Value: fmt.Sprintf("%s:%s", strings.TrimPrefix(entity.Metadata.Namespace, "ka-"), entity.Metadata.Name)},
			attribute.Attribute{Key: "topic", Value: topicName},
		)

		CollectMetricDefinitions(topicSample, metricSets, beanModifier(entity.Metadata.Name, topicName), conn)
	}
}

// CollectBrokerRequestMetrics collects request metrics from brokers
func CollectBrokerRequestMetrics(sample *metric.Set, metricSets []*JMXMetricSet, conn connection.JMXConnection) {
	notFoundMetrics := make([]string, 0)

	for _, metricSet := range metricSets {
		beanName := metricSet.MBean

		// Return all the results under a specific mBean
		results, err := conn.QueryMBeanAttributes(beanName)
		// If we fail we don't want a total failure as other metrics can be collected even if a single failure/timout occurs
		if err != nil {
			if jmxErr, ok := gojmx.IsJMXError(err); ok {
				log.Error("Unable to execute JMX query for MBean '%s': %v", beanName, jmxErr)
				continue
			}
			log.Error("Connection error for %s:%s : %s", jmx.HostName(), jmx.Port(), err)
			os.Exit(1)
		}

		// For each metric to collect, populate the sample if it is
		// found in results, otherwise save the mBeanKey as not found
		for _, metricDef := range metricSet.MetricDefs {
			versionRollup := 0.0
			found := false
			// Newer versions of Kafka have nest the request metrics under a version, so we have to roll these up
			for _, attr := range results {
				if strings.HasPrefix(attr.Name, metricSet.MetricPrefix+metricDef.JMXAttr) && strings.HasSuffix(attr.Name, "attr=OneMinuteRate") {
					found = true
					rate, ok := attr.GetValue().(float64)
					if !ok {
						log.Warn("Got non-float64 value for a rate")
						continue
					}
					versionRollup += rate
				}
			}

			if !found {
				notFoundMetrics = append(notFoundMetrics, metricDef.Name)
				continue
			}

			if err := sample.SetMetric(metricDef.Name, versionRollup, metricDef.SourceType); err != nil {
				log.Error("Error setting value: %s", err)
			}
		}
	}

	if len(notFoundMetrics) > 0 {
		log.Warn("Can't find raw metrics in results for keys: %v", notFoundMetrics)
	}
}

// CollectGarbageCollectorMetrics sums CollectionCount and CollectionTime across every garbage
// collector MBean present, and buckets the same values into young/old generation via
// classifyGCGeneration. Collector names vary by GC algorithm, so unlike the rest of
// jvmMetricDefs this can't be a fixed MetricDefinition - it aggregates by attribute suffix
// instead, regardless of which collector name reported it.
func CollectGarbageCollectorMetrics(sample *metric.Set, conn connection.JMXConnection) {
	results, err := conn.QueryMBeanAttributes(jvmGCMBean)
	if err != nil {
		if jmxErr, ok := gojmx.IsJMXError(err); ok {
			log.Error("Unable to execute JMX query for MBean '%s': %v", jvmGCMBean, jmxErr)
			return
		}
		log.Error("Connection error for %s:%s : %s", jmx.HostName(), jmx.Port(), err)
		os.Exit(1)
	}

	var collectionCount, collectionTime float64
	var youngCount, youngTime, oldCount, oldTime float64
	for _, attr := range results {
		if attr.ResponseType == gojmx.ResponseTypeErr {
			continue
		}

		value, err := attr.GetValueAsFloat()
		if err != nil {
			continue
		}

		var isCount, isTime bool
		switch {
		case strings.HasSuffix(attr.Name, jvmGCCollectionCountAttr):
			collectionCount += value
			isCount = true
		case strings.HasSuffix(attr.Name, jvmGCCollectionTimeAttr):
			collectionTime += value
			isTime = true
		default:
			continue
		}

		match := jvmMBeanNameRegex.FindStringSubmatch(attr.Name)
		if match == nil {
			continue
		}
		young, old := classifyGCGeneration(match[1])
		switch {
		case young && isCount:
			youngCount += value
		case young && isTime:
			youngTime += value
		case old && isCount:
			oldCount += value
		case old && isTime:
			oldTime += value
		}
	}

	setGCMetric := func(name string, value float64) {
		if err := sample.SetMetric(name, value, metric.RATE); err != nil {
			log.Error("Error setting value: %s", err)
		}
	}
	setGCMetric("jvm.gcCollectionsPerSecond", collectionCount)
	setGCMetric("jvm.gcTimePerSecond", collectionTime)
	setGCMetric("jvm.gcYoungGenCollectionsPerSecond", youngCount)
	setGCMetric("jvm.gcYoungGenTimePerSecond", youngTime)
	setGCMetric("jvm.gcOldGenCollectionsPerSecond", oldCount)
	setGCMetric("jvm.gcOldGenTimePerSecond", oldTime)
}

// CollectMemoryPoolMetrics buckets heap pool usage into eden/survivor/old-gen via
// classifyMemoryPool.
func CollectMemoryPoolMetrics(sample *metric.Set, conn connection.JMXConnection) {
	results, err := conn.QueryMBeanAttributes(jvmMemoryPoolMBean)
	if err != nil {
		if jmxErr, ok := gojmx.IsJMXError(err); ok {
			log.Error("Unable to execute JMX query for MBean '%s': %v", jvmMemoryPoolMBean, jmxErr)
			return
		}
		log.Error("Connection error for %s:%s : %s", jmx.HostName(), jmx.Port(), err)
		os.Exit(1)
	}

	var edenUsed, edenMax, survivorUsed, survivorMax, oldUsed, oldMax float64
	for _, attr := range results {
		if attr.ResponseType == gojmx.ResponseTypeErr {
			continue
		}

		var isUsed, isMax bool
		switch {
		case strings.HasSuffix(attr.Name, jvmMemoryPoolUsedAttr):
			isUsed = true
		case strings.HasSuffix(attr.Name, jvmMemoryPoolMaxAttr):
			isMax = true
		default:
			continue
		}

		value, err := attr.GetValueAsFloat()
		if err != nil {
			continue
		}

		match := jvmMBeanNameRegex.FindStringSubmatch(attr.Name)
		if match == nil {
			continue
		}
		eden, survivor, old := classifyMemoryPool(match[1])
		switch {
		case eden && isUsed:
			edenUsed += value
		case eden && isMax:
			edenMax += value
		case survivor && isUsed:
			survivorUsed += value
		case survivor && isMax:
			survivorMax += value
		case old && isUsed:
			oldUsed += value
		case old && isMax:
			oldMax += value
		}
	}

	setPoolMetric := func(name string, value float64) {
		if err := sample.SetMetric(name, value, metric.GAUGE); err != nil {
			log.Error("Error setting value: %s", err)
		}
	}
	setPoolMetric("jvm.heapEdenUsedBytes", edenUsed)
	setPoolMetric("jvm.heapEdenMaxBytes", edenMax)
	setPoolMetric("jvm.heapSurvivorUsedBytes", survivorUsed)
	setPoolMetric("jvm.heapSurvivorMaxBytes", survivorMax)
	setPoolMetric("jvm.heapOldGenUsedBytes", oldUsed)
	setPoolMetric("jvm.heapOldGenMaxBytes", oldMax)
}

// CollectMetricDefinitions collects the set of metrics from the current open JMX connection and add them to the sample
func CollectMetricDefinitions(sample *metric.Set, metricSets []*JMXMetricSet, beanModifier BeanModifier, conn connection.JMXConnection) {
	notFoundMetrics := make([]string, 0)

	for _, metricSet := range metricSets {
		beanName := metricSet.MBean

		if beanModifier != nil {
			beanName = beanModifier(beanName)
		}

		// Return all the results under a specific mBean
		results, err := conn.QueryMBeanAttributes(beanName)
		// If we fail we don't want a total failure as other metrics can be collected even if a single failure/timout occurs
		if err != nil {
			if jmxConnErr, ok := gojmx.IsJMXConnectionError(err); ok {
				log.Error("Connection error for %s:%s : %s", jmx.HostName(), jmx.Port(), jmxConnErr)
				os.Exit(1)
			}
			log.Error("Unable to execute JMX query for MBean '%s': %v", beanName, err)
			continue
		}

		attrsByName := make(map[string]*gojmx.AttributeResponse)
		for _, attr := range results {
			if attr.ResponseType == gojmx.ResponseTypeErr {
				log.Warn("Failed to process attribute for query: %s status: %s", attr.Name, attr.StatusMsg)
				continue
			}
			attrsByName[attr.Name] = attr
		}

		// For each metric to collect, populate the sample if it is
		// found in results, otherwise save the mBeanKey as not found
		for _, metricDef := range metricSet.MetricDefs {
			mBeanKey := metricSet.MetricPrefix + metricDef.JMXAttr
			if beanModifier != nil {
				mBeanKey = beanModifier(mBeanKey)
			}
			if value, ok := attrsByName[mBeanKey]; !ok {
				notFoundMetrics = append(notFoundMetrics, metricDef.Name)
			} else {
				if err := sample.SetMetric(metricDef.Name, value.GetValue(), metricDef.SourceType); err != nil {
					log.Error("Error setting value: %s", err)
				}
			}
		}
	}

	if len(notFoundMetrics) > 0 {
		log.Warn("Can't find raw metrics in results for keys: %v", notFoundMetrics)
	}
}

// getTopicListFromJMX discovers which topics a producer or consumer client is talking to.
// entityType ("Producer" or "Consumer") picks the client's own topic-metrics MBean via
// getAllTopicsFromJMX/topicMetricsMBeanForEntityType.
func getTopicListFromJMX(clientID, entityType string, conn connection.JMXConnection) ([]string, error) {
	switch strings.ToLower(args.GlobalArgs.TopicMode) {
	case "none":
		return []string{}, nil
	case "list":
		return args.GlobalArgs.TopicList, nil
	case "regex":
		if args.GlobalArgs.TopicRegex == "" {
			return nil, errors.New("regex topic mode requires the topic_regex argument to be set")
		}

		pattern, err := regexp.Compile(args.GlobalArgs.TopicRegex)
		if err != nil {
			return nil, fmt.Errorf("failed to compile topic regex: %s", err)
		}

		allTopics, err := getAllTopicsFromJMX(clientID, entityType, conn)
		if err != nil {
			return nil, fmt.Errorf("failed to get topics from client: %s", err)
		}

		filteredTopics := make([]string, 0, len(allTopics))
		for _, topic := range allTopics {
			if pattern.MatchString(topic) {
				filteredTopics = append(filteredTopics, topic)
			}
		}

		return filteredTopics, nil
	case "all":
		allTopics, err := getAllTopicsFromJMX(clientID, entityType, conn)
		if err != nil {
			return nil, fmt.Errorf("failed to get topics from client: %s", err)
		}
		return allTopics, nil
	default:
		log.Error("Invalid topic mode %s", args.GlobalArgs.TopicMode)
		return nil, fmt.Errorf("invalid topic_mode '%s'", args.GlobalArgs.TopicMode)
	}

}

// topicMetricsMBeanForEntityType returns the per-client, per-topic MBean pattern also used by
// ConsumerTopicMetricDefs/ProducerTopicMetricDefs.
func topicMetricsMBeanForEntityType(clientID, entityType string) string {
	if entityType == "Consumer" {
		return fmt.Sprintf("kafka.consumer:type=consumer-fetch-manager-metrics,client-id=%s,topic=*", clientID)
	}
	return fmt.Sprintf("kafka.producer:type=producer-topic-metrics,client-id=%s,topic=*", clientID)
}

func getAllTopicsFromJMX(clientID, entityType string, conn connection.JMXConnection) ([]string, error) {
	result, err := conn.QueryMBeanAttributes(topicMetricsMBeanForEntityType(clientID, entityType))
	// If we fail we don't want a total failure as other metrics can be collected even if a single failure/timout occurs
	if err != nil {
		if jmxErr, ok := gojmx.IsJMXError(err); ok {
			return nil, jmxErr
		}
		log.Error("Connection error for %s:%s : %s", jmx.HostName(), jmx.Port(), err)
		os.Exit(1)
	}

	r := regexp.MustCompile(`topic="?([^,"]+)"?`)
	uniqueTopics := make(map[string]struct{})
	for _, attr := range result {
		match := r.FindStringSubmatch(attr.Name)
		if match == nil {
			continue
		}

		uniqueTopics[match[1]] = struct{}{}
	}

	topics := make([]string, 0, len(uniqueTopics))
	for topic := range uniqueTopics {
		topics = append(topics, topic)
	}

	return topics, nil
}
