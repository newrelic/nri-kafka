package main

import (
	"reflect"
	"testing"

	"github.com/kr/pretty"
	sdkArgs "github.com/newrelic/infra-integrations-sdk/args"
	"github.com/newrelic/infra-integrations-sdk/integration"
)

func TestParseArgs(t *testing.T) {
	a := argumentList{
		DefaultArgumentList: sdkArgs.DefaultArgumentList{
			Verbose:   false,
			Pretty:    false,
			Inventory: false,
			Metrics:   false,
			Events:    false,
		},
		ZookeeperHosts:         `[{"host":"host1","port":2180},{"host":"host2"}]`,
		ZookeeperAuthScheme:    "",
		ZookeeperAuthSecret:    "",
		DefaultJMXUser:         "admin1",
		DefaultJMXPassword:     "admin2",
		DefaultJMXHost:         "test-default-host",
		DefaultJMXPort:         9998,
		CollectBrokerTopicData: true,
		Producers:              `[{"name":"producer1", "host":"producerhost","user":"a1","password":"p1","port":9995},{"name":"producer2"}]`,
		Consumers:              "[]",
		TopicMode:              "Specific",
		TopicList:              `["test1", "test2", "test3"]`,
		Timeout:                1000,
	}

	expectedArgs := &kafkaArguments{
		DefaultArgumentList: sdkArgs.DefaultArgumentList{
			Verbose:   false,
			Pretty:    false,
			Inventory: false,
			Metrics:   false,
			Events:    false,
		},
		ZookeeperHosts: []*zookeeperHost{
			{
				Host: "host1",
				Port: 2180,
			},
			{
				Host: "host2",
				Port: 2181,
			},
		},
		ZookeeperAuthScheme:    "",
		ZookeeperAuthSecret:    "",
		DefaultJMXUser:         "admin1",
		DefaultJMXPassword:     "admin2",
		CollectBrokerTopicData: true,
		Producers: []*jmxHost{
			{
				Name:     "producer1",
				Host:     "producerhost",
				Port:     9995,
				User:     "a1",
				Password: "p1",
			},
			{
				Name:     "producer2",
				Host:     "test-default-host",
				Port:     9998,
				User:     "admin1",
				Password: "admin2",
			},
		},
		Consumers: []*jmxHost{},
		TopicMode: "Specific",
		TopicList: []string{"test1", "test2", "test3"},
		Timeout:   1000,
	}
	parsedArgs, err := parseArgs(a)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(parsedArgs, expectedArgs) {
		t.Errorf("Argument parsing did not return expected results. %v", pretty.Diff(parsedArgs, expectedArgs))
	}
}

func TestDefaultArgs(t *testing.T) {
	var a argumentList
	_, err := integration.New(integrationName, integrationVersion, integration.Args(&a))

	expectedArgs := &kafkaArguments{
		DefaultArgumentList: sdkArgs.DefaultArgumentList{
			Verbose:   false,
			Pretty:    false,
			Inventory: false,
			Metrics:   false,
			Events:    false,
		},
		ZookeeperHosts: []*zookeeperHost{
			{
				Host: "localhost",
				Port: 2181,
			},
		},
		ZookeeperAuthScheme:    "",
		ZookeeperAuthSecret:    "",
		DefaultJMXUser:         "admin",
		DefaultJMXPassword:     "admin",
		CollectBrokerTopicData: true,
		Producers:              []*jmxHost{},
		Consumers:              []*jmxHost{},
		TopicMode:              "None",
		TopicList:              []string{},
		Timeout:                2000,
		CollectTopicSize:       false,
	}

	parsedArgs, err := parseArgs(a)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(parsedArgs, expectedArgs) {
		t.Errorf("Argument parsing did not return expected results. %v", pretty.Diff(parsedArgs, expectedArgs))
	}

}
