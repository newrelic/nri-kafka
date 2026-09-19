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
	jmxClient             connection.JMXConnection
	hostPort              string // Format: host:port to identify the broker used for metrics collection
	activeControllerCount int    // pre-computed sum of ActiveControllerCount across every broker
}

// NewCollector creates a new collector for cluster metrics. activeControllerCount is the
// pre-computed sum of every broker's own ActiveControllerCount (see countActiveControllers) -
// in a healthy cluster this is exactly 1; 0 or >1 both indicate a real problem.
func NewCollector(jmxClient connection.JMXConnection, hostPort string, activeControllerCount int) *Collector {
	return &Collector{
		jmxClient:             jmxClient,
		hostPort:              hostPort,
		activeControllerCount: activeControllerCount,
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
		populateClusterMetrics(clusterEntity, c.hostPort, c.jmxClient, c.activeControllerCount)
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

	// Get cluster name/ID from args if available
	clusterName := ""
	clusterID := ""
	if args.GlobalArgs != nil {
		clusterName = args.GlobalArgs.ClusterName
		clusterID = args.GlobalArgs.ClusterID
	}

	// Follow the broker entity pattern: use clusterName and clusterID as ID attributes
	clusterNameAttr := integration.NewIDAttribute("clusterName", clusterName)
	clusterIDAttr := integration.NewIDAttribute("clusterId", clusterID)

	// Don't include host and port attributes in the entity key
	// as they are already part of the entityName
	return i.Entity(entityName, ClusterName, clusterNameAttr, clusterIDAttr)
}

// populateClusterMetrics collects all cluster metrics and adds them to the entity
func populateClusterMetrics(entity *integration.Entity, hostPort string, conn connection.JMXConnection, activeControllerCount int) {
	// Create a sample metric set for the cluster
	sample := entity.NewMetricSet(ClusterEventType,
		attribute.Attribute{Key: "displayName", Value: hostPort},
		attribute.Attribute{Key: "entityName", Value: "cluster:" + hostPort},
		attribute.Attribute{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
		attribute.Attribute{Key: "clusterId", Value: args.GlobalArgs.ClusterID},
		attribute.Attribute{Key: "event_type", Value: ClusterEventType},
	)

	// Collect all cluster metrics
	metrics.CollectMetricDefinitions(sample, metrics.ClusterMetricDefs, nil, conn)

	// activeControllerCount is pre-computed by summing every broker's own ActiveControllerCount
	// (see kafka.go's countActiveControllers) rather than read from a single broker's JMX -
	// unlike the rest of ClusterMetricDefs, this is the one cluster metric Kafka itself has no
	// single authoritative source for.
	if err := sample.SetMetric("cluster.activeControllerCount", activeControllerCount, metric.GAUGE); err != nil {
		log.Error("Error setting value: %s", err)
	}
}
