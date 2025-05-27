import json
import sys
from datetime import datetime

def load_json_file(filepath):
    with open(filepath, 'r') as f:
        return json.load(f)

def compare_field_types(val1, val2, path=""):
    """Compare types and values of two fields"""
    type1 = type(val1).__name__
    type2 = type(val2).__name__
    
    if type1 != type2:
        return f"Type mismatch at {path}: {type1} vs {type2}"
    return None

def compare_objects(obj1, obj2, path=""):
    """Recursively compare two objects"""
    differences = []
    
    # Get all keys from both objects
    keys1 = set(obj1.keys()) if isinstance(obj1, dict) else set()
    keys2 = set(obj2.keys()) if isinstance(obj2, dict) else set()
    
    # Find missing keys
    only_in_current = keys1 - keys2
    only_in_new = keys2 - keys1
    
    if only_in_current:
        differences.append(f"Fields only in current API at {path}: {sorted(only_in_current)}")
    if only_in_new:
        differences.append(f"Fields only in new API at {path}: {sorted(only_in_new)}")
    
    # Compare common keys
    for key in keys1 & keys2:
        val1 = obj1[key]
        val2 = obj2[key]
        new_path = f"{path}.{key}" if path else key
        
        # Skip timestamp comparisons
        if key in ['lastRefreshed', 'lastUpdated', 'published', 'created']:
            continue
            
        # Check type differences
        type_diff = compare_field_types(val1, val2, new_path)
        if type_diff:
            differences.append(type_diff)
            continue
        
        # Compare values based on type
        if isinstance(val1, dict) and isinstance(val2, dict):
            differences.extend(compare_objects(val1, val2, new_path))
        elif isinstance(val1, list) and isinstance(val2, list):
            if len(val1) != len(val2):
                differences.append(f"Array length mismatch at {new_path}: {len(val1)} vs {len(val2)}")
            # Compare first item structure if exists
            if val1 and val2 and isinstance(val1[0], dict) and isinstance(val2[0], dict):
                differences.extend(compare_objects(val1[0], val2[0], f"{new_path}[0]"))
        elif val1 != val2 and key not in ['guid', 'id', 'description', 'content', 'title']:
            # Log value differences for non-content fields
            differences.append(f"Value difference at {new_path}: '{val1}' vs '{val2}'")
    
    return differences

# Load responses
try:
    current_api = load_json_file('/tmp/verge_current_api.json')
    new_api = load_json_file('/tmp/verge_new_api.json')
    
    print("=== API Response Comparison ===\n")
    
    # Check top-level structure
    print("Top-level keys:")
    print(f"Current API: {sorted(current_api.keys())}")
    print(f"New API: {sorted(new_api.keys())}")
    print()
    
    # Check feeds
    if 'feeds' in current_api and 'feeds' in new_api:
        current_feeds = current_api['feeds']
        new_feeds = new_api['feeds']
        
        print(f"Number of feeds:")
        print(f"Current API: {len(current_feeds)}")
        print(f"New API: {len(new_feeds)}")
        print()
        
        if current_feeds and new_feeds:
            # Compare first feed structure
            print("First feed comparison:")
            feed_diffs = compare_objects(current_feeds[0], new_feeds[0], "feeds[0]")
            
            if feed_diffs:
                print("\nDifferences found:")
                for diff in feed_diffs:
                    print(f"  - {diff}")
            else:
                print("  No structural differences found!")
            
            # Sample some actual values
            print("\nSample values comparison:")
            for key in ['type', 'status', 'feedTitle', 'feedUrl', 'language']:
                if key in current_feeds[0] and key in new_feeds[0]:
                    curr_val = current_feeds[0].get(key, 'MISSING')
                    new_val = new_feeds[0].get(key, 'MISSING')
                    if curr_val != new_val:
                        print(f"  {key}: '{curr_val}' → '{new_val}'")
                elif key in current_feeds[0]:
                    print(f"  {key}: '{current_feeds[0][key]}' → MISSING")
                elif key in new_feeds[0]:
                    print(f"  {key}: MISSING → '{new_feeds[0][key]}'")
            
            # Check items
            if 'items' in current_feeds[0] and 'items' in new_feeds[0]:
                curr_items = current_feeds[0]['items']
                new_items = new_feeds[0]['items']
                
                print(f"\nNumber of items:")
                print(f"  Current API: {len(curr_items)}")
                print(f"  New API: {len(new_items)}")
                
                if curr_items and new_items:
                    # Check item fields
                    print("\nFirst item fields:")
                    print(f"  Current API: {sorted(curr_items[0].keys())}")
                    print(f"  New API: {sorted(new_items[0].keys())}")
                    
                    # Check for specific fields
                    print("\nChecking specific item fields:")
                    check_fields = ['enclosures', 'thumbnail', 'duration', 'categories', 'content_encoded']
                    for field in check_fields:
                        curr_has = field in curr_items[0]
                        new_has = field in new_items[0]
                        if curr_has != new_has:
                            print(f"  {field}: Current has={curr_has}, New has={new_has}")
                        elif curr_has and new_has:
                            curr_val = curr_items[0][field]
                            new_val = new_items[0][field]
                            if type(curr_val) != type(new_val):
                                print(f"  {field}: Type mismatch - {type(curr_val).__name__} vs {type(new_val).__name__}")

    # Save formatted samples for manual inspection
    with open('/home/bumpyclock/Projects/digests-api/src/current_api_sample.json', 'w') as f:
        json.dump(current_api, f, indent=2)
    
    with open('/home/bumpyclock/Projects/digests-api/src/new_api_sample.json', 'w') as f:
        json.dump(new_api, f, indent=2)
        
    print("\n\nFull responses saved for inspection:")
    print("  - current_api_sample.json")
    print("  - new_api_sample.json")

except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()