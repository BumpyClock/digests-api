# Reader View API Client Usage Guide

## Understanding the JSON Response

When you call the `/getreaderview` endpoint, the API returns JSON with escaped newlines (`\n`). This is **normal JSON behavior** - all newlines in JSON strings are escaped.

## Correct Client Implementation

### JavaScript/TypeScript Example
```javascript
// Fetch the reader view
const response = await fetch('/getreaderview', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ urls: ['https://example.com/article'] })
});

// Parse the JSON response
const data = await response.json();

// The markdown field now contains ACTUAL newlines, not \n
const markdown = data[0].markdown;

// Render in your markdown viewer
markdownViewer.render(markdown); // This will show proper line breaks
```

### Python Example
```python
import requests
import json

# Make the request
response = requests.post('/getreaderview', 
    json={'urls': ['https://example.com/article']})

# Parse JSON - this converts \n to actual newlines
data = response.json()

# The markdown has real newlines now
markdown = data[0]['markdown']
print(markdown)  # This will print with proper line breaks
```

### Common Mistakes

❌ **Wrong**: Displaying the raw JSON string
```javascript
// This will show \n because it's showing the JSON string
console.log(JSON.stringify(data[0].markdown));
```

✅ **Correct**: Using the parsed value
```javascript
// This will have proper newlines
console.log(data[0].markdown);
```

## Testing in Development

If you're testing with curl or similar tools, you'll see `\n` in the output:
```bash
curl -X POST http://localhost:8080/getreaderview \
  -H "Content-Type: application/json" \
  -d '{"urls":["https://example.com"]}'
```

To see the formatted output, pipe through a JSON parser:
```bash
curl -X POST http://localhost:8080/getreaderview \
  -H "Content-Type: application/json" \
  -d '{"urls":["https://example.com"]}' | \
  jq -r '.[0].markdown'
```

The `-r` flag in `jq` outputs raw strings with proper newlines.