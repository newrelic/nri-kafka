package args

import (
	"reflect"
	"testing"

	"github.com/kr/pretty"
	sdkArgs "github.com/newrelic/infra-integrations-sdk/args"
	"github.com/newrelic/infra-integrations-sdk/integration"
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
		ZookeeperHosts:         `[{"host":"host1","port":2180},{"host":"host2"}]`,
		ZookeeperAuthScheme:    "",
		ZookeeperAuthSecret:    "",
		ZookeeperPath:          "/test",
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
		ConsumerOffset:         false,
		ConsumerGroups:         "[]",
	}

	expectedArgs := &KafkaArguments{
		DefaultArgumentList: sdkArgs.DefaultArgumentList{
			Verbose:   false,
			Pretty:    false,
			Inventory: false,
			Metrics:   false,
			Events:    false,
		},
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
		ZookeeperAuthScheme:    "",
		ZookeeperAuthSecret:    "",
		ZookeeperPath:          "/test",
		DefaultJMXUser:         "admin1",
		DefaultJMXPassword:     "admin2",
		CollectBrokerTopicData: true,
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
		Consumers:      []*JMXHost{},
		TopicMode:      "Specific",
		TopicList:      []string{"test1", "test2", "test3"},
		Timeout:        1000,
		ConsumerOffset: false,
		ConsumerGroups: nil,
	}
	parsedArgs, err := ParseArgs(a)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(parsedArgs, expectedArgs) {
		t.Errorf("Argument parsing did not return expected results. %v", pretty.Diff(parsedArgs, expectedArgs))
	}
}

func TestDefaultArgs(t *testing.T) {
	var a ArgumentList
	_, err := integration.New("name", "1.0.0", integration.Args(&a))

	expectedArgs := &KafkaArguments{
		DefaultArgumentList: sdkArgs.DefaultArgumentList{
			Verbose:   false,
			Pretty:    false,
			Inventory: false,
			Metrics:   false,
			Events:    false,
		},
		ZookeeperHosts:         []*ZookeeperHost{},
		ZookeeperAuthScheme:    "",
		ZookeeperAuthSecret:    "",
		ZookeeperPath:          "",
		DefaultJMXUser:         "admin",
		DefaultJMXPassword:     "admin",
		CollectBrokerTopicData: true,
		Producers:              []*JMXHost{},
		Consumers:              []*JMXHost{},
		TopicMode:              "None",
		TopicList:              []string{},
		Timeout:                2000,
		CollectTopicSize:       false,
		ConsumerOffset:         false,
		ConsumerGroups:         nil,
	}

	parsedArgs, err := ParseArgs(a)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(parsedArgs, expectedArgs) {
		t.Errorf("Argument parsing did not return expected results. %v", pretty.Diff(parsedArgs, expectedArgs))
	}

}

func Test_unmarshalConsumerGroups_All(t *testing.T) {
	expected := make(ConsumerGroups, 0)

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
