# Advanced Usage

### Complex Scenarios and Customization

#### Customizing Asset Patterns

The `AssetPattern` field in `UpdateConfig` is highly flexible. It allows you to match arbitrary asset naming conventions on your GitHub releases.

*   `{version}`: The release tag name (e.g., `v1.2.3`).
*   `{os}`: The target operating system (`windows`, `linux`, `darwin`).
*   `{arch}`: The target architecture (`amd64`, `arm64`, `386`).
*   `{ext}`: The executable extension (`.exe` on Windows, empty string otherwise).

**Examples:**

| Your Asset Name              | `AssetPattern`                                  |
| :--------------------------- | :---------------------------------------------- |
| `mycli-v1.0.0-linux-amd64`   | `mycli-{version}-{os}-{arch}`                   |
| `app_1.2.3_macOS_arm64.zip`  | `app_{version}_macOS_{arch}.zip`                |
| `server-v2.0.0-windows.exe`  | `server-{version}-{os}{ext}` (if arch omitted)  |

Ensure your Go build process names the binaries exactly to match your pattern.

#### Cross-Platform Considerations

`ghupdate` automatically detects the current operating system (`runtime.GOOS`) and architecture (`runtime.GOARCH`). However, you can override these values in `UpdateConfig` if you need to download an asset for a *different* platform (e.g., an admin downloading a Windows executable from a Linux machine).

```go
config := ghupdate.UpdateConfig{
	// ... other fields
	OS:   "darwin", // Forces download for macOS
	Arch: "arm64",  // Forces download for ARM64 architecture
}
```

#### Optimization: Update Frequency

Avoid checking for updates too frequently. Excessive API calls can lead to GitHub rate limiting. Good strategies include:

*   **On Application Startup**: A single check when the application first launches.
*   **Time-Based**: Every 24 hours, or only after a certain period of application uptime.
*   **User-Initiated**: Provide a command-line flag (e.g., `--update`) or a menu option in a GUI application for users to manually trigger an update check.

### Optimization: Build Configuration

Ensure your Go build produces correctly named and permissioned binaries.

*   **Permissions**: On Unix-like systems, downloaded binaries need executable permissions. `ghupdate` automatically handles `os.Chmod(updatePath, 0755)`.
*   **Deterministic Builds**: For robust updates, ensure your build process is deterministic, so the same source code always produces the same binary for a given version.

---
### 🤖 AI Agent Guidance

```json
{
  "decisionPoints": [
    "IF asset_name_is_complex THEN construct_AssetPattern_using_placeholders {version}, {os}, {arch}, {ext}",
    "IF cross_platform_download_needed THEN override_config_OS_and_config_Arch",
    "IF managing_github_api_rate_limits THEN implement_update_checks_periodically_or_on_demand"
  ],
  "verificationSteps": [
    "Check: Test `AssetPattern` by manually constructing the expected name and verifying it exists on GitHub.",
    "Check: For cross-platform downloads, verify the downloaded asset's properties match the specified `OS` and `Arch`.",
    "Check: Monitor GitHub API usage if frequent checks are implemented."
  ],
  "quickPatterns": [
    "Pattern: Overriding OS/Arch for `UpdateConfig`\n```go\nconfig := ghupdate.UpdateConfig{\n\t// ...\n\tOS:   \"windows\", // Target Windows\n\tArch: \"amd64\",   // Target AMD64\n}\n```"
  ],
  "diagnosticPaths": [
    "Error: Update fails, but no specific error from `ghupdate` functions -> Symptom: Application restarts to old version or crashes silently during update -> Check: Verify the build process for your release assets. Is the asset truly executable? Does it match the expected OS/arch? -> Fix: Rebuild release asset, test local execution, verify `file` command output (on Unix) or properties (on Windows)."
  ]
}
```

---
*Generated using Gemini AI on 6/16/2025, 3:26:10 PM. Review and refine as needed.*