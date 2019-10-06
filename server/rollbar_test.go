package main

import (
	"encoding/json"
	"testing"
)

type EventNameToTitleTest struct {
	data     string
	expected string
}

func TestEventNameToTitle(t *testing.T) {
	for name, test := range map[string]struct {
		TestFile      string
		ExpectedTitle string
	}{
		"ok - new item": {
			TestFile:      "new_item.json",
			ExpectedTitle: "New Error",
		},
		"ok - reactivated item": {
			TestFile:      "reactivated_item.json",
			ExpectedTitle: "Reactivated Error",
		},
		"ok - exp repeat item": {
			TestFile:      "exp_repeat_item.json",
			ExpectedTitle: "10th Error",
		},
		"ok - reopened": {
			TestFile:      "reopened_item.json",
			ExpectedTitle: "Reopened Error",
		},
		"ok - resolved": {
			TestFile:      "resolved_item.json",
			ExpectedTitle: "Resolved Error",
		},
		"ok - occurrence": {
			TestFile:      "occurrence.json",
			ExpectedTitle: "Occurrence - Error",
		},
		"ok - high velocity": {
			TestFile:      "item_velocity.json",
			ExpectedTitle: "5 occurrences in 5 minutes",
		},
	} {
		t.Run(name, func(t *testing.T) {
			var rollbar Rollbar
			data := loadJsonFile(t, test.TestFile)
			err := json.Unmarshal([]byte(data), &rollbar)
			if err != nil {
				t.Fatal(err)
			}
			actualTitle := rollbar.eventNameToTitle()
			if actualTitle != test.ExpectedTitle {
				t.Errorf("Expected: %s\nActual: %s", test.ExpectedTitle, actualTitle)
			}
		})
	}

}
