# Task-Based Guide

### 1. Integrating Self-Updates into a Go CLI Application

This is the primary use case for `ghupdate`. The key is to embed the update logic directly into your application's `main` function.

**Steps:**

1.  **Define build-time variables**: Use `go build -ldflags "-X main.Version=$(VERSION)"` to inject your application's current version dynamically.
2.  **Call `HandleUpdateMode()` early**: This is critical for the in-place replacement mechanism.
3.  **Clean up old update files**: Use `CleanupUpdate()` to maintain a tidy system.
4.  **Periodically check for updates**: Decide when to call `CheckAndPrepareUpdate()` (e.g., on startup, hourly, or via a `--check-update` command-line flag).
5.  **Apply updates conditionally**: Only call `ApplyUpdate()` if `CheckAndPrepareUpdate()` indicates a new version is available.

**Example (see Getting Started section for full code)**

```go
func main() {
	if ghupdate.HandleUpdateMode() {
		// The app just updated, new binary is running.
	}

	dataDir, _ := getAppDataDir(appName)
	_ = ghupdate.CleanupUpdate(dataDir)

	// Your application's main logic

	checkUpdates(dataDir) // Initiate update check
}
```

### 2. Configuring Update Parameters

The `ghupdate.UpdateConfig` struct dictates how `ghupdate` behaves. Proper configuration is vital for success.

**Key fields to configure:**

*   `GitHubOwner` & `GitHubRepo`: Point to your GitHub repository.
*   `CurrentVersion`: Should match your application's current semantic version (e.g., `v1.2.3`).
*   `DataDir`: A user-writable directory for temporary files (e.g., `os.UserCacheDir()`).
*   `ExecutablePath`: The full path to the currently running executable (`os.Executable()`).
*   `AssetPattern`: Matches the naming convention of your release binaries.

**Considerations:**

*   For public repositories, `GitHubToken` is optional but recommended to avoid rate limiting.
*   For private repositories, `GitHubToken` is mandatory and requires appropriate scope (e.g., `repo`).
*   `OS` and `Arch` can be left empty to default to `runtime.GOOS` and `runtime.GOARCH`.

### 3. Handling GitHub Authentication

To interact with private GitHub repositories or to increase API rate limits for public ones, provide a GitHub Personal Access Token (PAT).

**Steps:**

1.  **Generate a PAT**: Go to GitHub -> Settings -> Developer settings -> Personal access tokens. Grant `repo` scope for private repos, or `public_repo` for public ones.
2.  **Store the PAT securely**: Do **not** hardcode it in your source code. Use environment variables.
3.  **Pass to `UpdateConfig`**: Retrieve the token from the environment variable and assign it to `config.GitHubToken`.

**Example:**

```go
config := ghupdate.UpdateConfig{
	// ... other fields
	GitHubToken: os.Getenv("GITHUB_TOKEN"), // Reads from GITHUB_TOKEN environment variable
	// ...
}
```

### 4. Releasing and Versioning Best Practices

Effective release management is crucial for a smooth update experience.

*   **Semantic Versioning**: Always tag your GitHub releases with semantic versions (e.g., `v1.0.0`, `v1.0.1`, `v2.0.0`). `ghupdate` relies on `golang.org/x/mod/semver` for comparisons.
*   **Dynamic Version Injection**: Inject `CurrentVersion` into your Go binary at build time using `-ldflags`. This prevents discrepancies between your code and the actual binary version.

    ```bash
    # Example Makefile snippet for building:
    VERSION := $(shell git describe --tags --abbrev=0 --always)
    BUILD_DATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

    build:
    	go build -ldflags "-X main.Version=$(VERSION) -X main.BuildDate=$(BUILD_DATE)" -o dist/my-app ./main.go
    ```
*   **Consistent Asset Naming**: Ensure your release assets uploaded to GitHub consistently follow the pattern you define in `AssetPattern`.

    *Correct*: `my-app-v1.0.0-linux-amd64`, `my-app-v1.0.0-windows-amd64.exe`
    *Incorrect*: `my-app_linux_amd64`, `myapp.zip` (if `AssetPattern` expects `myapp-{version}-{os}-{arch}{ext}`)


---
### 🤖 AI Agent Guidance

```json
{
  "decisionPoints": [
    "IF requiring_authenticated_github_access THEN set_environment_variable GITHUB_TOKEN",
    "IF using_semantic_release_for_versioning THEN configure_ldflags_to_inject_version",
    "IF releasing_new_application_version THEN ensure_release_asset_names_match_AssetPattern"
  ],
  "verificationSteps": [
    "Check: `CurrentVersion` in `UpdateConfig` matches the version embedded in the binary.",
    "Check: Release asset names on GitHub correspond exactly to `AssetPattern`.",
    "Check: `GITHUB_TOKEN` environment variable is set if private repo or rate limit is a concern."
  ],
  "quickPatterns": [
    "Pattern: Go ldflags for version injection\n```bash\ngo build -ldflags \"-X 'main.Version=$(git describe --tags --abbrev=0)'\" -o myapp .\n```",
    "Pattern: Configuring `UpdateConfig` for private repo\n```go\nconfig := ghupdate.UpdateConfig{\n\tGitHubOwner:    \"myorg\",\n\tGitHubRepo:     \"my-private-app\",\n\tGitHubToken:    os.Getenv(\"GITHUB_TOKEN\"), // Must be set in env\n\tCurrentVersion: Version,\n\tDataDir:        dataDir,\n\tExecutablePath: executablePath,\n\tAssetPattern:   \"my-private-app-{version}-{os}-{arch}{ext}\",\n}\n```"
  ],
  "diagnosticPaths": [
    "Error: `CurrentVersion` in `UpdateInfo` is 'dev' or wrong -> Symptom: Application does not report correct version, or updates don't trigger when they should -> Check: Verify `ldflags` in your build script. Ensure `main.Version` variable is correctly targeted. -> Fix: Correct `ldflags` to embed the actual version string.",
    "Error: `fetchLatestRelease` returns 403 Forbidden -> Symptom: Cannot access private repository or hit rate limit -> Check: Ensure `GITHUB_TOKEN` is set in environment and passed to `UpdateConfig`. For private repos, confirm PAT has `repo` scope. -> Fix: Generate new PAT with correct permissions, or set existing PAT environment variable."
  ]
}
```

---
*Generated using Gemini AI on 6/16/2025, 3:26:10 PM. Review and refine as needed.*