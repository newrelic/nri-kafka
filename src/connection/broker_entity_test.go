package connection_test

import (
	"testing"

	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/connection"
	"github.com/newrelic/nri-kafka/src/connection/mocks"
	"github.com/newrelic/nri-kafka/src/testutils"
	"github.com/stretchr/testify/assert"
)

func Test_Broker_Entity_IncludesClusterID(t *testing.T) {
	testutils.SetupTestArgs()
	args.GlobalArgs.ClusterID = "lkc-abc123"

	mockBroker := &mocks.SaramaBroker{}
	mockBroker.On("Addr").Return("kafkabroker:9090")

	b := &connection.Broker{
		Host:         "kafkabroker",
		ID:           "0",
		SaramaBroker: mockBroker,
	}

	i, err := integration.New("kafka", "1.0.0")
	assert.NoError(t, err)

	entity, err := b.Entity(i)
	assert.NoError(t, err)

	assert.Contains(t, entity.Metadata.IDAttrs, integration.NewIDAttribute("clusterID", "lkc-abc123"))
}
