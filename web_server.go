package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"pkg.botr.me/yamusic"
)

// WebServer handles HTTP requests for the web interface
type WebServer struct {
	client *yamusic.Client
	ctx    context.Context
}

// TrackResponse represents a track in API responses
type TrackResponse struct {
	ID        int      `json:"id"`
	Title     string   `json:"title"`
	Artist    string   `json:"artist"`
	Artists   []string `json:"artists"`
	Album     string   `json:"album,omitempty"`
	Duration  int      `json:"duration"`
	CoverURL  string   `json:"coverUrl,omitempty"`
	Available bool     `json:"available"`
}

// SearchResponse represents search results
type SearchResponse struct {
	Tracks []TrackResponse `json:"tracks"`
	Total  int             `json:"total"`
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
	err := godotenv.Load()
	if err != nil {
		return nil, err
	}
	token := os.Getenv("YA_MUSIC_TOKEN")
	uid, err := strconv.Atoi(os.Getenv("YA_MUSIC_ID"))
	if err != nil {
		return nil, err
	}
	client := yamusic.NewClient(yamusic.AccessToken(uid, token))
	return &WebServer{client: client, ctx: ctx}, nil
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

	s, resp, err := ws.client.Search().Tracks(ws.ctx, query, &yamusic.SearchOptions{Page: 0, NoCorrect: false})
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

	results := s.Result.Tracks.Results
	tracks := make([]TrackResponse, len(results))
	for i, result := range results {
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

		album := ""
		coverURL := ""
		if len(result.Albums) > 0 {
			album = result.Albums[0].Title
			if result.Albums[0].CoverURI != "" {
				coverURL = "https://" + result.Albums[0].CoverURI
			}
		}

		tracks[i] = TrackResponse{
			ID:        result.ID,
			Title:     result.Title,
			Artist:    artistStr,
			Artists:   artists,
			Album:     album,
			Duration:  result.DurationMs,
			CoverURL:  coverURL,
			Available: result.Available,
		}
	}

	json.NewEncoder(w).Encode(SearchResponse{
		Tracks: tracks,
		Total:  len(tracks),
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

// StartWebServer starts the HTTP server
func StartWebServer(port string) error {
	ctx := context.Background()
	ws, err := NewWebServer(ctx)
	if err != nil {
		return err
	}

	// Serve static files
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// API endpoints
	http.HandleFunc("/api/search", enableCORS(ws.handleSearch))
	http.HandleFunc("/api/download-url", enableCORS(ws.handleDownloadURL))

	addr := ":" + port
	log.Printf("Starting web server on http://localhost%s\n", addr)
	return http.ListenAndServe(addr, nil)
}
