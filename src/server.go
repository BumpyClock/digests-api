package main

import (
	"compress/gzip"
	"flag"
	"sync"
	"time"

	"net/http"
	"strings"

	"golang.org/x/time/rate"

	digestsCache "digests-app-api/cache"

	"github.com/sirupsen/logrus"
)

var limiter = rate.NewLimiter(1, 3) // Allow 1 request per second with a burst of 3 requests
var cache *digestsCache.RedisCache
var cacheErr error
var log = logrus.New()
var urlList []string
var urlListMutex = &sync.Mutex{}
var refresh_timer = 15

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

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers for all responses, including preflights
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		// Immediately respond to OPTIONS method for CORS preflight request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	port := flag.String("port", "8000", "port to run the application on")
	timer := flag.Int("timer", 15, "timer to refresh the cache")
	flag.Parse()
	mux := http.NewServeMux()
	InitializeRoutes(mux) // Assuming you've defined this to set up routes

	// Wrap the mux with the middleware
	handlerChain := CORSMiddleware(mux)
	handlerChain = RateLimitMiddleware(handlerChain)
	handlerChain = GzipMiddleware(handlerChain)

	log.Info("Opening cache connection...")
	cache, cacheErr = digestsCache.NewRedisCache(redis_address, redis_password, redis_db)
	if cacheErr != nil {
		log.Fatalf("Failed to open cache connection: %v", cacheErr)
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
		ticker := time.NewTicker(time.Duration(*timer) * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			log.Info("Refreshing cache...")
			refreshFeeds()
			log.Infof("Cache refreshed %v, %v", time.Now().Format(time.RFC3339), urlList)
		}
	}()

	log.Infof("Server is starting on port %v...", *port)

	err := http.ListenAndServe(":"+*port, handlerChain)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
