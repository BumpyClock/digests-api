package main

import (
	"html/template"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
)

func getFeedHandler(w http.ResponseWriter, r *http.Request) {
	urls := []string{
		"https://www.theverge.com/rss/index.xml",
		"https://www.wired.com/feed/tag/ai/latest/rss",
		"https://www.wired.com/feed/category/ideas/latest/rss",
		"https://ryanmulligan.dev/feed.xml",
		"https://techcrunch.com/feed/",
		"https://pluralistic.net/feed/",
		"https://www.makeuseof.com/feed/",
		"https://daringfireball.net/feeds/main",
		"https://www.ghacks.net/feed/",
		"https://engadget.com/rss.xml",
	}

	feeds := processURLs(urls)

	// Flatten all feed items into a single slice
	var allItems []FeedResponseItem
	for _, feed := range feeds {
		for _, item := range *feed.Items {
			item.SiteTitle = feed.SiteTitle
			item.FeedTitle = feed.FeedTitle
			item.Favicon = feed.Favicon
			time, err := parseTime(item.Published)
			log.Info("Time for item: ", time, " Error: ", err)
			if err == nil {
				item.Published = time.Format("02 Jan 06 3:04 PM")
			}
			allItems = append(allItems, item)
		}
		//print the total number of items processed
	}
	log.Printf("Total number of items processed: %d", len(allItems))

	// Sort all items by published date
	sortAllFeedItems(&allItems)

	// Print allItems to the console
	for _, item := range allItems {
		log.Info(item.Published, " - ", item.Title, " - ", item.SiteTitle, " - ", item.FeedTitle, " - ", item.Favicon)
	}

	tmpl, err := loadTemplates()
	if err != nil {
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		log.Printf("Failed to load template: %v", err)
		return
	}

	// Render the template with the sorted items
	err = tmpl.ExecuteTemplate(w, "feed", struct{ Items []FeedResponseItem }{Items: allItems})
	if err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		log.Printf("Failed to render template: %v", err)
		return
	}
}

// sortAllFeedItems sorts a slice of FeedResponseItem by Published date in descending order.
func sortAllFeedItems(items *[]FeedResponseItem) {
	sort.Slice(*items, func(i, j int) bool {
		timeI, errI := parseTime((*items)[i].Published)
		timeJ, errJ := parseTime((*items)[j].Published)

		// Convert times to local time zone
		timeI = timeI.Local()
		timeJ = timeJ.Local()

		// If both times are unparsable, keep their original order
		if errI != nil && errJ != nil {
			return i < j
		}

		// If only timeI is unparsable, place it after timeJ
		if errI != nil {
			return false
		}

		// If only timeJ is unparsable, place it after timeI
		if errJ != nil {
			return true
		}

		// If both times are parsable, compare them
		return timeI.After(timeJ) // Descending order
	})
}

func loadTemplates() (*template.Template, error) {
	tmpl := template.New("")

	// Load the main feed template
	_, err := tmpl.ParseFiles("templates/components/feed/feed.tmpl")
	if err != nil {
		return nil, err
	}

	// Load component templates
	componentDirs, err := filepath.Glob("templates/components/*")
	if err != nil {
		return nil, err
	}

	for _, dir := range componentDirs {
		// Ensure it's a directory
		if !strings.HasSuffix(dir, ".tmpl") { // Assuming directories don't end with '.tmpl'
			files, err := filepath.Glob(dir + "/*.tmpl")
			if err != nil {
				log.Printf("Failed to find templates in %s: %v", dir, err)
				continue
			}

			if _, err := tmpl.ParseFiles(files...); err != nil {
				log.Printf("Failed to parse templates in %s: %v", dir, err)
			}
		}
	}

	return tmpl, nil
}
