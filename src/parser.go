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

// The date/time layout format used throughout the code.
const layout = "2006-01-02T15:04:05Z07:00"

// createHash returns a SHA-256 hash of the given string s.
func createHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// parseHTMLContent attempts to parse htmlContent as HTML,
// extracting and returning only the text content. If parsing fails,
// the original htmlContent is returned unchanged.
func parseHTMLContent(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		// Fallback to the raw HTML if parse fails
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

// getBaseDomain attempts to parse rawURL and returns its scheme + hostname (e.g., https://example.com).
// If parsing fails, an empty string is returned.
func getBaseDomain(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		log.Printf("[getBaseDomain] Failed to parse URL %s: %v", rawURL, err)
		return ""
	}
	return parsedURL.Scheme + "://" + parsedURL.Host
}

// decodeRequest reads and unmarshals the request body into a ParseRequest object.
func decodeRequest(r *http.Request) (ParseRequest, error) {
	var req ParseRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

// processURLs concurrently processes each feed URL using processURL,
// returns all feed responses (with pagination applied) as a slice.
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

// isCacheStale checks whether lastRefreshed is older than refresh_timer (in minutes).
func isCacheStale(lastRefreshed string) bool {
	parsedTime, err := time.Parse(layout, lastRefreshed)
	if err != nil {
		log.Printf("[isCacheStale] Failed to parse LastRefreshed: %v", err)
		return false
	}

	if time.Since(parsedTime) > time.Duration(refresh_timer)*time.Minute {
		log.Printf("[isCacheStale] Cache is stale (older than %d minutes)", refresh_timer)
		return true
	}
	return false
}

// fetchAndCacheFeed fetches the remote feed from feedURL, merges with existing items (if any),
// and caches the final FeedResponse. Returns the FeedResponse or an error if any step fails.
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
		return FeedResponse{}, mergeErr
	}

	baseDomain := getBaseDomain(feed.Link)
	addURLToList(feedURL)

	// Possibly fetch additional metadata from cache
	var metaData link2json.MetaDataResponseItem

	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	baseDomainKey := createHash(baseDomain)
	if err := cache.Get(metaData_prefix, baseDomainKey, &metaData); err != nil {
		if isValidURL(baseDomain) {
			tempMeta, errGet := GetMetaData(baseDomain)
			if errGet != nil {
				log.Printf("[fetchAndCacheFeed] Failed to get metadata for %s: %v", baseDomain, errGet)
			} else {
				metaData = tempMeta
				if errSet := cache.Set(metaData_prefix, baseDomainKey, metaData, 24*time.Hour); errSet != nil {
					log.Printf("[fetchAndCacheFeed] Failed to cache metadata for %s: %v", baseDomain, errSet)
				}
			}
		} else {
			metaData = link2json.MetaDataResponseItem{}
			log.Printf("[fetchAndCacheFeed] Invalid baseDomain %s", baseDomain)
		}
	} else {
		log.Printf("[fetchAndCacheFeed] Loaded metadata from cache for %s", baseDomain)
	}

	// Build final FeedResponse from the merged items
	finalFeedResponse := createFeedResponse(feed, feedURL, metaData, mergedItems)

	// Cache the final feed response
	if err := cache.Set(feed_prefix, cacheKey, finalFeedResponse, 24*time.Hour); err != nil {
		log.WithFields(logrus.Fields{"url": feedURL, "error": err}).
			Error("[fetchAndCacheFeed] Failed to cache feed details")
		return FeedResponse{}, err
	}

	log.Infof("[fetchAndCacheFeed] Successfully cached feed details for %s", feedURL)
	return finalFeedResponse, nil
}

// processURL checks the cache for a feed URL; if found and not stale, returns the cached feed.
// Otherwise, calls fetchAndCacheFeed to retrieve and cache a fresh feed. Pagination is then applied
// to the final list of items before returning.
func processURL(rawURL string, page, itemsPerPage int) FeedResponse {
	feedURL := sanitizeURL(rawURL)
	cacheKey := feedURL

	var cachedFeed FeedResponse
	// Try retrieving from cache first
	if err := cache.Get(feed_prefix, cacheKey, &cachedFeed); err == nil && cachedFeed.SiteTitle != "" {
		// Cache hit
		log.WithFields(logrus.Fields{"url": feedURL}).
			Info("[processURL] [Cache Hit] Using cached feed details")

		// Check staleness
		if isCacheStale(cachedFeed.LastRefreshed) {
			log.WithFields(logrus.Fields{"url": feedURL}).
				Info("[processURL] Cache is stale, refreshing in background")
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
	log.WithFields(logrus.Fields{"url": feedURL}).
		Info("[processURL] [Cache Miss] Fetching fresh feed")

	newResp, errNew := fetchAndCacheFeed(feedURL, cacheKey)
	if errNew != nil {
		log.WithFields(logrus.Fields{"url": feedURL, "error": errNew}).
			Error("[processURL] Failed to fetch and cache feed")

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

// applyPagination modifies the feed items in place, slicing to the requested page and itemsPerPage
// (e.g. page=2, itemsPerPage=10 => skip first 10 items, return next 10).
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

// updateFeedItemsWithThumbnailColors iterates over existing items in a feed,
// calling updateThumbnailColorForItem to finalize or skip color checks.
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

// updateThumbnailColorForItem checks if we have a cached color for the item’s thumbnail.
// If so, sets item.ThumbnailColor. Otherwise, logs that color is not yet available.
func updateThumbnailColorForItem(item FeedResponseItem) FeedResponseItem {
	var cachedColor RGBColor
	err := cache.Get(thumbnailColorPrefix, item.Thumbnail, &cachedColor)

	switch {
	case err != nil:
		log.Printf("[updateThumbnailColorForItem] No cached color for %s yet", item.Thumbnail)
	case item.ThumbnailColorComputed == "set":
		// Already set
	case item.ThumbnailColorComputed == "computed":
		item.ThumbnailColor = cachedColor
		item.ThumbnailColorComputed = "set"
		log.Printf("[updateThumbnailColorForItem] Updated color for %s: %v", item.Thumbnail, item.ThumbnailColor)
	default:
		// No additional logic
	}
	return item
}

// processFeedItems validates feed, concurrency processes each item, returning a slice of FeedResponseItem.
func processFeedItems(feed *gofeed.Feed) []FeedResponseItem {
	// Safeguard feed == nil or feed.Items is nil/empty
	if feed == nil {
		log.Error("[processFeedItems] feed is nil; returning empty slice")
		return nil
	}
	if feed.Items == nil || len(feed.Items) == 0 {
		log.Warnf("[processFeedItems] feed.Items is nil or empty for feed: %q", feed.Title)
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

// processFeedItem creates a FeedResponseItem from a single gofeed.Item,
// attempting to discover a thumbnail if not set, and sets a default or cached color.
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
				log.Printf("[processFeedItem] Failed to cache color for %s: %v", thURL, cErr)
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

// standardizeDate parses dateStr in various known formats (RFC3339, RFC1123, etc.).
// Returns the date in a standard layout or empty if parse fails.
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
	log.Infof("[standardizeDate] Failed to parse date: %v", dateStr)
	return ""
}

// createFeedResponse builds a FeedResponse struct from a parsed feed object, feed metadata, and items.
func createFeedResponse(feed *gofeed.Feed, feedURL string, metaData link2json.MetaDataResponseItem, feedItems []FeedResponseItem) FeedResponse {
	if feed == nil {
		log.Errorf("[createFeedResponse] feed is nil for %s", feedURL)
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

// collectResponses reads FeedResponse objects from a channel, returning them as a slice.
func collectResponses(responses chan FeedResponse) []FeedResponse {
	var all []FeedResponse
	for resp := range responses {
		all = append(all, resp)
	}
	return all
}

// collectItemResponses reads FeedResponseItem objects from a channel, returning them as a slice.
// The items are then sorted by descending Published date.
func collectItemResponses(itemResponses chan FeedResponseItem) []FeedResponseItem {
	var feedItems []FeedResponseItem
	for itemResponse := range itemResponses {
		feedItems = append(feedItems, itemResponse)
	}
	// Sort by published date (descending)
	sort.Slice(feedItems, func(i, j int) bool {
		timeI, errI := time.Parse(layout, feedItems[i].Published)
		if errI != nil {
			log.Printf("[collectItemResponses] Failed to parse time for item I: %v", errI)
			return false
		}
		timeJ, errJ := time.Parse(layout, feedItems[j].Published)
		if errJ != nil {
			log.Printf("[collectItemResponses] Failed to parse time for item J: %v", errJ)
			return true
		}
		return timeI.After(timeJ)
	})
	return feedItems
}

// sendResponse writes a JSON-encoded Feeds struct to the ResponseWriter.
func sendResponse(w http.ResponseWriter, responses []FeedResponse) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)

	feeds := Feeds{Feeds: responses}
	if err := enc.Encode(feeds); err != nil {
		log.Errorf("[sendResponse] Failed to encode feed responses: %v", err)
	}
}

// refreshFeeds retrieves all known feed URLs from the cache and re-processes them
// (useful for cron-based or ticker-based refreshing).
func refreshFeeds() {
	urls := getAllCachedURLs()
	for _, url := range urls {
		log.Printf("[refreshFeeds] Refreshing feed for URL: %s", url)
		_ = processURL(url, 1, 20) // or any default paging
	}
}

// addURLToList ensures the feed URL is tracked in urlList (the list of subscribed or known feeds).
func addURLToList(url string) {
	urlListMutex.Lock()
	defer urlListMutex.Unlock()

	if !stringInSlice(url, urlList) {
		urlList = append(urlList, url)
	}
}

// getAllCachedURLs returns all known feed URLs from the cache or from an in-memory list.
// If none are found, returns an empty slice.
func getAllCachedURLs() []string {
	urlListMutex.Lock()
	defer urlListMutex.Unlock()

	if len(urlList) == 0 {
		startTime := time.Now()

		var err error
		urlList, err = cache.GetSubscribedListsFromCache(feed_prefix)
		if err != nil {
			log.Warnf("[getAllCachedURLs] Failed to get subscribed feeds from cache: %v", err)
			return nil
		}

		duration := time.Since(startTime)
		log.Infof("[getAllCachedURLs] Loaded urlList from cache: %v, took %v", urlList, duration)
	}

	// return a copy
	return append([]string(nil), urlList...)
}

// isValidURL checks if str is a syntactically valid URL with a resolvable host or IP.
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

// sanitizeURL ensures URLs always use https:// if no scheme or if http://
func sanitizeURL(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.Scheme == "" {
		return "https://" + rawURL
	} else if parsedURL.Scheme == "http" {
		return strings.Replace(rawURL, "http://", "https://", 1)
	}
	return rawURL
}

// getItemAuthor returns an item’s author if set, checking iTunesExt first (for podcasts).
func getItemAuthor(item *gofeed.Item) string {
	if item.ITunesExt != nil && item.ITunesExt.Author != "" {
		return item.ITunesExt.Author
	}
	if item.Author != nil && item.Author.Name != "" {
		return item.Author.Name
	}
	return ""
}

// determineItemTypeAndDuration checks if the feed item is a podcast. If yes,
// parses the item’s duration. Returns a type (podcast or article) and the duration in seconds.
func determineItemTypeAndDuration(item *gofeed.Item) (string, int) {
	if item.ITunesExt != nil {
		return "podcast", parseDuration(item.ITunesExt.Duration)
	}
	return "article", 0
}

// parseDuration attempts to convert a time-like string (e.g., 3600 or HH:MM:SS) into total seconds.
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

// stringInSlice returns true if str is found in the slice list.
func stringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}

// mergeFeedItemsAtParserLevel merges old items from the cache with newly fetched items, deduplicating by ID
// and retaining only items from within the last 24 hours (cachePeriod). Also updates items if content changed.
func mergeFeedItemsAtParserLevel(feedURL string, newItems []FeedResponseItem) ([]FeedResponseItem, error) {
	cacheKey := feedURL
	var existingFeedResponse FeedResponse
	var existingItems []FeedResponseItem

	// Attempt to get an existing feed from the cache
	if err := cache.Get(feed_prefix, cacheKey, &existingFeedResponse); err != nil {
		log.Errorf("[mergeFeedItemsAtParserLevel] Error getting existing items for feed %s: %v", feedURL, err)
		existingItems = nil
	} else {
		if existingFeedResponse.Items != nil {
			existingItems = *existingFeedResponse.Items
			log.Printf("[mergeFeedItemsAtParserLevel] Found %d existing items for feed %s", len(existingItems), feedURL)
		}
	}

	itemMap := make(map[string]FeedResponseItem)
	log.Printf("[mergeFeedItemsAtParserLevel] Merging %d existing items with %d new items", len(existingItems), len(newItems))
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
		return nil, err
	}

	return merged, nil
}

// isWithinPeriod returns true if the FeedResponseItem’s Published date is within 'days' days of now.
func isWithinPeriod(item FeedResponseItem, days int) bool {
	t, err := time.Parse(layout, item.Published)
	if err != nil {
		return false
	}
	return time.Since(t) <= time.Duration(days)*24*time.Hour
}

// isUpdatedContent returns true if newIt is more recent than oldIt by published date
// or if newIt’s content differs from oldIt’s content.
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
