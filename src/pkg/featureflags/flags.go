// ABOUTME: Feature flag management for gradual rollout and A/B testing
// ABOUTME: Provides interface-based feature toggling with multiple backends

package featureflags

import (
	"context"
	"os"
	"strings"
	"sync"
)

// FeatureFlag represents a single feature flag
type FeatureFlag string

// Defined feature flags
const (
	// NewFeedParser enables the new feed parsing implementation
	NewFeedParser FeatureFlag = "new_feed_parser"
	
	// SearchEnabled enables the search functionality
	SearchEnabled FeatureFlag = "search_enabled"
	
	// ShareEnabled enables the share functionality
	ShareEnabled FeatureFlag = "share_enabled"
	
	// MetricsEnabled enables the metrics endpoint
	MetricsEnabled FeatureFlag = "metrics_enabled"
	
	// RateLimitEnabled enables rate limiting
	RateLimitEnabled FeatureFlag = "rate_limit_enabled"
	
	// CacheEnabled enables caching
	CacheEnabled FeatureFlag = "cache_enabled"
)

// Manager defines the interface for feature flag management
type Manager interface {
	// IsEnabled checks if a feature flag is enabled
	IsEnabled(ctx context.Context, flag FeatureFlag) bool
	
	// IsEnabledForUser checks if a feature is enabled for a specific user
	IsEnabledForUser(ctx context.Context, flag FeatureFlag, userID string) bool
	
	// SetEnabled sets a feature flag's state (for testing)
	SetEnabled(flag FeatureFlag, enabled bool)
	
	// GetAllFlags returns the state of all flags
	GetAllFlags() map[FeatureFlag]bool
}

// EnvManager implements Manager using environment variables
type EnvManager struct {
	mu        sync.RWMutex
	overrides map[FeatureFlag]bool
	prefix    string
}

// NewEnvManager creates a new environment-based feature flag manager
func NewEnvManager(prefix string) *EnvManager {
	if prefix == "" {
		prefix = "FEATURE_"
	}
	return &EnvManager{
		overrides: make(map[FeatureFlag]bool),
		prefix:    prefix,
	}
}

// IsEnabled checks if a feature flag is enabled
func (m *EnvManager) IsEnabled(ctx context.Context, flag FeatureFlag) bool {
	m.mu.RLock()
	if enabled, ok := m.overrides[flag]; ok {
		m.mu.RUnlock()
		return enabled
	}
	m.mu.RUnlock()
	
	// Check environment variable
	envKey := m.prefix + strings.ToUpper(string(flag))
	value := os.Getenv(envKey)
	
	return strings.ToLower(value) == "true" || value == "1" || strings.ToLower(value) == "enabled"
}

// IsEnabledForUser checks if a feature is enabled for a specific user
// For EnvManager, this is the same as IsEnabled (no per-user control)
func (m *EnvManager) IsEnabledForUser(ctx context.Context, flag FeatureFlag, userID string) bool {
	return m.IsEnabled(ctx, flag)
}

// SetEnabled sets a feature flag's state (mainly for testing)
func (m *EnvManager) SetEnabled(flag FeatureFlag, enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.overrides[flag] = enabled
}

// GetAllFlags returns the state of all defined flags
func (m *EnvManager) GetAllFlags() map[FeatureFlag]bool {
	ctx := context.Background()
	flags := map[FeatureFlag]bool{
		NewFeedParser:    m.IsEnabled(ctx, NewFeedParser),
		SearchEnabled:    m.IsEnabled(ctx, SearchEnabled),
		ShareEnabled:     m.IsEnabled(ctx, ShareEnabled),
		MetricsEnabled:   m.IsEnabled(ctx, MetricsEnabled),
		RateLimitEnabled: m.IsEnabled(ctx, RateLimitEnabled),
		CacheEnabled:     m.IsEnabled(ctx, CacheEnabled),
	}
	return flags
}

// StaticManager implements Manager with static configuration
type StaticManager struct {
	flags map[FeatureFlag]bool
	mu    sync.RWMutex
}

// NewStaticManager creates a manager with predefined flag states
func NewStaticManager(flags map[FeatureFlag]bool) *StaticManager {
	if flags == nil {
		flags = make(map[FeatureFlag]bool)
	}
	return &StaticManager{
		flags: flags,
	}
}

// IsEnabled checks if a feature flag is enabled
func (m *StaticManager) IsEnabled(ctx context.Context, flag FeatureFlag) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.flags[flag]
}

// IsEnabledForUser checks if a feature is enabled for a specific user
func (m *StaticManager) IsEnabledForUser(ctx context.Context, flag FeatureFlag, userID string) bool {
	return m.IsEnabled(ctx, flag)
}

// SetEnabled sets a feature flag's state
func (m *StaticManager) SetEnabled(flag FeatureFlag, enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.flags[flag] = enabled
}

// GetAllFlags returns all flag states
func (m *StaticManager) GetAllFlags() map[FeatureFlag]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make(map[FeatureFlag]bool)
	for k, v := range m.flags {
		result[k] = v
	}
	return result
}

// ContextKey for storing feature flags in context
type contextKey struct{}

// WithManager adds a feature flag manager to the context
func WithManager(ctx context.Context, manager Manager) context.Context {
	return context.WithValue(ctx, contextKey{}, manager)
}

// FromContext retrieves the feature flag manager from context
func FromContext(ctx context.Context) Manager {
	if manager, ok := ctx.Value(contextKey{}).(Manager); ok {
		return manager
	}
	// Return a default manager that disables all features
	return NewStaticManager(nil)
}

// IsEnabled is a convenience function to check if a feature is enabled
func IsEnabled(ctx context.Context, flag FeatureFlag) bool {
	return FromContext(ctx).IsEnabled(ctx, flag)
}

// IsEnabledForUser is a convenience function to check if a feature is enabled for a user
func IsEnabledForUser(ctx context.Context, flag FeatureFlag, userID string) bool {
	return FromContext(ctx).IsEnabledForUser(ctx, flag, userID)
}