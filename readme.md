# Yandex Music Player

A music player for Yandex Music written in Go. Available both as a CLI application and a Progressive Web App (PWA). Stream and download tracks from Yandex Music service.

## Features

- **PWA Web Interface** - Modern, accessible web application
- **CLI Interface** - Command-line music player for terminal users
- Search and play tracks from Yandex Music
- Playback controls (next, previous, pause/resume)
- Download tracks locally
- High-quality MP3 streaming
- Screen reader accessibility
- Offline capability (PWA)
- Responsive design for mobile and desktop

## Prerequisites

- Go 1.24 or higher
- Yandex Music account and token
- Python 3.x (if you want to use the helper script for token setup)

## Installation

1. Clone the repository
2. Install dependencies:
```sh
go mod download
```

## Setup

1. First, get your Yandex Music token (use @MusoadBot on Telegram)

2. Run the Python script to create `.env` file with your credentials:
```sh
pip install -U yandex-music
python get_id.py
```

3. Enter your token when prompted

## Usage

### Web App (PWA)

1. Start the web server:
```sh
go run web_server.go web_main.go
```

Or build and run:
```sh
go build -o web_server web_server.go web_main.go
./web_server
```

2. Open your browser and navigate to `http://localhost:8080`

3. Use the web interface to:
   - Search for tracks using the search bar
   - Click on search results to play tracks
   - Use the native HTML5 audio player controls
   - Navigate between tracks with Previous/Next buttons
   - Download tracks using the Download button

4. Install as PWA:
   - In Chrome/Edge: Click the install icon in the address bar
   - In mobile browsers: Use "Add to Home Screen" option

### API Endpoints

The web server exposes the following REST API endpoints:

- `GET /api/search?q=<query>` - Search for tracks
  - Returns: JSON array of track results with metadata
- `GET /api/download-url?id=<track_id>` - Get download URL for a track
  - Returns: JSON object with streaming URL

All endpoints return JSON and support CORS for browser access.

### CLI Application

Run the CLI player:
```sh
go run .
```

### CLI Controls

- `<search query>` - Search for a track
- `n` - Play next track
- `p` - Play previous track
- `pp` - Pause/Resume playback
- `dl` or `download` - Download current track
- `exit` or `ctrl+c` - Quit the player

## Project Structure

- `main.go` - CLI application entry point
- `web_main.go` - Web server entry point
- `web_server.go` - HTTP server and API handlers
- `player.go` - Music player implementation
- `play.go` - Audio streaming and playback (CLI only)
- `structs.go` - Data structures
- `static/` - Web application files
  - `index.html` - Main HTML page
  - `css/styles.css` - Styles with accessibility features
  - `js/app.js` - JavaScript application
  - `manifest.json` - PWA manifest
  - `sw.js` - Service worker for offline support
- `get_id.py` - Helper script for token setup

## PWA Features

The web application includes:

- **Progressive Web App** - Installable on desktop and mobile devices
- **Offline Support** - Service worker caches static assets for offline access
- **Responsive Design** - Works seamlessly on mobile, tablet, and desktop
- **Accessibility** - Full ARIA labels and screen reader support
- **Native Audio Player** - Uses HTML5 audio element for maximum compatibility
- **Touch-Friendly** - Optimized for touch and mouse input
- **Dark Mode** - Respects system color scheme preferences
- **Keyboard Navigation** - Full keyboard support for accessibility

### Accessibility Features

- ARIA labels on all interactive elements
- Semantic HTML structure
- Screen reader announcements for status updates
- High contrast UI elements
- Focus indicators for keyboard navigation
- Reduced motion support for users with vestibular disorders
- Proper heading hierarchy
- Alternative text for images

## Dependencies

### Backend
- [godotenv](github.com/joho/godotenv) - Environment configuration
- [yamusic](pkg.botr.me/yamusic) - Yandex Music API client

### CLI Only
- [oto](github.com/ebitengine/oto/v3) - Audio playback
- [go-mp3](github.com/hajimehoshi/go-mp3) - MP3 decoding