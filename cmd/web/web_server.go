package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"pkg.botr.me/yamusic"
)

// WebServer handles HTTP requests for the web interface
type WebServer struct {
	client   *yamusic.Client
	ctx      context.Context
	basePath string
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
	
	client := yamusic.NewClient(yamusic.AccessToken(uid, token))
	return &WebServer{client: client, ctx: ctx, basePath: basePath}, nil
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
	
	// Inject base path using a <base> tag and window.BASE_PATH variable
	basePath := ws.basePath
	if basePath == "" {
		basePath = "/"
	} else if basePath[len(basePath)-1] != '/' {
		basePath = basePath + "/"
	}
	
	baseTag := "<base href=\"" + basePath + "\">\n    "
	basePathScript := "<script>window.BASE_PATH = '" + ws.basePath + "';</script>\n    "
	
	// Insert after <head> tag
	content = strings.Replace(content, "<head>\n", "<head>\n    "+baseTag+basePathScript, 1)
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(content))
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
	if ws.basePath != "" {
		// Strip base path before serving static files
		mux.Handle(ws.basePath+"/", http.StripPrefix(ws.basePath, fs))
		// Handle index specifically to inject base path
		mux.HandleFunc(ws.basePath, ws.handleIndex)
		// API endpoints with base path
		mux.HandleFunc(ws.basePath+"/api/search", enableCORS(ws.handleSearch))
		mux.HandleFunc(ws.basePath+"/api/download-url", enableCORS(ws.handleDownloadURL))
		mux.HandleFunc(ws.basePath+"/api/album-tracks", enableCORS(ws.handleAlbumTracks))
		mux.HandleFunc(ws.basePath+"/api/artist-tracks", enableCORS(ws.handleArtistTracks))
		log.Printf("Starting web server on http://localhost:%s%s\n", port, ws.basePath)
	} else {
		// No base path - serve at root
		mux.Handle("/", fs)
		// Handle index specifically to inject base path
		mux.HandleFunc("/index.html", ws.handleIndex)
		// Also handle root path
		indexHandler := func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				ws.handleIndex(w, r)
			} else {
				fs.ServeHTTP(w, r)
			}
		}
		mux.HandleFunc("/", indexHandler)
		// API endpoints at root
		mux.HandleFunc("/api/search", enableCORS(ws.handleSearch))
		mux.HandleFunc("/api/download-url", enableCORS(ws.handleDownloadURL))
		mux.HandleFunc("/api/album-tracks", enableCORS(ws.handleAlbumTracks))
		mux.HandleFunc("/api/artist-tracks", enableCORS(ws.handleArtistTracks))
		log.Printf("Starting web server on http://localhost:%s\n", port)
	}

	addr := ":" + port
	return http.ListenAndServe(addr, mux)
}
