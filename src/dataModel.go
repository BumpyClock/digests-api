package main

import (
	"github.com/mmcdole/gofeed"
)

// ParseRequest represents the expected incoming JSON payload structure.
// ParseRequest represents the expected JSON body for the /parse endpoint.
type ParseRequest struct {
	URLs         []string `json:"urls"`
	Page         int      `json:"page"`
	ItemsPerPage int      `json:"itemsperpage"`
}

type RGBColor struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
}

// Add these structs to your existing code
type MediaContent struct {
	URL    string `xml:"url,attr"`
	Width  int    `xml:"width,attr"`
	Height int    `xml:"height,attr"`
}

type ExtendedItem struct {
	*gofeed.Item
	MediaContent []MediaContent `xml:"http://search.yahoo.com/mrss/ content"`
}

// FeedResponseItem represents an enriched structure for an individual feed item.
type FeedResponseItem struct {
	Type                   string                      `json:"type"`
	ID                     string                      `json:"id"`
	Title                  string                      `json:"title"`
	Description            string                      `json:"description"`
	Link                   string                      `json:"link"`
	Author                 string                      `json:"author"`
	Published              string                      `json:"published"`
	Content                string                      `json:"content"`
	Created                string                      `json:"created"`
	Content_Encoded        string                      `json:"content_encoded"`
	Categories             string                      `json:"categories"`
	Enclosures             []*gofeed.Enclosure         `json:"enclosures"`
	Thumbnail              string                      `json:"thumbnail"`
	ThumbnailColor         RGBColor                    `json:"thumbnailColor"`
	ThumbnailColorComputed string                      `json:"thumbnailColorComputed"`
	EpisodeType            string                      `json:"episodeType,omitempty"`
	Subtitle               []PodcastTranscriptsDetails `json:"subtitle,omitempty"`
	Duration               int                         `json:"duration,omitempty"`
}

// FeedResponse represents the structure for the overall feed, including metadata and items.
type FeedResponse struct {
	Type          string              `json:"type"`
	GUID          string              `json:"guid"`
	Status        string              `json:"status"`
	Error         error               `json:"error,omitempty"`
	SiteTitle     string              `json:"siteTitle"`
	FeedTitle     string              `json:"feedTitle"`
	FeedUrl       string              `json:"feedUrl"`
	Description   string              `json:"description"`
	Link          string              `json:"link"`
	LastUpdated   string              `json:"lastUpdated"`
	LastRefreshed string              `json:"lastRefreshed"`
	Published     string              `json:"published"`
	Author        *gofeed.Person      `json:"author"`
	Language      string              `json:"language"`
	Favicon       string              `json:"favicon"`
	SiteImage     string              `json:"siteImage,omitempty"`
	Categories    string              `json:"categories"`
	Items         *[]FeedResponseItem `json:"items,omitempty"`
}

type Feeds struct {
	Feeds []FeedResponse `json:"feeds"`
}

type ReaderViewResult struct {
	URL         string `json:"url"`
	Status      string `json:"status"`
	ReaderView  string `json:"content"`
	Error       error  `json:"error,omitempty"`
	Title       string `json:"title"`
	SiteName    string `json:"siteName"`
	Image       string `json:"image"`
	Favicon     string `json:"favicon"`
	TextContent string `json:"textContent"`
}

type FeedSearchAPIResponseItem struct {
	Bozo           int      `json:"bozo"`
	Content_length int      `json:"content_length"`
	Content_type   string   `json:"content_type"`
	Description    string   `json:"description"`
	Favicon        string   `json:"favicon"`
	Hubs           []string `json:"hubs"`
	Is_Podcast     bool     `json:"is_podcast"`
	Is_Push        bool     `json:"is_push"`
	Item_Count     int      `json:"item_count"`
	Last_Seen      string   `json:"last_seen"`
	Last_Updated   string   `json:"last_updated"`
	Score          float64  `json:"score"`
	Site_name      string   `json:"site_name"`
	Site_url       string   `json:"site_url"`
	Title          string   `json:"title"`
	Url            string   `json:"url"`
	Velocity       float64  `json:"velocity"`
	Version        string   `json:"version"`
	Self_Url       string   `json:"self_url"`
}

type FeedSearchResponseItem struct {
	Title        string  `json:"title"`
	Url          string  `json:"url"`
	Site_name    string  `json:"site_name"`
	Site_url     string  `json:"site_url"`
	Description  string  `json:"description"`
	Favicon      string  `json:"favicon"`
	Is_Podcast   bool    `json:"is_podcast"`
	Is_Push      bool    `json:"is_push"`
	Item_Count   int     `json:"item_count"`
	Last_Seen    string  `json:"last_seen"`
	Last_Updated string  `json:"last_updated"`
	Score        float64 `json:"score"`
}

type PodcastSearchResponseItem struct {
	Title             string   `json:"title"`
	Url               string   `json:"url"`
	Author            string   `json:"author"`
	Description       string   `json:"description"`
	FeedImage         string   `json:"feedImage"`
	Image             string   `json:"image"`
	Artwork           string   `json:"artwork"`
	Categories        []string `json:"categories"`
	PodcastGUID       string   `json:"podcastGuid"`
	EpisodeCount      int      `json:"episodeCount"`
	NewestItemPubdate float32  `json:"newestItemPubdate"`
}

type PodcastAPIResponseItem struct {
	Id                     int                         `json:"id"`
	Title                  string                      `json:"title"`
	Url                    string                      `json:"url"`
	OriginalUrl            string                      `json:"originalUrl"`
	Link                   string                      `json:"link"`
	Description            string                      `json:"description"`
	Author                 string                      `json:"author"`
	Language               string                      `json:"language"`
	OwnerName              string                      `json:"ownerName"`
	Image                  string                      `json:"image"`
	Artwork                string                      `json:"artwork"`
	FeedImage              string                      `json:"feedImage"`
	FeedID                 string                      `json:"feedId"`
	PodcastGUID            string                      `json:"podcastGuid"`
	LastUpdatedTime        string                      `json:"lastUpdatedTime"`
	LastCrawlTime          int                         `json:"lastCrawlTime"`
	LastParseTime          int                         `json:"lastParseTime"`
	InPollingQueue         int                         `json:"inPollingQueue"`
	Priority               int                         `json:"priority"`
	LastGoodHttpStatusTime int                         `json:"lastGoodHttpStatusTime"`
	LastHttpStatus         int                         `json:"lastHttpStatus"`
	ContentType            string                      `json:"contentType"`
	ItunedId               int                         `json:"itunedId"`
	Generator              string                      `json:"generator"`
	Dead                   int                         `json:"dead"`
	CrawlErrors            int                         `json:"crawlErrors"`
	ParseErrors            int                         `json:"parseErrors"`
	Categories             []string                    `json:"podCast_Categories"`
	Locked                 int                         `json:"locked"`
	Medium                 string                      `json:"medium"`
	EpisodeCount           int                         `json:"episodeCount"`
	ImageUrlHash           float64                     `json:"imageUrlHash"`
	NewestItemPubdate      float32                     `json:"newestItemPubdate"`
	Transcripts            []PodcastTranscriptsDetails `json:"transcripts"`
}

type PodcastTranscriptsDetails struct {
	Url  string                 `json:"url"`
	Type PodcastTranscriptsType `json:"type"`
}

type PodcastTranscriptsType string

// Define constants of the custom type
const (
	TextHTML       PodcastTranscriptsType = "text/html"
	ApplicationSRT PodcastTranscriptsType = "application/srt"
	ApplicationVTT PodcastTranscriptsType = "application/vtt"
)

type PodcastSearchAPIResponse struct {
	Items       []PodcastAPIResponseItem `json:"feeds"`
	Status      string                   `json:"status"`
	Count       int                      `json:"count"`
	Query       string                   `json:"query"`
	Description string                   `json:"description"`
}

type TTSRequest struct {
	Text         string `json:"text"`
	LanguageCode string `json:"languageCode"`
	SsmlGender   string `json:"ssmlGender"`
	VoiceName    string `json:"voiceName"`
	Url          string `json:"url"`
}

// CONSTANTS

const redis_password = ""
const redis_db = 0
const feed_prefix = "feed:"
const metaData_prefix = "metaData:"
const readerView_prefix = "readerViewContent:"
const feedsearch_prefix = "feedsearch:"
const thumbnailColor_prefix = "thumbnailColor:"

const audio_prefix = "tts:"

const DefaultRed = uint8(128)
const DefaultGreen = uint8(128)
const DefaultBlue = uint8(128)

// const thumbnailColorPrefix = "thumbnailColor_"

// var colorComputeSemaphore = make(chan struct{}, numWorkers)

// const redis_feedsItems_key = "feedsItems"
// const redis_feedDetails_key = "feedDetails"
