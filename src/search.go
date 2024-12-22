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
)

// createRSSSearchResponse converts an array of FeedSearchAPIResponseItem to an array of FeedSearchResponseItem
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

func searchRSS(queryURL string) []FeedSearchResponseItem {

	log.Println("Search request received for URL: ", queryURL)
	queryURLCacheKey := createHash(queryURL)

	// Check the cache if the URL has been searched before
	var cachedResults []FeedSearchResponseItem
	if err := cache.Get(feedsearch_prefix, queryURLCacheKey, &cachedResults); err == nil {
		log.Println("Cache hit for URL: ", queryURL)
		return cachedResults
	} else {
		log.Println("Cache miss for URL: ", queryURL)
	}

	// Construct the external API URL
	apiURL := "https://feedsearch.dev/api/v1/search?url=" + url.QueryEscape(queryURL)

	// Make the request to the external API
	resp, err := http.Get(apiURL)
	if err != nil {
		log.Println("Error making request to external API: ", err)
		return nil
	}
	defer resp.Body.Close()

	// Read the response from the external API
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading response from external API: ", err)
		return nil
	}

	// Unmarshal the JSON response into the FeedSearchAPIResponseItem structure
	var searchResults []FeedSearchAPIResponseItem
	err = json.Unmarshal(body, &searchResults)
	if err != nil {
		log.Println("Error unmarshalling response from external API: ", err)
		return nil
	}

	// Convert the API response to the desired response format
	responseItems := createRSSSearchResponse(searchResults)

	// Cache the search results
	if err := cache.Set(feedsearch_prefix, queryURLCacheKey, responseItems, 24*time.Hour); err != nil {
		log.Printf("Failed to cache search results for URL %s: %v", queryURL, err)
	} else {
		log.Printf("Successfully cached search results for URL %s", queryURL)
	}

	return responseItems
}
func calculateAuth(key, secret, datestr string) string {

	h := sha1.New()
	h.Write([]byte(key + secret + datestr))

	log.Println("Hash Calculated as: ", h)
	return hex.EncodeToString(h.Sum(nil))
}
func searchPodcast(r *http.Request, query string) []PodcastSearchResponseItem {
	log.Println("Search request received for Podcast with query: ", query)
	key := os.Getenv("PODCAST_INDEX_API_KEY")
	secret := os.Getenv("PODCAST_INDEX_API_SECRET")
	baseURL := "https://api.podcastindex.org/api/1.0/"
	apiURL := baseURL + "search/byterm?q=" + url.QueryEscape(query)

	log.Println("API URL: ", apiURL)
	log.Println("API Key: ", key)
	log.Println("API Secret: ", secret)
	log.Println("Request Headers: ", apiURL)

	client := &http.Client{}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Println("Error creating request: ", err)
		return nil
	}
	now := strconv.FormatInt(time.Now().Unix(), 10)
	authorization := calculateAuth(key, secret, now)
	log.Println("Authorization: ", authorization)
	log.Println("Date: ", now)
	log.Println("User-Agent: MyPodcastApp")

	req.Header.Set("User-Agent", "MyPodcastApp")
	req.Header.Set("X-Auth-Key", key)
	req.Header.Set("X-Auth-Date", now)
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error making request to Podcast Index API: ", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading response from Podcast Index API: ", err)
		return nil
	}

	var searchResults PodcastSearchAPIResponse
	err = json.Unmarshal(body, &searchResults)
	if err != nil {
		log.Println("API Response: ", resp.Body)
		log.Println("Error unmarshalling response from Podcast Index API: ", err)
		return nil
	}

	responseItems := createPodcastSearchResponse(searchResults.Items)
	return responseItems
}
