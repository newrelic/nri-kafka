# New Relic Infrastructure Integration for Kafka


The New Relic Infrastructure Integration for Kafka captures critical performance metrics and inventory reported by Kafka clusters. Data on Brokers, Topics, Java Producers, and Java Consumers is collected.

Inventory data is obtained mainly from Zookeeper nodes and metrics are collected through JMX.

See our [documentation web site](https://docs.newrelic.com/docs/integrations/host-integrations/host-integrations-list/kafka-monitoring-integration) for more details.

## Requirements

JMX is required to be enabled on the following entities in order to be able to collect metrics:

- Brokers
- Java Producers
- Java Consumers

Information on configuring JMX can be found [here](https://docs.oracle.com/javase/8/docs/technotes/guides/management/agent.html).

## Installation

- download an archive file for the `Kafka` Integration
- extract `kafka-definition.yml` and `/bin` directory into `/var/db/newrelic-infra/newrelic-integrations`
- add execute permissions for the binary file `nri-kafka` (if required)
- extract `kafka-config.yml.sample` into `/etc/newrelic-infra/integrations.d`

## Usage

This is the description about how to run the Kafka Integration with New Relic Infrastructure agent, so it is required to have the agent installed (see [agent installation](https://docs.newrelic.com/docs/infrastructure/new-relic-infrastructure/installation/install-infrastructure-linux)).

In order to use the Kafka Integration it is required to configure `kafka-config.yml.sample` file. Firstly, rename the file to `kafka-config.yml`. Then, depending on your needs, specify all instances that you want to monitor. Once this is done, restart the Infrastructure agent.

You can view your data in Insights by creating your own custom NRQL queries. To do so use the **KafkaBrokerSample**, **KafkaTopicSample**, **KafkaProducerSample**, or **KafkaConsumerSample** event type.

## Compatibility

* Supported OS: No limitations
* Kafka versions: 0.8+

## Integration Development usage

Assuming that you have source code you can build and run the Kafka Integration locally.

* Go to directory of the Kafka Integration and build it
```bash
$ make
```
* The command above will execute tests for the Kafka Integration and build an executable file called `nri-kafka` in `bin` directory.
```bash
$ ./bin/nri-kafka
```
* If you want to know more about usage of `./nri-kafka` check
```bash
$ ./bin/nri-kafka -help
```

For managing external dependencies [govendor tool](https://github.com/kardianos/govendor) is used. It is required to lock all external dependencies to specific version (if possible) into vendor directory.
