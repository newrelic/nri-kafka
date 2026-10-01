package connection

import "github.com/IBM/sarama"

// MetadataRequestVersionForClusterID is the minimum protocol version that returns ClusterID (KIP-78).
const MetadataRequestVersionForClusterID = 2

// ClusterIDFromMetadata returns "" if the broker didn't report a cluster ID.
func ClusterIDFromMetadata(metadata *sarama.MetadataResponse) string {
	if metadata == nil || metadata.ClusterID == nil {
		return ""
	}

	return *metadata.ClusterID
}
