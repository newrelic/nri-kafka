package zookeeper

import (
	"net/url"
	"testing"

	"github.com/newrelic/nri-kafka/src/testutils"
	"github.com/samuel/go-zookeeper/zk"
)

func Test_GetBrokerConnectionInfo_WithHost(t *testing.T) {
	testutils.SetupTestArgs()

	brokerID := 0
	zkConn := MockConnection{}
	zkConn.On("Get", "/brokers/ids/0").Return([]byte(`{"listener_security_protocol_map":{"SASL_SSL":"SASL_SSL","SSL":"SSL", "PLAINTEXT":"PLAINTEXT"},"endpoints":["SASL_SSL://my-broker.host:9193","SSL://my-broker.host:9093","PLAINTEXT://my-broker.host:9092"],"rack":"us-east-1d","jmx_port":9999,"host":null,"timestamp":"1542127633364","port":-1,"version":4}`), new(zk.Stat), nil)

	expectedValues := []BrokerConnection{
		{Scheme: "https", BrokerHost: "my-broker.host", JmxPort: 9999, BrokerPort: 9093},
		{Scheme: "http", BrokerHost: "my-broker.host", JmxPort: 9999, BrokerPort: 9092},
	}

	brokerConnections, err := GetBrokerConnectionInfo(brokerID, &zkConn)
	if err != nil {
		t.Fatalf("Unexpected error %s", err.Error())
	}

	if len(expectedValues) != len(brokerConnections) {
		t.Errorf("Expected %d entires got %d", len(expectedValues), len(brokerConnections))
	}

	for i, brokerConnection := range brokerConnections {
		if brokerConnection.Scheme != expectedValues[i].Scheme {
			t.Errorf("Expected %s got %s", expectedValues[i].Scheme, brokerConnection.Scheme)
		}
		if brokerConnection.BrokerHost != expectedValues[i].BrokerHost {
			t.Errorf("Expected %s got %s", expectedValues[i].BrokerHost, brokerConnection.BrokerHost)
		}
		if brokerConnection.JmxPort != expectedValues[i].JmxPort {
			t.Errorf("Expected %d got %d", expectedValues[i].JmxPort, brokerConnection.JmxPort)
		}
		if brokerConnection.BrokerPort != expectedValues[i].BrokerPort {
			t.Errorf("Expected %d got %d", expectedValues[i].BrokerPort, brokerConnection.BrokerPort)
		}
	}
}

func Test_GetBrokerConnectionInfo_WithProtocolMap(t *testing.T) {
	testutils.SetupTestArgs()

	brokerID := 0
	zkConn := MockConnection{}
	zkConn.On("Get", "/brokers/ids/0").Return([]byte(`{"listener_security_protocol_map":{"EXTERNAL":"SASL_SSL","INTERNAL":"PLAINTEXT"},"endpoints":["EXTERNAL://my-broker.host:9193", "INTERNAL://my-broker.host:9093"],"rack":"us-east-1d","jmx_port":9999,"host":null,"timestamp":"1542127633364","port":-1,"version":4}`), new(zk.Stat), nil)

	expectedValues := []BrokerConnection{
		{Scheme: "http", BrokerHost: "my-broker.host", JmxPort: 9999, BrokerPort: 9093},
	}

	brokerConnections, err := GetBrokerConnectionInfo(brokerID, &zkConn)

	if err != nil {
		t.Fatalf("Unexpected error %s", err.Error())
	}

	if len(expectedValues) != len(brokerConnections) {
		t.Errorf("Expected %d entires got %d", len(expectedValues), len(brokerConnections))
	}

	for i, brokerConnection := range brokerConnections {
		if brokerConnection.Scheme != expectedValues[i].Scheme {
			t.Errorf("Expected %s got %s", expectedValues[i].Scheme, brokerConnection.Scheme)
		}
		if brokerConnection.BrokerHost != expectedValues[i].BrokerHost {
			t.Errorf("Expected %s got %s", expectedValues[i].BrokerHost, brokerConnection.BrokerHost)
		}
		if brokerConnection.JmxPort != expectedValues[i].JmxPort {
			t.Errorf("Expected %d got %d", expectedValues[i].JmxPort, brokerConnection.JmxPort)
		}
		if brokerConnection.BrokerPort != expectedValues[i].BrokerPort {
			t.Errorf("Expected %d got %d", expectedValues[i].BrokerPort, brokerConnection.BrokerPort)
		}
	}
}

func Test_getUrlStringAndSchemeFromEndpoints_WithVanilla(t *testing.T) {
	testutils.SetupTestArgs()

	endpoints := []string{"SASL_SSL://my-broker.host:9193", "SSL://my-broker.host:9093", "PLAINTEXT://my-broker.host:9092"}
	protocolMap := map[string]string{
		"SASL_SSL":  "SASL_SSL",
		"SSL":       "SSL",
		"PLAINTEXT": "PLAINTEXT",
	}

	expectedSchemes := []string{"https", "http"}
	expectedHosts := []*url.URL{}
	host, err := url.Parse("SSL://my-broker.host:9093")
	expectedHosts = append(expectedHosts, host)
	host, err = url.Parse("PLAINTEXT://my-broker.host:9092")
	expectedHosts = append(expectedHosts, host)

	schemes, hosts, err := getURLStringAndSchemeFromEndpoints(endpoints, protocolMap)

	if err != nil {
		t.Fatalf("Unexpected error %s", err.Error())
	}
	for i, scheme := range schemes {
		if scheme != expectedSchemes[i] {
			t.Errorf("Expected '%s' got '%s'", expectedSchemes[i], scheme)
		}
	}
	for i, host := range hosts {
		if *host != *expectedHosts[i] {
			t.Errorf("Expected %s got %s", expectedHosts[i], host)
		}
	}

}

func Test_getUrlStringAndSchemeFromEndpoints_WithProtocolMap(t *testing.T) {
	testutils.SetupTestArgs()

	endpoints := []string{"EXTERNAL://my-broker.host:9193", "INTERNAL://my-broker.host:9093"}
	protocolMap := map[string]string{
		"EXTERNAL": "SASL_SSL",
		"INTERNAL": "PLAINTEXT",
	}

	expectedScheme := "http"
	expectedHost, err := url.Parse("INTERNAL://my-broker.host:9093")

	schemes, hosts, err := getURLStringAndSchemeFromEndpoints(endpoints, protocolMap)

	if err != nil {
		t.Fatalf("Unexpected error %s", err.Error())
	}

	for _, scheme := range schemes {
		if scheme != expectedScheme {
			t.Errorf("Expected '%s' got '%s'", expectedScheme, scheme)
		}
	}
	for _, host := range hosts {
		if *host != *expectedHost {
			t.Errorf("Expected %s got %s", expectedHost, host)
		}
	}

}
