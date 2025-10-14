package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestHandleSearchMissingQuery tests search endpoint with missing query parameter
func TestHandleSearchMissingQuery(t *testing.T) {
	ws := &WebServer{ctx: context.Background()}

	req := httptest.NewRequest("GET", "/api/search", nil)
	w := httptest.NewRecorder()

	ws.handleSearch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errResp.Error == "" {
		t.Error("Expected error message, got empty string")
	}
}

// TestHandleDownloadURLMissingID tests download URL endpoint with missing ID
func TestHandleDownloadURLMissingID(t *testing.T) {
	ws := &WebServer{ctx: context.Background()}

	req := httptest.NewRequest("GET", "/api/download-url", nil)
	w := httptest.NewRecorder()

	ws.handleDownloadURL(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestHandleDownloadURLInvalidID tests download URL endpoint with invalid ID
func TestHandleDownloadURLInvalidID(t *testing.T) {
	ws := &WebServer{ctx: context.Background()}

	req := httptest.NewRequest("GET", "/api/download-url?id=invalid", nil)
	w := httptest.NewRecorder()

	ws.handleDownloadURL(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errResp.Error != "invalid track ID" {
		t.Errorf("Expected 'invalid track ID', got '%s'", errResp.Error)
	}
}

// TestHandleAlbumTracksMissingParams tests album tracks endpoint with missing parameters
func TestHandleAlbumTracksMissingParams(t *testing.T) {
	ws := &WebServer{ctx: context.Background()}

	// Test missing both params
	req := httptest.NewRequest("GET", "/api/album-tracks", nil)
	w := httptest.NewRecorder()
	ws.handleAlbumTracks(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Test missing name param
	req = httptest.NewRequest("GET", "/api/album-tracks?id=123", nil)
	w = httptest.NewRecorder()
	ws.handleAlbumTracks(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Test missing id param
	req = httptest.NewRequest("GET", "/api/album-tracks?name=test", nil)
	w = httptest.NewRecorder()
	ws.handleAlbumTracks(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestHandleAlbumTracksInvalidID tests album tracks endpoint with invalid ID
func TestHandleAlbumTracksInvalidID(t *testing.T) {
	ws := &WebServer{ctx: context.Background()}

	req := httptest.NewRequest("GET", "/api/album-tracks?id=invalid&name=test", nil)
	w := httptest.NewRecorder()

	ws.handleAlbumTracks(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errResp.Error != "invalid album ID" {
		t.Errorf("Expected 'invalid album ID', got '%s'", errResp.Error)
	}
}

// TestHandleArtistTracksMissingParams tests artist tracks endpoint with missing parameters
func TestHandleArtistTracksMissingParams(t *testing.T) {
	ws := &WebServer{ctx: context.Background()}

	// Test missing both params
	req := httptest.NewRequest("GET", "/api/artist-tracks", nil)
	w := httptest.NewRecorder()
	ws.handleArtistTracks(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestHandleArtistTracksInvalidID tests artist tracks endpoint with invalid ID
func TestHandleArtistTracksInvalidID(t *testing.T) {
	ws := &WebServer{ctx: context.Background()}

	req := httptest.NewRequest("GET", "/api/artist-tracks?id=invalid&name=test", nil)
	w := httptest.NewRecorder()

	ws.handleArtistTracks(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errResp.Error != "invalid artist ID" {
		t.Errorf("Expected 'invalid artist ID', got '%s'", errResp.Error)
	}
}

// TestEnableCORS tests CORS middleware
func TestEnableCORS(t *testing.T) {
	handler := enableCORS(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	// Check CORS headers
	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("Expected Access-Control-Allow-Origin: *, got %s", origin)
	}

	if methods := w.Header().Get("Access-Control-Allow-Methods"); methods != "GET, POST, OPTIONS" {
		t.Errorf("Expected Access-Control-Allow-Methods: GET, POST, OPTIONS, got %s", methods)
	}

	if headers := w.Header().Get("Access-Control-Allow-Headers"); headers != "Content-Type" {
		t.Errorf("Expected Access-Control-Allow-Headers: Content-Type, got %s", headers)
	}
}

// TestEnableCORSOptions tests CORS OPTIONS request
func TestEnableCORSOptions(t *testing.T) {
	handler := enableCORS(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for OPTIONS request")
	})

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d for OPTIONS, got %d", http.StatusOK, w.Code)
	}
}

// TestResponseStructures tests that response structures can be properly marshaled
func TestResponseStructures(t *testing.T) {
	// Test TrackResponse
	track := TrackResponse{
		ID:        123,
		Title:     "Test Track",
		Artist:    "Test Artist",
		Artists:   []string{"Test Artist"},
		ArtistIDs: []int{456},
		Album:     "Test Album",
		AlbumID:   789,
		Duration:  180000,
		CoverURL:  "https://example.com/cover.jpg",
		Available: true,
	}

	data, err := json.Marshal(track)
	if err != nil {
		t.Fatalf("Failed to marshal TrackResponse: %v", err)
	}

	var decoded TrackResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal TrackResponse: %v", err)
	}

	if decoded.ID != track.ID || decoded.Title != track.Title {
		t.Error("TrackResponse encoding/decoding mismatch")
	}

	// Test AlbumResponse
	album := AlbumResponse{
		ID:         123,
		Title:      "Test Album",
		Artist:     "Test Artist",
		Artists:    []string{"Test Artist"},
		Year:       2024,
		CoverURL:   "https://example.com/cover.jpg",
		TrackCount: 10,
	}

	data, err = json.Marshal(album)
	if err != nil {
		t.Fatalf("Failed to marshal AlbumResponse: %v", err)
	}

	// Test SearchResponse
	searchResp := SearchResponse{
		Tracks:            []TrackResponse{track},
		Albums:            []AlbumResponse{album},
		Artists:           []ArtistResponse{{ID: 456, Name: "Test Artist"}},
		Total:             3,
		MisspellCorrected: false,
		CorrectedText:     "",
	}

	data, err = json.Marshal(searchResp)
	if err != nil {
		t.Fatalf("Failed to marshal SearchResponse: %v", err)
	}

	var decodedSearch SearchResponse
	if err := json.Unmarshal(data, &decodedSearch); err != nil {
		t.Fatalf("Failed to unmarshal SearchResponse: %v", err)
	}

	if len(decodedSearch.Tracks) != 1 || len(decodedSearch.Albums) != 1 {
		t.Error("SearchResponse encoding/decoding mismatch")
	}
}

// Integration tests that require actual API credentials
// These will be skipped if credentials are not available

// TestSearchIntegration tests the search endpoint with actual API
func TestSearchIntegration(t *testing.T) {
	if os.Getenv("YA_MUSIC_TOKEN") == "" || os.Getenv("YA_MUSIC_ID") == "" {
		t.Skip("Skipping integration test: YA_MUSIC_TOKEN or YA_MUSIC_ID not set")
	}

	ctx := context.Background()
	ws, err := NewWebServer(ctx)
	if err != nil {
		t.Fatalf("Failed to create web server: %v", err)
	}

	// Test search with a known query
	req := httptest.NewRequest("GET", "/api/search?q=test", nil)
	w := httptest.NewRecorder()

	ws.handleSearch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	var resp SearchResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode search response: %v", err)
	}

	t.Logf("Search returned %d total results", resp.Total)
	t.Logf("Tracks: %d, Albums: %d, Artists: %d", len(resp.Tracks), len(resp.Albums), len(resp.Artists))
}

// TestHandleIndexWithBasePath tests that index.html is served with base tag injection
func TestHandleIndexWithBasePath(t *testing.T) {
	// Create a temporary static directory with index.html
	tmpDir := t.TempDir()
	staticDir := tmpDir + "/static"
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		t.Fatalf("Failed to create temp static dir: %v", err)
	}

	// Create a simple index.html
	indexHTML := "<head>\n<title>Test</title>\n</head>\n<body>Test</body>"
	if err := os.WriteFile(staticDir+"/index.html", []byte(indexHTML), 0644); err != nil {
		t.Fatalf("Failed to write index.html: %v", err)
	}

	// Change to temp directory
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	// Create WebServer with base path
	ws := &WebServer{
		ctx:      context.Background(),
		basePath: "/music",
	}

	// Test both /music and /music/ paths
	testCases := []struct {
		path string
		name string
	}{
		{"/music", "base path without trailing slash"},
		{"/music/", "base path with trailing slash"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.path, nil)
			w := httptest.NewRecorder()

			ws.handleIndex(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
			}

			body := w.Body.String()

			// Check that base tag is injected
			if !strings.Contains(body, `<base href="/music/">`) {
				t.Errorf("Expected base tag with href='/music/', but not found in response")
				t.Logf("Response body: %s", body)
			}

			// Check that BASE_PATH script is injected
			if !strings.Contains(body, `window.BASE_PATH = '/music'`) {
				t.Errorf("Expected BASE_PATH script, but not found in response")
				t.Logf("Response body: %s", body)
			}
		})
	}
}

// TestBasePathNoRedirect tests that accessing base path doesn't cause redirect
func TestBasePathNoRedirect(t *testing.T) {
	// Create a temporary static directory with index.html
	tmpDir := t.TempDir()
	staticDir := tmpDir + "/static"
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		t.Fatalf("Failed to create temp static dir: %v", err)
	}

	// Create a simple index.html
	indexHTML := "<head>\n<title>Test</title>\n</head>\n<body>Test</body>"
	if err := os.WriteFile(staticDir+"/index.html", []byte(indexHTML), 0644); err != nil {
		t.Fatalf("Failed to write index.html: %v", err)
	}

	// Change to temp directory
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	// Create WebServer with base path
	ws := &WebServer{
		ctx:      context.Background(),
		basePath: "/music",
	}

	// Create a test server to verify no redirects occur
	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir("./static"))
	
	basePathHandler := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		
		if path == ws.basePath || path == ws.basePath+"/" {
			ws.handleIndex(w, r)
			return
		}
		
		if strings.HasPrefix(path, ws.basePath+"/") {
			http.StripPrefix(ws.basePath, fs).ServeHTTP(w, r)
			return
		}
		
		http.NotFound(w, r)
	}
	
	mux.HandleFunc(ws.basePath, basePathHandler)
	mux.HandleFunc(ws.basePath+"/", basePathHandler)

	// Test /music without trailing slash - should NOT redirect
	req := httptest.NewRequest("GET", "/music", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code == http.StatusMovedPermanently || w.Code == http.StatusFound {
		t.Errorf("Expected no redirect for /music, but got %d with Location: %s", 
			w.Code, w.Header().Get("Location"))
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d for /music, got %d", http.StatusOK, w.Code)
	}

	// Test /music/ with trailing slash - should also work
	req = httptest.NewRequest("GET", "/music/", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d for /music/, got %d", http.StatusOK, w.Code)
	}
}

// TestAlbumTracksIntegration tests the album tracks endpoint with actual API
func TestAlbumTracksIntegration(t *testing.T) {
	if os.Getenv("YA_MUSIC_TOKEN") == "" || os.Getenv("YA_MUSIC_ID") == "" {
		t.Skip("Skipping integration test: YA_MUSIC_TOKEN or YA_MUSIC_ID not set")
	}

	ctx := context.Background()
	ws, err := NewWebServer(ctx)
	if err != nil {
		t.Fatalf("Failed to create web server: %v", err)
	}

	// First, search for "Chick Corea solo" to get an album ID
	searchReq := httptest.NewRequest("GET", "/api/search?q=Chick+Corea+solo", nil)
	searchW := httptest.NewRecorder()

	ws.handleSearch(searchW, searchReq)

	if searchW.Code != http.StatusOK {
		t.Fatalf("Search failed with status %d", searchW.Code)
	}

	var searchResp SearchResponse
	if err := json.NewDecoder(searchW.Body).Decode(&searchResp); err != nil {
		t.Fatalf("Failed to decode search response: %v", err)
	}

	if len(searchResp.Albums) == 0 {
		t.Skip("No albums found in search results")
	}

	// Get the first album
	firstAlbum := searchResp.Albums[0]
	t.Logf("Testing with album: %s (ID: %d) by %s", firstAlbum.Title, firstAlbum.ID, firstAlbum.Artist)
	t.Logf("Album has %d tracks according to search results", firstAlbum.TrackCount)

	// Now test album tracks endpoint
	albumReq := httptest.NewRequest("GET", "/api/album-tracks?id="+strconv.Itoa(firstAlbum.ID)+"&name="+url.QueryEscape(firstAlbum.Title), nil)
	albumW := httptest.NewRecorder()

	ws.handleAlbumTracks(albumW, albumReq)

	if albumW.Code != http.StatusOK {
		t.Errorf("Album tracks request failed with status %d", albumW.Code)
		t.Logf("Response body: %s", albumW.Body.String())
	}

	var albumResp SearchResponse
	if err := json.NewDecoder(albumW.Body).Decode(&albumResp); err != nil {
		t.Fatalf("Failed to decode album tracks response: %v", err)
	}

	t.Logf("Album tracks endpoint returned %d tracks", len(albumResp.Tracks))

	// Verify we got tracks
	if len(albumResp.Tracks) == 0 {
		t.Errorf("Expected tracks for album '%s' (ID: %d) but got 0",
			firstAlbum.Title, firstAlbum.ID)
		t.Logf("Album reports %d tracks in trackCount", firstAlbum.TrackCount)
	} else {
		t.Logf("Successfully retrieved %d tracks from album", len(albumResp.Tracks))
		// Log first few track titles
		for i, track := range albumResp.Tracks {
			if i >= 3 {
				break
			}
			t.Logf("Track %d: %s by %s", i+1, track.Title, track.Artist)
		}
	}
}
