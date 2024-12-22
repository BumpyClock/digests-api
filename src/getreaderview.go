package main

import (
	"time"

	readability "github.com/go-shiori/go-readability"
)

func getReaderViewResult(url string) ReaderViewResult {
	readerView, err := getReaderView(url)
	if err != nil {
		log.Errorf("[ReaderView] Error retrieving content for %s: %v", url, err)
		return ReaderViewResult{
			URL:    url,
			Status: "error",
			Error:  err,
		}
	}
	return ReaderViewResult{
		URL:         url,
		Status:      "ok",
		ReaderView:  readerView.Content,
		Title:       readerView.Title,
		SiteName:    readerView.SiteName,
		Image:       readerView.Image,
		Favicon:     readerView.Favicon,
		TextContent: readerView.TextContent,
	}
}

func getReaderView(url string) (readability.Article, error) {
	article, err := readability.FromURL(url, 30*time.Second)
	if err != nil {
		log.Errorf("[ReaderView] Failed to parse %s: %v", url, err)
		return readability.Article{}, err
	}
	return article, nil
}
