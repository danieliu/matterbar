package main

import (
	"testing"
)

func TestTruncateString(t *testing.T) {
	for name, test := range map[string]struct {
		Input    string
		Length   int
		Expected string
	}{
		"Empty string": {
			Input:    "",
			Length:   3,
			Expected: "",
		},
		"String shorter than length": {
			Input:    "ab",
			Length:   3,
			Expected: "ab",
		},
		"String equal to length": {
			Input:    "abc",
			Length:   3,
			Expected: "abc",
		},
		"String longer than length": {
			Input:    "abcd",
			Length:   3,
			Expected: "abc...",
		},
	} {
		t.Run(name, func(t *testing.T) {
			result := TruncateString(test.Input, test.Length)
			if result != test.Expected {
				t.Errorf("Expected: %s\nActual: %s", test.Expected, result)
			}
		})
	}
}
