// Package connection implements connection code
package connection

import "github.com/IBM/sarama"

// ClusterIDFromMetadata extracts the Kafka-native cluster ID from a metadata response.
// ClusterID is only populated by brokers when the request negotiates Metadata protocol
// version 2 or higher (see sarama.NewMetadataRequest). Returns "" if unavailable.
func ClusterIDFromMetadata(metadata *sarama.MetadataResponse) string {
	if metadata == nil || metadata.ClusterID == nil {
		return ""
	}

	return *metadata.ClusterID
}
