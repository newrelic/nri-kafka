package zookeeper

import (
	"errors"
	"regexp"

	"github.com/samuel/go-zookeeper/zk"
)

// MockConnection implements Connectionto facilitate testing.
// Includes two properties
type MockConnection struct {
	ReturnGetError      bool // If true, a Get() method call will return an error
	ReturnChildrenError bool // If true, a Children() method call will return an error
}

// Get function of the Connection interface
func (m MockConnection) Get(s string) ([]byte, *zk.Stat, error) {
	if m.ReturnGetError {
		return nil, nil, errors.New("This is a test error")
	}

	// Mock responses from the Zookeeper connection
	if matched, _ := regexp.Match("/brokers/topics/[^/]*/partitions/[0-9]/state", []byte(s)); matched {
		outbytes := []byte(`{"controller_epoch":9,"leader":2,"version":1,"leader_epoch":0,"isr":[2,0,1]}`)
		outstat := new(zk.Stat)
		return outbytes, outstat, nil
	} else if matched, _ := regexp.Match("/brokers/topics/test1", []byte(s)); matched {
		outbytes := []byte(`{"version":1,"partitions":{"0":[1,2,0]}}`)
		outstat := new(zk.Stat)
		return outbytes, outstat, nil
	} else if matched, _ := regexp.Match("/brokers/topics/[^/]*", []byte(s)); matched {
		outbytes := []byte(`{"version":1,"partitions":{"2":[1,2,0],"1":[0,1,2],"0":[2,0,1]}}`)
		outstat := new(zk.Stat)
		return outbytes, outstat, nil
	} else if matched, _ := regexp.Match("/brokers/ids/[0-9]", []byte(s)); matched {
		outbytes := []byte(`{"listener_security_protocol_map":{"PLAINTEXT":"PLAINTEXT"},"endpoints":["PLAINTEXT://kafkabroker:9092"],"jmx_port":9999,"host":"kafkabroker","timestamp":"1530886155628","port":9092,"version":4}`)
		outstat := new(zk.Stat)
		return outbytes, outstat, nil
	} else if matched, _ := regexp.Match("/config/topics/[^/]*", []byte(s)); matched {
		outbytes := []byte(`{"version":1,"config":{"flush.messages":"12345"}}`)
		outstat := new(zk.Stat)
		return outbytes, outstat, nil
	} else if matched, _ := regexp.Match("/config/brokers/10", []byte(s)); matched {
		return nil, nil, errors.New("zk: node does not exist")
	} else if matched, _ := regexp.Match("/config/brokers/[0-9]", []byte(s)); matched {
		outbytes := []byte(`{"version":1,"config":{"leader.replication.throttled.replicas":"10000"}}`)
		outstat := new(zk.Stat)
		return outbytes, outstat, nil
	}

	return nil, nil, errors.New("The endpoint " + s + " is not defined in the mock connection interface.")
}

// Children method of the Connection interface
func (m MockConnection) Children(s string) ([]string, *zk.Stat, error) {
	if m.ReturnChildrenError {
		return nil, nil, errors.New("This is a test error")
	}

	// Mock responses from the Zookeeper connection
	if s == "/brokers/topics" {
		outstrings := []string{"test1", "test2", "test3"}
		outstat := new(zk.Stat)
		return outstrings, outstat, nil
	} else if matched, _ := regexp.Match("/brokers/topics/[^/]*/partitions", []byte(s)); matched {
		outstrings := []string{"0", "1", "2"}
		outstat := new(zk.Stat)
		return outstrings, outstat, nil
	} else if s == "/brokers/ids" {
		outstrings := []string{"0", "1", "2"}
		outstat := new(zk.Stat)
		return outstrings, outstat, nil
	}
	return nil, nil, errors.New("The endpoint " + s + " is not defined in the tests")
}
