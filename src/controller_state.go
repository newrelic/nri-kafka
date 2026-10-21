package main

import (
	"strings"

	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/connection"
	"github.com/newrelic/nrjmx/gojmx"
)

const activeControllerMBean = "kafka.controller:type=KafkaController,name=ActiveControllerCount"

type controllerState struct {
	count      int
	answered   int
	absent     int                // brokers without the MBean
	known      bool               // false when a count of 0 only reflects unreadable brokers
	controller *connection.Broker // a listed broker that is the active controller, if any
}

type queryResult int

const (
	queryFailed queryResult = iota
	queryAnswered
	queryAbsent
)

// localOnlyCluster: LOCAL_ONLY_COLLECTION narrows the broker list to one only with bootstrap discovery.
func localOnlyCluster(a *args.ParsedArguments) bool {
	return a.LocalOnlyCollection && a.AutodiscoverStrategy == discoverBootstrap
}

// readControllerState sums each broker's own 0/1 ActiveControllerCount. A zero is trusted only if every broker
// answered: a failed, errored or absent read is unknown, not "no active controller". KRaft's raft-metrics
// current-leader is not used because it stays stale for minutes after the controller dies.
func readControllerState(brokers []*connection.Broker, provider connection.JMXProvider) controllerState {
	st := controllerState{}
	for _, b := range brokers {
		value, res := queryActiveControllerCount(b, provider)
		switch res {
		case queryAnswered:
			st.answered++
			st.count += value
			if value == 1 && st.controller == nil {
				st.controller = b
			}
		case queryAbsent:
			st.absent++
		}
	}
	st.known = st.count > 0 || st.answered == len(brokers)
	// Brokers without the MBean cannot be the controller, so only failed reads make a positive count a lower bound.
	if st.count > 0 && st.answered+st.absent < len(brokers) {
		log.Warn("ActiveControllerCount was read from %d of %d brokers; the reported count is a lower bound", st.answered+st.absent, len(brokers))
	}
	return st
}

func queryActiveControllerCount(b *connection.Broker, provider connection.JMXProvider) (int, queryResult) {
	conn, err := provider.NewConnection(jmxConfigFor(b))
	if err != nil {
		log.Error("Failed to create JMX connection to broker '%s' for ActiveControllerCount: %s", b.Host, err)
		return 0, queryFailed
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Debug("Unable to close JMX connection to broker '%s': %v", b.Host, err)
		}
	}()

	results, err := conn.QueryMBeanAttributes(activeControllerMBean)
	if err != nil {
		if e, ok := gojmx.IsJMXError(err); ok && strings.Contains(e.Message, "can't find mBean") {
			return 0, queryAbsent
		}
		log.Error("Failed to query ActiveControllerCount from broker '%s': %s", b.Host, err)
		return 0, queryFailed
	}
	if len(results) == 0 {
		return 0, queryAbsent
	}
	for _, attr := range results {
		if !strings.HasSuffix(attr.Name, "attr=Value") {
			continue
		}
		// nrjmx returns a failed attribute read as an error response that GetValueAsFloat turns into 0.
		if attr.ResponseType == gojmx.ResponseTypeErr {
			log.Warn("Failed to read ActiveControllerCount from broker '%s': %s", b.Host, attr.StatusMsg)
			return 0, queryFailed
		}
		if v, err := attr.GetValueAsFloat(); err == nil {
			return int(v), queryAnswered
		}
	}
	log.Warn("No numeric ActiveControllerCount value from broker '%s'", b.Host)
	return 0, queryFailed
}

// clusterMetricsTarget returns the broker to read cluster metrics from, or nil to skip this cycle. A local-only
// instance lists one broker, so each reports its own count; only a controller broker's row carries the other gauges.
func clusterMetricsTarget(brokers []*connection.Broker, st controllerState, localOnly bool) *connection.Broker {
	if len(brokers) == 0 {
		return nil
	}
	if !st.known {
		if st.absent == len(brokers) {
			logf := log.Warn
			if localOnly { // expected on the broker-only nodes of a mixed KRaft cluster
				logf = log.Info
			}
			logf("The ActiveControllerCount MBean was not found on any of the %d listed broker(s) (KRaft with dedicated controller nodes?); KafkaClusterSample is not sent", len(brokers))
		} else {
			log.Warn("Could not read ActiveControllerCount from every broker and none reported a controller; skipping cluster metrics this cycle")
		}
		return nil
	}
	if st.controller != nil {
		return st.controller
	}
	return brokers[0]
}
