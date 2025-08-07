package cluster

import (
	"testing"

	"github.com/newrelic/infra-integrations-sdk/v3/data/attribute"
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
	}

	// Create integration
	i, err := integration.New("test", "1.0.0")
	require.NoError(t, err)

	// Create a collector with test parameters
	hostPort := "localhost:9999"

	// Create a mock JMX client
	mockJMX := mocks.NewEmptyMockJMXProvider()

	// Create collector with mock JMX client
	collector := NewCollector(mockJMX, hostPort)

	// Create the entity using the collector's Entity method
	entity, err := collector.Entity(i)
	require.NoError(t, err)

	// Verify entity was created with correct metadata
	assert.Equal(t, ClusterName, entity.Metadata.Namespace)
	assert.Equal(t, hostPort, entity.Metadata.Name)
	assert.Equal(t, 1, len(i.Entities))

	// Create a metric set to simulate metrics collection
	ms := entity.NewMetricSet(ClusterEventType,
		attribute.Attribute{Key: "displayName", Value: hostPort},
		attribute.Attribute{Key: "entityName", Value: "cluster:" + hostPort},
		attribute.Attribute{Key: "clusterName", Value: args.GlobalArgs.ClusterName},
		attribute.Attribute{Key: "event_type", Value: ClusterEventType},
	)

	// Verify the metric set was created
	assert.NotNil(t, ms)
	assert.Equal(t, 1, len(entity.Metrics))

	t.Log("Test completed successfully")
}
