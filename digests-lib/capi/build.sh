#!/bin/bash
# Build script for shared libraries on different platforms

set -e

echo "Building Digests shared library..."

# Create output directory
mkdir -p ../lib

# Detect platform
OS=$(uname -s)

case "$OS" in
    Darwin*)
        echo "Building for macOS..."
        # Build for macOS (universal binary for Intel and Apple Silicon)
        GOOS=darwin GOARCH=amd64 go build -buildmode=c-shared -o ../lib/digests_amd64.dylib digests_capi.go
        GOOS=darwin GOARCH=arm64 go build -buildmode=c-shared -o ../lib/digests_arm64.dylib digests_capi.go
        
        # Create universal binary
        lipo -create ../lib/digests_amd64.dylib ../lib/digests_arm64.dylib -output ../lib/digests.dylib
        rm ../lib/digests_amd64.dylib ../lib/digests_arm64.dylib ../lib/digests_amd64.h ../lib/digests_arm64.h
        
        # Copy header
        mv ../lib/digests.h ../lib/digests_c.h
        cp digests.h ../lib/
        
        echo "Built: ../lib/digests.dylib (universal binary)"
        ;;
        
    Linux*)
        echo "Building for Linux..."
        go build -buildmode=c-shared -o ../lib/digests.so digests_capi.go
        mv ../lib/digests.h ../lib/digests_c.h
        cp digests.h ../lib/
        echo "Built: ../lib/digests.so"
        ;;
        
    MINGW* | MSYS* | CYGWIN*)
        echo "Building for Windows..."
        go build -buildmode=c-shared -o ../lib/digests.dll digests_capi.go
        mv ../lib/digests.h ../lib/digests_c.h
        cp digests.h ../lib/
        echo "Built: ../lib/digests.dll"
        ;;
        
    *)
        echo "Unsupported platform: $OS"
        exit 1
        ;;
esac

echo "Build complete!"
echo "Header file: ../lib/digests.h"
echo "You can now use the library in your native applications."