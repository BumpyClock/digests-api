// ABOUTME: Main entry point for the Digests API server
// ABOUTME: Wires together all components and starts the HTTP server

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"digests-app-api/api"
	"digests-app-api/api/handlers"
	"digests-app-api/core/feed"
	"digests-app-api/core/interfaces"
	"digests-app-api/core/reader"
	"digests-app-api/core/search"
	"digests-app-api/core/services"
	"digests-app-api/infrastructure/cache/memory"
	"digests-app-api/infrastructure/cache/redis"
	"digests-app-api/infrastructure/cache/sqlite"
	stdhttp "digests-app-api/infrastructure/http/standard"
	stdlogger "digests-app-api/infrastructure/logger/standard"
	"digests-app-api/pkg/config"
)

func main() {
	// Load configuration
	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Create logger
	logger := stdlogger.NewStandardLogger()
	logger.Info("Starting Digests API", map[string]interface{}{
		"port":         cfg.Server.Port,
		"cache_type":   cfg.Cache.Type,
		"refresh_timer": cfg.Server.RefreshTimer,
	})

	// Create cache
	var cache interfaces.Cache
	switch cfg.Cache.Type {
	case "redis":
		redisCache, err := redis.NewRedisCache(cfg.Cache.Redis)
		if err != nil {
			logger.Error("Failed to create Redis cache, falling back to memory", map[string]interface{}{
				"error": err.Error(),
			})
			cache = memory.NewMemoryCache()
		} else {
			cache = redisCache
			logger.Info("Using Redis cache", map[string]interface{}{
				"address": cfg.Cache.Redis.Address,
			})
		}
	case "sqlite":
		sqliteCache, err := sqlite.NewSQLiteCache(cfg.Cache.SQLite.FilePath)
		if err != nil {
			logger.Error("Failed to create SQLite cache, falling back to memory", map[string]interface{}{
				"error": err.Error(),
			})
			cache = memory.NewMemoryCache()
		} else {
			cache = sqliteCache
			logger.Info("Using SQLite cache", map[string]interface{}{
				"file_path": cfg.Cache.SQLite.FilePath,
			})
		}
	default:
		cache = memory.NewMemoryCache()
		logger.Info("Using memory cache", nil)
	}

	// Create HTTP client
	httpClient := stdhttp.NewStandardHTTPClient(30 * time.Second)

	// Create dependencies container
	deps := interfaces.Dependencies{
		Cache:      cache,
		HTTPClient: httpClient,
		Logger:     logger,
	}

	// Create services
	feedService := feed.NewFeedService(deps)
	searchService := search.NewSearchService(deps)
	readerService := reader.NewService(cache, logger)
	
	// Create unified enrichment service with configurable cache TTL
	colorCacheTTL := time.Duration(cfg.Cache.ColorCacheDays) * 24 * time.Hour
	enrichmentService := services.NewContentEnrichmentService(deps, colorCacheTTL)
	
	// Note: Share service would need a storage implementation
	_ = searchService // Will be used when we add search handlers

	// Create API with middleware
	apiConfig := api.APIConfig{
		Logger:      logger,
		RateLimit:   100,                    // 100 requests per minute
		RateWindow:  time.Minute,
	}
	humaAPI, router := api.NewAPIWithMiddleware(apiConfig)

	// Create and register handlers
	feedHandler := handlers.NewFeedHandler(feedService, enrichmentService)
	feedHandler.RegisterRoutes(humaAPI)
	
	discoverHandler := handlers.NewDiscoverHandler(httpClient)
	discoverHandler.RegisterRoutes(humaAPI)
	
	metadataHandler := handlers.NewMetadataHandler()
	metadataHandler.RegisterRoutes(humaAPI)
	
	validateHandler := handlers.NewValidateHandler(httpClient)
	validateHandler.RegisterRoutes(humaAPI)
	
	readerHandler := handlers.NewReaderHandler(readerService)
	readerHandler.RegisterRoutes(humaAPI)

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("HTTP server starting", map[string]interface{}{
			"address": srv.Addr,
		})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", map[string]interface{}{
				"error": err.Error(),
			})
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...", nil)

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", map[string]interface{}{
			"error": err.Error(),
		})
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server stopped", nil)
}

func init() {
	// Print banner
	fmt.Println(`
    ____  _                  __        ___    ____  ____
   / __ \(_)___ ____  ______/ /____   /   |  / __ \/  _/
  / / / / / __ '/ _ \/ ___/ __/ ___/ / /| | / /_/ // /  
 / /_/ / / /_/ /  __(__  ) /_(__  ) / ___ |/ ____// /   
/_____/_/\__, /\___/____/\__/____/ /_/  |_/_/   /___/   
        /____/                                           
	`)
}