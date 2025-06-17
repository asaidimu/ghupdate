# Integration Guide

## Environment Requirements

Go 1.24.4+ is required for compilation and runtime. Standard operating system environments (Linux, macOS, Windows) are supported. Write permissions are needed for the `DataDir` (temporary update files) and `ExecutablePath` (where the binary is located) to perform updates without elevated privileges. For `ExecutablePath`, this implies the application should be in a user-writable location.

## Initialization Patterns

### The primary initialization pattern involves calling `HandleUpdateMode()` first in your `main` function, followed by `CleanupUpdate()`, before any other application logic. This ensures correct handling of update-initiated restarts and a clean state.
```[DETECTED_LANGUAGE]
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/asaidimu/ghupdate"
)

var Version = "v1.0.0" // Injected at build time

func main() {
	// IMPORTANT: This must be the very first executable line.
	// If true, the application was just updated and is now the new version.
	if ghupdate.HandleUpdateMode() {
		fmt.Println("Successfully updated! Running new version.")
	}

	// Determine application data directory (e.g., for temporary update files).
	dataDir, err := os.UserCacheDir()
	if err != nil {
		log.Fatalf("Error getting cache dir: %v", err)
	}
	appDataDir := filepath.Join(dataDir, "my-app-name")
	if err := os.MkdirAll(appDataDir, 0755); err != nil {
		log.Fatalf("Error creating app data dir: %v", err)
	}

	// Clean up any old temporary update files.
	if err := ghupdate.CleanupUpdate(appDataDir); err != nil {
		log.Printf("Warning: Failed to clean up old update files: %v\n", err)
	}

	// Your application's main logic begins here.
	fmt.Printf("My Application - Version %s running from %s\n", Version, os.Args[0])
	// ... rest of your application ...
}
```

## Common Integration Pitfalls

- **Issue**: Not calling `ghupdate.HandleUpdateMode()` as the very first operation in `main()`.
  - **Solution**: This is critical. If `HandleUpdateMode()` is not called first, the newly spawned executable will not perform the file replacement, and the old version will continue running, or the update will fail silently. Always place it at the absolute beginning of your `main` function.

- **Issue**: Hardcoding `CurrentVersion` instead of injecting it at build time.
  - **Solution**: Hardcoding makes releases cumbersome and prone to errors. Use Go's `-ldflags` to embed the version (e.g., from `git describe --tags`) directly into the binary during compilation. This ensures the application always knows its true version.

- **Issue**: Incorrect `AssetPattern` or inconsistent GitHub release asset naming.
  - **Solution**: `ghupdate` relies on an exact match between the `AssetPattern` and your uploaded GitHub release binary names. Double-check the pattern, including all placeholders (`{version}`, `{os}`, `{arch}`, `{ext}`), against the actual file names on your release page.

## Lifecycle Dependencies

The `ghupdate` library expects to manage its lifecycle during application startup and during explicit update checks. `HandleUpdateMode()` must run at the very beginning to capture control when an update is being applied. `ApplyUpdate()` causes the current process to exit, transferring control to the newly spawned executable. Therefore, any resources or goroutines started *before* `ApplyUpdate()` is called will be terminated. Cleanup should be done during startup, not shutdown, to ensure temporary files from failed attempts are removed.



---
*Generated using Gemini AI on 6/16/2025, 8:26:16 PM. Review and refine as needed.*