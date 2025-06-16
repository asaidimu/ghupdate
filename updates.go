package ghupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/mod/semver"
)

// UpdateConfig holds configuration for the update process.
type UpdateConfig struct {
	// GitHubOwner is the username or organization name of the GitHub repository owner (e.g., "my-org").
	GitHubOwner string
	// GitHubRepo is the name of the GitHub repository where releases are hosted (e.g., "my-app").
	GitHubRepo string
	// GitHubToken is an optional GitHub personal access token. This is optional for public repositories
	// but highly recommended for private repositories or to avoid rate limiting for public ones.
	GitHubToken string
	// CurrentVersion is the semantic version of the currently running application (e.g., "v1.2.3" or "1.2.3").
	CurrentVersion string
	// DataDir is the absolute path to a directory where temporary update files (like the downloaded new executable)
	// will be stored. This directory must be writable by the application.
	DataDir string
	// ExecutablePath is the absolute path to the currently running executable. This is used by the update process
	// to know where to copy the new executable.
	ExecutablePath string
	// AssetPattern is a pattern string used to identify the correct release asset to download.
	// It supports placeholders:
	// - {version}: Will be replaced by the release tag name.
	// - {os}: Will be replaced by the target operating system (e.g., "windows", "linux", "darwin").
	// - {arch}: Will be replaced by the target architecture (e.g., "amd64", "arm64").
	// - {ext}: Will be replaced by ".exe" on Windows, and an empty string on other OS.
	// Example: "myapp-{version}-{os}-{arch}{ext}"
	AssetPattern string
	// OS is the target operating system for the update asset. If left empty, runtime.GOOS will be used.
	OS string
	// Arch is the target architecture for the update asset. If left empty, runtime.GOARCH will be used.
	Arch string
}

// UpdateInfo contains information about an available update.
type UpdateInfo struct {
	// CurrentVersion is the version of the currently running application.
	CurrentVersion string
	// LatestVersion is the semantic version of the latest available release.
	LatestVersion string
	// DownloadURL is the direct URL to download the update asset.
	DownloadURL string
	// AssetName is the name of the update asset on GitHub.
	AssetName string
	// ReleaseNotes is the body/description of the latest GitHub release, often containing changelog information.
	ReleaseNotes string
}

// GitHubAsset represents a release asset from GitHub API.
type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// GitHubRelease represents a release from GitHub API.
type GitHubRelease struct {
	TagName    string        `json:"tag_name"`
	Name       string        `json:"name"`
	Body       string        `json:"body"`
	Draft      bool          `json:"draft"`
	Prerelease bool          `json:"prerelease"`
	Assets     []GitHubAsset `json:"assets"`
}

// CheckAndPrepareUpdate checks for available updates and downloads the new executable if a newer version is found.
// It validates the provided UpdateConfig, fetches the latest release information from the specified GitHub repository,
// and determines if a newer version is available. If an update is found, it downloads the appropriate executable
// asset based on the AssetPattern and the target OS/architecture, storing it in the DataDir.
// The downloaded file is also made executable on Unix-like systems.
//
// It returns an UpdateInfo struct containing details about the available update if one is found,
// or nil if no update is needed. An error is returned if any step in the process fails,
// such as invalid configuration, network issues, or inability to find a matching asset.
func CheckAndPrepareUpdate(config UpdateConfig) (*UpdateInfo, error) {
	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Auto-detect platform if not specified
	targetOS := config.OS
	targetArch := config.Arch
	if targetOS == "" {
		targetOS = runtime.GOOS
	}
	if targetArch == "" {
		targetArch = runtime.GOARCH
	}

	// Fetch latest release from GitHub
	release, err := fetchLatestRelease(config)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
	}

	// Check if update is needed
	if !isNewerVersion(config.CurrentVersion, release.TagName) {
		return nil, nil // No update needed
	}

	// Find matching asset
	asset, err := findMatchingAsset(release.Assets, config.AssetPattern, release.TagName, targetOS, targetArch)
	if err != nil {
		return nil, fmt.Errorf("failed to find matching asset: %w", err)
	}

	// Download the update
	updatePath := filepath.Join(config.DataDir, "update"+getExecutableExtension())
	if err := downloadAsset(asset.BrowserDownloadURL, updatePath, config.GitHubToken); err != nil {
		return nil, fmt.Errorf("failed to download update: %w", err)
	}

	// Make executable on Unix systems
	if runtime.GOOS != "windows" {
		if err := os.Chmod(updatePath, 0755); err != nil {
			return nil, fmt.Errorf("failed to make update executable: %w", err)
		}
	}

	return &UpdateInfo{
		CurrentVersion: config.CurrentVersion,
		LatestVersion:  release.TagName,
		DownloadURL:    asset.BrowserDownloadURL,
		AssetName:      asset.Name,
		ReleaseNotes:   release.Body,
	}, nil
}

// ApplyUpdate applies a previously prepared update.
// It assumes that CheckAndPrepareUpdate has already been successfully called and
// the update file exists in the DataDir.
//
// This function spawns a new process (the downloaded update) with special arguments
// (--perform-update, --original-path, --pid) that instruct the new process to perform
// the actual file replacement. After successfully starting the new process,
// the current application exits, allowing the new process to take over.
//
// Note: If this function succeeds, the current process will call os.Exit(0) and terminate,
// so the return value will typically not be observed in a successful scenario.
func ApplyUpdate(config UpdateConfig) error {
	updatePath := filepath.Join(config.DataDir, "update"+getExecutableExtension())

	// Check if update file exists
	if _, err := os.Stat(updatePath); os.IsNotExist(err) {
		return fmt.Errorf("no prepared update found at %s", updatePath)
	}

	// Get current process PID
	currentPID := os.Getpid()

	// Spawn the update process
	// The new process will run with the --perform-update flag, instructing it
	// to replace the original executable and then continue as the main application.
	cmd := exec.Command(updatePath, "--perform-update",
		"--original-path="+config.ExecutablePath,
		"--pid="+strconv.Itoa(currentPID))

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start update process: %w", err)
	}

	// Exit current process - the update will take over
	os.Exit(0)
	return nil // Never reached
}

// CleanupUpdate removes leftover temporary update files from the data directory.
// It should typically be called at the startup of your application to ensure that
// no partially downloaded or old update executables remain from previous update attempts.
//
// It returns nil if no update file is found or if cleanup is successful.
// An error is returned if the cleanup operation fails (e.g., permission issues).
func CleanupUpdate(dataDir string) error {
	updatePath := filepath.Join(dataDir, "update"+getExecutableExtension())

	if _, err := os.Stat(updatePath); os.IsNotExist(err) {
		return nil // Nothing to clean up
	}

	if err := os.Remove(updatePath); err != nil {
		return fmt.Errorf("failed to cleanup update file: %w", err)
	}

	return nil
}

// HandleUpdateMode is a crucial function designed to be called early in your application's `main` function.
// It checks if the application was launched in "update mode" (i.e., with the `--perform-update` argument).
// If so, it takes over the execution flow, waits for the original process to exit,
// copies itself (the newly updated executable) to the original application's path,
// and then allows the application to continue running normally.
//
// This mechanism ensures a seamless in-place update without requiring user intervention.
// If the application is not in update mode, it simply returns `false` and the application
// proceeds with its normal startup.
//
// If an error occurs during the update mode handling (e.g., invalid arguments,
// failure to wait for the old process, or failure to copy the file),
// it prints an error to os.Stderr and calls os.Exit(1).
func HandleUpdateMode() bool {
	args := os.Args[1:]
	if len(args) == 0 || args[0] != "--perform-update" {
		return false // Not in update mode
	}

	// Parse arguments
	var originalPath string
	var pidToWait int

	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "--original-path=") {
			originalPath = strings.TrimPrefix(arg, "--original-path=")
		} else if strings.HasPrefix(arg, "--pid=") {
			pidStr := strings.TrimPrefix(arg, "--pid=")
			if pid, err := strconv.Atoi(pidStr); err == nil {
				pidToWait = pid
			}
		}
	}

	if originalPath == "" || pidToWait == 0 {
		fmt.Fprintf(os.Stderr, "Invalid update mode arguments: original-path=%q, pid=%d\n", originalPath, pidToWait)
		os.Exit(1)
	}

	// Wait for old process to exit
	// This is critical to ensure the old executable file is not locked
	// before attempting to overwrite it.
	if err := waitForProcessExit(pidToWait, 30*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to wait for old process (PID %d): %v\n", pidToWait, err)
		os.Exit(1)
	}

	// Copy ourselves to the original location
	currentPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get current executable path: %v\n", err)
		os.Exit(1)
	}

	if err := copyFile(currentPath, originalPath); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to replace original executable from %q to %q: %v\n", currentPath, originalPath, err)
		os.Exit(1)
	}

	// Continue running normally - we are now the updated application
	return true
}

// validateConfig validates the essential fields of the UpdateConfig struct.
// It returns an error if any required field is missing.
func validateConfig(config UpdateConfig) error {
	if config.GitHubOwner == "" {
		return fmt.Errorf("GitHubOwner is required")
	}
	if config.GitHubRepo == "" {
		return fmt.Errorf("GitHubRepo is required")
	}
	if config.CurrentVersion == "" {
		return fmt.Errorf("CurrentVersion is required")
	}
	if config.DataDir == "" {
		return fmt.Errorf("DataDir is required")
	}
	if config.ExecutablePath == "" {
		return fmt.Errorf("ExecutablePath is required")
	}
	if config.AssetPattern == "" {
		return fmt.Errorf("AssetPattern is required")
	}
	return nil
}

// fetchLatestRelease fetches the latest published release from the specified GitHub repository
// using the GitHub API. It includes an Authorization header if a GitHubToken is provided.
//
// It returns a pointer to a GitHubRelease struct on success or an error if the API request fails,
// returns a non-OK status code, or if JSON decoding fails.
func fetchLatestRelease(config UpdateConfig) (*GitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", config.GitHubOwner, config.GitHubRepo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if config.GitHubToken != "" {
		req.Header.Set("Authorization", "token "+config.GitHubToken)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d for %s", resp.StatusCode, url)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode GitHub release JSON: %w", err)
	}

	return &release, nil
}

// isNewerVersion compares two semantic versions (current and latest).
// It ensures that both versions are prefixed with 'v' for correct comparison using golang.org/x/mod/semver.
//
// It returns true if the latest version is semantically newer than the current version, false otherwise.
func isNewerVersion(current, latest string) bool {
	// Ensure versions start with 'v'
	if !strings.HasPrefix(current, "v") {
		current = "v" + current
	}
	if !strings.HasPrefix(latest, "v") {
		latest = "v" + latest
	}

	return semver.Compare(latest, current) > 0
}

// findMatchingAsset finds the GitHubAsset from a list of assets that matches the given pattern,
// version, operating system, and architecture.
// It constructs the expected asset name using buildAssetName and then searches for a match.
//
// It returns a pointer to the matching GitHubAsset on success, or an error if no matching asset is found.
func findMatchingAsset(assets []GitHubAsset, pattern, version, os, arch string) (*GitHubAsset, error) {
	expectedName := buildAssetName(pattern, version, os, arch)

	for _, asset := range assets {
		if asset.Name == expectedName {
			return &asset, nil
		}
	}

	return nil, fmt.Errorf("no asset found matching pattern: %s (expected: %s) for version %s, os %s, arch %s", pattern, expectedName, version, os, arch)
}

// buildAssetName constructs the expected name of the release asset based on the provided pattern,
// version, operating system, and architecture.
// It replaces placeholders ({version}, {os}, {arch}, {ext}) in the pattern with actual values.
// The {ext} placeholder is replaced with ".exe" for Windows and an empty string for other OSes.
func buildAssetName(pattern, version, os, arch string) string {
	name := pattern
	name = strings.ReplaceAll(name, "{version}", version)
	name = strings.ReplaceAll(name, "{os}", os)
	name = strings.ReplaceAll(name, "{arch}", arch)

	// Handle extension
	ext := ""
	if os == "windows" {
		ext = ".exe"
	}
	name = strings.ReplaceAll(name, "{ext}", ext)

	return name
}

// downloadAsset downloads a file from the given URL to the specified destination path.
// It creates the necessary directories if they don't exist.
// An optional GitHub token can be provided for authenticated downloads.
//
// It returns an error if the directory creation fails, the HTTP request fails,
// the download returns a non-OK status code, or if writing to the destination file fails.
func downloadAsset(url, destPath, token string) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %q: %w", destPath, err)
	}

	// Create the request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request for %q: %w", url, err)
	}

	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	// Download the file
	client := &http.Client{Timeout: 5 * time.Minute} // Allow sufficient time for large downloads
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download from %q: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download from %q failed with status %d", url, resp.StatusCode)
	}

	// Create the destination file
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file %q: %w", destPath, err)
	}
	defer out.Close()

	// Copy the data
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write downloaded data to %q: %w", destPath, err)
	}
	return nil
}

// waitForProcessExit waits for a process with the given PID to exit.
// It polls the process status periodically until it exits or the timeout is reached.
//
// It returns nil if the process exits within the timeout, or an error if the timeout is reached.
func waitForProcessExit(pid int, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for process %d to exit after %s", pid, timeout)
		case <-ticker.C:
			if !isProcessRunning(pid) {
				return nil
			}
		}
	}
}

// isProcessRunning checks if a process with the given PID is currently running.
// On Unix-like systems, it attempts to send signal 0, which checks for process existence.
// On Windows, it uses a platform-specific method to verify if the process is active.
//
// It returns true if the process is running, false otherwise.
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false // Process not found or error accessing it
	}

	// On Unix systems, we can send signal 0 to check if process exists
	if runtime.GOOS != "windows" {
		err := process.Signal(syscall.Signal(0))
		return err == nil
	}

	// On Windows, os.FindProcess always succeeds even if the process doesn't exist,
	// so we need a different approach.
	return isWindowsProcessRunning(pid)
}

// isWindowsProcessRunning specifically checks if a Windows process with the given PID is running.
// It executes the `tasklist` command and parses its output to determine process existence.
//
// It returns true if the Windows process is running, false otherwise.
func isWindowsProcessRunning(pid int) bool {
	// Use tasklist command to check if process exists
	// /FI "PID eq %d" filters by PID, /FO CSV formats the output as CSV.
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH") // /NH for No Header
	output, err := cmd.Output()
	if err != nil {
		return false // Command failed or process not found
	}

	// If the process exists, tasklist will return a line with process info.
	// If not found, it returns only "No tasks are running for the specified criteria."
	// or similar, or just headers if /NH is not used.
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	return len(lines) > 0 && !strings.Contains(lines[0], "No tasks are running")
}

// copyFile copies a file from the source path to the destination path.
// It also attempts to preserve the original file's permissions.
//
// It returns an error if any step of the copy operation fails (e.g., file open/create, write, chmod).
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %q: %w", src, err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %q: %w", dst, err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy content from %q to %q: %w", src, dst, err)
	}

	// Copy permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to get source file info %q: %w", src, err)
	}

	if err := os.Chmod(dst, sourceInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set permissions for %q: %w", dst, err)
	}

	return nil
}

// getExecutableExtension returns the platform-specific executable file extension.
// It returns ".exe" for Windows and an empty string for other operating systems.
func getExecutableExtension() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}
