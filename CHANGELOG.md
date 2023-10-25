# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

## Unreleased

## v3.4.9 - 2023-10-25

### ⛓️ Dependencies
- Updated golang.org/x/sync to v0.4.0
- Updated golang version to 1.21

## v3.4.8 - 2023-08-02

### ⛓️ Dependencies
- Updated golang to v1.20.7

## v3.4.7 - 2023-07-19

### ⛓️ Dependencies
- Updated github.com/newrelic/nrjmx/gojmx digest

## v3.4.6 - 2023-06-28

### ⛓️ Dependencies
- Updated golang.org/x/sync to v0.3.0

## v3.4.5 - 2023-05-31

### ⛓️ Dependencies
- Updated github.com/stretchr/testify to v1.8.4 - [Changelog 🔗](https://github.com/stretchr/testify/releases/tag/v1.8.4)

## v3.4.4 - 2023-05-24

### ⛓️ Dependencies
- Updated github.com/stretchr/testify to v1.8.3 - [Changelog 🔗](https://github.com/stretchr/testify/releases/tag/v1.8.3)

## v3.4.3 - 2023-04-26

### ⛓️ Dependencies
- Updated github.com/newrelic/nrjmx/gojmx digest

## v3.4.2 - 2023-04-05

### ⛓️ Dependencies
- Updated golang to v1.20

## v3.4.1 - 2023-03-20

### 🐞 Bug fixes
- Compile the integration statically

## v3.4.0 - 2023-03-16

### 🛡️ Security notices
- Bump dependencies: `golang.org/x/net` v0.6.0 --> v0.7.0

### Dependencies
- Bump to golang 1.20
- Bump github.com/stretchr/testify from 1.8.1 to 1.8.2 (#212)

## v3.3.0 - 2023-02-20

### 🛡️ Security notices
- Bump dependencies: `github.com/Shopify/sarama` v1.37.2 --> v1.38.1

### Dependencies
- Bump to golang 1.19
- `golang.org/x/sync` v0.0.0-20220923202941-7f9b1623fab7 --> v0.1.0

## 3.2.3 (2022-12-1)
### Changed
- Bump dependencies:
  `nrjmx` v2.0.1 -> v2.3.2

## 3.2.2 (2022-11-14)
### Fixed
- Fixed a bug caused by the sarama library in some scenarios. It can return a nil assigment and nil error for an unassigned consumer in a topic.

## 3.2.1 (2022-10-18)
### Changed
- Bump dependencies:
  `sarama` v1.36.0 -> v1.37.2
  `gojmx` v0.0.0-20220818135048-2c786ece1d31 -> v0.0.0-20221004104925-6819f176e889
  `sync` v0.0.0-20220722155255-886fb9371eb4 -> v0.0.0-20220923202941-7f9b1623fab7
- Changed Dependency:
  `github.com/xdg/scram` v0.0.0-20180814205039-7eeb5667e42c -> `github.com/xdg-go/scram` v1.1.1

## 3.2.0 (2022-08-24)
### Changed
- Fixed a bug that prevented -collect_topic_size and -collect_topic_offset from working as expected 
- Bump dependencies:
  `gojmx` v0.0.0-20220801092610-d842b96425bf -> v0.0.0-20220818135048-2c786ece1d31

## 3.1.0 (2022-08-16)
### Changed
- Bump dependencies:
  `sarama` v1.34.1 -> v1.36.0
  `gojmx` v0.0.0-20220422131558-879b33f09229 -> v0.0.0-20220801092610-d842b96425bf
  `sync` v0.0.0-20210220032951-036812b2e83c -> v0.0.0-20220722155255-886fb9371eb4

## 3.0.0 (2022-07-12)

This new version adds two new features to monitor inactive consumers. We encourage you to take a look at this new features in [here](https://docs.newrelic.com/docs/infrastructure/host-integrations/host-integrations-list/kafka/kafka-integration/#offset-monitoring).

### Breaking changes
- CONSUMER_GROUPS flag removed, CONSUMER_GROUP_REGEX should be used instead.
- COLLECT_BROKER_TOPIC_DATA has been removed (It was not having any effect).
- BOOTSTRAP_BROKER_JMX_USER and BOOTSTRAP_BROKER_JMX_PASSWORD are honored taking precedence to DEFAULT_JMX_USER and DEFAULT_JMX_BROKER on Bootsrap discovery.

### Added
- Autodetect Consumer/Producer names (clientID) (https://github.com/newrelic/nri-kafka/pull/178) 
- Offset collection for consumer groups with inactive consumers using flag INACTIVE_CONSUMER_GROUP_OFFSET (https://github.com/newrelic/nri-kafka/pull/172)
- Report consumer-group offset metrics by topic using flag CONSUMER_GROUP_OFFSET_BY_TOPIC (https://github.com/newrelic/nri-kafka/pull/172)

## 2.19.1 (2022-05-16)
### Added
- Extra logging for broker collection on verbose mode

## 2.19.0 (2022-05-02)
### Changed
- Upgrade pipeline to Go 1.18
- Bump dependencies:
  `sarama` v1.31.1 -> v1.32.0
  `infra-integrations-sdk` v3.7.1+incompatible -> v3.7.2+incompatible
  `gojmx` v0.0.0-20220119164809-485b71bffc97 -> v0.0.0-20220422131558-879b33f09229
  `go-zookeeper` v0.0.0-20190923202752-2cc03de413da -> v0.0.0-20201211165307-7117e9ea2414

## 2.18.1 (2022-02-22)
### Changed
- Add the creation of the consumer image for testing/canaries.

## 2.18.0 (2022-01-12)
### Changed
- Upgraded Sarama library to 1.30.1 in order to support Kafka latest versions (up to 3.0)
- Added consumer-offset integration test

## 2.17.0 (2021-06-27)
### Changed
- Moved default config.sample to [V4](https://docs.newrelic.com/docs/create-integrations/infrastructure-integrations-sdk/specifications/host-integrations-newer-configuration-format/), added a dependency for infra-agent version 1.20.0

Please notice that old [V3](https://docs.newrelic.com/docs/create-integrations/infrastructure-integrations-sdk/specifications/host-integrations-standard-configuration-format/) configuration format is deprecated, but still supported.

## 2.16.2 (2021-06-08)
### Changed
- Support for ARM

## 2.16.1 (2021-06-07)
### Changed
- New argument topic_source to be either broker or zookeeper (default broker)  

## 2.16.0 (2021-06-02)
### Changed
- Upgraded github.com/newrelic/infra-integrations-sdk to v3.6.7
- Added integration tests
- JMX logging shows host and port

## 2.15.1 (2021-05-26)
### Changed
- Add debug broker logs from Zookeeper

## 2.15.0 (2021-03-24)
### Changed
- Upgraded github.com/newrelic/infra-integrations-sdk to v3.6.6
- Switched to go modules
- Upgraded pipeline to go 1.16
- Replaced gometalinter with golangci-lint

## 2.14.1 (2021-03-24)
### Changed
- Fixed a bug that prevented -sasl_gssapi_disable_fast_negotiation from working as expected
## 2.14.0 (2021-03-17)
### Changed
- expose option to disable FAST negotiation to resolve issues with Active Directory
- Adds a basic TCP connection check to JMX port of all brokers
- Make nrjmx binary name relative to PATH

## 2.13.9 (2020-12-07)
### Changed
- Added closing of Zookeeper connections to avoid having them timeout and flood Zookeeper logs


## 2.13.8 (2019-10-08)
### Changed
- Removed the consumer group limit for consumer offset collection

## 2.13.7 (2019-09-28)
### Changed
- Updated Sarama to latest version

## 2.13.6 (2019-08-26)
### Fixed
- Incorrect command in sample log

## 2.13.5 (2019-08-18)
### Fixed
- Exit after a connection error is returned from a JMX query

## 2.13.4 (2019-05-29)
### Fixed
- Bug in which bootstrap broker collection would combine multi-broker collection results

## 2.13.3 (2019-05-29)
### Fixed
- Broker ID mismatch

## 2.13.2 (2019-04-30)
### Changed
- Add host to producer and consumer entity IDs

## 2.13.1 (2019-04-30)
### Changed
- Raise topic limit to 10000

## 2.13.0 (2019-04-20)
### Added
- SASL/PLAIN and SASL/SCRAM auth methods

## 2.12.0 (2019-03-30)
### Added
- TLS configuration for both SSL and SASL_SSL

## 2.11.3 (2019-03-26)
### Added
- Kerberos authentication

## 2.11.2 (2019-02-06)
### Changed
- Make consumer/producer config independent of broker config

## 2.10.0 (2019-02-06)
### Added
- `collect_topic_offset` argument. Useful when you want offsets for topics that aren't actively being consumed

## 2.10.0 (2019-02-06)
### Added
- kafka_version argument to lower the targeted API version if necessary
- Cluster name attribute on metrics

## 2.9.3 (2019-02-06)
### Added
- Capture internal logging for Sarama

## 2.9.2 (2019-02-05)
### Fixed
- Panic on failed client connection

## 2.9.1 (2019-01-29)
### Fixed
- Broken makefile

## 2.9.0 (2019-01-29)
### Added
- Connect to kafka with a bootstrap broker

### Changed
- Removed hard dependency on zookeeper configuration
- Local-only broker metric collection

## 2.8.2 (2019-01-22)
### Added
- Additional debug logging around offset collection

## 2.8.1 (2019-01-18)
### Fixed
- Avoid segfault by detecting error connecting

## 2.8.0 (2019-12-05)
### Added
- Topic bucket option that allows topic collection to be split between multiple instances

## 2.7.0 (2019-12-04)
### Changed
- Bundle nrjmx

## 2.6.0 (2019-11-18)
### Changed
- Renamed the integration executable from nr-kafka to nri-kafka in order to be consistent with the package naming. **Important Note:** if you have any security module rules (eg. SELinux), alerts or automation that depends on the name of this binary, these will have to be updated.

## 2.5.1 - 2019-11-15
### Fixed
- Remove windows definition from linux build

## 2.5.0 - 2019-11-14
### Added
- Support for attempting multiple connection protocols to broker

## 2.4.1 - 2019-11-12
### Added
- Rollup metrics for consumer offsets

## 2.4.0 - 2019-10-25
### Added
- `consumer_group_regex` argument
-
### Changed
- Deprecated `consumer_groups` in favor of `consumer_group_regex`

## 2.3.2 - 2019-09-11
### Fixed
- Add port to broker entityName to increase uniqueness

## 2.3.1 - 2019-08-29
### Fixed
- Better log messages for failed hwm collection

## 2.3.0 - 2019-08-20
### Added
- Support for PROTOCOL_MAP configuration in Zookeeper

## 2.2.1 - 2019-07-15
### Changed
- Added documentation in sample config file for regex list mode

## 2.2.0 - 2019-07-11
### Added
- Regex list mode for Kafka

## 2.1.0 - 2019-06-25
### Added
- SSL support for JMX connections

## 2.0.0 - 2019-04-22
### Changed
- Changed the entity namespaces to be kafka-scoped
- ClusterName is a required argument to better enforce uniquene

## 1.1.1 - 2019-02-04
### Changed
- Updated Definition file protocol version to 2

## 1.1.0 - 2018-11-20
### Fixed
- Host broker hosts and ports are parsed from Zookeeper
- Added SSL support for consumer offset feature

## 1.0.0 - 2018-11-16
### Changed
- Updated to 1.0.0

## 0.3.0 - 2018-11-07
### Changed
- Updated sample file with correct offset example
- Renamed `topic.bytesWritten` to `broker.bytesWrittenToTopicPerSecond`

## 0.2.3 - 2018-10-23
### Changed
- Values for `topic_mode` are now lower cased

## 0.2.2 - 2018-10-01
### Added
- Hardcoded limit for topic collection

## 0.2.1 - 2018-09-25
### Changed
- Source Type on all metrics with `attr=Count` to `Rate`
### Removed
- Dynamic lookup of Consumer Groups and Topics on Consumer Offset mode. The dynamic lookup would only correctly work in very specific set of circumstances.

## 0.2.0 - 2018-09-17
### Added
- Consumer offsets and lag

## 0.1.5 - 2018-09-12
### Added
- Renamed kafka-config.yml.template to kafka-config.yml.sample
- Added comments to kafka-config.yml.sample file
- Fixed spellings in spec.csv file

## 0.1.4 - 2018-09-10
### Added
- Added nrjmx dependency when installing the package

## 0.1.3 - 2018-09-06
### Added
- Added additional debug logging statements to assist in debugging a customers environment

## 0.1.2 - 2018-08-30
### Added
- Fixed topic_mode argument to parse `List` as a mode rather than `Specific` to match documentation
- Changed `kafka-config.yml.sample` to `kafka-config.yml.template`

## 0.1.1 - 2018-08-29
### Added
- Added zookeeper_path argument

## 0.1.0 - 2018-07-03
### Added
- Initial version: Includes Metrics and Inventory data
