package consumeroffset

import (
	"testing"

	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/testutils"
	"github.com/stretchr/testify/assert"
)

func Test_setMetrics(t *testing.T) {
	testutils.SetupTestArgs()

	i, _ := integration.New("test", "test")
	offsetData := []*partitionOffsets{
		{
			Topic:          "testTopic",
			Partition:      "0",
			ConsumerOffset: func() *int64 { i := int64(123); return &i }(),
			HighWaterMark:  func() *int64 { i := int64(125); return &i }(),
			ConsumerLag:    func() *int64 { i := int64(2); return &i }(),
		},
	}

	err := setMetrics("testGroup", offsetData, i)
	assert.NoError(t, err)

	clusterIDAttr := integration.NewIDAttribute("clusterName", args.GlobalArgs.ClusterName)
	resultEntity, err := i.Entity("testGroup", "ka-consumerGroup", clusterIDAttr)
	assert.NoError(t, err)
	assert.Len(t, resultEntity.Metrics, 1)
	assert.Len(t, resultEntity.Metrics[0].Metrics, 9)
}
