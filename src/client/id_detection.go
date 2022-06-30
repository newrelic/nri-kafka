package client

import (
	"strings"

	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/connection"
)

const (
	consumerAppInfoPattern = "kafka.consumer:type=app-info,id=*"
	consumerMetricsPattern = "kafka.consumer:type=consumer-fetch-manager-metrics,client-id=*"

	producerAppInfoPattern = "kafka.producer:type=app-info,id=*"
	producerMetricsPattern = "kafka.producer:type=producer-metrics,client-id=*"
)

// idFromMBeanNameFn defines a function to extract the identifier from an MBean name.
type idFromMBeanNameFn func(string) string

type clientIDExtractInfo struct {
	pattern   string
	extractor idFromMBeanNameFn
}

func detectConsumerIDs(jmxInfo *args.JMXHost, conn connection.JMXConnection) ([]string, error) {
	return getClientIDS(
		[]clientIDExtractInfo{
			{pattern: consumerMetricsPattern, extractor: idFromMBeanWithClientIDField},
			{pattern: consumerAppInfoPattern, extractor: idFromAppInfo},
		},
		jmxInfo,
		conn,
	)
}

func detectProducerIDs(jmxInfo *args.JMXHost, conn connection.JMXConnection) ([]string, error) {
	return getClientIDS(
		[]clientIDExtractInfo{
			{pattern: producerMetricsPattern, extractor: idFromMBeanWithClientIDField},
			{pattern: producerAppInfoPattern, extractor: idFromAppInfo},
		},
		jmxInfo,
		conn,
	)
}

// getClientIDs tries to obtain clientIDs from JMX connection using each extractInfo entry until it success or items are over.
// this allows implementing a primary extractor and one or many fallbacks.
func getClientIDS(extractInfo []clientIDExtractInfo, jmxInfo *args.JMXHost, conn connection.JMXConnection) ([]string, error) {
	if jmxInfo.Name != "" {
		return []string{jmxInfo.Name}, nil
	}
	var err error
	var names []string
	for _, info := range extractInfo {
		names, err = detectClientIDs(info.pattern, info.extractor, conn)
		if err == nil && len(names) > 0 {
			return names, nil
		}
	}
	return names, err
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

// idFromAppInfo Gets the identifier given a type=app-info MBean name. Example "name:type=app-info,id=my-id"
func idFromAppInfo(mBeanName string) string {
	return idFromMBean(mBeanName, "id")
}

// idFromMBeanWithClientIDField Gets the identifier given a MBean name including the client-id field. Example: "name:type=producer-metrics,client-id=my-id"
func idFromMBeanWithClientIDField(mBeanName string) string {
	return idFromMBean(mBeanName, "client-id")
}

func idFromMBean(mBeanName string, idField string) string {
	_, info, valid := strings.Cut(mBeanName, ":")
	if !valid {
		return ""
	}
	for _, field := range strings.Split(info, ",") {
		if _, id, isIDField := strings.Cut(field, idField+"="); isIDField {
			return id
		}
	}
	return ""
}
