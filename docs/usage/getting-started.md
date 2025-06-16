# Getting Started

### Overview and Core Concepts

`ghupdate` simplifies the distribution and maintenance of your Go applications by enabling them to update themselves. The core idea is a 'spawn-and-replace' mechanism:

1.  **Preparation**: Your application checks GitHub for a newer version and downloads the new executable to a temporary location.
2.  **Application**: When an update is ready, your running application *launches the newly downloaded executable* as a child process.
3.  **Replacement**: The *new* executable (the child process) then waits for the *original* parent process to exit, and once the original executable's file lock is released, the new executable copies itself over the old one, becoming the new active version.
4.  **Resumption**: The new executable then continues its `main` function, now running from the original path as the updated application.

This sequence ensures an atomic and robust update without requiring elevated privileges for the replacement itself in most common user-writable scenarios.

### Quick Setup Guide

To integrate `ghupdate` into your Go project:

1.  **Install the library**:

    ```bash
    go get github.com/asaidimu/ghupdate
    ```

2.  **Define application constants**:

    ```go
    // In your main package, define constants for your GitHub repo and app details
    const (
    	githubOwner    = "your-github-owner"
    	githubRepo     = "your-repo-name"
    	assetPattern   = "your-app-name-{version}-{os}-{arch}{ext}" // Must match your release asset naming
    	appName        = "your-app-name"
    )

    // Version should be dynamically set at build time (e.g., via ldflags)
    var Version = "dev" // Placeholder, will be replaced by build process
    ```

3.  **Implement the core update flow in `main()`**:

    ```go
    package main

    import (
    	"fmt"
    	"log"
    	"os"
    	"path/filepath"
    	"time"

    	"github.com/asaidimu/ghupdate"
    )

    // ... (constants and Version variable as above)

    func main() {
    	// Step 1: Crucially, handle update mode first.
    	// If this returns true, the app was just updated and is now running the new version.
    	if ghupdate.HandleUpdateMode() {
    		fmt.Println("🎉 Successfully updated! Resuming normal operation.")
    	}

    	// Determine a cross-platform data directory for temporary files.
    	dataDir, err := getAppDataDir(appName)
    	if err != nil {
    		log.Fatalf("Error determining application data directory: %v", err)
    	}

    	// Step 2: Clean up any leftover temporary update files.
    	if err := ghupdate.CleanupUpdate(dataDir); err != nil {
    		log.Printf("Warning: Failed to clean up old update files: %v\n", err)
    	}

    	// Step 3: Your application's normal logic goes here.
    	fmt.Printf("%s - Version %s\n", appName, Version)
    	fmt.Println("Running application logic...")
    	time.Sleep(2 * time.Second)

    	// Step 4: Periodically check for and apply updates.
    	checkUpdates(dataDir) // Or trigger via user command/goroutine

    	fmt.Println("Application finished.")
    }

    // Helper to get a cross-platform temporary data directory
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

    // Function to encapsulate the update check logic
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
    		GitHubToken:    os.Getenv("GITHUB_TOKEN"), // Optional: For private repos or rate limits
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

    	fmt.Println("Applying update...")
    	err = ghupdate.ApplyUpdate(config)
    	if err != nil {
    		log.Fatalf("Error applying update: %v\n", err)
    	}
    	// If ApplyUpdate succeeds, the program exits here.
    	// The new executable has taken over.
    }
    ```

### First Tasks with Decision Patterns

**Task 1: Initialize the Update System at Application Startup**

- **Goal**: Ensure the application can handle an incoming update and prepare for future checks.
- **Decision Pattern**: IF application is launching THEN call `HandleUpdateMode()` first to check if it's an update process, then call `CleanupUpdate()` to clear old temporary files. THEN proceed with normal application logic.
- **Expected Outcome**: Application starts cleanly, able to self-update.


---
### 🤖 AI Agent Guidance

```json
{
  "decisionPoints": [
    "IF application_startup THEN call method:HandleUpdateMode AND call method:CleanupUpdate",
    "IF method:HandleUpdateMode returns true THEN log \"Update successful\" AND proceed_with_new_executable",
    "IF method:HandleUpdateMode returns false THEN continue_with_normal_startup_logic"
  ],
  "verificationSteps": [
    "Check: `ghupdate.HandleUpdateMode()` is the first executable line in `main()`.",
    "Check: `ghupdate.CleanupUpdate(dataDir)` is called after `HandleUpdateMode()`.",
    "Check: `dataDir` is a user-writable path, preferably `os.UserCacheDir()`."
  ],
  "quickPatterns": [
    "Pattern: Go main function boilerplate\n```go\npackage main\n\nimport (\n\t\"fmt\"\n\t\"log\"\n\t\"os\"\n\t\"path/filepath\"\n\t\"time\"\n\t\"github.com/asaidimu/ghupdate\"\n)\n\nvar Version = \"dev\"\nconst appName = \"my-app\"\nconst githubOwner = \"my-owner\"\nconst githubRepo = \"my-repo\"\nconst assetPattern = \"my-app-{version}-{os}-{arch}{ext}\"\n\nfunc main() {\n\tif ghupdate.HandleUpdateMode() {\n\t\tfmt.Println(\"Updated!\")\n\t}\n\n\tdataDir, err := os.UserCacheDir()\n\tif err != nil {\n\t\tlog.Fatalf(\"Error: %v\", err)\n\t}\n\tdataDir = filepath.Join(dataDir, appName)\n\t_ = os.MkdirAll(dataDir, 0755) // Ensure dir exists\n\n\tif err := ghupdate.CleanupUpdate(dataDir); err != nil {\n\t\tlog.Printf(\"Cleanup warning: %v\\n\", err)\n\t}\n\n\tfmt.Printf(\"Running %s version %s\\n\", appName, Version)\n\n\t// Application's core logic here\n\n\tcheckUpdates(dataDir)\n}\n\nfunc checkUpdates(dataDir string) {\n\texecPath, _ := os.Executable()\n\tconfig := ghupdate.UpdateConfig{\n\t\tGitHubOwner: githubOwner,\n\t\tGitHubRepo: githubRepo,\n\t\tCurrentVersion: Version,\n\t\tDataDir: dataDir,\n\t\tExecutablePath: execPath,\n\t\tAssetPattern: assetPattern,\n\t}\n\tupdateInfo, err := ghupdate.CheckAndPrepareUpdate(config)\n\tif err != nil {\n\t\tlog.Printf(\"Update check error: %v\\n\", err)\n\t\treturn\n\t}\n\tif updateInfo != nil {\n\t\tfmt.Printf(\"Update available: %s -> %s\\n\", updateInfo.CurrentVersion, updateInfo.LatestVersion)\n\t\tif err := ghupdate.ApplyUpdate(config); err != nil {\n\t\t\tlog.Fatalf(\"Apply update error: %v\\n\", err)\n\t\t}\n\t}\n}\n```"
  ],
  "diagnosticPaths": [
    "Error: Application not updating -> Symptom: Old version keeps running after `ApplyUpdate` call -> Check: Is `ghupdate.HandleUpdateMode()` the very first function call in `main()`? -> Fix: Move `ghupdate.HandleUpdateMode()` to the top of `main()` before any other logic."
  ]
}
```

---
*Generated using Gemini AI on 6/16/2025, 3:26:10 PM. Review and refine as needed.*