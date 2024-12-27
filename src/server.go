// Package main provides the main functionality for the web server.
package main

import (
	"compress/gzip"
	"flag"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"strings"
	"sync"
	"time"

	digestsCache "digests-app-api/cache"

	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

var (
	limiter       = rate.NewLimiter(5, 15) // Rate limiter: 5 requests/sec, burst 15
	cache         digestsCache.Cache       // Cache instance
	log           = logrus.New()           // Logger instance
	urlList       []string                 // List of URLs to refresh
	urlListMutex  = &sync.Mutex{}          // Mutex for urlList
	refresh_timer = 60                     // Refresh timer in minutes
	redis_address = "localhost:6379"       // Redis server address
	numWorkers    = runtime.NumCPU()       // Number of worker goroutines
	cacheMutex    = &sync.Mutex{}          // Mutex for cache operations
	httpClient    = &http.Client{Timeout: 20 * time.Second}
	cachePeriod   = 30 // Cache period in days
)

func main() {
	// Start pprof profiling in a goroutine
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	// Command-line flags
	port := flag.String("port", "8000", "port to run the application on")
	timer := flag.Int("timer", refresh_timer, "timer to refresh the cache")
	redis := flag.String("redis", "localhost:6379", "redis address")
	flag.Parse()

	// HTTP request multiplexer
	mux := http.NewServeMux()
	log.Infof("Number of workers: %v", numWorkers)

	// Setup routes
	InitializeRoutes(mux)

	// Wrap mux with middlewares
	var handler http.Handler = mux

	// 1) Recover from panics
	handler = errorRecoveryMiddleware(handler)

	// 2) CORS
	handler = cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodOptions, http.MethodPut, http.MethodDelete},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization"},
		AllowCredentials: true,
	}).Handler(handler)

	// 3) Rate limit
	handler = RateLimitMiddleware(handler)

	// 4) GZIP compression
	handler = GzipMiddleware(handler)

	// Cache setup
	redis_address = *redis
	log.Info("Opening cache connection...")

	// Attempt to connect to Redis, otherwise use in-memory cache
	redisCache, redisErr := digestsCache.NewRedisCache(redis_address, redis_password, redis_db)
	if redisErr != nil {
		log.Warnf("Failed to open Redis cache (%v); falling back to in-memory cache", redisErr)
		cache = digestsCache.NewGoCache(5*time.Minute, 10*time.Minute)
	} else {
		cache = redisCache
	}

	// Log cache size
	cachesize, cacheerr := cache.Count()
	if cacheerr != nil {
		log.Errorf("Failed to get cache size: %v", cacheerr)
	} else {
		log.Infof("Cache has %d items", cachesize)
	}

	// Set refresh timer from command-line
	refresh_timer = *timer
	refreshFeeds()

	// Periodic refresh in a separate goroutine
	go func() {
		ticker := time.NewTicker(time.Duration(refresh_timer*4) * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			log.Info("Refreshing cache (periodic)...")
			refreshFeeds()
			log.Infof("Cache refreshed at %v, urlList=%v", time.Now().Format(time.RFC3339), urlList)
		}
	}()

	// Start the server
	log.Infof("Server is starting on port %v", *port)
	log.Infof("Refresh timer is %v minutes", refresh_timer)
	log.Infof("Redis address is %v", redis_address)

	err := http.ListenAndServe(":"+*port, handler)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

/**
 * @function errorRecoveryMiddleware
 * @description Middleware that recovers from panics, logs the error, and sends a 500 response.
 * @param {http.Handler} next The next handler in the chain.
 * @returns {http.Handler} The wrapped handler.
 */
func errorRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Errorf("Panic recovered: %v", rec)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

/**
 * @function GzipMiddleware
 * @description Middleware that adds gzip compression to responses if the client accepts it.
 * @param {http.Handler} next The next handler in the chain.
 * @returns {http.Handler} The wrapped handler.
 */
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		wrw := gzipResponseWriter{ResponseWriter: w}
		wrw.Header().Set("Content-Encoding", "gzip")
		defer wrw.Flush()

		next.ServeHTTP(&wrw, r)
	})
}

// gzipResponseWriter is a custom ResponseWriter that wraps the original ResponseWriter
// and compresses the response body using gzip.
type gzipResponseWriter struct {
	http.ResponseWriter
	wroteHeader bool
	writer      *gzip.Writer
}

/**
 * @function Write
 * @description Writes data to the gzip writer.
 * @param {[]byte} b The data to write.
 * @returns {(int, error)} The number of bytes written and any error that occurred.
 */
func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.Header().Del("Content-Length")
		w.writer = gzip.NewWriter(w.ResponseWriter)
		w.wroteHeader = true
	}
	return w.writer.Write(b)
}

/**
 * @function WriteHeader
 * @description Writes the HTTP status code to the response.
 *              Closes the gzip writer if it was created.
 * @param {int} status The HTTP status code.
 * @returns {void}
 */
func (w *gzipResponseWriter) WriteHeader(status int) {
	w.ResponseWriter.WriteHeader(status)
	if w.wroteHeader && w.writer != nil {
		w.writer.Close()
	}
}

/**
 * @function Flush
 * @description Flushes the gzip writer and the underlying ResponseWriter.
 * @returns {void}
 */
func (w *gzipResponseWriter) Flush() {
	if w.wroteHeader {
		_ = w.writer.Flush()
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

/**
 * @function RateLimitMiddleware
 * @description Middleware that applies rate limiting to incoming requests.
 * @param {http.Handler} next The next handler in the chain.
 * @returns {http.Handler} The wrapped handler.
 */
func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
