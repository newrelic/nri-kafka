// Package connection implements connection code
package connection

import "github.com/IBM/sarama"

// MetadataRequestVersionForClusterID is the minimum Kafka Metadata protocol version whose
// response includes ClusterID (KIP-78, Kafka 0.10.1+). Use this exact version, rather than
// negotiating up to whatever KafkaVersion is configured, when the only thing a metadata call
// needs is the broker list and cluster ID.
const MetadataRequestVersionForClusterID = 2

// ClusterIDFromMetadata extracts the Kafka-native cluster ID from a metadata response.
// ClusterID is only populated by brokers when the request negotiates Metadata protocol
// version 2 or higher. Returns "" if unavailable.
func ClusterIDFromMetadata(metadata *sarama.MetadataResponse) string {
	if metadata == nil || metadata.ClusterID == nil {
		return ""
	}

	return *metadata.ClusterID
}
