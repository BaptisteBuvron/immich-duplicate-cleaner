package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

// MockHTTPClient is a mock implementation of HTTPClient for testing
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

// TestValidateConfig tests the configuration validation
func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				ImmichURL: "http://localhost:2283",
				APIKey:    "test-key",
			},
			wantErr: false,
		},
		{
			name: "missing URL",
			config: &Config{
				APIKey: "test-key",
			},
			wantErr: true,
		},
		{
			name: "missing API key",
			config: &Config{
				ImmichURL: "http://localhost:2283",
			},
			wantErr: true,
		},
		{
			name: "URL with trailing slash",
			config: &Config{
				ImmichURL: "http://localhost:2283/",
				APIKey:    "test-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Check that trailing slash is removed
			if !tt.wantErr && tt.config.ImmichURL[len(tt.config.ImmichURL)-1] == '/' {
				t.Error("validateConfig() should remove trailing slash from URL")
			}
		})
	}
}

// TestSelectBestQualityAsset tests the quality comparison algorithm
func TestSelectBestQualityAsset(t *testing.T) {
	tests := []struct {
		name   string
		assets map[string]*AssetDetails
		want   string
	}{
		{
			name: "prefer larger file size",
			assets: map[string]*AssetDetails{
				"asset1": {
					ID:               "asset1",
					OriginalFileName: "photo.jpg",
					ExifInfo: &ExifInfo{
						FileSizeInByte: 1000000,
					},
				},
				"asset2": {
					ID:               "asset2",
					OriginalFileName: "photo.jpg",
					ExifInfo: &ExifInfo{
						FileSizeInByte: 2000000,
					},
				},
			},
			want: "asset2",
		},
		{
			name: "same size, prefer original filename",
			assets: map[string]*AssetDetails{
				"asset1": {
					ID:               "asset1",
					OriginalFileName: "IMG_1234.jpg",
					ExifInfo: &ExifInfo{
						FileSizeInByte: 1000000,
					},
				},
				"asset2": {
					ID:               "asset2",
					OriginalFileName: "vacation.jpg",
					ExifInfo: &ExifInfo{
						FileSizeInByte: 1000000,
					},
				},
			},
			want: "asset2",
		},
		{
			name: "same size and filename type, prefer earlier date",
			assets: map[string]*AssetDetails{
				"asset1": {
					ID:               "asset1",
					OriginalFileName: "photo.jpg",
					FileCreatedAt:    time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
					ExifInfo: &ExifInfo{
						FileSizeInByte: 1000000,
					},
				},
				"asset2": {
					ID:               "asset2",
					OriginalFileName: "image.jpg",
					FileCreatedAt:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					ExifInfo: &ExifInfo{
						FileSizeInByte: 1000000,
					},
				},
			},
			want: "asset2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectBestQualityAsset(tt.assets)
			if got != tt.want {
				t.Errorf("selectBestQualityAsset() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsOriginalFilename tests the filename originality detection
func TestIsOriginalFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"original filename", "vacation_2023.jpg", true},
		{"IMG prefix", "IMG_1234.jpg", false},
		{"DSC prefix", "DSC_5678.jpg", false},
		{"DSCN prefix", "DSCN0001.jpg", false},
		{"VID prefix", "VID_20230101.mp4", false},
		{"lowercase img", "img_1234.jpg", false},
		{"custom name", "my_photo.jpg", true},
		{"date name", "2023-01-01_photo.jpg", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isOriginalFilename(tt.filename)
			if got != tt.want {
				t.Errorf("isOriginalFilename(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

// TestGetDuplicates tests the getDuplicates function with a mock HTTP client
func TestGetDuplicates(t *testing.T) {
	tests := []struct {
		name       string
		mockResp   *http.Response
		mockErr    error
		wantGroups int
		wantErr    bool
	}{
		{
			name: "successful response",
			mockResp: &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(bytes.NewBufferString(`[
					{
						"duplicateId": "dup1",
						"assets": [
							{"id": "asset1"},
							{"id": "asset2"}
						]
					}
				]`)),
			},
			wantGroups: 1,
			wantErr:    false,
		},
		{
			name: "empty response",
			mockResp: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`[]`)),
			},
			wantGroups: 0,
			wantErr:    false,
		},
		{
			name: "HTTP error",
			mockResp: &http.Response{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(bytes.NewBufferString(`{"message": "Unauthorized"}`)),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Replace global HTTP client with mock
			oldClient := httpClient
			defer func() { httpClient = oldClient }()

			httpClient = &MockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					return tt.mockResp, tt.mockErr
				},
			}

			config := &Config{
				ImmichURL: "http://localhost:2283",
				APIKey:    "test-key",
			}

			groups, err := getDuplicates(config)

			if (err != nil) != tt.wantErr {
				t.Errorf("getDuplicates() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(groups) != tt.wantGroups {
				t.Errorf("getDuplicates() returned %d groups, want %d", len(groups), tt.wantGroups)
			}
		})
	}
}

// TestGetAlbumsForAsset tests the getAlbumsForAsset function
func TestGetAlbumsForAsset(t *testing.T) {
	mockAlbums := []Album{
		{ID: "album1", AlbumName: "Vacation", AssetCount: 10},
		{ID: "album2", AlbumName: "Family", AssetCount: 20},
	}

	albumsJSON, _ := json.Marshal(mockAlbums)

	oldClient := httpClient
	defer func() { httpClient = oldClient }()

	httpClient = &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBuffer(albumsJSON)),
			}, nil
		},
	}

	config := &Config{
		ImmichURL: "http://localhost:2283",
		APIKey:    "test-key",
	}

	albums, err := getAlbumsForAsset(config, "asset1")
	if err != nil {
		t.Errorf("getAlbumsForAsset() error = %v", err)
		return
	}

	if len(albums) != 2 {
		t.Errorf("getAlbumsForAsset() returned %d albums, want 2", len(albums))
	}

	if albums[0].AlbumName != "Vacation" {
		t.Errorf("First album name = %s, want Vacation", albums[0].AlbumName)
	}
}

// TestGetAssetDetails tests the getAssetDetails function
func TestGetAssetDetails(t *testing.T) {
	mockAsset := &AssetDetails{
		ID:               "asset1",
		OriginalFileName: "photo.jpg",
		FileCreatedAt:    time.Now(),
		ExifInfo: &ExifInfo{
			FileSizeInByte: 1500000,
			ImageWidth:     1920,
			ImageHeight:    1080,
		},
	}

	assetJSON, _ := json.Marshal(mockAsset)

	oldClient := httpClient
	defer func() { httpClient = oldClient }()

	httpClient = &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBuffer(assetJSON)),
			}, nil
		},
	}

	config := &Config{
		ImmichURL: "http://localhost:2283",
		APIKey:    "test-key",
	}

	details, err := getAssetDetails(config, "asset1")
	if err != nil {
		t.Errorf("getAssetDetails() error = %v", err)
		return
	}

	if details.ID != "asset1" {
		t.Errorf("Asset ID = %s, want asset1", details.ID)
	}

	if details.ExifInfo.FileSizeInByte != 1500000 {
		t.Errorf("File size = %d, want 1500000", details.ExifInfo.FileSizeInByte)
	}
}

// TestAddAssetsToAlbum tests the addAssetsToAlbum function
func TestAddAssetsToAlbum(t *testing.T) {
	oldClient := httpClient
	defer func() { httpClient = oldClient }()

	var capturedRequest *http.Request

	httpClient = &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			capturedRequest = req
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"success": true}`)),
			}, nil
		},
	}

	config := &Config{
		ImmichURL: "http://localhost:2283",
		APIKey:    "test-key",
	}

	assetIDs := []string{"asset1", "asset2", "asset3"}
	err := addAssetsToAlbum(config, "album1", assetIDs)

	if err != nil {
		t.Errorf("addAssetsToAlbum() error = %v", err)
		return
	}

	// Verify request method
	if capturedRequest.Method != "PUT" {
		t.Errorf("Request method = %s, want PUT", capturedRequest.Method)
	}

	// Verify request body
	var body AddAssetsRequest
	bodyBytes, _ := io.ReadAll(capturedRequest.Body)
	json.Unmarshal(bodyBytes, &body)

	if len(body.IDs) != 3 {
		t.Errorf("Request body contains %d IDs, want 3", len(body.IDs))
	}
}

// TestDeleteAsset tests the deleteAsset function
func TestDeleteAsset(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{"successful delete - 204", http.StatusNoContent, false},
		{"successful delete - 200", http.StatusOK, false},
		{"failed delete - 404", http.StatusNotFound, true},
		{"failed delete - 500", http.StatusInternalServerError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldClient := httpClient
			defer func() { httpClient = oldClient }()

			httpClient = &MockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: tt.statusCode,
						Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
					}, nil
				},
			}

			config := &Config{
				ImmichURL: "http://localhost:2283",
				APIKey:    "test-key",
			}

			err := deleteAsset(config, "asset1")

			if (err != nil) != tt.wantErr {
				t.Errorf("deleteAsset() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestTruncateID tests the ID truncation helper function
func TestTruncateID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{"long ID", "1234567890abcdef", "12345678"},
		{"short ID", "12345", "12345"},
		{"exactly 8 chars", "12345678", "12345678"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateID(tt.id)
			if got != tt.want {
				t.Errorf("truncateID(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}

// BenchmarkSelectBestQualityAsset benchmarks the quality selection algorithm
func BenchmarkSelectBestQualityAsset(b *testing.B) {
	assets := map[string]*AssetDetails{
		"asset1": {
			ID:               "asset1",
			OriginalFileName: "IMG_1234.jpg",
			FileCreatedAt:    time.Now(),
			ExifInfo: &ExifInfo{
				FileSizeInByte: 1000000,
			},
		},
		"asset2": {
			ID:               "asset2",
			OriginalFileName: "photo.jpg",
			FileCreatedAt:    time.Now(),
			ExifInfo: &ExifInfo{
				FileSizeInByte: 2000000,
			},
		},
		"asset3": {
			ID:               "asset3",
			OriginalFileName: "DSC_5678.jpg",
			FileCreatedAt:    time.Now(),
			ExifInfo: &ExifInfo{
				FileSizeInByte: 1500000,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		selectBestQualityAsset(assets)
	}
}
