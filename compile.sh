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
mkdir web
cp -r seanime-web/out/* web/

# Build the Go server (Linux/macOS default, can be adjusted for Windows)
go build -o seanime -trimpath -ldflags="-s -w"

cd /aeternae/functional/dockers/animechanica && TMPDIR=/aeternae/functional/tmp podman build --root /aeternae/functional/containers --runroot /aeternae/functional/containers/run -t seanime:latest .

echo -e "\nBuild complete! Web assets are in ./web and server binary is ./seanime"

# Ask if user wants to run the application
read -p "Would you like to run? (yes/no): " answer
if [ "${answer,,}" = "yes" ]; then
    sudo bash run.sh
fi
