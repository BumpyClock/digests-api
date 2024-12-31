// Package main provides the main functionality for the web server.
package main

import (
	"encoding/json"
	"net/http"
	"time"

	"math/rand"

	"go.uber.org/zap"
)

/**
 * @function createShareHandler
 * @description Handles HTTP requests to the /create endpoint.
 *              It expects a POST request with a JSON body containing an array of URLs.
 *              It generates a random 6-character key, stores the URLs in the cache with that key,
 *              and returns a link to the /share endpoint with the generated key.
 * @param {http.ResponseWriter} w The HTTP response writer.
 * @param {*http.Request} r The HTTP request.
 * @returns {void}
 * @dependencies cache, log, rand.Seed, rand.Intn
 */
func createShareHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		zap.L().Warn("[createShareHandler] Invalid method", zap.String("method", r.Method))
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	var req createShareRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		zap.L().Error("[createShareHandler] Error decoding request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Urls) == 0 {
		zap.L().Warn("[createShareHandler] No URLs provided")
		http.Error(w, "No URLs provided", http.StatusBadRequest)
		response := map[string]string{
			"status": "error",
			"error":  "No URLs provided",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	zap.L().Info("[createShareHandler] Create request received", zap.Strings("urls", req.Urls))

	// Generate a random key of maximum 6 characters
	rand.Seed(time.Now().UnixNano())
	chars := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	randomKey := make([]rune, 6)
	for i := range randomKey {
		randomKey[i] = chars[rand.Intn(len(chars))]
	}

	cacheKey := "share:" + string(randomKey)

	// Save the URLs to the cache
	err = cache.Set(cacheKey, "urls", req.Urls, 0) // setting exp 0 to keep it forever
	if err != nil {
		zap.L().Error("[createShareHandler] Failed to save shared URLs", zap.String("key", cacheKey), zap.Strings("urls", req.Urls), zap.Error(err))
		http.Error(w, "Failed to save shared URLs", http.StatusInternalServerError)
		return
	}

	// Respond with the link
	response := map[string]string{"status": "ok", "link": "https://www.digests.app/share/" + string(randomKey)}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

/**
 * @function shareHandler
 * @description Handles HTTP requests to the /share endpoint.
 *              It expects a POST request with a JSON body containing a key.
 *              It retrieves the URLs associated with that key from the cache and returns them as JSON.
 * @param {http.ResponseWriter} w The HTTP response writer.
 * @param {*http.Request} r The HTTP request.
 * @returns {void}
 * @dependencies cache, log
 */
func shareHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		zap.L().Warn("[shareHandler] Invalid method", zap.String("method", r.Method))
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	zap.L().Info("[shareHandler] Share request received")

	// Decode the request body
	var req fetchShareRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		zap.L().Error("[shareHandler] Error decoding request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get the key from the request body
	cacheKey := "share:" + req.Key

	// Get the URLs from the cache
	var urls []string
	err = cache.Get(cacheKey, "urls", &urls)
	if err != nil {
		zap.L().Error("[shareHandler] Error getting URLs from cache", zap.String("key", cacheKey), zap.Error(err))
		http.Error(w, "Invalid share link", http.StatusBadRequest)
		return
	}

	// Respond with the URLs
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(urls)

	zap.L().Info("[shareHandler] Share request processed", zap.String("key", req.Key), zap.Strings("urls", urls))
}
