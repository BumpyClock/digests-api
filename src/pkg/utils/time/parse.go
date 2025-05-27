// ABOUTME: Time parsing utilities for flexible date/time parsing
// ABOUTME: Handles various time formats commonly found in RSS/Atom feeds

package time

import (
	"strings"
	"time"
)

// Common time formats found in RSS/Atom feeds
var timeFormats = []string{
	time.RFC3339,
	time.RFC3339Nano,
	time.RFC1123,
	time.RFC1123Z,
	time.RFC822,
	time.RFC822Z,
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
	"02 Jan 2006 15:04:05 MST",
	"02 Jan 2006 15:04:05 -0700",
	"Mon, 02 Jan 2006 15:04:05 MST",
	"Mon, 02 Jan 2006 15:04:05 -0700",
	"Mon, 2 Jan 2006 15:04:05 MST",
	"Mon, 2 Jan 2006 15:04:05 -0700",
}

// ParseFlexibleTime attempts to parse a time string using various formats
func ParseFlexibleTime(timeStr string) time.Time {
	if timeStr == "" {
		return time.Time{}
	}
	
	// Clean up the time string
	timeStr = strings.TrimSpace(timeStr)
	
	// Try each format
	for _, format := range timeFormats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t
		}
	}
	
	return time.Time{}
}

// ParseWithDefault attempts to parse a time string, returning a default if parsing fails
func ParseWithDefault(timeStr string, defaultTime time.Time) time.Time {
	if parsed := ParseFlexibleTime(timeStr); !parsed.IsZero() {
		return parsed
	}
	return defaultTime
}

// ParseWithNow attempts to parse a time string, returning current time if parsing fails
func ParseWithNow(timeStr string) time.Time {
	return ParseWithDefault(timeStr, time.Now())
}