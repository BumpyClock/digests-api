package main

import (
	"flag"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"gopkg.in/natefinch/lumberjack.v2"

	digestsCache "digests-app-api/cache"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/sirupsen/logrus"
)

var limiter = rate.NewLimiter(1, 3) // Allow 1 request per second with a burst of 3 requests
var cache *digestsCache.RedisCache
var cacheErr error
var log = logrus.New()
var urlList []string
var urlListMutex = &sync.Mutex{}
var refresh_timer = 15
var redis_address = "localhost:6379"
var numWorkers = runtime.NumCPU()

func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{"message": "Too Many Requests"})
			c.Abort()
			return
		}

		c.Next()
	}
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

	log.SetOutput(&lumberjack.Logger{
		Filename:   "app.log",
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     28,   //days
		Compress:   true, // disabled by default
	})

	logFile, err := os.OpenFile("logs/app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		logrus.Fatal(err)
	}

	log.SetOutput(logFile)
	port := flag.String("port", "8000", "port to run the application on")
	timer := flag.Int("timer", refresh_timer, "timer to refresh the cache")
	redis := flag.String("redis", "localhost:6379", "redis address")
	flag.Parse()

	router := gin.Default()
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type"}

	router.Use(cors.New(config)) // Create a new Gin engine

	// Wrap the mux with the middleware
	router.Use(RateLimitMiddleware())
	if err != nil {
		log.Fatalf("Failed to create gzip handler: %v", err)
	}
	router.Use(gzip.Gzip(gzip.DefaultCompression))

	InitializeRoutes(router) // Assuming you've defined this to set up routes

	redis_address = *redis

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

	err = router.Run(":" + *port) // Start the server
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
