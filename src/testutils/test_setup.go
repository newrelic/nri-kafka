// Package testutils contains setup functions for tests that initialize common variables in the utils package.
package testutils

import (
	"github.com/newrelic/nri-kafka/src/args"
)

// SetupTestArgs sets up a basic value for KafkaArgs with CollectBrokerTopicData
// set to true
func SetupTestArgs() {
	args.GlobalArgs = &args.ParsedArguments{ZookeeperPath: ""}
}
