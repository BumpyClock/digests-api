// Package main provides the main functionality for the web server.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

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
		log.WithFields(logrus.Fields{
			"url":   url,
			"error": err,
		}).Error("[ReaderView] Error retrieving content")
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
		log.WithFields(logrus.Fields{
			"method": r.Method,
		}).Warn("[ReaderView] Invalid method")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var urls Urls
	if err := json.NewDecoder(r.Body).Decode(&urls); err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Error("[ReaderView] Error decoding request body")
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
				log.WithFields(logrus.Fields{
					"url": url,
				}).Info("[ReaderView] Cache miss")
				result = getReaderViewResult(url)
				if len(result.TextContent) < 100 {
					result.TextContent = `<div id="readability-page-1" class="page"><p id="cmsg">Error getting reader view or site requires subscription. Please open the link in a new tab.</p></div>`
					result.ReaderView = result.TextContent
				}
				if err := cache.Set(readerView_prefix, cacheKey, result, 24*time.Hour); err != nil {
					log.WithFields(logrus.Fields{
						"url":   url,
						"error": err,
					}).Error("[ReaderView] Failed to cache reader view")
				}
			} else {
				log.WithFields(logrus.Fields{
					"url": url,
				}).Info("[ReaderView] Cache hit")
			}
			results[i] = result
		}(i, url)
	}
	wg.Wait()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Error("[ReaderView] Failed to encode JSON")
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
		log.WithFields(logrus.Fields{
			"url":   url,
			"error": err,
		}).Error("[ReaderView] Failed to parse URL")
		return readability.Article{}, fmt.Errorf("failed to parse URL: %w", err)
	}
	return article, nil
}
