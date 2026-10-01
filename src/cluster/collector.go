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

// Collector collects metrics at the cluster level, ideally from the controller broker.
type Collector struct {
	jmxClient             connection.JMXConnection
	activeControllerCount int
}

// NewCollector creates a new collector for cluster metrics. activeControllerCount is computed
// by countActiveControllers in kafka.go.
func NewCollector(jmxClient connection.JMXConnection, activeControllerCount int) *Collector {
	return &Collector{
		jmxClient:             jmxClient,
		activeControllerCount: activeControllerCount,
	}
}

// CollectMetrics collects metrics from the Kafka controller
func (c *Collector) CollectMetrics(integration *integration.Integration) error {
	clusterEntity, err := c.Entity(integration)
	if err != nil {
		return fmt.Errorf("failed to create cluster entity: %v", err)
	}

	if args.GlobalArgs.HasMetrics() {
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

	// No ID attributes: clusterName is already the entity Name, so repeating it as an ID
	// attribute would be redundant, and clusterId is deliberately not used to disambiguate -
	// two different clusters sharing a cluster_name will collide into one entity.
	return i.Entity(entityName, ClusterName)
}

// populateClusterMetrics collects all cluster metrics and adds them to the entity
func populateClusterMetrics(entity *integration.Entity, conn connection.JMXConnection, activeControllerCount int) {
	// entity.Metadata.Name is the single source of truth for display purposes - it already
	// reflects whichever of clusterName/clusterId Entity() resolved to.
	attrs := []attribute.Attribute{
		{Key: "displayName", Value: entity.Metadata.Name},
		{Key: "entityName", Value: "cluster:" + entity.Metadata.Name},
		{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
	}
	attrs = append(attrs, args.ClusterIDAttribute()...)
	attrs = append(attrs, attribute.Attribute{Key: "event_type", Value: ClusterEventType})
	sample := entity.NewMetricSet(ClusterEventType, attrs...)

	metrics.CollectMetricDefinitions(sample, metrics.ClusterMetricDefs, nil, conn)

	if err := sample.SetMetric("cluster.activeControllerCount", activeControllerCount, metric.GAUGE); err != nil {
		log.Error("Error setting value: %s", err)
	}
}
