# 🚀 Releasing with GoReleaser

Auto MCP uses [GoReleaser](https://goreleaser.com/) to build and publish releases. This automates the build process for multiple platforms and creates versioned artifacts like binaries and container images.

## Automated GitHub Releases

This project uses GitHub Actions to automate the release process. When you push a new tag starting with 'v' (e.g., v0.1.0), the workflow will automatically:

1. Build binaries for all supported platforms (Linux, macOS, Windows)
2. Create GitHub Container Registry images (ghcr.io)
3. Create a GitHub release with changelog and artifacts

To create a new release:

```bash
# Update your code and commit changes
git commit -sm "Your changes"

# Create and push a new version tag
git tag -s v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

You can also manually trigger a release from the GitHub Actions tab.

## Required Permissions

The GitHub Actions workflow uses the built-in `GITHUB_TOKEN` which automatically has the necessary permissions to:

- Create releases
- Push container images to GitHub Container Registry (ghcr.io)

No additional secrets need to be configured.

## Manual Release Process

If you prefer to release manually:

1. **Install GoReleaser** (if not already installed):

   ```bash
   go install github.com/goreleaser/goreleaser@latest
   ```

   Or use the `make deps` command which includes GoReleaser installation.

2. **Create a release**:

   ```bash
   make release
   ```

3. **Test a release locally** without publishing:
   ```bash
   make release-snapshot
   ```

## Available Release Artifacts

When you run a release, GoReleaser creates:

- Cross-compiled binaries for Linux, macOS, and Windows (amd64 and arm64)
- Container images published to GitHub Container Registry (ghcr.io/brizzai/auto-mcp)
- Checksums for verification

## Version Information

The CLI exposes version information through the `--version` or `-v` flag:

```bash
auto-mcp --version
```
