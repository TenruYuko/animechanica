#!/bin/bash
set -e

echo "Starting Seanime container..."

# Function to kill processes using specific ports
kill_process_on_port() {
  local port=$1
  echo "Checking for processes using port $port..."
  
  # Find process IDs using the port
  local pids=$(sudo lsof -t -i:$port 2>/dev/null)
  
  if [ -n "$pids" ]; then
    echo "Found processes using port $port. Stopping them..."
    for pid in $pids; do
      echo "Killing process $pid"
      sudo kill -9 $pid 2>/dev/null || true
    done
    echo "Processes on port $port stopped."
  else
    echo "No processes found using port $port."
  fi
}

# Stop any processes using our required ports
kill_process_on_port 43211
kill_process_on_port 8085

# Stop any existing Podman containers using our ports
echo "Stopping any Podman containers using our ports..."
podman container ls --format "{{.ID}} {{.Ports}}" | grep -E "43211|8085" | awk '{print $1}' | xargs -r podman stop || true

# Check if our container already exists
if podman container exists seanime; then
  echo "Container already exists. Stopping and removing..."
  podman stop seanime || true
  podman rm seanime || true
fi
