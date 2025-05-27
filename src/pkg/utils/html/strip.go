// ABOUTME: HTML utilities for stripping tags and decoding entities
// ABOUTME: Provides common HTML processing functions used across the application

package html

import (
	"strings"
)

// StripHTML removes HTML tags and decodes common entities from a string
func StripHTML(html string) string {
	// Simple HTML tag removal - in production you'd use a proper HTML parser
	// This is a simplified version
	text := html
	
	// Remove script and style content
	text = strings.ReplaceAll(text, "<script>", "<script><!--")
	text = strings.ReplaceAll(text, "</script>", "--></script>")
	text = strings.ReplaceAll(text, "<style>", "<style><!--")
	text = strings.ReplaceAll(text, "</style>", "--></style>")
	
	// Remove HTML tags
	for strings.Contains(text, "<") && strings.Contains(text, ">") {
		start := strings.Index(text, "<")
		end := strings.Index(text, ">")
		if start < end && start >= 0 && end >= 0 {
			text = text[:start] + " " + text[end+1:]
		} else {
			break
		}
	}
	
	// Decode common HTML entities
	text = DecodeEntities(text)
	
	// Clean up whitespace
	text = strings.TrimSpace(text)
	
	// Replace multiple spaces with single space
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}
	
	return text
}

// DecodeEntities decodes common HTML entities
func DecodeEntities(text string) string {
	replacements := map[string]string{
		"&nbsp;":   " ",
		"&amp;":    "&",
		"&lt;":     "<",
		"&gt;":     ">",
		"&quot;":   "\"",
		"&#39;":    "'",
		"&apos;":   "'",
		"&#8230;":  "...",
		"&#8217;":  "'",
		"&#8220;":  "\"",
		"&#8221;":  "\"",
		"&ldquo;":  "\"",
		"&rdquo;":  "\"",
		"&lsquo;":  "'",
		"&rsquo;":  "'",
		"&mdash;":  "-",
		"&ndash;":  "-",
		"&hellip;": "...",
		"&copy;":   "(c)",
		"&reg;":    "(R)",
		"&trade;":  "(TM)",
	}
	
	result := text
	for entity, replacement := range replacements {
		result = strings.ReplaceAll(result, entity, replacement)
	}
	
	return result
}