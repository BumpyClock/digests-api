// Package main provides the main functionality for the web server.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"

	"github.com/mmcdole/gofeed"
	"github.com/sirupsen/logrus"
)

// The date/time layout format used throughout the code.
const layout = "2006-01-02T15:04:05Z07:00"

/**
 * @function createHash
 * @description Computes the SHA-256 hash of a given string.
 * @param {string} s The string to hash.
 * @returns {string} The hex-encoded SHA-256 hash of the string.
 */
func createHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

/**
 * @function parseHTMLContent
 * @description Parses HTML content and extracts the text content.
 *              If parsing fails, it returns the original content.
 * @param {string} htmlContent The HTML content to parse.
 * @returns {string} The extracted text content, or the original htmlContent if parsing fails.
 */
func parseHTMLContent(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		// Fallback to the raw HTML if parse fails
		log.WithFields(logrus.Fields{
			"error": err,
		}).Warn("[parseHTMLContent] Failed to parse HTML content")
		return htmlContent
	}
	var f func(*html.Node)
	var textContent strings.Builder

	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			textContent.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return textContent.String()
}

/**
 * @function getBaseDomain
 * @description Extracts the base domain (scheme + hostname) from a URL.
 * @param {string} rawURL The URL to parse.
 * @returns {string} The base domain of the URL, or an empty string if parsing fails.
 */
func getBaseDomain(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		log.WithFields(logrus.Fields{
			"url":   rawURL,
			"error": err,
		}).Warn("[getBaseDomain] Failed to parse URL")
		return ""
	}
	return parsedURL.Scheme + "://" + parsedURL.Host
}

/**
 * @function metadataHandler
 * @description Handles HTTP requests to the /metadata endpoint.
 *              It expects a POST request with a JSON body containing an array of URLs.
 *              It fetches metadata for each URL using GetMetaData and returns the results as JSON.
 * @param {http.ResponseWriter} w The HTTP response writer.
 * @param {*http.Request} r The HTTP request.
 * @returns {void}
 */
func metadataHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		log.WithFields(logrus.Fields{
			"method": r.Method,
		}).Warn("[metadataHandler] Invalid method")
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decode the request body
	var urls Urls
	err := json.NewDecoder(r.Body).Decode(&urls)
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Error("[metadataHandler] Error decoding request body")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Process each URL concurrently
	var wg sync.WaitGroup
	results := make([]MetaDataResponseItem, len(urls.Urls))

	for i, url := range urls.Urls {
		wg.Add(1)
		go func(i int, url string) {
			defer wg.Done()
			log.WithFields(logrus.Fields{
				"url": url,
			}).Debug("[metadataHandler] Fetching metadata")

			// Fetch metadata for the URL
			result, err := GetMetaData(url)
			if err != nil {
				log.WithFields(logrus.Fields{
					"url":   url,
					"error": err,
				}).Error("[metadataHandler] Error fetching metadata")
			} else {
				results[i] = result
			}
		}(i, url)
	}

	wg.Wait()

	// Encode the results as JSON and send the response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]MetaDataResponseItem{"metadata": results})
}

/**
 * @function parseHandler
 * @description Handles HTTP requests to the /parse endpoint.
 *              It expects a POST request with a JSON body containing an array of feed URLs to parse,
 *              along with optional pagination parameters (Page, ItemsPerPage).
 *              It processes each feed URL, applies pagination, and returns the results as JSON.
 * @param {http.ResponseWriter} w The HTTP response writer.
 * @param {*http.Request} r The HTTP request.
 * @returns {void}
 */
func parseHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		log.WithFields(logrus.Fields{
			"method": r.Method,
		}).Warn("[parseHandler] Invalid method")
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	// Default values for pagination
	page := 1
	itemsPerPage := 50

	// Decode request into ParseRequest
	req, err := decodeRequest(r)
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Error("[parseHandler] Error decoding request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// If user provided a page > 0, use it; otherwise keep default
	if req.Page > 0 {
		page = req.Page
	}
	// If user provided itemsPerPage > 0, use it; otherwise keep default
	if req.ItemsPerPage > 0 {
		itemsPerPage = req.ItemsPerPage
	}

	// Process the URLs and send the response
	responses := processURLs(req.URLs, page, itemsPerPage)
	sendResponse(w, responses)
}

/**
 * @function decodeRequest
 * @description Decodes the request body into a ParseRequest object.
 * @param {*http.Request} r The HTTP request.
 * @returns {(ParseRequest, error)} The parsed ParseRequest object, or an error if unmarshalling fails.
 */
func decodeRequest(r *http.Request) (ParseRequest, error) {
	var req ParseRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

/**
 * @function processURLs
 * @description Concurrently processes each feed URL using processURL.
 *              Returns all feed responses (with pagination applied) as a slice.
 * @param {[]string} urls The list of feed URLs to process.
 * @param {int} page The page number for pagination.
 * @param {int} itemsPerPage The number of items per page for pagination.
 * @returns {[]FeedResponse} A slice of FeedResponse objects, with pagination applied.
 */
func processURLs(urls []string, page, itemsPerPage int) []FeedResponse {
	var wg sync.WaitGroup
	sem := make(chan struct{}, numWorkers)
	responses := make(chan FeedResponse, len(urls))

	for _, url := range urls {
		wg.Add(1)
		sem <- struct{}{}
		go func(feedURL string) {
			defer func() {
				wg.Done()
				<-sem
			}()
			log.WithFields(logrus.Fields{
				"url": feedURL,
			}).Debug("[processURLs] Processing feed")

			// Process each URL
			response := processURL(feedURL, page, itemsPerPage)
			responses <- response
		}(url)
	}

	go func() {
		wg.Wait()
		close(responses)
	}()

	return collectResponses(responses)
}

/**
 * @function isCacheStale
 * @description Checks if the cached data is stale based on the last refresh time and the refresh timer.
 * @param {string} lastRefreshed The timestamp of the last refresh, in the format defined by the 'layout' constant.
 * @returns {bool} True if the cache is stale, false otherwise.
 */
func isCacheStale(lastRefreshed string) bool {
	parsedTime, err := time.Parse(layout, lastRefreshed)
	if err != nil {
		log.WithFields(logrus.Fields{
			"lastRefreshed": lastRefreshed,
			"error":         err,
		}).Error("[isCacheStale] Failed to parse LastRefreshed")
		return false
	}

	if time.Since(parsedTime) > time.Duration(refresh_timer)*time.Minute {
		log.WithFields(logrus.Fields{
			"lastRefreshed": lastRefreshed,
			"refresh_timer": refresh_timer,
		}).Info("[isCacheStale] Cache is stale")
		return true
	}
	return false
}

/**
 * @function fetchAndCacheFeed
 * @description Fetches and parses a feed from a given URL, merges it with existing items from the cache,
 *              and caches the result.
 * @param {string} feedURL The URL of the feed to fetch.
 * @param {string} cacheKey The key to use for caching the feed.
 * @returns {(FeedResponse, error)} The fetched and processed FeedResponse, or an error if any step fails.
 */
func fetchAndCacheFeed(feedURL, cacheKey string) (FeedResponse, error) {
	parser := gofeed.NewParser()
	feed, err := parser.ParseURL(feedURL)
	if err != nil {
		log.WithFields(logrus.Fields{
			"url":   feedURL,
			"error": err,
		}).Error("[fetchAndCacheFeed] Failed to parse feedURL")
		return FeedResponse{}, err
	}

	// Convert newly fetched feed to items.
	newItems := processFeedItems(feed)

	// Merge new items with existing items from the last 24 hours.
	mergedItems, mergeErr := mergeFeedItemsAtParserLevel(feedURL, newItems)
	if mergeErr != nil {
		log.WithFields(logrus.Fields{
			"url":   feedURL,
			"error": mergeErr,
		}).Error("[fetchAndCacheFeed] Failed to merge feed items")
		return FeedResponse{}, mergeErr
	}

	baseDomain := getBaseDomain(feed.Link)
	addURLToList(feedURL)

	// Possibly fetch additional metadata from cache
	var metaData MetaDataResponseItem

	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	baseDomainKey := createHash(baseDomain)
	if err := cache.Get(metaData_prefix, baseDomainKey, &metaData); err != nil {
		if isValidURL(baseDomain) {
			tempMeta, errGet := GetMetaData(baseDomain)
			if errGet != nil {
				log.WithFields(logrus.Fields{
					"baseDomain": baseDomain,
					"error":      errGet,
				}).Warn("[fetchAndCacheFeed] Failed to get metadata")
			} else {
				metaData = tempMeta
				if errSet := cache.Set(metaData_prefix, baseDomainKey, metaData, 24*time.Hour); errSet != nil {
					log.WithFields(logrus.Fields{
						"baseDomain": baseDomain,
						"error":      errSet,
					}).Error("[fetchAndCacheFeed] Failed to cache metadata")
				}
			}
		} else {
			metaData = MetaDataResponseItem{}
			log.WithFields(logrus.Fields{
				"baseDomain": baseDomain,
			}).Warn("[fetchAndCacheFeed] Invalid baseDomain")
		}
	} else {
		log.WithFields(logrus.Fields{
			"baseDomain": baseDomain,
		}).Debug("[fetchAndCacheFeed] Loaded metadata from cache")
	}

	// Build final FeedResponse from the merged items
	finalFeedResponse := createFeedResponse(feed, feedURL, metaData, mergedItems)

	// Cache the final feed response
	if err := cache.Set(feed_prefix, cacheKey, finalFeedResponse, 24*time.Hour); err != nil {
		log.WithFields(logrus.Fields{
			"url":   feedURL,
			"error": err,
		}).Error("[fetchAndCacheFeed] Failed to cache feed details")
		return FeedResponse{}, err
	}

	log.WithFields(logrus.Fields{
		"url": feedURL,
	}).Info("[fetchAndCacheFeed] Successfully cached feed details")
	return finalFeedResponse, nil
}

/**
 * @function processURL
 * @description Processes a single feed URL. It first checks the cache for the feed.
 *              If the feed is found in the cache and is not stale, it returns the cached feed.
 *              Otherwise, it fetches the feed, caches it, and returns the result.
 *              Pagination is applied to the final list of items before returning.
 * @param {string} rawURL The raw URL of the feed to process.
 * @param {int} page The page number for pagination.
 * @param {int} itemsPerPage The number of items per page for pagination.
 * @returns {FeedResponse} The processed FeedResponse, with pagination applied.
 */
func processURL(rawURL string, page, itemsPerPage int) FeedResponse {
	feedURL := sanitizeURL(rawURL)
	cacheKey := feedURL

	var cachedFeed FeedResponse
	// Try retrieving from cache first
	if err := cache.Get(feed_prefix, cacheKey, &cachedFeed); err == nil && cachedFeed.SiteTitle != "" {
		// Cache hit
		log.WithFields(logrus.Fields{
			"url": feedURL,
		}).Info("[processURL] [Cache Hit] Using cached feed details")

		// Check staleness
		if isCacheStale(cachedFeed.LastRefreshed) {
			log.WithFields(logrus.Fields{
				"url": feedURL,
			}).Info("[processURL] Cache is stale, refreshing in background")
			go func() {
				if _, errRefresh := fetchAndCacheFeed(feedURL, cacheKey); errRefresh != nil {
					log.WithFields(logrus.Fields{
						"url":   feedURL,
						"error": errRefresh,
					}).Error("[processURL] Failed to refresh feed in background")
				}
			}()
		}

		// Optionally re-check or skip thumbnail colors
		updatedItems := updateFeedItemsWithThumbnailColors(cachedFeed.Items)
		// Reassign updated items
		cachedFeed.Items = &updatedItems

		// **Apply pagination** to the final items
		applyPagination(cachedFeed.Items, page, itemsPerPage)

		return cachedFeed
	}

	// Cache miss or empty feed
	log.WithFields(logrus.Fields{
		"url": feedURL,
	}).Info("[processURL] [Cache Miss] Fetching fresh feed")

	newResp, errNew := fetchAndCacheFeed(feedURL, cacheKey)
	if errNew != nil {
		log.WithFields(logrus.Fields{
			"url":   feedURL,
			"error": errNew,
		}).Error("[processURL] Failed to fetch and cache feed")

		return FeedResponse{
			Type:    "unknown",
			FeedUrl: feedURL,
			GUID:    cacheKey,
			Status:  "error",
			Error:   errNew,
		}
	}

	// Optionally re-check or skip thumbnail colors
	updatedItems := updateFeedItemsWithThumbnailColors(newResp.Items)
	newResp.Items = &updatedItems

	// **Apply pagination** to the final items
	applyPagination(newResp.Items, page, itemsPerPage)

	return newResp
}

/**
 * @function applyPagination
 * @description Applies pagination to a slice of FeedResponseItem objects.
 * @param {*[]FeedResponseItem} items A pointer to a slice of FeedResponseItem objects.
 * @param {int} page The page number to retrieve.
 * @param {int} itemsPerPage The number of items per page.
 * @returns {void}
 */
func applyPagination(items *[]FeedResponseItem, page, itemsPerPage int) {
	if items == nil || len(*items) == 0 {
		return
	}
	if page < 1 {
		page = 1
	}
	if itemsPerPage < 1 {
		itemsPerPage = 1
	}

	totalItems := len(*items)
	start := (page - 1) * itemsPerPage
	if start >= totalItems {
		// If start is beyond total items, return empty
		*items = []FeedResponseItem{}
		return
	}

	end := start + itemsPerPage
	if end > totalItems {
		end = totalItems
	}

	// Slice the items
	*items = (*items)[start:end]
}

/**
 * @function updateFeedItemsWithThumbnailColors
 * @description Updates the thumbnail colors for a slice of FeedResponseItem objects.
 * @param {*[]FeedResponseItem} items A pointer to a slice of FeedResponseItem objects.
 * @returns {[]FeedResponseItem} A new slice of FeedResponseItem objects with updated thumbnail colors.
 */
func updateFeedItemsWithThumbnailColors(items *[]FeedResponseItem) []FeedResponseItem {
	if items == nil {
		return nil
	}
	var updatedItems []FeedResponseItem
	for _, item := range *items {
		updatedItem := updateThumbnailColorForItem(item)
		updatedItems = append(updatedItems, updatedItem)
	}
	return updatedItems
}

/**
 * @function updateThumbnailColorForItem
 * @description Updates the thumbnail color for a single FeedResponseItem object.
 * @param {FeedResponseItem} item The FeedResponseItem to update.
 * @returns {FeedResponseItem} The updated FeedResponseItem.
 */
func updateThumbnailColorForItem(item FeedResponseItem) FeedResponseItem {
	var cachedColor RGBColor
	err := cache.Get(thumbnailColor_prefix, item.Thumbnail, &cachedColor)

	switch {
	case err != nil:
		log.WithFields(logrus.Fields{
			"thumbnail": item.Thumbnail,
		}).Debug("[updateThumbnailColorForItem] No cached color")
	case item.ThumbnailColorComputed == "set":
		// Already set
	case item.ThumbnailColorComputed == "computed":
		item.ThumbnailColor = cachedColor
		item.ThumbnailColorComputed = "set"
		log.WithFields(logrus.Fields{
			"thumbnail": item.Thumbnail,
			"color":     item.ThumbnailColor,
		}).Debug("[updateThumbnailColorForItem] Updated color")
	case item.ThumbnailColorComputed == "no":
		if cachedColor != (RGBColor{}) {
			item.ThumbnailColor = cachedColor
			item.ThumbnailColorComputed = "set"
			log.WithFields(logrus.Fields{
				"thumbnail": item.Thumbnail,
				"color":     item.ThumbnailColor,
			}).Debug("[updateThumbnailColorForItem] Updated color")
		}
	default:
		// No additional logic
	}
	return item
}

/**
 * @function processFeedItems
 * @description Processes a list of feed items concurrently.
 * @param {*gofeed.Feed} feed The parsed feed to process.
 * @returns {[]FeedResponseItem} A slice of processed FeedResponseItem objects.
 */
func processFeedItems(feed *gofeed.Feed) []FeedResponseItem {
	// Safeguard feed == nil or feed.Items is nil/empty
	if feed == nil {
		log.Error("[processFeedItems] feed is nil; returning empty slice")
		return nil
	}
	if len(feed.Items) == 0 {
		log.WithFields(logrus.Fields{
			"feedTitle": feed.Title,
		}).Warn("[processFeedItems] feed.Items is empty")
		return nil
	}

	thumbnail := ""
	defaultThumbnailColor := RGBColor{128, 128, 128}
	// If iTunes image is present, compute default color
	if feed.ITunesExt != nil && feed.ITunesExt.Image != "" {
		thumbnail = feed.ITunesExt.Image
		r, g, b := extractColorFromThumbnail_prominentColor(thumbnail)
		defaultThumbnailColor = RGBColor{r, g, b}
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, numWorkers)
	itemResponses := make(chan FeedResponseItem, len(feed.Items))

	for _, item := range feed.Items {
		if item == nil {
			log.Warn("[processFeedItems] Skipping nil item")
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(it *gofeed.Item) {
			defer func() {
				wg.Done()
				<-sem
			}()
			itemResponse := processFeedItem(it, thumbnail, defaultThumbnailColor)
			itemResponses <- itemResponse
		}(item)
	}

	go func() {
		wg.Wait()
		close(itemResponses)
	}()

	return collectItemResponses(itemResponses)
}

/**
 * @function processFeedItem
 * @description Processes a single feed item, extracting relevant information and optionally computing the thumbnail color.
 * @param {*gofeed.Item} item The feed item to process.
 * @param {string} thumbnail The default thumbnail URL for the feed.
 * @param {RGBColor} thumbnailColor The default thumbnail color for the feed.
 * @returns {FeedResponseItem} The processed FeedResponseItem.
 */
func processFeedItem(item *gofeed.Item, thumbnail string, thumbnailColor RGBColor) FeedResponseItem {
	author := getItemAuthor(item)
	categories := strings.Join(item.Categories, ", ")

	// Possibly override the feed-level thumbnail with item enclosures
	if len(item.Enclosures) > 0 {
		for _, enclosure := range item.Enclosures {
			if enclosure.URL != "" && strings.HasPrefix(enclosure.Type, "image/") {
				thumbnail = enclosure.URL
				break
			}
		}
	}
	if thumbnail == "" && item.Image != nil && item.Image.URL != "" {
		thumbnail = item.Image.URL
	}
	if item.ITunesExt != nil && item.ITunesExt.Image != "" {
		thumbnail = item.ITunesExt.Image
	}

	// Attempt to discover a thumbnail from content if still empty
	if thumbnail == "" {
		finder := NewThumbnailFinder()
		discovered := finder.FindThumbnailForItem(item)
		if discovered != "" {
			thumbnail = discovered
		}
	}

	thumbnailColorComputed := "no"
	// If thumbnail color is cached, set it directly; otherwise compute async
	var cachedColor RGBColor
	cacheMutex.Lock()
	err := cache.Get(thumbnailColor_prefix, thumbnail, &cachedColor)
	cacheMutex.Unlock()
	if err == nil {
		thumbnailColor = cachedColor
		thumbnailColorComputed = "set"
	} else {
		// Async compute if not cached
		go func(thURL string) {
			if thURL == "" {
				return
			}
			r, g, b := extractColorFromThumbnail_prominentColor(thURL)
			actualColor := RGBColor{r, g, b}
			if cErr := cache.Set(thumbnailColor_prefix, thURL, actualColor, 24*time.Hour); cErr != nil {
				log.WithFields(logrus.Fields{
					"thumbnail": thURL,
					"color":     actualColor,
					"error":     cErr,
				}).Error("[processFeedItem] Failed to cache color")
			}
		}(thumbnail)
	}

	desc := item.Description
	if desc == "" {
		desc = parseHTMLContent(item.Content)
	}
	desc = parseHTMLContent(desc)

	// Standardize item published date
	standardizedPublished := standardizeDate(item.Published)

	// Identify if it's a podcast and parse duration if so
	itemType, duration := determineItemTypeAndDuration(item)

	return FeedResponseItem{
		Type:                   itemType,
		ID:                     createHash(item.Link),
		Title:                  item.Title,
		Description:            desc,
		Link:                   item.Link,
		Duration:               duration,
		Author:                 author,
		Published:              standardizedPublished,
		Created:                standardizedPublished,
		Content:                parseHTMLContent(item.Content),
		Content_Encoded:        item.Content,
		Categories:             categories,
		Enclosures:             item.Enclosures,
		Thumbnail:              thumbnail,
		ThumbnailColor:         thumbnailColor,
		ThumbnailColorComputed: thumbnailColorComputed,
	}
}

/**
 * @function standardizeDate
 * @description Standardizes a given date string to the common layout used in the application.
 * @param {string} dateStr The date string to standardize.
 * @returns {string} The standardized date string, or an empty string if parsing fails.
 */
func standardizeDate(dateStr string) string {
	if dateStr == "" {
		log.Info("[standardizeDate] Empty date string")
		return ""
	}
	const outputLayout = "2006-01-02T15:04:05Z07:00"
	dateFormats := []string{
		time.RFC1123,
		time.RFC1123Z,
		time.RFC3339,
		time.RFC822,
		time.RFC850,
		time.ANSIC,
		"Mon, 02 Jan 2006 15:04:05 -0700",
	}
	for _, layout := range dateFormats {
		if parsedTime, err := time.Parse(layout, dateStr); err == nil {
			return parsedTime.Format(outputLayout)
		}
	}
	log.WithFields(logrus.Fields{
		"date": dateStr,
	}).Info("[standardizeDate] Failed to parse date")
	return ""
}

/**
 * @function createFeedResponse
 * @description Creates a FeedResponse object from a parsed feed, metadata, and processed feed items.
 * @param {*gofeed.Feed} feed The parsed feed.
 * @param {string} feedURL The URL of the feed.
 * @param {MetaDataResponseItem} metaData The metadata associated with the feed.
 * @param {[]FeedResponseItem} feedItems The processed feed items.
 * @returns {FeedResponse} The constructed FeedResponse object.
 */
func createFeedResponse(feed *gofeed.Feed, feedURL string, metaData MetaDataResponseItem, feedItems []FeedResponseItem) FeedResponse {
	if feed == nil {
		log.WithFields(logrus.Fields{
			"url": feedURL,
		}).Error("[createFeedResponse] feed is nil")
		return FeedResponse{}
	}

	var feedType, thumbnail string
	if feed.ITunesExt != nil {
		feedType = "podcast"
		if feed.Image != nil && feed.Image.URL != "" {
			thumbnail = feed.Image.URL
		}
	} else {
		feedType = "article"
		if metaData.Favicon != "" {
			thumbnail = metaData.Favicon
		} else if feed.Image != nil && feed.Image.URL != "" {
			thumbnail = feed.Image.URL
		}
	}

	siteTitle := metaData.Title
	if siteTitle == "" {
		siteTitle = feed.Title
	}

	return FeedResponse{
		Status:        "ok",
		GUID:          createHash(feedURL),
		Type:          feedType,
		SiteTitle:     siteTitle,
		FeedTitle:     feed.Title,
		FeedUrl:       feedURL,
		Description:   feed.Description,
		Link:          metaData.Domain,
		LastUpdated:   standardizeDate(feed.Updated),
		LastRefreshed: time.Now().Format(layout),
		Published:     feed.Published,
		Author:        feed.Author,
		Language:      feed.Language,
		Favicon:       thumbnail,
		Categories:    strings.Join(feed.Categories, ", "),
		Items:         &feedItems,
	}
}

/**
 * @function collectResponses
 * @description Collects FeedResponse objects from a channel into a slice.
 * @param {chan FeedResponse} responses The channel to receive FeedResponse objects from.
 * @returns {[]FeedResponse} A slice of FeedResponse objects.
 */
func collectResponses(responses chan FeedResponse) []FeedResponse {
	var all []FeedResponse
	for resp := range responses {
		all = append(all, resp)
	}
	return all
}

/**
 * @function collectItemResponses
 * @description Collects FeedResponseItem objects from a channel into a slice and sorts them by published date.
 * @param {chan FeedResponseItem} itemResponses The channel to receive FeedResponseItem objects from.
 * @returns {[]FeedResponseItem} A slice of FeedResponseItem objects sorted by published date in descending order.
 */
func collectItemResponses(itemResponses chan FeedResponseItem) []FeedResponseItem {
	var feedItems []FeedResponseItem
	for itemResponse := range itemResponses {
		feedItems = append(feedItems, itemResponse)
	}
	// Sort by published date (descending)
	sort.Slice(feedItems, func(i, j int) bool {
		timeI, errI := time.Parse(layout, feedItems[i].Published)
		if errI != nil {
			log.WithFields(logrus.Fields{
				"item":  feedItems[i],
				"error": errI,
			}).Error("[collectItemResponses] Failed to parse time for item I")
			return false
		}
		timeJ, errJ := time.Parse(layout, feedItems[j].Published)
		if errJ != nil {
			log.WithFields(logrus.Fields{
				"item":  feedItems[j],
				"error": errJ,
			}).Error("[collectItemResponses] Failed to parse time for item J")
			return true
		}
		return timeI.After(timeJ)
	})
	return feedItems
}

/**
 * @function sendResponse
 * @description Sends a JSON response containing a list of FeedResponse objects.
 * @param {http.ResponseWriter} w The HTTP response writer.
 * @param {[]FeedResponse} responses The list of FeedResponse objects to send.
 * @returns {void}
 */
func sendResponse(w http.ResponseWriter, responses []FeedResponse) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)

	feeds := Feeds{Feeds: responses}
	if err := enc.Encode(feeds); err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Error("[sendResponse] Failed to encode feed responses")
	}
}

/**
 * @function refreshFeeds
 * @description Refreshes all cached feeds by retrieving them from the cache and reprocessing them.
 * @returns {void}
 */
func refreshFeeds() {
	urls := getAllCachedURLs()
	for _, url := range urls {
		log.WithFields(logrus.Fields{
			"url": url,
		}).Info("[refreshFeeds] Refreshing feed")
		_ = processURL(url, 1, 20) // or any default paging
	}
}

/**
 * @function addURLToList
 * @description Adds a URL to the list of URLs if it's not already present.
 * @param {string} url The URL to add.
 * @returns {void}
 */
func addURLToList(url string) {
	urlListMutex.Lock()
	defer urlListMutex.Unlock()

	if !stringInSlice(url, urlList) {
		urlList = append(urlList, url)
	}
}

/**
 * @function getAllCachedURLs
 * @description Retrieves all cached URLs from the cache or the in-memory list.
 * @returns {[]string} A slice of cached URLs.
 */
func getAllCachedURLs() []string {
	urlListMutex.Lock()
	defer urlListMutex.Unlock()

	if len(urlList) == 0 {
		startTime := time.Now()

		var err error
		urlList, err = cache.GetSubscribedListsFromCache(feed_prefix)
		if err != nil {
			log.WithFields(logrus.Fields{
				"error": err,
			}).Warn("[getAllCachedURLs] Failed to get subscribed feeds from cache")
			return nil
		}

		duration := time.Since(startTime)
		log.WithFields(logrus.Fields{
			"urlList":  urlList,
			"duration": duration,
		}).Info("[getAllCachedURLs] Loaded urlList from cache")
	}

	// return a copy
	return append([]string(nil), urlList...)
}

/**
 * @function isValidURL
 * @description Checks if a given string is a valid URL with a resolvable host.
 * @param {string} str The string to check.
 * @returns {bool} True if the string is a valid URL, false otherwise.
 */
func isValidURL(str string) bool {
	parsedURL, err := url.ParseRequestURI(str)
	if err != nil {
		log.WithFields(logrus.Fields{
			"url":   str,
			"error": err,
		}).Info("[isValidURL] Invalid URL")
		return false
	}
	host := parsedURL.Hostname()
	if net.ParseIP(host) != nil {
		return true
	}
	return strings.Contains(host, ".")
}

/**
 * @function sanitizeURL
 * @description Sanitizes a URL by ensuring it has an https:// scheme.
 * @param {string} rawURL The URL to sanitize.
 * @returns {string} The sanitized URL.
 */
func sanitizeURL(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.Scheme == "" {
		return "https://" + rawURL
	} else if parsedURL.Scheme == "http" {
		return strings.Replace(rawURL, "http://", "https://", 1)
	}
	return rawURL
}

/**
 * @function getItemAuthor
 * @description Extracts the author of a feed item, prioritizing the iTunes extension author if available.
 * @param {*gofeed.Item} item The feed item.
 * @returns {string} The author of the feed item, or an empty string if not found.
 */
func getItemAuthor(item *gofeed.Item) string {
	if item.ITunesExt != nil && item.ITunesExt.Author != "" {
		return item.ITunesExt.Author
	}
	if item.Author != nil && item.Author.Name != "" {
		return item.Author.Name
	}
	return ""
}

/**
 * @function determineItemTypeAndDuration
 * @description Determines the type of a feed item (podcast or article) and extracts the duration for podcasts.
 * @param {*gofeed.Item} item The feed item.
 * @returns {(string, int)} The type of the feed item and the duration in seconds (0 for articles).
 */
func determineItemTypeAndDuration(item *gofeed.Item) (string, int) {
	if item.ITunesExt != nil {
		return "podcast", parseDuration(item.ITunesExt.Duration)
	}
	return "article", 0
}

/**
 * @function parseDuration
 * @description Parses a duration string in HH:MM:SS format or seconds into an integer representing the duration in seconds.
 * @param {string} durationStr The duration string to parse.
 * @returns {int} The duration in seconds, or 0 if parsing fails.
 */
func parseDuration(durationStr string) int {
	if durationStr == "" {
		return 0
	}

	// Try integer first
	if durationInt, err := strconv.Atoi(durationStr); err == nil {
		return durationInt
	}

	// Possibly HH:MM:SS or MM:SS
	parts := strings.Split(durationStr, ":")
	switch len(parts) {
	case 3:
		hours, _ := strconv.Atoi(parts[0])
		minutes, _ := strconv.Atoi(parts[1])
		seconds, _ := strconv.Atoi(parts[2])
		return hours*3600 + minutes*60 + seconds
	case 2:
		minutes, _ := strconv.Atoi(parts[0])
		seconds, _ := strconv.Atoi(parts[1])
		return minutes*60 + seconds
	default:
		return 0
	}
}

/**
 * @function stringInSlice
 * @description Checks if a string is present in a slice of strings.
 * @param {string} str The string to search for.
 * @param {[]string} list The slice of strings to search in.
 * @returns {bool} True if the string is found in the slice, false otherwise.
 */
func stringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}

/**
 * @function mergeFeedItemsAtParserLevel
 * @description Merges new feed items with existing cached items, deduplicating by ID and removing old items.
 * @param {string} feedURL The URL of the feed.
 * @param {[]FeedResponseItem} newItems The new feed items to merge.
 * @returns {([]FeedResponseItem, error)} The merged slice of FeedResponseItem objects, or an error if retrieval from cache fails.
 */
func mergeFeedItemsAtParserLevel(feedURL string, newItems []FeedResponseItem) ([]FeedResponseItem, error) {
	cacheKey := feedURL
	var existingFeedResponse FeedResponse
	var existingItems []FeedResponseItem

	// Attempt to get an existing feed from the cache
	if err := cache.Get(feed_prefix, cacheKey, &existingFeedResponse); err != nil {
		log.WithFields(logrus.Fields{
			"url":   feedURL,
			"error": err,
		}).Error("[mergeFeedItemsAtParserLevel] Error getting existing items")
		existingItems = nil
	} else {
		if existingFeedResponse.Items != nil {
			existingItems = *existingFeedResponse.Items
			log.WithFields(logrus.Fields{
				"url":   feedURL,
				"count": len(existingItems),
			}).Debug("[mergeFeedItemsAtParserLevel] Found existing items")
		}
	}

	itemMap := make(map[string]FeedResponseItem)
	log.WithFields(logrus.Fields{
		"url":      feedURL,
		"existing": len(existingItems),
		"new":      len(newItems),
	}).Debug("[mergeFeedItemsAtParserLevel] Merging items")
	for _, oldItem := range existingItems {
		if isWithinPeriod(oldItem, cachePeriod) {
			itemMap[oldItem.ID] = oldItem
		}
	}

	// Merge new items
	for _, newIt := range newItems {
		if oldIt, found := itemMap[newIt.ID]; found {
			if isUpdatedContent(oldIt, newIt) {
				itemMap[newIt.ID] = newIt
			}
		} else {
			if isWithinPeriod(newIt, cachePeriod) {
				itemMap[newIt.ID] = newIt
			}
		}
	}

	merged := make([]FeedResponseItem, 0, len(itemMap))
	for _, v := range itemMap {
		merged = append(merged, v)
	}

	// Store merged items in the cache so subsequent fetches have updated items
	if err := cache.Set(feed_prefix, feedURL, merged, 24*time.Hour); err != nil {
		log.WithFields(logrus.Fields{
			"url":   feedURL,
			"error": err,
		}).Error("[mergeFeedItemsAtParserLevel] Failed to cache merged items")
		return nil, err
	}

	return merged, nil
}

/**
 * @function isWithinPeriod
 * @description Checks if a FeedResponseItem's published date is within a certain number of days from the current time.
 * @param {FeedResponseItem} item The FeedResponseItem to check.
 * @param {int} days The number of days to check within.
 * @returns {bool} True if the item's published date is within the specified period, false otherwise.
 */
func isWithinPeriod(item FeedResponseItem, days int) bool {
	t, err := time.Parse(layout, item.Published)
	if err != nil {
		return false
	}
	return time.Since(t) <= time.Duration(days)*24*time.Hour
}

/**
 * @function isUpdatedContent
 * @description Checks if a new FeedResponseItem has updated content compared to an old one based on published date and content.
 * @param {FeedResponseItem} oldIt The old FeedResponseItem.
 * @param {FeedResponseItem} newIt The new FeedResponseItem.
 * @returns {bool} True if the new item is more recent or has different content, false otherwise.
 */
func isUpdatedContent(oldIt, newIt FeedResponseItem) bool {
	oldTime, _ := time.Parse(layout, oldIt.Published)
	newTime, _ := time.Parse(layout, newIt.Published)

	// If new item is more recent
	if newTime.After(oldTime) {
		return true
	}
	// If content changed
	if newIt.Content != oldIt.Content {
		return true
	}
	return false
}
