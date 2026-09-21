// Package cluster handles collection of Kafka cluster-level metrics
package cluster

import (
	"errors"
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
	activeControllerCount int // pre-computed sum of ActiveControllerCount across every broker
}

// NewCollector creates a new collector for cluster metrics. activeControllerCount is the
// pre-computed sum of every broker's own ActiveControllerCount (see countActiveControllers) -
// in a healthy cluster this is exactly 1; 0 or >1 both indicate a real problem.
func NewCollector(jmxClient connection.JMXConnection, activeControllerCount int) *Collector {
	return &Collector{
		jmxClient:             jmxClient,
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
		populateClusterMetrics(clusterEntity, c.jmxClient, c.activeControllerCount)
	}

	return nil
}

// Entity gets the entity object for the cluster, identified by clusterName/clusterId - not by
// whichever broker's JMX happened to answer this collection run, which can differ across
// polling intervals and hosts (controller re-election, discovery order) and would otherwise
// fragment one logical cluster into many entities.
func (c *Collector) Entity(i *integration.Integration) (*integration.Entity, error) {
	clusterName := ""
	clusterID := ""
	if args.GlobalArgs != nil {
		clusterName = args.GlobalArgs.ClusterName
		clusterID = args.GlobalArgs.ClusterID
	}

	// clusterName is optional (no default) - fall back to clusterId, which is auto-populated
	// from broker metadata, rather than passing the SDK an empty entity name and failing to
	// collect any cluster metrics at all just because cluster_name wasn't set.
	entityName := clusterName
	if entityName == "" {
		entityName = clusterID
	}
	if entityName == "" {
		return nil, errors.New("cluster_name is not set and no clusterId was available from broker metadata; set the cluster_name config option to enable cluster metrics")
	}

	clusterNameAttr := integration.NewIDAttribute("clusterName", clusterName)
	clusterIDAttr := integration.NewIDAttribute("clusterId", clusterID)

	return i.Entity(entityName, ClusterName, clusterNameAttr, clusterIDAttr)
}

// populateClusterMetrics collects all cluster metrics and adds them to the entity
func populateClusterMetrics(entity *integration.Entity, conn connection.JMXConnection, activeControllerCount int) {
	// entity.Metadata.Name is the single source of truth for display purposes - it already
	// reflects whichever of clusterName/clusterId Entity() resolved to.
	sample := entity.NewMetricSet(ClusterEventType,
		attribute.Attribute{Key: "displayName", Value: entity.Metadata.Name},
		attribute.Attribute{Key: "entityName", Value: "cluster:" + entity.Metadata.Name},
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
