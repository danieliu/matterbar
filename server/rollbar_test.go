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
