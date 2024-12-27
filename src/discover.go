// Package main provides the main functionality for the web server.
package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/sirupsen/logrus"
)

var (
	feedResultPool = &sync.Pool{
		New: func() interface{} {
			return &FeedResult{}
		},
	}
)

/**
 * @function discoverHandler
 * @description Handles HTTP requests to the /discover endpoint.
 *              It expects a POST request with a JSON body containing an array of URLs.
 *              It attempts to discover the RSS feed URL for each provided URL and returns the results as JSON.
 * @param {http.ResponseWriter} w The HTTP response writer.
 * @param {*http.Request} r The HTTP request.
 * @returns {void}
 */
func discoverHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		log.WithFields(logrus.Fields{
			"method": r.Method,
		}).Warn("[discoverHandler] Invalid method")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decode the request body
	var urls Urls
	err := json.NewDecoder(r.Body).Decode(&urls)
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Error("[discoverHandler] Error decoding request body")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Process each URL concurrently
	var wg sync.WaitGroup
	results := make([]FeedResult, len(urls.Urls))
	for i, url := range urls.Urls {
		wg.Add(1)
		go func(i int, url string) {
			defer wg.Done()

			// Discover the RSS feed URL
			feedLink, err := discoverRssFeedUrl(url)

			// Get a FeedResult from the pool, populate it, and return it to the pool when done
			result := feedResultPool.Get().(*FeedResult)
			defer feedResultPool.Put(result)

			if err != nil {
				result.URL = url
				result.Status = "error"
				result.Error = err.Error()
				result.FeedLink = ""
			} else {
				result.URL = url
				result.Status = "ok"
				result.FeedLink = feedLink
			}
			results[i] = *result
		}(i, url)
	}
	wg.Wait()

	// Encode the results as JSON and send the response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]FeedResult{"feeds": results})
}

/**
 * @function discoverRssFeedUrl
 * @description Discovers the RSS feed URL for a given URL.
 *              It first checks if the URL is a GitHub or Reddit URL and generates the feed URL accordingly.
 *              Otherwise, it sends an HTTP GET request to the URL, parses the HTML response,
 *              and searches for link elements with type="application/rss+xml".
 * @param {string} urlStr The URL to discover the RSS feed for.
 * @returns {(string, error)} The RSS feed URL if found, or an error if not found or an error occurred.
 */
func discoverRssFeedUrl(urlStr string) (string, error) {
	// Special handling for GitHub and Reddit URLs
	if strings.HasPrefix(urlStr, "https://github.com") {
		return generateGitHubRssUrl(urlStr), nil
	}
	if strings.HasPrefix(urlStr, "https://www.reddit.com") {
		return generateRedditRssUrl(urlStr), nil
	}

	// Prepare the HTTP request
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		log.WithFields(logrus.Fields{
			"url":   urlStr,
			"error": err,
		}).Error("[discoverRssFeedUrl] Error creating request")
		return "", err
	}

	// Set a custom User-Agent to avoid being blocked by some websites
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3")

	// Send the HTTP request
	resp, err := httpClient.Do(req)
	if err != nil {
		log.WithFields(logrus.Fields{
			"url":   urlStr,
			"error": err,
		}).Error("[discoverRssFeedUrl] Error sending request")
		return "", err
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		log.WithFields(logrus.Fields{
			"url":        urlStr,
			"statusCode": resp.StatusCode,
		}).Error("[discoverRssFeedUrl] Request failed")
		return "", errors.New("request failed")
	}

	// Parse the HTML response
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.WithFields(logrus.Fields{
			"url":   urlStr,
			"error": err,
		}).Error("[discoverRssFeedUrl] Error parsing HTML")
		return "", err
	}

	// Find the RSS feed link in the HTML
	rssLink, exists := doc.Find(`link[type="application/rss+xml"]`).Attr("href")
	if !exists {
		log.WithFields(logrus.Fields{
			"url": urlStr,
		}).Warn("[discoverRssFeedUrl] RSS feed not found in HTML")
		return "", errors.New("RSS feed not found")
	}

	// Ensure the RSS feed link is an absolute URL
	rssLink, err = ensureAbsoluteUrl(urlStr, rssLink)
	if err != nil {
		log.WithFields(logrus.Fields{
			"url":     urlStr,
			"rssLink": rssLink,
			"error":   err,
		}).Error("[discoverRssFeedUrl] Error ensuring absolute URL")
		return "", err
	}

	return rssLink, nil
}

/**
 * @function generateGitHubRssUrl
 * @description Generates a GitHub RSS feed URL from a GitHub repository URL.
 * @param {string} url The GitHub repository URL.
 * @returns {string} The corresponding GitHub RSS feed URL.
 */
func generateGitHubRssUrl(url string) string {
	return strings.TrimRight(url, "/") + "/commits/master.atom"
}

/**
 * @function generateRedditRssUrl
 * @description Generates a Reddit RSS feed URL from a Reddit URL.
 * @param {string} url The Reddit URL.
 * @returns {string} The corresponding Reddit RSS feed URL.
 */
func generateRedditRssUrl(url string) string {
	return strings.TrimRight(url, "/") + "/.rss"
}

/**
 * @function ensureAbsoluteUrl
 * @description Ensures that a given URL is absolute.
 *              If the URL is relative, it resolves it against the base URL.
 * @param {string} baseUrl The base URL to use for resolving relative URLs.
 * @param {string} relativeOrAbsoluteUrl The URL to ensure is absolute.
 * @returns {(string, error)} The absolute URL, or an error if the URL is invalid.
 */
func ensureAbsoluteUrl(baseUrl, relativeOrAbsoluteUrl string) (string, error) {
	// Parse the URL to check if it's absolute
	u, err := url.Parse(relativeOrAbsoluteUrl)
	if err != nil || !u.IsAbs() {
		// If the URL is relative, parse the base URL and resolve the relative URL against it
		u, err = url.Parse(baseUrl)
		if err != nil {
			return "", err
		}
		rel, err := url.Parse(relativeOrAbsoluteUrl)
		if err != nil {
			return "", err
		}
		return u.ResolveReference(rel).String(), nil
	}
	return relativeOrAbsoluteUrl, nil
}
