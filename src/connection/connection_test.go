package connection

import (
	"testing"

	"github.com/newrelic/nri-kafka/src/testutils"
	"github.com/newrelic/nri-kafka/src/zookeeper"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/stretchr/testify/assert"
)

func Test_GetBrokerFromZookeeper(t *testing.T) {
	testutils.SetupTestArgs()

	zkConn := &zookeeper.MockConnection{}
	zkConn.On("Get", "/brokers/ids/0").Return([]byte(`{"listener_security_protocol_map":{"SASL_SSL":"SASL_SSL","SSL":"SSL", "PLAINTEXT":"PLAINTEXT"},"endpoints":["SASL_SSL://my-broker.host:9193","SSL://my-broker.host:9093","PLAINTEXT://my-broker.host:9092"],"rack":"us-east-1d","jmx_port":9999,"host":null,"timestamp":"1542127633364","port":-1,"version":4}`), new(zk.Stat), nil)
	zkConn.On("Server").Return("localhost:2181")

	_, err := GetBrokerFromZookeeper(zkConn, "0", "PLAINTEXT")
	assert.Error(t, err, "Expected error getting broker information from mock")

}

func parseEndpointTest(t *testing.T, input, listener, host string, port int, isErr bool) {
	l, h, p, e := parseEndpoint(input)
	if isErr {
		assert.Error(t, e, "Expected an error")
	} else {
		assert.NoError(t, e, "Unexpected error")
	}

	assert.Equal(t, listener, l)
	assert.Equal(t, host, h)
	assert.Equal(t, port, p)
}

func Test_parseEndpoint_1(t *testing.T) {
	parseEndpointTest(t, "TEST_LISTENER://myhost.com:1265", "TEST_LISTENER", "myhost.com", 1265, false)
}

func Test_parseEndpoint_2(t *testing.T) {
	parseEndpointTest(t, "TEST_LISTENER://myhost.com:asdf", "", "", 0, true)
}
