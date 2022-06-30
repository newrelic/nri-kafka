package client

import (
	"strings"
	"testing"

	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/connection"
	"github.com/newrelic/nri-kafka/src/connection/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdFromAppInfoMBean(t *testing.T) {
	cases := []struct {
		MBeanName string
		Expected  string
	}{
		{
			MBeanName: "kafka.consumer:type=consumer-fetch-manager-metrics,client-id=my_consumer",
			Expected:  "my_consumer",
		},
		{
			MBeanName: "kafka.consumer:client-id=my_consumer,type=consumer-fetch-manager-metrics",
			Expected:  "my_consumer",
		},
		{
			MBeanName: "kafka.producer:type=producer-metrics,client-id=my_producer",
			Expected:  "my_producer",
		},
		{
			MBeanName: "kafka.producer:type=producer-metrics,no-id-here",
			Expected:  "",
		},
		{
			MBeanName: "id=my_consumer,type=app-info,invalid-mbean=true",
			Expected:  "",
		},
	}

	for _, c := range cases {
		t.Run(c.MBeanName, func(t *testing.T) {
			assert.Equal(t, c.Expected, idFromMBeanWithClientIDField(c.MBeanName))
		})
	}
}

func TestIdsFromMBeanNames(t *testing.T) {
	mBeanNames := []string{"_id1", "_id2", "invalid_id", "_id3"}
	idExtractor := func(name string) string {
		if strings.HasPrefix(name, "_") {
			return strings.TrimLeft(name, "_")
		}
		return ""
	}
	expected := []string{"id1", "id2", "id3"}
	assert.Equal(t, expected, idsFromMBeanNames(mBeanNames, idExtractor))
}

func TestDetectClientIDsConnError(t *testing.T) {
	pattern := "some-pattern"
	conn := &mocks.MockJMXProvider{MBeanNamePattern: "other-pattern-causes-error"}
	_, err := detectClientIDs(pattern, nil, conn)
	assert.Error(t, err)
}

func TestDetectClientIDs(t *testing.T) {
	pattern := "pattern"
	conn := &mocks.MockJMXProvider{MBeanNamePattern: pattern, Names: []string{"a", "b", "c"}}
	ids, err := detectClientIDs(pattern, strings.ToUpper, conn)
	require.NoError(t, err)
	assert.Equal(t, []string{"A", "B", "C"}, ids)
}

func TestGetClientIDs(t *testing.T) {
	pattern := "pattern"
	conn := &mocks.MockJMXProvider{MBeanNamePattern: pattern, Names: []string{"a", "b", "c"}}

	jmxInfo := &args.JMXHost{Name: "D"}
	ids, err := getClientIDS(nil, jmxInfo, conn)
	require.NoError(t, err)
	assert.Equal(t, []string{"D"}, ids, "Expected only the JMXHost.Name when it is defined")

	jmxInfo = &args.JMXHost{}
	ids, _ = getClientIDS([]clientIDExtractInfo{{pattern: pattern, extractor: strings.ToUpper}}, jmxInfo, conn)
	assert.Equal(t, []string{"A", "B", "C"}, ids, "Detect clients should be executed when JMXHost.Name is not defined")

	ids, _ = getClientIDS(
		[]clientIDExtractInfo{
			{pattern: pattern, extractor: func(string) string { return "" }},
			{pattern: pattern, extractor: strings.ToUpper},
		},
		jmxInfo, conn,
	)
	assert.Equal(t, []string{"A", "B", "C"}, ids, "Fallback should be called when first extract info does not work")
}

func TestDetectConsumerAndProducerIDs(t *testing.T) {
	cases := []struct {
		Name     string
		Conn     *mocks.MockJMXProvider
		Detector func(jmxInfo *args.JMXHost, conn connection.JMXConnection) ([]string, error)
		Expected []string
	}{
		{
			Name: "consumer app-info pattern", // app-info fallback
			Conn: &mocks.MockJMXProvider{
				MBeanNamePattern: consumerAppInfoPattern,
				Names:            []string{"kafka.consumer:type=app-info,id=consumer-id"},
			},
			Detector: detectConsumerIDs,
			Expected: []string{"consumer-id"},
		},
		{
			Name: "consumer metrics pattern",
			Conn: &mocks.MockJMXProvider{
				MBeanNamePattern: consumerMetricsPattern,
				Names:            []string{"kafka.consumer:type=consumer-fetch-manager-metrics,client-id=consumer-id"},
			},
			Detector: detectConsumerIDs,
			Expected: []string{"consumer-id"},
		},
		{
			Name: "producer app-info pattern", // app-info fallback
			Conn: &mocks.MockJMXProvider{
				MBeanNamePattern: producerAppInfoPattern,
				Names:            []string{"kafka.producer:type=app-info,id=my-id"},
			},
			Detector: detectProducerIDs,
			Expected: []string{"my-id"},
		},
		{
			Name: "producer metrics pattern",
			Conn: &mocks.MockJMXProvider{
				MBeanNamePattern: producerMetricsPattern,
				Names:            []string{"kafka.producer:type=producer-metrics,client-id=producer-id"},
			},
			Detector: detectProducerIDs,
			Expected: []string{"producer-id"},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			ids, err := c.Detector(&args.JMXHost{}, c.Conn)
			require.NoError(t, err)
			assert.Equal(t, c.Expected, ids)
		})
	}
}
