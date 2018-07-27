package main

import (
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-kafka/args"
)

func setupJmxTesting() {
	jmxOpenFunc = func(hostname, port, username, password string) error { return nil }
	jmxCloseFunc = func() {}
	queryFunc = func(query string, timeout int) (map[string]interface{}, error) { return map[string]interface{}{}, nil }
}

func setupTestLogger() {
	logger = log.NewStdErr(false)
}

func setupTestArgs() {
	kafkaArgs = &args.KafkaArguments{CollectBrokerTopicData: true}
}
