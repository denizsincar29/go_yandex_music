# Yandex Music Player

A music player for Yandex Music written in Go. Available both as a CLI application and a Progressive Web App (PWA). Stream and download tracks from Yandex Music service.

## Features

- **PWA Web Interface** - Modern, accessible web application
- **CLI Interface** - Command-line music player for terminal users
- Search for tracks, albums, and artists from Yandex Music
- Browse album tracks and artist discographies
- Playback controls (next, previous, pause/resume)
- Media key support (hardware next/previous buttons)
- Download tracks locally (actual file download, not streaming)
- Spelling correction for search queries
- High-quality MP3 streaming
- Screen reader accessibility with semantic headings
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

## Building

### Quick Build (Recommended)

Use the build script to build the web server:
```sh
./build.sh
```

This will create the `ya_music_web` executable in the root directory.

### Manual Build

**Important:** You cannot run `go build` from the root directory. Navigate to the specific application directory (`cmd/cli` or `cmd/web`) to build.

**Web Server (No special requirements):**
```sh
cd cmd/web
go build -o ../../ya_music_web
```

**CLI Application (Requires audio libraries):**
```sh
cd cmd/cli
go build -o ../../ya_music_cli
```

**Note:** The CLI version requires platform-specific audio libraries:
- Linux: ALSA development libraries (`libasound2-dev` on Ubuntu/Debian)
- macOS: CoreAudio (included by default)
- Windows: Windows Audio API (included by default)

## Usage

### Web App (PWA)

1. Start the web server:
```sh
cd cmd/web
go run .
```

Or build and run:
```sh
./build.sh
./ya_music_web
```

Or build manually:
```sh
cd cmd/web
go build -o ../../ya_music_web
cd ../..
./ya_music_web
```

2. Open your browser and navigate to `http://localhost:8080`

3. Use the web interface to:
   - Search for tracks, albums, and artists using the search bar
   - Click on tracks to play them instantly
   - Navigate to albums by clicking album names in track listings
   - Browse artist tracks by clicking on artist results
   - Use hardware media keys (Next/Previous) for navigation
   - Use the native HTML5 audio player controls
   - Navigate between tracks with Previous/Next buttons
   - Download tracks using the Download button (triggers actual file download)

4. Install as PWA:
   - In Chrome/Edge: Click the install icon in the address bar
   - In mobile browsers: Use "Add to Home Screen" option

### API Endpoints

The web server exposes the following REST API endpoints:

- `GET /api/search?q=<query>` - Search for tracks, albums, and artists
  - Returns: JSON object with tracks, albums, artists arrays and metadata
  - Includes spelling correction information if applicable
- `GET /api/download-url?id=<track_id>` - Get download URL for a track
  - Returns: JSON object with streaming URL
- `GET /api/album-tracks?id=<album_id>&name=<album_name>` - Get tracks from an album
  - Returns: JSON object with array of tracks
- `GET /api/artist-tracks?id=<artist_id>&name=<artist_name>` - Get tracks by an artist
  - Returns: JSON object with array of tracks

All endpoints return JSON and support CORS for browser access.

### Reverse Proxy Configuration

If you're hosting the web app behind a reverse proxy (e.g., Apache, Nginx) at a subpath, you need to set the `BASE_PATH` environment variable.

**Example: Hosting at `denizsincar.ru/music/`**

1. Set the `BASE_PATH` environment variable:
```sh
export BASE_PATH=/music
```

Or add it to your `.env` file:
```
BASE_PATH=/music
```

2. Configure your reverse proxy to forward requests to the Go server:

**Apache Example:**
```apache
ProxyPass /music/ http://localhost:8080/music/
ProxyPassReverse /music/ http://localhost:8080/music/
```

**Nginx Example:**
```nginx
location /music/ {
    proxy_pass http://localhost:8080/music/;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}
```

3. Start the web server normally - it will automatically use the `BASE_PATH`:
```sh
./ya_music_web
```

The application will automatically adjust all URLs (static files, API endpoints) to work with the specified base path.

**Note:** The `BASE_PATH` should start with `/` and NOT end with `/` (e.g., `/music`, not `music` or `/music/`).

### CLI Application

Run the CLI player:
```sh
cd cmd/cli
go run .
```

Or build and run (requires ALSA/audio libraries on Linux):
```sh
cd cmd/cli
go build -o ../../ya_music_cli
cd ../..
./ya_music_cli
```

**Note:** The CLI version requires audio libraries (ALSA on Linux, CoreAudio on macOS, etc.)

### CLI Controls

- `<search query>` - Search for a track
- `n` - Play next track
- `p` - Play previous track
- `pp` - Pause/Resume playback
- `dl` or `download` - Download current track
- `exit` or `ctrl+c` - Quit the player

## Project Structure

```
.
├── cmd/
│   ├── cli/                    # CLI application
│   │   ├── main.go            # CLI entry point
│   │   ├── player.go          # Music player implementation
│   │   ├── play.go            # Audio streaming and playback
│   │   ├── help.go            # CLI help and welcome messages
│   │   ├── update.go          # Auto-update functionality
│   │   └── decoder_wrapper.go # MP3 decoder wrapper
│   └── web/                    # Web application
│       ├── web_main.go        # Web server entry point
│       └── web_server.go      # HTTP server and API handlers
├── internal/
│   └── common/                 # Shared code
│       └── structs.go         # Common data structures
├── static/                     # Web application files
│   ├── index.html             # Main HTML page
│   ├── css/styles.css         # Styles with accessibility features
│   ├── js/app.js              # JavaScript application
│   ├── manifest.json          # PWA manifest
│   └── sw.js                  # Service worker for offline support
├── build.sh                    # Build script
└── get_id.py                  # Helper script for token setup
```

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