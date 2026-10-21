package main

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/connection"
	"github.com/newrelic/nri-kafka/src/testutils"
	"github.com/newrelic/nrjmx/gojmx"
	"github.com/stretchr/testify/assert"
)

// brokerFake describes how one broker's JMX answers the ActiveControllerCount query.
type brokerFake struct {
	connectErr error
	queryErr   error
	result     []*gojmx.AttributeResponse
}

type fakeJMX struct{ byHost map[string]brokerFake }

func (f *fakeJMX) NewConnection(cfg *gojmx.JMXConfig) (connection.JMXConnection, error) {
	b, ok := f.byHost[cfg.Hostname]
	if !ok {
		return nil, errors.New("connection refused")
	}
	if b.connectErr != nil {
		return nil, b.connectErr
	}
	return &fakeConn{b: b}, nil
}

type fakeConn struct{ b brokerFake }

func (c *fakeConn) QueryMBeanAttributes(string) ([]*gojmx.AttributeResponse, error) {
	return c.b.result, c.b.queryErr
}
func (c *fakeConn) QueryMBeanNames(string) ([]string, error) { return nil, nil }
func (c *fakeConn) Close() error                             { return nil }

func countAttr(v int64) []*gojmx.AttributeResponse {
	return []*gojmx.AttributeResponse{{Name: activeControllerMBean + ",attr=Value", ResponseType: gojmx.ResponseTypeInt, IntValue: v}}
}

var (
	// nrjmx reports an absent MBean as a JMX error, not as an empty result.
	mbeanAbsent = brokerFake{queryErr: &gojmx.JMXError{Message: "can't find mBean: " + activeControllerMBean}}
	// nrjmx reports a failed attribute read as a successful query holding an error response.
	attrError = brokerFake{result: []*gojmx.AttributeResponse{{Name: activeControllerMBean + ",attr=Value", ResponseType: gojmx.ResponseTypeErr, StatusMsg: "can't parse attribute"}}}
	// anything else a query can fail with.
	queryTimeout = brokerFake{queryErr: errors.New("request timeout exceeded")}
)

func answer(v int64) brokerFake { return brokerFake{result: countAttr(v)} }

func brokersNamed(ids ...string) []*connection.Broker {
	var bs []*connection.Broker
	for _, id := range ids {
		bs = append(bs, &connection.Broker{Host: "b" + id, ID: id})
	}
	return bs
}

func TestReadControllerState(t *testing.T) {
	testutils.SetupTestArgs()

	cases := []struct {
		name           string
		brokers        map[string]brokerFake // keyed by host; a missing host refuses the connection
		wantCount      int
		wantKnown      bool
		wantController string // broker ID, "" for none
	}{
		{"one controller among two", map[string]brokerFake{"b1": answer(0), "b2": answer(1)}, 1, true, "2"},
		{"no controller, every broker answered 0", map[string]brokerFake{"b1": answer(0), "b2": answer(0)}, 0, true, ""},
		{"split brain sums to 2 and keeps the first controller", map[string]brokerFake{"b1": answer(1), "b2": answer(1)}, 2, true, "1"},
		{"three controllers are summed, not overwritten", map[string]brokerFake{"b1": answer(1), "b2": answer(1), "b3": answer(1)}, 3, true, "1"},
		{"one refused and the other says 0 is unknown", map[string]brokerFake{"b2": answer(0)}, 0, false, ""},
		{"one refused but the other found the controller", map[string]brokerFake{"b2": answer(1)}, 1, true, "2"},
		{"query error on one and 0 on the other is unknown", map[string]brokerFake{"b1": queryTimeout, "b2": answer(0)}, 0, false, ""},
		{"per-attribute error response is not an answer of 0", map[string]brokerFake{"b1": attrError, "b2": answer(0)}, 0, false, ""},
		{"per-attribute error on every broker is unknown", map[string]brokerFake{"b1": attrError, "b2": attrError}, 0, false, ""},
		{"absent MBean on every broker (dedicated KRaft controllers) is unknown", map[string]brokerFake{"b1": mbeanAbsent, "b2": mbeanAbsent}, 0, false, ""},
		{"a broker without the MBean next to a controller is fine", map[string]brokerFake{"b1": answer(1), "b2": mbeanAbsent}, 1, true, "1"},
		{"a zero next to a broker without the MBean stays unknown", map[string]brokerFake{"b1": answer(0), "b2": mbeanAbsent}, 0, false, ""},
		{"empty result on every broker is unknown", map[string]brokerFake{"b1": {}, "b2": {}}, 0, false, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ids := []string{"1", "2"}
			if _, three := c.brokers["b3"]; three {
				ids = append(ids, "3")
			}
			st := readControllerState(brokersNamed(ids...), &fakeJMX{byHost: c.brokers})
			assert.Equal(t, c.wantCount, st.count)
			assert.Equal(t, c.wantKnown, st.known)
			if c.wantController == "" {
				assert.Nil(t, st.controller)
			} else if assert.NotNil(t, st.controller) {
				assert.Equal(t, c.wantController, st.controller.ID)
			}
		})
	}
}

func TestClusterMetricsTarget(t *testing.T) {
	local := brokersNamed("1")
	two := brokersNamed("1", "2")

	cases := []struct {
		name      string
		brokers   []*connection.Broker
		st        controllerState
		localOnly bool
		want      *connection.Broker
	}{
		{"local-only on the controller reports", local, controllerState{known: true, count: 1, controller: local[0]}, true, local[0]},
		{"local-only on a non-controller skips", local, controllerState{known: true, count: 0}, true, nil},
		{"local-only with an unknown count skips", local, controllerState{known: false}, true, nil},
		{"non-local reads the gauges from the controller", two, controllerState{known: true, count: 1, controller: two[1]}, false, two[1]},
		{"non-local with a confirmed zero still reports, from the first broker", two, controllerState{known: true, count: 0}, false, two[0]},
		{"non-local with an unknown count skips", two, controllerState{known: false}, false, nil},
		{"no brokers skips", nil, controllerState{known: true}, false, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, clusterMetricsTarget(c.brokers, c.st, c.localOnly))
		})
	}
}

func TestLocalOnlyCluster(t *testing.T) {
	cases := []struct {
		strategy string
		flag     bool
		want     bool
	}{
		{discoverBootstrap, true, true},
		{discoverBootstrap, false, false},
		// ZooKeeper discovery lists every broker whatever the flag says, so instances are not local.
		{discoverZookeeper, true, false},
		{discoverZookeeper, false, false},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, localOnlyCluster(&args.ParsedArguments{AutodiscoverStrategy: c.strategy, LocalOnlyCollection: c.flag}), "%s/%v", c.strategy, c.flag)
	}
}

func TestReadControllerState_LowerBoundWarning(t *testing.T) {
	testutils.SetupTestArgs()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	// A broker-only node (MBean absent) next to the controller is a healthy mixed cluster: no warning.
	readControllerState(brokersNamed("1", "2"), &fakeJMX{byHost: map[string]brokerFake{"b1": answer(1), "b2": mbeanAbsent}})
	assert.NotContains(t, buf.String(), "lower bound")

	// A failed read next to a controller means the count may be too low: warn.
	readControllerState(brokersNamed("1", "2"), &fakeJMX{byHost: map[string]brokerFake{"b1": answer(1), "b2": queryTimeout}})
	assert.Contains(t, buf.String(), "lower bound")
}

func TestClusterMetricsTarget_AbsentMBeanLogLevel(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	st := controllerState{known: false, absent: 1}

	assert.Nil(t, clusterMetricsTarget(brokersNamed("1"), st, false))
	assert.Contains(t, buf.String(), "[WARN]", "a non-local instance is expected to work, so this is a warning")

	buf.Reset()
	assert.Nil(t, clusterMetricsTarget(brokersNamed("1"), st, true))
	assert.Contains(t, buf.String(), "[INFO]", "broker-only nodes of a mixed KRaft cluster hit this every cycle")
	assert.NotContains(t, buf.String(), "[WARN]")
}
