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

// handleAlbumTracks handles requests for album tracks by searching for the album name
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

	// Search for the album name to get tracks
	s, resp, err := ws.client.Search().Tracks(ws.ctx, albumName, &yamusic.SearchOptions{Page: 0, NoCorrect: false})
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

	// Filter tracks by album ID
	var filteredTracks []TrackResponse
	for _, result := range s.Result.Tracks.Results {
		if len(result.Albums) > 0 && result.Albums[0].ID == albumID {
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

			coverURL := ""
			if result.Albums[0].CoverURI != "" {
				coverURL = "https://" + result.Albums[0].CoverURI
			}

			filteredTracks = append(filteredTracks, TrackResponse{
				ID:        result.ID,
				Title:     result.Title,
				Artist:    artistStr,
				Artists:   artists,
				ArtistIDs: artistIDs,
				Album:     result.Albums[0].Title,
				AlbumID:   result.Albums[0].ID,
				Duration:  result.DurationMs,
				CoverURL:  coverURL,
				Available: result.Available,
			})
		}
	}

	json.NewEncoder(w).Encode(SearchResponse{
		Tracks: filteredTracks,
		Total:  len(filteredTracks),
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
	http.HandleFunc("/api/album-tracks", enableCORS(ws.handleAlbumTracks))
	http.HandleFunc("/api/artist-tracks", enableCORS(ws.handleArtistTracks))

	addr := ":" + port
	log.Printf("Starting web server on http://localhost%s\n", addr)
	return http.ListenAndServe(addr, nil)
}
