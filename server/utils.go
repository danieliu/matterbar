package main

import (
	"fmt"
)

// TruncateString shortens an input string to a given length and adds ellipsis to
// denote that the string has been shortened. Used specifically for shortening the
// post fallback and message strings.
func TruncateString(input string, length int) string {
	result := input
	if len(result) > length {
		result = fmt.Sprintf("%s...", result[:length])
	}

	return result
}
