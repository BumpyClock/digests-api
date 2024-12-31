// Package main provides the main functionality for the web server.
package main

import (
	"net/http"

	"go.uber.org/zap"
)

/**
 * @function errorMiddlewareFunc
 * @description Middleware function that recovers from panics, logs the error, and sends a 500 response.
 * @param {http.HandlerFunc} next The next handler function in the chain.
 * @returns {http.HandlerFunc} The wrapped handler function.
 * @dependencies log
 */
func errorMiddlewareFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				zap.L().Error("An error occurred", zap.Any("error", err))
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next(w, r)
	}
}

/**
 * @function InitializeRoutes
 * @description Sets up all the routes for the application.
 * @param {*http.ServeMux} mux The HTTP request multiplexer.
 * @returns {void}
 * @dependencies errorMiddlewareFunc, validateURLsHandler, parseHandler, discoverHandler,
 *               getReaderViewHandler, createShareHandler, shareHandler, searchHandler,
 *               streamAudioHandler, metadataHandler
 */
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
