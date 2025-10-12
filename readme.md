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

## Dependencies

- [oto](github.com/ebitengine/oto/v3) - Audio playback
- [go-mp3](github.com/hajimehoshi/go-mp3) - MP3 decoding
- [godotenv](github.com/joho/godotenv) - Environment configuration
- [yamusic](pkg.botr.me/yamusic) - Yandex Music API client