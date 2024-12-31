// Package main provides the main functionality for the web server.
package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"go.uber.org/zap"
)

/**
 * @function createRSSSearchResponse
 * @description Converts an array of FeedSearchAPIResponseItem to an array of FeedSearchResponseItem.
 * @param {[]FeedSearchAPIResponseItem} apiResults The array of FeedSearchAPIResponseItem to convert.
 * @returns {[]FeedSearchResponseItem} The converted array of FeedSearchResponseItem.
 */
func createRSSSearchResponse(apiResults []FeedSearchAPIResponseItem) []FeedSearchResponseItem {
	var responseItems []FeedSearchResponseItem
	for _, item := range apiResults {
		responseItem := FeedSearchResponseItem{
			Title:        item.Title,
			Url:          item.Url,
			Site_name:    item.Site_name,
			Site_url:     item.Site_url,
			Description:  item.Description,
			Favicon:      item.Favicon,
			Is_Podcast:   item.Is_Podcast,
			Is_Push:      item.Is_Push,
			Item_Count:   item.Item_Count,
			Last_Seen:    item.Last_Seen,
			Last_Updated: item.Last_Updated,
			Score:        item.Score,
		}
		responseItems = append(responseItems, responseItem)
	}
	return responseItems
}

/**
 * @function createPodcastSearchResponse
 * @description Converts an array of PodcastAPIResponseItem to an array of PodcastSearchResponseItem.
 * @param {[]PodcastAPIResponseItem} apiResults The array of PodcastAPIResponseItem to convert.
 * @returns {[]PodcastSearchResponseItem} The converted array of PodcastSearchResponseItem.
 */
func createPodcastSearchResponse(apiResults []PodcastAPIResponseItem) []PodcastSearchResponseItem {
	var responseItems []PodcastSearchResponseItem
	for _, item := range apiResults {
		responseItem := PodcastSearchResponseItem{
			Title:             item.Title,
			Url:               item.Url,
			Author:            item.Author,
			Description:       item.Description,
			FeedImage:         item.Image,
			Image:             item.Image,
			Artwork:           item.Artwork,
			Categories:        item.Categories,
			PodcastGUID:       item.PodcastGUID,
			EpisodeCount:      item.EpisodeCount,
			NewestItemPubdate: item.NewestItemPubdate,
		}
		responseItems = append(responseItems, responseItem)
	}
	return responseItems
}

/**
 * @function searchRSS
 * @description Searches for RSS feeds matching a given URL using an external API.
 *              It caches the results for 24 hours.
 * @param {string} queryURL The URL to search for.
 * @returns {[]FeedSearchResponseItem} An array of FeedSearchResponseItem representing the search results.
 * @dependencies createHash, cache, httpClient, log
 */
func searchRSS(queryURL string) []FeedSearchResponseItem {

	zap.L().Info("[searchRSS] Search request received", zap.String("queryURL", queryURL))
	queryURLCacheKey := createHash(queryURL)

	// Check the cache if the URL has been searched before
	var cachedResults []FeedSearchAPIResponseItem
	if err := cache.Get(feedsearch_prefix, queryURLCacheKey, &cachedResults); err == nil {
		zap.L().Info("[searchRSS] Cache hit", zap.String("queryURL", queryURL))
		return createRSSSearchResponse(cachedResults)
	} else {
		zap.L().Info("[searchRSS] Cache miss", zap.String("queryURL", queryURL))
	}

	// Construct the external API URL
	apiURL := "https://feedsearch.dev/api/v1/search?url=" + url.QueryEscape(queryURL)

	// Make the request to the external API
	resp, err := httpClient.Get(apiURL)
	if err != nil {
		zap.L().Error("[searchRSS] Error making request to external API", zap.String("queryURL", queryURL), zap.String("apiURL", apiURL), zap.Error(err))
		return nil
	}
	defer resp.Body.Close()

	// Read the response from the external API
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		zap.L().Error("[searchRSS] Error reading response from external API", zap.String("queryURL", queryURL), zap.String("apiURL", apiURL), zap.Error(err))
		return nil
	}

	// Unmarshal the JSON response into the FeedSearchAPIResponseItem structure
	var searchResults []FeedSearchAPIResponseItem
	err = json.Unmarshal(body, &searchResults)
	if err != nil {
		zap.L().Error("[searchRSS] Error unmarshalling response from external API", zap.String("queryURL", queryURL), zap.String("apiURL", apiURL), zap.Error(err))
		return nil
	}

	// Convert the API response to the desired response format
	responseItems := createRSSSearchResponse(searchResults)

	// Cache the search results
	if err := cache.Set(feedsearch_prefix, queryURLCacheKey, responseItems, 24*time.Hour); err != nil {
		zap.L().Error("[searchRSS] Failed to cache search results", zap.String("queryURL", queryURL), zap.Error(err))
	} else {
		zap.L().Info("[searchRSS] Successfully cached search results", zap.String("queryURL", queryURL))
	}

	return responseItems
}

/**
 * @function calculateAuth
 * @description Calculates the authorization header for the Podcast Index API.
 * @param {string} key The API key.
 * @param {string} secret The API secret.
 * @param {string} datestr The current date string in Unix timestamp format.
 * @returns {string} The calculated authorization header.
 * @dependencies sha1.New, hex.EncodeToString
 */
func calculateAuth(key, secret, datestr string) string {
	h := sha1.New()
	h.Write([]byte(key + secret + datestr))
	return hex.EncodeToString(h.Sum(nil))
}

/**
 * @function searchPodcast
 * @description Searches for podcasts matching a given query using the Podcast Index API.
 * @param {*http.Request} _ The HTTP request (not used in this function).
 * @param {string} query The search query.
 * @returns {[]PodcastSearchResponseItem} An array of PodcastSearchResponseItem representing the search results.
 * @dependencies calculateAuth, httpClient, log
 */
func searchPodcast(_ *http.Request, query string) []PodcastSearchResponseItem {
	zap.L().Info("[searchPodcast] Search request received", zap.String("query", query))
	key := os.Getenv("PODCAST_INDEX_API_KEY")
	secret := os.Getenv("PODCAST_INDEX_API_SECRET")
	baseURL := "https://api.podcastindex.org/api/1.0/"
	apiURL := baseURL + "search/byterm?q=" + url.QueryEscape(query)

	zap.L().Debug("[searchPodcast] API URL", zap.String("apiURL", apiURL))

	client := &http.Client{}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		zap.L().Error("[searchPodcast] Error creating request", zap.String("query", query), zap.String("apiURL", apiURL), zap.Error(err))
		return nil
	}
	now := strconv.FormatInt(time.Now().Unix(), 10)
	authorization := calculateAuth(key, secret, now)

	req.Header.Set("User-Agent", "MyPodcastApp")
	req.Header.Set("X-Auth-Key", key)
	req.Header.Set("X-Auth-Date", now)
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		zap.L().Error("[searchPodcast] Error making request to Podcast Index API", zap.String("query", query), zap.String("apiURL", apiURL), zap.Error(err))
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		zap.L().Error("[searchPodcast] Error reading response from Podcast Index API", zap.String("query", query), zap.String("apiURL", apiURL), zap.Error(err))
		return nil
	}

	var searchResults PodcastSearchAPIResponse
	err = json.Unmarshal(body, &searchResults)
	if err != nil {
		zap.L().Error("[searchPodcast] Error unmarshalling response from Podcast Index API", zap.String("query", query), zap.String("apiURL", apiURL), zap.Error(err), zap.String("apiResponse", string(body)))
		return nil
	}

	responseItems := createPodcastSearchResponse(searchResults.Items)
	return responseItems
}

/**
 * @function searchHandler
 * @description Handles HTTP requests to the /search endpoint.
 *              It supports searching for both RSS feeds and podcasts based on the 'type' query parameter.
 *              It calls the appropriate search function (searchRSS or searchPodcast) and returns the results as JSON.
 * @param {http.ResponseWriter} w The HTTP response writer.
 * @param {*http.Request} r The HTTP request.
 * @returns {void}
 * @dependencies searchRSS, searchPodcast, log
 */
func searchHandler(w http.ResponseWriter, r *http.Request) {
	// Get the 'url' query parameter
	searchType := r.URL.Query().Get("type")
	switch searchType {
	case "rss":
		queryURL := r.URL.Query().Get("q")
		if queryURL == "" {
			zap.L().Warn("[searchHandler] No url provided for RSS search")
			http.Error(w, "No url provided", http.StatusBadRequest)
			response := map[string]string{
				"status": "error",
				"error":  "No url provided",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
		var searchResults []FeedSearchResponseItem = searchRSS(queryURL)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchResults)

	case "podcast":
		query := r.URL.Query().Get("q")
		if query == "" {
			zap.L().Warn("[searchHandler] No query provided for podcast search")
			http.Error(w, "No query provided", http.StatusBadRequest)
			response := map[string]string{
				"status": "error",
				"error":  "No query provided",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
		var searchResults []PodcastSearchResponseItem = searchPodcast(r, query)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchResults)

	default:
		zap.L().Warn("[searchHandler] No or invalid type provided", zap.String("type", searchType))
		http.Error(w, "No or invalid type provided", http.StatusBadRequest)
		response := map[string]string{
			"status": "error",
			"error":  "No or invalid type provided",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
