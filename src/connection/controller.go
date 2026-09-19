// Package connection implements connection code
package connection

import (
	"fmt"

	"github.com/newrelic/infra-integrations-sdk/v3/log"
)

// FindControllerBroker identifies and returns the controller broker from a list of brokers
// If the controller cannot be found, it returns nil
func FindControllerBroker(brokers []*Broker) *Broker {
	if len(brokers) == 0 {
		return nil
	}

	// Create a client from the broker list to make API calls
	client, err := NewSaramaClientFromBrokerList(brokers)
	if err != nil {
		log.Error("Failed to create client to find controller: %s", err)
		return nil
	}
	defer client.Close()

	// Get the controller broker from the client
	controllerBroker, err := client.Controller()
	if err != nil {
		log.Error("Failed to get controller broker: %s", err)
		return nil
	}

	// Get the broker ID as a string
	controllerIDStr := fmt.Sprintf("%d", controllerBroker.ID())
	log.Debug("Found controller broker with ID: %s", controllerIDStr)

	// Find the broker in our list that matches the controller ID
	for _, broker := range brokers {
		// The broker ID in our struct is a string, compare with the string version of controller ID
		if broker.ID == controllerIDStr {
			log.Debug("Found controller broker: %s (ID: %s)", broker.Host, broker.ID)
			return broker
		}
	}

	log.Debug("Controller broker with ID %s not found in provided broker list", controllerIDStr)
	return nil
}
