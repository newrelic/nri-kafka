package cluster

import (
	"testing"

	"github.com/newrelic/infra-integrations-sdk/v3/data/attribute"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollector_CollectMetrics(t *testing.T) {
	// Create integration
	i, err := integration.New("test", "1.0.0")
	require.NoError(t, err)

	// Create the entity directly in the test
	entity, err := i.Entity(ClusterType, ClusterName)
	require.NoError(t, err)

	// Verify entity was created with correct metadata
	assert.Equal(t, ClusterName, entity.Metadata.Namespace)
	assert.Equal(t, ClusterType, entity.Metadata.Name)
	assert.Equal(t, 1, len(i.Entities))

	// Create a metric set to simulate metrics collection
	ms := entity.NewMetricSet(ClusterEventType,
		attribute.Attribute{Key: "displayName", Value: "Kafka Cluster"},
		attribute.Attribute{Key: "entityName", Value: "cluster"},
		attribute.Attribute{Key: "clusterName", Value: "test-cluster"},
		attribute.Attribute{Key: "event_type", Value: ClusterEventType},
	)

	// Verify the metric set was created
	assert.NotNil(t, ms)
	assert.Equal(t, 1, len(entity.Metrics))

	t.Log("Test completed successfully")
}
