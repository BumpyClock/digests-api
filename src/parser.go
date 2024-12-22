// parser.go
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

	link2json "github.com/BumpyClock/go-link2json"
	"github.com/mmcdole/gofeed"
	"github.com/sirupsen/logrus"
)

const layout = "2006-01-02T15:04:05Z07:00"

func createHash(s string) string {
	hash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hash[:])
}

func parseHTMLContent(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return htmlContent // Return the original content if parsing fails
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

func getBaseDomain(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		log.Printf("Failed to parse URL %s: %v", rawURL, err)
		return ""
	}
	return parsedURL.Scheme + "://" + parsedURL.Host
}

func parseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	req, err := decodeRequest(r)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	responses := processURLs(req.URLs)
	sendResponse(w, responses)
}

func decodeRequest(r *http.Request) (ParseRequest, error) {
	var req ParseRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

func processURLs(urls []string) []FeedResponse {
	var wg sync.WaitGroup
	sem := make(chan struct{}, numWorkers)
	responses := make(chan FeedResponse, len(urls))

	for _, url := range urls {
		wg.Add(1)
		sem <- struct{}{}
		go func(url string) {
			defer func() {
				wg.Done()
				<-sem
			}()
			response := processURL(url)
			responses <- response
		}(url)
	}

	go func() {
		wg.Wait()
		close(responses)
	}()

	return collectResponses(responses)
}

func isCacheStale(lastRefreshed string) bool {
	parsedTime, err := time.Parse(layout, lastRefreshed)
	if err != nil {
		log.Printf("Failed to parse LastRefreshed: %v", err)
		return false
	}

	if time.Since(parsedTime) > time.Duration(refresh_timer)*time.Minute {
		log.Println("[Cache Stale] Cache is stale")
		return true
	}
	return false
}

// func fetchAndCacheFeed(feedURL string, cacheKey string) (FeedResponse, error) {
// 	parser := gofeed.NewParser()
// 	feed, err := parser.ParseURL(feedURL)
// 	if err != nil {
// 		log.WithFields(logrus.Fields{
// 			"url":   feedURL,
// 			"error": err,
// 		}).Error("Failed to parse URL")
// 		return FeedResponse{}, err
// 	}

// 	feedItems := processFeedItems(feed)
// 	baseDomain := getBaseDomain(feed.Link)
// 	addURLToList(feedURL)

// 	baseDomainCacheKey := createHash(baseDomain)
// 	var metaData link2json.MetaDataResponseItem

// 	cacheMutex.Lock()
// 	defer cacheMutex.Unlock()

// 	if err := cache.Get(metaData_prefix, baseDomainCacheKey, &metaData); err != nil {
// 		if isValidURL(baseDomain) {
// 			tempMetaData, err := GetMetaData(baseDomain)
// 			if err != nil {
// 				log.Printf("Failed to get metadata for %s: %v", baseDomain, err)
// 			} else {
// 				metaData = tempMetaData
// 				if err := cache.Set(metaData_prefix, baseDomainCacheKey, metaData, 24*time.Hour); err != nil {
// 					log.Printf("Failed to cache metadata for %s: %v", baseDomain, err)
// 				}
// 			}
// 		} else {
// 			metaData = link2json.MetaDataResponseItem{}
// 			log.Printf("[fetchAndCacheFeed] Invalid URL %s", baseDomain)
// 		}
// 	} else {
// 		log.Printf("Loaded metadata from cache for %s", baseDomain)
// 	}

// 	response := createFeedResponse(feed, feedURL, metaData, feedItems)

// 	if err := cache.Set(feed_prefix, cacheKey, response, 24*time.Hour); err != nil {
// 		log.WithFields(logrus.Fields{
// 			"url":   feedURL,
// 			"error": err,
// 		}).Error("Failed to cache feed details")
// 		return FeedResponse{}, err
// 	}

// 	log.Infof("Successfully cached feed details for %s", feedURL)
// 	return response, nil
// }

// fetchAndCacheFeed fetches a remote feed, merges with old items, and caches the final FeedResponse.
func fetchAndCacheFeed(feedURL, cacheKey string) (FeedResponse, error) {
	parser := gofeed.NewParser()
	feed, err := parser.ParseURL(feedURL)
	if err != nil {
		log.WithFields(logrus.Fields{"url": feedURL, "error": err}).
			Error("Failed to parse URL")
		return FeedResponse{}, err
	}

	// 1) Convert the newly fetched feed into a slice of items
	newItems := processFeedItems(feed)

	// 2) Merge them with existing items (deduplicate, keep last 24h, update changed)
	mergedItems, mergeErr := mergeFeedItemsAtParserLevel(feedURL, newItems)
	if mergeErr != nil {
		return FeedResponse{}, mergeErr
	}

	// 3) Build feed metadata
	baseDomain := getBaseDomain(feed.Link)
	addURLToList(feedURL) // track feed subscription or known feed

	// Possibly fetch additional metadata from the cache
	var metaData link2json.MetaDataResponseItem

	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	baseDomainKey := createHash(baseDomain)

	if err := cache.Get(metaData_prefix, baseDomainKey, &metaData); err != nil {
		if isValidURL(baseDomain) {
			tempMeta, errGet := GetMetaData(baseDomain)
			if errGet != nil {
				log.Printf("Failed to get metadata for %s: %v", baseDomain, errGet)
			} else {
				metaData = tempMeta
				if errSet := cache.Set(metaData_prefix, baseDomainKey, metaData, 24*time.Hour); errSet != nil {
					log.Printf("Failed to cache metadata for %s: %v", baseDomain, errSet)
				}
			}
		} else {
			metaData = link2json.MetaDataResponseItem{}
			log.Printf("[fetchAndCacheFeed] Invalid URL %s", baseDomain)
		}
	} else {
		log.Printf("Loaded metadata from cache for %s", baseDomain)
	}

	// 4) Create final feed response with merged items
	finalFeedResponse := createFeedResponse(feed, feedURL, metaData, mergedItems)

	// 5) Cache the final feed response
	if err := cache.Set(feed_prefix, cacheKey, finalFeedResponse, 24*time.Hour); err != nil {
		log.WithFields(logrus.Fields{"url": feedURL, "error": err}).
			Error("Failed to cache feed details")
		return FeedResponse{}, err
	}

	log.Infof("Successfully cached feed details for %s", feedURL)
	return finalFeedResponse, nil
}

func processURL(rawURL string) FeedResponse {
	feedURL := sanitizeURL(rawURL)
	cacheKey := feedURL

	var cachedFeed FeedResponse
	err := cache.Get(feed_prefix, cacheKey, &cachedFeed)
	if err == nil && cachedFeed.SiteTitle != "" {
		// Cache hit
		log.WithFields(logrus.Fields{
			"url": feedURL,
		}).Info("[Cache Hit] Using cached feed details")

		// If feed is stale, refresh in background
		if isCacheStale(cachedFeed.LastRefreshed) {
			log.WithFields(logrus.Fields{"url": feedURL}).
				Info("[Cache Stale] Cache is stale, refreshing in background")
			go func() {
				_, errRefresh := fetchAndCacheFeed(feedURL, cacheKey)
				if errRefresh != nil {
					log.WithFields(logrus.Fields{"url": feedURL, "error": errRefresh}).
						Error("Failed to refresh feed")
				}
			}()
		}

		// Optionally re-check thumbnail colors
		updatedItems := updateFeedItemsWithThumbnailColors(cachedFeed.Items)
		cachedFeed.Items = &updatedItems
		return cachedFeed
	}

	// Cache miss or empty
	log.WithFields(logrus.Fields{"url": feedURL}).
		Info("[Cache Miss] Cache miss")

	newResp, errNew := fetchAndCacheFeed(feedURL, cacheKey)
	if errNew != nil {
		log.WithFields(logrus.Fields{"url": feedURL, "error": errNew}).
			Error("Failed to fetch and cache feed")

		return FeedResponse{
			Type:    "unknown",
			FeedUrl: feedURL,
			GUID:    cacheKey,
			Status:  "error",
			Error:   errNew,
		}
	}
	return newResp
}
func updateFeedItemsWithThumbnailColors(items *[]FeedResponseItem) []FeedResponseItem {
	var updatedItems []FeedResponseItem
	for _, item := range *items {
		// Update the thumbnail color for each item
		updatedItem := updateThumbnailColorForItem(item)
		updatedItems = append(updatedItems, updatedItem)
	}
	return updatedItems
}
func updateThumbnailColorForItem(item FeedResponseItem) FeedResponseItem {
	var cachedColor RGBColor
	err := cache.Get(thumbnailColorPrefix, item.Thumbnail, &cachedColor)

	switch {
	case err != nil:
		log.Printf("Thumbnail color not yet available for %s", item.Thumbnail)
	case item.ThumbnailColorComputed == "set":
		// log.Printf("Thumbnail color already set for %s: %v", item.Thumbnail, item.ThumbnailColor)
	case item.ThumbnailColorComputed == "computed":
		item.ThumbnailColor = cachedColor
		item.ThumbnailColorComputed = "set"
		log.Printf("Updated thumbnail color for %s: %v", item.Thumbnail, item.ThumbnailColor)
	default:
		// log.Printf("Thumbnail color already set for %s: %v", item.Thumbnail, item.ThumbnailColor)
	}

	return item
}
func processFeedItems(feed *gofeed.Feed) []FeedResponseItem {
	// 1) Check if feed or feed.Items is nil/empty
	if feed == nil {
		log.Error("[processFeedItems] feed is nil; returning empty slice")
		return nil
	}
	if feed.Items == nil || len(feed.Items) == 0 {
		log.Warnf("[processFeedItems] feed.Items is nil or empty for feed: %q", feed.Title)
		return nil
	}

	// 2) Initialize thumbnail and default color if iTunesExt image is present
	thumbnail := ""
	defaultThumbnailColor := RGBColor{128, 128, 128}
	if feed.ITunesExt != nil && feed.ITunesExt.Image != "" {
		thumbnail = feed.ITunesExt.Image
		r, g, b := extractColorFromThumbnail_prominentColor(thumbnail)
		defaultThumbnailColor = RGBColor{r, g, b}
	}

	// 3) Prepare concurrency
	var wg sync.WaitGroup
	sem := make(chan struct{}, numWorkers)
	itemResponses := make(chan FeedResponseItem, len(feed.Items))

	// 4) Iterate over feed.Items
	for _, item := range feed.Items {
		// Skip nil items gracefully
		if item == nil {
			log.Warn("[processFeedItems] Encountered a nil item; skipping")
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

	// 5) Close channel after all goroutines finish
	go func() {
		wg.Wait()
		close(itemResponses)
	}()

	// 6) Collect the responses and return
	return collectItemResponses(itemResponses)
}

func processFeedItem(item *gofeed.Item, thumbnail string, thumbnailColor RGBColor) FeedResponseItem {
	Link := item.Link
	Duration := 0

	author := getItemAuthor(item)
	categories := strings.Join(item.Categories, ", ")

	if len(item.Enclosures) > 0 {
		for _, enclosure := range item.Enclosures {
			if enclosure.URL != "" && strings.HasPrefix(enclosure.Type, "image/") {
				thumbnail = enclosure.URL
				break
			}
		}
	}

	if thumbnail == "" && item.Image != nil {
		thumbnail = item.Image.URL
	}

	if item.ITunesExt != nil && item.ITunesExt.Image != "" {
		thumbnail = item.ITunesExt.Image
	}

	thumbnailFinder := NewThumbnailFinder()

	if thumbnail == "" {
		thumbnail = thumbnailFinder.FindThumbnailForItem(item)
	}

	// Initialize thumbnail color to default
	thumbnailColorComputed := "no"

	// Check if the thumbnail color is already cached
	cachePrefix := thumbnailColor_prefix
	var cachedColor RGBColor
	cacheMutex.Lock()
	err := cache.Get(cachePrefix, thumbnail, &cachedColor)
	cacheMutex.Unlock()
	if err == nil {
		// Use the cached color
		thumbnailColor = cachedColor
		thumbnailColorComputed = "set"
	} else {
		go func(thumbnailURL string) {
			if thumbnailURL != "" {
				r, g, b := extractColorFromThumbnail_prominentColor(thumbnailURL)
				actualColor := RGBColor{r, g, b}
				// log.Printf("Asynchronously extracted color for %s: %v", thumbnailURL, actualColor)
				if err := cache.Set(thumbnailColorPrefix, thumbnailURL, actualColor, 24*time.Hour); err != nil {
					log.Printf("Failed to cache color for %s: %v", thumbnailURL, err)
				} else {
					// log.Printf("Successfully cached color for %s", thumbnailURL)
				}
			}
		}(thumbnail)
	}

	description := item.Description
	if description == "" {
		description = parseHTMLContent(item.Content)
	}
	description = parseHTMLContent(description)

	standardizedPublished := standardizeDate(item.Published)
	itemType, duration := determineItemTypeAndDuration(item)
	Duration = duration

	responseItem := FeedResponseItem{
		Type:                   itemType,
		ID:                     createHash(item.Link),
		Title:                  item.Title,
		Description:            description,
		Link:                   Link,
		Duration:               Duration,
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

	return responseItem
}

func standardizeDate(dateStr string) string {
	if dateStr == "" {
		log.Info("[Standardize Date] Received empty date string")
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

	log.Infof("[Standardize Date] Failed to parse date: %v", dateStr)
	return ""
}

func createFeedResponse(feed *gofeed.Feed, feedURL string, metaData link2json.MetaDataResponseItem, feedItems []FeedResponseItem) FeedResponse {
	var feedType string
	var thumbnail string

	if feed == nil {
		log.Println("createFeedResponse: feed is nil for", feedURL)
		return FeedResponse{}
	}

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

func collectResponses(responses chan FeedResponse) []FeedResponse {
	var allResponses []FeedResponse
	for response := range responses {
		allResponses = append(allResponses, response)
	}
	return allResponses
}

func collectItemResponses(itemResponses chan FeedResponseItem) []FeedResponseItem {
	var feedItems []FeedResponseItem
	for itemResponse := range itemResponses {
		feedItems = append(feedItems, itemResponse)
	}

	sort.Slice(feedItems, func(i, j int) bool {
		timeI, errI := time.Parse(layout, feedItems[i].Published)
		if errI != nil {
			log.Printf("[Sort] Failed to parse time for item I: %v", errI)
			return false
		}
		timeJ, errJ := time.Parse(layout, feedItems[j].Published)
		if errJ != nil {
			log.Printf("[Sort] Failed to parse time for item J: %v", errJ)
			return true
		}
		return timeI.After(timeJ)
	})

	return feedItems
}

func sendResponse(w http.ResponseWriter, responses []FeedResponse) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)

	feeds := Feeds{Feeds: responses}

	if err := enc.Encode(feeds); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func refreshFeeds() {
	urls := getAllCachedURLs()

	for _, url := range urls {
		log.Printf("Refreshing feed for URL: %s", url)
		_ = processURL(url)
	}
}

func addURLToList(url string) {
	urlListMutex.Lock()
	defer urlListMutex.Unlock()

	if !stringInSlice(url, urlList) {
		urlList = append(urlList, url)
	}
}

func getAllCachedURLs() []string {
	urlListMutex.Lock()
	defer urlListMutex.Unlock()

	if len(urlList) == 0 {
		startTime := time.Now()

		var err error
		urlList, err = cache.GetSubscribedListsFromCache(feed_prefix)
		if err != nil {
			log.Println("Failed to get subscribed lists from cache:", err)
			return nil
		}

		duration := time.Since(startTime)
		log.Infof("Loaded urlList from cache: %v", urlList)
		log.Infof("Time taken to load urlList: %v", duration)
	}

	return append([]string(nil), urlList...)
}

func isValidURL(str string) bool {
	parsedURL, err := url.ParseRequestURI(str)
	if err != nil {
		logrus.Info(err.Error())
		return false
	}

	host := parsedURL.Hostname()
	if net.ParseIP(host) != nil {
		return true
	}

	return strings.Contains(host, ".")
}

func sanitizeURL(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.Scheme == "" {
		rawURL = "https://" + rawURL
	} else if parsedURL.Scheme == "http" {
		rawURL = strings.Replace(rawURL, "http://", "https://", 1)
	}
	return rawURL
}

func getItemAuthor(item *gofeed.Item) string {
	if item.ITunesExt != nil && item.ITunesExt.Author != "" {
		return item.ITunesExt.Author
	}
	if item.Author != nil && item.Author.Name != "" {
		return item.Author.Name
	}
	return ""
}

func determineItemTypeAndDuration(item *gofeed.Item) (string, int) {
	if item.ITunesExt != nil {
		itemType := "podcast"
		duration := parseDuration(item.ITunesExt.Duration)
		return itemType, duration
	}
	return "article", 0
}

func parseDuration(durationStr string) int {
	if durationStr == "" {
		return 0
	}

	if durationInt, err := strconv.Atoi(durationStr); err == nil {
		return durationInt
	}

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

func stringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}

// Cache merging logic
// mergeFeedItemsAtParserLevel merges old items from the cache with newly fetched items.
// We only keep items from the last 24 hours, deduplicate by GUID, and update changed content.
func mergeFeedItemsAtParserLevel(feedURL string, newItems []FeedResponseItem) ([]FeedResponseItem, error) {
	cacheKey := feedURL // or any unique suffix for the feed items
	var existingItems []FeedResponseItem
	var existingFeedResponse FeedResponse

	// 1) Fetch existing items from cache if available
	if err := cache.Get(feed_prefix, cacheKey, &existingFeedResponse); err != nil {

		log.Errorf("Error getting existing items for feed %s: %v", feedURL, err)
		// treat as empty if it's a non-fatal error

		existingItems = nil
	} else {
		existingItems = *existingFeedResponse.Items
		log.Print("Existing items fetched")
	}

	// 2) Convert existing items into a map (GUID -> FeedResponseItem) for easy dedup
	itemMap := make(map[string]FeedResponseItem)
	// only keep items from last 24h
	log.Printf("Merging %d existing items with %d new items", len(existingItems), len(newItems))
	for _, oldItem := range existingItems {
		if isWithinPeriod(oldItem, cachePeriod) {
			itemMap[oldItem.ID] = oldItem
		}
	}

	log.Print("Existing items map created")

	// 3) Merge new items
	for _, newIt := range newItems {
		if oldIt, found := itemMap[newIt.ID]; found {
			// item with same GUID found, check if updated
			if isUpdatedContent(oldIt, newIt) {
				itemMap[newIt.ID] = newIt
			}
		} else {
			// new item -> keep only if within time period
			if isWithinPeriod(newIt, cachePeriod) {
				itemMap[newIt.ID] = newIt
			}
		}
	}

	// 4) Convert map back to slice
	merged := make([]FeedResponseItem, 0, len(itemMap))
	for _, v := range itemMap {
		merged = append(merged, v)
	}

	// 5) Store merged items back to the cache
	if err := cache.Set(feed_prefix, feedURL, merged, 24*time.Hour); err != nil {
		return nil, err
	}
	return merged, nil
}

// isWithinPeriod checks if the item was published within the configurable period.
func isWithinPeriod(item FeedResponseItem, days int) bool {
	t, err := time.Parse(layout, item.Published)
	if err != nil {
		return false
	}
	return time.Since(t) <= time.Duration(days)*24*time.Hour
}

// isUpdatedContent decides if new item is "more recent" or has changed content.
func isUpdatedContent(oldIt, newIt FeedResponseItem) bool {
	oldTime, _ := time.Parse(layout, oldIt.Published)
	newTime, _ := time.Parse(layout, newIt.Published)

	// If new item is more recent
	if newTime.After(oldTime) {
		return true
	}
	// or if content changed
	if newIt.Content != oldIt.Content {
		return true
	}
	return false
}
