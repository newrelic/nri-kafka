// Package args contains a structs for a passed in argument list. Also, the KafkaArguments struct
// which is a specially parsed version of the args to be used within the integration.
package args

import (
	sdkArgs "github.com/newrelic/infra-integrations-sdk/args"
)

// Define the default ports for zookeeper and JMX
const (
	defaultZookeeperPort = 2181
	defaultJMXPort       = 9999
)

// ArgumentList is the raw arguments passed into the integration via yaml or CLI args
type ArgumentList struct {
	sdkArgs.DefaultArgumentList
	ClusterName         string `default:"" help:"A user-defined name to uniquely identify the cluster"`
	ZookeeperHosts      string `default:"[]" help:"JSON array of ZooKeeper hosts with the following fields: host, port. Port defaults to 2181"`
	ZookeeperAuthScheme string `default:"" help:"ACL scheme for authenticating ZooKeeper connection."`
	ZookeeperAuthSecret string `default:"" help:"Authentication string for ZooKeeper."`
	ZookeeperPath       string `default:"" help:"The Zookeeper path which contains the Kafka configuration. A leading slash is required."`
	DefaultJMXPort      int    `default:"9999" help:"Default port for JMX collection."`
	DefaultJMXHost      string `default:"localhost" help:"Default host for JMX collection."`
	DefaultJMXUser      string `default:"admin" help:"Default JMX username. Useful if all JMX hosts use the same JMX username and password."`
	DefaultJMXPassword  string `default:"admin" help:"Default JMX password. Useful if all JMX hosts use the same JMX username and password."`

	CollectBrokerTopicData bool   `default:"true" help:"Signals to collect Broker and Topic inventory and metrics. Should only be turned off when specifying a Zookeeper Host and not intending to collect Broker or detailed Topic data."`
	TopicMode              string `default:"None" help:"Possible options are All, None, or List. If List, must also specify the list of topics to collect with the topic_list option."`
	TopicList              string `default:"[]" help:"JSON array of strings with the names of topics to monitor. Only used if collect_topics is set to 'List'"`
	TopicRegex             string `default:"" help:"A regex pattern that matches the list of topics to collect. Only used if collect_topics is set to 'Regex'"`
	CollectTopicSize       bool   `default:"false" help:"Enablement of on disk Topic size metric collection. This metric can be very resource intensive to collect especially against many topics."`
	Producers              string `default:"[]" help:"JSON array of producer key:value maps with the keys 'name', 'host', 'port', 'user', 'password'. The 'name' key is required, the others default to the specified defaults in the default_jmx_* options.  "`
	Consumers              string `default:"[]" help:"JSON array of consumer key:value maps with the keys 'name', 'host', 'port', 'user', 'password'. The 'name' key is required, the others default to the specified defaults in the default_jmx_* options.  "`
	Timeout                int    `default:"10000" help:"Timeout in milliseconds per single JMX query."`

	// SSL options
	KeyStore           string `default:"" help:"The location for the keystore containing JMX Client's SSL certificate"`
	KeyStorePassword   string `default:"" help:"Password for the SSL Key Store"`
	TrustStore         string `default:"" help:"The location for the keystore containing JMX Server's SSL certificate"`
	TrustStorePassword string `default:"" help:"Password for the SSL Trust Store"`

	// Consumer offset arguments
	ConsumerOffset     bool   `default:"false" help:"Populate consumer offset data"`
	ConsumerGroups     string `default:"{}" help:"DEPRECATED -- JSON Object whitelist of consumer groups to their topics and topics to their partitions, in which to collect consumer offsets for."`
	ConsumerGroupRegex string `default:"" help:"A regex pattern matching the consumer groups to collect"`
}
