# Core Operations

### Essential Functions

The `ghupdate` library provides four primary functions that orchestrate the entire self-update process:

*   [`ghupdate.HandleUpdateMode()`](#method:HandleUpdateMode): Called on application startup to perform the actual executable replacement if an update was initiated.
*   [`ghupdate.CleanupUpdate(dataDir string)`](#method:CleanupUpdate): Removes temporary update files.
*   [`ghupdate.CheckAndPrepareUpdate(config UpdateConfig)`](#method:CheckAndPrepareUpdate): Checks for new releases on GitHub, downloads the correct asset, and prepares it for application.
*   [`ghupdate.ApplyUpdate(config UpdateConfig)`](#method:ApplyUpdate): Triggers the replacement process by spawning the newly downloaded executable.

### Workflows with Decision Trees

The typical workflow for integrating `ghupdate` is a sequence of calls that ensure robustness and atomicity.

#### Application Startup Flow

This workflow describes the critical steps your application should take immediately upon starting.

1.  **Check for Update Mode**: The application first determines if it was launched as part of an update process (`ghupdate.HandleUpdateMode`). If it was, it performs the file replacement and then continues execution as the newly updated application.
2.  **Clean Temporary Files**: Regardless of update mode, temporary files from previous update attempts are cleaned up (`ghupdate.CleanupUpdate`).
3.  **Normal Application Logic**: The application proceeds with its intended functionality.

#### Update Initiation Flow

This workflow describes how your application proactively checks for and applies updates.

1.  **Check for New Version**: The application queries GitHub for the latest release and compares it against its current version (`ghupdate.CheckAndPrepareUpdate`). If a newer version is found, the asset is downloaded.
2.  **Apply Update**: If an update is prepared, the application initiates the replacement (`ghupdate.ApplyUpdate`), which causes the current process to exit and the newly downloaded executable to take over.



---
### 🤖 AI Agent Guidance

```json
{
  "decisionPoints": [
    "IF ghupdate.HandleUpdateMode returns true THEN log \"Update applied, resuming\" AND proceed with updated application flow",
    "IF ghupdate.HandleUpdateMode returns false THEN proceed to ghupdate.CleanupUpdate(dataDir)",
    "IF ghupdate.CleanupUpdate(dataDir) returns error THEN log \"Cleanup warning\" AND continue",
    "IF ghupdate.CheckAndPrepareUpdate(config) returns nil (no updateInfo) AND nil (error) THEN log \"No update available\" AND continue_normal_app_flow",
    "IF ghupdate.CheckAndPrepareUpdate(config) returns updateInfo AND nil (error) THEN log \"Update available\" AND call ghupdate.ApplyUpdate(config)",
    "IF ghupdate.CheckAndPrepareUpdate(config) returns nil AND error THEN log \"Error checking for update\" AND continue_normal_app_flow",
    "IF ghupdate.ApplyUpdate(config) returns error THEN log \"Error applying update\" AND exit_with_failure"
  ],
  "verificationSteps": [
    "Check: `ghupdate.HandleUpdateMode()` is always executed first.",
    "Check: `ghupdate.CleanupUpdate()` is called once per application launch.",
    "Check: `ghupdate.CheckAndPrepareUpdate()` is called periodically or on demand.",
    "Check: `ghupdate.ApplyUpdate()` is only called if `CheckAndPrepareUpdate()` returns a valid `UpdateInfo`."
  ],
  "quickPatterns": [
    "Pattern: Full update cycle within application\n```go\nfunc main() {\n\tif ghupdate.HandleUpdateMode() {\n\t\tfmt.Println(\"Update complete.\")\n\t\t// New version is running, continue normal operations.\n\t}\n\n\tdataDir, _ := os.UserCacheDir()\n\tdataDir = filepath.Join(dataDir, \"my-app\")\n\t_ = os.MkdirAll(dataDir, 0755)\n\n\tif err := ghupdate.CleanupUpdate(dataDir); err != nil {\n\t\tlog.Printf(\"Cleanup warning: %v\\n\", err)\n\t}\n\n\t// Core application logic...\n\n\texecPath, _ := os.Executable()\n\tconfig := ghupdate.UpdateConfig{\n\t\tGitHubOwner:    \"owner\",\n\t\tGitHubRepo:     \"repo\",\n\t\tCurrentVersion: \"v1.0.0\",\n\t\tDataDir:        dataDir,\n\t\tExecutablePath: execPath,\n\t\tAssetPattern:   \"my-app-{version}-{os}-{arch}{ext}\",\n\t}\n\n\tupdateInfo, err := ghupdate.CheckAndPrepareUpdate(config)\n\tif err != nil {\n\t\tlog.Printf(\"Error checking update: %v\", err)\n\t} else if updateInfo != nil {\n\t\tfmt.Printf(\"Update found: %s -> %s\\n\", updateInfo.CurrentVersion, updateInfo.LatestVersion)\n\t\t// Notify user, then apply.\n\t\tif err := ghupdate.ApplyUpdate(config); err != nil {\n\t\t\tlog.Fatalf(\"Error applying update: %v\", err)\n\t\t}\n\t\t// This line is not reached on success.\n\t}\n}\n```"
  ],
  "diagnosticPaths": [
    "Error: `CheckAndPrepareUpdate` fails with \"failed to fetch latest release\" -> Symptom: Network error, invalid GitHubOwner/GitHubRepo, or GitHub API rate limit -> Check: Verify internet connectivity, check `GitHubOwner` and `GitHubRepo` names, ensure `GITHUB_TOKEN` is set for private repos or to avoid rate limits. -> Fix: Correct configuration, set `GITHUB_TOKEN` environment variable.",
    "Error: `CheckAndPrepareUpdate` fails with \"no asset found matching pattern\" -> Symptom: Downloaded asset name does not match expected pattern -> Check: Compare `AssetPattern` with actual GitHub release asset names, including `{version}`, `{os}`, `{arch}`, `{ext}` placeholders. Ensure release is not a draft or pre-release unless intended. -> Fix: Adjust `AssetPattern` to exactly match your release binaries."
  ]
}
```

---
*Generated using Gemini AI on 6/16/2025, 3:26:10 PM. Review and refine as needed.*