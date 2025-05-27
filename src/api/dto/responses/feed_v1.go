// ABOUTME: Response DTOs matching the v1 API schema for compatibility
// ABOUTME: Ensures backward compatibility with existing API consumers

package responses

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
	
	"digests-app-api/core/domain"
)

// FeedV1Response represents a feed in the v1 API response format
type FeedV1Response struct {
	Type         string          `json:"type,omitempty"`         // e.g., "podcast"
	GUID         string          `json:"guid"`                   // Unique identifier
	Status       string          `json:"status"`                 // e.g., "ok"
	SiteTitle    string          `json:"siteTitle,omitempty"`
	FeedTitle    string          `json:"feedTitle"`
	FeedURL      string          `json:"feedUrl"`
	Description  string          `json:"description"`
	Link         string          `json:"link"`
	LastUpdated  string          `json:"lastUpdated"`            // RFC3339 format
	LastRefreshed string         `json:"lastRefreshed"`          // RFC3339 format
	Published    string          `json:"published,omitempty"`
	Author       *AuthorV1       `json:"author,omitempty"`
	Language     string          `json:"language,omitempty"`
	Favicon      string          `json:"favicon,omitempty"`
	Image        string          `json:"image,omitempty"`        // Feed image
	Categories   string          `json:"categories,omitempty"`
	Subtitle     string          `json:"subtitle,omitempty"`     // Podcast subtitle
	Summary      string          `json:"summary,omitempty"`      // Podcast summary
	Items        []ItemV1Response `json:"items"`
}

// AuthorV1 represents author information in v1 format
type AuthorV1 struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

// ItemV1Response represents a feed item in the v1 API response format
type ItemV1Response struct {
	Type           string       `json:"type,omitempty"`             // e.g., "podcast"
	ID             string       `json:"id"`                         // UUID
	Title          string       `json:"title"`
	Description    string       `json:"description"`
	Link           string       `json:"link"`
	Author         string       `json:"author,omitempty"`
	Published      string       `json:"published"`                  // RFC3339 format
	Created        string       `json:"created,omitempty"`          // RFC3339 format
	Content        string       `json:"content,omitempty"`
	ContentEncoded string       `json:"content_encoded,omitempty"`
	Categories     []string     `json:"categories,omitempty"`
	Duration       string       `json:"duration,omitempty"`         // e.g., "00:28:19"
	Thumbnail      string       `json:"thumbnail,omitempty"`        // Image URL
	ThumbnailColor *ColorV1     `json:"thumbnailColor,omitempty"`
	ThumbnailColorComputed string `json:"thumbnailColorComputed,omitempty"`
	Enclosures     []EnclosureV1 `json:"enclosures,omitempty"`     // Media enclosures
	
	// Podcast metadata
	Episode      int    `json:"episode,omitempty"`          // Episode number
	Season       int    `json:"season,omitempty"`           // Season number
	EpisodeType  string `json:"episodeType,omitempty"`      // e.g., "full", "trailer"
	Subtitle     string `json:"subtitle,omitempty"`         // Episode subtitle
	Summary      string `json:"summary,omitempty"`          // Episode summary
	Image        string `json:"image,omitempty"`            // Episode image
	
	// Podcast specific fields (legacy, kept for compatibility)
	URL      string `json:"url,omitempty"`              // Media URL
	Length   string `json:"length,omitempty"`           // File size
	MimeType string `json:"mime_type,omitempty"`       // MIME type for media
}

// EnclosureV1 represents media enclosure information
type EnclosureV1 struct {
	URL    string `json:"url"`
	Length string `json:"length,omitempty"`
	Type   string `json:"type,omitempty"`
}

// ColorV1 represents RGB color values
type ColorV1 struct {
	R int `json:"r"`
	G int `json:"g"`
	B int `json:"b"`
}

// ParseFeedsV1Response represents the v1 API response for parsing multiple feeds
type ParseFeedsV1Response struct {
	Feeds []FeedV1Response `json:"feeds"`
	// Note: No $schema field to match current API exactly
}

// ConvertToV1Response converts from our internal format to v1 API format
func ConvertToV1Response(feeds []FeedResponse) ParseFeedsV1Response {
	v1Feeds := make([]FeedV1Response, 0, len(feeds))
	
	for _, feed := range feeds {
		v1Feed := convertFeedToV1(feed)
		v1Feeds = append(v1Feeds, v1Feed)
	}
	
	return ParseFeedsV1Response{
		Feeds: v1Feeds,
	}
}

// ConvertDomainFeedsToV1Response converts domain feeds directly to v1 format
func ConvertDomainFeedsToV1Response(feeds []*domain.Feed) ParseFeedsV1Response {
	return ConvertDomainFeedsToV1ResponseWithColors(feeds, nil)
}

// ConvertDomainFeedsToV1ResponseWithColors converts domain feeds with thumbnail colors
func ConvertDomainFeedsToV1ResponseWithColors(feeds []*domain.Feed, thumbnailColors map[string]*domain.RGBColor) ParseFeedsV1Response {
	v1Feeds := make([]FeedV1Response, 0, len(feeds))
	
	for _, feed := range feeds {
		if feed == nil {
			continue
		}
		
		// Generate GUID from URL
		guid := generateGUID(feed.URL)
		
		v1Feed := FeedV1Response{
			Type:          feed.FeedType,
			GUID:          guid,
			Status:        "ok",
			SiteTitle:     feed.Title, // TODO: fetch from metadata
			FeedTitle:     feed.Title,
			FeedURL:       feed.URL,
			Description:   feed.Description,
			Link:          stripProtocol(feed.Link),
			LastUpdated:   formatTimeWithOriginalZone(feed.LastUpdated),
			LastRefreshed: time.Now().UTC().Format(time.RFC3339),
			Language:      feed.Language,
			Favicon:       feed.Favicon,
			Categories:    feed.Categories,
			Image:         feed.Image,
			Subtitle:      feed.Subtitle,
			Summary:       feed.Subtitle, // Use subtitle as summary if not available
			Items:         make([]ItemV1Response, 0, len(feed.Items)),
		}
		
		// Set author
		if feed.Author != nil {
			v1Feed.Author = &AuthorV1{
				Name:  feed.Author.Name,
				Email: feed.Author.Email,
			}
		} else {
			// Include null author field
			v1Feed.Author = nil
		}
		
		// Set published
		if feed.Published != nil {
			v1Feed.Published = formatTimeWithOriginalZone(*feed.Published)
		} else {
			v1Feed.Published = ""
		}
		
		// Convert items
		for _, item := range feed.Items {
			v1Item := ItemV1Response{
				Type:           feed.FeedType, // Use feed type for items
				ID:             item.ID,
				Title:          item.Title,
				Description:    item.Description,
				Link:           item.Link,
				Author:         item.Author,
				Published:      formatTimeWithOriginalZone(item.Published),
				Content:        item.Content,
				ContentEncoded: item.ContentEncoded,
				Duration:       formatDuration(item.Duration),
				Thumbnail:      item.Thumbnail,
				Subtitle:       item.Subtitle,
				Summary:        item.Summary,
				Image:          item.Image,
				Episode:        item.Episode,
				Season:         item.Season,
				EpisodeType:    item.EpisodeType,
			}
			
			// Set created
			if item.Created != nil {
				v1Item.Created = formatTimeWithOriginalZone(*item.Created)
			}
			
			// Set categories
			if len(item.Categories) > 0 {
				v1Item.Categories = item.Categories
			}
			
			// Set enclosures
			if len(item.Enclosures) > 0 {
				v1Item.Enclosures = make([]EnclosureV1, len(item.Enclosures))
				for k, enc := range item.Enclosures {
					v1Item.Enclosures[k] = EnclosureV1{
						URL:    enc.URL,
						Length: enc.Length,
						Type:   enc.Type,
					}
				}
			}
			
			// Set thumbnail color
			if v1Item.Thumbnail != "" && thumbnailColors != nil {
				if color, exists := thumbnailColors[v1Item.Thumbnail]; exists && color != nil {
					v1Item.ThumbnailColor = &ColorV1{
						R: int(color.R),
						G: int(color.G),
						B: int(color.B),
					}
					v1Item.ThumbnailColorComputed = "yes"
				} else {
					// Default gray color
					v1Item.ThumbnailColor = &ColorV1{R: 128, G: 128, B: 128}
					v1Item.ThumbnailColorComputed = "no"
				}
			} else {
				// Default gray color
				v1Item.ThumbnailColor = &ColorV1{R: 128, G: 128, B: 128}
				v1Item.ThumbnailColorComputed = "no"
			}
			
			v1Feed.Items = append(v1Feed.Items, v1Item)
		}
		
		v1Feeds = append(v1Feeds, v1Feed)
	}
	
	return ParseFeedsV1Response{
		Feeds: v1Feeds,
	}
}

// convertFeedToV1 converts a FeedResponse to FeedV1Response
func convertFeedToV1(feed FeedResponse) FeedV1Response {
	// This is for the simple FeedResponse conversion
	guid := generateGUID(feed.URL)
	
	v1Feed := FeedV1Response{
		Type:          "rss", // Default type
		GUID:          guid,
		Status:        "ok",
		FeedTitle:     feed.Title,
		FeedURL:       feed.URL,
		Description:   feed.Description,
		Link:          feed.URL,
		LastUpdated:   feed.LastUpdated.Format(time.RFC3339),
		LastRefreshed: time.Now().UTC().Format(time.RFC3339),
		Items:         make([]ItemV1Response, len(feed.Items)),
	}
	
	// Convert items
	for j, item := range feed.Items {
		v1Feed.Items[j] = ItemV1Response{
			ID:          item.ID,
			Title:       item.Title,
			Description: item.Description,
			Link:        item.Link,
			Author:      item.Author,
			Published:   item.Published.Format(time.RFC3339),
		}
	}
	
	return v1Feed
}

// generateGUID creates a consistent GUID from a URL
func generateGUID(url string) string {
	hash := sha256.Sum256([]byte(url))
	return hex.EncodeToString(hash[:])
}

// detectFeedType tries to determine if this is a podcast or regular RSS feed
func detectFeedType(feed FeedResponse) string {
	// Simple heuristic - could be enhanced
	// Check if any items have media enclosures (would need to add this field)
	return "rss"
}

// stripProtocol removes the protocol (http:// or https://) from a URL
func stripProtocol(url string) string {
	if strings.HasPrefix(url, "https://") {
		return strings.TrimPrefix(url, "https://")
	}
	if strings.HasPrefix(url, "http://") {
		return strings.TrimPrefix(url, "http://")
	}
	return url
}

// formatTimeWithOriginalZone formats time preserving the original timezone information
func formatTimeWithOriginalZone(t time.Time) string {
	// The current API uses RFC3339 format which preserves timezone
	return t.Format(time.RFC3339)
}

// formatDuration converts seconds (as string) to HH:MM:SS format
func formatDuration(durationStr string) string {
	if durationStr == "" {
		return ""
	}
	
	// Parse seconds
	seconds, err := strconv.Atoi(durationStr)
	if err != nil {
		// If it's already formatted or can't parse, return as-is
		return durationStr
	}
	
	// Convert to HH:MM:SS
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	
	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
	}
	return fmt.Sprintf("%02d:%02d", minutes, secs)
}