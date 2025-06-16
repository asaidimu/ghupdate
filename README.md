# ghupdate: Seamless Self-Updating for Go Applications

[![Go Reference](https://pkg.go.dev/badge/github.com/asaidimu/ghupdate.svg)](https://pkg.go.dev/github.com/asaidimu/ghupdate)
[![Build Status](https://github.com/asaidimu/ghupdate/workflows/Test%20Workflow/badge.svg)](https://github.com/asaidimu/ghupdate/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/go-1.24%2B-00ADD8?logo=go)](https://golang.org)

A robust and flexible solution for self-updating Go applications directly from GitHub releases.

---

## Table of Contents

*   [Overview](#overview)
*   [Features](#features)
*   [Installation](#installation)
*   [Usage](#usage)
    *   [Basic Integration](#basic-integration)
    *   [UpdateConfig Details](#updateconfig-details)
    *   [Asset Pattern](#asset-pattern)
    *   [Best Practices](#best-practices)
*   [Project Architecture](#project-architecture)
*   [Development & Contributing](#development--contributing)
*   [Troubleshooting](#troubleshooting)
*   [FAQ](#faq)
*   [License](#license)
*   [Acknowledgments](#acknowledgments)

---

## Overview

Distributing and updating command-line tools and desktop applications can be a significant challenge. `ghupdate` addresses this by providing a simple yet powerful library for Go applications to self-update directly from GitHub releases. It handles checking for newer versions, downloading the correct binary for the user's operating system and architecture, and performing an atomic, in-place replacement of the currently running executable.

This library is designed for resilience, ensuring that updates are applied smoothly and safely. It manages temporary files, gracefully handles various error conditions, and is built to integrate seamlessly into your application's startup flow, allowing for a truly hands-off update experience for your users.

## Features

✨ **Seamless GitHub Releases Integration**: Automatically fetches the latest release information and assets from your public or private GitHub repository.
⚡️ **Semantic Versioning (SemVer) Compliance**: Accurately compares `vX.Y.Z` versions to determine if an update is available, using `golang.org/x/mod/semver`.
🚀 **Cross-Platform Support**: Works flawlessly on Windows, Linux, and macOS, handling platform-specific requirements like executable permissions and file extensions.
🔄 **Atomic In-Place Updates**: Replaces the running executable without requiring administrative privileges (in most user-writable scenarios) or interrupting user workflow.
🗑️ **Automatic Cleanup**: Manages temporary update files, ensuring a clean and efficient update process.
🧩 **Flexible Asset Pattern Matching**: Customizable pattern (`{version}-{os}-{arch}{ext}`) to reliably identify the correct binary asset among multiple release files.
🔑 **GitHub Token Support**: Optionally use a GitHub Personal Access Token (PAT) for private repositories or to avoid public API rate limits.
💪 **Resilient Error Handling**: Provides clear error reporting for network issues, file system problems, or invalid configurations.

## Installation

### Prerequisites

*   Go 1.24.4 or higher

### Installation Steps

To integrate `ghupdate` into your Go project, simply use `go get`:

```bash
go get github.com/asaidimu/ghupdate
```

This command will download the library and add it to your `go.mod` file.

### Configuration

`ghupdate` is configured programmatically through the `ghupdate.UpdateConfig` struct. There are no external configuration files required by the library itself, though your application might manage its own settings.

For authenticated access to private repositories or to mitigate GitHub API rate limiting for public ones, you can set the `GITHUB_TOKEN` environment variable, which `ghupdate` will pick up if provided in `UpdateConfig`.

```bash
export GITHUB_TOKEN="YOUR_GITHUB_PERSONAL_ACCESS_TOKEN"
```

## Usage

Integrating `ghupdate` into your application involves a few key steps to ensure a robust and smooth update experience.

### Basic Integration

The typical flow for an application using `ghupdate` is as follows:

1.  **Handle Update Mode First**: Call `ghupdate.HandleUpdateMode()` at the very beginning of your `main` function. This is critical as it allows a newly launched executable (spawned by a previous `ApplyUpdate` call) to replace the old one before any other application logic runs.
2.  **Clean Up Old Updates**: After handling update mode, call `ghupdate.CleanupUpdate()` to remove any leftover temporary update files from previous failed or successful update attempts.
3.  **Check and Prepare Update**: Periodically (e.g., on startup, hourly, or on user command), call `ghupdate.CheckAndPrepareUpdate()` to see if a newer version is available and download it.
4.  **Apply Update**: If `CheckAndPrepareUpdate()` indicates an update is ready, call `ghupdate.ApplyUpdate()`. This will spawn the newly downloaded executable, which in turn will take over and replace the currently running one. The current process will then exit.

Here's a condensed example demonstrating this flow, similar to the `example/main.go` provided in the codebase:

```go
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/asaidimu/ghupdate"
)

// These variables are typically injected at build time using linker flags
var (
	Version   = "dev" // e.g., "v1.0.0"
	BuildDate = "unknown"
)

const (
	githubOwner  = "your-github-owner" // Replace with your GitHub username/org
	githubRepo   = "your-app-repo"     // Replace with your repository name
	assetPattern = "{appname}-{version}-{os}-{arch}{ext}" // Example pattern
	appName      = "mycli" // Your application's base name
)

func main() {
	// 1. Handle update mode first. This is crucial!
	// If true, the application was just updated and is now running the new version.
	if ghupdate.HandleUpdateMode() {
		fmt.Println("🎉 Successfully updated! Resuming normal operation.")
		// The program flow continues here with the *new* executable.
	}

	// Determine a cross-platform data directory for temporary update files.
	dataDir, err := getAppDataDir(appName)
	if err != nil {
		log.Fatalf("Error determining application data directory: %v", err)
	}

	// 2. Clean up any leftover update files from previous runs.
	if err := ghupdate.CleanupUpdate(dataDir); err != nil {
		log.Printf("Warning: Failed to clean up old update files: %v\n", err)
	}

	// 3. Perform regular application logic.
	fmt.Printf("%s - Version %s (Built: %s)\n", appName, Version, BuildDate)
	fmt.Println("Running application logic...")
	time.Sleep(2 * time.Second) // Simulate work

	// 4. Check for updates (e.g., on startup, periodically, or on user command).
	checkUpdates(dataDir)

	fmt.Println("Application finished.")
}

// getAppDataDir returns a cross-platform path suitable for storing application-specific data.
// For temporary update files, os.UserCacheDir() is generally preferred.
func getAppDataDir(appName string) (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir, err = os.UserConfigDir() // Fallback
		if err != nil {
			return "", fmt.Errorf("could not determine user cache or config directory: %w", err)
		}
	}
	appDataDir := filepath.Join(dir, appName)
	if err := os.MkdirAll(appDataDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create application data directory %q: %w", appDataDir, err)
	}
	return appDataDir, nil
}

func checkUpdates(dataDir string) {
	fmt.Println("\nChecking for updates...")

	executablePath, err := os.Executable()
	if err != nil {
		log.Printf("Error getting executable path: %v", err)
		return
	}

	config := ghupdate.UpdateConfig{
		GitHubOwner:    githubOwner,
		GitHubRepo:     githubRepo,
		GitHubToken:    os.Getenv("GITHUB_TOKEN"), // Optional: use for private repos or rate limiting
		CurrentVersion: Version,
		DataDir:        dataDir,
		ExecutablePath: executablePath,
		AssetPattern:   strings.ReplaceAll(assetPattern, "{appname}", appName),
		// OS and Arch can be left empty to use runtime.GOOS/GOARCH
	}

	updateInfo, err := ghupdate.CheckAndPrepareUpdate(config)
	if err != nil {
		log.Printf("Error checking for update: %v\n", err)
		return
	}

	if updateInfo == nil {
		fmt.Println("No update available. You are on the latest version.")
		return
	}

	fmt.Printf("Update available! Current: %s, Latest: %s\n", updateInfo.CurrentVersion, updateInfo.LatestVersion)
	fmt.Printf("Release Notes:\n%s\n", updateInfo.ReleaseNotes)
	fmt.Printf("Downloaded asset: %s\n", updateInfo.AssetName)

	fmt.Println("Applying update...")
	err = ghupdate.ApplyUpdate(config)
	if err != nil {
		log.Fatalf("Error applying update: %v\n", err)
	}
	// Note: If ApplyUpdate succeeds, the program exits here.
	// The new executable has taken over.
}
```

### UpdateConfig Details

The `ghupdate.UpdateConfig` struct holds all necessary parameters for the update process:

| Field            | Type     | Description                                                                                                                                                                             | Required |
| :--------------- | :------- | :--------------------------------------------------------------------------------------------------------------------------------------------- | :------- |
| `GitHubOwner`    | `string` | The GitHub username or organization name that owns the repository (e.g., `"octocat"`).                                                         | Yes      |
| `GitHubRepo`     | `string` | The name of the GitHub repository where releases are hosted (e.g., `"Spoon-Knife"`).                                                           | Yes      |
| `GitHubToken`    | `string` | An optional GitHub personal access token. Recommended for private repositories or to avoid public API rate limits.                               | No       |
| `CurrentVersion` | `string` | The semantic version of the currently running application (e.g., `"v1.2.3"` or `"1.2.3"`). This should ideally be injected at build time.        | Yes      |
| `DataDir`        | `string` | Absolute path to a writable directory for temporary update files. `os.UserCacheDir()` is a good choice.                                        | Yes      |
| `ExecutablePath` | `string` | Absolute path to the currently running executable (`os.Executable()`). This is where the new binary will be copied.                              | Yes      |
| `AssetPattern`   | `string` | A pattern string to identify the correct release asset. Supports `{version}`, `{os}`, `{arch}`, `{ext}` placeholders.                          | Yes      |
| `OS`             | `string` | The target operating system for the update asset (e.g., `"windows"`, `"linux"`, `"darwin"`). If empty, `runtime.GOOS` is used.                 | No       |
| `Arch`           | `string` | The target architecture for the update asset (e.g., `"amd64"`, `"arm64"`). If empty, `runtime.GOARCH` is used.                                   | No       |

### Asset Pattern

The `AssetPattern` is crucial for `ghupdate` to find the correct binary in your GitHub release. It uses placeholders that are replaced dynamically based on the target system and release version.

*   `{version}`: Replaced by the release's `tag_name` (e.g., `v1.0.0`).
*   `{os}`: Replaced by the target operating system (e.g., `windows`, `linux`, `darwin`).
*   `{arch}`: Replaced by the target architecture (e.g., `amd64`, `arm64`).
*   `{ext}`: Replaced by `.exe` on Windows, and an empty string on other OS.

**Example:** If your release assets are named `myapp-v1.2.3-linux-amd64` and `myapp-v1.2.3-windows-amd64.exe`, your `AssetPattern` should be:

```go
const assetPattern = "myapp-{version}-{os}-{arch}{ext}"
```

### Best Practices

*   **Version Injection**: Dynamically inject `CurrentVersion` at build time using Go linker flags (`-ldflags "-X main.Version=$(VERSION)"`) rather than hardcoding it. This ensures your application always knows its true version.
*   **Data Directory**: Use `os.UserCacheDir()` or `os.UserConfigDir()` to determine a cross-platform, user-specific directory for storing temporary update files.
*   **GitHub Token**: For public repositories, a token can help avoid API rate limits. For private repositories, a token is mandatory. Ensure the token has `repo` scope for private repos, or `public_repo` scope for public ones.
*   **Update Frequency**: Don't check for updates excessively. On application startup, daily, or on user command are good strategies.
*   **User Notification**: Inform users when an update is available or applied. `UpdateInfo.ReleaseNotes` can be displayed to show changelog.
*   **Error Handling**: Log errors from `CheckAndPrepareUpdate` and `CleanupUpdate`, but don't necessarily exit the application unless the error is critical to basic functionality. `ApplyUpdate` failures are critical and should lead to an exit.

## Project Architecture

`ghupdate` is designed as a single, self-contained Go package for easy integration.

### Core Components

The primary functionality resides within `updates.go` and revolves around these key functions and structs:

*   **`UpdateConfig`**: A struct to configure all parameters for the update process, including GitHub repository details, current version, and file paths.
*   **`UpdateInfo`**: A struct returned by `CheckAndPrepareUpdate` that provides details about an available update, such as the latest version, download URL, and release notes.
*   **`CheckAndPrepareUpdate(config UpdateConfig)`**:
    *   Connects to GitHub API to fetch the latest release.
    *   Compares `CurrentVersion` with the `latest_release.tag_name` using semantic versioning.
    *   If a newer version exists, it identifies the correct asset based on `AssetPattern`, `runtime.GOOS`, and `runtime.GOARCH`.
    *   Downloads the identified asset to the `DataDir`.
    *   Returns `*UpdateInfo` if an update is found and prepared, or `nil` otherwise.
*   **`ApplyUpdate(config UpdateConfig)`**:
    *   Initiates the actual update process.
    *   Spawns the newly downloaded executable (from `DataDir`) as a new process.
    *   Passes special arguments to the new process (`--perform-update`, `--original-path`, `--pid`).
    *   Exits the current process, allowing the newly launched process to take over.
*   **`HandleUpdateMode()`**:
    *   **Crucial for in-place updates.** Designed to be called as the very first thing in `main()`.
    *   Checks if the application was launched with the `--perform-update` argument (which `ApplyUpdate` does).
    *   If in update mode, it waits for the original process (which launched it) to exit.
    *   Copies its own executable (the new version) to the `original-path` specified by the arguments, effectively overwriting the old executable.
    *   Returns `true` if it successfully handled an update and the application should continue running normally (now as the new version), or `false` if not in update mode.
*   **`CleanupUpdate(dataDir string)`**:
    *   Removes any temporary update files (`update.exe` or `update`) from the specified `dataDir`.
    *   Should be called early in the application lifecycle to clean up after previous update attempts.

### Data Flow

1.  **Initial Launch**: Application starts. `main()` calls `HandleUpdateMode()`.
2.  **Update Mode Check**: `HandleUpdateMode()` returns `false` (first launch, not in update mode).
3.  **Cleanup**: `CleanupUpdate()` runs, ensuring no old temporary update files are present.
4.  **Regular Logic**: Application performs its normal operations.
5.  **Check for Updates**: At some point, `CheckAndPrepareUpdate()` is called with `UpdateConfig`.
6.  **Download**: If an update is found, `CheckAndPrepareUpdate()` downloads the new binary to `DataDir` (e.g., `~/.cache/myapp/update`).
7.  **Apply Update**: `ApplyUpdate()` is called. It launches the downloaded `update` binary as a child process, passing arguments like `--perform-update --original-path=<current-app-path> --pid=<current-app-pid>`. The original process then calls `os.Exit(0)`.
8.  **New Process Takes Over**: The newly launched `update` binary's `main()` function executes. The first call `HandleUpdateMode()` now returns `true`.
9.  **Replacement**: `HandleUpdateMode()` waits for the *old* parent process (which just exited) to fully terminate, then copies itself (`~/.cache/myapp/update`) over the `original-path` (`/usr/local/bin/myapp` or similar).
10. **Resumed Execution**: The new executable continues its `main()` function, now running from the original path as the updated version.

## Development & Contributing

We welcome contributions! Please follow these guidelines to help us maintain the quality and consistency of `ghupdate`.

### Development Setup

1.  **Clone the Repository**:
    ```bash
    git clone https://github.com/asaidimu/ghupdate.git
    cd ghupdate
    ```
2.  **Download Dependencies**:
    ```bash
    go mod tidy
    ```

### Scripts

The `Makefile` provides convenient commands for development:

*   **`make build`**: Builds the `example/main.go` application for Linux AMD64 and places it in the `dist/` directory. It also injects `VERSION` and `BUILD_DATE` using `ldflags`.
    ```bash
    make build
    ```
    To specify a version: `VERSION=v1.2.3 make build`
*   **`make test`**: Runs all Go tests for the project.
    ```bash
    make test
    ```
*   **`make clean`**: Removes the `dist/` directory and any built binaries.
    ```bash
    make clean
    ```

### Testing

Run tests locally with:

```bash
go test -v ./...
```

Please ensure all new features come with corresponding unit tests, and existing tests pass before submitting a pull request.

### Contributing Guidelines

1.  **Fork the repository** and create your branch from `main`.
2.  **Make your changes**.
3.  **Write tests** for your changes.
4.  **Ensure tests pass** (`go test ./...`).
5.  **Format your code** (`go fmt ./...`).
6.  **Create a clear and concise commit message** following conventional commits (e.g., `feat: add new update mechanism`).
7.  **Submit a Pull Request** against the `main` branch.

### Issue Reporting

Please report any bugs, feature requests, or questions through the GitHub Issues page. Provide as much detail as possible, including:

*   **Steps to reproduce** the bug.
*   **Expected behavior** vs. **actual behavior**.
*   **Error messages** or stack traces.
*   **Your operating system and Go version**.

## Troubleshooting

Here are solutions to some common issues you might encounter:

*   **"Error checking for update: Failed to find matching asset"**:
    *   **Cause**: The `AssetPattern` in your `UpdateConfig` does not match the actual file names in your GitHub release.
    *   **Solution**: Double-check your release asset names (e.g., `your-app-v1.0.0-linux-amd64`) and ensure your `AssetPattern` matches it exactly, including `{version}`, `{os}`, `{arch}`, and `{ext}` placeholders. Also, ensure the release is *not* a `draft` or `prerelease` unless specifically configured to look for those.
*   **"GitHub API returned status 403" or "API rate limit exceeded"**:
    *   **Cause**: You've hit GitHub's API rate limits, or you're trying to access a private repository without proper authentication.
    *   **Solution**: Provide a `GitHubToken` in your `UpdateConfig` (`os.Getenv("GITHUB_TOKEN")`). For private repositories, ensure the token has sufficient permissions (e.g., `repo` scope).
*   **"Permission denied" during `ApplyUpdate` or `CleanupUpdate`**:
    *   **Cause**: The application does not have write permissions to `DataDir` or `ExecutablePath`. This can happen if the executable is in a system-wide location (e.g., `/usr/local/bin`) and the user does not have administrative privileges.
    *   **Solution**:
        *   Ensure `DataDir` is in a user-writable location (like `os.UserCacheDir()`).
        *   For `ExecutablePath`, `ghupdate`'s design aims to avoid needing elevated privileges for the file replacement itself on *most* systems where the executable is in a user-owned directory. If your app is installed system-wide (e.g., `/usr/bin`), you might need to reconsider your deployment strategy or prompt the user for elevation (which `ghupdate` doesn't directly handle). In some cases, copying to a temporary location and then moving with elevation might be necessary, but this library focuses on the common non-admin use case.
*   **Application doesn't update, or `HandleUpdateMode()` isn't called**:
    *   **Cause**: `ghupdate.HandleUpdateMode()` is not called as the very first line in your `main` function, or the arguments passed during `ApplyUpdate` are incorrect (though this is managed internally by the library).
    *   **Solution**: Always ensure `if ghupdate.HandleUpdateMode() { ... }` is the absolute first thing that happens in `main()`.

## FAQ

**Q: How does `ghupdate` replace the running executable without causing issues?**
A: `ghupdate` employs a "spawn-and-replace" strategy. When `ApplyUpdate` is called, it launches the *newly downloaded binary* (which is a copy of itself) as a separate process. This new process then waits for the *original* running application to exit (which `ApplyUpdate` causes immediately after spawning). Once the old process exits and releases its file lock, the new process copies itself over the old executable's path, effectively replacing it. The new process then continues running as the updated application.

**Q: What happens if the update process fails midway (e.g., power loss, disk full)?**
A: The process is designed to be as atomic as possible. The new binary is first downloaded to a temporary location. The replacement only occurs after the new binary is fully downloaded and verified (implicitly by being executable). If a failure occurs during download or before the copy, the old executable remains untouched and functional. If a failure occurs *during* the file copy, it might leave a corrupted executable, but `ghupdate` uses standard file copy operations which are generally robust. Calling `CleanupUpdate()` on startup helps remove potentially corrupted temporary update files from prior attempts.

**Q: Can I use `ghupdate` for private GitHub repositories?**
A: Yes, set the `GitHubToken` field in your `UpdateConfig` struct with a Personal Access Token that has `repo` scope.

**Q: How do I dynamically set `CurrentVersion` for my application?**
A: The most common and robust way is to use Go's linker flags during your build process. For example, in your `main` package, declare a variable:
`var Version string`
Then, when building:
`go build -ldflags "-X main.Version=$(git describe --tags --abbrev=0 --always)" -o myapp`
This embeds the version string directly into your binary.

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details.

## Acknowledgments

*   Developed by Saidimu.
*   Inspired by existing self-update mechanisms in various Go projects and the need for a robust, reusable solution.
