package args

import (
	"slices"

	"github.com/newrelic/infra-integrations-sdk/v3/data/attribute"
)

// idAttribute wraps key/value as an attribute, or returns nil if the customer disabled key
// via DisableAttributes. Callers append the result to their sample's attribute list.
func idAttribute(key, value string) []attribute.Attribute {
	if slices.Contains(GlobalArgs.DisableAttributes, key) {
		return nil
	}
	return []attribute.Attribute{{Key: key, Value: value}}
}

// ClusterIDAttribute returns the clusterId attribute, or nil if disabled.
func ClusterIDAttribute() []attribute.Attribute {
	return idAttribute("clusterId", GlobalArgs.ClusterID)
}

// BrokerIDAttribute returns the brokerId attribute for the given broker, or nil if disabled.
func BrokerIDAttribute(brokerID string) []attribute.Attribute {
	return idAttribute("brokerId", brokerID)
}
