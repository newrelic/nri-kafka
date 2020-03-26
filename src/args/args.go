// Package args contains a structs for a passed in argument list. Also, the KafkaArguments struct
// which is a specially parsed version of the args to be used within the integration.
package args

import (
	sdkArgs "github.com/newrelic/infra-integrations-sdk/args"
)

// ArgumentList is the raw arguments passed into the integration via yaml or CLI args
type ArgumentList struct {
	sdkArgs.DefaultArgumentList
	ClusterName  string `default:"" help:"A user-defined name to uniquely identify the cluster"`
	KafkaVersion string `default:"1.0.0" help:"What version of the API to use for connecting to Kafka. Versions older than 1.0.0 may be missing some features."`

	AutodiscoverStrategy string `default:"zookeeper" help:"How to discover broker nodes to collect metrics from. One of 'zookeeper' (default) or 'bootstrap'"`

	// Zookeeper autodiscovery. Only required if using zookeeper to autodiscover brokers
	ZookeeperHosts      string `default:"[]" help:"JSON array of ZooKeeper hosts with the following fields: host, port. Port defaults to 2181"`
	ZookeeperAuthScheme string `default:"" help:"ACL scheme for authenticating ZooKeeper connection."`
	ZookeeperAuthSecret string `default:"" help:"Authentication string for ZooKeeper."`
	ZookeeperPath       string `default:"" help:"The Zookeeper path which contains the Kafka configuration. A leading slash is required."`
	PreferredListener   string `default:"" help:"Override which broker listener to attempt to connect to. Defaults to the first that works"`

	// Bootstrap broker autodiscovery. Only required if using `bootstrap` as your AutodiscoverStrategy
	BootstrapBrokerHost          string `default:"localhost" help:"The bootstrap broker host"`
	BootstrapBrokerKafkaPort     int    `default:"9092" help:"The bootstrap broker Kafka port"`
	BootstrapBrokerKafkaProtocol string `default:"PLAINTEXT" help:"The protocol to connect to the bootstrap broker with"`
	BootstrapBrokerJMXPort       int    `default:"0" help:"The JMX port for the bootstrap broker"`
	BootstrapBrokerJMXUser       string `default:"" help:"The JMX username for the bootstrap broker"`
	BootstrapBrokerJMXPassword   string `default:"" help:"The JMX password for the bootstrap broker"`

	// Producer and consumer connection info. No autodiscovery is supported for producers and consumers
	Producers string `default:"[]" help:"JSON array of producer key:value maps with the keys 'name', 'host', 'port', 'user', 'password'. The 'name' key is required, the others default to the specified defaults in the default_jmx_* options.  "`
	Consumers string `default:"[]" help:"JSON array of consumer key:value maps with the keys 'name', 'host', 'port', 'user', 'password'. The 'name' key is required, the others default to the specified defaults in the default_jmx_* options.  "`

	// JMX defaults
	DefaultJMXPort     int    `default:"9999" help:"Default port for JMX collection."`
	DefaultJMXHost     string `default:"localhost" help:"Default host for JMX collection."`
	DefaultJMXUser     string `default:"admin" help:"Default JMX username. Useful if all JMX hosts use the same JMX username and password."`
	DefaultJMXPassword string `default:"admin" help:"Default JMX password. Useful if all JMX hosts use the same JMX username and password."`

	// JMX SSL options
	KeyStore           string `default:"" help:"The location for the keystore containing JMX Client's SSL certificate"`
	KeyStorePassword   string `default:"" help:"Password for the SSL Key Store"`
	TrustStore         string `default:"" help:"The location for the keystore containing JMX Server's SSL certificate"`
	TrustStorePassword string `default:"" help:"Password for the SSL Trust Store"`

	NrJmx string `default:"/usr/bin/nrjmx" help:"Path to the nrjmx executable"`

	// Kerberos auth args
	SaslGssapiRealm              string `default:"" help:"Kerberos realm"`
	SaslGssapiServiceName        string `default:"" help:"Kerberos service name"`
	SaslGssapiUsername           string `default:"" help:"Kerberos username"`
	SaslGssapiKeyTabPath         string `default:"" help:"Kerberos key tab path"`
	SaslGssapiKerberosConfigPath string `default:"/etc/krb5.conf" help:"Kerberos config path"`

	// Collection configuration
	LocalOnlyCollection    bool   `default:"false" help:"Collect only the metrics related to the configured bootstrap broker. Useful for distributed metric collection"`
	CollectBrokerTopicData bool   `default:"true" help:"DEPRECATED -- Signals to collect Broker and Topic inventory and metrics. Should only be turned off when specifying a Zookeeper Host and not intending to collect Broker or detailed Topic data."`
	TopicMode              string `default:"None" help:"Possible options are All, None, or List. If List, must also specify the list of topics to collect with the topic_list option."`
	TopicList              string `default:"[]" help:"JSON array of strings with the names of topics to monitor. Only used if collect_topics is set to 'List'"`
	TopicRegex             string `default:"" help:"A regex pattern that matches the list of topics to collect. Only used if collect_topics is set to 'Regex'"`
	TopicBucket            string `default:"1/1" help:"Allows the partitioning of topic collection across multiple instances. The second number is the number of instances topics are partitioned across. The first number is the bucket number of the current instance, which should be between 1 and the second number."`
	CollectTopicSize       bool   `default:"false" help:"Enablement of on disk Topic size metric collection. This metric can be very resource intensive to collect especially against many topics."`
	CollectTopicOffset     bool   `default:"false" help:"Enablement of Topic offsets collection. This metric can be very resource intensive to collect especially against many topics."`

	// Consumer offset arguments
	ConsumerOffset     bool   `default:"false" help:"Populate consumer offset data"`
	ConsumerGroups     string `default:"{}" help:"DEPRECATED -- JSON Object whitelist of consumer groups to their topics and topics to their partitions, in which to collect consumer offsets for."`
	ConsumerGroupRegex string `default:"" help:"A regex pattern matching the consumer groups to collect"`

	Timeout int `default:"10000" help:"Timeout in milliseconds per single JMX query."`
}
