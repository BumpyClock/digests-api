// ABOUTME: Request DTOs for feed-related API endpoints
// ABOUTME: Provides validation and default values for incoming requests

package requests

// ParseFeedsRequest represents the request body for parsing multiple feeds
type ParseFeedsRequest struct {
	// URLs is the list of feed URLs to parse
	URLs []string `json:"urls" minItems:"1" maxItems:"100" doc:"List of feed URLs to parse"`
	
	// Page is the page number for pagination (1-based)
	Page int `json:"page,omitempty" minimum:"1" default:"1" doc:"Page number (1-based)"`
	
	// ItemsPerPage is the number of items per page
	ItemsPerPage int `json:"items_per_page,omitempty" minimum:"1" maximum:"100" default:"50" doc:"Number of items per page"`
}

// ApplyDefaults sets default values for optional fields
func (r *ParseFeedsRequest) ApplyDefaults() {
	if r.Page == 0 {
		r.Page = 1
	}
	if r.ItemsPerPage == 0 {
		r.ItemsPerPage = 50
	}
}

// SingleFeedRequest represents the request for parsing a single feed
type SingleFeedRequest struct {
	// URL is the feed URL to parse
	URL string `json:"url" required:"true" format:"uri" doc:"Feed URL to parse"`
	
	// Page is the page number for pagination (1-based)
	Page int `json:"page,omitempty" minimum:"1" default:"1" doc:"Page number (1-based)"`
	
	// ItemsPerPage is the number of items per page
	ItemsPerPage int `json:"items_per_page,omitempty" minimum:"1" maximum:"100" default:"50" doc:"Number of items per page"`
}

// ApplyDefaults sets default values for optional fields
func (r *SingleFeedRequest) ApplyDefaults() {
	if r.Page == 0 {
		r.Page = 1
	}
	if r.ItemsPerPage == 0 {
		r.ItemsPerPage = 50
	}
}