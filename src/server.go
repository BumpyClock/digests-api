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

	"golang.org/x/time/rate"

	digestsCache "digests-app-api/cache"

	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
)

var limiter = rate.NewLimiter(1, 3) // Allow 1 request per second with a burst of 3 requests
var cache digestsCache.Cache        // Use the Cache interface
var cacheErr error
var log = logrus.New()
var urlList []string
var urlListMutex = &sync.Mutex{}
var refresh_timer = 15
var redis_address = "localhost:6379"
var numWorkers = runtime.NumCPU()

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

type gzipResponseWriter struct {
	http.ResponseWriter
	wroteHeader bool
	writer      *gzip.Writer
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.Header().Del("Content-Length")
		w.writer = gzip.NewWriter(w.ResponseWriter)
		w.wroteHeader = true
	}
	return w.writer.Write(b)
}

func (w *gzipResponseWriter) WriteHeader(status int) {
	w.ResponseWriter.WriteHeader(status)
	if w.wroteHeader && w.writer != nil {
		w.writer.Close()
	}
}

func (w *gzipResponseWriter) Flush() {
	if w.wroteHeader {
		w.writer.Flush()
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {

	// Start pprof profiling
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	port := flag.String("port", "8000", "port to run the application on")
	timer := flag.Int("timer", refresh_timer, "timer to refresh the cache")
	redis := flag.String("redis", "localhost:6379", "redis address")
	flag.Parse()
	mux := http.NewServeMux()
	log.Printf("mux: %v", mux)

	// Apply CORS middleware
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization"},
		AllowCredentials: true,
	}).Handler

	// Wrap the mux with the middleware
	handlerChain := corsMiddleware(mux)              // Apply CORS first
	handlerChain = RateLimitMiddleware(handlerChain) // Apply rate limiting next
	handlerChain = GzipMiddleware(handlerChain)      // Apply Gzip compression last
	InitializeRoutes(mux)                            // Assuming you've defined this to set up routes

	redis_address = *redis

	log.Info("Opening cache connection...")
	cache, cacheErr = digestsCache.NewRedisCache(redis_address, redis_password, redis_db)
	if cacheErr != nil {
		log.Warn("Failed to open Redis cache, falling back to in-memory cache")
		cache = digestsCache.NewGoCache(5*time.Minute, 10*time.Minute)
	}

	cachesize, cacheerr := cache.Count()
	if cacheerr == nil {
		log.Infof("Cache has %d items", cachesize)
	} else {
		log.Errorf("Failed to get cache size: %v", cacheerr)
	}

	refresh_timer = *timer

	refreshFeeds()

	go func() {
		ticker := time.NewTicker(time.Duration(refresh_timer*4) * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			log.Info("Refreshing cache...")
			refreshFeeds()
			log.Infof("Cache refreshed %v, %v", time.Now().Format(time.RFC3339), urlList)
		}
	}()

	log.Infof("Server is starting on port %v...", *port)
	log.Infof("Cache auto refresh timer is %v minutes", refresh_timer*4)
	log.Infof("Feed freshness timer is %v minutes", refresh_timer)
	log.Infof("Rate limit is %v requests per second", limiter.Limit())
	log.Infof("Redis address is %v", redis_address)
	log.Infof("Number of workers is %v", numWorkers)

	err := http.ListenAndServe(":"+*port, handlerChain)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
