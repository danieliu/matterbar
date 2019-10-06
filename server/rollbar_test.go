package main

import (
	"encoding/json"
	"testing"
)

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

func TestEventText(t *testing.T) {
	for name, test := range map[string]struct {
		TestFile     string
		ExpectedText string
	}{
		"ok - new item exception data under last_occurrence": {
			TestFile:     "new_item.json",
			ExpectedText: "TypeError: unsupported operand type(s) for +=: 'int' and 'str'",
		},
		"ok - new item log message no traceback": {
			TestFile:     "new_item_log_message.json",
			ExpectedText: "User 8563892 is missing permissions",
		},
		"ok - occurrence exception data under occurrence": {
			TestFile:     "occurrence.json",
			ExpectedText: "TypeError: 'NoneType' object has no attribute '__getitem__'",
		},
		"ok - high velocity missing occurrence data": {
			TestFile:     "item_velocity.json",
			ExpectedText: "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			var rollbar Rollbar
			data := loadJsonFile(t, test.TestFile)
			err := json.Unmarshal([]byte(data), &rollbar)
			if err != nil {
				t.Fatal(err)
			}
			actualText := rollbar.eventText()
			if test.ExpectedText != actualText {
				t.Errorf("Expected: %s\nActual: %s", test.ExpectedText, actualText)
			}
		})
	}
}
