#!/bin/bash
set -e

# Build the web interface
cd seanime-web
npm install
npm run build
cd ..

# Move the built web assets to the root-level web directory (if needed)
if [ -d "web" ]; then
  rm -rf web
fi
cp -r seanime-webc/out web

# Also copy built web assets to the data assets directory for the server
DATA_ASSETS_DIR="/aeternae/configurations/animechanica/data/assets"
sudo mkdir -p "$DATA_ASSETS_DIR"
sudo cp -r seanime-web/out/* "$DATA_ASSETS_DIR/"

# Build the Go server (Linux/macOS default, can be adjusted for Windows)
go build -o seanime -trimpath -ldflags="-s -w"

echo "\nBuild complete! Web assets are in ./web and server binary is ./seanime"
