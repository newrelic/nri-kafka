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

// CollectMetrics collects cluster-wide metrics via conn, ideally the controller broker's JMX connection.
func CollectMetrics(i *integration.Integration, conn connection.JMXConnection, activeControllerCount int) error {
	clusterEntity, err := Entity(i)
	if err != nil {
		return fmt.Errorf("failed to create cluster entity: %v", err)
	}

	if args.GlobalArgs.HasMetrics() {
		populateClusterMetrics(clusterEntity, conn, activeControllerCount)
	}

	return nil
}

// Entity gets the entity object for the cluster, keyed by clusterName/clusterId rather than
// whichever broker's JMX answered this run, to avoid fragmenting one cluster into many entities.
func Entity(i *integration.Integration) (*integration.Entity, error) {
	clusterName := ""
	clusterID := ""
	if args.GlobalArgs != nil {
		clusterName = args.GlobalArgs.ClusterName
		clusterID = args.GlobalArgs.ClusterID
	}

	// clusterName has no default; fall back to the auto-populated clusterId rather than
	// failing to collect cluster metrics just because cluster_name wasn't set.
	entityName := clusterName
	if entityName == "" {
		entityName = clusterID
	}
	if entityName == "" {
		return nil, errors.New("cluster_name is not set and no clusterId was available from broker metadata; set the cluster_name config option to enable cluster metrics")
	}

	// No ID attributes: clusterName is already the entity Name, and clusterId is deliberately
	// not used to disambiguate - two clusters sharing a cluster_name collide into one entity.
	return i.Entity(entityName, ClusterName)
}

// populateClusterMetrics collects all cluster metrics and adds them to the entity
func populateClusterMetrics(entity *integration.Entity, conn connection.JMXConnection, activeControllerCount int) {
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
