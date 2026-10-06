package connection

import (
	"testing"

	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/stretchr/testify/assert"
)

func TestFindControllerBrokerWithEmptyList(t *testing.T) {
	log.SetupLogging(false)

	controllerBroker := FindControllerBroker([]*Broker{})
	assert.Nil(t, controllerBroker, "Should return nil for empty broker list")
}
