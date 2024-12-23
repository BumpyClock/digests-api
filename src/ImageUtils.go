// Package main provides the main functionality for the web server.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"

	"github.com/EdlinOrg/prominentcolor"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/mmcdole/gofeed"
	"github.com/sirupsen/logrus"
)

const (
	thumbnailColorPrefix = thumbnailColor_prefix
	httpTimeout          = 10 * time.Second
	cacheDuration        = 24 * time.Hour
	defaultColor         = 128
	userAgent            = "Mozilla/5.0 (compatible; FeedParser/1.0)"
	collyUserAgent       = "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)"
)

type ThumbnailFinder struct {
	cache sync.Map
}

func NewThumbnailFinder() *ThumbnailFinder {
	return &ThumbnailFinder{}
}

/**
 * @function GetMetaData
 * @description Fetches a web page (targetURL) using Colly, extracting Open Graph tags
 * and JSON-LD data to produce a MetaDataResponseItem. It also attempts to discover
 * the favicon and domain if not provided by OG or JSON-LD.
 * @param {string} targetURL The URL to visit and parse.
 * @returns {MetaDataResponseItem, error} On success, returns a struct with combined
 * metadata from both OG tags and JSON-LD. Returns an error if the fetch or parse fails.
 * @dependencies colly.NewCollector, colly.UserAgent, c.OnHTML, c.OnRequest, c.Visit, url.Parse, strconv.Atoi, json.Unmarshal, goquery.Selection, log
 */
func GetMetaData(targetURL string) (MetaDataResponseItem, error) {
	// Basic validation
	if targetURL == "" || targetURL == "http://" || targetURL == "://" || targetURL == "about:blank" {
		log.WithFields(logrus.Fields{
			"url": targetURL,
		}).Error("[GetMetaData] URL is empty or invalid")
		return MetaDataResponseItem{}, fmt.Errorf("URL is empty or invalid")
	}

	// Prepare the collector with a friendly user agent that tricks sites
	c := colly.NewCollector(
		colly.UserAgent(collyUserAgent),
	)

	// Our enriched metadata struct
	metaData := MetaDataResponseItem{
		Images: []WebMedia{},
		Videos: []WebMedia{},
	}

	// STEP 1: OnHTML("meta", ...) to capture Open Graph fields.
	c.OnHTML("meta", func(e *colly.HTMLElement) {
		property := e.Attr("property")
		content := e.Attr("content")
		name := e.Attr("name")
		if (property == "" && name == "") || content == "" {
			return
		}

		// Check for og: fields
		parts := strings.Split(property, ":")
		if len(parts) < 2 || parts[0] != "og" {
			// Check for theme-color
			if name == "theme-color" {
				metaData.ThemeColor = content
			}
			return
		}

		switch property {
		case "og:title":
			metaData.Title = content
		case "og:description":
			metaData.Description = content
		case "og:site_name":
			metaData.Sitename = content
		case "og:url":
			metaData.URL = content
		case "og:type":
			metaData.Type = content
		case "og:locale":
			metaData.Locale = content
		case "og:determiner":
			metaData.Determiner = content

		// Images
		case "og:image":
			metaData.Images = append(metaData.Images, WebMedia{URL: content})
		case "og:image:width", "og:image:height", "og:image:alt", "og:image:type", "og:image:secure_url":
			if len(metaData.Images) > 0 {
				idx := len(metaData.Images) - 1
				switch property {
				case "og:image:width":
					if w, err := strconv.Atoi(content); err == nil {
						metaData.Images[idx].Width = w
					}
				case "og:image:height":
					if h, err := strconv.Atoi(content); err == nil {
						metaData.Images[idx].Height = h
					}
				case "og:image:alt":
					metaData.Images[idx].Alt = content
				case "og:image:type":
					metaData.Images[idx].Type = content
				case "og:image:secure_url":
					metaData.Images[idx].SecureURL = content
				}
			}

		// Videos
		case "og:video:url":
			metaData.Videos = append(metaData.Videos, WebMedia{URL: content})
		case "og:video:width", "og:video:height", "og:video:type", "og:video:secure_url", "og:video:duration", "og:video:release_date", "og:video:tag":
			if len(metaData.Videos) > 0 {
				idx := len(metaData.Videos) - 1
				switch property {
				case "og:video:width":
					if w, err := strconv.Atoi(content); err == nil {
						metaData.Videos[idx].Width = w
					}
				case "og:video:height":
					if h, err := strconv.Atoi(content); err == nil {
						metaData.Videos[idx].Height = h
					}
				case "og:video:type":
					metaData.Videos[idx].Type = content
				case "og:video:secure_url":
					metaData.Videos[idx].SecureURL = content
				case "og:video:duration":
					if d, err := strconv.Atoi(content); err == nil {
						metaData.Videos[idx].Duration = d
					}
				case "og:video:release_date":
					metaData.Videos[idx].ReleaseDate = content
				case "og:video:tag":
					tags := strings.Split(content, ",")
					for _, t := range tags {
						metaData.Videos[idx].Tags = append(metaData.Videos[idx].Tags, strings.TrimSpace(t))
					}
				}
			}
		}
	})

	// STEP 2: OnHTML("script[type='application/ld+json']", ...) to parse JSON-LD.
	c.OnHTML("script[type='application/ld+json']", func(e *colly.HTMLElement) {
		rawJSON := e.Text
		// Attempt to parse the JSON-LD
		var ldData interface{}
		if err := json.Unmarshal([]byte(rawJSON), &ldData); err == nil {
			// If we have no JSONLD in metaData, store this entire ldData.
			// If you want to further parse Title, etc., you can do so here.
			metaData.Raw = ldData
			// Optionally, you can write logic to unify some fields from ld+json
			// with your metaData if you want to override or fill in missing values.
		}
	})

	// STEP 3: Attempt domain detection on request
	c.OnRequest(func(r *colly.Request) {
		if metaData.Domain == "" {
			if parsedURL, err := url.Parse(r.URL.String()); err == nil {
				metaData.Domain = parsedURL.Host
			}
		}
	})

	// STEP 4: After we parse <head>, try to find a favicon if none is set
	c.OnHTML("head", func(e *colly.HTMLElement) {
		if metaData.Favicon == "" {
			e.DOM.Find("link[rel]").Each(func(_ int, s *goquery.Selection) {
				rel := s.AttrOr("rel", "")
				href := s.AttrOr("href", "")
				relValues := strings.Fields(rel)
				for _, rv := range relValues {
					if rv == "icon" || rv == "shortcut" || rv == "apple-touch-icon" {
						if href != "" {
							metaData.Favicon = e.Request.AbsoluteURL(href)
							return
						}
					}
				}
			})
		}
	})

	// STEP 5: Visit the target page
	if err := c.Visit(targetURL); err != nil {
		log.WithFields(logrus.Fields{
			"url":   targetURL,
			"error": err,
		}).Error("[GetMetaData] Error visiting URL")
		return MetaDataResponseItem{}, fmt.Errorf("error visiting URL %s: %w", targetURL, err)
	}

	return metaData, nil
}

/**
 * @function FindThumbnailForItem
 * @description Finds a thumbnail for a given feed item.
 *              It first checks the cache, then enclosures, then content, and finally fetches metadata.
 * @param {*gofeed.Item} item The feed item to find a thumbnail for.
 * @returns {string} The URL of the thumbnail, or an empty string if no thumbnail was found.
 * @dependencies extractThumbnailFromEnclosures, extractThumbnailFromContent, GetMetaData, log
 */
func (tf *ThumbnailFinder) FindThumbnailForItem(item *gofeed.Item) string {
	if thumb, ok := tf.cache.Load(item.Link); ok {
		log.WithFields(logrus.Fields{
			"url": item.Link,
		}).Debug("[FindThumbnailForItem] Found cached thumbnail")
		return thumb.(string)
	}

	thumbnail := tf.extractThumbnailFromEnclosures(item.Enclosures)
	if thumbnail != "" {
		log.WithFields(logrus.Fields{
			"url": item.Link,
		}).Debug("[FindThumbnailForItem] Found thumbnail in enclosures")
		tf.cache.Store(item.Link, thumbnail)
		return thumbnail
	}

	thumbnail = tf.extractThumbnailFromContent(item.Content)
	if thumbnail != "" {
		log.WithFields(logrus.Fields{
			"url": item.Link,
		}).Debug("[FindThumbnailForItem] Found thumbnail in content")
		tf.cache.Store(item.Link, thumbnail)
		return thumbnail
	}

	if item.Link != "" {
		log.WithFields(logrus.Fields{
			"url": item.Link,
		}).Debug("[FindThumbnailForItem] Fetching metadata")
		metaData, err := GetMetaData(item.Link)
		if err != nil {
			log.WithFields(logrus.Fields{
				"url":   item.Link,
				"error": err,
			}).Error("[FindThumbnailForItem] Error getting metadata")
			return ""
		}

		if len(metaData.Images) > 0 {
			thumbnail = metaData.Images[0].URL
		}
		if thumbnail != "" {
			log.WithFields(logrus.Fields{
				"url": item.Link,
			}).Debug("[FindThumbnailForItem] Found thumbnail in metadata")
			tf.cache.Store(item.Link, thumbnail)
			return thumbnail
		}

		return ""
	}
	return ""
}

/**
 * @function extractThumbnailFromEnclosures
 * @description Extracts a thumbnail URL from a list of enclosures.
 * @param {[]*gofeed.Enclosure} enclosures The list of enclosures to search.
 * @returns {string} The URL of the first image enclosure found, or an empty string if none were found.
 */
func (tf *ThumbnailFinder) extractThumbnailFromEnclosures(enclosures []*gofeed.Enclosure) string {
	for _, e := range enclosures {
		if strings.HasPrefix(e.Type, "image/") {
			return e.URL
		}
	}
	return ""
}

/**
 * @function extractThumbnailFromContent
 * @description Extracts a thumbnail URL from HTML content.
 * @param {string} content The HTML content to search.
 * @returns {string} The URL of the first image found in the content, or an empty string if none were found.
 * @dependencies goquery.NewDocumentFromReader, log
 */
func (tf *ThumbnailFinder) extractThumbnailFromContent(content string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Error("[extractThumbnailFromContent] Error parsing content")
		return ""
	}

	if src, exists := doc.Find("img").First().Attr("src"); exists {
		return src
	}
	return ""
}

/**
 * @function extractColorFromThumbnail_prominentColor
 * @description Extracts the prominent color from an image URL using the prominentcolor library.
 *              It first checks the cache, then validates the URL, downloads the image, decodes it,
 *              and finally extracts the color using the K-means algorithm.
 * @param {string} imageURL The URL of the image to extract the color from.
 * @returns {r, g, b uint8} The red, green, and blue components of the prominent color.
 * @dependencies httpClient, cache, prominentcolor, image.Decode, log
 */
func extractColorFromThumbnail_prominentColor(imageURL string) (r, g, b uint8) {
	defer func() {
		if rec := recover(); rec != nil {
			log.WithFields(logrus.Fields{
				"url":   imageURL,
				"panic": rec,
			}).Error("[extractColorFromThumbnail_prominentColor] Recovered from panic")
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
		log.WithFields(logrus.Fields{
			"url":   imageURL,
			"color": cachedColor,
		}).Debug("[extractColorFromThumbnail_prominentColor] Found cached color")
		return cachedColor.R, cachedColor.G, cachedColor.B
	}

	// Validate the image URL
	parsedURL, err := url.Parse(imageURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		log.WithFields(logrus.Fields{
			"url":   imageURL,
			"error": err,
		}).Error("[extractColorFromThumbnail_prominentColor] Invalid image URL")
		cacheDefaultColor(imageURL)
		return 128, 128, 128
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		log.WithFields(logrus.Fields{
			"url":   imageURL,
			"error": err,
		}).Error("[extractColorFromThumbnail_prominentColor] Error creating request")
		cacheDefaultColor(imageURL)
		return 128, 128, 128
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		log.WithFields(logrus.Fields{
			"url":   imageURL,
			"error": err,
		}).Error("[extractColorFromThumbnail_prominentColor] Failed to download image")
		cacheDefaultColor(imageURL)
		return 128, 128, 128
	}
	defer resp.Body.Close()

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		log.WithFields(logrus.Fields{
			"url":   imageURL,
			"error": err,
		}).Error("[extractColorFromThumbnail_prominentColor] Failed to decode image")
		cacheDefaultColor(imageURL)
		return 128, 128, 128
	}
	log.WithFields(logrus.Fields{
		"url": imageURL,
	}).Debug("[extractColorFromThumbnail_prominentColor] Starting color extraction")

	bounds := img.Bounds()
	imgNRGBA := image.NewNRGBA(bounds)
	draw.Draw(imgNRGBA, bounds, img, bounds.Min, draw.Src)

	if imgNRGBA == nil {
		log.WithFields(logrus.Fields{
			"url": imageURL,
		}).Error("[extractColorFromThumbnail_prominentColor] imgNRGBA is nil")
		cacheDefaultColor(imageURL)
		return 128, 128, 128
	}

	colors, err := prominentcolor.KmeansWithAll(prominentcolor.ArgumentDefault, imgNRGBA, prominentcolor.DefaultK, 1, prominentcolor.GetDefaultMasks())
	if err != nil || len(colors) == 0 {
		log.WithFields(logrus.Fields{
			"url":   imageURL,
			"error": err,
		}).Error("[extractColorFromThumbnail_prominentColor] Error extracting prominent color with background mask")
		colors, err = prominentcolor.KmeansWithAll(prominentcolor.ArgumentDefault, imgNRGBA, prominentcolor.DefaultK, 1, nil)
		if err != nil || len(colors) == 0 {
			log.WithFields(logrus.Fields{
				"url":   imageURL,
				"error": err,
			}).Error("[extractColorFromThumbnail_prominentColor] Error extracting prominent color without background mask")
			cacheDefaultColor(imageURL)
			return 128, 128, 128
		}
	}

	if len(colors) > 0 {
		extractedColor := RGBColor{uint8(colors[0].Color.R), uint8(colors[0].Color.G), uint8(colors[0].Color.B)}
		log.WithFields(logrus.Fields{
			"url":   imageURL,
			"color": extractedColor,
		}).Debug("[extractColorFromThumbnail_prominentColor] Extracted color")
		if err := cache.Set(cachePrefix, imageURL, extractedColor, 24*time.Hour); err != nil {
			log.WithFields(logrus.Fields{
				"url":   imageURL,
				"color": extractedColor,
				"error": err,
			}).Error("[extractColorFromThumbnail_prominentColor] Failed to cache color")
		}
		return extractedColor.R, extractedColor.G, extractedColor.B
	}

	// Cache the default color if extraction fails
	cacheDefaultColor(imageURL)
	return 128, 128, 128
}

/**
 * @function cacheDefaultColor
 * @description Caches the default color for a given image URL.
 * @param {string} imageURL The URL of the image to cache the default color for.
 * @dependencies cache, log
 */
func cacheDefaultColor(imageURL string) {
	cachePrefix := thumbnailColorPrefix
	defaultColor := RGBColor{defaultColor, defaultColor, defaultColor}

	if err := cache.Set(cachePrefix, imageURL, defaultColor, cacheDuration); err != nil {
		log.WithFields(logrus.Fields{
			"url":   imageURL,
			"error": err,
		}).Error("[cacheDefaultColor] Failed to cache default color")
	}
}

/**
 * @function DiscoverFavicon
 * @description Discovers the favicon for a given page URL.
 *              It sends an HTTP GET request to the page, parses the HTML response,
 *              and searches for link elements with rel attributes containing "icon", "shortcut", or "apple-touch-icon".
 * @param {string} pageURL The URL of the page to discover the favicon for.
 * @returns {string} The URL of the favicon, or an empty string if no favicon was found.
 * @dependencies httpClient, html.Parse, findFavicon, url.Parse, log
 */
func DiscoverFavicon(pageURL string) string {
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		log.WithFields(logrus.Fields{
			"url":   pageURL,
			"error": err,
		}).Error("[DiscoverFavicon] Error creating request")
		return ""
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		log.WithFields(logrus.Fields{
			"url":   pageURL,
			"error": err,
		}).Error("[DiscoverFavicon] Error fetching page")
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.WithFields(logrus.Fields{
			"url":    pageURL,
			"status": resp.Status,
		}).Error("[DiscoverFavicon] Error fetching page")
		return ""
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		log.WithFields(logrus.Fields{
			"url":   pageURL,
			"error": err,
		}).Error("[DiscoverFavicon] Error parsing HTML")
		return ""
	}

	favicon := findFavicon(doc)

	if favicon != "" && !strings.HasPrefix(favicon, "http") {
		if parsedFaviconURL, err := url.Parse(favicon); err == nil {
			if baseURL, err := url.Parse(pageURL); err == nil {
				favicon = baseURL.ResolveReference(parsedFaviconURL).String()
			}
		}
	}

	return favicon
}

/**
 * @function findFavicon
 * @description Recursively searches an HTML node tree for a favicon link.
 * @param {*html.Node} n The root node of the HTML tree to search.
 * @returns {string} The URL of the favicon, or an empty string if no favicon was found.
 */
func findFavicon(n *html.Node) string {
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
					return hrefAttr
				}
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if favicon := findFavicon(c); favicon != "" {
			return favicon
		}
	}

	return ""
}

/**
 * @function DiscoverFaviconWithColly
 * @description Discovers the favicon for a given page URL using colly.
 * @param {*colly.Collector} c The colly collector to use for the request.
 * @param {string} pageURL The URL of the page to discover the favicon for.
 * @returns {string} The URL of the favicon, or an empty string if no favicon was found.
 * @dependencies colly.HTMLElement, c.OnHTML, c.Visit
 */
func DiscoverFaviconWithColly(c *colly.Collector, pageURL string) string {
	var favicon string

	c.OnHTML("link[rel]", func(e *colly.HTMLElement) {
		rel := e.Attr("rel")
		href := e.Attr("href")

		relValues := strings.Fields(rel)
		for _, relValue := range relValues {
			if relValue == "icon" || relValue == "shortcut" || relValue == "apple-touch-icon" {
				if href != "" {
					favicon = e.Request.AbsoluteURL(href)
					return
				}
			}
		}
	})

	c.Visit(pageURL)

	return favicon
}
