package main

import (
	"encoding/json"
	"fmt"
	"testing"
)

type EventNameToTitleTest struct {
	data     string
	expected string
}

func TestEventNameToTitle(t *testing.T) {
	data := `{"event_name": "%s", "data": {"item": {"last_occurrence": {"level": "error"}}, "trigger": {"threshold": 10, "window_size_description": "5 minutes"}, "occurrences": 10}}`
	occurrenceSpecific := `{"event_name": "%s", "data": {"item": {"level": 40}, "occurrence": {"level": "error"}}, "trigger": {"threshold": 10, "window_size_description": "5 minutes"}, "occurrences": 10}`
	testcases := []EventNameToTitleTest{
		{fmt.Sprintf(data, "new_item"), "New Error"},
		{fmt.Sprintf(data, "reactivated_item"), "Reactivated Error"},
		{fmt.Sprintf(data, "reopened_item"), "Reopened Error"},
		{fmt.Sprintf(data, "resolved_item"), "Resolved Error"},
		{fmt.Sprintf(data, "exp_repeat_item"), "10th Error"},
		{fmt.Sprintf(data, "item_velocity"), "10 occurrences in 5 minutes"},
		{fmt.Sprintf(occurrenceSpecific, "occurrence"), "Occurrence - Error"},
	}

	for _, test := range testcases {
		var rollbar Rollbar

		err := json.Unmarshal([]byte(test.data), &rollbar)
		if err != nil {
			t.Fatal(err)
		}

		actual := rollbar.eventNameToTitle()
		if actual != test.expected {
			t.Errorf("Expected: %s\nActual: %s", test.expected, actual)
		}
	}
}
