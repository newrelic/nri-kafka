package main

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/nri-kafka/src/args"
	"github.com/newrelic/nri-kafka/src/jmxwrapper"
	"github.com/newrelic/nri-kafka/src/zookeeper"
	"github.com/samuel/go-zookeeper/zk"
)

func benchCoreCollection(b *testing.B, numBrokers, numTopics, numPartitions int) {
	b.StopTimer()

	args.GlobalArgs = &args.KafkaArguments{
		TopicMode:              "All",
		CollectBrokerTopicData: true,
	}

	zkConn := zookeeper.MockConnection{}
	returnTopics := make([]string, numTopics, numTopics)
	partitionsList := make([]string, numPartitions, numPartitions)
	for i := 0; i < numPartitions; i++ {
		partitionsList[i] = strconv.Itoa(i)
	}
	partitionsListString := strings.Join(partitionsList, ",")
	topicState := []byte(`{"version":1,"partitions":{"0":[` + partitionsListString + `]}}`)
	partitionState := []byte(`{"controller_epoch":9,"leader":2,"version":1,"leader_epoch":0,"isr":[` + partitionsListString + `]}`)

	for i := 0; i < numTopics; i++ {
		returnTopics[i] = "topic" + strconv.Itoa(i)
		zkConn.On("Get", "/config/topics/topic"+strconv.Itoa(i)).Return([]byte(`{"version":1,"config":{"flush.messages":"12345"}}`), &zk.Stat{}, nil)
		zkConn.On("Get", "/brokers/topics/topic"+strconv.Itoa(i)).Return(topicState, &zk.Stat{}, nil)
		for _, partition := range partitionsList {
			zkConn.On("Get", "/brokers/topics/topic"+strconv.Itoa(i)+"/partitions/"+partition+"/state").Return(partitionState, &zk.Stat{}, nil)
		}
	}
	returnBrokers := make([]string, numBrokers, numBrokers)
	brokerConfigBytes := []byte(`{"version":1,"config":{"flush.messages":"12345"}}`)
	for i := 0; i < numBrokers; i++ {
		returnBrokers[i] = strconv.Itoa(i)
		brokerConnectionBytes := []byte(`{"listener_security_protocol_map":{"PLAINTEXT":"PLAINTEXT"},"endpoints":["PLAINTEXT://kafkabroker:9092"],"jmx_port":9999,"host":"kafkabroker` + strconv.Itoa(i) + `","timestamp":"1530886155628","port":9092,"version":4}`)
		zkConn.On("Get", "/brokers/ids/"+strconv.Itoa(i)).Return(brokerConnectionBytes, &zk.Stat{}, nil)
		zkConn.On("Get", "/config/brokers/"+strconv.Itoa(i)).Return(brokerConfigBytes, &zk.Stat{}, nil)
	}

	zkConn.On("Children", "/brokers/topics").Return(returnTopics, &zk.Stat{}, nil)
	zkConn.On("Children", "/brokers/ids").Return(returnBrokers, &zk.Stat{}, nil)
	jmxwrapper.JMXQuery = createMockQuery(numTopics, numPartitions)

	b.StartTimer()
	for n := 0; n < b.N; n++ {
		integration, _ := integration.New("test"+strconv.Itoa(n), "test")
		coreCollection(zkConn, integration)
	}

}

func createMockQuery(numTopics, numPartitions int) func(string, int) (map[string]interface{}, error) {
	return func(objectPattern string, timeout int) (map[string]interface{}, error) {
		var result map[string]interface{}
		if strings.HasPrefix(objectPattern, "kafka.consumer:type=consumer-fetch-manager-metrics,client-id=") {
			err := json.Unmarshal([]byte("{}"), &result)
			return result, err
		} else if strings.HasPrefix(objectPattern, "kafka.consumer:type=ZookeeperConsumerConnector,name=*,clientId=") {
			err := json.Unmarshal([]byte("{}"), &result)
			return result, err
		} else if strings.HasPrefix(objectPattern, "kafka.consumer:type=consumer-fetch-manager-metrics,client-id=") {
			err := json.Unmarshal([]byte("{}"), &result)
			return result, err
		} else if strings.HasPrefix(objectPattern, "kafka.producer:type=producer-metrics,client-id=") {
			err := json.Unmarshal([]byte("{}"), &result)
			return result, err
		} else if strings.HasPrefix(objectPattern, "kafka.producer:type=ProducerTopicMetrics,name=MessagesPerSec,clientId=") {
			err := json.Unmarshal([]byte("{}"), &result)
			return result, err
		} else if strings.HasPrefix(objectPattern, "kafka.producer:type=producer-topic-metrics,client-id=") {
			err := json.Unmarshal([]byte("{}"), &result)
			return result, err
		} else if strings.HasPrefix(objectPattern, "kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Metadata") {
			err := json.Unmarshal([]byte(`{
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Metadata,attr=Count": 391,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Metadata,attr=StdDev": 1.2214072017196433,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Metadata,attr=98thPercentile": 1,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Metadata,attr=99thPercentile": 1,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Metadata,attr=999thPercentile": 1,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Metadata,attr=Min": 0,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Metadata,attr=Mean": 0.4296675191815857,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Metadata,attr=Max": 12,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Metadata,attr=95thPercentile": 1,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Metadata,attr=50thPercentile": 0,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Metadata,attr=75thPercentile": 0
			}`), &result)
			return result, err
		} else if strings.HasPrefix(objectPattern, "kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Fetch") {
			err := json.Unmarshal([]byte(`{
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Fetch,attr=Min": 0,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Fetch,attr=StdDev": 96.06489428964748,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Fetch,attr=Max": 528,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Fetch,attr=98thPercentile": 1,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Fetch,attr=75thPercentile": 1,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Fetch,attr=50thPercentile": 1,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Fetch,attr=Mean": 463.23688906798685,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Fetch,attr=999thPercentile": 1,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Fetch,attr=Count": 17371,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Fetch,attr=95thPercentile": 1,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Fetch,attr=99thPercentile": 1
			}`), &result)
			return result, err
		} else if strings.HasPrefix(objectPattern, "kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Offsets") {
			err := json.Unmarshal([]byte("{}"), &result)
			return result, err
		} else if strings.HasPrefix(objectPattern, "kafka.network:type=RequestMetrics,name=TotalTimeMs,request=UpdateMetadata") {
			err := json.Unmarshal([]byte(`{
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=UpdateMetadata,attr=95thPercentile": 0,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=UpdateMetadata,attr=Max": 12,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=UpdateMetadata,attr=Count": 12,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=UpdateMetadata,attr=Mean": 1.3333333333333333,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=UpdateMetadata,attr=StdDev": 3.420083288016492,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=UpdateMetadata,attr=75thPercentile": 0,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=UpdateMetadata,attr=99thPercentile": 0,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=UpdateMetadata,attr=50thPercentile": 0,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=UpdateMetadata,attr=Min": 0,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=UpdateMetadata,attr=999thPercentile": 0,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=UpdateMetadata,attr=98thPercentile": 0
			}`), &result)
			return result, err
		} else if strings.HasPrefix(objectPattern, "kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Produce") {
			err := json.Unmarshal([]byte(`{
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Produce,attr=Max": 12,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Produce,attr=98thPercentile": 1,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Produce,attr=75thPercentile": 0,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Produce,attr=50thPercentile": 0,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Produce,attr=StdDev": 0.2984625003592795,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Produce,attr=999thPercentile": 2.9420000000000073,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Produce,attr=95thPercentile": 1,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Produce,attr=99thPercentile": 1,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Produce,attr=Mean": 0.0606366851945427,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Produce,attr=Count": 17811,
				"kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Produce,attr=Min": 0
			}`), &result)
			return result, err
		} else if strings.HasPrefix(objectPattern, "kafka.server:type=ReplicaManager,name=*") {
			err := json.Unmarshal([]byte(`{
				"kafka.server:type=ReplicaManager,name=IsrShrinksPerSec,attr=MeanRate": 0,
				"kafka.server:type=ReplicaManager,name=FailedIsrUpdatesPerSec,attr=FifteenMinuteRate": 0,
				"kafka.server:type=ReplicaManager,name=FailedIsrUpdatesPerSec,attr=EventType": "failedUpdates",
				"kafka.server:type=ReplicaManager,name=UnderReplicatedPartitions,attr=Value": 0,
				"kafka.server:type=ReplicaManager,name=IsrExpandsPerSec,attr=MeanRate": 0,
				"kafka.server:type=ReplicaManager,name=IsrExpandsPerSec,attr=FifteenMinuteRate": 0,
				"kafka.server:type=ReplicaManager,name=IsrShrinksPerSec,attr=OneMinuteRate": 0,
				"kafka.server:type=ReplicaManager,name=IsrShrinksPerSec,attr=FiveMinuteRate": 0,
				"kafka.server:type=ReplicaManager,name=IsrExpandsPerSec,attr=FiveMinuteRate": 0,
				"kafka.server:type=ReplicaManager,name=PartitionCount,attr=Value": 59,
				"kafka.server:type=ReplicaManager,name=IsrExpandsPerSec,attr=OneMinuteRate": 0,
				"kafka.server:type=ReplicaManager,name=IsrExpandsPerSec,attr=EventType": "expands",
				"kafka.server:type=ReplicaManager,name=FailedIsrUpdatesPerSec,attr=OneMinuteRate": 0,
				"kafka.server:type=ReplicaManager,name=IsrShrinksPerSec,attr=FifteenMinuteRate": 0,
				"kafka.server:type=ReplicaManager,name=IsrShrinksPerSec,attr=EventType": "shrinks",
				"kafka.server:type=ReplicaManager,name=OfflineReplicaCount,attr=Value": 0,
				"kafka.server:type=ReplicaManager,name=IsrShrinksPerSec,attr=Count": 0,
				"kafka.server:type=ReplicaManager,name=FailedIsrUpdatesPerSec,attr=Count": 0,
				"kafka.server:type=ReplicaManager,name=FailedIsrUpdatesPerSec,attr=MeanRate": 0,
				"kafka.server:type=ReplicaManager,name=LeaderCount,attr=Value": 59,
				"kafka.server:type=ReplicaManager,name=IsrExpandsPerSec,attr=Count": 0,
				"kafka.server:type=ReplicaManager,name=FailedIsrUpdatesPerSec,attr=FiveMinuteRate": 0,
				"kafka.server:type=ReplicaManager,name=UnderMinIsrPartitionCount,attr=Value": 0
			}`), &result)
			return result, err
		} else if strings.HasPrefix(objectPattern, "kafka.controller:type=ControllerStats,name=*") {
			err := json.Unmarshal([]byte(`{
				"kafka.controller:type=ControllerStats,name=AutoLeaderBalanceRateAndTimeMs,attr=999thPercentile": 0.371844,
				"kafka.controller:type=ControllerStats,name=ManualLeaderBalanceRateAndTimeMs,attr=FiveMinuteRate": 0,
				"kafka.controller:type=ControllerStats,name=AutoLeaderBalanceRateAndTimeMs,attr=Min": 0.158353,
				"kafka.controller:type=ControllerStats,name=AutoLeaderBalanceRateAndTimeMs,attr=MeanRate": 0.003333752268587028,
				"kafka.controller:type=ControllerStats,name=LeaderAndIsrResponseReceivedRateAndTimeMs,attr=Mean": 0.19580290909090908,
				"kafka.controller:type=ControllerStats,name=LeaderElectionRateAndTimeMs,attr=MeanRate": 1.1511543960807962e-06,
				"kafka.controller:type=ControllerStats,name=IsrChangeRateAndTimeMs,attr=FifteenMinuteRate": 0,
				"kafka.controller:type=ControllerStats,name=LeaderAndIsrResponseReceivedRateAndTimeMs,attr=999thPercentile": 0.041989,
				"kafka.controller:type=ControllerStats,name=LogDirChangeRateAndTimeMs,attr=MeanRate": 0,
				"kafka.controller:type=ControllerStats,name=ManualLeaderBalanceRateAndTimeMs,attr=50thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=ControllerChangeRateAndTimeMs,attr=MeanRate": 2.302313001208997e-06,
				"kafka.controller:type=ControllerStats,name=ControllerChangeRateAndTimeMs,attr=75thPercentile": 88.008219,
				"kafka.controller:type=ControllerStats,name=PartitionReassignmentRateAndTimeMs,attr=75thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=LogDirChangeRateAndTimeMs,attr=Max": 0,
				"kafka.controller:type=ControllerStats,name=UncleanLeaderElectionsPerSec,attr=FiveMinuteRate": 0,
				"kafka.controller:type=ControllerStats,name=ManualLeaderBalanceRateAndTimeMs,attr=95thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=TopicDeletionRateAndTimeMs,attr=OneMinuteRate": 0,
				"kafka.controller:type=ControllerStats,name=UncleanLeaderElectionsPerSec,attr=Count": 0,
				"kafka.controller:type=ControllerStats,name=UncleanLeaderElectionsPerSec,attr=FifteenMinuteRate": 0,
				"kafka.controller:type=ControllerStats,name=ManualLeaderBalanceRateAndTimeMs,attr=Count": 0,
				"kafka.controller:type=ControllerStats,name=ManualLeaderBalanceRateAndTimeMs,attr=99thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=LogDirChangeRateAndTimeMs,attr=EventType": "calls",
				"kafka.controller:type=ControllerStats,name=TopicDeletionRateAndTimeMs,attr=StdDev": 0,
				"kafka.controller:type=ControllerStats,name=IsrChangeRateAndTimeMs,attr=Count": 0,
				"kafka.controller:type=ControllerStats,name=AutoLeaderBalanceRateAndTimeMs,attr=StdDev": 0.08903059448522639,
				"kafka.controller:type=ControllerStats,name=TopicDeletionRateAndTimeMs,attr=50thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=ControllerChangeRateAndTimeMs,attr=OneMinuteRate": 2.964393875e-314,
				"kafka.controller:type=ControllerStats,name=TopicDeletionRateAndTimeMs,attr=Min": 0,
				"kafka.controller:type=ControllerStats,name=ManualLeaderBalanceRateAndTimeMs,attr=98thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=LogDirChangeRateAndTimeMs,attr=StdDev": 0,
				"kafka.controller:type=ControllerStats,name=PartitionReassignmentRateAndTimeMs,attr=999thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=PartitionReassignmentRateAndTimeMs,attr=Count": 0,
				"kafka.controller:type=ControllerStats,name=ControlledShutdownRateAndTimeMs,attr=OneMinuteRate": 0,
				"kafka.controller:type=ControllerStats,name=AutoLeaderBalanceRateAndTimeMs,attr=EventType": "calls",
				"kafka.controller:type=ControllerStats,name=LeaderElectionRateAndTimeMs,attr=StdDev": 0,
				"kafka.controller:type=ControllerStats,name=TopicChangeRateAndTimeMs,attr=FiveMinuteRate": 1.4821969375e-313,
				"kafka.controller:type=ControllerStats,name=LeaderElectionRateAndTimeMs,attr=Count": 1,
				"kafka.controller:type=ControllerStats,name=AutoLeaderBalanceRateAndTimeMs,attr=FiveMinuteRate": 0.002870040641368859,
				"kafka.controller:type=ControllerStats,name=ControlledShutdownRateAndTimeMs,attr=Mean": 0,
				"kafka.controller:type=ControllerStats,name=ControlledShutdownRateAndTimeMs,attr=75thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=IsrChangeRateAndTimeMs,attr=StdDev": 0,
				"kafka.controller:type=ControllerStats,name=TopicDeletionRateAndTimeMs,attr=Max": 0,
				"kafka.controller:type=ControllerStats,name=LogDirChangeRateAndTimeMs,attr=FifteenMinuteRate": 0,
				"kafka.controller:type=ControllerStats,name=TopicChangeRateAndTimeMs,attr=95thPercentile": 5.912408,
				"kafka.controller:type=ControllerStats,name=LogDirChangeRateAndTimeMs,attr=Count": 0,
				"kafka.controller:type=ControllerStats,name=ManualLeaderBalanceRateAndTimeMs,attr=MeanRate": 0,
				"kafka.controller:type=ControllerStats,name=LeaderElectionRateAndTimeMs,attr=999thPercentile": 35.367419,
				"kafka.controller:type=ControllerStats,name=ControlledShutdownRateAndTimeMs,attr=FifteenMinuteRate": 0,
				"kafka.controller:type=ControllerStats,name=IsrChangeRateAndTimeMs,attr=Max": 0,
				"kafka.controller:type=ControllerStats,name=TopicDeletionRateAndTimeMs,attr=99thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=TopicChangeRateAndTimeMs,attr=98thPercentile": 5.912408,
				"kafka.controller:type=ControllerStats,name=LeaderElectionRateAndTimeMs,attr=OneMinuteRate": 2.964393875e-314,
				"kafka.controller:type=ControllerStats,name=TopicChangeRateAndTimeMs,attr=OneMinuteRate": 2.964393875e-314,
				"kafka.controller:type=ControllerStats,name=TopicChangeRateAndTimeMs,attr=75thPercentile": 5.88630125,
				"kafka.controller:type=ControllerStats,name=TopicDeletionRateAndTimeMs,attr=95thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=TopicDeletionRateAndTimeMs,attr=75thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=ControlledShutdownRateAndTimeMs,attr=StdDev": 0,
				"kafka.controller:type=ControllerStats,name=IsrChangeRateAndTimeMs,attr=Min": 0,
				"kafka.controller:type=ControllerStats,name=TopicDeletionRateAndTimeMs,attr=98thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=TopicChangeRateAndTimeMs,attr=Count": 9,
				"kafka.controller:type=ControllerStats,name=TopicDeletionRateAndTimeMs,attr=Mean": 0,
				"kafka.controller:type=ControllerStats,name=TopicChangeRateAndTimeMs,attr=50thPercentile": 5.7669195,
				"kafka.controller:type=ControllerStats,name=TopicDeletionRateAndTimeMs,attr=EventType": "calls",
				"kafka.controller:type=ControllerStats,name=ManualLeaderBalanceRateAndTimeMs,attr=Min": 0,
				"kafka.controller:type=ControllerStats,name=LeaderAndIsrResponseReceivedRateAndTimeMs,attr=MeanRate": 1.266276185336283e-05,
				"kafka.controller:type=ControllerStats,name=TopicChangeRateAndTimeMs,attr=Max": 190.932273,
				"kafka.controller:type=ControllerStats,name=ManualLeaderBalanceRateAndTimeMs,attr=EventType": "calls",
				"kafka.controller:type=ControllerStats,name=UncleanLeaderElectionsPerSec,attr=OneMinuteRate": 0,
				"kafka.controller:type=ControllerStats,name=ControllerChangeRateAndTimeMs,attr=StdDev": 62.22483883716583,
				"kafka.controller:type=ControllerStats,name=TopicDeletionRateAndTimeMs,attr=FiveMinuteRate": 0,
				"kafka.controller:type=ControllerStats,name=LeaderElectionRateAndTimeMs,attr=Min": 35.367419,
				"kafka.controller:type=ControllerStats,name=UncleanLeaderElectionsPerSec,attr=EventType": "elections",
				"kafka.controller:type=ControllerStats,name=LogDirChangeRateAndTimeMs,attr=Min": 0,
				"kafka.controller:type=ControllerStats,name=ControlledShutdownRateAndTimeMs,attr=Count": 0,
				"kafka.controller:type=ControllerStats,name=TopicChangeRateAndTimeMs,attr=StdDev": 61.55285222263863,
				"kafka.controller:type=ControllerStats,name=PartitionReassignmentRateAndTimeMs,attr=OneMinuteRate": 0,
				"kafka.controller:type=ControllerStats,name=LogDirChangeRateAndTimeMs,attr=FiveMinuteRate": 0,
				"kafka.controller:type=ControllerStats,name=LogDirChangeRateAndTimeMs,attr=50thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=TopicChangeRateAndTimeMs,attr=999thPercentile": 5.912408,
				"kafka.controller:type=ControllerStats,name=LeaderElectionRateAndTimeMs,attr=Max": 35.367419,
				"kafka.controller:type=ControllerStats,name=TopicChangeRateAndTimeMs,attr=Min": 4.743521,
				"kafka.controller:type=ControllerStats,name=LogDirChangeRateAndTimeMs,attr=95thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=LogDirChangeRateAndTimeMs,attr=98thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=LogDirChangeRateAndTimeMs,attr=75thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=ControlledShutdownRateAndTimeMs,attr=999thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=PartitionReassignmentRateAndTimeMs,attr=MeanRate": 0,
				"kafka.controller:type=ControllerStats,name=PartitionReassignmentRateAndTimeMs,attr=EventType": "calls",
				"kafka.controller:type=ControllerStats,name=ControlledShutdownRateAndTimeMs,attr=Min": 0,
				"kafka.controller:type=ControllerStats,name=LeaderAndIsrResponseReceivedRateAndTimeMs,attr=Count": 11,
				"kafka.controller:type=ControllerStats,name=LogDirChangeRateAndTimeMs,attr=999thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=ManualLeaderBalanceRateAndTimeMs,attr=StdDev": 0,
				"kafka.controller:type=ControllerStats,name=LogDirChangeRateAndTimeMs,attr=99thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=UncleanLeaderElectionsPerSec,attr=MeanRate": 0,
				"kafka.controller:type=ControllerStats,name=LeaderAndIsrResponseReceivedRateAndTimeMs,attr=Max": 1.326809,
				"kafka.controller:type=ControllerStats,name=PartitionReassignmentRateAndTimeMs,attr=Mean": 0,
				"kafka.controller:type=ControllerStats,name=ControlledShutdownRateAndTimeMs,attr=50thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=ControllerChangeRateAndTimeMs,attr=Max": 88.008219,
				"kafka.controller:type=ControllerStats,name=TopicChangeRateAndTimeMs,attr=Mean": 26.88580166666667,
				"kafka.controller:type=ControllerStats,name=IsrChangeRateAndTimeMs,attr=Mean": 0,
				"kafka.controller:type=ControllerStats,name=ControlledShutdownRateAndTimeMs,attr=Max": 0,
				"kafka.controller:type=ControllerStats,name=PartitionReassignmentRateAndTimeMs,attr=95thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=ControlledShutdownRateAndTimeMs,attr=FiveMinuteRate": 0,
				"kafka.controller:type=ControllerStats,name=ControllerChangeRateAndTimeMs,attr=EventType": "calls",
				"kafka.controller:type=ControllerStats,name=PartitionReassignmentRateAndTimeMs,attr=Max": 0,
				"kafka.controller:type=ControllerStats,name=ControllerChangeRateAndTimeMs,attr=99thPercentile": 88.008219,
				"kafka.controller:type=ControllerStats,name=TopicDeletionRateAndTimeMs,attr=FifteenMinuteRate": 0,
				"kafka.controller:type=ControllerStats,name=AutoLeaderBalanceRateAndTimeMs,attr=Count": 2896,
				"kafka.controller:type=ControllerStats,name=ControllerChangeRateAndTimeMs,attr=98thPercentile": 88.008219,
				"kafka.controller:type=ControllerStats,name=PartitionReassignmentRateAndTimeMs,attr=FifteenMinuteRate": 0,
				"kafka.controller:type=ControllerStats,name=LeaderAndIsrResponseReceivedRateAndTimeMs,attr=Min": 0.032214,
				"kafka.controller:type=ControllerStats,name=PartitionReassignmentRateAndTimeMs,attr=50thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=PartitionReassignmentRateAndTimeMs,attr=StdDev": 0,
				"kafka.controller:type=ControllerStats,name=ControlledShutdownRateAndTimeMs,attr=EventType": "calls",
				"kafka.controller:type=ControllerStats,name=IsrChangeRateAndTimeMs,attr=MeanRate": 0,
				"kafka.controller:type=ControllerStats,name=AutoLeaderBalanceRateAndTimeMs,attr=FifteenMinuteRate": 0.0032002778326585424,
				"kafka.controller:type=ControllerStats,name=ManualLeaderBalanceRateAndTimeMs,attr=999thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=LeaderElectionRateAndTimeMs,attr=FifteenMinuteRate": 4.44659081257e-313,
				"kafka.controller:type=ControllerStats,name=PartitionReassignmentRateAndTimeMs,attr=Min": 0,
				"kafka.controller:type=ControllerStats,name=ManualLeaderBalanceRateAndTimeMs,attr=75thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=PartitionReassignmentRateAndTimeMs,attr=99thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=TopicDeletionRateAndTimeMs,attr=Count": 0,
				"kafka.controller:type=ControllerStats,name=ManualLeaderBalanceRateAndTimeMs,attr=Max": 0,
				"kafka.controller:type=ControllerStats,name=LogDirChangeRateAndTimeMs,attr=Mean": 0,
				"kafka.controller:type=ControllerStats,name=PartitionReassignmentRateAndTimeMs,attr=98thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=ControllerChangeRateAndTimeMs,attr=95thPercentile": 88.008219,
				"kafka.controller:type=ControllerStats,name=AutoLeaderBalanceRateAndTimeMs,attr=99thPercentile": 0.36135800999999984,
				"kafka.controller:type=ControllerStats,name=ControlledShutdownRateAndTimeMs,attr=95thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=ControlledShutdownRateAndTimeMs,attr=99thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=IsrChangeRateAndTimeMs,attr=FiveMinuteRate": 0,
				"kafka.controller:type=ControllerStats,name=ControlledShutdownRateAndTimeMs,attr=98thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=ControllerChangeRateAndTimeMs,attr=50thPercentile": 44.0086135,
				"kafka.controller:type=ControllerStats,name=IsrChangeRateAndTimeMs,attr=999thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=PartitionReassignmentRateAndTimeMs,attr=FiveMinuteRate": 0,
				"kafka.controller:type=ControllerStats,name=TopicChangeRateAndTimeMs,attr=99thPercentile": 5.912408,
				"kafka.controller:type=ControllerStats,name=ControllerChangeRateAndTimeMs,attr=Mean": 44.0086135,
				"kafka.controller:type=ControllerStats,name=LeaderAndIsrResponseReceivedRateAndTimeMs,attr=OneMinuteRate": 2.964393875e-314,
				"kafka.controller:type=ControllerStats,name=AutoLeaderBalanceRateAndTimeMs,attr=OneMinuteRate": 0.0008015516515939636,
				"kafka.controller:type=ControllerStats,name=ControllerChangeRateAndTimeMs,attr=Count": 2,
				"kafka.controller:type=ControllerStats,name=TopicChangeRateAndTimeMs,attr=EventType": "calls",
				"kafka.controller:type=ControllerStats,name=AutoLeaderBalanceRateAndTimeMs,attr=50thPercentile": 0.21522249999999998,
				"kafka.controller:type=ControllerStats,name=ManualLeaderBalanceRateAndTimeMs,attr=OneMinuteRate": 0,
				"kafka.controller:type=ControllerStats,name=LeaderAndIsrResponseReceivedRateAndTimeMs,attr=50thPercentile": 0.0365935,
				"kafka.controller:type=ControllerStats,name=LeaderAndIsrResponseReceivedRateAndTimeMs,attr=75thPercentile": 0.0397555,
				"kafka.controller:type=ControllerStats,name=TopicChangeRateAndTimeMs,attr=FifteenMinuteRate": 4.44659081257e-313,
				"kafka.controller:type=ControllerStats,name=LeaderElectionRateAndTimeMs,attr=99thPercentile": 35.367419,
				"kafka.controller:type=ControllerStats,name=LeaderElectionRateAndTimeMs,attr=Mean": 35.367419,
				"kafka.controller:type=ControllerStats,name=AutoLeaderBalanceRateAndTimeMs,attr=75thPercentile": 0.24486875000000002,
				"kafka.controller:type=ControllerStats,name=LeaderElectionRateAndTimeMs,attr=EventType": "calls",
				"kafka.controller:type=ControllerStats,name=LeaderElectionRateAndTimeMs,attr=98thPercentile": 35.367419,
				"kafka.controller:type=ControllerStats,name=ManualLeaderBalanceRateAndTimeMs,attr=Mean": 0,
				"kafka.controller:type=ControllerStats,name=LeaderAndIsrResponseReceivedRateAndTimeMs,attr=98thPercentile": 0.041989,
				"kafka.controller:type=ControllerStats,name=LeaderElectionRateAndTimeMs,attr=75thPercentile": 35.367419,
				"kafka.controller:type=ControllerStats,name=LeaderAndIsrResponseReceivedRateAndTimeMs,attr=99thPercentile": 0.041989,
				"kafka.controller:type=ControllerStats,name=AutoLeaderBalanceRateAndTimeMs,attr=98thPercentile": 0.35689086,
				"kafka.controller:type=ControllerStats,name=LeaderAndIsrResponseReceivedRateAndTimeMs,attr=FifteenMinuteRate": 4.44659081257e-313,
				"kafka.controller:type=ControllerStats,name=IsrChangeRateAndTimeMs,attr=99thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=AutoLeaderBalanceRateAndTimeMs,attr=Mean": 0.24170627486187846,
				"kafka.controller:type=ControllerStats,name=ControllerChangeRateAndTimeMs,attr=FifteenMinuteRate": 4.44659081257e-313,
				"kafka.controller:type=ControllerStats,name=LeaderElectionRateAndTimeMs,attr=95thPercentile": 35.367419,
				"kafka.controller:type=ControllerStats,name=IsrChangeRateAndTimeMs,attr=98thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=LeaderAndIsrResponseReceivedRateAndTimeMs,attr=FiveMinuteRate": 1.4821969375e-313,
				"kafka.controller:type=ControllerStats,name=IsrChangeRateAndTimeMs,attr=75thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=LeaderAndIsrResponseReceivedRateAndTimeMs,attr=EventType": "calls",
				"kafka.controller:type=ControllerStats,name=AutoLeaderBalanceRateAndTimeMs,attr=95thPercentile": 0.2991629499999999,
				"kafka.controller:type=ControllerStats,name=IsrChangeRateAndTimeMs,attr=95thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=LeaderElectionRateAndTimeMs,attr=50thPercentile": 35.367419,
				"kafka.controller:type=ControllerStats,name=TopicDeletionRateAndTimeMs,attr=999thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=IsrChangeRateAndTimeMs,attr=EventType": "calls",
				"kafka.controller:type=ControllerStats,name=IsrChangeRateAndTimeMs,attr=50thPercentile": 0,
				"kafka.controller:type=ControllerStats,name=LeaderAndIsrResponseReceivedRateAndTimeMs,attr=StdDev": 0.39324162938489987,
				"kafka.controller:type=ControllerStats,name=ControllerChangeRateAndTimeMs,attr=FiveMinuteRate": 1.4821969375e-313,
				"kafka.controller:type=ControllerStats,name=TopicDeletionRateAndTimeMs,attr=MeanRate": 0,
				"kafka.controller:type=ControllerStats,name=IsrChangeRateAndTimeMs,attr=OneMinuteRate": 0,
				"kafka.controller:type=ControllerStats,name=LogDirChangeRateAndTimeMs,attr=OneMinuteRate": 0,
				"kafka.controller:type=ControllerStats,name=AutoLeaderBalanceRateAndTimeMs,attr=Max": 3.160458,
				"kafka.controller:type=ControllerStats,name=ManualLeaderBalanceRateAndTimeMs,attr=FifteenMinuteRate": 0,
				"kafka.controller:type=ControllerStats,name=TopicChangeRateAndTimeMs,attr=MeanRate": 1.0360380361506925e-05,
				"kafka.controller:type=ControllerStats,name=ControllerChangeRateAndTimeMs,attr=999thPercentile": 88.008219,
				"kafka.controller:type=ControllerStats,name=ControlledShutdownRateAndTimeMs,attr=MeanRate": 0,
				"kafka.controller:type=ControllerStats,name=LeaderAndIsrResponseReceivedRateAndTimeMs,attr=95thPercentile": 0.041989,
				"kafka.controller:type=ControllerStats,name=ControllerChangeRateAndTimeMs,attr=Min": 0.009008,
				"kafka.controller:type=ControllerStats,name=LeaderElectionRateAndTimeMs,attr=FiveMinuteRate": 1.4821969375e-313
			}`), &result)
			return result, err
		} else if strings.HasPrefix(objectPattern, "kafka.server:type=BrokerTopicMetrics,name=*") {
			output := `
			"kafka.server:type=BrokerTopicMetrics,name=TotalFetchRequestsPerSec,topic=testtopic5,attr=Count": 2,
			"kafka.server:type=BrokerTopicMetrics,name=TotalFetchRequestsPerSec,topic=testtopic5,attr=EventType": "requests",
			"kafka.server:type=BrokerTopicMetrics,name=TotalProduceRequestsPerSec,topic=testtopic5,attr=OneMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=FailedFetchRequestsPerSec,topic=testtopic5,attr=FifteenMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=TotalProduceRequestsPerSec,topic=testtopic5,attr=EventType": "requests",
			"kafka.server:type=BrokerTopicMetrics,name=TotalFetchRequestsPerSec,topic=testtopic5,attr=FifteenMinuteRate": 4.44659081257e-313,
			"kafka.server:type=BrokerTopicMetrics,name=FetchMessageConversionsPerSec,topic=testtopic5,attr=FifteenMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=ProduceMessageConversionsPerSec,topic=testtopic5,attr=OneMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=BytesOutPerSec,topic=testtopic5,attr=MeanRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=BytesInPerSec,topic=testtopic5,attr=EventType": "bytes",
			"kafka.server:type=BrokerTopicMetrics,name=MessagesInPerSec,topic=testtopic5,attr=MeanRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=MessagesInPerSec,topic=testtopic5,attr=Count": 0,
			"kafka.server:type=BrokerTopicMetrics,name=FetchMessageConversionsPerSec,topic=testtopic5,attr=EventType": "requests",
			"kafka.server:type=BrokerTopicMetrics,name=FetchMessageConversionsPerSec,topic=testtopic5,attr=MeanRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=TotalProduceRequestsPerSec,topic=testtopic5,attr=MeanRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=TotalProduceRequestsPerSec,topic=testtopic5,attr=Count": 0,
			"kafka.server:type=BrokerTopicMetrics,name=FailedProduceRequestsPerSec,topic=testtopic5,attr=Count": 0,
			"kafka.server:type=BrokerTopicMetrics,name=BytesInPerSec,topic=testtopic5,attr=FiveMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=TotalProduceRequestsPerSec,topic=testtopic5,attr=FiveMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=FetchMessageConversionsPerSec,topic=testtopic5,attr=FiveMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=BytesRejectedPerSec,topic=testtopic5,attr=EventType": "bytes",
			"kafka.server:type=BrokerTopicMetrics,name=BytesRejectedPerSec,topic=testtopic5,attr=Count": 0,
			"kafka.server:type=BrokerTopicMetrics,name=FailedProduceRequestsPerSec,topic=testtopic5,attr=FifteenMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=FailedFetchRequestsPerSec,topic=testtopic5,attr=MeanRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=FetchMessageConversionsPerSec,topic=testtopic5,attr=Count": 0,
			"kafka.server:type=BrokerTopicMetrics,name=BytesOutPerSec,topic=testtopic5,attr=EventType": "bytes",
			"kafka.server:type=BrokerTopicMetrics,name=ProduceMessageConversionsPerSec,topic=testtopic5,attr=FifteenMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=ProduceMessageConversionsPerSec,topic=testtopic5,attr=FiveMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=ProduceMessageConversionsPerSec,topic=testtopic5,attr=Count": 0,
			"kafka.server:type=BrokerTopicMetrics,name=FailedFetchRequestsPerSec,topic=testtopic5,attr=OneMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=FailedFetchRequestsPerSec,topic=testtopic5,attr=EventType": "requests",
			"kafka.server:type=BrokerTopicMetrics,name=TotalFetchRequestsPerSec,topic=testtopic5,attr=FiveMinuteRate": 1.4821969375e-313,
			"kafka.server:type=BrokerTopicMetrics,name=BytesRejectedPerSec,topic=testtopic5,attr=OneMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=BytesRejectedPerSec,topic=testtopic5,attr=MeanRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=MessagesInPerSec,topic=testtopic5,attr=EventType": "messages",
			"kafka.server:type=BrokerTopicMetrics,name=MessagesInPerSec,topic=testtopic5,attr=OneMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=MessagesInPerSec,topic=testtopic5,attr=FifteenMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=BytesRejectedPerSec,topic=testtopic5,attr=FiveMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=FailedProduceRequestsPerSec,topic=testtopic5,attr=EventType": "requests",
			"kafka.server:type=BrokerTopicMetrics,name=FailedFetchRequestsPerSec,topic=testtopic5,attr=FiveMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=FailedProduceRequestsPerSec,topic=testtopic5,attr=OneMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=BytesOutPerSec,topic=testtopic5,attr=FifteenMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=BytesOutPerSec,topic=testtopic5,attr=FiveMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=BytesInPerSec,topic=testtopic5,attr=Count": 0,
			"kafka.server:type=BrokerTopicMetrics,name=BytesOutPerSec,topic=testtopic5,attr=Count": 0,
			"kafka.server:type=BrokerTopicMetrics,name=BytesRejectedPerSec,topic=testtopic5,attr=FifteenMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=FailedFetchRequestsPerSec,topic=testtopic5,attr=Count": 0,
			"kafka.server:type=BrokerTopicMetrics,name=TotalProduceRequestsPerSec,topic=testtopic5,attr=FifteenMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=FetchMessageConversionsPerSec,topic=testtopic5,attr=OneMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=TotalFetchRequestsPerSec,topic=testtopic5,attr=OneMinuteRate": 2.964393875e-314,
			"kafka.server:type=BrokerTopicMetrics,name=FailedProduceRequestsPerSec,topic=testtopic5,attr=MeanRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=FailedProduceRequestsPerSec,topic=testtopic5,attr=FiveMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=BytesInPerSec,topic=testtopic5,attr=OneMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=BytesInPerSec,topic=testtopic5,attr=MeanRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=ProduceMessageConversionsPerSec,topic=testtopic5,attr=EventType": "requests",
			"kafka.server:type=BrokerTopicMetrics,name=BytesInPerSec,topic=testtopic5,attr=FifteenMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=BytesOutPerSec,topic=testtopic5,attr=OneMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=ProduceMessageConversionsPerSec,topic=testtopic5,attr=MeanRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=MessagesInPerSec,topic=testtopic5,attr=FiveMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=TotalFetchRequestsPerSec,topic=testtopic5,attr=MeanRate": 2.565954606076927e-06,`

			resultString := ""
			for i := 0; i < numTopics; i++ {
				newResultString := output
				newResultString = strings.Replace(newResultString, "testtopic5", "topic"+strconv.Itoa(i), 1)
				resultString += newResultString
			}
			resultString = "{" + resultString[0:len(resultString)-1] + "}"
			err := json.Unmarshal([]byte(resultString), &result)
			return result, err
		} else if strings.HasPrefix(objectPattern, "kafka.log:type=LogFlushStats,name=*") {
			err := json.Unmarshal([]byte("{}"), &result)
			return result, err
		} else if strings.HasPrefix(objectPattern, "kafka.server:type=KafkaRequestHandlerPool,name=RequestHandlerAvgIdlePercent") {
			err := json.Unmarshal([]byte(`{
				"kafka.server:type=KafkaRequestHandlerPool,name=RequestHandlerAvgIdlePercent,attr=Count": 869146551721145,
				"kafka.server:type=KafkaRequestHandlerPool,name=RequestHandlerAvgIdlePercent,attr=FiveMinuteRate": 0.9999226954608703,
				"kafka.server:type=KafkaRequestHandlerPool,name=RequestHandlerAvgIdlePercent,attr=EventType": "percent",
				"kafka.server:type=KafkaRequestHandlerPool,name=RequestHandlerAvgIdlePercent,attr=MeanRate": 0.9999933053026674,
				"kafka.server:type=KafkaRequestHandlerPool,name=RequestHandlerAvgIdlePercent,attr=FifteenMinuteRate": 0.9999732347527868,
				"kafka.server:type=KafkaRequestHandlerPool,name=RequestHandlerAvgIdlePercent,attr=OneMinuteRate": 0.999596701808481
			}`), &result)
			return result, err
		} else if strings.HasPrefix(objectPattern, "kafka.server:type=DelayedFetchMetrics,name=ExpiresPerSec,fetcherType=*") {
			err := json.Unmarshal([]byte(`{
				"kafka.server:type=DelayedFetchMetrics,name=ExpiresPerSec,fetcherType=follower,attr=EventType": "requests",
				"kafka.server:type=DelayedFetchMetrics,name=ExpiresPerSec,fetcherType=consumer,attr=FiveMinuteRate": 1.4821969375e-313,
				"kafka.server:type=DelayedFetchMetrics,name=ExpiresPerSec,fetcherType=consumer,attr=Count": 11780,
				"kafka.server:type=DelayedFetchMetrics,name=ExpiresPerSec,fetcherType=follower,attr=MeanRate": 0,
				"kafka.server:type=DelayedFetchMetrics,name=ExpiresPerSec,fetcherType=follower,attr=Count": 0,
				"kafka.server:type=DelayedFetchMetrics,name=ExpiresPerSec,fetcherType=follower,attr=FifteenMinuteRate": 0,
				"kafka.server:type=DelayedFetchMetrics,name=ExpiresPerSec,fetcherType=consumer,attr=OneMinuteRate": 2.964393875e-314,
				"kafka.server:type=DelayedFetchMetrics,name=ExpiresPerSec,fetcherType=consumer,attr=FifteenMinuteRate": 4.44659081257e-313,
				"kafka.server:type=DelayedFetchMetrics,name=ExpiresPerSec,fetcherType=follower,attr=OneMinuteRate": 0,
				"kafka.server:type=DelayedFetchMetrics,name=ExpiresPerSec,fetcherType=follower,attr=FiveMinuteRate": 0,
				"kafka.server:type=DelayedFetchMetrics,name=ExpiresPerSec,fetcherType=consumer,attr=MeanRate": 0.013553251593636828,
				"kafka.server:type=DelayedFetchMetrics,name=ExpiresPerSec,fetcherType=consumer,attr=EventType": "requests"
			}`), &result)
			return result, err
		} else if strings.HasPrefix(objectPattern, "kafka.server:type=BrokerTopicMetrics,name=BytesInPerSec,topic=") {
			output := `
			"kafka.server:type=BrokerTopicMetrics,name=BytesInPerSec,topic=testtopic5,attr=MeanRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=BytesInPerSec,topic=testtopic5,attr=EventType": "bytes",
			"kafka.server:type=BrokerTopicMetrics,name=BytesInPerSec,topic=testtopic5,attr=FiveMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=BytesInPerSec,topic=testtopic5,attr=Count": 0,
			"kafka.server:type=BrokerTopicMetrics,name=BytesInPerSec,topic=testtopic5,attr=OneMinuteRate": 0,
			"kafka.server:type=BrokerTopicMetrics,name=BytesInPerSec,topic=testtopic5,attr=FifteenMinuteRate": 0,`
			resultString := ""
			for i := 0; i < numTopics; i++ {
				newResultString := output
				newResultString = strings.Replace(newResultString, "testtopic5", "topic"+strconv.Itoa(i), 1)
				resultString += newResultString
			}
			resultString = "{" + resultString[0:len(resultString)-1] + "}"
			err := json.Unmarshal([]byte(resultString), &result)
			return result, err
		} else if strings.HasPrefix(objectPattern, "kafka.log:type=Log,name=Size,topic=") {

			output := `"kafka.log:type=Log,name=Size,topic=testtopic5,partition=0,attr=Value": 0`
			resultString := ""
			for i := 0; i < numTopics; i++ {
				for j := 0; j < numPartitions; j++ {
					newResultString := output
					newResultString = strings.Replace(newResultString, "testtopic5", "topic"+strconv.Itoa(i), 1)
					newResultString = strings.Replace(newResultString, "partition=0", "partition="+strconv.Itoa(i), 1)
					resultString += newResultString
				}
			}
			resultString = "{" + resultString[0:len(resultString)-1] + "}"
			err := json.Unmarshal([]byte(`{}`), &result)
			return result, err
		}

		return nil, nil

	}

}

func Benchmark_1b1t1p(b *testing.B) {
	benchCoreCollection(b, 1, 1, 1)
}
func Benchmark_10b1t1p(b *testing.B) {
	benchCoreCollection(b, 10, 1, 1)
}
func Benchmark_100b1t1p(b *testing.B) {
	benchCoreCollection(b, 100, 1, 1)
}
func Benchmark_1b10t1p(b *testing.B) {
	benchCoreCollection(b, 1, 10, 1)
}
func Benchmark_10b10t1p(b *testing.B) {
	benchCoreCollection(b, 10, 10, 1)
}
func Benchmark_100b10t1p(b *testing.B) {
	benchCoreCollection(b, 100, 10, 1)
}
func Benchmark_1b100t1p(b *testing.B) {
	benchCoreCollection(b, 1, 100, 1)
}
func Benchmark_10b100t1p(b *testing.B) {
	benchCoreCollection(b, 10, 100, 1)
}
func Benchmark_100b100t1p(b *testing.B) {
	benchCoreCollection(b, 100, 100, 1)
}
func Benchmark_1b1t10p(b *testing.B) {
	benchCoreCollection(b, 1, 1, 10)
}
func Benchmark_10b1t10p(b *testing.B) {
	benchCoreCollection(b, 10, 1, 10)
}
func Benchmark_100b1t10p(b *testing.B) {
	benchCoreCollection(b, 100, 1, 10)
}
func Benchmark_1b10t10p(b *testing.B) {
	benchCoreCollection(b, 1, 10, 10)
}
func Benchmark_10b10t10p(b *testing.B) {
	benchCoreCollection(b, 10, 10, 10)
}
func Benchmark_100b10t10p(b *testing.B) {
	benchCoreCollection(b, 100, 10, 10)
}
func Benchmark_1b100t10p(b *testing.B) {
	benchCoreCollection(b, 1, 100, 10)
}
func Benchmark_10b100t10p(b *testing.B) {
	benchCoreCollection(b, 10, 100, 10)
}
func Benchmark_100b100t10p(b *testing.B) {
	benchCoreCollection(b, 100, 100, 10)
}
func Benchmark_1b1t100p(b *testing.B) {
	benchCoreCollection(b, 1, 1, 100)
}
func Benchmark_10b1t100p(b *testing.B) {
	benchCoreCollection(b, 10, 1, 100)
}
func Benchmark_100b1t100p(b *testing.B) {
	benchCoreCollection(b, 100, 1, 100)
}
func Benchmark_1b10t100p(b *testing.B) {
	benchCoreCollection(b, 1, 10, 100)
}
func Benchmark_10b10t100p(b *testing.B) {
	benchCoreCollection(b, 10, 10, 100)
}
func Benchmark_100b10t100p(b *testing.B) {
	benchCoreCollection(b, 100, 10, 100)
}
func Benchmark_1b100t100p(b *testing.B) {
	benchCoreCollection(b, 1, 100, 100)
}
func Benchmark_10b100t100p(b *testing.B) {
	benchCoreCollection(b, 10, 100, 100)
}
func Benchmark_100b100t100p(b *testing.B) {
	benchCoreCollection(b, 100, 100, 100)
}
