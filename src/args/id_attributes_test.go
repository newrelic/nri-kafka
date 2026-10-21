package args

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
