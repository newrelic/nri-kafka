package utils

import (
	"github.com/newrelic/nri-kafka/args"
)

// KafkaArgs are the global set of passed in arguments
var KafkaArgs *args.KafkaArguments

// PanicOnErr checks if err is nil and if not panic
func PanicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}
