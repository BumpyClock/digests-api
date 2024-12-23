package main

import (
	"net/http"
)

func errorMiddlewareFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("An error occurred: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next(w, r)
	}
}

// InitializeRoutes sets up all the routes for the application
func InitializeRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/validate", errorMiddlewareFunc(validateURLsHandler))
	mux.HandleFunc("/parse", errorMiddlewareFunc(parseHandler))
	mux.HandleFunc("/discover", errorMiddlewareFunc(discoverHandler))
	mux.HandleFunc("/getreaderview", errorMiddlewareFunc(getReaderViewHandler))
	mux.HandleFunc("/create", errorMiddlewareFunc(createShareHandler))
	mux.HandleFunc("/share", errorMiddlewareFunc(shareHandler))
	mux.HandleFunc("/search", errorMiddlewareFunc(searchHandler))
	mux.HandleFunc("/streamaudio", errorMiddlewareFunc(streamAudioHandler))
	mux.HandleFunc("/metadata", errorMiddlewareFunc(metadataHandler))
}
