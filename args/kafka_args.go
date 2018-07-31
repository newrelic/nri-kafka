package args

import (
	"encoding/json"
	"errors"

	sdkArgs "github.com/newrelic/infra-integrations-sdk/args"
)

// KafkaArguments is an special version of the config arguments that has advanced parsing
// to allow arguments to be consumed easier.
type KafkaArguments struct {
	sdkArgs.DefaultArgumentList
	ZookeeperHosts         []*ZookeeperHost
	ZookeeperAuthScheme    string
	ZookeeperAuthSecret    string
	DefaultJMXUser         string
	DefaultJMXPassword     string
	CollectBrokerTopicData bool
	Producers              []*JMXHost
	Consumers              []*JMXHost
	TopicMode              string
	TopicList              []string
	Timeout                int
	CollectTopicSize       bool
}

// ZookeeperHost is a storage struct for ZooKeeper connection information
type ZookeeperHost struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// JMXHost is a storage struct for producer and consumer connection information
type JMXHost struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
}

// ParseArgs validates the arguments in argumentList and parses them
// into more easily used structs
func ParseArgs(a ArgumentList) (*KafkaArguments, error) {

	// Parse ZooKeeper hosts
	var zookeeperHosts []*ZookeeperHost
	if a.ZookeeperHosts != "" {
		err := json.Unmarshal([]byte(a.ZookeeperHosts), &zookeeperHosts)
		if err != nil {
			return nil, err
		}
	} else {
		zookeeperHosts = make([]*ZookeeperHost, 0)
	}

	for _, zookeeperHost := range zookeeperHosts {
		// Set port to default if unset
		if zookeeperHost.Port == 0 {
			zookeeperHost.Port = defaultZookeeperPort
		}
	}

	// Parse consumers
	consumers, err := unmarshalJMXHosts([]byte(a.Consumers), &a)
	if err != nil {
		return nil, err
	}

	// Parse producers
	producers, err := unmarshalJMXHosts([]byte(a.Producers), &a)
	if err != nil {
		return nil, err
	}

	// Parse topics
	var topics []string
	if err = json.Unmarshal([]byte(a.TopicList), &topics); err != nil {
		return nil, err
	}

	parsedArgs := &KafkaArguments{
		DefaultArgumentList:    a.DefaultArgumentList,
		ZookeeperHosts:         zookeeperHosts,
		ZookeeperAuthScheme:    a.ZookeeperAuthScheme,
		ZookeeperAuthSecret:    a.ZookeeperAuthSecret,
		DefaultJMXUser:         a.DefaultJMXUser,
		DefaultJMXPassword:     a.DefaultJMXPassword,
		CollectBrokerTopicData: a.CollectBrokerTopicData,
		Producers:              producers,
		Consumers:              consumers,
		TopicMode:              a.TopicMode,
		TopicList:              topics,
		Timeout:                a.Timeout,
		CollectTopicSize:       a.CollectTopicSize,
	}

	return parsedArgs, nil
}

// unmarshalJMXHosts parses the user-provided JSON map for a producer
// or consumers into a jmxHost structs and sets default values
func unmarshalJMXHosts(data []byte, a *ArgumentList) ([]*JMXHost, error) {

	// Parse the producer or consumer
	var v []*JMXHost
	if err := json.Unmarshal([]byte(data), &v); err != nil {
		return nil, err
	}

	// Set default values
	for _, p := range v {
		if p.Name == "" {
			return nil, errors.New("Must specify a name for each producer in the list")
		}
		if p.User == "" {
			p.User = a.DefaultJMXUser
		}
		if p.Password == "" {
			p.Password = a.DefaultJMXPassword
		}
		if p.Port == 0 {
			p.Port = a.DefaultJMXPort
		}
		if p.Host == "" {
			p.Host = a.DefaultJMXHost
		}
	}

	return v, nil
}
