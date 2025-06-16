package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/asaidimu/ghupdate"
)

var (
	Version = "dev"
	BuildDate =  "unknown"
)

const (
	githubOwner    = "asaidimu"
	githubRepo     = "ghupdate"
	currentVersion = "v1.0.0"
	assetPattern   = "example-{version}-{os}-{arch}{ext}"
	appName = "example"
)

func main() {
	// 1. Handle update mode first
	// This ensures that if the application was launched to perform an update,
	// it completes that task before anything else.
	if ghupdate.HandleUpdateMode() {
		// If HandleUpdateMode returns true, it means the update was applied,
		// and the application is now running the new version.
		// You might want to log this or perform post-update setup.
		fmt.Println("Successfully updated! Resuming normal operation.")
		// The program flow will continue here with the *new* executable.
	}

	// Determine data dir
	dataDir, err := getAppDataDir(appName)
	if err != nil {
		log.Fatalf("Error determining application data directory: %v", err)
	}

	// 2. Clean up any leftover update files from previous runs
	if err := ghupdate.CleanupUpdate(dataDir); err != nil {
		log.Printf("Warning: Failed to clean up old update files: %v\n", err)
	}

	// 3. Perform regular application logic
	fmt.Printf("My Application - Version %s\n", currentVersion)
	fmt.Println("Running application logic...")

	// Simulate some work
	time.Sleep(1 * time.Second)

	// 4. Check for updates periodically (or on user command)
	// In a real application, you might do this in a goroutine
	// or trigger it via a UI button.
	checkUpdates(dataDir)

	fmt.Println("Application finished.")
}

// getAppDataDir returns a cross-platform path suitable for storing application-specific data.
// It prioritizes user cache directory for temporary update files.
// For persistent configuration, os.UserConfigDir() would be more appropriate.
//
// Conventions:
// - Linux:   ~/.cache/appName/ (or $XDG_CACHE_HOME/appName)
// - macOS:   ~/Library/Caches/appName/
// - Windows: %LOCALAPPDATA%\appName\ (e.g., C:\Users\Username\AppData\Local\appName\)
func getAppDataDir(appName string) (string, error) {
	// For temporary files like downloaded updates, UserCacheDir is often the best fit.
	// It's intended for non-essential, transient data.
	dir, err := os.UserCacheDir()
	if err != nil {
		// Fallback to UserConfigDir if cache dir is not available or errors,
		// though this is less ideal for temporary files.
		dir, err = os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("could not determine user cache or config directory: %w", err)
		}
	}

	// Append a unique application-specific subdirectory
	appDataDir := filepath.Join(dir, appName)

	// Ensure the directory exists
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
		GitHubToken:    os.Getenv("GITHUB_TOKEN"), // Good practice to use an env var
		CurrentVersion: Version,
		DataDir:        dataDir,
		ExecutablePath: executablePath,
		AssetPattern:   assetPattern,
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
}
