package main

import (
	"reflect"
	"testing"

	"github.com/newrelic/nri-kafka/src/args"
	"github.com/stretchr/testify/assert"
)

func Test_enforceTopicLimit(t *testing.T) {
	overLimit := make([]string, maxTopics+1)

	for i := 0; i < maxTopics+1; i++ {
		overLimit[i] = "topic"
	}

	testCases := []struct {
		name   string
		topics []string
		want   []string
	}{
		{
			"Over Limit",
			overLimit,
			[]string{},
		},
		{
			"Under Limit",
			[]string{"topic"},
			[]string{"topic"},
		},
	}

	for _, tc := range testCases {
		out := enforceTopicLimit(tc.topics)
		if !reflect.DeepEqual(out, tc.want) {
			t.Errorf("Test Case %s Failed: Expected '%+v' got '%+v'", tc.name, tc.want, out)
		}
	}
}

func Test_filterTopicsByBucket(t *testing.T) {
	initialTopics := []string{"a", "b", "c", "d", "e", "f", "g", "h"}

	expectedTopics1 := []string{"c", "f"}
	filteredTopics1 := filterTopicsByBucket(initialTopics, args.TopicBucket{1, 3})
	assert.Equal(t, expectedTopics1, filteredTopics1)

	expectedTopics2 := []string{"b", "e", "h"}
	filteredTopics2 := filterTopicsByBucket(initialTopics, args.TopicBucket{2, 3})
	assert.Equal(t, expectedTopics2, filteredTopics2)

	expectedTopics3 := []string{"a", "d", "g"}
	filteredTopics3 := filterTopicsByBucket(initialTopics, args.TopicBucket{3, 3})
	assert.Equal(t, expectedTopics3, filteredTopics3)
}
