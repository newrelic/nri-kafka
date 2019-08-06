package prodconcollect

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/jmx"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/jmxwrapper"
	"github.com/newrelic/nri-kafka/src/testutils"
)

func TestStartWorkerPool(t *testing.T) {
	var wg sync.WaitGroup
	collectedTopics := []string{"test1", "test2"}
	i, err := integration.New("kafka", "1.0.0")
	if err != nil {
		t.Error(err)
	}

	consumerHosts := StartWorkerPool(3, &wg, i, collectedTopics, ConsumerWorker)

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
	jmxHostChan := make(chan *args.JMXHost, 10)
	jmxHosts := []*args.JMXHost{
		{},
		{},
	}
	go FeedWorkerPool(jmxHostChan, jmxHosts)

	var retrievedJmxHosts []*args.JMXHost
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
	testutils.SetupJmxTesting()
	consumerChan := make(chan *args.JMXHost, 10)
	var wg sync.WaitGroup
	i, err := integration.New("kafka", "1.0.0")
	if err != nil {
		t.Error(err)
	}
	collectedTopics := []string{"test1", "test2"}

	testutils.SetupTestArgs()

	wg.Add(1)
	go ConsumerWorker(consumerChan, &wg, i, collectedTopics)

	newJmx := &args.JMXHost{
		Name: "test",
	}
	consumerChan <- newJmx
	close(consumerChan)

	wg.Wait()

}

func TestConsumerWorker_JmxOpenFuncErr(t *testing.T) {
	testutils.SetupJmxTesting()
	jmxwrapper.JMXOpen = func(hostname, port, username, password string, options ...jmx.Option) error {
		return errors.New("test")
	}
	consumerChan := make(chan *args.JMXHost, 10)
	var wg sync.WaitGroup
	i, err := integration.New("kafka", "1.0.0")
	if err != nil {
		t.Error(err)
	}
	collectedTopics := []string{"test1", "test2"}

	testutils.SetupTestArgs()

	wg.Add(1)
	go ConsumerWorker(consumerChan, &wg, i, collectedTopics)

	newJmx := &args.JMXHost{
		Name: "test",
	}
	consumerChan <- newJmx
	close(consumerChan)

	wg.Wait()
}

func TestProducerWorker(t *testing.T) {
	testutils.SetupJmxTesting()
	producerChan := make(chan *args.JMXHost, 10)
	var wg sync.WaitGroup
	i, err := integration.New("kafka", "1.0.0")
	if err != nil {
		t.Error(err)
	}
	collectedTopics := []string{"test1", "test2"}

	testutils.SetupTestArgs()

	wg.Add(1)
	go ProducerWorker(producerChan, &wg, i, collectedTopics)

	newJmx := &args.JMXHost{
		Name: "test",
	}
	producerChan <- newJmx
	close(producerChan)

	wg.Wait()

}

func TestProducerWorker_JmxOpenFuncErr(t *testing.T) {
	testutils.SetupJmxTesting()
	jmxwrapper.JMXOpen = func(hostname, port, username, password string, options ...jmx.Option) error {
		return errors.New("test")
	}
	producerChan := make(chan *args.JMXHost, 10)
	var wg sync.WaitGroup
	i, err := integration.New("kafka", "1.0.0")
	if err != nil {
		t.Error(err)
	}
	collectedTopics := []string{"test1", "test2"}

	testutils.SetupTestArgs()

	wg.Add(1)
	go ProducerWorker(producerChan, &wg, i, collectedTopics)

	newJmx := &args.JMXHost{
		Name: "test",
	}
	producerChan <- newJmx
	close(producerChan)

	wg.Wait()
}
