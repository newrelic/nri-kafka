// Package testutils contains setup functions for tests that initialize common variables in the utils package.
package testutils

import (
	"github.com/newrelic/nri-kafka/args"
	"github.com/newrelic/nri-kafka/utils"
)

// SetupTestArgs sets up a basic value for KafkaArgs with CollectBrokerTopicData
// set to true
func SetupTestArgs() {
	utils.KafkaArgs = &args.KafkaArguments{CollectBrokerTopicData: true}
}

// SetupJmxTesting sets all JMX wrapper variables to basic shells
func SetupJmxTesting() {
	utils.JMXOpen = func(hostname, port, username, password string) error { return nil }
	utils.JMXClose = func() {}
	utils.JMXQuery = func(query string, timeout int) (map[string]interface{}, error) { return map[string]interface{}{}, nil }
}
