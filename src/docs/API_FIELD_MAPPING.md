# Complete API Field Mapping

This document provides a comprehensive mapping of all fields in the current API response.

## Feed Object Fields

| Field | Type | Required | Description | Example |
|-------|------|----------|-------------|---------|
| `type` | string | No | Feed type (e.g., "podcast", "rss") | `"podcast"` |
| `guid` | string | Yes | Unique identifier (SHA256 of URL) | `"fb64054dde2c5f52384b5f5caab567b26b8faa8980ef04edfaebe280f17fecd3"` |
| `status` | string | Yes | Feed status | `"ok"` |
| `siteTitle` | string | No | Website title | `"CRC"` |
| `feedTitle` | string | Yes | Feed title | `"Circle Round Club"` |
| `feedUrl` | string | Yes | Feed URL | `"https://rss.wbur.org/circle-round-club/podcast"` |
| `description` | string | Yes | Feed description | `"Stories for kids"` |
| `link` | string | Yes | Website link | `"www.wbur.org"` |
| `lastUpdated` | string | Yes | Last update time (RFC3339) | `"2025-05-26T22:30:21-04:00"` |
| `lastRefreshed` | string | Yes | API refresh time (RFC3339) | `"2025-05-27T02:41:02Z"` |
| `published` | string | No | Publication date | `""` |
| `author` | object | No | Author information | `{"name": "WBUR"}` |
| `author.name` | string | No | Author name | `"WBUR"` |
| `author.email` | string | No | Author email | `""` |
| `language` | string | No | Feed language | `"en-us"` |
| `favicon` | string | No | Feed favicon URL | `"https://wordpress.wbur.org/wp-content/uploads/2023/04/circle-round-club.jpeg"` |
| `image` | string | No | Feed image URL | `"https://example.com/feed-image.jpg"` |
| `categories` | string | No | Feed categories | `"Kids & Family"` |
| `subtitle` | string | No | Feed subtitle (podcasts) | `"Stories for the whole family"` |
| `summary` | string | No | Feed summary (podcasts) | `"Award-winning stories"` |
| `items` | array | Yes | Feed items | `[...]` |

## Item Object Fields

| Field | Type | Required | Description | Example |
|-------|------|----------|-------------|---------|
| `type` | string | No | Item type (e.g., "podcast") | `"podcast"` |
| `id` | string | Yes | Unique item ID (UUID) | `"7516e2be-1f95-4671-aa7d-6e5a6f0c9c69"` |
| `title` | string | Yes | Item title | `"Granny Snowstorm"` |
| `description` | string | Yes | Item description | `"A winter tale..."` |
| `link` | string | Yes | Item link | `"https://www.wbur.org/circle-round-club/2024/07/30/granny-snowstorm-crc"` |
| `author` | string | No | Item author | `"WBUR"` |
| `published` | string | Yes | Publication date (RFC3339) | `"2024-07-30T15:00:00-04:00"` |
| `created` | string | No | Creation date (RFC3339) | `"2024-07-30T15:00:00-04:00"` |
| `content` | string | No | Plain text content | `"Episode content..."` |
| `content_encoded` | string | No | HTML content | `"<p>Episode content...</p>"` |
| `categories` | array | No | Item categories | `["Kids & Family", "Stories"]` |
| `duration` | string | No | Duration (HH:MM:SS) | `"00:28:19"` |
| `thumbnail` | string | No | Thumbnail image URL | `"https://wordpress.wbur.org/wp-content/uploads/2024/07/grannySnowstorm.jpg"` |
| `thumbnailColor` | object | No | Thumbnail dominant color | `{"r": 220, "g": 180, "b": 140}` |
| `thumbnailColorComputed` | string | No | Color computation status | `"set"` |
| `enclosures` | array | No | Media enclosures | See below |
| `episode` | number | No | Episode number | `3` |
| `season` | number | No | Season number | `8` |
| `episodeType` | string | No | Episode type | `"full"` |
| `subtitle` | string | No | Episode subtitle | `"A winter tale"` |
| `summary` | string | No | Episode summary | `"In this episode..."` |
| `image` | string | No | Episode image | `"https://example.com/episode.jpg"` |
| `url` | string | No | Media URL (legacy) | `"https://traffic.megaphone.fm/BUR9553652211.mp3"` |
| `length` | string | No | File size (legacy) | `"26827360"` |

## Enclosure Object Fields

| Field | Type | Required | Description | Example |
|-------|------|----------|-------------|---------|
| `url` | string | Yes | Media file URL | `"https://traffic.megaphone.fm/BUR9553652211.mp3"` |
| `length` | string | No | File size in bytes | `"26827360"` |
| `type` | string | No | MIME type | `"audio/mpeg"` |

## Color Object Fields

| Field | Type | Required | Description | Example |
|-------|------|----------|-------------|---------|
| `r` | number | Yes | Red value (0-255) | `220` |
| `g` | number | Yes | Green value (0-255) | `180` |
| `b` | number | Yes | Blue value (0-255) | `140` |

## Implementation Status

All fields listed above are implemented in the new API:
- ✅ All required fields are populated
- ✅ Optional fields are included when available
- ✅ Nested objects (author, enclosures, thumbnailColor) are supported
- ✅ Arrays (items, categories, enclosures) are properly handled
- ✅ Time formats match (RFC3339)
- ✅ GUID generation is consistent (SHA256 of URL)

## Notes

1. **Podcast Detection**: The `type` field is set based on feed content analysis
2. **Legacy Fields**: `url` and `length` at the item level are maintained for backward compatibility
3. **Enclosures**: Modern podcast feeds use the `enclosures` array instead of direct `url`/`length` fields
4. **Categories**: Can be either a string (feed level) or array (item level)
5. **Thumbnail Colors**: Computed by the original API, set to null in new implementation unless computed