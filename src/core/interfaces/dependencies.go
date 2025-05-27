// ABOUTME: Dependencies container provides dependency injection for core services
// ABOUTME: Defines the contract for dependencies required by the core business logic

package interfaces

// Dependencies holds all external dependencies required by the core business logic
type Dependencies struct {
	// Cache provides caching functionality
	Cache Cache

	// HTTPClient provides HTTP request functionality
	HTTPClient HTTPClient

	// Logger provides structured logging
	Logger Logger
}