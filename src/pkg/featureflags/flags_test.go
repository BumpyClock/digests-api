package featureflags

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFeedParser_DisabledByDefault(t *testing.T) {
	manager := NewEnvManager("TEST_FEATURE_")
	ctx := context.Background()
	
	// Should be disabled when env var not set
	assert.False(t, manager.IsEnabled(ctx, NewFeedParser))
}

func TestNewFeedParser_EnabledWhenFlagSet(t *testing.T) {
	// Set environment variable
	os.Setenv("TEST_FEATURE_NEW_FEED_PARSER", "true")
	defer os.Unsetenv("TEST_FEATURE_NEW_FEED_PARSER")
	
	manager := NewEnvManager("TEST_FEATURE_")
	ctx := context.Background()
	
	assert.True(t, manager.IsEnabled(ctx, NewFeedParser))
}

func TestEnvManager_MultipleValues(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"true lowercase", "true", true},
		{"TRUE uppercase", "TRUE", true},
		{"1 numeric", "1", true},
		{"enabled", "enabled", true},
		{"ENABLED", "ENABLED", true},
		{"false", "false", false},
		{"0", "0", false},
		{"empty", "", false},
		{"other", "yes", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("TEST_FLAG", tt.value)
			defer os.Unsetenv("TEST_FLAG")
			
			manager := NewEnvManager("TEST_")
			ctx := context.Background()
			
			assert.Equal(t, tt.expected, manager.IsEnabled(ctx, "FLAG"))
		})
	}
}

func TestEnvManager_SetEnabled(t *testing.T) {
	manager := NewEnvManager("TEST_")
	ctx := context.Background()
	
	// Initially disabled
	assert.False(t, manager.IsEnabled(ctx, SearchEnabled))
	
	// Enable via SetEnabled
	manager.SetEnabled(SearchEnabled, true)
	assert.True(t, manager.IsEnabled(ctx, SearchEnabled))
	
	// Disable via SetEnabled
	manager.SetEnabled(SearchEnabled, false)
	assert.False(t, manager.IsEnabled(ctx, SearchEnabled))
}

func TestEnvManager_OverrideTakesPrecedence(t *testing.T) {
	// Set env var to true
	os.Setenv("TEST_FEATURE_CACHE_ENABLED", "true")
	defer os.Unsetenv("TEST_FEATURE_CACHE_ENABLED")
	
	manager := NewEnvManager("TEST_FEATURE_")
	ctx := context.Background()
	
	// Should be true from env
	assert.True(t, manager.IsEnabled(ctx, CacheEnabled))
	
	// Override to false
	manager.SetEnabled(CacheEnabled, false)
	
	// Override should take precedence
	assert.False(t, manager.IsEnabled(ctx, CacheEnabled))
}

func TestStaticManager(t *testing.T) {
	flags := map[FeatureFlag]bool{
		NewFeedParser: true,
		SearchEnabled: false,
		ShareEnabled:  true,
	}
	
	manager := NewStaticManager(flags)
	ctx := context.Background()
	
	assert.True(t, manager.IsEnabled(ctx, NewFeedParser))
	assert.False(t, manager.IsEnabled(ctx, SearchEnabled))
	assert.True(t, manager.IsEnabled(ctx, ShareEnabled))
	assert.False(t, manager.IsEnabled(ctx, MetricsEnabled)) // Not in initial map
}

func TestStaticManager_SetEnabled(t *testing.T) {
	manager := NewStaticManager(nil)
	ctx := context.Background()
	
	// All disabled by default
	assert.False(t, manager.IsEnabled(ctx, RateLimitEnabled))
	
	// Enable flag
	manager.SetEnabled(RateLimitEnabled, true)
	assert.True(t, manager.IsEnabled(ctx, RateLimitEnabled))
}

func TestGetAllFlags(t *testing.T) {
	flags := map[FeatureFlag]bool{
		NewFeedParser:    true,
		SearchEnabled:    false,
		ShareEnabled:     true,
		MetricsEnabled:   false,
		RateLimitEnabled: true,
		CacheEnabled:     true,
	}
	
	manager := NewStaticManager(flags)
	allFlags := manager.GetAllFlags()
	
	assert.Equal(t, flags, allFlags)
}

func TestContextIntegration(t *testing.T) {
	manager := NewStaticManager(map[FeatureFlag]bool{
		NewFeedParser: true,
	})
	
	ctx := context.Background()
	ctx = WithManager(ctx, manager)
	
	// Using convenience functions
	assert.True(t, IsEnabled(ctx, NewFeedParser))
	assert.False(t, IsEnabled(ctx, SearchEnabled))
}

func TestFromContext_DefaultManager(t *testing.T) {
	ctx := context.Background()
	
	// Without manager in context, should return default (all disabled)
	assert.False(t, IsEnabled(ctx, NewFeedParser))
	assert.False(t, IsEnabled(ctx, SearchEnabled))
}

func TestIsEnabledForUser(t *testing.T) {
	manager := NewStaticManager(map[FeatureFlag]bool{
		ShareEnabled: true,
	})
	
	ctx := context.Background()
	
	// For both EnvManager and StaticManager, user-specific is same as global
	assert.True(t, manager.IsEnabledForUser(ctx, ShareEnabled, "user123"))
	assert.False(t, manager.IsEnabledForUser(ctx, SearchEnabled, "user123"))
}

func TestConcurrentAccess(t *testing.T) {
	manager := NewStaticManager(nil)
	ctx := context.Background()
	
	// Run concurrent reads and writes
	done := make(chan bool)
	
	// Writers
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				manager.SetEnabled(NewFeedParser, j%2 == 0)
			}
			done <- true
		}()
	}
	
	// Readers
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = manager.IsEnabled(ctx, NewFeedParser)
			}
			done <- true
		}()
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestFeatureFlagNames(t *testing.T) {
	// Ensure flag names are what we expect
	assert.Equal(t, FeatureFlag("new_feed_parser"), NewFeedParser)
	assert.Equal(t, FeatureFlag("search_enabled"), SearchEnabled)
	assert.Equal(t, FeatureFlag("share_enabled"), ShareEnabled)
	assert.Equal(t, FeatureFlag("metrics_enabled"), MetricsEnabled)
	assert.Equal(t, FeatureFlag("rate_limit_enabled"), RateLimitEnabled)
	assert.Equal(t, FeatureFlag("cache_enabled"), CacheEnabled)
}