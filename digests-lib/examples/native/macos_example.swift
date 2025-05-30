// Example: Using Digests dylib in a macOS Swift application

import Foundation

// Define the C function signatures
typealias DigestsInitFunc = @convention(c) () -> Int32
typealias DigestsCloseFunc = @convention(c) () -> Void
typealias DigestsParseFeedFunc = @convention(c) (UnsafePointer<CChar>) -> UnsafeMutablePointer<CChar>
typealias DigestsFreeStringFunc = @convention(c) (UnsafeMutablePointer<CChar>) -> Void

class DigestsClient {
    private let handle: UnsafeMutableRawPointer
    private let digestsInit: DigestsInitFunc
    private let digestsClose: DigestsCloseFunc
    private let digestsParseFeed: DigestsParseFeedFunc
    private let digestsFreeString: DigestsFreeStringFunc
    
    init?() {
        // Load the dynamic library
        guard let handle = dlopen("@rpath/digests.dylib", RTLD_NOW) else {
            print("Failed to load digests.dylib: \(String(cString: dlerror()))")
            return nil
        }
        self.handle = handle
        
        // Get function pointers
        guard let initPtr = dlsym(handle, "DigestsInit"),
              let closePtr = dlsym(handle, "DigestsClose"),
              let parsePtr = dlsym(handle, "DigestsParseFeed"),
              let freePtr = dlsym(handle, "DigestsFreeString") else {
            print("Failed to load functions")
            dlclose(handle)
            return nil
        }
        
        self.digestsInit = unsafeBitCast(initPtr, to: DigestsInitFunc.self)
        self.digestsClose = unsafeBitCast(closePtr, to: DigestsCloseFunc.self)
        self.digestsParseFeed = unsafeBitCast(parsePtr, to: DigestsParseFeedFunc.self)
        self.digestsFreeString = unsafeBitCast(freePtr, to: DigestsFreeStringFunc.self)
        
        // Initialize the library
        if digestsInit() != 0 {
            print("Failed to initialize Digests")
            dlclose(handle)
            return nil
        }
    }
    
    deinit {
        digestsClose()
        dlclose(handle)
    }
    
    func parseFeed(url: String) -> [String: Any]? {
        let cResult = digestsParseFeed(url)
        defer { digestsFreeString(cResult) }
        
        let jsonString = String(cString: cResult)
        guard let data = jsonString.data(using: .utf8),
              let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            return nil
        }
        
        return json
    }
}

// Usage example
if let client = DigestsClient() {
    if let feed = client.parseFeed(url: "https://daringfireball.net/feeds/main") {
        print("Feed title: \(feed["title"] ?? "Unknown")")
        
        if let items = feed["items"] as? [[String: Any]] {
            print("Number of items: \(items.count)")
            
            for (index, item) in items.prefix(3).enumerated() {
                print("\n\(index + 1). \(item["title"] ?? "No title")")
            }
        }
    }
}

// For a SwiftUI app, you might wrap this in an ObservableObject:
import SwiftUI

@MainActor
class FeedViewModel: ObservableObject {
    @Published var feeds: [[String: Any]] = []
    @Published var isLoading = false
    @Published var error: String?
    
    private let client = DigestsClient()
    
    func loadFeed(url: String) async {
        isLoading = true
        error = nil
        
        await Task.detached {
            if let feed = self.client?.parseFeed(url: url) {
                await MainActor.run {
                    self.feeds = [feed]
                    self.isLoading = false
                }
            } else {
                await MainActor.run {
                    self.error = "Failed to load feed"
                    self.isLoading = false
                }
            }
        }.value
    }
}