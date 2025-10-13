#!/bin/bash
# Build script for Yandex Music Player

set -e

echo "Building Yandex Music Player..."

# Build Web Server
echo "Building Web Server..."
cd cmd/web
go build -o ../../ya_music_web
cd ../..

echo ""
echo "Build complete!"
echo ""
echo "To run Web Server: ./ya_music_web"
echo "Then open http://localhost:8080 in your browser"
echo ""
echo "Note: CLI version requires audio libraries (ALSA on Linux)"
echo "To build CLI (requires ALSA/audio libs):"
echo "  cd cmd/cli && go build -o ../../ya_music_cli"
echo ""
