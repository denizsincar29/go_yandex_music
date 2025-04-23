# Yandex Music CLI Player

A command-line music player for Yandex Music written in Go. Stream and download tracks from Yandex Music service.

## Features

- Search and play tracks from Yandex Music
- Playback controls (next, previous, pause/resume)
- Download tracks locally
- High-quality MP3 streaming
- Simple command-line interface

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

Run the player:
```sh
go run .
```

### Controls

- `s <search query>` - Search for a track
- `n` - Play next track
- `p` - Play previous track
- `pp` - Pause/Resume playback
- `dl` or `download` - Download current track
- `exit` or `ctrl+c` - Quit the player

## Project Structure

- `main.go` - Main application entry point
- `player.go` - Music player implementation
- `play.go` - Audio streaming and playback
- `structs.go` - Data structures
- `get_id.py` - Helper script for token setup

## Dependencies

- [oto](github.com/ebitengine/oto/v3) - Audio playback
- [go-mp3](github.com/hajimehoshi/go-mp3) - MP3 decoding
- [godotenv](github.com/joho/godotenv) - Environment configuration
- [yamusic](pkg.botr.me/yamusic) - Yandex Music API client