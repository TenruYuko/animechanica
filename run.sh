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

# Run the container
podman run -d --name seanime \
  --security-opt label=type:container_runtime_t \
  --security-opt seccomp=unconfined \
  --cap-add=NET_ADMIN \
  -v /aeternae/configurations/animechanica/data:/data:Z \
  -v /aeternae/theater/anime/completed/:/media/anime:Z \
  -v /aeternae/theater/dl_anime/:/media/dl_anime:Z \
  -v /aeternae/library/dl_manga/:/media/dl_manga:Z \
  -v /aeternae/configurations/animechanica/qbittorrent:/root/.config/qBittorrent:Z \
  -v /aeternae/configurations/animechanica/mullvad:/mullvad:Z \
  -p 43211:43211 -p 8085:8085 \
  -e TZ=America/Los_Angeles \
  --restart unless-stopped \
  localhost/seanime:latest

echo "Seanime container started!"
echo "Seanime is available at: http://localhost:43211"
echo "qBittorrent is available at: http://localhost:8085"
echo ""
echo "To view logs:"
echo "podman logs -f seanime"
echo ""
echo "To stop the container:"
echo "podman stop seanime"
echo ""
echo "To restart the container:"
echo "podman restart seanime"
