package logger

import (
	"testing"

	"github.com/newrelic/infra-integrations-sdk/log"
)

func TestSetLogger(t *testing.T) {
	stdLogger := log.NewStdErr(false)

	SetLogger(stdLogger)

	if stdLogger != integrationLogger {
		t.Errorf("Expected %+v got %+v", stdLogger, integrationLogger)
	}
}

func TestCallsSet(t *testing.T) {
	testCases := []struct {
		name    string
		logFunc func(string, ...interface{})
	}{
		{
			"Debugf",
			Debugf,
		},
		{
			"Warnf",
			Warnf,
		},
		{
			"Infof",
			Infof,
		},
		{
			"Errorf",
			Errorf,
		},
	}

	for _, tc := range testCases {
		integrationLogger = nil
		tc.logFunc("")
		if integrationLogger == nil {
			t.Errorf("Test Case %s Failed to set logger", tc.name)
		}
	}
}
