# Problem Solving

### Troubleshooting Common Issues

This section addresses common problems encountered when integrating and using `ghupdate`.

#### 1. "Error checking for update: failed to find matching asset"

*   **Cause**: The `AssetPattern` in your `UpdateConfig` does not accurately describe the file names of the assets in your GitHub release. This is the most common issue.
*   **Diagnosis**: 
    1.  Go to your GitHub release page and note the exact file names of your binaries (e.g., `mycli-v1.0.0-linux-amd64`, `mycli-v1.0.0-windows-amd64.exe`).
    2.  Compare these names against the pattern you provided in `config.AssetPattern`. Remember `ghupdate` replaces `{version}`, `{os}`, `{arch}`, `{ext}` placeholders.
    3.  Ensure the release is *not* a `draft` or `prerelease` (unless you're using a custom GitHub API call to fetch those, which `ghupdate` doesn't do by default for `releases/latest`).
*   **Solution**: Adjust your `AssetPattern` to precisely match the naming convention of your uploaded release assets. For example, if your asset is named `my-app-v1.2.3-linux-amd64`, your pattern should be `my-app-{version}-{os}-{arch}`.

#### 2. "GitHub API returned status 403" or "API rate limit exceeded"

*   **Cause**: You have hit GitHub's API rate limits (for unauthenticated requests, typically 60 requests per hour), or you are trying to access a private repository without providing a valid `GitHubToken`.
*   **Diagnosis**: 
    1.  Check if you're making too many requests (e.g., checking for updates on every application launch for many users).
    2.  Verify if `os.Getenv("GITHUB_TOKEN")` is returning an empty string when it shouldn't be.
    3.  If accessing a private repository, ensure your GitHub PAT has the `repo` scope enabled.
*   **Solution**: 
    1.  Provide a `GitHubToken` in your `UpdateConfig`. It's good practice even for public repositories to increase the rate limit to 5000 requests per hour.
    2.  Adjust your update check frequency to be less aggressive (e.g., once a day, or on user command).

#### 3. "Permission denied" during `ApplyUpdate` or `CleanupUpdate`

*   **Cause**: The application lacks the necessary write permissions to the `DataDir` (for temporary files) or the `ExecutablePath` (for the final replacement).
*   **Diagnosis**: 
    1.  Check the permissions of the directory specified in `config.DataDir`. This should typically be a user-writable location like `os.UserCacheDir()` or `os.UserConfigDir()`.
    2.  Check the permissions of the `ExecutablePath`. If your application is installed in a system-wide location (e.g., `/usr/local/bin` on Linux, `Program Files` on Windows), standard users might not have write permissions to overwrite the binary directly.
*   **Solution**: 
    1.  Ensure `DataDir` is set to a user-writable location, such as one returned by `os.UserCacheDir()`.
    2.  If the application is installed in a privileged location, consider deploying updates through a system package manager or an installer that handles privilege elevation, as `ghupdate` is primarily designed for user-writable locations. For CLI tools, encouraging users to install into `~/bin` or similar user-owned paths is common.

#### 4. Application Doesn't Update, or `HandleUpdateMode()` Isn't Called

*   **Cause**: The `ghupdate.HandleUpdateMode()` function is not being called as the *very first* executable statement in your `main` function, or the `ApplyUpdate` process failed to correctly spawn the new process with the required arguments.
*   **Diagnosis**: 
    1.  Inspect your `main` function to confirm `if ghupdate.HandleUpdateMode() { ... }` is the first line of code.
    2.  Check `ApplyUpdate`'s error logs (if any are returned) from the previous run to see if the new process failed to start.
*   **Solution**: Always ensure `if ghupdate.HandleUpdateMode() { ... }` is the absolute first thing that happens in your `main()` function, even before any logging or path determination. `ApplyUpdate` is designed to be robust in spawning the new process; issues here usually point to fundamental system problems.

---
### 🤖 AI Agent Guidance

```json
{
  "decisionPoints": [
    "IF error_is_about_missing_asset THEN check_AssetPattern_against_release_names",
    "IF error_is_403_or_rate_limit THEN check_GITHUB_TOKEN_and_update_frequency",
    "IF error_is_permission_denied THEN check_DataDir_and_ExecutablePath_permissions",
    "IF update_fails_silently THEN verify_HandleUpdateMode_position_in_main_function"
  ],
  "verificationSteps": [
    "Check: Manually download release asset and verify its name against generated pattern string.",
    "Check: Inspect `GITHUB_TOKEN` environment variable value in runtime environment.",
    "Check: Run `ls -ld <DataDir>` and `ls -l <ExecutablePath>` to inspect permissions.",
    "Check: Review source code of `main` function to ensure `ghupdate.HandleUpdateMode()` is the very first statement."
  ],
  "quickPatterns": [
    "Pattern: Debugging `AssetPattern`\n```go\n// In ghupdate source (for debugging), or temporarily in your app:\nfmt.Printf(\"Expected asset name: %s\\n\", buildAssetName(config.AssetPattern, release.TagName, targetOS, targetArch))\n// Compare this output with actual GitHub asset names.\n```"
  ],
  "diagnosticPaths": [
    "Error: \"failed to find matching asset\" -> Symptom: Application cannot locate release binary -> Check: Verify `AssetPattern` matches GitHub asset exact names. Check for typos in `GitHubOwner` or `GitHubRepo`. Ensure release is not a draft/prerelease. -> Fix: Adjust `AssetPattern` or release asset names. Confirm GitHub repository details.",
    "Error: `os.Executable()` fails or returns unexpected path -> Symptom: `ExecutablePath` is incorrect, causing update failure -> Check: Run `fmt.Println(os.Executable())` to confirm the path. -> Fix: Ensure application is launched in a standard manner, or adjust how `ExecutablePath` is determined if using unusual launch methods.",
    "Error: `log.Fatalf` on `ApplyUpdate` -> Symptom: Application immediately exits during update attempt -> Check: Verify that `CheckAndPrepareUpdate` successfully downloaded a file to `DataDir`. Is the downloaded file actually executable? -> Fix: Investigate `CheckAndPrepareUpdate` errors first, then ensure the downloaded asset is valid and executable for the target system."
  ]
}
```

---
*Generated using Gemini AI on 6/16/2025, 3:26:10 PM. Review and refine as needed.*