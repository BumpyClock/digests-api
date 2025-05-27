# API Compatibility with v1

The new API implementation maintains backward compatibility with the existing API at https://api.digests.app/parse.

## Endpoint Mapping

| v1 Endpoint | New Endpoint | Method | Description |
|------------|--------------|--------|-------------|
| `/parse` | `/parse` | POST | Parse multiple RSS/Atom feeds |
| `/feed` | `/feed` | GET | Parse single feed (additional endpoint) |

## Request Format

The request format remains the same:

```json
{
    "urls": [
        "https://feeds.megaphone.fm/VMP1684715893",
        "https://rss.wbur.org/circle-round-club/podcast"
    ]
}
```

## Response Format

The response maintains the v1 structure:

```json
{
    "feeds": [
        {
            "type": "podcast",
            "guid": "fb64054dde2c5f52384b5f5caab567b26b8faa8980ef04edfaebe280f17fecd3",
            "status": "ok",
            "siteTitle": "CRC",
            "feedTitle": "CRC",
            "feedUrl": "https://rss.wbur.org/circle-round-club/podcast",
            "description": "Thank you for Circling Round with us!",
            "link": "www.wbur.org",
            "lastUpdated": "2025-05-26T22:30:21-04:00",
            "lastRefreshed": "2025-05-27T02:41:02Z",
            "published": "",
            "author": {
                "name": "WBUR"
            },
            "language": "en-us",
            "favicon": "https://wordpress.wbur.org/wp-content/uploads/2023/04/circle-round-club.jpeg",
            "categories": "Kids & Family",
            "items": [
                {
                    "type": "podcast",
                    "id": "7516e2be-1f95-4671-aa7d-6e5a6f0c9c69",
                    "title": "Granny Snowstorm",
                    "description": "Episode description...",
                    "link": "https://www.wbur.org/circle-round-club/2024/07/30/granny-snowstorm-crc",
                    "author": "WBUR",
                    "published": "2024-07-30T15:00:00-04:00",
                    "content": "...",
                    "created": "...",
                    "content_encoded": "...",
                    "categories": ["Kids & Family"],
                    "url": "https://media.url",
                    "length": "12345678",
                    "type": "audio/mpeg"
                }
            ]
        }
    ]
}
```

## Field Mappings

### Feed Level

| v1 Field | Internal Field | Notes |
|----------|---------------|--------|
| `guid` | Generated from URL | SHA256 hash of feed URL |
| `status` | Always "ok" | Error feeds excluded from results |
| `feedTitle` | `Title` | Direct mapping |
| `feedUrl` | `URL` | Direct mapping |
| `description` | `Description` | Direct mapping |
| `link` | `URL` | Uses feed URL |
| `lastUpdated` | `LastUpdated` | RFC3339 format |
| `lastRefreshed` | Current time | When request was made |
| `type` | Detected | "rss" or "podcast" |

### Item Level

| v1 Field | Internal Field | Notes |
|----------|---------------|--------|
| `id` | `ID` | UUID format |
| `title` | `Title` | Direct mapping |
| `description` | `Description` | Direct mapping |
| `link` | `Link` | Direct mapping |
| `author` | `Author` | Direct mapping |
| `published` | `Published` | RFC3339 format |

## Migration Guide

For clients currently using the v1 API:

1. **No changes required** - The new API maintains the same endpoint and response format
2. **Optional enhancements available**:
   - Rate limiting headers provide usage information
   - OpenAPI documentation at `/openapi.json`
   - Swagger UI at `/docs`
   - Additional `/feed` endpoint for single feed parsing

## Differences from v1

While maintaining compatibility, the new implementation adds:

1. **Better error handling** - Structured error responses following RFC 7807
2. **Request validation** - Automatic validation of input
3. **Rate limiting** - 100 requests/minute per IP with headers
4. **Caching** - 1-hour cache for parsed feeds
5. **Concurrent processing** - Up to 10 feeds parsed simultaneously
6. **OpenAPI documentation** - Auto-generated from code

## Testing Compatibility

To verify compatibility:

```bash
# Test with current API
curl -X POST https://api.digests.app/parse \
  -H "Content-Type: application/json" \
  -d '{"urls": ["https://example.com/feed.rss"]}' \
  > v1_response.json

# Test with new API
curl -X POST http://localhost:8080/parse \
  -H "Content-Type: application/json" \
  -d '{"urls": ["https://example.com/feed.rss"]}' \
  > new_response.json

# Compare structure (ignoring dynamic fields like timestamps)
jq 'del(.feeds[].lastRefreshed)' v1_response.json > v1_normalized.json
jq 'del(.feeds[].lastRefreshed)' new_response.json > new_normalized.json
diff v1_normalized.json new_normalized.json
```