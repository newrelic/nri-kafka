package args

import (
	"reflect"
	"regexp"
	"testing"

	"github.com/Shopify/sarama"
	sdkArgs "github.com/newrelic/infra-integrations-sdk/args"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		AutodiscoverStrategy:             "zookeeper",
		KafkaVersion:                     "1.1.0",
		ZookeeperHosts:                   `[{"host":"host1","port":2180},{"host":"host2"}]`,
		ZookeeperAuthScheme:              "",
		ZookeeperAuthSecret:              "",
		ZookeeperPath:                    "/test",
		BootstrapBrokerKafkaPort:         9092,
		BootstrapBrokerJMXUser:           "",
		BootstrapBrokerJMXPassword:       "",
		DefaultJMXUser:                   "admin1",
		DefaultJMXPassword:               "admin2",
		DefaultJMXHost:                   "test-default-host",
		DefaultJMXPort:                   9998,
		NrJmx:                            "/usr/bin/nrjmx",
		Producers:                        `[{"name":"producer1", "host":"producerhost","user":"a1","password":"p1","port":9995},{"name":"producer2"}]`,
		Consumers:                        "[]",
		TopicMode:                        "Specific",
		TopicList:                        `["test1", "test2", "test3"]`,
		TopicBucket:                      `1/3`,
		Timeout:                          1000,
		ConsumerOffset:                   false,
		ConsumerGroupRegex:               ".*",
		SaslMechanism:                    "PLAIN",
		SaslUsername:                     "admin3",
		SaslPassword:                     "secret1",
		SaslGssapiDisableFASTNegotiation: true,
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
		BootstrapBroker:     nil,
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
		Consumers:                        []*JMXHost{},
		TopicMode:                        "Specific",
		TopicList:                        []string{"test1", "test2", "test3"},
		TopicBucket:                      TopicBucket{1, 3},
		Timeout:                          1000,
		ConsumerOffset:                   false,
		ConsumerGroupRegex:               regexp.MustCompile(".*"),
		SaslMechanism:                    "PLAIN",
		SaslUsername:                     "admin3",
		SaslPassword:                     "secret1",
		SaslGssapiDisableFASTNegotiation: true,
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
		ZookeeperAuthScheme:              "",
		ZookeeperAuthSecret:              "",
		ZookeeperPath:                    "",
		BootstrapBroker:                  nil,
		DefaultJMXUser:                   "admin",
		DefaultJMXPassword:               "admin",
		MaxJMXConnections:                3,
		NrJmx:                            "nrjmx",
		Producers:                        []*JMXHost{},
		Consumers:                        []*JMXHost{},
		TopicMode:                        "None",
		TopicList:                        []string{},
		TopicBucket:                      TopicBucket{1, 1},
		Timeout:                          10000,
		CollectTopicSize:                 false,
		CollectTopicOffset:               false,
		ConsumerOffset:                   false,
		ConsumerGroupRegex:               nil,
		SaslGssapiKerberosConfigPath:     "/etc/krb5.conf",
		SaslMechanism:                    "GSSAPI",
		SaslGssapiDisableFASTNegotiation: false,
		TopicSource:                      "broker",
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

func TestUnMarshalJMXHosts(t *testing.T) {
	arguments := &ArgumentList{
		DefaultJMXUser:     "user",
		DefaultJMXPassword: "password",
		DefaultJMXPort:     42,
		DefaultJMXHost:     "host",
	}
	cases := []struct {
		Name     string
		Input    string
		Expected []*JMXHost
	}{
		{
			Name:     "Empty",
			Input:    `[]`,
			Expected: []*JMXHost{},
		},
		{
			Name:  "Default with no alias",
			Input: `[{}]`,
			Expected: []*JMXHost{
				{
					User:     arguments.DefaultJMXUser,
					Password: arguments.DefaultJMXPassword,
					Port:     arguments.DefaultJMXPort,
					Host:     arguments.DefaultJMXHost,
				},
			},
		},
		{
			Name:  "Default with alias",
			Input: `default`,
			Expected: []*JMXHost{
				{
					User:     arguments.DefaultJMXUser,
					Password: arguments.DefaultJMXPassword,
					Port:     arguments.DefaultJMXPort,
					Host:     arguments.DefaultJMXHost,
				},
			},
		},
		{
			Name:  "Only name set",
			Input: `[{"name": "client.id"}]`,
			Expected: []*JMXHost{
				{
					Name:     "client.id",
					User:     arguments.DefaultJMXUser,
					Password: arguments.DefaultJMXPassword,
					Port:     arguments.DefaultJMXPort,
					Host:     arguments.DefaultJMXHost,
				},
			},
		},
		{
			Name:  "No name set",
			Input: `[{"user": "my.user", "password": "my.pass", "port": 1088, "host": "localhost"}]`,
			Expected: []*JMXHost{
				{
					User:     "my.user",
					Password: "my.pass",
					Port:     1088,
					Host:     "localhost",
				},
			},
		},
		{
			Name:  "All values set",
			Input: `[{"name": "my.name", "user": "my.user", "password": "my.pass", "port": 1088, "host": "localhost"}]`,
			Expected: []*JMXHost{
				{
					Name:     "my.name",
					User:     "my.user",
					Password: "my.pass",
					Port:     1088,
					Host:     "localhost",
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			parsed, err := unmarshalJMXHosts([]byte(c.Input), arguments)
			require.NoError(t, err)
			require.Equal(t, c.Expected, parsed)
		})
	}
}
