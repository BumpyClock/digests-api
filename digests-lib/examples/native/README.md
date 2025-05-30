# Using Digests Library in Native Applications

This guide shows how to use the Digests library as a shared library (DLL/dylib/so) in native applications.

## Building the Shared Library

1. Navigate to the C API directory:
```bash
cd digests-lib/capi
```

2. Run the build script:
```bash
./build.sh
```

This will create:
- **Windows**: `../lib/digests.dll`
- **macOS**: `../lib/digests.dylib` (universal binary)
- **Linux**: `../lib/digests.so`

## Windows (WinUI 3 / .NET)

### Setup

1. Copy `digests.dll` to your project's output directory
2. Add the `DigestsClient.cs` file to your project
3. Install required NuGet packages:
   ```xml
   <PackageReference Include="CommunityToolkit.Mvvm" Version="8.2.2" />
   ```

### Usage

```csharp
using Digests;

// Initialize with default settings
using var client = new DigestsClient();

// Or with SQLite cache
using var client = new DigestsClient("sqlite", "feeds_cache.db");

// Parse a feed asynchronously
var feed = await client.ParseFeedAsync("https://example.com/feed.xml");

// Parse without enrichment (faster)
var feed = await client.ParseFeedWithoutEnrichmentAsync("https://example.com/feed.xml");

// Parse multiple feeds
var feeds = await client.ParseFeedsAsync(new[] {
    "https://feed1.com/rss",
    "https://feed2.com/rss"
});

// Search for feeds
var results = await client.SearchAsync("technology news");
```

### WinUI 3 Integration

See `FeedViewModel.cs` and `MainPage.xaml` for a complete example of MVVM integration with data binding.

### Deployment

Include these files with your application:
- `digests.dll` - The main library
- `feeds_cache.db` - Will be created if using SQLite cache

## macOS (Swift/SwiftUI)

### Setup

1. Add `digests.dylib` to your Xcode project
2. Set up the library search path in Build Settings
3. Enable "Hardened Runtime" with "Disable Library Validation" if needed

### Usage

```swift
// Initialize the client
guard let client = DigestsClient() else {
    print("Failed to initialize Digests")
    return
}

// Parse a feed
if let feed = client.parseFeed(url: "https://example.com/feed.xml") {
    print("Feed: \(feed["title"] ?? "")")
}
```

### SwiftUI Integration

```swift
@MainActor
class FeedStore: ObservableObject {
    @Published var feeds: [Feed] = []
    private let client = DigestsClient()
    
    func loadFeed(url: String) async {
        // Load feed in background
        let feed = await Task.detached {
            self.client?.parseFeed(url: url)
        }.value
        
        // Update UI on main thread
        if let feed = feed {
            self.feeds.append(feed)
        }
    }
}
```

## Linux (C/C++)

### Usage

```c
#include "digests.h"
#include <stdio.h>
#include <stdlib.h>

int main() {
    // Initialize
    if (DigestsInit() != 0) {
        fprintf(stderr, "Failed to initialize\n");
        return 1;
    }
    
    // Parse feed
    char* result = DigestsParseFeed("https://example.com/feed.xml");
    printf("Result: %s\n", result);
    
    // Clean up
    DigestsFreeString(result);
    DigestsClose();
    
    return 0;
}
```

### Compilation

```bash
gcc -o feedreader main.c -L. -ldigests -Wl,-rpath,'$ORIGIN'
```

## Platform-Specific Notes

### Windows
- The DLL uses the C calling convention (`__cdecl`)
- Ensure Visual C++ Redistributables are installed
- For .NET Framework apps, use `DllImport` with full path if needed

### macOS
- The library is built as a universal binary (Intel + Apple Silicon)
- You may need to sign the dylib for distribution
- Use `@rpath` for flexible deployment

### Linux
- Set `LD_LIBRARY_PATH` or use rpath for library loading
- The library is built with CGO enabled

## Error Handling

All functions return JSON. Check for an "error" field:

```json
{
  "error": "Failed to parse feed: invalid URL"
}
```

In C#:
```csharp
try {
    var feed = await client.ParseFeedAsync(url);
} catch (Exception ex) {
    Console.WriteLine($"Error: {ex.Message}");
}
```

## Performance Tips

1. **Use SQLite cache** for better performance across app restarts
2. **Disable enrichment** when you only need basic feed data
3. **Parse feeds concurrently** using `ParseFeedsAsync`
4. **Reuse the client instance** - don't create/destroy for each operation

## Memory Management

- **C#/.NET**: The wrapper handles memory automatically
- **Swift**: Use the provided wrapper that manages memory
- **C/C++**: Always call `DigestsFreeString` on returned strings

## Troubleshooting

### DLL Not Found
- Ensure the DLL is in the same directory as your executable
- Check architecture compatibility (x64 vs x86)
- On Windows, use Dependency Walker to check for missing dependencies

### Initialization Fails
- Check if another instance is already running
- Verify write permissions for cache files
- Check system resources (memory, file handles)

### Parsing Errors
- Verify the URL is accessible
- Check network connectivity
- Enable logging by setting environment variables