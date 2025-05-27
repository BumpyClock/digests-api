import json
import sys

# Read the response file
with open('/tmp/current_api_response.json', 'rb') as f:
    # Read in chunks to handle potential issues
    content = f.read()
    
# Try to find a complete feed object
try:
    # Parse the full JSON
    data = json.loads(content)
    
    print("=== FULL STRUCTURE ANALYSIS ===\n")
    
    # Analyze top level
    print(f"Top level keys: {list(data.keys())}")
    print(f"Number of feeds: {len(data.get('feeds', []))}")
    
    if data.get('feeds'):
        # Analyze first feed
        feed = data['feeds'][0]
        print(f"\nFirst feed keys ({len(feed)} fields):")
        for key in sorted(feed.keys()):
            value_type = type(feed[key]).__name__
            sample = str(feed[key])[:50] + "..." if len(str(feed[key])) > 50 else str(feed[key])
            print(f"  - {key}: {value_type} = {sample}")
        
        # Analyze author structure if present
        if feed.get('author'):
            print(f"\nAuthor structure:")
            for key, value in feed['author'].items():
                print(f"  - {key}: {value}")
        
        # Analyze first item if present
        if feed.get('items') and len(feed['items']) > 0:
            item = feed['items'][0]
            print(f"\nFirst item keys ({len(item)} fields):")
            for key in sorted(item.keys()):
                value_type = type(item[key]).__name__
                sample = str(item[key])[:50] + "..." if len(str(item[key])) > 50 else str(item[key])
                print(f"  - {key}: {value_type} = {sample}")
        
        # Check second feed for different structure (if exists)
        if len(data['feeds']) > 1:
            feed2 = data['feeds'][1]
            print(f"\nSecond feed keys ({len(feed2)} fields):")
            feed2_keys = set(feed2.keys())
            feed1_keys = set(feed.keys())
            
            if feed2_keys != feed1_keys:
                print("  Different fields in second feed:")
                print(f"  - Only in feed 2: {feed2_keys - feed1_keys}")
                print(f"  - Only in feed 1: {feed1_keys - feed2_keys}")
            else:
                print("  (Same structure as first feed)")
                
    # Save a clean sample for inspection
    sample_data = {
        "feeds": data.get("feeds", [])[:1]  # Just first feed
    }
    
    with open('/home/bumpyclock/Projects/digests-api/src/current_api_sample.json', 'w') as f:
        json.dump(sample_data, f, indent=2)
        print("\nSample saved to current_api_sample.json")
        
except json.JSONDecodeError as e:
    print(f"JSON decode error: {e}")
    print(f"Error position: {e.pos}")
    
    # Try to extract a portion that might be valid
    try:
        # Find the first complete feed object
        start = content.find(b'{"type"')
        end = content.find(b'}],"', start) + 2
        
        if start > 0 and end > start:
            partial = b'{"feeds":[' + content[start:end] + b']}'
            data = json.loads(partial)
            
            print("\n=== PARTIAL ANALYSIS (First Feed Only) ===")
            feed = data['feeds'][0]
            print(f"\nFeed keys: {list(feed.keys())}")
            
    except:
        print("Could not extract partial data")