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

type ThumbnailFinder struct {
	cache sync.Map
}

func NewThumbnailFinder() *ThumbnailFinder {
	return &ThumbnailFinder{}
}

func GetMetaData(url string) (link2json.MetaDataResponseItem, error) {
	// Get metadata from the URL
	if url == "" || url == "http://" || url == "://" || url == "about:blank" {
		return link2json.MetaDataResponseItem{}, fmt.Errorf("URL is empty")
	}
	metaData, err := link2json.GetMetadata(url)
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
		return thumbnail
	}

	thumbnail = tf.extractThumbnailFromContent(item.Content)
	if thumbnail != "" {
		return thumbnail
	}

	if item.Link != "" {
		thumbnail, err := tf.fetchImageFromSource(item.Link)
		if err != nil {
			fmt.Println("Error fetching image:", err)
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
		fmt.Println("Error parsing content:", err)
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
			return parsedURL.Scheme + "://" + parsedURL.Host + src, nil
		}
		return src, nil
	}
	return "", nil
}
func extractColorFromThumbnail_prominentColor(url string) (r, g, b uint8) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("Recovered from panic while processing URL %s: %v", url, rec)
			r, g, b = 128, 128, 128
			cacheDefaultColor(url)
		}
	}()

	if url == "" {
		return 128, 128, 128 // RGB values for gray
	}
	cachePrefix := "thumbnailColor_"
	var cachedColor RGBColor

	if err := cache.Get(cachePrefix, url, &cachedColor); err == nil {
		log.Printf("[extractColorFromThumbnail_prominentColor] Found cached color for %s : %v", url, cachedColor)
		return cachedColor.R, cachedColor.G, cachedColor.B
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		log.Printf("Error creating request for prominentColor: %v", err)
		cacheDefaultColor(url)
		return 128, 128, 128
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		cacheDefaultColor(url)
		return 128, 128, 128
	}
	defer resp.Body.Close()

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		cacheDefaultColor(url)
		return 128, 128, 128
	}
	log.Printf("Starting color extraction for %s", url)

	resizedImage := img
	bounds := resizedImage.Bounds()
	imgNRGBA := image.NewNRGBA(bounds)
	draw.Draw(imgNRGBA, bounds, resizedImage, bounds.Min, draw.Src)

	if imgNRGBA == nil {
		log.Printf("imgNRGBA is nil for URL %s", url)
		cacheDefaultColor(url)
		return 128, 128, 128
	}

	// Try extracting colors with the background mask
	colors, err := prominentcolor.KmeansWithAll(prominentcolor.ArgumentDefault, imgNRGBA, prominentcolor.DefaultK, 1, prominentcolor.GetDefaultMasks())
	if err != nil || len(colors) == 0 {
		log.Println("Error extracting prominent color with background mask: ", err)
		// Retry without the background mask
		colors, err = prominentcolor.KmeansWithAll(prominentcolor.ArgumentDefault, imgNRGBA, prominentcolor.DefaultK, 1, nil)
		if err != nil || len(colors) == 0 {
			log.Println("Error extracting prominent color without background mask: ", err)
			cacheDefaultColor(url)
			return 128, 128, 128
		}
	}

	if len(colors) > 0 {
		if err := cache.Set(cachePrefix, url, colors[0].Color, 24*time.Hour); err != nil {
			log.Printf("[extractColorFromThumbnail_prominentColor] Failed to cache thumbnailColor for %s: %v", url, err)
		}
		return uint8(colors[0].Color.R), uint8(colors[0].Color.G), uint8(colors[0].Color.B)
	}

	cacheDefaultColor(url)
	return 128, 128, 128
}

func cacheDefaultColor(url string) {
	cachePrefix := "thumbnailColor_"
	defaultColor := RGBColor{128, 128, 128}
	if err := cache.Set(cachePrefix, url, defaultColor, 24*time.Hour); err != nil {
		log.Printf("[extractColorFromThumbnail_prominentColor] Failed to cache default color for %s: %v", url, err)
	}
}

func DiscoverFavicon(pageURL string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", pageURL, nil)
	if err != nil {
		log.Printf("Error creating request for favicon discovery: %v", err)
		return ""
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.36")

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("Error fetching page for favicon: %v", err)
		return ""
	}
	defer resp.Body.Close()

	// Parse the HTML document
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
						return // Found the favicon, no need to continue searching
					}
				}
			}
		}

		for c := n.FirstChild; c != nil && favicon == ""; c = c.NextSibling {
			findNodes(c) // Recursive search
		}
	}

	findNodes(doc)

	// Resolve relative favicon URL
	if favicon != "" && !strings.HasPrefix(favicon, "http") {
		if parsedFaviconURL, err := url.Parse(favicon); err == nil {
			if baseUrl, err := url.Parse(pageURL); err == nil {
				favicon = baseUrl.ResolveReference(parsedFaviconURL).String()
			}
		}
	}

	return favicon
}
