// ABOUTME: Request DTOs for reader view API endpoints
// ABOUTME: Defines the structure for reader view extraction requests

package requests

// ReaderViewRequest represents a request to extract reader views from URLs
type ReaderViewRequest struct {
	// URLs to extract reader views from
	URLs []string `json:"urls" required:"true" example:"[\"https://example.com/article\"]" doc:"List of URLs to extract reader views from"`
}