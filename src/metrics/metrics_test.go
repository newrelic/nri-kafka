package metrics

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/connection/mocks"
	"github.com/newrelic/nrjmx/gojmx"

	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/nri-kafka/src/testutils"
)

var (
	errTest = errors.New("this is an error")
)

func TestGetBrokerMetrics_LeaderCount(t *testing.T) {
	testutils.SetupTestArgs()

	mockResponse := &mocks.MockJMXResponse{
		Result: []*gojmx.AttributeResponse{
			{
				Name:         "kafka.server:type=ReplicaManager,name=LeaderCount,attr=Value",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     28,
			},
		},
	}

	mockJMXProvider := &mocks.MockJMXProvider{Response: mockResponse}

	i, _ := integration.New("test", "1.0.0")
	e, _ := i.Entity("leaderCountEntity", "leaderCountNamespace")
	m := e.NewMetricSet("testMetrics")

	GetBrokerMetrics(m, mockJMXProvider)

	if got := m.Metrics["broker.leaderCount"]; got != float64(28) {
		t.Errorf("expected broker.leaderCount = 28, got %v", got)
	}
}

func TestGetBrokerMetrics_IsActiveController(t *testing.T) {
	testutils.SetupTestArgs()

	mockResponse := &mocks.MockJMXResponse{
		Result: []*gojmx.AttributeResponse{
			{
				Name:         "kafka.controller:type=KafkaController,name=ActiveControllerCount,attr=Value",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     1,
			},
		},
	}

	mockJMXProvider := &mocks.MockJMXProvider{Response: mockResponse}

	i, _ := integration.New("test", "1.0.0")
	e, _ := i.Entity("isActiveControllerEntity", "isActiveControllerNamespace")
	m := e.NewMetricSet("testMetrics")

	GetBrokerMetrics(m, mockJMXProvider)

	if got := m.Metrics["broker.isActiveController"]; got != float64(1) {
		t.Errorf("expected broker.isActiveController = 1, got %v", got)
	}
}

func TestGetBrokerMetrics(t *testing.T) {
	expected := map[string]interface{}{
		"request.avgTimeFetch": float64(24),
		"event_type":           "testMetrics",
	}

	mockResponse := &mocks.MockJMXResponse{
		Err: nil,
		Result: []*gojmx.AttributeResponse{
			{
				Name:         "kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Fetch,attr=Mean",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     24,
			},
		},
	}

	mockJMXProvider := &mocks.MockJMXProvider{
		Response: mockResponse,
	}

	testutils.SetupTestArgs()

	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	e, err := i.Entity("testEntity", "testNamespace")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	m := e.NewMetricSet("testMetrics")

	GetBrokerMetrics(m, mockJMXProvider)

	if !reflect.DeepEqual(expected, m.Metrics) {
		t.Errorf("Expected %+v got %+v", expected, m.Metrics)
	}
}

func TestGetConsumerMetrics(t *testing.T) {
	expected := map[string]interface{}{
		"consumer.maxLag": float64(24),
		"event_type":      "testMetrics",
	}

	consumerName := "consumer"

	mockResponse := &mocks.MockJMXResponse{
		Err: nil,
		Result: []*gojmx.AttributeResponse{
			{
				Name:         "kafka.consumer:type=consumer-fetch-manager-metrics,client-id=" + consumerName + ",attr=records-lag-max",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     24,
			},
		},
	}

	mockJMXProvider := &mocks.MockJMXProvider{
		Response: mockResponse,
	}

	testutils.SetupTestArgs()

	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	e, err := i.Entity("testEntity", "testNamespace")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	m := e.NewMetricSet("testMetrics")

	GetConsumerMetrics(consumerName, m, mockJMXProvider)

	if !reflect.DeepEqual(expected, m.Metrics) {
		t.Errorf("Expected %+v got %+v", expected, m.Metrics)
	}
}

func TestGetConsumerMetrics_RebalanceChurn(t *testing.T) {
	testutils.SetupTestArgs()

	consumerName := "consumer"

	mockResponse := &mocks.MockJMXResponse{
		Result: []*gojmx.AttributeResponse{
			{
				Name:         "kafka.consumer:type=consumer-coordinator-metrics,client-id=" + consumerName + ",attr=rebalance-total",
				ResponseType: gojmx.ResponseTypeDouble,
				DoubleValue:  1,
			},
			{
				Name:         "kafka.consumer:type=consumer-coordinator-metrics,client-id=" + consumerName + ",attr=failed-rebalance-total",
				ResponseType: gojmx.ResponseTypeDouble,
				DoubleValue:  1,
			},
		},
	}

	mockJMXProvider := &mocks.MockJMXProvider{Response: mockResponse}

	i, _ := integration.New("test", "1.0.0")
	e, _ := i.Entity("rebalanceEntity", "rebalanceNamespace")
	m := e.NewMetricSet("testMetrics")

	GetConsumerMetrics(consumerName, m, mockJMXProvider)

	if _, ok := m.Metrics["consumer.rebalanceTotal"]; !ok {
		t.Error("expected consumer.rebalanceTotal to be collected")
	}
	if _, ok := m.Metrics["consumer.failedRebalanceTotal"]; !ok {
		t.Error("expected consumer.failedRebalanceTotal to be collected")
	}
}

func TestGetConsumerMetrics_HeartbeatAndCommitHealth(t *testing.T) {
	testutils.SetupTestArgs()

	consumerName := "consumer"

	mockResponse := &mocks.MockJMXResponse{
		Result: []*gojmx.AttributeResponse{
			{
				Name:         "kafka.consumer:type=consumer-coordinator-metrics,client-id=" + consumerName + ",attr=heartbeat-rate",
				ResponseType: gojmx.ResponseTypeDouble,
				DoubleValue:  0.1,
			},
			{
				Name:         "kafka.consumer:type=consumer-coordinator-metrics,client-id=" + consumerName + ",attr=last-heartbeat-seconds-ago",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     2,
			},
			{
				Name:         "kafka.consumer:type=consumer-coordinator-metrics,client-id=" + consumerName + ",attr=assigned-partitions",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     2,
			},
			{
				Name:         "kafka.consumer:type=consumer-coordinator-metrics,client-id=" + consumerName + ",attr=commit-rate",
				ResponseType: gojmx.ResponseTypeDouble,
				DoubleValue:  0.05,
			},
		},
	}

	mockJMXProvider := &mocks.MockJMXProvider{Response: mockResponse}

	i, _ := integration.New("test", "1.0.0")
	e, _ := i.Entity("heartbeatEntity", "heartbeatNamespace")
	m := e.NewMetricSet("testMetrics")

	GetConsumerMetrics(consumerName, m, mockJMXProvider)

	if got := m.Metrics["consumer.heartbeatRate"]; got != float64(0.1) {
		t.Errorf("expected consumer.heartbeatRate = 0.1, got %v", got)
	}
	if got := m.Metrics["consumer.lastHeartbeatSecondsAgo"]; got != float64(2) {
		t.Errorf("expected consumer.lastHeartbeatSecondsAgo = 2, got %v", got)
	}
	if got := m.Metrics["consumer.assignedPartitions"]; got != float64(2) {
		t.Errorf("expected consumer.assignedPartitions = 2, got %v", got)
	}
	if got := m.Metrics["consumer.commitRate"]; got != float64(0.05) {
		t.Errorf("expected consumer.commitRate = 0.05, got %v", got)
	}
}

func TestGetConsumerMetrics_FetchLatency(t *testing.T) {
	testutils.SetupTestArgs()

	consumerName := "consumer"

	mockResponse := &mocks.MockJMXResponse{
		Result: []*gojmx.AttributeResponse{
			{
				Name:         "kafka.consumer:type=consumer-fetch-manager-metrics,client-id=" + consumerName + ",attr=fetch-latency-avg",
				ResponseType: gojmx.ResponseTypeDouble,
				DoubleValue:  154.8,
			},
			{
				Name:         "kafka.consumer:type=consumer-fetch-manager-metrics,client-id=" + consumerName + ",attr=fetch-throttle-time-max",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     0,
			},
		},
	}

	mockJMXProvider := &mocks.MockJMXProvider{Response: mockResponse}

	i, _ := integration.New("test", "1.0.0")
	e, _ := i.Entity("fetchLatencyEntity", "fetchLatencyNamespace")
	m := e.NewMetricSet("testMetrics")

	GetConsumerMetrics(consumerName, m, mockJMXProvider)

	if got := m.Metrics["consumer.fetchLatencyAvg"]; got != float64(154.8) {
		t.Errorf("expected consumer.fetchLatencyAvg = 154.8, got %v", got)
	}
	if _, ok := m.Metrics["consumer.fetchThrottleTimeMax"]; !ok {
		t.Error("expected consumer.fetchThrottleTimeMax to be collected")
	}
}

func TestGetProducerMetrics(t *testing.T) {
	expected := map[string]interface{}{
		"producer.ageMetadataUsedInMilliseconds": float64(24),
		"event_type":                             "testMetrics",
	}

	producerName := "producer"

	mockResponse := &mocks.MockJMXResponse{
		Err: nil,
		Result: []*gojmx.AttributeResponse{
			{
				Name:         "kafka.producer:type=producer-metrics,client-id=" + producerName + ",attr=metadata-age",
				ResponseType: gojmx.ResponseTypeInt,
				IntValue:     24,
			},
		},
	}

	mockJMXProvider := &mocks.MockJMXProvider{
		Response: mockResponse,
	}

	testutils.SetupTestArgs()

	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	e, err := i.Entity("testEntity", "testNamespace")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	m := e.NewMetricSet("testMetrics")

	GetProducerMetrics(producerName, m, mockJMXProvider)

	if !reflect.DeepEqual(expected, m.Metrics) {
		t.Errorf("Expected %+v got %+v", expected, m.Metrics)
	}
}

func TestGetProducerMetrics_ErrorRetryRate(t *testing.T) {
	testutils.SetupTestArgs()

	producerName := "producer"

	mockResponse := &mocks.MockJMXResponse{
		Result: []*gojmx.AttributeResponse{
			{
				Name:         "kafka.producer:type=producer-metrics,client-id=" + producerName + ",attr=record-error-rate",
				ResponseType: gojmx.ResponseTypeDouble,
				DoubleValue:  0.0,
			},
			{
				Name:         "kafka.producer:type=producer-metrics,client-id=" + producerName + ",attr=record-retry-rate",
				ResponseType: gojmx.ResponseTypeDouble,
				DoubleValue:  0.02,
			},
		},
	}

	mockJMXProvider := &mocks.MockJMXProvider{Response: mockResponse}

	i, _ := integration.New("test", "1.0.0")
	e, _ := i.Entity("producerErrorEntity", "producerErrorNamespace")
	m := e.NewMetricSet("testMetrics")

	GetProducerMetrics(producerName, m, mockJMXProvider)

	if got := m.Metrics["producer.recordErrorRate"]; got != float64(0.0) {
		t.Errorf("expected producer.recordErrorRate = 0.0, got %v", got)
	}
	if got := m.Metrics["producer.recordRetryRate"]; got != float64(0.02) {
		t.Errorf("expected producer.recordRetryRate = 0.02, got %v", got)
	}
}

// beanRecordingJMXProvider wraps mocks.MockJMXProvider purely to record which MBean pattern
// getAllTopicsFromJMX actually queries. Deliberately not using MockJMXProvider's
// MBeanNamePattern mismatch check for this: that path returns a plain (non-JMX) error, which
// getAllTopicsFromJMX's error handling escalates to os.Exit(1) for any unrecognized error
// type - fine today since the fix is correct and no mismatch occurs, but it would make a
// future regression here crash the test binary instead of failing as a normal assertion.
type beanRecordingJMXProvider struct {
	*mocks.MockJMXProvider
	queriedBeans []string
}

func (r *beanRecordingJMXProvider) QueryMBeanAttributes(mBeanNamePattern string) ([]*gojmx.AttributeResponse, error) {
	r.queriedBeans = append(r.queriedBeans, mBeanNamePattern)
	return r.MockJMXProvider.QueryMBeanAttributes(mBeanNamePattern)
}

func TestCollectTopicSubMetrics_ConsumerQueriesConsumerBean(t *testing.T) {
	testutils.SetupTestArgs()
	args.GlobalArgs.TopicMode = "all"

	consumerName := "myconsumer"

	// Before the fix, getAllTopicsFromJMX always queried the producer bean regardless of
	// caller - a consumer-only client has no kafka.producer:... MBeans, so this returned zero
	// topics and ConsumerTopicMetricDefs never populated.
	mockResponse := &mocks.MockJMXResponse{
		Result: []*gojmx.AttributeResponse{
			{
				Name:         "kafka.consumer:type=consumer-fetch-manager-metrics,client-id=" + consumerName + ",topic=orders,attr=fetch-size-avg",
				ResponseType: gojmx.ResponseTypeDouble,
				DoubleValue:  1024,
			},
		},
	}

	provider := &beanRecordingJMXProvider{MockJMXProvider: &mocks.MockJMXProvider{Response: mockResponse}}

	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Fatalf("Unexpected error %s", err.Error())
	}

	consumerEntity, err := i.Entity(consumerName, "ka-consumer")
	if err != nil {
		t.Fatalf("Unexpected error %s", err.Error())
	}

	CollectTopicSubMetrics(consumerEntity, ConsumerTopicMetricDefs, ApplyConsumerTopicName, provider)

	// Discovery queries the wildcarded bean once, then CollectMetricDefinitions queries the
	// same wildcarded bean again per topic found (it isn't narrowed to a specific topic) - so
	// every query should be this consumer's own bean, never the producer's.
	wantBean := "kafka.consumer:type=consumer-fetch-manager-metrics,client-id=" + consumerName + ",topic=*"
	if len(provider.queriedBeans) == 0 {
		t.Fatal("expected at least one JMX query, got none")
	}
	for _, got := range provider.queriedBeans {
		if got != wantBean {
			t.Errorf("expected topic discovery/collection to query %q, got %q", wantBean, got)
		}
	}

	found := false
	for _, ms := range consumerEntity.Metrics {
		if ms.Metrics["topic"] == "orders" && ms.Metrics["consumer.avgFetchSizeInBytes"] == float64(1024) {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a KafkaConsumerSample for topic 'orders' with consumer.avgFetchSizeInBytes = 1024, got %+v", consumerEntity.Metrics)
	}
}

func TestCollectTopicSubMetrics_ProducerQueriesProducerBean(t *testing.T) {
	testutils.SetupTestArgs()
	args.GlobalArgs.TopicMode = "all"

	producerName := "myproducer"

	mockResponse := &mocks.MockJMXResponse{
		Result: []*gojmx.AttributeResponse{
			{
				Name:         "kafka.producer:type=producer-topic-metrics,client-id=" + producerName + ",topic=orders,attr=record-send-rate",
				ResponseType: gojmx.ResponseTypeDouble,
				DoubleValue:  5,
			},
		},
	}

	provider := &beanRecordingJMXProvider{MockJMXProvider: &mocks.MockJMXProvider{Response: mockResponse}}

	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Fatalf("Unexpected error %s", err.Error())
	}

	producerEntity, err := i.Entity(producerName, "ka-producer")
	if err != nil {
		t.Fatalf("Unexpected error %s", err.Error())
	}

	CollectTopicSubMetrics(producerEntity, ProducerTopicMetricDefs, ApplyProducerTopicName, provider)

	wantBean := "kafka.producer:type=producer-topic-metrics,client-id=" + producerName + ",topic=*"
	if len(provider.queriedBeans) == 0 {
		t.Fatal("expected at least one JMX query, got none")
	}
	for _, got := range provider.queriedBeans {
		if got != wantBean {
			t.Errorf("expected topic discovery/collection to query %q, got %q", wantBean, got)
		}
	}

	found := false
	for _, ms := range producerEntity.Metrics {
		if ms.Metrics["topic"] == "orders" && ms.Metrics["producer.avgRecordsSentPerTopicPerSecond"] == float64(5) {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a KafkaProducerSample for topic 'orders' with producer.avgRecordsSentPerTopicPerSecond = 5, got %+v", producerEntity.Metrics)
	}
}

func TestCollectMetricDefinitions_QueryError(t *testing.T) {
	testutils.SetupTestArgs()

	mockResponse := &mocks.MockJMXResponse{
		Err: errTest,
	}

	mockJMXProvider := &mocks.MockJMXProvider{
		Response: mockResponse,
	}

	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	e, err := i.Entity("testEntity", "testNamespace")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	m := e.NewMetricSet("testMetrics")

	CollectMetricDefinitions(m, brokerMetricDefs, nil, mockJMXProvider)

	if len(m.Metrics) != 1 {
		t.Error("Metrics where inserted even with a bad query")
	}
}

func TestCollectMetricDefinitions_MetricError(t *testing.T) {
	testutils.SetupTestArgs()
	expected := map[string]interface{}{
		"event_type": "testMetrics",
	}

	mockResponse := &mocks.MockJMXResponse{
		Err: nil,
		Result: []*gojmx.AttributeResponse{
			{
				Name:         "kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Fetch,attr=Mean",
				ResponseType: gojmx.ResponseTypeString,
				StringValue:  "stuff",
			},
		},
	}

	mockJMXProvider := &mocks.MockJMXProvider{
		Response: mockResponse,
	}

	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	e, err := i.Entity("testEntity", "testNamespace")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	m := e.NewMetricSet("testMetrics")

	CollectMetricDefinitions(m, brokerMetricDefs, nil, mockJMXProvider)

	if !reflect.DeepEqual(expected, m.Metrics) {
		t.Errorf("Expected %+v got %+v", expected, m.Metrics)
	}
}

func TestCollectMetricDefinitions_BeanModifier(t *testing.T) {
	testutils.SetupTestArgs()
	testMetricSet := []*JMXMetricSet{
		{
			MBean:        "kafka.network:replace=%REPLACE_ME%",
			MetricPrefix: "kafka.network:replace=%REPLACE_ME%,",
			MetricDefs: []*MetricDefinition{
				{
					Name:       "my.metric",
					SourceType: metric.GAUGE,
					JMXAttr:    "attr=Metric",
				},
			},
		},
	}

	expectedBean := "kafka.network:replace=Replaced"

	mockResponse := &mocks.MockJMXResponse{
		Err: nil,
		Result: []*gojmx.AttributeResponse{
			{
				Name:         "kafka.network:replace=Replaced,attr=Metric",
				ResponseType: gojmx.ResponseTypeDouble,
				DoubleValue:  float64(24),
			},
		},
	}

	mockJMXProvider := &mocks.MockJMXProvider{
		Response:         mockResponse,
		MBeanNamePattern: expectedBean,
	}

	expected := map[string]interface{}{
		"my.metric":  float64(24),
		"event_type": "testMetrics",
	}

	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	e, err := i.Entity("testEntity", "testNamespace")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	m := e.NewMetricSet("testMetrics")

	renameFunc := func(replaceName string) func(string) string {
		return func(bean string) string {
			return strings.Replace(bean, "%REPLACE_ME%", replaceName, -1)
		}
	}

	CollectMetricDefinitions(m, testMetricSet, renameFunc("Replaced"), mockJMXProvider)

	if !reflect.DeepEqual(expected, m.Metrics) {
		t.Errorf("Expected %+v got %+v", expected, m.Metrics)
	}
}
