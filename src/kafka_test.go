package main

import (
	"reflect"
	"testing"
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
