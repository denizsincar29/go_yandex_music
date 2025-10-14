# Release Process

This document describes how to create releases for the Yandex Music Player project.

## Overview

The project uses a combination of GitHub Actions, GoReleaser, and version tracking through `version.txt` to automate the release process.

## Release Methods

### Method 1: GitHub Actions (Recommended)

The easiest way to create a release is using the GitHub Actions workflow:

1. Update `version.txt` with the new version and changelog:
   ```
   1.0.3
   Added systemd service creator script, URL parameter support, and updated branding
   ```

2. Commit and push changes:
   ```bash
   git add version.txt
   git commit -m "Bump version to 1.0.3"
   git push
   ```

3. Trigger the release workflow:
   - Go to: https://github.com/denizsincar29/go_yandex_music/actions/workflows/release.yml
   - Click "Run workflow"
   - Leave version empty to use `version.txt`, or specify a version
   - Click "Run workflow"

The workflow will:
- Read the version from `version.txt`
- Generate a changelog based on commits since the last release
- Use the changelog from `version.txt` if provided (lines after version number)
- Create and push a git tag (e.g., `v1.0.3`)
- Run GoReleaser to build binaries and create a GitHub release

### Method 2: Python Script (Manual)

Use the `release.py` script for manual releases:

1. Update `version.txt`:
   ```
   1.0.3
   Added systemd service creator script, URL parameter support, and updated branding
   ```

2. Run the script:
   ```bash
   python release.py
   ```

The script will:
- Check if the repository is clean
- Compare the version in `version.txt` with the latest git tag
- Create and push a new tag if the version is greater
- Use the changelog from `version.txt` as the tag message

### Method 3: Manual Tag (Advanced)

Create tags manually and let GoReleaser handle the release:

```bash
git tag -a v1.0.3 -m "Release v1.0.3"
git push origin v1.0.3
```

## version.txt Format

The `version.txt` file supports a simple format:

```
<version>
<changelog description (optional)>
```

Example:
```
1.0.3
Added systemd service creator script, URL parameter support, and updated branding
```

- **First line**: Version number (e.g., `1.0.3`)
- **Subsequent lines**: Optional changelog description that will be used in the release notes

## Changelog Generation

The GitHub Actions workflow automatically generates changelogs:

1. **From version.txt**: If you add a changelog description in `version.txt` (lines after the version), it will be included in the release notes

2. **From commits**: The workflow automatically includes a list of commits since the last release

3. **Combined**: Both sources are combined to create comprehensive release notes

## Release Artifacts

GoReleaser creates the following artifacts:

- **Windows**: `go_yandex_music_Windows_x86_64.zip`
- **macOS**: `go_yandex_music_Darwin_x86_64.tar.gz`
- **macOS (ARM)**: `go_yandex_music_Darwin_arm64.tar.gz`

All artifacts are automatically attached to the GitHub release.

## Troubleshooting

### Tag already exists
If the tag already exists, the workflow will skip tag creation and use the existing tag.

### Repository is dirty
Make sure all changes are committed before creating a release:
```bash
git status
git add .
git commit -m "Prepare release"
git push
```

### GoReleaser fails
Check the workflow logs for detailed error messages. Common issues:
- Go modules not properly configured
- Build errors in the code
- Missing dependencies

## Examples

### Simple version bump
```
1.0.4
```

### Version with detailed changelog
```
1.0.4
## New Features
- Added systemd service creator script
- Implemented URL parameter support for search and album autoplay
- Updated branding and footer

## Bug Fixes
- Fixed audio playback issues
- Improved error handling
```

### Version with markdown changelog
```
1.0.4
### What's New
- **Systemd Support**: Auto-generate systemd service files with `create_systemd_service.sh`
- **Deep Linking**: Use URL parameters like `?search=query&autoplay=1`
- **Branding Update**: New footer with creator attribution

### Technical Improvements
- Enhanced error handling in audio player
- Improved URL parameter parsing
- Better logging configuration
```
