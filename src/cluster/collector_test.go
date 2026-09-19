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

	// Create a collector with test parameters
	hostPort := "localhost:9999"

	// Create a mock JMX client
	mockJMX := mocks.NewEmptyMockJMXProvider()

	// Create collector with mock JMX client and a pre-computed active controller count
	collector := NewCollector(mockJMX, hostPort, 1)

	// Create the entity using the collector's Entity method
	entity, err := collector.Entity(i)
	require.NoError(t, err)

	// Verify entity was created with correct metadata
	assert.Equal(t, ClusterName, entity.Metadata.Namespace)
	assert.Equal(t, hostPort, entity.Metadata.Name)
	assert.Equal(t, 1, len(i.Entities))
	assert.Contains(t, entity.Metadata.IDAttrs, integration.NewIDAttribute("clusterId", "lkc-abc123"))

	err = collector.CollectMetrics(i)
	require.NoError(t, err)

	require.Len(t, entity.Metrics, 1)
	sample := entity.Metrics[0]

	expected := map[string]interface{}{
		"event_type":                    ClusterEventType,
		"displayName":                   hostPort,
		"entityName":                    "cluster:" + hostPort,
		"clusterName":                   args.GlobalArgs.ClusterName,
		"clusterId":                     args.GlobalArgs.ClusterID,
		"cluster.activeControllerCount": float64(1),
	}
	assert.Equal(t, expected, sample.Metrics)
}
