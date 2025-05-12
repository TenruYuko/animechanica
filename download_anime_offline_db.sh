#!/bin/bash
# Downloads the latest anime-offline-database.json from the official GitHub repo
set -e

TARGET_DIR="/aeternae/functional/dockers/animechanica/external"
mkdir -p "$TARGET_DIR"

curl -L "https://raw.githubusercontent.com/manami-project/anime-offline-database/master/anime-offline-database.json" -o "$TARGET_DIR/anime-offline-database.json"
echo "Downloaded anime-offline-database.json to $TARGET_DIR"
