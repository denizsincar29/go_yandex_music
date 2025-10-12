#!/bin/bash
# Build script for Yandex Music Player

set -e

echo "Building Yandex Music Player..."

# Build Web Server
echo "Building Web Server..."
go build -o ya_music_web web_server.go web_main.go

echo ""
echo "Build complete!"
echo ""
echo "To run Web Server: ./ya_music_web"
echo "Then open http://localhost:8080 in your browser"
echo ""
echo "Note: CLI version requires audio libraries (ALSA on Linux)"
echo "To build CLI: go build -o ya_music_cli main.go player.go play.go structs.go help.go update.go decoder_wrapper.go"
echo ""
