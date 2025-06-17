// Package cluster handles collection of Kafka cluster-level metrics
package cluster

import (
	"fmt"

	"github.com/newrelic/infra-integrations-sdk/v3/data/attribute"
	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/connection"
	"github.com/newrelic/nri-kafka/src/metrics"
)

const (
	// ClusterType is the entity type for the Kafka cluster
	ClusterType = "kafka:cluster"

	// ClusterName is the entity name for the Kafka cluster
	ClusterName = "ka-cluster"

	// ClusterEventType is the event type for cluster metrics
	ClusterEventType = "KafkaClusterSample"
)

// Collector collects metrics at the cluster level
// Ideally, this collector should be initialized with a connection to the controller broker,
// but it will work with any broker that has JMX enabled.
type Collector struct {
	jmxClient connection.JMXConnection
	hostPort  string // Format: host:port to identify the broker used for metrics collection
}

// NewCollector creates a new collector for cluster metrics
func NewCollector(jmxClient connection.JMXConnection, hostPort string) *Collector {
	return &Collector{
		jmxClient: jmxClient,
		hostPort:  hostPort,
	}
}

// CollectMetrics collects metrics from the Kafka controller
func (c *Collector) CollectMetrics(integration *integration.Integration) error {
	// Create entity for the cluster
	clusterEntity, err := integration.Entity(ClusterType, ClusterName)
	if err != nil {
		return fmt.Errorf("failed to create cluster entity: %v", err)
	}

	// Collect cluster metrics
	populateClusterMetrics(clusterEntity, c.hostPort, c.jmxClient)

	return nil
}

// populateClusterMetrics collects all cluster metrics and adds them to the entity
func populateClusterMetrics(entity *integration.Entity, hostPort string, conn connection.JMXConnection) {
	// Create metrics sample
	sample := entity.NewMetricSet(ClusterEventType,
		attribute.Attribute{Key: "displayName", Value: "Kafka Cluster"},
		attribute.Attribute{Key: "entityName", Value: "cluster"},
		attribute.Attribute{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
		attribute.Attribute{Key: "event_type", Value: ClusterEventType},
	)

	// Add attributes to identify the broker we collected from
	if err := sample.SetMetric("collectedFrom", hostPort, metric.ATTRIBUTE); err != nil {
		log.Error("Failed to set collectedFrom metric: %v", err)
	}

	// Collect all cluster metrics
	metrics.CollectMetricDefinitions(sample, metrics.ClusterMetricDefs, nil, conn)
}
