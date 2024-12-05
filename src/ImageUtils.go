package main

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	link2json "github.com/BumpyClock/go-link2json"
	"golang.org/x/net/html"

	"github.com/EdlinOrg/prominentcolor"
	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
)

const thumbnailColorPrefix = thumbnailColor_prefix

type ThumbnailFinder struct {
	cache sync.Map
}

func NewThumbnailFinder() *ThumbnailFinder {
	return &ThumbnailFinder{}
}

func GetMetaData(targetURL string) (link2json.MetaDataResponseItem, error) {
	if targetURL == "" || targetURL == "http://" || targetURL == "://" || targetURL == "about:blank" {
		return link2json.MetaDataResponseItem{}, fmt.Errorf("URL is empty or invalid")
	}

	metaData, err := link2json.GetMetadata(targetURL)
	if err != nil {
		return link2json.MetaDataResponseItem{}, err
	}

	return *metaData, nil
}

func (tf *ThumbnailFinder) FindThumbnailForItem(item *gofeed.Item) string {
	if thumb, ok := tf.cache.Load(item.Link); ok {
		return thumb.(string)
	}

	thumbnail := tf.extractThumbnailFromEnclosures(item.Enclosures)
	if thumbnail != "" {
		tf.cache.Store(item.Link, thumbnail)
		return thumbnail
	}

	thumbnail = tf.extractThumbnailFromContent(item.Content)
	if thumbnail != "" {
		tf.cache.Store(item.Link, thumbnail)
		return thumbnail
	}

	if item.Link != "" {
		thumbnail, err := tf.fetchImageFromSource(item.Link)
		if err != nil {
			log.Printf("Error fetching image for %s: %v", item.Link, err)
			return ""
		}
		tf.cache.Store(item.Link, thumbnail)
		return thumbnail
	}

	return ""
}

func (tf *ThumbnailFinder) extractThumbnailFromEnclosures(enclosures []*gofeed.Enclosure) string {
	for _, e := range enclosures {
		if strings.HasPrefix(e.Type, "image/") {
			return e.URL
		}
	}
	return ""
}

func (tf *ThumbnailFinder) extractThumbnailFromContent(content string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		log.Printf("Error parsing content: %v", err)
		return ""
	}

	if src, exists := doc.Find("img").First().Attr("src"); exists {
		return src
	}
	return ""
}

func (tf *ThumbnailFinder) fetchImageFromSource(pageURL string) (string, error) {
	resp, err := http.Get(pageURL)
	if err != nil {
		return "", fmt.Errorf("error fetching page: %w", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error loading HTTP response body: %w", err)
	}

	src, exists := doc.Find("article img, .content img").First().Attr("src")
	if exists {
		if !strings.HasPrefix(src, "http") {
			parsedURL, err := url.Parse(pageURL)
			if err != nil {
				return "", err
			}
			return parsedURL.ResolveReference(&url.URL{Path: src}).String(), nil
		}
		return src, nil
	}
	return "", nil
}

func extractColorFromThumbnail_prominentColor(imageURL string) (r, g, b uint8) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("Recovered from panic while processing URL %s: %v", imageURL, rec)
			r, g, b = 128, 128, 128
		}
	}()

	if imageURL == "" {
		return 128, 128, 128
	}

	cachePrefix := thumbnailColorPrefix
	var cachedColor RGBColor

	// Attempt to retrieve the color from the cache
	err := cache.Get(cachePrefix, imageURL, &cachedColor)
	if err == nil {
		log.Printf("[extractColorFromThumbnail_prominentColor] Found cached color for %s: %v", imageURL, cachedColor)
		return cachedColor.R, cachedColor.G, cachedColor.B
	}

	// Validate the image URL
	parsedURL, err := url.Parse(imageURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		log.Printf("Invalid image URL %s", imageURL)
		cacheDefaultColor(imageURL)
		return 128, 128, 128
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		log.Printf("Error creating request for prominentColor: %v", err)
		cacheDefaultColor(imageURL)
		return 128, 128, 128
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to download image %s: %v", imageURL, err)
		cacheDefaultColor(imageURL)
		return 128, 128, 128
	}
	defer resp.Body.Close()

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		log.Printf("Failed to decode image %s: %v", imageURL, err)
		cacheDefaultColor(imageURL)
		return 128, 128, 128
	}
	log.Printf("Starting color extraction for %s", imageURL)

	bounds := img.Bounds()
	imgNRGBA := image.NewNRGBA(bounds)
	draw.Draw(imgNRGBA, bounds, img, bounds.Min, draw.Src)

	if imgNRGBA == nil {
		log.Printf("imgNRGBA is nil for URL %s", imageURL)
		cacheDefaultColor(imageURL)
		return 128, 128, 128
	}

	colors, err := prominentcolor.KmeansWithAll(prominentcolor.ArgumentDefault, imgNRGBA, prominentcolor.DefaultK, 1, prominentcolor.GetDefaultMasks())
	if err != nil || len(colors) == 0 {
		log.Printf("Error extracting prominent color with background mask for %s: %v", imageURL, err)
		colors, err = prominentcolor.KmeansWithAll(prominentcolor.ArgumentDefault, imgNRGBA, prominentcolor.DefaultK, 1, nil)
		if err != nil || len(colors) == 0 {
			log.Printf("Error extracting prominent color without background mask for %s: %v", imageURL, err)
			cacheDefaultColor(imageURL)
			return 128, 128, 128
		}
	}

	if len(colors) > 0 {
		extractedColor := RGBColor{uint8(colors[0].Color.R), uint8(colors[0].Color.G), uint8(colors[0].Color.B)}
		log.Printf("Extracted color for %s: %v", imageURL, extractedColor)
		if err := cache.Set(cachePrefix, imageURL, extractedColor, 24*time.Hour); err != nil {
			log.Printf("Failed to cache color for %s: %v", imageURL, err)
		}
		return extractedColor.R, extractedColor.G, extractedColor.B
	}

	// Cache the default color if extraction fails
	cacheDefaultColor(imageURL)
	return 128, 128, 128
}

func cacheDefaultColor(imageURL string) {
	cachePrefix := thumbnailColorPrefix
	defaultColor := RGBColor{128, 128, 128}

	if err := cache.Set(cachePrefix, imageURL, defaultColor, 24*time.Hour); err != nil {
		log.Printf("[cacheDefaultColor] Failed to cache default color for %s: %v", imageURL, err)
	}
}

func DiscoverFavicon(pageURL string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		log.Printf("Error creating request for favicon discovery: %v", err)
		return ""
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; FeedParser/1.0)")

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("Error fetching page for favicon: %v", err)
		return ""
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		log.Printf("Error parsing HTML for favicon: %v", err)
		return ""
	}

	var favicon string
	var findNodes func(*html.Node)
	findNodes = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "link" {
			relAttr := ""
			hrefAttr := ""
			for _, attr := range n.Attr {
				if attr.Key == "rel" {
					relAttr = attr.Val
				} else if attr.Key == "href" {
					hrefAttr = attr.Val
				}
			}

			relValues := strings.Fields(relAttr)
			for _, relValue := range relValues {
				if relValue == "icon" || relValue == "shortcut" || relValue == "apple-touch-icon" {
					if hrefAttr != "" {
						favicon = hrefAttr
						return
					}
				}
			}
		}

		for c := n.FirstChild; c != nil && favicon == ""; c = c.NextSibling {
			findNodes(c)
		}
	}

	findNodes(doc)

	if favicon != "" && !strings.HasPrefix(favicon, "http") {
		if parsedFaviconURL, err := url.Parse(favicon); err == nil {
			if baseURL, err := url.Parse(pageURL); err == nil {
				favicon = baseURL.ResolveReference(parsedFaviconURL).String()
			}
		}
	}

	return favicon
}
