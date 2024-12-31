// Package main provides the main functionality for the web server.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	readability "github.com/go-shiori/go-readability"
)

/**
 * @function getReaderViewResult
 * @description Retrieves the reader view of a given URL using the go-readability library.
 *              It returns a ReaderViewResult containing the parsed content or an error.
 * @param {string} url The URL to retrieve the reader view for.
 * @returns {ReaderViewResult} A struct containing the reader view result or an error.
 * @dependencies getReaderView, log
 */
func getReaderViewResult(url string) ReaderViewResult {
	readerView, err := getReaderView(url)
	if err != nil {
		zap.L().Error("[ReaderView] Error retrieving content", zap.String("url", url), zap.Error(err))
		return ReaderViewResult{
			URL:    url,
			Status: "error",
			Error:  err,
		}
	}
	return ReaderViewResult{
		URL:         url,
		Status:      "ok",
		ReaderView:  readerView.Content,
		Title:       readerView.Title,
		SiteName:    readerView.SiteName,
		Image:       readerView.Image,
		Favicon:     readerView.Favicon,
		TextContent: readerView.TextContent,
	}
}

/**
 * @function getReaderViewHandler
 * @description Handles HTTP requests to the /getreaderview endpoint.
 *              It expects a POST request with a JSON body containing an array of URLs.
 *              It retrieves the reader view for each URL, caches the results, and returns a JSON response.
 * @param {http.ResponseWriter} w The HTTP response writer.
 * @param {*http.Request} r The HTTP request.
 * @returns {void}
 * @dependencies getReaderViewResult, cache, createHash, log
 */
func getReaderViewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		zap.L().Warn("[ReaderView] Invalid method", zap.String("method", r.Method))
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var urls Urls
	if err := json.NewDecoder(r.Body).Decode(&urls); err != nil {
		zap.L().Error("[ReaderView] Error decoding request body", zap.Error(err))
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var wg sync.WaitGroup
	results := make([]ReaderViewResult, len(urls.Urls))

	for i, url := range urls.Urls {
		wg.Add(1)
		go func(i int, url string) {
			defer wg.Done()

			cacheKey := createHash(url)
			var result ReaderViewResult

			if err := cache.Get(readerView_prefix, cacheKey, &result); err != nil {
				zap.L().Info("[ReaderView] Cache miss", zap.String("url", url))
				result = getReaderViewResult(url)
				if len(result.TextContent) < 100 {
					result.TextContent = `<div id="readability-page-1" class="page"><p id="cmsg">Error getting reader view or site requires subscription. Please open the link in a new tab.</p></div>`
					result.ReaderView = result.TextContent
				}
				if err := cache.Set(readerView_prefix, cacheKey, result, 24*time.Hour); err != nil {
					zap.L().Error("[ReaderView] Failed to cache reader view", zap.String("url", url), zap.Error(err))
				}
			} else {
				zap.L().Info("[ReaderView] Cache hit", zap.String("url", url))
			}
			results[i] = result
		}(i, url)
	}
	wg.Wait()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		zap.L().Error("[ReaderView] Failed to encode JSON", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

/**
 * @function getReaderView
 * @description Retrieves the reader view of a given URL using the go-readability library.
 *              It sets a timeout for the request and returns the parsed article or an error.
 * @param {string} url The URL to retrieve the reader view for.
 * @returns {readability.Article, error} The parsed article or an error.
 * @dependencies readability.FromURL, log
 */
func getReaderView(url string) (readability.Article, error) {
	article, err := readability.FromURL(url, 30*time.Second)
	if err != nil {
		zap.L().Error("[ReaderView] Failed to parse URL", zap.String("url", url), zap.Error(err))
		return readability.Article{}, fmt.Errorf("failed to parse URL: %w", err)
	}
	return article, nil
}
