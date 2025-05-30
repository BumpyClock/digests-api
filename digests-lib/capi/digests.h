// ABOUTME: C header file for the Digests library
// ABOUTME: Defines the C API for native application integration

#ifndef DIGESTS_H
#define DIGESTS_H

#ifdef __cplusplus
extern "C" {
#endif

// Initialize the digests client with default settings
// Returns 0 on success, -1 on error
int DigestsInit(void);

// Initialize the digests client with cache configuration
// cacheType: "memory" or "sqlite"
// cachePath: path to SQLite database (ignored for memory cache)
// Returns 0 on success, -1 on error
int DigestsInitWithCache(const char* cacheType, const char* cachePath);

// Close the digests client and clean up resources
void DigestsClose(void);

// Parse a single feed
// url: RSS/Atom feed URL
// Returns: JSON string with feed data or error
// Note: Caller must free the returned string using DigestsFreeString
char* DigestsParseFeed(const char* url);

// Parse multiple feeds
// urlsJson: JSON array of feed URLs (e.g., ["url1", "url2"])
// Returns: JSON string with feeds array or error
// Note: Caller must free the returned string using DigestsFreeString
char* DigestsParseFeeds(const char* urlsJson);

// Parse a feed without enrichment (faster)
// url: RSS/Atom feed URL
// Returns: JSON string with feed data or error
// Note: Caller must free the returned string using DigestsFreeString
char* DigestsParseFeedWithoutEnrichment(const char* url);

// Search for RSS feeds
// query: Search query string
// Returns: JSON string with search results or error
// Note: Caller must free the returned string using DigestsFreeString
char* DigestsSearch(const char* query);

// Free a string returned by the library
void DigestsFreeString(char* str);

#ifdef __cplusplus
}
#endif

#endif // DIGESTS_H