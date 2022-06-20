package client

import (
	"strings"

	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/connection"
)

const (
	consumerDetectionPattern = "kafka.consumer:type=consumer-fetch-manager-metrics,client-id=*"
	producerDetectionPattern = "kafka.producer:type=producer-metrics,client-id=*"
)

// idFromMBeanNameFn defines a function to extract the identifier from an MBean name.
type idFromMBeanNameFn func(string) string

func getClientIDS(jmxInfo *args.JMXHost, mBeanPattern string, idExtractor idFromMBeanNameFn, conn connection.JMXConnection) ([]string, error) {
	if jmxInfo.Name != "" {
		return []string{jmxInfo.Name}, nil
	}
	return detectClientIDs(mBeanPattern, idExtractor, conn)
}

func detectClientIDs(pattern string, idExtractor idFromMBeanNameFn, conn connection.JMXConnection) ([]string, error) {
	mBeanNames, err := conn.QueryMBeanNames(pattern)
	if err != nil {
		return nil, err
	}
	return idsFromMBeanNames(mBeanNames, idExtractor), nil
}

func idsFromMBeanNames(mBeanNames []string, idExtractor idFromMBeanNameFn) []string {
	ids := []string{}
	for _, mBeanName := range mBeanNames {
		if id := idExtractor(mBeanName); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// idFromMBeanWithClientIDField Gets the identifier given a type=app-info MBean name. Example: "name:type=app-info,client-id=my-id"
func idFromMBeanWithClientIDField(mBeanName string) string {
	_, info, valid := strings.Cut(mBeanName, ":")
	if !valid {
		return ""
	}
	for _, field := range strings.Split(info, ",") {
		if _, id, isIDField := strings.Cut(field, "client-id="); isIDField {
			return id
		}
	}
	return ""
}
