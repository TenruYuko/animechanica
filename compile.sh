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
cp -r seanime-web/out/* web/

# Build the Go server (Linux/macOS default, can be adjusted for Windows)
go build -o seanime -trimpath -ldflags="-s -w"

echo "\nBuild complete! Web assets are in ./web and server binary is ./seanime"
