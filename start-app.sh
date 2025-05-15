#!/bin/bash

# Set the data directory path
DATA_DIR="/aeternae/configurations/animechanica/data/"

# Create the data directory if it doesn't exist
mkdir -p "$DATA_DIR"

# Check if Node.js is installed
if ! command -v node &> /dev/null; then
    echo "Node.js is not installed. Please install Node.js to run the proxy server."
    exit 1
fi

# Create a temporary directory for the proxy server
TEMP_DIR=$(mktemp -d)
cd "$TEMP_DIR"

# Create package.json
cat > package.json << 'EOF'
{
  "name": "seanime-proxy",
  "version": "1.0.0",
  "description": "Proxy server for Seanime",
  "main": "proxy.js",
  "dependencies": {
    "express": "^4.18.2",
    "http-proxy-middleware": "^2.0.6"
  }
}
EOF

# Create proxy.js
cat > proxy.js << 'EOF'
const express = require('express');
const { createProxyMiddleware } = require('http-proxy-middleware');
const path = require('path');
const fs = require('fs');
const app = express();

const BACKEND_URL = 'http://localhost:43211';

// Configure proxy options with better error handling
const proxyOptions = {
  target: BACKEND_URL,
  changeOrigin: true,
  ws: true, // Enable WebSocket proxying
  onProxyRes: function(proxyRes, req, res) {
    // Handle CORS headers
    proxyRes.headers['Access-Control-Allow-Origin'] = '*';
    
    // Fix content-type issues that might cause JSON parsing errors
    if (proxyRes.headers['content-type'] && 
        proxyRes.headers['content-type'].includes('text/html') && 
        req.path.includes('/api/v1/manga')) {
      proxyRes.headers['content-type'] = 'application/json';
    }
  },
  onError: function(err, req, res) {
    console.error('Proxy error:', err);
    res.writeHead(500, {
      'Content-Type': 'application/json'
    });
    res.end(JSON.stringify({ error: 'Proxy error', message: err.message }));
  }
};

// Proxy API requests to the backend with special handling for manga endpoints
app.use('/api/v1/manga', createProxyMiddleware({
  ...proxyOptions,
  onProxyReq: function(proxyReq, req, res) {
    // Add specific headers for manga API requests
    proxyReq.setHeader('Accept', 'application/json');
    proxyReq.setHeader('Content-Type', 'application/json');
  }
}));

// Proxy other API requests to the backend
app.use('/api', createProxyMiddleware(proxyOptions));

// Proxy asset requests to the backend
app.use('/assets', createProxyMiddleware(proxyOptions));

// Proxy offline-assets requests to the backend
app.use('/offline-assets', createProxyMiddleware(proxyOptions));

// Special handling for auth callback
app.get('/auth/callback', (req, res) => {
  // Inject script to help Next.js recognize the current route
  const html = `
    <!DOCTYPE html>
    <html>
    <head>
      <title>Redirecting...</title>
      <script>
        // Store the hash fragment for the Next.js router
        if (window.location.hash) {
          sessionStorage.setItem('auth_callback_hash', window.location.hash);
        }
        // Redirect to the frontend app
        window.location.href = '/';
      </script>
    </head>
    <body>
      <p>Redirecting to application...</p>
    </body>
    </html>
  `;
  res.send(html);
});

// For all other requests, proxy to the backend
app.use('/', createProxyMiddleware(proxyOptions));

const PORT = 3000;
app.listen(PORT, () => {
  console.log(`Proxy server running on http://localhost:${PORT}`);
  console.log(`Forwarding requests to backend at ${BACKEND_URL}`);
});
EOF

# Install dependencies
npm install

# Start the backend with the specified data directory
echo "Starting Seanime backend..."
cd -
./seanime --datadir "$DATA_DIR" &
BACKEND_PID=$!

# Wait a moment for the backend to start
sleep 2

# Start the proxy server
echo "Starting proxy server on port 3000..."
cd "$TEMP_DIR"
node proxy.js

# Cleanup function
cleanup() {
    echo "Shutting down servers..."
    kill $BACKEND_PID
    exit 0
}

# Set up trap to catch termination signals
trap cleanup SIGINT SIGTERM

# Wait for the proxy server to exit
wait
