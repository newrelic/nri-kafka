// Package testutils contains setup functions for tests that initialize common variables in the utils package.
package testutils

import (
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/jmxwrapper"
)

// SetupTestArgs sets up a basic value for KafkaArgs with CollectBrokerTopicData
// set to true
func SetupTestArgs() {
	args.GlobalArgs = &args.KafkaArguments{CollectBrokerTopicData: true}
}

// SetupJmxTesting sets all JMX wrapper variables to basic shells
func SetupJmxTesting() {
	jmxwrapper.JMXOpen = func(hostname, port, username, password string) error { return nil }
	jmxwrapper.JMXClose = func() {}
	jmxwrapper.JMXQuery = func(query string, timeout int) (map[string]interface{}, error) { return map[string]interface{}{}, nil }
}
