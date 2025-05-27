# Digests API Documentation

## Overview

The Digests API is a RESTful service for parsing and managing RSS/Atom feeds. It provides endpoints for fetching feed content, searching for feeds, and sharing feed collections.

**Base URL**: `https://api.digests.app` (production) or `http://localhost:8080` (local)

**API Version**: 1.0.0

## Authentication

Currently, the API does not require authentication. This may change in future versions.

## Rate Limiting

The API implements rate limiting to ensure fair usage:
- **Rate Limit**: 100 requests per minute per IP address
- **Headers**: Rate limit information is included in response headers:
  - `X-RateLimit-Limit`: Maximum requests allowed
  - `X-RateLimit-Window`: Time window for rate limit
  - `Retry-After`: Seconds to wait before retrying (only on 429 responses)

## Common Response Formats

### Success Response

```json
{
  "data": {...},
  "meta": {
    "request_id": "uuid-string"
  }
}
```

### Error Response

```json
{
  "error": {
    "status": 400,
    "title": "Bad Request",
    "detail": "Detailed error message",
    "instance": "/feeds"
  }
}
```

## Endpoints

### 1. Parse Multiple Feeds

Parse multiple RSS/Atom feeds in a single request.

**Endpoint**: `POST /feeds`

**Request Body**:
```json
{
  "urls": [
    "https://example.com/feed1.rss",
    "https://example.com/feed2.atom"
  ],
  "page": 1,
  "items_per_page": 50
}
```

**Parameters**:
- `urls` (required): Array of feed URLs to parse (min: 1, max: 100)
- `page` (optional): Page number for pagination (default: 1, min: 1)
- `items_per_page` (optional): Number of items per page (default: 50, min: 1, max: 100)

**Response** (200 OK):
```json
{
  "feeds": [
    {
      "id": "feed-uuid",
      "title": "Example Blog",
      "description": "A blog about examples",
      "url": "https://example.com/feed1.rss",
      "items": [
        {
          "id": "item-uuid",
          "title": "Blog Post Title",
          "description": "Post summary or full content",
          "link": "https://example.com/post1",
          "author": "John Doe",
          "published": "2024-01-15T10:30:00Z"
        }
      ],
      "last_updated": "2024-01-15T12:00:00Z"
    }
  ],
  "total_feeds": 2,
  "page": 1,
  "per_page": 50
}
```

**Error Responses**:
- `400 Bad Request`: Invalid request body or URL format
- `500 Internal Server Error`: Server error during feed parsing

**Example cURL**:
```bash
curl -X POST https://api.digests.app/feeds \
  -H "Content-Type: application/json" \
  -d '{
    "urls": ["https://blog.golang.org/feed.atom"],
    "page": 1,
    "items_per_page": 20
  }'
```

### 2. Parse Single Feed

Parse a single RSS/Atom feed.

**Endpoint**: `GET /feed`

**Query Parameters**:
- `url` (required): Feed URL to parse
- `page` (optional): Page number for items (default: 1, min: 1)
- `items_per_page` (optional): Number of items per page (default: 50, min: 1, max: 100)

**Response** (200 OK):
```json
{
  "id": "feed-uuid",
  "title": "Example Blog",
  "description": "A blog about examples",
  "url": "https://example.com/feed.rss",
  "items": [
    {
      "id": "item-uuid",
      "title": "Blog Post Title",
      "description": "Post content",
      "link": "https://example.com/post1",
      "author": "Jane Doe",
      "published": "2024-01-15T10:30:00Z"
    }
  ],
  "last_updated": "2024-01-15T12:00:00Z"
}
```

**Error Responses**:
- `400 Bad Request`: Missing or invalid URL parameter
- `404 Not Found`: Feed not found or inaccessible
- `500 Internal Server Error`: Server error during feed parsing

**Example cURL**:
```bash
curl "https://api.digests.app/feed?url=https://blog.golang.org/feed.atom&page=1&items_per_page=10"
```

### 3. Search for RSS Feeds (Coming Soon)

Search for RSS feeds by keyword.

**Endpoint**: `GET /search`

**Query Parameters**:
- `q` (required): Search query (min: 2 chars, max: 100 chars)
- `page` (optional): Page number (default: 1)
- `per_page` (optional): Results per page (default: 20, max: 100)

**Response** (200 OK):
```json
{
  "results": [
    {
      "title": "Tech Blog",
      "description": "Latest technology news",
      "url": "https://techblog.com/feed.rss",
      "subscribers": 1500
    }
  ],
  "total": 45,
  "page": 1,
  "per_page": 20
}
```

### 4. Create Share Link (Coming Soon)

Create a shareable link for a collection of feeds.

**Endpoint**: `POST /shares`

**Request Body**:
```json
{
  "urls": [
    "https://example.com/feed1.rss",
    "https://example.com/feed2.rss"
  ],
  "expires_in": 86400
}
```

**Response** (201 Created):
```json
{
  "id": "share-uuid",
  "share_url": "https://api.digests.app/shares/share-uuid",
  "expires_at": "2024-01-16T12:00:00Z"
}
```

### 5. Get Shared Feeds (Coming Soon)

Retrieve feeds from a share link.

**Endpoint**: `GET /shares/{id}`

**Response** (200 OK):
```json
{
  "id": "share-uuid",
  "urls": [
    "https://example.com/feed1.rss",
    "https://example.com/feed2.rss"
  ],
  "created_at": "2024-01-15T12:00:00Z",
  "expires_at": "2024-01-16T12:00:00Z"
}
```

## OpenAPI Specification

The complete OpenAPI 3.0 specification is available at:
- JSON: `https://api.digests.app/openapi.json`
- Interactive Docs: `https://api.digests.app/docs`

## Error Codes Reference

| Status Code | Title | Description |
|------------|-------|-------------|
| 400 | Bad Request | Invalid request format or parameters |
| 404 | Not Found | Resource not found |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Server error |
| 503 | Service Unavailable | External service temporarily unavailable |

## SDK Examples

### Go Client

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

func main() {
    client := &http.Client{}
    
    reqBody := map[string]interface{}{
        "urls": []string{
            "https://blog.golang.org/feed.atom",
        },
        "page": 1,
        "items_per_page": 20,
    }
    
    body, _ := json.Marshal(reqBody)
    req, _ := http.NewRequest("POST", "https://api.digests.app/feeds", bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")
    
    resp, _ := client.Do(req)
    defer resp.Body.Close()
    
    // Handle response...
}
```

### Python Client

```python
import requests

response = requests.post(
    "https://api.digests.app/feeds",
    json={
        "urls": ["https://blog.golang.org/feed.atom"],
        "page": 1,
        "items_per_page": 20
    }
)

if response.status_code == 200:
    feeds = response.json()["feeds"]
    for feed in feeds:
        print(f"Feed: {feed['title']}")
```

### JavaScript/Node.js Client

```javascript
const axios = require('axios');

async function parseFeeds() {
    try {
        const response = await axios.post('https://api.digests.app/feeds', {
            urls: ['https://blog.golang.org/feed.atom'],
            page: 1,
            items_per_page: 20
        });
        
        const { feeds } = response.data;
        feeds.forEach(feed => {
            console.log(`Feed: ${feed.title}`);
        });
    } catch (error) {
        console.error('Error:', error.response?.data || error.message);
    }
}
```

## Best Practices

1. **Caching**: Feed responses are cached for 1 hour. Consider implementing client-side caching for better performance.

2. **Concurrent Requests**: When parsing multiple feeds, use the batch endpoint (`POST /feeds`) instead of making multiple individual requests.

3. **Error Handling**: Always check for both HTTP status codes and error response bodies.

4. **Pagination**: For feeds with many items, use pagination to reduce response size and improve performance.

5. **URL Validation**: Ensure feed URLs are properly formatted and accessible before sending requests.

## Changelog

### Version 1.0.0 (2024-01-15)
- Initial release
- Feed parsing endpoints
- Rate limiting
- OpenAPI documentation

## Support

For issues, feature requests, or questions:
- GitHub Issues: [https://github.com/yourorg/digests-api/issues](https://github.com/yourorg/digests-api/issues)
- Email: support@digests.app