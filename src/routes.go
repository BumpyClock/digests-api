package main

import (
	"net/http"
)

func errorHandlingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("An error occurred: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// InitializeRoutes sets up all the routes for the application
func InitializeRoutes(mux *http.ServeMux) {
	// Apply the errorHandlingMiddleware to the validateURLsHandler
	mux.Handle("/validate", errorHandlingMiddleware(http.HandlerFunc(validateURLsHandler)))
	// Apply the errorHandlingMiddleware to the parseHandler
	mux.Handle("/parse", errorHandlingMiddleware(http.HandlerFunc(parseHandler)))

	mux.Handle("/discover", errorHandlingMiddleware(http.HandlerFunc(discoverHandler)))

	mux.Handle("/getreaderview", errorHandlingMiddleware(http.HandlerFunc(getReaderViewHandler)))

	mux.Handle("/create", errorHandlingMiddleware(http.HandlerFunc(createShareHandler)))

	mux.Handle("/share", errorHandlingMiddleware(http.HandlerFunc(shareHandler)))

	mux.Handle("/search", errorHandlingMiddleware(http.HandlerFunc(searchHandler)))
}
