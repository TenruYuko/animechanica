#!/bin/bash

# Set the data directory path
DATA_DIR="${DATA_DIR:-/app/data}"

# Create the data directory if it doesn't exist
mkdir -p "$DATA_DIR"

# Function to check if a port is in use
port_in_use() {
    netstat -tuln | grep -q ":$1 "
    return $?
}

# Function to kill processes using a specific port
kill_process_on_port() {
    local port=$1
    local pid=$(lsof -t -i:$port)
    if [ -n "$pid" ]; then
        echo "Killing process $pid using port $port"
        kill -9 $pid 2>/dev/null
        sleep 1
    fi
}

# Check if backend port is in use and kill the process
if port_in_use 43211; then
    echo "Port 43211 is already in use. Stopping existing backend process..."
    kill_process_on_port 43211
fi

# Check if frontend port is in use and kill the process
if port_in_use 3000; then
    echo "Port 3000 is already in use. Stopping existing frontend process..."
    kill_process_on_port 3000
fi

# Check if Node.js is installed
if ! command -v node &> /dev/null; then
    echo "Node.js is not installed. Please install Node.js to run the proxy server."
    exit 1
fi

# Create a temporary directory for the proxy server
TEMP_DIR=$(mktemp -d)
cd "$TEMP_DIR"

# Install dependencies
npm install

# Start the backend with the specified data directory
echo "Starting Seanime backend..."
cd -
export SEANIME_DATA_DIR="$DATA_DIR"
./seanime --datadir "$DATA_DIR" &
BACKEND_PID=$!

# Wait a moment for the backend to start
sleep 3

# Start the proxy server
echo "Starting proxy server on port 3000..."
cd "$TEMP_DIR"
node proxy.js &
PROXY_PID=$!

# Cleanup function for graceful shutdown
cleanup() {
    echo "Shutting down servers..."
    # Kill the backend process if it's running
    if [ -n "$BACKEND_PID" ] && kill -0 $BACKEND_PID 2>/dev/null; then
        echo "Stopping backend process..."
        kill -15 $BACKEND_PID 2>/dev/null
        sleep 1
        # Force kill if still running
        if kill -0 $BACKEND_PID 2>/dev/null; then
            kill -9 $BACKEND_PID 2>/dev/null
        fi
    fi
    
    # Kill the proxy process if needed
    if [ -n "$PROXY_PID" ] && kill -0 $PROXY_PID 2>/dev/null; then
        echo "Stopping proxy process..."
        kill -15 $PROXY_PID 2>/dev/null
    fi
    
    # Clean up temp directory
    if [ -d "$TEMP_DIR" ]; then
        echo "Cleaning up temporary files..."
        rm -rf "$TEMP_DIR"
    fi
    
    echo "Shutdown complete."
    exit 0
}

# Set up trap to catch termination signals
trap cleanup SIGINT SIGTERM EXIT

# Wait for the proxy server to exit
wait $PROXY_PID
