// Package logger contains a singleton logger to be used across
// the entire integration
package logger

import (
	"github.com/newrelic/infra-integrations-sdk/log"
)

var integrationLogger log.Logger

// SetLogger sets the package logger
func SetLogger(newLogger log.Logger) {
	integrationLogger = newLogger
}

// checkSetLogger is a failsafe where if the logger isn't set
// set it to a basic StdErr logger without debugging
func checkSetLogger() {
	if integrationLogger == nil {
		integrationLogger = log.NewStdErr(false)
	}
}

// Debugf is a wrapper for log.Logger.Debugf
func Debugf(format string, args ...interface{}) {
	checkSetLogger()
	integrationLogger.Debugf(format, args)
}

// Warnf is a wrapper for log.Logger.Warnf
func Warnf(format string, args ...interface{}) {
	checkSetLogger()
	integrationLogger.Warnf(format, args)
}

// Infof is a wrapper for log.Logger.Infof
func Infof(format string, args ...interface{}) {
	checkSetLogger()
	integrationLogger.Infof(format, args)
}

// Errorf is a wrapper for log.Logger.Errorf
func Errorf(format string, args ...interface{}) {
	checkSetLogger()
	integrationLogger.Errorf(format, args)
}
