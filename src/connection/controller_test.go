package connection

import (
	"testing"

	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/stretchr/testify/assert"
)

// Since we can't easily override NewSaramaClientFromBrokerList for testing,
// we'll focus on testing that an empty broker list returns nil
func TestFindControllerBrokerWithEmptyList(t *testing.T) {
	// Setup logging
	log.SetupLogging(false)
	
	// Test case: Empty broker list
	emptyBrokers := []*Broker{}
	controllerBroker := FindControllerBroker(emptyBrokers)
	assert.Nil(t, controllerBroker, "Should return nil for empty broker list")
}

// In a real environment, we'd use dependency injection for testability
// For now, we'll just test the empty case which doesn't require mocking
