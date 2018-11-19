package zookeeper

import (
	"testing"

	"github.com/newrelic/nri-kafka/src/testutils"
	"github.com/samuel/go-zookeeper/zk"
)

func Test_GetBrokerConnectionInfo_WithHost(t *testing.T) {
	testutils.SetupTestArgs()

	brokerID := 0
	zkConn := MockConnection{}
	zkConn.On("Get", "/brokers/ids/0").Return([]byte(`{"listener_security_protocol_map":{"SASL_SSL":"SASL_SSL","SSL":"SSL"},"endpoints":["SASL_SSL://my-broker.host:9193","SSL://my-broker.host:9093"],"rack":"us-east-1d","jmx_port":9999,"host":null,"timestamp":"1542127633364","port":-1,"version":4}`), new(zk.Stat), nil)

	expectedScheme, expectedHost, expectedJMXPort, expectedKafkaPort := "https", "my-broker.host", 9999, 9093

	scheme, host, jmxPort, kafkaPort, err := GetBrokerConnectionInfo(brokerID, &zkConn)
	if err != nil {
		t.Fatalf("Unexpected error %s", err.Error())
	}

	if scheme != expectedScheme {
		t.Errorf("Expected %s got %s", expectedScheme, scheme)
	}
	if host != expectedHost {
		t.Errorf("Expected %s got %s", expectedHost, host)
	}
	if jmxPort != expectedJMXPort {
		t.Errorf("Expected %d got %d", expectedJMXPort, jmxPort)
	}
	if kafkaPort != expectedKafkaPort {
		t.Errorf("Expected %d got %d", expectedKafkaPort, kafkaPort)
	}
}
