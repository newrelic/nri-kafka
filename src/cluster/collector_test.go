package cluster

import (
	"testing"

	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/connection/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollector_CollectMetrics(t *testing.T) {
	// Set up mock arguments
	args.GlobalArgs = &args.ParsedArguments{
		ClusterName: "test-cluster",
		ClusterID:   "lkc-abc123",
	}

	// Create integration
	i, err := integration.New("test", "1.0.0")
	require.NoError(t, err)

	// Create a mock JMX client
	mockJMX := mocks.NewEmptyMockJMXProvider()

	// Create collector with mock JMX client and a pre-computed active controller count
	collector := NewCollector(mockJMX, 1)

	// Create the entity using the collector's Entity method
	entity, err := collector.Entity(i)
	require.NoError(t, err)

	// Verify entity was created with correct metadata - identified by clusterName, not by
	// whichever broker's JMX happened to answer this collection run. No ID attributes: the
	// entity key is just namespace:name.
	assert.Equal(t, ClusterName, entity.Metadata.Namespace)
	assert.Equal(t, args.GlobalArgs.ClusterName, entity.Metadata.Name)
	assert.Equal(t, 1, len(i.Entities))
	assert.Empty(t, entity.Metadata.IDAttrs)

	err = collector.CollectMetrics(i)
	require.NoError(t, err)

	require.Len(t, entity.Metrics, 1)
	sample := entity.Metrics[0]

	expected := map[string]interface{}{
		"event_type":                    ClusterEventType,
		"displayName":                   args.GlobalArgs.ClusterName,
		"entityName":                    "cluster:" + args.GlobalArgs.ClusterName,
		"clusterName":                   args.GlobalArgs.ClusterName,
		"clusterId":                     args.GlobalArgs.ClusterID,
		"cluster.activeControllerCount": float64(1),
	}
	assert.Equal(t, expected, sample.Metrics)
}

func TestCollector_Entity_FallsBackToClusterIDWhenClusterNameUnset(t *testing.T) {
	args.GlobalArgs = &args.ParsedArguments{
		ClusterName: "",
		ClusterID:   "lkc-abc123",
	}

	i, err := integration.New("test", "1.0.0")
	require.NoError(t, err)

	collector := NewCollector(mocks.NewEmptyMockJMXProvider(), 1)

	entity, err := collector.Entity(i)
	require.NoError(t, err)

	// cluster_name is optional and has no default - clusterId is auto-populated from broker
	// metadata, so it's what the entity should be named when cluster_name isn't set.
	assert.Equal(t, "lkc-abc123", entity.Metadata.Name)
	assert.Empty(t, entity.Metadata.IDAttrs)
}

func TestCollector_Entity_ErrorsWhenNeitherClusterNameNorClusterIDSet(t *testing.T) {
	args.GlobalArgs = &args.ParsedArguments{
		ClusterName: "",
		ClusterID:   "",
	}

	i, err := integration.New("test", "1.0.0")
	require.NoError(t, err)

	collector := NewCollector(mocks.NewEmptyMockJMXProvider(), 1)

	_, err = collector.Entity(i)
	require.Error(t, err)
}
