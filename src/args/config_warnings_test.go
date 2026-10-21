package args

import (
	"bytes"
	"os"
	"testing"

	sdkArgs "github.com/newrelic/infra-integrations-sdk/v3/args"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testArgs(disableAttributes string) ArgumentList {
	return ArgumentList{
		DefaultArgumentList:  sdkArgs.DefaultArgumentList{},
		AutodiscoverStrategy: "zookeeper",
		KafkaVersion:         "1.1.0",
		ZookeeperHosts:       `[{"host":"host1","port":2180}]`,
		NrJmx:                "/usr/bin/nrjmx",
		Producers:            "[]",
		Consumers:            "[]",
		DisableAttributes:    disableAttributes,
		TopicMode:            "None",
		TopicList:            "[]",
		TopicBucket:          "1/1",
		ConsumerGroupRegex:   ".*",
	}
}

func TestParseArgs_WarnsAboutUnknownDisabledAttribute(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	parsed, err := ParseArgs(testArgs(`["clusterID"]`))
	require.NoError(t, err)
	assert.Equal(t, []string{"clusterID"}, parsed.DisableAttributes, "the name is kept; it is only reported")
	assert.Contains(t, buf.String(), "[WARN]")
	assert.Contains(t, buf.String(), "clusterID")

	buf.Reset()
	_, err = ParseArgs(testArgs(`["clusterId","brokerId"]`))
	require.NoError(t, err)
	assert.NotContains(t, buf.String(), "[WARN]", "valid names must not warn")
}

func TestConfigWarnings(t *testing.T) {
	cases := []struct {
		name string
		args ParsedArguments
		want bool // whether a warning is expected
	}{
		{"nothing set", ParsedArguments{}, false},
		{"local-only alone", ParsedArguments{LocalOnlyCollection: true}, false},
		{"topic extended alone", ParsedArguments{CollectTopicExtendedMetrics: true}, false},
		{"topic extended with local-only", ParsedArguments{LocalOnlyCollection: true, CollectTopicExtendedMetrics: true}, true},
		{"topic extended with local-only and force", ParsedArguments{LocalOnlyCollection: true, CollectTopicExtendedMetrics: true, ForceTopicSampleCollection: true}, false},
		{"unknown disabled name", ParsedArguments{DisableAttributes: []string{"clusterID"}}, true},
		{"supported disabled names", ParsedArguments{DisableAttributes: []string{"clusterId", "brokerId"}}, false},
		{"offset mode with cluster metrics", ParsedArguments{ConsumerOffset: true, CollectClusterMetrics: true}, true},
		{"offset mode with broker extended metrics", ParsedArguments{ConsumerOffset: true, CollectBrokerExtendedMetrics: true}, true},
		{"offset mode with topic extended metrics", ParsedArguments{ConsumerOffset: true, CollectTopicExtendedMetrics: true}, true},
		{"offset mode with offset extended metrics", ParsedArguments{ConsumerOffset: true, CollectConsumerOffsetExtendedMetrics: true}, false},
		{"offset extended metrics without offset mode", ParsedArguments{CollectConsumerOffsetExtendedMetrics: true}, true},
		{"cluster metrics outside offset mode", ParsedArguments{CollectClusterMetrics: true}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := c.args
			assert.Equal(t, c.want, len(configWarnings(&a)) > 0, "%v", configWarnings(&a))
		})
	}
}

func TestIDAttributes(t *testing.T) {
	t.Cleanup(func() { GlobalArgs = nil })

	GlobalArgs = &ParsedArguments{ClusterID: "lkc-abc123"}
	assert.Equal(t, "lkc-abc123", ClusterIDAttribute()[0].Value)
	assert.Equal(t, "7", BrokerIDAttribute("7")[0].Value)

	GlobalArgs = &ParsedArguments{ClusterID: ""}
	assert.Empty(t, ClusterIDAttribute(), "an empty cluster id must not be sent")
	assert.Empty(t, BrokerIDAttribute(""), "an empty broker id must not be sent")

	GlobalArgs = &ParsedArguments{ClusterID: "lkc-abc123", DisableAttributes: []string{"clusterId"}}
	assert.Empty(t, ClusterIDAttribute())
	assert.NotEmpty(t, BrokerIDAttribute("7"), "disabling one attribute leaves the other")

	GlobalArgs = &ParsedArguments{ClusterID: "lkc-abc123", DisableAttributes: []string{"brokerId"}}
	assert.NotEmpty(t, ClusterIDAttribute())
	assert.Empty(t, BrokerIDAttribute("7"))
}
