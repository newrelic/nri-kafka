// Package cluster handles collection of Kafka cluster-level metrics
package cluster

import (
	"fmt"
	"strings"

	"github.com/newrelic/infra-integrations-sdk/v3/data/attribute"
	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/connection"
	"github.com/newrelic/nri-kafka/src/metrics"
)

const (
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
	clusterEntity, err := c.Entity(integration)
	if err != nil {
		return fmt.Errorf("failed to create cluster entity: %v", err)
	}

	// Collect metrics only if metrics collection is enabled
	if args.GlobalArgs.HasMetrics() {
		// Collect cluster metrics
		populateClusterMetrics(clusterEntity, c.hostPort, c.jmxClient)
	}

	return nil
}

// Entity gets the entity object for the cluster
func (c *Collector) Entity(i *integration.Integration) (*integration.Entity, error) {
	host := c.hostPort
	if host == "" {
		host = "unknown:0"
	}

	// Get hostname and port from hostPort
	hostParts := strings.Split(host, ":")
	hostname := hostParts[0]
	port := "0"
	if len(hostParts) > 1 {
		port = hostParts[1]
	}

	// For broker entities, the entityName is just the host:port
	// and the namespace is "ka-broker". Let's follow the same pattern.
	entityName := fmt.Sprintf("%s:%s", hostname, port)

	// Get cluster name from args if available
	clusterName := ""
	if args.GlobalArgs != nil {
		clusterName = args.GlobalArgs.ClusterName
	}

	// Follow the broker entity pattern: only use clusterName as an ID attribute
	clusterIDAttr := integration.NewIDAttribute("clusterName", clusterName)

	// Don't include host and port attributes in the entity key
	// as they are already part of the entityName
	return i.Entity(entityName, ClusterName, clusterIDAttr)
}

// populateClusterMetrics collects all cluster metrics and adds them to the entity
func populateClusterMetrics(entity *integration.Entity, hostPort string, conn connection.JMXConnection) {
	// Create metrics sample
	sample := entity.NewMetricSet(ClusterEventType,
		attribute.Attribute{Key: "displayName", Value: hostPort},
		attribute.Attribute{Key: "entityName", Value: "cluster:" + hostPort},
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
