# Quick Start Guide - Yandex Music PWA

This guide will help you get the Yandex Music PWA up and running quickly.

## Prerequisites

1. Go 1.24 or higher installed
2. Yandex Music account and token

## Getting Your Token

1. Go to Telegram and message [@MusoadBot](https://t.me/MusoadBot)
2. Follow the bot's instructions to get your Yandex Music token

## Setup (5 minutes)

1. **Clone the repository:**
   ```bash
   git clone https://github.com/denizsincar29/go_yandex_music.git
   cd go_yandex_music
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   ```

3. **Create your credentials file:**
   
   Option A - Using the helper script:
   ```bash
   pip install -U yandex-music
   python get_id.py
   ```
   
   Option B - Manual setup:
   ```bash
   cp .env.example .env
   # Edit .env with your credentials
   ```

4. **Build the web server:**
   ```bash
   ./build.sh
   ```
   
   Or manually:
   ```bash
   go build -o ya_music_web web_server.go web_main.go
   ```

5. **Run the server:**
   ```bash
   ./ya_music_web
   ```

6. **Open your browser:**
   - Navigate to `http://localhost:8080`
   - Start searching and playing music!

## Installing as PWA

### Desktop (Chrome/Edge/Brave)
1. Click the install icon (⊕) in the address bar
2. Click "Install"
3. The app will open in its own window

### Mobile (iOS/Android)
1. Open the menu (⋮ or share button)
2. Select "Add to Home Screen"
3. Tap "Add"
4. Launch from your home screen

## Features

- 🔍 Search for any track, artist, or album
- ▶️ Stream music with the native HTML5 player
- ⬇️ Download tracks for offline listening
- 📱 Works on mobile, tablet, and desktop
- ♿ Full screen reader support
- 🌓 Dark mode support
- 📶 Offline capability (static files cached)

## Troubleshooting

**Server won't start:**
- Make sure port 8080 is not already in use
- Check that your `.env` file exists and has valid credentials

**Search not working:**
- Verify your token is valid in `.env`
- Check your internet connection
- Look at the browser console for error messages

**Audio won't play:**
- Ensure your browser supports HTML5 audio (all modern browsers do)
- Check browser console for CORS or network errors
- Try a different track

## Support

For issues or questions, please open an issue on GitHub.

Enjoy your music! 🎵
