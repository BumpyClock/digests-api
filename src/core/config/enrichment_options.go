// ABOUTME: Enrichment configuration for service-level control of optional features
// ABOUTME: Provides configuration options independent of HTTP request structures

package config

// EnrichmentConfig controls which enrichment features are enabled
type EnrichmentConfig struct {
	// ExtractMetadata controls whether to extract metadata from URLs
	ExtractMetadata bool
	
	// ExtractColors controls whether to extract colors from images
	ExtractColors bool
}

// DefaultEnrichmentConfig returns the default configuration with all features enabled
func DefaultEnrichmentConfig() EnrichmentConfig {
	return EnrichmentConfig{
		ExtractMetadata: true,
		ExtractColors:   true,
	}
}

// EnrichmentOption is a functional option for configuring enrichment
type EnrichmentOption func(*EnrichmentConfig)

// WithMetadata enables or disables metadata extraction
func WithMetadata(enabled bool) EnrichmentOption {
	return func(c *EnrichmentConfig) {
		c.ExtractMetadata = enabled
	}
}

// WithColors enables or disables color extraction
func WithColors(enabled bool) EnrichmentOption {
	return func(c *EnrichmentConfig) {
		c.ExtractColors = enabled
	}
}

// WithoutMetadata disables metadata extraction
func WithoutMetadata() EnrichmentOption {
	return WithMetadata(false)
}

// WithoutColors disables color extraction
func WithoutColors() EnrichmentOption {
	return WithColors(false)
}

// NewEnrichmentConfig creates a new enrichment configuration with the given options
func NewEnrichmentConfig(opts ...EnrichmentOption) EnrichmentConfig {
	config := DefaultEnrichmentConfig()
	
	for _, opt := range opts {
		opt(&config)
	}
	
	return config
}