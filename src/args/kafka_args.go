package args

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	sdkArgs "github.com/newrelic/infra-integrations-sdk/args"
	"github.com/newrelic/infra-integrations-sdk/log"
)

// GlobalArgs represents the global arguments that were passed in
var GlobalArgs *KafkaArguments

// KafkaArguments is an special version of the config arguments that has advanced parsing
// to allow arguments to be consumed easier.
type KafkaArguments struct {
	sdkArgs.DefaultArgumentList
	ClusterName            string
	ZookeeperHosts         []*ZookeeperHost
	ZookeeperAuthScheme    string
	ZookeeperAuthSecret    string
	ZookeeperPath          string
	DefaultJMXUser         string
	DefaultJMXPassword     string
	CollectBrokerTopicData bool
	Producers              []*JMXHost
	Consumers              []*JMXHost
	TopicMode              string
	TopicList              []string
	TopicRegex             string
	Timeout                int
	CollectTopicSize       bool

	// SSL options
	KeyStore           string
	KeyStorePassword   string
	TrustStore         string
	TrustStorePassword string

	// Consumer offset arguments
	ConsumerOffset     bool
	ConsumerGroups     ConsumerGroups
	ConsumerGroupRegex *regexp.Regexp
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
	err := json.Unmarshal([]byte(a.ZookeeperHosts), &zookeeperHosts)
	if err != nil {
		log.Error("Failed to parse zookeepers from json")
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
		log.Error("Failed to parse consumers from json")
		return nil, err
	}

	// Parse producers
	producers, err := unmarshalJMXHosts([]byte(a.Producers), &a)
	if err != nil {
		log.Error("Failed to parse producers from json")
		return nil, err
	}

	// Parse topics
	var topics []string
	if err = json.Unmarshal([]byte(a.TopicList), &topics); err != nil {
		log.Error("Failed to parse topics from json")
		return nil, err
	}

	// Parse consumser offset args
	consumerGroups, err := unmarshalConsumerGroups(a.ConsumerOffset, a.ConsumerGroups)
	if err != nil {
		log.Error("Error with Consumer Group configuration: %s", err.Error())
		return nil, err
	}

	var consumerGroupRegex *regexp.Regexp
	if a.ConsumerGroupRegex != "" {
		consumerGroupRegex, err = regexp.Compile(a.ConsumerGroupRegex)
		if err != nil {
			log.Error("Error parsing consumer_group_regex as a regex pattern")
			return nil, err
		}
	}

	parsedArgs := &KafkaArguments{
		DefaultArgumentList:    a.DefaultArgumentList,
		ClusterName:            a.ClusterName,
		ZookeeperHosts:         zookeeperHosts,
		ZookeeperAuthScheme:    a.ZookeeperAuthScheme,
		ZookeeperAuthSecret:    a.ZookeeperAuthSecret,
		ZookeeperPath:          a.ZookeeperPath,
		DefaultJMXUser:         a.DefaultJMXUser,
		DefaultJMXPassword:     a.DefaultJMXPassword,
		CollectBrokerTopicData: a.CollectBrokerTopicData,
		Producers:              producers,
		Consumers:              consumers,
		TopicMode:              a.TopicMode,
		TopicList:              topics,
		TopicRegex:             a.TopicRegex,
		Timeout:                a.Timeout,
		KeyStore:               a.KeyStore,
		KeyStorePassword:       a.KeyStorePassword,
		TrustStore:             a.TrustStore,
		TrustStorePassword:     a.TrustStorePassword,
		CollectTopicSize:       a.CollectTopicSize,
		ConsumerOffset:         a.ConsumerOffset,
		ConsumerGroups:         consumerGroups,
		ConsumerGroupRegex:     consumerGroupRegex,
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
			return nil, errors.New("must specify a name for each producer in the list")
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

// ConsumerGroups is the structure to represent the whitelist for
// consumer_groups argument
type ConsumerGroups map[string]map[string][]int32

func unmarshalConsumerGroups(consumerOffset bool, consumerGroupsArg string) (ConsumerGroups, error) {
	// not in consumer offset mode so don't bother to unmarshal
	if !consumerOffset {
		return nil, nil
	}

	data := []byte(consumerGroupsArg)
	var consumerGroups ConsumerGroups
	if err := json.Unmarshal(data, &consumerGroups); err != nil {
		return nil, err
	}

	return consumerGroups, validateConsumerGroups(consumerGroups)
}

func validateConsumerGroups(groups ConsumerGroups) error {
	for groupName, topics := range groups {
		if len(topics) == 0 {
			return fmt.Errorf("consumer group '%s' contains no topics, at least one topic must be specified", groupName)
		}
	}

	return nil
}
