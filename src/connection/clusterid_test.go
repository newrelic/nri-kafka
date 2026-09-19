package connection

import (
	"testing"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
)

func Test_ClusterIDFromMetadata_ReturnsClusterID(t *testing.T) {
	clusterID := "lkc-abc123"
	metadata := &sarama.MetadataResponse{ClusterID: &clusterID}

	assert.Equal(t, "lkc-abc123", ClusterIDFromMetadata(metadata))
}

func Test_ClusterIDFromMetadata_NilClusterID(t *testing.T) {
	metadata := &sarama.MetadataResponse{ClusterID: nil}

	assert.Equal(t, "", ClusterIDFromMetadata(metadata))
}

func Test_ClusterIDFromMetadata_NilMetadata(t *testing.T) {
	assert.Equal(t, "", ClusterIDFromMetadata(nil))
}
