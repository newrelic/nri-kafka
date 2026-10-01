package cluster

import (
	"testing"

	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/connection/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectMetrics(t *testing.T) {
	args.GlobalArgs = &args.ParsedArguments{
		ClusterName: "test-cluster",
		ClusterID:   "lkc-abc123",
	}

	i, err := integration.New("test", "1.0.0")
	require.NoError(t, err)

	entity, err := Entity(i)
	require.NoError(t, err)

	assert.Equal(t, ClusterName, entity.Metadata.Namespace)
	assert.Equal(t, args.GlobalArgs.ClusterName, entity.Metadata.Name)
	assert.Equal(t, 1, len(i.Entities))
	assert.Empty(t, entity.Metadata.IDAttrs)

	err = CollectMetrics(i, mocks.NewEmptyMockJMXProvider(), 1)
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

func TestEntity_FallsBackToClusterIDWhenClusterNameUnset(t *testing.T) {
	args.GlobalArgs = &args.ParsedArguments{
		ClusterName: "",
		ClusterID:   "lkc-abc123",
	}

	i, err := integration.New("test", "1.0.0")
	require.NoError(t, err)

	entity, err := Entity(i)
	require.NoError(t, err)

	assert.Equal(t, "lkc-abc123", entity.Metadata.Name)
	assert.Empty(t, entity.Metadata.IDAttrs)
}

func TestEntity_ErrorsWhenNeitherClusterNameNorClusterIDSet(t *testing.T) {
	args.GlobalArgs = &args.ParsedArguments{
		ClusterName: "",
		ClusterID:   "",
	}

	i, err := integration.New("test", "1.0.0")
	require.NoError(t, err)

	_, err = Entity(i)
	require.Error(t, err)
}
