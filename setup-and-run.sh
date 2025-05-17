#!/bin/bash

set -e

# Create necessary directories
mkdir -p ./data ./qbittorrent/config ./gluetun

echo "=== Seanime Docker Environment Setup ==="
echo "This script will set up and run Seanime in a Docker environment with Mullvad VPN using Gluetun."

# Update docker-compose.yml with user inputs
echo ""
echo "=== Mullvad VPN Configuration ==="
read -p "Enter your Mullvad account number: " mullvad_account
echo ""

echo "=== VPN Location Configuration ==="
read -p "Enter your preferred VPN server country (e.g., usa, netherlands, sweden): " server_country
server_country=${server_country:-usa}  # Default to USA if not specified
echo ""

echo "=== Path Configuration ==="
echo "Please provide the full paths to your anime collection and download folders."
read -p "Path to anime collection (e.g., /home/user/anime): " anime_path
read -p "Path to downloads folder (e.g., /home/user/downloads): " downloads_path
echo ""

# Replace placeholders in docker-compose.yml
sed -i "s|OPENVPN_USER=mullvad_username|OPENVPN_USER=$mullvad_account|g" docker-compose.yml
sed -i "s|SERVER_COUNTRIES=usa|SERVER_COUNTRIES=$server_country|g" docker-compose.yml
sed -i "s|/path/to/anime|$anime_path|g" docker-compose.yml
sed -i "s|/path/to/downloads|$downloads_path|g" docker-compose.yml

echo "=== Starting Docker Containers ==="
docker-compose up -d --build

echo ""
echo "=== Setup Complete! ==="
echo "Seanime web interface: http://localhost:3000"
echo "qBittorrent web UI: http://localhost:8080 (default login: admin/adminadmin)"
echo "Gluetun health server: http://localhost:9999/healthcheck"
echo ""
echo "For more information and troubleshooting, refer to the DOCKER_README.md file."
