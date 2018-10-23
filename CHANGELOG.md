# Change Log

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

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
