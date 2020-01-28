package topic

var (
	partitionState = []byte(`{"controller_epoch":9,"leader":2,"version":1,"leader_epoch":0,"isr":[2,0,1]}`)
	topicState     = []byte(`{"version":1,"partitions":{"0":[1,2,0]}}`)
)
