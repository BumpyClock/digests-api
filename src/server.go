package main

import (
	"compress/gzip"
	"flag"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	digestsCache "digests-app-api/cache"

	"github.com/rs/cors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/time/rate"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	limiter       = rate.NewLimiter(5, 15) // e.g., 5 requests/sec, burst 15
	cache         digestsCache.Cache
	log           *zap.Logger
	urlList       []string
	urlListMutex  = &sync.Mutex{}
	refresh_timer = 60
	redis_address = "localhost:6379"
	numWorkers    = runtime.NumCPU()
	cacheMutex    = &sync.Mutex{}
	httpClient    = &http.Client{Timeout: 20 * time.Second}
	cachePeriod   = 30
)

func initializeLogger() {
	// Temporary logger for early messages during initialization
	tempLogger := zap.NewNop() // No-op logger

	logRetentionDays := flag.String("log-retention", "2", "Number of days to retain log files")
	// Get log retention days from flag, with error handling
	retentionDaysStr := *logRetentionDays
	retentionDays, err := strconv.Atoi(retentionDaysStr)
	if err != nil {
		// Use tempLogger here
		tempLogger.Error("Error parsing log retention days from flag", zap.Error(err))
		retentionDays = 2
	}

	// Ensure log file exists
	logFilePath := "./application.log" // Or a configurable path
	if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
		file, err := os.Create(logFilePath)
		if err != nil {
			// Use tempLogger here
			tempLogger.Error("Failed to create log file", zap.Error(err))
		}
		if file != nil {
			file.Close() // Close the file immediately as lumberjack will handle it
		}
	}

	// Configure lumberjack for file rotation
	logFileWriter := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    100, // Megabytes
		MaxBackups: 5,
		MaxAge:     retentionDays, // Days
		Compress:   true,
	}

	// Determine the logging level based on environment variable or default
	logLevel := os.Getenv("LOG_LEVEL")
	var zapLevel zapcore.Level
	switch strings.ToLower(logLevel) {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	case "production": // Consider production as error level by default
		zapLevel = zapcore.ErrorLevel
	default:
		// Default to debug in development, info otherwise
		if os.Getenv("GIN_MODE") == "debug" || os.Getenv("GIN_MODE") == "" {
			zapLevel = zapcore.DebugLevel
		} else {
			zapLevel = zapcore.InfoLevel
		}
	}

	// Configure console logging
	consoleEncoderConfig := zap.NewDevelopmentEncoderConfig()
	consoleEncoder := zapcore.NewJSONEncoder(consoleEncoderConfig) // Or NewConsoleEncoder for human-readable

	// Configure file logging
	fileEncoderConfig := zap.NewProductionEncoderConfig()
	fileEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(fileEncoderConfig)

	// Create core with different outputs and levels
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, zapcore.Lock(os.Stdout), zapLevel),
		zapcore.NewCore(fileEncoder, zapcore.AddSync(logFileWriter), zapLevel),
	)

	// Create the logger with caller information
	logger := zap.New(core, zap.AddCaller())

	// Replace the global logger
	log = logger

	// Now that the global logger is set, use zap.L() for subsequent logs
	zap.ReplaceGlobals(log)
}

func main() {
	// Initialize the logger first!
	initializeLogger()
	// Start pprof profiling in a goroutine
	go func() {
		zap.L().Info("pprof running on port 6060")
		zap.L().Error("Error starting pprof", zap.Error(http.ListenAndServe("localhost:6060", nil)))
	}()

	port := flag.String("port", "8000", "port to run the application on")
	timer := flag.Int("timer", refresh_timer, "timer to refresh the cache")
	redis := flag.String("redis", "localhost:6379", "redis address")
	flag.Parse()

	mux := http.NewServeMux()
	zap.L().Info("Number of workers", zap.Int("workers", numWorkers))

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
	zap.L().Info("Opening cache connection...")

	redisCache, redisErr := digestsCache.NewRedisCache(redis_address, redis_password, redis_db)
	if redisErr != nil {
		zap.L().Warn("Failed to open Redis cache, falling back to in-memory cache", zap.Error(redisErr))
		cache = digestsCache.NewGoCache(5*time.Minute, 10*time.Minute)
	} else {
		cache = redisCache
	}

	cachesize, cacheerr := cache.Count()
	if cacheerr != nil {
		zap.L().Error("Failed to get cache size", zap.Error(cacheerr))
	} else {
		zap.L().Info("Cache has items", zap.Int64("items", cachesize))
	}

	// Set refresh timer from command-line
	refresh_timer = *timer
	refreshFeeds()

	// Periodic refresh
	go func() {
		ticker := time.NewTicker(time.Duration(refresh_timer*4) * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			zap.L().Info("Refreshing cache (periodic)...")
			refreshFeeds()
			zap.L().Info("Cache refreshed", zap.Time("time", time.Now()), zap.Strings("urlList", urlList))
		}
	}()

	zap.L().Info("Server is starting", zap.String("port", *port))
	zap.L().Info("Refresh timer", zap.Int("timer", refresh_timer))
	zap.L().Info("Redis address", zap.String("address", redis_address))

	err := http.ListenAndServe(":"+*port, handler)
	if err != nil {
		zap.L().Fatal("Server failed to start", zap.Error(err))
	}
}

// errorRecoveryMiddleware ensures the server recovers from unexpected panics
func errorRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				zap.L().Error("Panic recovered", zap.Any("error", rec))
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// GzipMiddleware adds gzip compression
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
		_ = w.writer.Flush()
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// RateLimitMiddleware applies a rate limiter
func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
