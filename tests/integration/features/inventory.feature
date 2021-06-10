# file: inventory.feature
Feature: inventory
  In order to see the kafka entities in NR
  As a NROne user
  I need to be able to get only inventory metrics

  Scenario: should get inventory
    Given a 3 node kafka is running and has 3 topics
    When nri-kafa is set to inventory and executed
    Then the response should match the inventory json schema
    And the response should have the 3 topics metrics
