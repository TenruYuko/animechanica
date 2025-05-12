#!/usr/bin/env bash

# Compile script for animechanica
# Checks Go version and builds the project

set -e

REQUIRED_GO_VERSION="1.24.1"

# Function to compare Go versions
version_gte() {
  # returns 0 if $1 >= $2
  [ "$1" = "$2" ] && return 0
  local IFS=.
  local i ver1=($1) ver2=($2)
  # fill empty fields in ver1 with zeros
  for ((i=${#ver1[@]}; i<${#ver2[@]}; i++)); do
      ver1[i]=0
  done
  for ((i=0; i<${#ver1[@]}; i++)); do
      if [[ -z ${ver2[i]} ]]; then
          # fill empty fields in ver2 with zeros
          ver2[i]=0
      fi
      if ((10#${ver1[i]} > 10#${ver2[i]})); then
          return 0
      fi
      if ((10#${ver1[i]} < 10#${ver2[i]})); then
          return 1
      fi
  done
  return 0
}

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')

if ! version_gte "$GO_VERSION" "$REQUIRED_GO_VERSION"; then
  echo "Error: Go version $REQUIRED_GO_VERSION or higher is required. Current version: $GO_VERSION"
  exit 1
fi

echo "Go version $GO_VERSION detected."

# Build Go backend
printf '\n[1/4] Building Go backend...\n'
go build ./...
if [ $? -eq 0 ]; then
  echo "Go backend build succeeded."
else
  echo "Go backend build failed."
  exit 1
fi

# Build Web Frontend (Next.js)
printf '\n[2/4] Installing dependencies and building web frontend...\n'
cd seanime-web
npm install
if [ $? -ne 0 ]; then
  echo "npm install failed in seanime-web."
  exit 1
fi
npm run build
if [ $? -eq 0 ]; then
  echo "Web frontend build succeeded."
else
  echo "Web frontend build failed."
  exit 1
fi
cd ..

# Build Desktop Frontend (Tauri)
printf '\n[3/4] Installing dependencies and building desktop frontend...\n'
cd seanime-desktop
npm install
if [ $? -ne 0 ]; then
  echo "npm install failed in seanime-desktop."
  exit 1
fi
npm run build
if [ $? -eq 0 ]; then
  echo "Desktop frontend build succeeded."
else
  echo "Desktop frontend build failed."
  exit 1
fi
cd ..

echo '\n[4/4] All components built successfully.'
