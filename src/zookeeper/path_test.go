package zookeeper

import (
	"testing"

	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/testutils"
)

func TestPath(t *testing.T) {
	testutils.SetupTestArgs()

	testCases := []struct {
		rootPath string
		input    string
		output   string
	}{
		{"", "/test", "/test"},
		{"/root", "/test", "/root/test"},
		{"/root/root2", "/test", "/root/root2/test"},
	}

	for _, tc := range testCases {
		args.GlobalArgs.ZookeeperPath = tc.rootPath

		if Path(tc.input) != tc.output {
			t.Errorf("Expected path %s, got path %s", tc.output, Path(tc.input))
		}
	}
}
