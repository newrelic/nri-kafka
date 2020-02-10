package args

import (
	"reflect"
	"regexp"
	"testing"

	"github.com/Shopify/sarama"
	sdkArgs "github.com/newrelic/infra-integrations-sdk/args"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/stretchr/testify/assert"
)

func TestParseArgs(t *testing.T) {
	a := ArgumentList{
		DefaultArgumentList: sdkArgs.DefaultArgumentList{
			Verbose:   false,
			Pretty:    false,
			Inventory: false,
			Metrics:   false,
			Events:    false,
		},
		AutodiscoverStrategy:       "zookeeper",
		KafkaVersion:               "1.1.0",
		ZookeeperHosts:             `[{"host":"host1","port":2180},{"host":"host2"}]`,
		ZookeeperAuthScheme:        "",
		ZookeeperAuthSecret:        "",
		ZookeeperPath:              "/test",
		BootstrapBrokerKafkaPort:   9092,
		BootstrapBrokerJMXUser:     "",
		BootstrapBrokerJMXPassword: "",
		DefaultJMXUser:             "admin1",
		DefaultJMXPassword:         "admin2",
		DefaultJMXHost:             "test-default-host",
		DefaultJMXPort:             9998,
		NrJmx:                      "/usr/bin/nrjmx",
		CollectBrokerTopicData: true,
		Producers:              `[{"name":"producer1", "host":"producerhost","user":"a1","password":"p1","port":9995},{"name":"producer2"}]`,
		Consumers:              "[]",
		TopicMode:              "Specific",
		TopicList:              `["test1", "test2", "test3"]`,
		TopicBucket:            `1/3`,
		Timeout:                1000,
		ConsumerOffset:         false,
		ConsumerGroups:         "[]",
		ConsumerGroupRegex:     ".*",
	}

	expectedArgs := &ParsedArguments{
		DefaultArgumentList: sdkArgs.DefaultArgumentList{
			Verbose:   false,
			Pretty:    false,
			Inventory: false,
			Metrics:   false,
			Events:    false,
		},
		AutodiscoverStrategy: "zookeeper",
		KafkaVersion:         sarama.V1_1_0_0,
		ZookeeperHosts: []*ZookeeperHost{
			{
				Host: "host1",
				Port: 2180,
			},
			{
				Host: "host2",
				Port: 2181,
			},
		},
		ZookeeperAuthScheme: "",
		ZookeeperAuthSecret: "",
		ZookeeperPath:       "/test",
		DefaultJMXUser:      "admin1",
		DefaultJMXPassword:  "admin2",
		NrJmx:               "/usr/bin/nrjmx",
		BootstrapBroker: &BrokerHost{
			Host:        "",
			KafkaPort:   9092,
			JMXPort:     9999,
			JMXUser:     "admin1",
			JMXPassword: "admin2",
		},
		Producers: []*JMXHost{
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
		Consumers:          []*JMXHost{},
		TopicMode:          "Specific",
		TopicList:          []string{"test1", "test2", "test3"},
		TopicBucket:        TopicBucket{1, 3},
		Timeout:            1000,
		ConsumerOffset:     false,
		ConsumerGroups:     nil,
		ConsumerGroupRegex: regexp.MustCompile(".*"),
	}
	parsedArgs, err := ParseArgs(a)
	assert.NoError(t, err)
	assert.Equal(t, expectedArgs, parsedArgs)
}

func TestDefaultArgs(t *testing.T) {
	var a ArgumentList
	_, err := integration.New("name", "1.0.0", integration.Args(&a))
	assert.NoError(t, err)
	a.ZookeeperHosts = `[{"host":"localhost", "port":2181}]`

	expectedArgs := &ParsedArguments{
		DefaultArgumentList: sdkArgs.DefaultArgumentList{
			Verbose:   false,
			Pretty:    false,
			Inventory: false,
			Metrics:   false,
			Events:    false,
		},
		AutodiscoverStrategy: "zookeeper",
		KafkaVersion:         sarama.V1_0_0_0,
		ZookeeperHosts: []*ZookeeperHost{
			{
				Host: "localhost",
				Port: 2181,
			},
		},
		ZookeeperAuthScheme: "",
		ZookeeperAuthSecret: "",
		ZookeeperPath:       "",
		BootstrapBroker: &BrokerHost{
			Host:          "localhost",
			KafkaPort:     9092,
			JMXPort:       9999,
			JMXUser:       "admin",
			JMXPassword:   "admin",
			KafkaProtocol: "PLAINTEXT",
		},
		DefaultJMXUser:     "admin",
		DefaultJMXPassword: "admin",
		NrJmx:              "/usr/bin/nrjmx",
		Producers:          []*JMXHost{},
		Consumers:          []*JMXHost{},
		TopicMode:          "None",
		TopicList:          []string{},
		TopicBucket:        TopicBucket{1, 1},
		Timeout:            10000,
		CollectTopicSize:   false,
		ConsumerOffset:     false,
		ConsumerGroups:     nil,
		ConsumerGroupRegex: nil,
	}

	parsedArgs, err := ParseArgs(a)
	assert.NoError(t, err)
	assert.Equal(t, expectedArgs, parsedArgs)
}

func Test_unmarshalConsumerGroups_All(t *testing.T) {
	expected := make(ConsumerGroups)

	out, err := unmarshalConsumerGroups(true, "{}")
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		t.FailNow()
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("Expected %+v got %+v", expected, out)
	}
}

func Test_unmarshalConsumerGroups(t *testing.T) {
	input := `{
		"group_1": {
			"topic_1": [
				1,
				2
			]
		}
	}`

	expected := ConsumerGroups{
		"group_1": {
			"topic_1": []int32{
				int32(1),
				int32(2),
			},
		},
	}

	out, err := unmarshalConsumerGroups(true, input)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		t.FailNow()
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("Expected %+v got %+v", expected, out)
	}
}

func Test_unmarshalConsumerGroups_NoTopics(t *testing.T) {
	input := `{
		"group_1": {
		}
	}`

	_, err := unmarshalConsumerGroups(true, input)
	if err == nil {
		t.Error("Expected error")
	}
}
