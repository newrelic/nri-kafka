package main

import (
	"encoding/json"
	"errors"

	sdkArgs "github.com/newrelic/infra-integrations-sdk/args"
)

// Define the default ports for zookeeper and JMX
const (
	defaultZookeeperPort = 2181
	defaultJMXPort       = 9999
)

// argumentList is the raw arguments passed into the integration via yaml or CLI args
type argumentList struct {
	sdkArgs.DefaultArgumentList
	ZookeeperHosts         string `default:"[{\"host\":\"localhost\", \"port\":2181}]" help:"JSON array of ZooKeeper hosts with the following fields: host, port. Port defaults to 2181"`
	ZookeeperAuthScheme    string `default:"" help:"ACL scheme for authenticating ZooKeeper connection."`
	ZookeeperAuthSecret    string `default:"" help:"Authentication string for ZooKeeper."`
	DefaultJMXPort         int    `default:"9999" help:"Default port for JMX collection."`
	DefaultJMXHost         string `default:"localhost" help:"Default host for JMX collection."`
	DefaultJMXUser         string `default:"admin" help:"Default JMX username. Useful if all JMX hosts use the same JMX username and password."`
	DefaultJMXPassword     string `default:"admin" help:"Default JMX password. Useful if all JMX hosts use the same JMX username and password."`
	CollectBrokerTopicData bool   `default:"true" help:"Signals to collect Broker and Topic inventory and metrics. Should only be turned off when specifying a Zookeeper Host and not intending to collect Broker or detailed Topic data."`
	TopicMode              string `default:"None" help:"Possible options are All, None, or List. If List, must also specify the list of topics to collect with the topic_list option."`
	TopicList              string `default:"[]" help:"JSON array of strings with the names of topics to monitor. Only used if collect_topics is set to 'List'"`
	CollectTopicSize       bool   `default:"false" help:"Enablement of on disk Topic size metric collection. This metric can be very resource intensive to collect especially against many topics."`
	Producers              string `default:"[]" help:"JSON array of producer key:value maps with the keys 'name', 'host', 'port', 'user', 'password'. The 'name' key is required, the others default to the specified defaults in the default_jmx_* options.  "`
	Consumers              string `default:"[]" help:"JSON array of consumer key:value maps with the keys 'name', 'host', 'port', 'user', 'password'. The 'name' key is required, the others default to the specified defaults in the default_jmx_* options.  "`
	Timeout                int    `default:"2000" help:"Timeout in milliseconds per single JMX query."`
}

// kafkaArguments is an special version of the config arguments that has advanced parsing
// to allow arguments to be consumed easier.
type kafkaArguments struct {
	sdkArgs.DefaultArgumentList
	ZookeeperHosts         []*zookeeperHost
	ZookeeperAuthScheme    string
	ZookeeperAuthSecret    string
	DefaultJMXUser         string
	DefaultJMXPassword     string
	CollectBrokerTopicData bool
	Producers              []*jmxHost
	Consumers              []*jmxHost
	TopicMode              string
	TopicList              []string
	Timeout                int
	CollectTopicSize       bool
}

// zookeeperHost is a storage struct for ZooKeeper connection information
type zookeeperHost struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// jmxHost is a storage struct for producer and consumer connection information
type jmxHost struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
}

// parseArgs validates the arguments in argumentList and parses them
// into more easily used structs
func parseArgs(a argumentList) (*kafkaArguments, error) {

	// Parse ZooKeeper hosts
	var zookeeperHosts []*zookeeperHost
	err := json.Unmarshal([]byte(a.ZookeeperHosts), &zookeeperHosts)
	if err != nil {
		return nil, err
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

	parsedArgs := &kafkaArguments{
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
func unmarshalJMXHosts(data []byte, a *argumentList) ([]*jmxHost, error) {

	// Parse the producer or consumer
	var v []*jmxHost
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
