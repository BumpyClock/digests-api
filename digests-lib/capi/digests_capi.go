// ABOUTME: C API wrapper for the Digests library to enable FFI usage
// ABOUTME: Provides C-compatible functions for use in native applications

package main

import "C"
import (
	"context"
	"encoding/json"
	"unsafe"
	
	"github.com/BumpyClock/digests-api/digests-lib"
)

// Global client instance (you might want to handle multiple clients differently)
var client *digests.Client

//export DigestsInit
func DigestsInit() C.int {
	var err error
	client, err = digests.NewClient()
	if err != nil {
		return -1
	}
	return 0
}

//export DigestsInitWithCache
func DigestsInitWithCache(cacheType *C.char, cachePath *C.char) C.int {
	var err error
	
	cacheTypeStr := C.GoString(cacheType)
	cachePathStr := C.GoString(cachePath)
	
	var opt digests.Option
	if cacheTypeStr == "sqlite" {
		opt = digests.WithCacheOption(digests.CacheOption{
			Type:     digests.CacheTypeSQLite,
			FilePath: cachePathStr,
		})
	} else {
		opt = digests.WithCacheOption(digests.CacheOption{
			Type: digests.CacheTypeMemory,
		})
	}
	
	client, err = digests.NewClient(opt)
	if err != nil {
		return -1
	}
	return 0
}

//export DigestsClose
func DigestsClose() {
	if client != nil {
		client.Close()
		client = nil
	}
}

//export DigestsParseFeed
func DigestsParseFeed(url *C.char) *C.char {
	if client == nil {
		return C.CString(`{"error": "client not initialized"}`)
	}
	
	feedURL := C.GoString(url)
	feed, err := client.ParseFeed(context.Background(), feedURL)
	if err != nil {
		errResp := map[string]string{"error": err.Error()}
		data, _ := json.Marshal(errResp)
		return C.CString(string(data))
	}
	
	data, err := json.Marshal(feed)
	if err != nil {
		errResp := map[string]string{"error": "failed to marshal response"}
		data, _ := json.Marshal(errResp)
		return C.CString(string(data))
	}
	
	return C.CString(string(data))
}

//export DigestsParseFeeds
func DigestsParseFeeds(urlsJson *C.char) *C.char {
	if client == nil {
		return C.CString(`{"error": "client not initialized"}`)
	}
	
	urlsStr := C.GoString(urlsJson)
	var urls []string
	if err := json.Unmarshal([]byte(urlsStr), &urls); err != nil {
		errResp := map[string]string{"error": "invalid JSON input"}
		data, _ := json.Marshal(errResp)
		return C.CString(string(data))
	}
	
	feeds, err := client.ParseFeeds(context.Background(), urls)
	if err != nil {
		errResp := map[string]string{"error": err.Error()}
		data, _ := json.Marshal(errResp)
		return C.CString(string(data))
	}
	
	data, err := json.Marshal(feeds)
	if err != nil {
		errResp := map[string]string{"error": "failed to marshal response"}
		data, _ := json.Marshal(errResp)
		return C.CString(string(data))
	}
	
	return C.CString(string(data))
}

//export DigestsParseFeedWithoutEnrichment
func DigestsParseFeedWithoutEnrichment(url *C.char) *C.char {
	if client == nil {
		return C.CString(`{"error": "client not initialized"}`)
	}
	
	feedURL := C.GoString(url)
	feed, err := client.ParseFeed(
		context.Background(), 
		feedURL,
		digests.WithoutEnrichment(),
	)
	if err != nil {
		errResp := map[string]string{"error": err.Error()}
		data, _ := json.Marshal(errResp)
		return C.CString(string(data))
	}
	
	data, err := json.Marshal(feed)
	if err != nil {
		errResp := map[string]string{"error": "failed to marshal response"}
		data, _ := json.Marshal(errResp)
		return C.CString(string(data))
	}
	
	return C.CString(string(data))
}

//export DigestsSearch
func DigestsSearch(query *C.char) *C.char {
	if client == nil {
		return C.CString(`{"error": "client not initialized"}`)
	}
	
	searchQuery := C.GoString(query)
	results, err := client.Search(context.Background(), searchQuery)
	if err != nil {
		errResp := map[string]string{"error": err.Error()}
		data, _ := json.Marshal(errResp)
		return C.CString(string(data))
	}
	
	data, err := json.Marshal(results)
	if err != nil {
		errResp := map[string]string{"error": "failed to marshal response"}
		data, _ := json.Marshal(errResp)
		return C.CString(string(data))
	}
	
	return C.CString(string(data))
}

//export DigestsFreeString
func DigestsFreeString(str *C.char) {
	C.free(unsafe.Pointer(str))
}

// Required for building as shared library
func main() {}