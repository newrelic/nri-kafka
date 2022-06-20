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

	"github.com/newrelic/infra-integrations-sdk/integration"
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

func TestCollectConsumerMetricsNameProvided(t *testing.T) {
	i, err := integration.New(t.Name(), "1.0.0")
	require.NoError(t, err)
	connProvider := mocks.NewEmptyMockJMXProvider()
	connProvider.Names = []string{
		"kafka.consumer:type=consumer-fetch-manager-metrics,client-id=consumer-1",
		"kafka.consumer:type=consumer-fetch-manager-metrics,client-id=consumer-2",
	}
	clientID := "consumer-1"
	jmxInfo := &args.JMXHost{Name: clientID}
	testutils.SetupTestArgs()
	CollectConsumerMetrics(i, jmxInfo, connProvider)
	require.Len(t, i.Entities, 1, "only metrics from the provided consumer should be fetched")
	assert.Equal(t, clientID, i.Entities[0].Metadata.Name)
}

func TestCollectProducerMetricsCreateEntity(t *testing.T) {
	i, err := integration.New(t.Name(), "1.0.0")
	require.NoError(t, err)
	connProvider := mocks.NewEmptyMockJMXProvider()
	connProvider.Names = []string{
		"kafka.producer:type=producer-metrics,client-id=producer-1",
		"kafka.producer:type=producer-metrics,client-id=producer-2",
	}
	clientID := "producer-1"
	jmxInfo := &args.JMXHost{Name: clientID}
	testutils.SetupTestArgs()
	CollectProducerMetrics(i, jmxInfo, connProvider)
	require.Len(t, i.Entities, 1, "only metrics from the provided producer should be fetched")
	assert.Equal(t, clientID, i.Entities[0].Metadata.Name)
}

func TestCollectConsumerMetricsNoNameProvided(t *testing.T) {
	i, err := integration.New(t.Name(), "1.0.0")
	require.NoError(t, err)
	connProvider := mocks.NewEmptyMockJMXProvider()
	connProvider.Names = []string{
		"kafka.consumer:type=consumer-fetch-manager-metrics,client-id=consumer-1",
		"kafka.consumer:type=consumer-fetch-manager-metrics,client-id=consumer-2",
	}
	jmxInfo := &args.JMXHost{}
	testutils.SetupTestArgs()
	CollectConsumerMetrics(i, jmxInfo, connProvider)
	require.Len(t, i.Entities, 2)
	var entityNames []string
	for _, entity := range i.Entities {
		entityNames = append(entityNames, entity.Metadata.Name)
	}
	assert.ElementsMatch(t, []string{"consumer-1", "consumer-2"}, entityNames)
}

func TestCollectProducerMetricsNoNameProvided(t *testing.T) {
	i, err := integration.New(t.Name(), "1.0.0")
	require.NoError(t, err)
	connProvider := mocks.NewEmptyMockJMXProvider()
	connProvider.Names = []string{
		"kafka.producer:type=producer-metrics,client-id=producer-1",
		"kafka.producer:type=producer-metrics,client-id=producer-2",
	}
	jmxInfo := &args.JMXHost{}
	testutils.SetupTestArgs()
	CollectProducerMetrics(i, jmxInfo, connProvider)
	require.Len(t, i.Entities, 2)
	var entityNames []string
	for _, entity := range i.Entities {
		entityNames = append(entityNames, entity.Metadata.Name)
	}
	assert.ElementsMatch(t, []string{"producer-1", "producer-2"}, entityNames)
}
