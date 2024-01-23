package main

import (
	"compress/gzip"
	"log"
	"net/http"
	"strings"
)

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

func main() {
	mux := http.NewServeMux()
	InitializeRoutes(mux) // Set up routes

	// Wrap the mux with the GzipMiddleware
	wrappedMux := GzipMiddleware(mux)

	log.Println("Server is starting on port 8080...")

	err := http.ListenAndServe(":8080", wrappedMux)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
