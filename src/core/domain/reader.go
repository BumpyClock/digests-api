// ABOUTME: Domain models and types for reader view functionality
// ABOUTME: Defines the structure for extracted article content

package domain

// ReaderView represents extracted article content from a webpage
type ReaderView struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Content     string `json:"content"`         // HTML content
	Markdown    string `json:"markdown"`        // Markdown content
	TextContent string `json:"textContent"`     // Plain text content
	SiteName    string `json:"siteName"`
	Image       string `json:"image"`
	Favicon     string `json:"favicon"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
}