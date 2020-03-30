# Change Log

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

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
