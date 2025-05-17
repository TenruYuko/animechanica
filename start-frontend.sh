#!/bin/bash

echo "Starting Seanime Mirror Frontend on port 43210..."
echo "This frontend will connect to the backend at 10.147.20.1:43211"

# Change to the frontend directory
cd "$(dirname "$0")/seanime-web"

# Check if the backend is accessible
echo "Checking if backend is accessible..."
curl -s http://10.147.20.1:43211/api/v1/settings > /dev/null
if [ $? -ne 0 ]; then
  echo "Warning: Backend at 10.147.20.1:43211 doesn't seem to be accessible."
  echo "Make sure the backend is running before continuing."
  read -p "Press Enter to continue anyway, or Ctrl+C to cancel..." 
fi

# Install dependencies if needed
echo "Checking and installing dependencies..."
npm install --no-fund --no-audit

# Set environment variables directly
export NEXT_PUBLIC_PLATFORM="mirror"
export NEXT_PUBLIC_BACKEND_URL="http://10.147.20.1:43211"
export NEXT_PUBLIC_BACKEND_HOST="10.147.20.1"
export NEXT_PUBLIC_BACKEND_PORT="43211"

# Run the Next.js dev server directly
echo "Starting Next.js on port 43210..."
npx next dev --hostname=0.0.0.0 --port=43210 --turbo
