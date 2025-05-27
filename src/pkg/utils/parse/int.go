// ABOUTME: Utility functions for parsing integers from strings
// ABOUTME: Provides safe parsing with default values

package parse

import "strconv"

// IntOrZero safely parses an integer from a string, returning 0 if parsing fails
func IntOrZero(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}