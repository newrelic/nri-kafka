package main

import "github.com/jportasa/infra-integrations-sdk/log"

type saramaLogger struct{}

func (l saramaLogger) Printf(format string, v ...interface{}) {
	log.Debug(format, v...)
}

func (l saramaLogger) Println(v ...interface{}) {
	log.Debug("%v", v)
}

func (l saramaLogger) Print(v ...interface{}) {
	log.Debug("%v", v)
}
