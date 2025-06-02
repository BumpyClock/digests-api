# GitHub Actions Workflows

This directory contains the CI/CD workflows for the Digests API project.

## Workflows

### 1. Build Check (`go.yml`)
- **Trigger**: Push to main, Pull requests to main
- **Purpose**: Quick build verification for Linux platforms
- **What it does**:
  - Builds for Linux AMD64 and ARM64
  - Runs tests on AMD64
  - Ensures code compiles correctly

### 2. Build All Platforms (`build-all.yml`)
- **Trigger**: Manual workflow dispatch, Pull requests affecting source code
- **Purpose**: Comprehensive build testing
- **What it does**:
  - Builds Linux binaries with CGO support (AMD64, ARM64)
  - Quick build check for Windows and macOS (without CGO)
  - Runs full test suite with race detection
  - Generates coverage reports

### 3. Release Binaries (`release.yml`)
- **Trigger**: Git tags starting with 'v*', Manual workflow dispatch
- **Purpose**: Create official releases with binaries for all platforms
- **What it does**:
  - Builds Linux binaries (AMD64, ARM64) with CGO
  - Builds Windows binaries (AMD64 with CGO, ARM64 without CGO)
  - Builds macOS binaries (AMD64, ARM64) with CGO
  - Creates GitHub release with all binaries
  - Generates SHA256 checksums

## Notes

### CGO Requirements
The SQLite cache implementation requires CGO for the `github.com/mattn/go-sqlite3` driver. This affects cross-compilation:

- **Linux**: Full CGO support with cross-compilation tools
- **Windows AMD64**: CGO support via MinGW
- **Windows ARM64**: Built without CGO (SQLite features disabled)
- **macOS**: Requires macOS runner for CGO support

### Build Flags
All release binaries are built with:
- `-ldflags="-s -w"`: Strip debug information for smaller binaries
- `CGO_ENABLED=1`: Enable CGO where supported

### Manual Triggering
Both `build-all.yml` and `release.yml` can be manually triggered from the Actions tab in GitHub.