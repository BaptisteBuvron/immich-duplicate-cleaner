# Immich Duplicate Cleaner

[![CI](https://github.com/BaptisteBuvron/immich-duplicate-cleaner/actions/workflows/ci.yml/badge.svg)](https://github.com/BaptisteBuvron/immich-duplicate-cleaner/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/BaptisteBuvron/immich-duplicate-cleaner)](https://goreportcard.com/report/github.com/BaptisteBuvron/immich-duplicate-cleaner)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A powerful command-line tool for managing duplicate assets in [Immich](https://immich.app/). This tool synchronizes albums across duplicate photos and videos, and can automatically remove lower-quality duplicates based on intelligent quality comparison.

## ✨ Features

- 🔄 **Album Synchronization**: Automatically synchronizes albums across all duplicate assets
- 🎯 **Smart Deduplication**: Intelligently selects the best quality asset based on:
  - File size (larger files typically indicate better quality)
  - Original filename preservation (avoids auto-generated names like IMG_*, DSC_*)
  - Creation date (keeps the original)
- 🔒 **Safe Operations**: 
  - Dry-run mode to preview changes
  - Confirmation prompts before deletion
  - Detailed logging of all actions
- ⚡ **Easy to Use**: Simple command-line interface with intuitive flags
- 🌐 **Cross-Platform**: Works on Linux, macOS, and Windows

## 📋 Prerequisites

- [Go](https://golang.org/dl/) 1.21 or later
- An [Immich](https://immich.app/) instance (v1.91.0 or later recommended)
- An Immich API key

## 🚀 Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/BaptisteBuvron/immich-duplicate-cleaner.git
cd immich-duplicate-cleaner

# Build the binary
go build -o immich-duplicate-cleaner .

# Run the tool
./immich-duplicate-cleaner --help
```

### Pre-built Binaries

Download the latest pre-built binaries from the [Releases](https://github.com/BaptisteBuvron/immich-duplicate-cleaner/releases) page.

## 🔑 Getting Your API Key

1. Log in to your Immich instance
2. Go to **Account Settings** → **API Keys**
3. Click **New API Key**
4. Give it a name (e.g., "Duplicate Cleaner") and save the key

## 📖 Usage

### Basic Usage

Synchronize albums across duplicates:

```bash
./immich-duplicate-cleaner --url http://your-immich-instance:2283 --api-key YOUR_API_KEY
```

### Preview Changes (Dry Run)

Preview what changes would be made without actually making them:

```bash
./immich-duplicate-cleaner -u http://localhost:2283 -k YOUR_API_KEY --dry-run
```

### Auto-Delete Duplicates

Synchronize albums and automatically delete lower-quality duplicates:

```bash
./immich-duplicate-cleaner -u http://localhost:2283 -k YOUR_API_KEY --auto-delete
```

### Skip Confirmation Prompts

Auto-delete without confirmation (use with caution!):

```bash
./immich-duplicate-cleaner -u http://localhost:2283 -k YOUR_API_KEY -d -y
```

### Verbose Logging

Enable detailed logging for troubleshooting:

```bash
./immich-duplicate-cleaner -u http://localhost:2283 -k YOUR_API_KEY -v
```

## 🎛️ Command-Line Flags Reference

### Required Flags

| Flag | Shorthand | Parameter | Description | Example |
|------|-----------|-----------|-------------|---------|
| `--url` | `-u` | `<string>` | Immich server URL (with http:// or https://) | `--url http://192.168.1.39:2283` |
| `--api-key` | `-k` | `<string>` | Immich API key for authentication | `--api-key YOUR_API_KEY` |

### Optional Flags

| Flag | Shorthand | Parameter | Default | Description |
|------|-----------|-----------|---------|-------------|
| `--auto-delete` | `-d` | none | `false` | Enable automatic deletion of lower-quality duplicates after album synchronization |
| `--dry-run` | | none | `false` | Preview all actions without making any changes to your Immich instance |
| `--yes` | `-y` | none | `false` | Skip all confirmation prompts (use with caution, especially with `--auto-delete`) |
| `--verbose` | `-v` | none | `false` | Enable detailed logging including album assignments and asset details |
| `--version` | | none | - | Display version information and exit |
| `--help` | `-h` | none | - | Show help message with usage examples and exit |

### Flag Combinations

| Combination | Behavior |
|-------------|----------|
| `--url --api-key` | Synchronize albums only (safe, no deletions) |
| `--url --api-key --dry-run` | Preview synchronization without making changes |
| `--url --api-key --auto-delete` | Synchronize albums + delete duplicates (prompts for each group) |
| `--url --api-key --auto-delete --yes` | Synchronize albums + delete duplicates without prompts ⚠️ |
| `--url --api-key --auto-delete --dry-run` | Preview which duplicates would be deleted |
| `--url --api-key --verbose` | Show detailed information during synchronization |

## 🔍 How It Works

### Album Synchronization

1. Fetches all duplicate groups from your Immich instance
2. For each duplicate group:
   - Identifies all albums containing any asset from the group
   - Ensures all duplicates are added to all albums
3. Logs all synchronization actions

### Quality Comparison Algorithm

When `--auto-delete` is enabled, the tool selects the best quality asset using this priority:

1. **File Size**: Larger files are preferred (better quality/resolution)
2. **Original Filename**: Files with custom names are preferred over auto-generated names (IMG_*, DSC_*, etc.)
3. **Creation Date**: Earlier creation dates are preferred (original photo)

The asset with the highest priority is kept; all others are deleted.

## 📊 Example Output

```
🚀 Starting Immich Duplicate Cleaner v1.0.0
🔍 Fetching duplicate groups...
✅ Found 3 duplicate group(s)

📁 Processing group 1/3 (2 assets)
📋 Current album assignments:
   Asset 12345678: [Vacation 2023, Best Photos]
   Asset 87654321: [Vacation 2023]
✅ Added 1 asset(s) to album abcd1234
✨ Synchronized 1 asset(s) across albums

🔍 Analyzing quality of 2 duplicate(s)...
🏆 Best quality asset: 12345678
   Size: 3145728 bytes, Resolution: 4032x3024
🗑️  Deleted duplicate asset 87654321

🎉 Processing complete!
```

## ⚠️ Safety Considerations

- **Backup First**: Always backup your Immich database before performing bulk operations
- **Test with Dry Run**: Use `--dry-run` to preview changes before applying them
- **Review Confirmation**: The tool will ask for confirmation before deleting duplicates (unless `--yes` is used)
- **Start Small**: Test on a small set of duplicates first to ensure the tool works as expected

## 🛠️ Development

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run tests with coverage
go test -v -coverprofile=coverage.out ./...

# View coverage report
go tool cover -html=coverage.out
```

### Running Linters

```bash
# Install golangci-lint (if not already installed)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linters
golangci-lint run
```

### Building for Multiple Platforms

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o immich-duplicate-cleaner-linux-amd64

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o immich-duplicate-cleaner-darwin-arm64

# Windows AMD64
GOOS=windows GOARCH=amd64 go build -o immich-duplicate-cleaner-windows-amd64.exe
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

### Guidelines

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Write tests for your changes
4. Ensure all tests pass (`go test ./...`)
5. Ensure code is properly formatted (`gofmt -s -w .`)
6. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
7. Push to the branch (`git push origin feature/AmazingFeature`)
8. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Immich](https://immich.app/) - The amazing self-hosted photo and video management solution
- The Go community for excellent tooling and libraries

## 🐛 Issues & Support

If you encounter any issues or have questions:

1. Check the [FAQ](#-how-it-works) section
2. Search [existing issues](https://github.com/BaptisteBuvron/immich-duplicate-cleaner/issues)
3. Create a [new issue](https://github.com/BaptisteBuvron/immich-duplicate-cleaner/issues/new) with:
   - Your Immich version
   - Tool version (`./immich-duplicate-cleaner --version`)
   - Relevant logs (use `--verbose` flag)
   - Steps to reproduce

## 📊 Project Status

This project is actively maintained. Star ⭐ the repository to show your support!

---

**Made with ❤️ for the Immich community**