// Package main provides a tool for managing duplicate assets in Immich.
// It synchronizes albums across duplicate assets and optionally removes
// lower-quality duplicates automatically.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	// API endpoint paths
	duplicatesEndpoint = "/api/duplicates"
	albumsEndpoint     = "/api/albums"
	assetsEndpoint     = "/api/assets"

	// HTTP timeouts
	defaultTimeout = 30 * time.Second

	// Version information
	version = "1.0.0"
)

// Config holds the application configuration
type Config struct {
	ImmichURL  string // Base URL of the Immich instance
	APIKey     string // API key for authentication
	AutoDelete bool   // Whether to automatically delete lower-quality duplicates
	DryRun     bool   // Preview mode - don't make any changes
	Yes        bool   // Skip confirmation prompts
	Verbose    bool   // Enable verbose logging
}

// DuplicateAsset represents a single asset in a duplicate group
type DuplicateAsset struct {
	ID string `json:"id"`
}

// DuplicateGroup represents a group of duplicate assets
type DuplicateGroup struct {
	DuplicateID string           `json:"duplicateId"`
	Assets      []DuplicateAsset `json:"assets"`
}

// Album represents an Immich album
type Album struct {
	ID         string  `json:"id"`
	AlbumName  string  `json:"albumName"`
	Assets     []Asset `json:"assets,omitempty"`
	AssetCount int     `json:"assetCount"`
}

// Asset represents a media asset with its metadata
type Asset struct {
	ID               string    `json:"id"`
	OriginalFileName string    `json:"originalFileName,omitempty"`
	ExifInfo         *ExifInfo `json:"exifInfo,omitempty"`
	FileCreatedAt    time.Time `json:"fileCreatedAt,omitempty"`
}

// ExifInfo contains EXIF metadata for an asset
type ExifInfo struct {
	FileSizeInByte int64 `json:"fileSizeInByte,omitempty"`
	ImageWidth     int   `json:"imageWidth,omitempty"`
	ImageHeight    int   `json:"imageHeight,omitempty"`
}

// AssetDetails represents detailed information about an asset
type AssetDetails struct {
	ID               string    `json:"id"`
	OriginalFileName string    `json:"originalFileName"`
	ExifInfo         *ExifInfo `json:"exifInfo"`
	FileCreatedAt    time.Time `json:"fileCreatedAt"`
}

// AddAssetsRequest is the payload for adding assets to an album
type AddAssetsRequest struct {
	IDs []string `json:"ids"`
}

// HTTPClient interface for easier testing
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

var (
	// Global HTTP client with timeout
	httpClient HTTPClient = &http.Client{Timeout: defaultTimeout}
)

func main() {
	// Parse command-line flags
	config := parseFlags()

	// Validate configuration
	if err := validateConfig(config); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	logInfo("🚀 Starting Immich Duplicate Cleaner v%s", version)
	if config.DryRun {
		logWarning("⚠️  DRY RUN MODE - No changes will be made")
	}

	// Fetch all duplicate groups
	logInfo("🔍 Fetching duplicate groups...")
	duplicates, err := getDuplicates(config)
	if err != nil {
		log.Fatalf("Failed to fetch duplicates: %v", err)
	}

	logInfo("✅ Found %d duplicate group(s)", len(duplicates))

	if len(duplicates) == 0 {
		logInfo("🎉 No duplicates found - nothing to do!")
		return
	}

	// Process each duplicate group
	for i, group := range duplicates {
		if err := processDuplicateGroup(config, i+1, len(duplicates), group); err != nil {
			logError("Failed to process group %d: %v", i+1, err)
			continue
		}
	}

	logInfo("\n🎉 Processing complete!")
	if !config.AutoDelete {
		logInfo("💡 Tip: Use --auto-delete flag to automatically remove lower-quality duplicates")
	}
}

// parseFlags parses command-line flags and returns a Config
func parseFlags() *Config {
	config := &Config{}

	flag.StringVar(&config.ImmichURL, "url", "", "Immich server URL (e.g., http://localhost:2283)")
	flag.StringVar(&config.ImmichURL, "u", "", "Immich server URL (shorthand)")
	flag.StringVar(&config.APIKey, "api-key", "", "Immich API key")
	flag.StringVar(&config.APIKey, "k", "", "Immich API key (shorthand)")
	flag.BoolVar(&config.AutoDelete, "auto-delete", false, "Automatically delete lower-quality duplicates")
	flag.BoolVar(&config.AutoDelete, "d", false, "Automatically delete lower-quality duplicates (shorthand)")
	flag.BoolVar(&config.DryRun, "dry-run", false, "Preview actions without making changes")
	flag.BoolVar(&config.Yes, "yes", false, "Skip confirmation prompts")
	flag.BoolVar(&config.Yes, "y", false, "Skip confirmation prompts (shorthand)")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&config.Verbose, "v", false, "Enable verbose logging (shorthand)")

	showVersion := flag.Bool("version", false, "Show version information")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Immich Duplicate Cleaner v%s\n\n", version)
		fmt.Fprintf(os.Stderr, "A tool to synchronize albums across duplicate assets and optionally remove duplicates.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s [flags]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Synchronize albums only (dry run)\n")
		fmt.Fprintf(os.Stderr, "  %s --url http://localhost:2283 --api-key YOUR_KEY --dry-run\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Synchronize albums and auto-delete duplicates\n")
		fmt.Fprintf(os.Stderr, "  %s -u http://localhost:2283 -k YOUR_KEY --auto-delete\n\n", os.Args[0])
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("Immich Duplicate Cleaner v%s\n", version)
		os.Exit(0)
	}

	// Handle shorthand flags taking precedence
	if flag.Lookup("u").Value.String() != "" {
		config.ImmichURL = flag.Lookup("u").Value.String()
	}
	if flag.Lookup("k").Value.String() != "" {
		config.APIKey = flag.Lookup("k").Value.String()
	}

	return config
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	if config.ImmichURL == "" {
		return fmt.Errorf("--url is required")
	}
	if config.APIKey == "" {
		return fmt.Errorf("--api-key is required")
	}

	// Trim trailing slash from URL
	config.ImmichURL = strings.TrimSuffix(config.ImmichURL, "/")

	return nil
}

// processDuplicateGroup handles a single duplicate group
func processDuplicateGroup(config *Config, groupNum, totalGroups int, group DuplicateGroup) error {
	logInfo("\n📁 Processing group %d/%d (%d assets)", groupNum, totalGroups, len(group.Assets))

	if len(group.Assets) < 2 {
		logWarning("⚠️  Skipping group - less than 2 assets")
		return nil
	}

	// Step 1: Synchronize albums
	syncCount, err := synchronizeAlbums(config, group)
	if err != nil {
		return fmt.Errorf("album synchronization failed: %w", err)
	}

	if syncCount > 0 {
		logInfo("✨ Synchronized %d asset(s) across albums", syncCount)
	} else {
		logInfo("✓ Albums already synchronized")
	}

	// Step 2: Auto-delete if enabled
	if config.AutoDelete {
		if err := autoDeleteDuplicates(config, group); err != nil {
			return fmt.Errorf("auto-delete failed: %w", err)
		}
	}

	return nil
}

// synchronizeAlbums ensures all duplicates are in the same albums
func synchronizeAlbums(config *Config, group DuplicateGroup) (int, error) {
	// Fetch albums for each asset
	assetAlbums := make(map[string][]Album)
	allAlbumIDs := make(map[string]bool)

	for _, asset := range group.Assets {
		albums, err := getAlbumsForAsset(config, asset.ID)
		if err != nil {
			logWarning("⚠️  Failed to fetch albums for asset %s: %v", truncateID(asset.ID), err)
			continue
		}
		assetAlbums[asset.ID] = albums

		for _, album := range albums {
			allAlbumIDs[album.ID] = true
		}
	}

	// Display current album assignments
	if config.Verbose {
		logInfo("📋 Current album assignments:")
		for assetID, albums := range assetAlbums {
			albumNames := make([]string, len(albums))
			for i, album := range albums {
				albumNames[i] = album.AlbumName
			}
			logInfo("   Asset %s: %v", truncateID(assetID), albumNames)
		}
	}

	// Synchronize albums
	syncCount := 0
	for albumID := range allAlbumIDs {
		assetsToAdd := []string{}

		// Find assets not in this album
		for _, asset := range group.Assets {
			inAlbum := false
			for _, album := range assetAlbums[asset.ID] {
				if album.ID == albumID {
					inAlbum = true
					break
				}
			}
			if !inAlbum {
				assetsToAdd = append(assetsToAdd, asset.ID)
			}
		}

		// Add missing assets to album
		if len(assetsToAdd) > 0 {
			if config.DryRun {
				logInfo("   [DRY RUN] Would add %d asset(s) to album %s", len(assetsToAdd), truncateID(albumID))
				syncCount += len(assetsToAdd)
			} else {
				if err := addAssetsToAlbum(config, albumID, assetsToAdd); err != nil {
					logError("❌ Failed to add assets to album %s: %v", truncateID(albumID), err)
				} else {
					logInfo("✅ Added %d asset(s) to album %s", len(assetsToAdd), truncateID(albumID))
					syncCount += len(assetsToAdd)
				}
			}
		}
	}

	return syncCount, nil
}

// autoDeleteDuplicates automatically deletes lower-quality duplicates
func autoDeleteDuplicates(config *Config, group DuplicateGroup) error {
	logInfo("\n🔍 Analyzing quality of %d duplicate(s)...", len(group.Assets))

	// Fetch detailed info for all assets
	assetDetails := make(map[string]*AssetDetails)
	for _, asset := range group.Assets {
		details, err := getAssetDetails(config, asset.ID)
		if err != nil {
			logWarning("⚠️  Failed to fetch details for asset %s: %v", truncateID(asset.ID), err)
			continue
		}
		assetDetails[asset.ID] = details
	}

	if len(assetDetails) < 2 {
		logWarning("⚠️  Not enough asset details to compare quality")
		return nil
	}

	// Find the best quality asset
	bestAssetID := selectBestQualityAsset(assetDetails)
	if bestAssetID == "" {
		return fmt.Errorf("failed to determine best quality asset")
	}

	logInfo("🏆 Best quality asset: %s", truncateID(bestAssetID))
	if config.Verbose && assetDetails[bestAssetID].ExifInfo != nil {
		logInfo("   Size: %d bytes, Resolution: %dx%d",
			assetDetails[bestAssetID].ExifInfo.FileSizeInByte,
			assetDetails[bestAssetID].ExifInfo.ImageWidth,
			assetDetails[bestAssetID].ExifInfo.ImageHeight)
	}

	// Identify assets to delete
	assetsToDelete := []string{}
	for assetID := range assetDetails {
		if assetID != bestAssetID {
			assetsToDelete = append(assetsToDelete, assetID)
		}
	}

	if len(assetsToDelete) == 0 {
		logInfo("✓ No duplicates to delete")
		return nil
	}

	// Confirm deletion unless --yes flag is set
	if !config.Yes && !config.DryRun {
		fmt.Printf("\n⚠️  About to delete %d duplicate(s). Continue? [y/N]: ", len(assetsToDelete))
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			// User cancelled or error reading input
			logInfo("❌ Deletion cancelled")
			return nil
		}
		if !strings.EqualFold(response, "y") && !strings.EqualFold(response, "yes") {
			logInfo("❌ Deletion cancelled by user")
			return nil
		}
	}

	// Delete duplicates
	for _, assetID := range assetsToDelete {
		if config.DryRun {
			logInfo("   [DRY RUN] Would delete asset %s", truncateID(assetID))
		} else {
			if err := deleteAsset(config, assetID); err != nil {
				logError("❌ Failed to delete asset %s: %v", truncateID(assetID), err)
			} else {
				logInfo("🗑️  Deleted duplicate asset %s", truncateID(assetID))
			}
		}
	}

	return nil
}

// selectBestQualityAsset determines which asset has the best quality
// Priority: 1) File size (larger is better), 2) Original filename, 3) Creation date
func selectBestQualityAsset(assets map[string]*AssetDetails) string {
	var bestID string
	var bestSize int64 = -1

	for assetID, details := range assets {
		if details.ExifInfo == nil {
			continue
		}

		size := details.ExifInfo.FileSizeInByte

		// Prefer larger files
		if size > bestSize {
			bestSize = size
			bestID = assetID
		} else if size == bestSize && bestID != "" {
			// If same size, prefer original filename (no IMG_, DSC_, etc.)
			if isOriginalFilename(details.OriginalFileName) && !isOriginalFilename(assets[bestID].OriginalFileName) {
				bestID = assetID
			} else if details.FileCreatedAt.Before(assets[bestID].FileCreatedAt) {
				// If same size and both/neither original, prefer earlier creation date
				bestID = assetID
			}
		}
	}

	return bestID
}

// isOriginalFilename checks if a filename appears to be an original (not auto-generated)
func isOriginalFilename(filename string) bool {
	upper := strings.ToUpper(filename)
	prefixes := []string{"IMG_", "DSC_", "DSCN", "P_", "PHOTO_", "VID_"}

	for _, prefix := range prefixes {
		if strings.HasPrefix(upper, prefix) {
			return false
		}
	}

	return true
}

// getDuplicates fetches all duplicate groups from Immich
func getDuplicates(config *Config) ([]DuplicateGroup, error) {
	url := fmt.Sprintf("%s%s", config.ImmichURL, duplicatesEndpoint)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", config.APIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logError("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("HTTP %d: failed to read response body: %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var duplicates []DuplicateGroup
	if err := json.NewDecoder(resp.Body).Decode(&duplicates); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return duplicates, nil
}

// getAlbumsForAsset fetches all albums containing a specific asset
func getAlbumsForAsset(config *Config, assetID string) ([]Album, error) {
	url := fmt.Sprintf("%s%s?assetId=%s", config.ImmichURL, albumsEndpoint, assetID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", config.APIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logError("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("HTTP %d: failed to read response body: %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var albums []Album
	if err := json.NewDecoder(resp.Body).Decode(&albums); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return albums, nil
}

// getAssetDetails fetches detailed information about an asset
func getAssetDetails(config *Config, assetID string) (*AssetDetails, error) {
	url := fmt.Sprintf("%s%s/%s", config.ImmichURL, assetsEndpoint, assetID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", config.APIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logError("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("HTTP %d: failed to read response body: %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var details AssetDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &details, nil
}

// addAssetsToAlbum adds assets to an album
func addAssetsToAlbum(config *Config, albumID string, assetIDs []string) error {
	url := fmt.Sprintf("%s%s/%s/assets", config.ImmichURL, albumsEndpoint, albumID)

	requestBody := AddAssetsRequest{IDs: assetIDs}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logError("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("HTTP %d: failed to read response body: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// deleteAsset deletes an asset from Immich
func deleteAsset(config *Config, assetID string) error {
	url := fmt.Sprintf("%s%s", config.ImmichURL, assetsEndpoint)

	requestBody := map[string]interface{}{
		"ids":   []string{assetID},
		"force": true,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logError("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("HTTP %d: failed to read response body: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Logging functions

func logInfo(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func logWarning(format string, args ...interface{}) {
	log.Printf("⚠️  "+format, args...)
}

func logError(format string, args ...interface{}) {
	log.Printf("❌ "+format, args...)
}

// truncateID returns a shortened version of an ID for display
func truncateID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}
