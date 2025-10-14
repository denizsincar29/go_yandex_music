#!/bin/bash
# Systemd service template file creator for Yandex Music Player
# This script creates a systemd service unit file for running ya_music_web

set -e

# Get current directory (where the script is run from)
CURRENT_DIR=$(pwd)

# Get current user and group
CURRENT_USER=$(whoami)
CURRENT_GROUP=$(id -gn)

# Define log directory
LOG_DIR="$CURRENT_DIR/logs"

# Create logs directory if it doesn't exist
mkdir -p "$LOG_DIR"

# Service file name
SERVICE_FILE="ya_music_web.service"

# Create the systemd service file
cat > "$SERVICE_FILE" << EOF
[Unit]
Description=Yandex Music Player Web Server
After=network.target

[Service]
Type=simple
User=$CURRENT_USER
Group=$CURRENT_GROUP
WorkingDirectory=$CURRENT_DIR
ExecStart=$CURRENT_DIR/ya_music_web
Restart=on-failure
RestartSec=5s

# Logging
StandardOutput=append:$LOG_DIR/ya_music_web.log
StandardError=append:$LOG_DIR/ya_music_web.error.log

# Security settings
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

echo "Systemd service file created: $SERVICE_FILE"
echo ""
echo "Service details:"
echo "  User: $CURRENT_USER"
echo "  Group: $CURRENT_GROUP"
echo "  Working Directory: $CURRENT_DIR"
echo "  Executable: $CURRENT_DIR/ya_music_web"
echo "  Log Directory: $LOG_DIR"
echo "  Standard Output: $LOG_DIR/ya_music_web.log"
echo "  Standard Error: $LOG_DIR/ya_music_web.error.log"
echo ""
echo "To install and start the service:"
echo "  sudo cp $SERVICE_FILE /etc/systemd/system/"
echo "  sudo systemctl daemon-reload"
echo "  sudo systemctl enable ya_music_web.service"
echo "  sudo systemctl start ya_music_web.service"
echo ""
echo "To check service status:"
echo "  sudo systemctl status ya_music_web.service"
echo ""
echo "To view logs:"
echo "  tail -f $LOG_DIR/ya_music_web.log"
echo "  tail -f $LOG_DIR/ya_music_web.error.log"
echo ""
