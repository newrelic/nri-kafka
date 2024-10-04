package client

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/newrelic/nri-kafka/src/connection/mocks"
	"github.com/newrelic/nrjmx/gojmx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/testutils"
)

var (
	errTest = errors.New("test")
)

func TestStartWorkerPool(t *testing.T) {
	var wg sync.WaitGroup
	i, err := integration.New("kafka", "1.0.0")
	if err != nil {
		t.Error(err)
	}

	consumerHosts := StartWorkerPool(3, &wg, i, Worker(CollectConsumerMetrics), nil)

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
	consumerChan := make(chan *args.JMXHost, 10)
	var wg sync.WaitGroup
	i, err := integration.New("kafka", "1.0.0")
	if err != nil {
		t.Error(err)
	}

	testutils.SetupTestArgs()

	mockResponse := &mocks.MockJMXResponse{
		Err:    nil,
		Result: []*gojmx.AttributeResponse{},
	}

	mockJMXProvider := &mocks.MockJMXProvider{
		Response: mockResponse,
	}

	wg.Add(1)
	consumerWorker := Worker(CollectConsumerMetrics)
	go consumerWorker(consumerChan, &wg, i, mockJMXProvider)

	newJmx := &args.JMXHost{
		Name: "test",
	}
	consumerChan <- newJmx
	close(consumerChan)

	wg.Wait()

}

func TestConsumerWorker_JmxOpenFuncErr(t *testing.T) {
	mockResponse := &mocks.MockJMXResponse{
		Err: errTest,
	}

	mockJMXProvider := &mocks.MockJMXProvider{
		Response: mockResponse,
	}

	consumerChan := make(chan *args.JMXHost, 10)
	var wg sync.WaitGroup
	i, err := integration.New("kafka", "1.0.0")
	if err != nil {
		t.Error(err)
	}

	testutils.SetupTestArgs()

	wg.Add(1)
	consumerWorker := Worker(CollectConsumerMetrics)
	go consumerWorker(consumerChan, &wg, i, mockJMXProvider)

	newJmx := &args.JMXHost{
		Name: "test",
	}
	consumerChan <- newJmx
	close(consumerChan)

	wg.Wait()
}

func TestProducerWorker(t *testing.T) {
	producerChan := make(chan *args.JMXHost, 10)
	var wg sync.WaitGroup
	i, err := integration.New("kafka", "1.0.0")
	if err != nil {
		t.Error(err)
	}

	testutils.SetupTestArgs()

	wg.Add(1)
	producerWorker := Worker(CollectProducerMetrics)
	go producerWorker(producerChan, &wg, i, mocks.NewEmptyMockJMXProvider())

	newJmx := &args.JMXHost{
		Name: "test",
	}
	producerChan <- newJmx
	close(producerChan)

	wg.Wait()

}

func TestProducerWorker_JmxOpenFuncErr(t *testing.T) {
	mockResponse := &mocks.MockJMXResponse{
		Err: errTest,
	}

	mockJMXProvider := &mocks.MockJMXProvider{
		Response: mockResponse,
	}

	producerChan := make(chan *args.JMXHost, 10)
	var wg sync.WaitGroup
	i, err := integration.New("kafka", "1.0.0")
	if err != nil {
		t.Error(err)
	}

	testutils.SetupTestArgs()

	wg.Add(1)
	producerWorker := Worker(CollectProducerMetrics)
	go producerWorker(producerChan, &wg, i, mockJMXProvider)

	newJmx := &args.JMXHost{
		Name: "test",
	}
	producerChan <- newJmx
	close(producerChan)

	wg.Wait()
}

func TestProducerConsumerEntitiesCreation(t *testing.T) {
	cases := []struct {
		Name                string
		JMXNames            []string
		JMXInfo             *args.JMXHost
		CollectionFn        CollectionFn
		ExpectedEntityNames []string
	}{
		{
			Name: "Collect consumer metrics with name provided",
			JMXNames: []string{
				"kafka.consumer:type=consumer-fetch-manager-metrics,client-id=consumer-1",
				"kafka.consumer:type=consumer-fetch-manager-metrics,client-id=consumer-2",
			},
			JMXInfo:             &args.JMXHost{Name: "consumer-1"},
			CollectionFn:        CollectConsumerMetrics,
			ExpectedEntityNames: []string{"consumer-1"},
		},
		{
			Name: "Collect consumer metrics with no name provided",
			JMXNames: []string{
				"kafka.consumer:type=consumer-fetch-manager-metrics,client-id=consumer-1",
				"kafka.consumer:type=consumer-fetch-manager-metrics,client-id=consumer-2",
			},
			JMXInfo:             &args.JMXHost{},
			CollectionFn:        CollectConsumerMetrics,
			ExpectedEntityNames: []string{"consumer-1", "consumer-2"},
		},
		{
			Name: "Collect producer metrics with name provided",
			JMXNames: []string{
				"kafka.producer:type=producer-metrics,client-id=producer-1",
				"kafka.producer:type=producer-metrics,client-id=producer-2",
			},
			JMXInfo:             &args.JMXHost{Name: "producer-1"},
			CollectionFn:        CollectProducerMetrics,
			ExpectedEntityNames: []string{"producer-1"},
		},
		{
			Name: "Collect producer metrics with no name provided",
			JMXNames: []string{
				"kafka.producer:type=producer-metrics,client-id=producer-1",
				"kafka.producer:type=producer-metrics,client-id=producer-2",
			},
			JMXInfo:             &args.JMXHost{},
			CollectionFn:        CollectProducerMetrics,
			ExpectedEntityNames: []string{"producer-1", "producer-2"},
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			// setup integration
			i, err := integration.New(c.Name, "1.0.0")
			require.NoError(t, err)
			testutils.SetupTestArgs()
			connProvider := mocks.NewEmptyMockJMXProvider()
			connProvider.Names = c.JMXNames
			// run collection
			c.CollectionFn(i, c.JMXInfo, connProvider)
			var entityNames []string
			for _, entity := range i.Entities {
				entityNames = append(entityNames, entity.Metadata.Name)
			}
			assert.ElementsMatch(t, c.ExpectedEntityNames, entityNames)
		})
	}
}
