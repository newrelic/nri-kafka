# file: all.feature
Feature: all
  In order to analyze Kafka performance
  As a NROne user
  I need to be able to get all metrics

  Scenario: should get metrics when zookeeper autodiscover strategy
    Given a 3 node kafka is running and has 3 topics
    When nri-kafa is set to zookeeper autodiscover strategy and executed
    Then the response should match the all json schema
    And the response should have the 3 topics metrics

  Scenario: should get metrics when broker bootstrap autodiscover strategy
    Given a 3 node kafka is running and has 3 topics
    When nri-kafa is set to bootstrap autodiscover strategy and executed
    Then the response should match the all json schema
    And the response should have the 3 topics metrics
