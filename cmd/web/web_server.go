package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/joho/godotenv"
	"pkg.botr.me/yamusic"
)

// WebServer handles HTTP requests for the web interface
type WebServer struct {
	client          *yamusic.Client
	ctx             context.Context
	basePath        string // Static base path from env var (fallback)
	useProxyHeaders bool   // Whether to check X-Forwarded-Prefix header
}

// TrackResponse represents a track in API responses
type TrackResponse struct {
	ID        int      `json:"id"`
	Title     string   `json:"title"`
	Artist    string   `json:"artist"`
	Artists   []string `json:"artists"`
	ArtistIDs []int    `json:"artistIds,omitempty"`
	Album     string   `json:"album,omitempty"`
	AlbumID   int      `json:"albumId,omitempty"`
	Duration  int      `json:"duration"`
	CoverURL  string   `json:"coverUrl,omitempty"`
	Available bool     `json:"available"`
}

// AlbumResponse represents an album in API responses
type AlbumResponse struct {
	ID         int      `json:"id"`
	Title      string   `json:"title"`
	Artist     string   `json:"artist"`
	Artists    []string `json:"artists"`
	Year       int      `json:"year,omitempty"`
	CoverURL   string   `json:"coverUrl,omitempty"`
	TrackCount int      `json:"trackCount,omitempty"`
}

// ArtistResponse represents an artist in API responses
type ArtistResponse struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	CoverURL string `json:"coverUrl,omitempty"`
}

// SearchResponse represents search results
type SearchResponse struct {
	Tracks            []TrackResponse  `json:"tracks"`
	Albums            []AlbumResponse  `json:"albums,omitempty"`
	Artists           []ArtistResponse `json:"artists,omitempty"`
	Total             int              `json:"total"`
	MisspellCorrected bool             `json:"misspellCorrected,omitempty"`
	CorrectedText     string           `json:"correctedText,omitempty"`
}

// DownloadURLResponse represents download URL response
type DownloadURLResponse struct {
	URL string `json:"url"`
}

// ErrorResponse represents error responses
type ErrorResponse struct {
	Error string `json:"error"`
}

// NewWebServer creates a new web server instance
func NewWebServer(ctx context.Context) (*WebServer, error) {
	// Try to load .env file from multiple locations
	// First try current directory (for production binary)
	err := godotenv.Load()
	if err != nil {
		// If not found, try parent directories (for tests running in cmd/web/)
		err = godotenv.Load("../../.env")
		if err != nil {
			// If still not found, continue anyway - env vars might be set externally
			log.Printf("Warning: Could not load .env file: %v", err)
		}
	}

	token := os.Getenv("YA_MUSIC_TOKEN")
	uid, err := strconv.Atoi(os.Getenv("YA_MUSIC_ID"))
	if err != nil {
		return nil, err
	}

	// Get base path from environment variable (e.g., "/music" for reverse proxy)
	// This serves as a fallback when X-Forwarded-Prefix header is not present
	basePath := os.Getenv("BASE_PATH")
	if basePath != "" {
		// Ensure base path starts with / and doesn't end with /
		if basePath[0] != '/' {
			basePath = "/" + basePath
		}
		if len(basePath) > 1 && basePath[len(basePath)-1] == '/' {
			basePath = basePath[:len(basePath)-1]
		}
	}

	// Check if we should use proxy headers (modern cloud-native approach)
	// Default to true for better reverse proxy compatibility
	useProxyHeaders := os.Getenv("USE_PROXY_HEADERS") != "false"

	client := yamusic.NewClient(yamusic.AccessToken(uid, token))
	return &WebServer{
		client:          client,
		ctx:             ctx,
		basePath:        basePath,
		useProxyHeaders: useProxyHeaders,
	}, nil
}

// getBasePath returns the base path for a request
// It checks X-Forwarded-Prefix header first (if enabled), then falls back to static BASE_PATH
func (ws *WebServer) getBasePath(r *http.Request) string {
	// Check X-Forwarded-Prefix header first (modern approach)
	if ws.useProxyHeaders {
		if prefix := r.Header.Get("X-Forwarded-Prefix"); prefix != "" {
			// Clean the prefix
			if prefix[0] != '/' {
				prefix = "/" + prefix
			}
			if len(prefix) > 1 && prefix[len(prefix)-1] == '/' {
				prefix = prefix[:len(prefix)-1]
			}
			return prefix
		}
	}
	
	// Fall back to static base path from env var
	return ws.basePath
}

// handleSearch handles track search requests
func (ws *WebServer) handleSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query().Get("q")
	if query == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "query parameter 'q' is required"})
		return
	}

	// Search for all content types (tracks, albums, artists)
	s, resp, err := ws.client.Search().All(ws.ctx, query, &yamusic.SearchOptions{Page: 0, NoCorrect: false})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}
	if resp.StatusCode != 200 {
		w.WriteHeader(resp.StatusCode)
		json.NewEncoder(w).Encode(ErrorResponse{Error: resp.Status})
		return
	}

	// Process tracks
	trackResults := s.Result.Tracks.Results
	tracks := make([]TrackResponse, len(trackResults))
	for i, result := range trackResults {
		artists := make([]string, len(result.Artists))
		artistIDs := make([]int, len(result.Artists))
		artistStr := ""
		for j, artist := range result.Artists {
			artists[j] = artist.Name
			artistIDs[j] = artist.ID
			if j == len(result.Artists)-1 {
				artistStr += artist.Name
			} else {
				artistStr += artist.Name + ", "
			}
		}

		album := ""
		albumID := 0
		coverURL := ""
		if len(result.Albums) > 0 {
			album = result.Albums[0].Title
			albumID = result.Albums[0].ID
			if result.Albums[0].CoverURI != "" {
				coverURL = "https://" + result.Albums[0].CoverURI
			}
		}

		tracks[i] = TrackResponse{
			ID:        result.ID,
			Title:     result.Title,
			Artist:    artistStr,
			Artists:   artists,
			ArtistIDs: artistIDs,
			Album:     album,
			AlbumID:   albumID,
			Duration:  result.DurationMs,
			CoverURL:  coverURL,
			Available: result.Available,
		}
	}

	// Process albums
	albumResults := s.Result.Albums.Results
	albums := make([]AlbumResponse, len(albumResults))
	for i, result := range albumResults {
		artists := make([]string, len(result.Artists))
		artistStr := ""
		for j, artist := range result.Artists {
			artists[j] = artist.Name
			if j == len(result.Artists)-1 {
				artistStr += artist.Name
			} else {
				artistStr += artist.Name + ", "
			}
		}

		coverURL := ""
		if result.CoverURI != "" {
			coverURL = "https://" + result.CoverURI
		}

		albums[i] = AlbumResponse{
			ID:         result.ID,
			Title:      result.Title,
			Artist:     artistStr,
			Artists:    artists,
			Year:       result.Year,
			CoverURL:   coverURL,
			TrackCount: result.TrackCount,
		}
	}

	// Process artists
	artistResults := s.Result.Artists.Results
	artists := make([]ArtistResponse, len(artistResults))
	for i, result := range artistResults {
		coverURL := ""
		if result.Cover.URI != "" {
			coverURL = "https://" + result.Cover.URI
		}

		artists[i] = ArtistResponse{
			ID:       result.ID,
			Name:     result.Name,
			CoverURL: coverURL,
		}
	}

	// Check for spelling correction
	misspellCorrected := s.Result.MisspellCorrected
	correctedText := ""
	if misspellCorrected && s.Result.MisspellResult != "" {
		correctedText = s.Result.MisspellResult
	}

	json.NewEncoder(w).Encode(SearchResponse{
		Tracks:            tracks,
		Albums:            albums,
		Artists:           artists,
		Total:             len(tracks) + len(albums) + len(artists),
		MisspellCorrected: misspellCorrected,
		CorrectedText:     correctedText,
	})
}

// handleDownloadURL handles requests for track download URLs
func (ws *WebServer) handleDownloadURL(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	trackIDStr := r.URL.Query().Get("id")
	if trackIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "query parameter 'id' is required"})
		return
	}

	trackID, err := strconv.Atoi(trackIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid track ID"})
		return
	}

	url, err := ws.client.Tracks().GetDownloadURL(ws.ctx, trackID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	json.NewEncoder(w).Encode(DownloadURLResponse{URL: url})
}

// handleAlbumTracks handles requests for album tracks using the direct API endpoint
func (ws *WebServer) handleAlbumTracks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	albumIDStr := r.URL.Query().Get("id")
	albumName := r.URL.Query().Get("name")

	if albumIDStr == "" || albumName == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "query parameters 'id' and 'name' are required"})
		return
	}

	albumID, err := strconv.Atoi(albumIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid album ID"})
		return
	}

	// Use the direct API endpoint to get album with tracks
	log.Printf("[Album Tracks] Fetching album '%s' (ID: %d) with tracks", albumName, albumID)

	// Create a request to the albums/{id}/with-tracks endpoint
	req, err := ws.client.NewRequest("GET", "albums/"+albumIDStr+"/with-tracks", nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	// Define struct to parse the API response
	var apiResp struct {
		Result struct {
			Volumes [][]struct {
				ID         json.Number `json:"id"`
				Title      string      `json:"title"`
				DurationMs int         `json:"durationMs"`
				Available  bool        `json:"available"`
				CoverURI   string      `json:"coverUri"`
				Artists    []struct {
					ID   int    `json:"id"`
					Name string `json:"name"`
				} `json:"artists"`
				Albums []struct {
					ID       int    `json:"id"`
					Title    string `json:"title"`
					CoverURI string `json:"coverUri"`
				} `json:"albums"`
			} `json:"volumes"`
		} `json:"result"`
	}

	resp, err := ws.client.Do(ws.ctx, req, &apiResp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}
	if resp.StatusCode != 200 {
		w.WriteHeader(resp.StatusCode)
		json.NewEncoder(w).Encode(ErrorResponse{Error: resp.Status})
		return
	}

	// Process the tracks from volumes
	var allTracks []TrackResponse
	for _, volume := range apiResp.Result.Volumes {
		for _, track := range volume {
			// Convert track ID from json.Number to int
			trackIDInt, err := track.ID.Int64()
			if err != nil {
				log.Printf("[Album Tracks] Failed to parse track ID: %v", err)
				continue
			}

			artists := make([]string, len(track.Artists))
			artistIDs := make([]int, len(track.Artists))
			artistStr := ""
			for j, artist := range track.Artists {
				artists[j] = artist.Name
				artistIDs[j] = artist.ID
				if j == len(track.Artists)-1 {
					artistStr += artist.Name
				} else {
					artistStr += artist.Name + ", "
				}
			}

			coverURL := ""
			if track.CoverURI != "" {
				coverURL = "https://" + track.CoverURI
			}

			albumTitle := albumName
			if len(track.Albums) > 0 {
				albumTitle = track.Albums[0].Title
				if track.Albums[0].CoverURI != "" {
					coverURL = "https://" + track.Albums[0].CoverURI
				}
			}

			allTracks = append(allTracks, TrackResponse{
				ID:        int(trackIDInt),
				Title:     track.Title,
				Artist:    artistStr,
				Artists:   artists,
				ArtistIDs: artistIDs,
				Album:     albumTitle,
				AlbumID:   albumID,
				Duration:  track.DurationMs,
				CoverURL:  coverURL,
				Available: track.Available,
			})
		}
	}

	log.Printf("[Album Tracks] Returning %d tracks for album '%s'", len(allTracks), albumName)

	json.NewEncoder(w).Encode(SearchResponse{
		Tracks: allTracks,
		Total:  len(allTracks),
	})
}

// handleArtistTracks handles requests for artist tracks by searching for the artist name
func (ws *WebServer) handleArtistTracks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	artistIDStr := r.URL.Query().Get("id")
	artistName := r.URL.Query().Get("name")

	if artistIDStr == "" || artistName == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "query parameters 'id' and 'name' are required"})
		return
	}

	artistID, err := strconv.Atoi(artistIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid artist ID"})
		return
	}

	// Search for the artist name to get tracks
	s, resp, err := ws.client.Search().Tracks(ws.ctx, artistName, &yamusic.SearchOptions{Page: 0, NoCorrect: false})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}
	if resp.StatusCode != 200 {
		w.WriteHeader(resp.StatusCode)
		json.NewEncoder(w).Encode(ErrorResponse{Error: resp.Status})
		return
	}

	// Filter tracks by artist ID
	var filteredTracks []TrackResponse
	for _, result := range s.Result.Tracks.Results {
		for _, artist := range result.Artists {
			if artist.ID == artistID {
				artists := make([]string, len(result.Artists))
				artistIDs := make([]int, len(result.Artists))
				artistStr := ""
				for j, a := range result.Artists {
					artists[j] = a.Name
					artistIDs[j] = a.ID
					if j == len(result.Artists)-1 {
						artistStr += a.Name
					} else {
						artistStr += a.Name + ", "
					}
				}

				album := ""
				albumID := 0
				coverURL := ""
				if len(result.Albums) > 0 {
					album = result.Albums[0].Title
					albumID = result.Albums[0].ID
					if result.Albums[0].CoverURI != "" {
						coverURL = "https://" + result.Albums[0].CoverURI
					}
				}

				filteredTracks = append(filteredTracks, TrackResponse{
					ID:        result.ID,
					Title:     result.Title,
					Artist:    artistStr,
					Artists:   artists,
					ArtistIDs: artistIDs,
					Album:     album,
					AlbumID:   albumID,
					Duration:  result.DurationMs,
					CoverURL:  coverURL,
					Available: result.Available,
				})
				break // Only add track once even if artist appears multiple times
			}
		}
	}

	json.NewEncoder(w).Encode(SearchResponse{
		Tracks: filteredTracks,
		Total:  len(filteredTracks),
	})
}

// enableCORS adds CORS headers to allow browser access
func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// handleIndex serves the index.html with base path injected
func (ws *WebServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Read the index.html file
	indexContent, err := os.ReadFile("./static/index.html")
	if err != nil {
		http.Error(w, "Failed to load index.html", http.StatusInternalServerError)
		return
	}

	content := string(indexContent)

	// Get dynamic base path (supports X-Forwarded-Prefix header)
	basePath := ws.getBasePath(r)
	basePathForTag := basePath
	if basePathForTag == "" {
		basePathForTag = "/"
	} else if basePathForTag[len(basePathForTag)-1] != '/' {
		basePathForTag = basePathForTag + "/"
	}

	baseTag := "<base href=\"" + basePathForTag + "\">\n    "
	basePathScript := "<script>window.BASE_PATH = '" + basePath + "';</script>\n    "

	// Insert after <head> tag
	content = strings.Replace(content, "<head>\n", "<head>\n    "+baseTag+basePathScript, 1)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(content))
}

// handleAlbumZip downloads all tracks from an album and streams them as a zip archive
func (ws *WebServer) handleAlbumZip(w http.ResponseWriter, r *http.Request) {
	albumIDStr := r.URL.Query().Get("id")
	albumName := r.URL.Query().Get("name")

	if albumIDStr == "" || albumName == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "query parameters 'id' and 'name' are required"})
		return
	}

	albumID, err := strconv.Atoi(albumIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid album ID"})
		return
	}
	_ = albumID

	// Fetch album tracks first
	req, err := ws.client.NewRequest("GET", "albums/"+albumIDStr+"/with-tracks", nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	var apiResp struct {
		Result struct {
			Volumes [][]struct {
				ID         json.Number `json:"id"`
				Title      string      `json:"title"`
				Available  bool        `json:"available"`
				Artists    []struct {
					Name string `json:"name"`
				} `json:"artists"`
			} `json:"volumes"`
		} `json:"result"`
	}

	resp, err := ws.client.Do(ws.ctx, req, &apiResp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}
	if resp.StatusCode != 200 {
		w.WriteHeader(resp.StatusCode)
		json.NewEncoder(w).Encode(ErrorResponse{Error: resp.Status})
		return
	}

	type trackInfo struct {
		id     int
		title  string
		artist string
	}

	var tracks []trackInfo
	for _, volume := range apiResp.Result.Volumes {
		for _, t := range volume {
			if !t.Available {
				continue
			}
			idInt, err := t.ID.Int64()
			if err != nil {
				continue
			}
			artistStr := ""
			for j, a := range t.Artists {
				if j > 0 {
					artistStr += ", "
				}
				artistStr += a.Name
			}
			tracks = append(tracks, trackInfo{int(idInt), t.Title, artistStr})
		}
	}

	log.Printf("[AlbumZip] Streaming %d tracks for album '%s'", len(tracks), albumName)

	// Sanitize album name for filename
	safeAlbum := strings.Map(func(r rune) rune {
		if strings.ContainsRune(`/\:*?"<>|`, r) {
			return '_'
		}
		return r
	}, albumName)

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.zip"`, safeAlbum))

	zw := zip.NewWriter(w)
	defer zw.Close()

	// Download tracks concurrently with a semaphore (max 3 parallel)
	type result struct {
		idx  int
		info trackInfo
		url  string
		err  error
	}

	sem := make(chan struct{}, 3)
	results := make([]result, len(tracks))
	var wg sync.WaitGroup

	for i, t := range tracks {
		wg.Add(1)
		go func(i int, t trackInfo) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			url, err := ws.client.Tracks().GetDownloadURL(ws.ctx, t.id)
			results[i] = result{i, t, url, err}
		}(i, t)
	}
	wg.Wait()

	// Write to zip in order
	for i, res := range results {
		if res.err != nil {
			log.Printf("[AlbumZip] Failed to get URL for track %d '%s': %v", res.info.id, res.info.title, res.err)
			continue
		}

		safeTitle := strings.Map(func(r rune) rune {
			if strings.ContainsRune(`/\:*?"<>|`, r) {
				return '_'
			}
			return r
		}, res.info.title)
		safeArtist := strings.Map(func(r rune) rune {
			if strings.ContainsRune(`/\:*?"<>|`, r) {
				return '_'
			}
			return r
		}, res.info.artist)

		filename := fmt.Sprintf("%02d. %s - %s.mp3", i+1, safeArtist, safeTitle)

		fw, err := zw.Create(filename)
		if err != nil {
			log.Printf("[AlbumZip] Failed to create zip entry for '%s': %v", filename, err)
			continue
		}

		trackResp, err := http.Get(res.url) //nolint:gosec
		if err != nil {
			log.Printf("[AlbumZip] Failed to download track '%s': %v", filename, err)
			continue
		}
		_, copyErr := io.Copy(fw, trackResp.Body)
		trackResp.Body.Close()
		if copyErr != nil {
			log.Printf("[AlbumZip] Failed to copy track '%s': %v", filename, copyErr)
		}
	}
	log.Printf("[AlbumZip] Done streaming zip for album '%s'", albumName)
}

// handleTrackInfo fetches metadata for a single track by ID.
// Uses the library's Tracks().Get() method (yamusic.TrackResp.Result []Track).
func (ws *WebServer) handleTrackInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	trackIDStr := r.URL.Query().Get("id")
	if trackIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "query parameter 'id' is required"})
		return
	}

	trackID, err := strconv.Atoi(trackIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid track ID"})
		return
	}

	// Use the library's native method — no manual NewRequest/Do needed
	trackResp, resp, err := ws.client.Tracks().Get(ws.ctx, trackID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}
	if resp.StatusCode != 200 {
		w.WriteHeader(resp.StatusCode)
		json.NewEncoder(w).Encode(ErrorResponse{Error: resp.Status})
		return
	}
	if len(trackResp.Result) == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "track not found"})
		return
	}

	t := trackResp.Result[0] // yamusic.Track

	artistStr := ""
	artists := make([]string, len(t.Artists))
	artistIDs := make([]int, len(t.Artists))
	for j, a := range t.Artists {
		artists[j] = a.Name
		artistIDs[j] = a.ID
		if j > 0 {
			artistStr += ", "
		}
		artistStr += a.Name
	}

	album := ""
	albumID := 0
	coverURL := ""
	if len(t.Albums) > 0 {
		album = t.Albums[0].Title
		albumID = t.Albums[0].ID
		if t.Albums[0].CoverURI != "" {
			coverURL = "https://" + t.Albums[0].CoverURI
		}
	}
	if coverURL == "" && t.CoverURI != "" {
		coverURL = "https://" + t.CoverURI
	}

	json.NewEncoder(w).Encode(TrackResponse{
		ID:        t.ID,
		Title:     t.Title,
		Artist:    artistStr,
		Artists:   artists,
		ArtistIDs: artistIDs,
		Album:     album,
		AlbumID:   albumID,
		Duration:  t.DurationMs,
		CoverURL:  coverURL,
		Available: t.Available,
	})
}

// StartWebServer starts the HTTP server
func StartWebServer(port string) error {
	ctx := context.Background()
	ws, err := NewWebServer(ctx)
	if err != nil {
		return err
	}

	// Create a new ServeMux for routing
	mux := http.NewServeMux()

	// Serve static files with base path
	fs := http.FileServer(http.Dir("./static"))
	
	// Universal handler that works with both static BASE_PATH and dynamic X-Forwarded-Prefix
	universalHandler := func(w http.ResponseWriter, r *http.Request) {
		basePath := ws.getBasePath(r)
		path := r.URL.Path
		
		// If we have a base path from header but no static BASE_PATH,
		// we're in proxy mode - serve everything and inject base path into HTML
		if basePath != "" && ws.basePath == "" && ws.useProxyHeaders {
			// Proxy has already stripped the prefix, serve normally
			if path == "/" || path == "/index.html" {
				ws.handleIndex(w, r)
			} else {
				fs.ServeHTTP(w, r)
			}
			return
		}
		
		// If we have a static base path, check if request matches it
		if basePath != "" {
			// Handle exact base path (with or without trailing slash) - serve index
			if path == basePath || path == basePath+"/" {
				ws.handleIndex(w, r)
				return
			}
			
			// Handle sub-paths under base path - strip prefix and serve static files
			if strings.HasPrefix(path, basePath+"/") {
				http.StripPrefix(basePath, fs).ServeHTTP(w, r)
				return
			}
			
			// Path doesn't match our base path
			http.NotFound(w, r)
			return
		}
		
		// No base path - serve at root
		if path == "/" || path == "/index.html" {
			ws.handleIndex(w, r)
		} else {
			fs.ServeHTTP(w, r)
		}
	}
	
	// API handler wrapper that supports dynamic base path
	apiHandler := func(handler http.HandlerFunc) http.HandlerFunc {
		return enableCORS(func(w http.ResponseWriter, r *http.Request) {
			handler(w, r)
		})
	}
	
	if ws.basePath != "" {
		// Static base path mode (backwards compatible)
		// Register handler for base path without trailing slash
		mux.HandleFunc(ws.basePath, universalHandler)
		// Register handler for base path with trailing slash to catch all sub-paths
		mux.HandleFunc(ws.basePath+"/", universalHandler)
		
		// API endpoints with base path
		mux.HandleFunc(ws.basePath+"/api/search", apiHandler(ws.handleSearch))
		mux.HandleFunc(ws.basePath+"/api/download-url", apiHandler(ws.handleDownloadURL))
		mux.HandleFunc(ws.basePath+"/api/album-tracks", apiHandler(ws.handleAlbumTracks))
		mux.HandleFunc(ws.basePath+"/api/artist-tracks", apiHandler(ws.handleArtistTracks))
		mux.HandleFunc(ws.basePath+"/api/album-zip", apiHandler(ws.handleAlbumZip))
		mux.HandleFunc(ws.basePath+"/api/track-info", apiHandler(ws.handleTrackInfo))
		
		if ws.useProxyHeaders {
			log.Printf("Starting web server on http://localhost:%s%s (X-Forwarded-Prefix enabled)\n", port, ws.basePath)
		} else {
			log.Printf("Starting web server on http://localhost:%s%s\n", port, ws.basePath)
		}
	} else {
		// Root path mode - also supports X-Forwarded-Prefix if enabled
		mux.HandleFunc("/", universalHandler)
		
		// API endpoints at root
		mux.HandleFunc("/api/search", apiHandler(ws.handleSearch))
		mux.HandleFunc("/api/download-url", apiHandler(ws.handleDownloadURL))
		mux.HandleFunc("/api/album-tracks", apiHandler(ws.handleAlbumTracks))
		mux.HandleFunc("/api/artist-tracks", apiHandler(ws.handleArtistTracks))
		mux.HandleFunc("/api/album-zip", apiHandler(ws.handleAlbumZip))
		mux.HandleFunc("/api/track-info", apiHandler(ws.handleTrackInfo))
		
		if ws.useProxyHeaders {
			log.Printf("Starting web server on http://localhost:%s (X-Forwarded-Prefix enabled)\n", port)
		} else {
			log.Printf("Starting web server on http://localhost:%s\n", port)
		}
	}

	addr := ":" + port
	return http.ListenAndServe(addr, mux)
}
