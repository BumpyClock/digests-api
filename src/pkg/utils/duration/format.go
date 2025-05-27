// ABOUTME: Duration formatting utilities for converting between different duration representations
// ABOUTME: Handles conversion between seconds, duration strings, and formatted time strings

package duration

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseToSeconds converts various duration formats to seconds string
func ParseToSeconds(durationStr string) string {
	if durationStr == "" {
		return ""
	}
	
	// If already a number, assume it's seconds
	if _, err := strconv.Atoi(durationStr); err == nil {
		return durationStr
	}
	
	// Try parsing as Go duration (e.g., "1h30m")
	if dur, err := time.ParseDuration(durationStr); err == nil {
		return strconv.Itoa(int(dur.Seconds()))
	}
	
	// Try parsing HH:MM:SS or MM:SS format
	parts := strings.Split(durationStr, ":")
	switch len(parts) {
	case 3: // HH:MM:SS
		hours, _ := strconv.Atoi(parts[0])
		minutes, _ := strconv.Atoi(parts[1])
		seconds, _ := strconv.Atoi(parts[2])
		return strconv.Itoa(hours*3600 + minutes*60 + seconds)
	case 2: // MM:SS
		minutes, _ := strconv.Atoi(parts[0])
		seconds, _ := strconv.Atoi(parts[1])
		return strconv.Itoa(minutes*60 + seconds)
	}
	
	return durationStr
}

// FormatSeconds converts seconds to HH:MM:SS or MM:SS format
func FormatSeconds(secondsStr string) string {
	if secondsStr == "" {
		return ""
	}
	
	seconds, err := strconv.Atoi(secondsStr)
	if err != nil {
		// If it's already formatted or can't parse, return as-is
		return secondsStr
	}
	
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	
	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
	}
	return fmt.Sprintf("%02d:%02d", minutes, secs)
}

// SecondsToHumanReadable converts seconds to a human-readable format
func SecondsToHumanReadable(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%d seconds", seconds)
	}
	
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	
	parts := []string{}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%d hour", hours))
		if hours > 1 {
			parts[len(parts)-1] += "s"
		}
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%d minute", minutes))
		if minutes > 1 {
			parts[len(parts)-1] += "s"
		}
	}
	
	return strings.Join(parts, " ")
}