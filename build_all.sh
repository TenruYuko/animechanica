#!/usr/bin/env bash
# Seanime Build Script (follows DEVELOPMENT_AND_BUILD.md)
# Usage: bash build_all.sh
set -e

cd "$(dirname "$0")"

# Colors for output
green='\033[0;32m'
red='\033[0;31m'
nc='\033[0m' # No Color

echo -e "${green}==> 1. Building the web interface (Next.js)${nc}"
cd seanime-web

# Remove .next directory to avoid permission issues
if [ -d ".next" ]; then
  echo -e "${green}Removing old .next build cache...${nc}"
  rm -rf .next
fi

npm run build

if [ ! -d "out" ]; then
  echo -e "${red}ERROR: Web build failed, 'out' directory not found.${nc}"
  exit 1
fi
cd ..

echo -e "${green}==> 2. Moving built web interface to root 'web' directory${nc}"
if [ -d "web" ]; then
  echo -e "${green}Removing old root 'web' directory...${nc}"
  rm -rf web
fi
mv seanime-web/out web

echo -e "${green}==> 3. Building the Go backend${nc}"
# Default: Linux/macOS build. For Windows, see DEVELOPMENT_AND_BUILD.md
if go build -o seanime -trimpath -ldflags="-s -w"; then
  echo -e "${green}Go backend built successfully!${nc}"
else
  echo -e "${red}ERROR: Go backend build failed.${nc}"
  exit 1
fi

echo -e "${green}==> Build completed successfully!${nc}"
echo -e "${green}Backend: ./seanime${nc}"
echo -e "${green}Frontend: ./web (static files)${nc}"
