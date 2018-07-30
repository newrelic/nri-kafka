package metrics

import (
	"github.com/newrelic/infra-integrations-sdk/data/metric"
)

const topicHolder = "%TOPIC%"

// MetricDefinition defines a single Infrastructure metric
type MetricDefinition struct {
	Name       string
	SourceType metric.SourceType
	JMXAttr    string
}

// JMXMetricSet defines a set of MetricDefinitions that
// can be collected from a specific MBean
type JMXMetricSet struct {
	MBean        string
	MetricPrefix string
	MetricDefs   []*MetricDefinition
}
