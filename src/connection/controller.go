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

	client, err := NewSaramaClientFromBrokerList(brokers)
	if err != nil {
		log.Error("Failed to create client to find controller: %s", err)
		return nil
	}
	defer client.Close()

	controllerBroker, err := client.Controller()
	if err != nil {
		log.Error("Failed to get controller broker: %s", err)
		return nil
	}

	controllerIDStr := fmt.Sprintf("%d", controllerBroker.ID())
	log.Debug("Found controller broker with ID: %s", controllerIDStr)

	for _, broker := range brokers {
		// broker.ID is a string in our struct; sarama's is an int32, hence the conversion above.
		if broker.ID == controllerIDStr {
			log.Debug("Found controller broker: %s (ID: %s)", broker.Host, broker.ID)
			return broker
		}
	}

	log.Debug("Controller broker with ID %s not found in provided broker list", controllerIDStr)
	return nil
}
