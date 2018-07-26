package main

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/newrelic/infra-integrations-sdk/integration"
)

func TestStartWorkerPool(t *testing.T) {
	var wg sync.WaitGroup
	collectedTopics := []string{"test1", "test2"}
	i, err := integration.New("kafka", "1.0.0")
	if err != nil {
		t.Error(err)
	}

	consumerHosts := startWorkerPool(3, &wg, i, collectedTopics, consumerWorker)

	c := make(chan int)
	go func() {
		wg.Wait()
		c <- 1
	}()

	close(consumerHosts)

	select {
	case <-c:
	case <-time.After(time.Millisecond * 10):
		t.Error("Failed to close wait group in a reasonable amount of time")
	}
}

func TestFeedWorkerPool(t *testing.T) {
	jmxHostChan := make(chan *jmxHost, 10)
	jmxHosts := []*jmxHost{
		{},
		{},
	}
	go feedWorkerPool(jmxHostChan, jmxHosts)

	var retrievedJmxHosts []*jmxHost
	for {
		jmxHost, ok := <-jmxHostChan
		if !ok {
			break
		}
		retrievedJmxHosts = append(retrievedJmxHosts, jmxHost)
	}
	if len(retrievedJmxHosts) != 2 {
		t.Error("Failed to send all JMX hosts down channel")
	}
}

func TestConsumerWorker(t *testing.T) {
	setupJmxTesting()
	consumerChan := make(chan *jmxHost, 10)
	var wg sync.WaitGroup
	i, err := integration.New("kafka", "1.0.0")
	if err != nil {
		t.Error(err)
	}
	logger = i.Logger()
	collectedTopics := []string{"test1", "test2"}

	setupTestArgs()

	wg.Add(1)
	go consumerWorker(consumerChan, &wg, i, collectedTopics)

	newJmx := &jmxHost{
		Name: "test",
	}
	consumerChan <- newJmx
	close(consumerChan)

	wg.Wait()

}

func TestConsumerWorker_JmxOpenFuncErr(t *testing.T) {
	setupJmxTesting()
	jmxOpenFunc = func(hostname, port, username, password string) error { return errors.New("Test") }
	consumerChan := make(chan *jmxHost, 10)
	var wg sync.WaitGroup
	i, err := integration.New("kafka", "1.0.0")
	if err != nil {
		t.Error(err)
	}
	logger = i.Logger()
	collectedTopics := []string{"test1", "test2"}

	setupTestArgs()

	wg.Add(1)
	go consumerWorker(consumerChan, &wg, i, collectedTopics)

	newJmx := &jmxHost{
		Name: "test",
	}
	consumerChan <- newJmx
	close(consumerChan)

	wg.Wait()
}

func TestProducerWorker(t *testing.T) {
	setupJmxTesting()
	producerChan := make(chan *jmxHost, 10)
	var wg sync.WaitGroup
	i, err := integration.New("kafka", "1.0.0")
	if err != nil {
		t.Error(err)
	}
	logger = i.Logger()
	collectedTopics := []string{"test1", "test2"}

	setupTestArgs()

	wg.Add(1)
	go producerWorker(producerChan, &wg, i, collectedTopics)

	newJmx := &jmxHost{
		Name: "test",
	}
	producerChan <- newJmx
	close(producerChan)

	wg.Wait()

}

func TestProducerWorker_JmxOpenFuncErr(t *testing.T) {
	setupJmxTesting()
	jmxOpenFunc = func(hostname, port, username, password string) error { return errors.New("Test") }
	producerChan := make(chan *jmxHost, 10)
	var wg sync.WaitGroup
	i, err := integration.New("kafka", "1.0.0")
	if err != nil {
		t.Error(err)
	}
	logger = i.Logger()
	collectedTopics := []string{"test1", "test2"}

	setupTestArgs()

	wg.Add(1)
	go producerWorker(producerChan, &wg, i, collectedTopics)

	newJmx := &jmxHost{
		Name: "test",
	}
	producerChan <- newJmx
	close(producerChan)

	wg.Wait()
}
