// Example: Using Digests DLL in a Windows C++ application
#include <windows.h>
#include <iostream>
#include <string>
#include "digests.h"

// Function pointer types
typedef int (*DigestsInitFunc)();
typedef void (*DigestsCloseFunc)();
typedef char* (*DigestsParseFeedFunc)(const char*);
typedef void (*DigestsFreeStringFunc)(char*);

int main() {
    // Load the DLL
    HMODULE digestsDll = LoadLibrary(TEXT("digests.dll"));
    if (!digestsDll) {
        std::cerr << "Failed to load digests.dll" << std::endl;
        return 1;
    }

    // Get function pointers
    auto DigestsInit = (DigestsInitFunc)GetProcAddress(digestsDll, "DigestsInit");
    auto DigestsClose = (DigestsCloseFunc)GetProcAddress(digestsDll, "DigestsClose");
    auto DigestsParseFeed = (DigestsParseFeedFunc)GetProcAddress(digestsDll, "DigestsParseFeed");
    auto DigestsFreeString = (DigestsFreeStringFunc)GetProcAddress(digestsDll, "DigestsFreeString");

    if (!DigestsInit || !DigestsClose || !DigestsParseFeed || !DigestsFreeString) {
        std::cerr << "Failed to get function pointers" << std::endl;
        FreeLibrary(digestsDll);
        return 1;
    }

    // Initialize the library
    if (DigestsInit() != 0) {
        std::cerr << "Failed to initialize Digests" << std::endl;
        FreeLibrary(digestsDll);
        return 1;
    }

    // Parse a feed
    const char* feedUrl = "https://news.ycombinator.com/rss";
    char* result = DigestsParseFeed(feedUrl);
    
    std::cout << "Feed data:" << std::endl;
    std::cout << result << std::endl;
    
    // Clean up
    DigestsFreeString(result);
    DigestsClose();
    FreeLibrary(digestsDll);

    return 0;
}

// Compile with:
// cl.exe windows_example.cpp