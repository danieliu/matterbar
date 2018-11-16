package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func helperLoadJsonFile(t *testing.T, name string) *Rollbar {
	path := filepath.Join("testdata", name)

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var rollbar Rollbar

	if err := json.Unmarshal(bytes, &rollbar); err != nil {
		t.Fatal(err)
	}

	return &rollbar
}

type InterpolateMessageTest struct {
	filename string
	template string
	expected string
}

func TestInterpolateMessage(t *testing.T) {
	everyOccurrenceMessage := "new_item - [production] - error"
	highOccurrenceMessage := "item_velocity - [production] - error"
	newItemMessage := "new_item - [production] - error"
	tenNthMessage := "exp_repeat_item - [production] - error"

	testcases := []InterpolateMessageTest{
		{"new_item.json", "", ""},
		{"every_occurrence.json", DefaultTemplate, everyOccurrenceMessage},
		{"high_occurrence_rate.json", DefaultTemplate, highOccurrenceMessage},
		{"new_item.json", DefaultTemplate, newItemMessage},
		{"ten_nth.json", DefaultTemplate, tenNthMessage},
	}

	for _, test := range testcases {
		rollbar := helperLoadJsonFile(t, test.filename)
		actual, err := rollbar.interpolateMessage(test.template)
		if err != nil {
			t.Errorf("Failed to interpolate message. %s", err)
		}

		if actual != test.expected {
			t.Errorf("Expected: %s\nActual: %s", test.expected, actual)
		}
	}
}

type EventNameToTitleTest struct {
	data string
	expected string
}

func TestEventNameToTitle(t *testing.T) {
	data := `{"event_name": "%s", "data": {"item": {"last_occurrence": {"level": "error"}}, "trigger": {"threshold": 10, "window_size_description": "5 minutes"}, "occurrences": 10}}`
	testcases := []EventNameToTitleTest{
		{fmt.Sprintf(data, "new_item"), "New Error"},
		{fmt.Sprintf(data, "reactivated_item"), "Reactivated Error"},
		{fmt.Sprintf(data, "reopened_item"), "Reopened Error"},
		{fmt.Sprintf(data, "resolved_item"), "Resolved Error"},
		{fmt.Sprintf(data, "exp_repeat_item"), "10th Error"},
		{fmt.Sprintf(data, "item_velocity"), "10 occurrences in 5 minutes"},
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
