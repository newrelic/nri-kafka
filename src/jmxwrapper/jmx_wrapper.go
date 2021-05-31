// Package jmxwrapper contains variables for using github.com/newrelic/infra-integrations-sdk/jmx package
// while allowing everything to be mocked for testing.
package jmxwrapper

import (
	"sync"

	"github.com/newrelic/infra-integrations-sdk/jmx"
)

// JMX variable
var (
	// JMXLock is intended to be used to lock around all JMX calls.
	JMXLock sync.Mutex

	// JMXQuery is a wrapper around infra-integrations-sdk/jmx functions to allow
	// easier mocking during tests
	JMXQuery = jmx.Query

	// JMXOpen is a wrapper around infra-integrations-sdk/jmx functions to allow
	// easier mocking during tests
	JMXOpen = jmx.Open

	// JMXClose is a wrapper around infra-integrations-sdk/jmx functions to allow
	// easier mocking during tests
	JMXClose = jmx.Close

	// JMXOpen stores the value of the host where a JMX connection has been opened
	JMXHost string

	// JMXOpen stores the value of the port where a JMX connection has been opened
	JMXPort int
)
