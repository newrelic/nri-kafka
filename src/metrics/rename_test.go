package metrics

import (
	"testing"
)

func TestApplyProducerTopicName(t *testing.T) {
	bean := "client-id=" + producerHolder + ",topic=" + topicHolder
	producerName, topicName := "producer", "topic"

	expected := "client-id=" + producerName + ",topic=" + topicName

	beanModifierFunc := ApplyProducerTopicName(producerName, topicName)

	if out := beanModifierFunc(bean); out != expected {
		t.Errorf("Expected '%s' got '%s'", expected, out)
	}
}

func TestApplyConsumerTopicName(t *testing.T) {
	bean := "client-id=" + consumerHolder + ",topic=" + topicHolder
	consumerName, topicName := "consumer", "topic"

	expected := "client-id=" + consumerName + ",topic=" + topicName

	beanModifierFunc := ApplyConsumerTopicName(consumerName, topicName)

	if out := beanModifierFunc(bean); out != expected {
		t.Errorf("Expected '%s' got '%s'", expected, out)
	}
}

func TestApplyTopicName(t *testing.T) {
	bean := "topic=" + topicHolder
	topicName := "topic"

	expected := "topic=" + topicName

	beanModifierFunc := ApplyTopicName(topicName)

	if out := beanModifierFunc(bean); out != expected {
		t.Errorf("Expected '%s' got '%s'", expected, out)
	}
}
