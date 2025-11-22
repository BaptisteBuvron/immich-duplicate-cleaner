# Release Process Guide

This guide explains how to create a new release for Immich Duplicate Cleaner.

## Prerequisites

- All changes must be committed and pushed to master
- All CI tests must be passing
- You have write access to the repository

## Creating a Release

### 1. Update Version (Optional)

If you want to update the version in the code:

```bash
# Edit main.go and update the version constant
# const version = "1.0.0"  →  const version = "1.1.0"
```

### 2. Create and Push a Git Tag

```bash
# Make sure you're on master and up to date
git checkout master
git pull

# Create an annotated tag with semantic versioning (v1.0.0, v1.1.0, etc.)
git tag -a v1.0.0 -m "Release v1.0.0: Initial stable release"

# Push the tag to GitHub
git push origin v1.0.0
```

### 3. Automatic Release Creation

Once you push the tag, GitHub Actions will automatically:

1. ✅ Create a new GitHub Release
2. ✅ Generate a changelog from git commits
3. ✅ Build binaries for:
   - Linux (amd64, arm64)
   - macOS (amd64, arm64 for Apple Silicon)
   - Windows (amd64)
4. ✅ Upload all binaries to the release
5. ✅ Generate SHA256 checksums for each binary

### 4. Monitor the Release

1. Go to: https://github.com/BaptisteBuvron/immich-duplicate-cleaner/actions
2. Watch the "Release" workflow
3. Once complete, check: https://github.com/BaptisteBuvron/immich-duplicate-cleaner/releases

## Semantic Versioning

Follow [Semantic Versioning](https://semver.org/):

- **v1.0.0** - Initial release
- **v1.0.1** - Bug fixes only
- **v1.1.0** - New features (backward compatible)
- **v2.0.0** - Breaking changes

## Release Checklist

Before creating a release:

- [ ] All tests pass (`go test ./...`)
- [ ] Code is linted (`go vet ./...`)
- [ ] README is up to date
- [ ] CHANGELOG or commits describe changes
- [ ] Version number follows semantic versioning

## Example Workflow

```bash
# 1. Make your changes
git add .
git commit -m "feat: add new awesome feature"
git push

# 2. Wait for CI to pass
# Check: https://github.com/BaptisteBuvron/immich-duplicate-cleaner/actions

# 3. Create and push tag
git tag -a v1.1.0 -m "Release v1.1.0: Add awesome feature"
git push origin v1.1.0

# 4. Wait for release workflow
# Binaries will automatically be built and attached to the release!
```

## Troubleshooting

### Tag already exists

```bash
# Delete local tag
git tag -d v1.0.0

# Delete remote tag
git push origin :refs/tags/v1.0.0

# Recreate and push
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

### Release workflow failed

1. Check the Actions tab for error messages
2. Fix any issues in the code
3. Delete and recreate the tag (see above)
4. Push the tag again to retrigger the workflow

## Manual Release (if needed)

If the automatic release fails, you can create a release manually:

1. Go to https://github.com/BaptisteBuvron/immich-duplicate-cleaner/releases/new
2. Choose your tag
3. Fill in the release notes
4. Build binaries locally:

```bash
# Linux amd64
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o immich-duplicate-cleaner-linux-amd64 .

# macOS arm64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o immich-duplicate-cleaner-darwin-arm64 .

# Windows amd64
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o immich-duplicate-cleaner-windows-amd64.exe .
```

5. Upload the binaries to the release
