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
	
	// EnrichmentOptions controls which enrichment features are enabled
	EnrichmentOptions *EnrichmentOptions `json:"enrichment,omitempty" doc:"Optional enrichment configuration"`
}

// EnrichmentOptions controls which optional enrichment features are enabled
type EnrichmentOptions struct {
	// ExtractMetadata enables metadata extraction from URLs (default: true)
	ExtractMetadata *bool `json:"extract_metadata,omitempty" default:"true" doc:"Extract metadata from article URLs"`
	
	// ExtractColors enables color extraction from images (default: true)
	ExtractColors *bool `json:"extract_colors,omitempty" default:"true" doc:"Extract dominant colors from images"`
}

// ApplyDefaults sets default values for optional fields
func (r *ParseFeedsRequest) ApplyDefaults() {
	if r.Page == 0 {
		r.Page = 1
	}
	if r.ItemsPerPage == 0 {
		r.ItemsPerPage = 50
	}
	
	// Set default enrichment options if not provided
	if r.EnrichmentOptions == nil {
		r.EnrichmentOptions = &EnrichmentOptions{}
	}
	if r.EnrichmentOptions.ExtractMetadata == nil {
		enabled := true
		r.EnrichmentOptions.ExtractMetadata = &enabled
	}
	if r.EnrichmentOptions.ExtractColors == nil {
		enabled := true
		r.EnrichmentOptions.ExtractColors = &enabled
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
	
	// EnrichmentOptions controls which enrichment features are enabled
	EnrichmentOptions *EnrichmentOptions `json:"enrichment,omitempty" doc:"Optional enrichment configuration"`
}

// ApplyDefaults sets default values for optional fields
func (r *SingleFeedRequest) ApplyDefaults() {
	if r.Page == 0 {
		r.Page = 1
	}
	if r.ItemsPerPage == 0 {
		r.ItemsPerPage = 50
	}
	
	// Set default enrichment options if not provided
	if r.EnrichmentOptions == nil {
		r.EnrichmentOptions = &EnrichmentOptions{}
	}
	if r.EnrichmentOptions.ExtractMetadata == nil {
		enabled := true
		r.EnrichmentOptions.ExtractMetadata = &enabled
	}
	if r.EnrichmentOptions.ExtractColors == nil {
		enabled := true
		r.EnrichmentOptions.ExtractColors = &enabled
	}
}