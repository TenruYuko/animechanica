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

# Create package.json with a properly terminated here-document
cat > package.json << 'EOFPACKAGE'
{
  "name": "seanime-proxy",
  "version": "1.0.0",
  "description": "Proxy server for Seanime",
  "main": "proxy.js",
  "dependencies": {
    "express": "^4.18.2",
    "http-proxy-middleware": "^2.0.6",
    "cookie-parser": "^1.4.6",
    "body-parser": "^1.20.2",
    "ws": "^8.13.0"
  }
}
EOFPACKAGE

# Create proxy.js
cat > proxy.js << 'EOFPROXYJS'
const express = require('express');
const { createProxyMiddleware } = require('http-proxy-middleware');
const cookieParser = require('cookie-parser');
const bodyParser = require('body-parser');
const path = require('path');
const fs = require('fs');
const app = express();

// Set up the backend URL
const BACKEND_URL = 'http://localhost:43211';

// Add middleware for parsing cookies and request bodies
app.use(cookieParser());
app.use(bodyParser.json({ limit: '50mb' }));
app.use(bodyParser.urlencoded({ extended: true, limit: '50mb' }));

// --- Dedicated video streaming proxy for /video-proxy and /api/v1/proxy routes ---

// Special handling for video streaming and proxying external content
app.use('/api/v1/proxy', async (req, res) => {
  // Set CORS headers for every response immediately
  res.setHeader('Access-Control-Allow-Origin', req.headers.origin || '*');
  res.setHeader('Access-Control-Allow-Credentials', 'true');
  res.setHeader('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, PATCH, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Range, X-Requested-With, Content-Type, Authorization, Referer, Origin, Accept, Cookie');

  // Handle OPTIONS preflight requests
  if (req.method === 'OPTIONS') {
    res.status(200).end();
    return;
  }

  // Extract the target URL
  let targetUrl = req.query.url;
  if (!targetUrl) {
    return res.status(400).json({ error: 'Missing URL parameter' });
  }

  // Special handling for kwikie.ru URLs (for .key files)
  const isKwikieUrl = targetUrl.includes('kwikie.ru');
  
  console.log(`Proxying request to: ${targetUrl}${isKwikieUrl ? ' (kwikie.ru request)' : ''}`);

  // Prepare headers to forward
  const forwardedHeaders = { ...req.headers };

  // If headers are provided via ?headers=, merge/override
  if (req.query.headers) {
    try {
      const extraHeaders = JSON.parse(req.query.headers);
      Object.assign(forwardedHeaders, extraHeaders);
    } catch (e) {
      // Ignore parse errors
    }
  }
  
  // For kwikie.ru requests, set the proper Referer header to kwik.si
  if (isKwikieUrl) {
    forwardedHeaders['referer'] = 'https://kwik.si';
    forwardedHeaders['origin'] = 'https://kwik.si';
    // Add additional headers that might be needed
    forwardedHeaders['user-agent'] = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36';
    // Clear any existing authorization headers that might interfere
    delete forwardedHeaders['authorization'];
    console.log('Using special headers for kwikie.ru request');
  }

  // Remove hop-by-hop headers and headers that should not be forwarded
  delete forwardedHeaders['host'];
  delete forwardedHeaders['connection'];
  delete forwardedHeaders['content-length'];
  delete forwardedHeaders['accept-encoding']; // Let Node handle compression

  // Forward cookies explicitly if present
  if (req.cookies && Object.keys(req.cookies).length > 0) {
    forwardedHeaders['cookie'] = Object.entries(req.cookies)
      .map(([key, value]) => `${key}=${value}`)
      .join('; ');
  }

  // Forward the request using http/https
  const http = targetUrl.startsWith('https') ? require('https') : require('http');
  const options = {
    method: req.method,
    headers: forwardedHeaders,
  };

  try {
    const proxyReq = http.request(targetUrl, options, proxyRes => {
      // Get response headers, excluding any CORS headers
      const responseHeaders = {};
      const isKwikieResponse = targetUrl.includes('kwikie.ru');
      
      if (isKwikieResponse) {
        console.log(`Got kwikie.ru response with status ${proxyRes.statusCode}`);
      }
      
      Object.entries(proxyRes.headers).forEach(([key, value]) => {
        // Skip CORS headers from remote
        if (!key.toLowerCase().startsWith('access-control-')) {
          // Handle cookies - strip Secure if not HTTPS
          if (key.toLowerCase() === 'set-cookie') {
            const cookies = Array.isArray(value) ? value : [value];
            const processedCookies = cookies.map(cookie => {
              return cookie.replace(/\s*Secure;?\s*/gi, '');
            });
            responseHeaders[key] = processedCookies;
          } else {
            responseHeaders[key] = value;
          }
        }
      });
      
      // Copy non-CORS headers to our response
      Object.entries(responseHeaders).forEach(([key, value]) => {
        res.setHeader(key, value);
      });
      
      // ALWAYS SET our own CORS headers AFTER copying other headers
      res.setHeader('Access-Control-Allow-Origin', req.headers.origin || '*');
      res.setHeader('Access-Control-Allow-Credentials', 'true');
      res.setHeader('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, PATCH, OPTIONS');
      res.setHeader('Access-Control-Allow-Headers', 'Range, X-Requested-With, Content-Type, Authorization, Referer, Origin, Accept, Cookie');
      
      // Set appropriate status code
      res.statusCode = proxyRes.statusCode;
      
      // Pipe the response data
      proxyRes.pipe(res);
    });

    proxyReq.on('error', err => {
      console.error('Proxy error:', err);
      if (!res.headersSent) {
        res.status(500).json({ error: 'Proxy error', message: err.message });
      }
    });

    // Pipe body data for POST/PUT/PATCH requests
    if (['POST', 'PUT', 'PATCH'].includes(req.method)) {
      req.pipe(proxyReq);
    } else {
      proxyReq.end();
    }
    
    // Handle client disconnect
    req.on('close', () => {
      try {
        proxyReq.destroy();
      } catch (e) {
        // Ignore any errors while destroying the request
      }
    });
  } catch (err) {
    console.error('Proxy setup error:', err);
    if (!res.headersSent) {
      res.status(500).json({ error: 'Proxy setup error', message: err.message });
    }
  }
});

// --- Dedicated video streaming proxy for /video-proxy route ---
app.get('/video-proxy', async (req, res) => {
  const videoUrl = req.query.url;
  if (!videoUrl) {
    res.status(400).json({ error: 'Missing video URL' });
    return;
  }
  try {
    const http = videoUrl.startsWith('https') ? require('https') : require('http');
    const videoReq = http.get(videoUrl, (videoRes) => {
      // Remove any CORS headers from the remote response
      delete videoRes.headers['access-control-allow-origin'];
      delete videoRes.headers['access-control-allow-credentials'];
      delete videoRes.headers['access-control-allow-methods'];
      delete videoRes.headers['access-control-allow-headers'];
      // Set CORS headers (force override)
      res.setHeader('Access-Control-Allow-Origin', req.headers.origin || '*');
      res.setHeader('Access-Control-Allow-Credentials', 'true');
      res.setHeader('Access-Control-Allow-Methods', 'GET, OPTIONS, POST, PUT, DELETE');
      res.setHeader('Access-Control-Allow-Headers', 'Range, X-Requested-With, Content-Type, Authorization');
      // Forward content headers
      if (videoRes.headers['content-type'])
        res.setHeader('Content-Type', videoRes.headers['content-type']);
      if (videoRes.headers['content-length'])
        res.setHeader('Content-Length', videoRes.headers['content-length']);
      if (videoRes.headers['accept-ranges'])
        res.setHeader('Accept-Ranges', videoRes.headers['accept-ranges']);
      if (videoRes.headers['content-range'])
        res.setHeader('Content-Range', videoRes.headers['content-range']);
      res.statusCode = videoRes.statusCode;
      videoRes.pipe(res);
    });
    videoReq.on('error', (err) => {
      console.error('Video proxy error:', err);
      if (!res.headersSent) {
        res.status(500).json({ error: 'Error streaming video' });
      }
    });
    req.on('close', () => videoReq.destroy());
  } catch (err) {
    console.error('Video proxy exception:', err);
    if (!res.headersSent) {
      res.status(500).json({ error: 'Video proxy exception' });
    }
  }
});
// --- End video streaming proxy ---
const proxyOptions = {
  target: BACKEND_URL,
  changeOrigin: true,
  ws: true, // Enable WebSocket proxying
  secure: false,
  followRedirects: true,
  cookieDomainRewrite: { '*': '' },
  cookiePathRewrite: { '*': '/' },
  preserveHeaderKeyCase: true,
  timeout: 60000, // Increase timeout for video streaming
  proxyTimeout: 60000,
  onProxyRes: function(proxyRes, req, res) {
    // Preserve cookies from the backend
    const proxyCookies = proxyRes.headers['set-cookie'];
    if (proxyCookies) {
      const cookies = Array.isArray(proxyCookies) ? proxyCookies : [proxyCookies];
      proxyRes.headers['set-cookie'] = cookies.map(cookie => {
        // Handle all cookie attributes properly
        // 1. Always remove Domain restriction
        // 2. If SameSite=None, change to Lax
        // 3. Always remove Secure flag since we're on HTTP
        let newCookie = cookie
          .replace(/Domain=[^;]+;/i, '')
          .replace(/SameSite=None;?/i, 'SameSite=Lax;')
          .replace(/\s*Secure;?\s*/gi, '');
        console.log('Modified cookie:', newCookie);
        return newCookie;
      });
    }
    
    // Handle CORS headers
    proxyRes.headers['Access-Control-Allow-Origin'] = req.headers.origin || '*';
    proxyRes.headers['Access-Control-Allow-Credentials'] = 'true';
    proxyRes.headers['Access-Control-Allow-Methods'] = 'GET, POST, PUT, DELETE, PATCH, OPTIONS';
    proxyRes.headers['Access-Control-Allow-Headers'] = 'X-Requested-With, Content-Type, Authorization';
    
    // Fix content-type issues that might cause JSON parsing errors
    if (req.path.includes('/api/v1/manga') && 
        proxyRes.headers['content-type'] && 
        proxyRes.headers['content-type'].includes('text/html')) {
      proxyRes.headers['content-type'] = 'application/json';
    }
  },
  onProxyReq: function(proxyReq, req, res) {
    // Copy cookies from client to backend
    if (req.cookies) {
      const cookieString = Object.entries(req.cookies)
        .filter(([key, value]) => typeof value === 'string' && value !== '')
        .map(([key, value]) => `${key}=${encodeURIComponent(value)}`)
        .join('; ');
      if (cookieString) {
        proxyReq.setHeader('Cookie', cookieString);
        console.log('[Proxy] Forwarding cookies:', cookieString);
      }
    }
    
    // Handle POST requests with JSON body
    if (req.method === 'POST' && req.body && Object.keys(req.body).length > 0) {
      const contentType = proxyReq.getHeader('Content-Type');
      let bodyData;
      
      if (contentType && contentType.includes('application/json')) {
        bodyData = JSON.stringify(req.body);
        proxyReq.setHeader('Content-Length', Buffer.byteLength(bodyData));
        proxyReq.write(bodyData);
      }
    }
  },
  onError: function(err, req, res) {
    console.error('Proxy error:', err);
    if (!res.headersSent) {
      res.writeHead(500, {
        'Content-Type': 'application/json'
      });
      res.end(JSON.stringify({ error: 'Proxy error', message: err.message }));
    }
  }
};

// Special handling for manga endpoints with custom error handling
app.use('/api/v1/manga/pages', (req, res, next) => {
  // Create a custom middleware to handle manga page requests
  const mangaPagesProxy = createProxyMiddleware({
    ...proxyOptions,
    selfHandleResponse: true, // Handle the response ourselves
    onProxyReq: function(proxyReq, req, res) {
      // Add specific headers for manga API requests
      proxyReq.setHeader('Accept', 'application/json, image/*, */*');
      
      // Copy cookies and handle POST body
      if (req.cookies) {
        const cookieString = Object.entries(req.cookies)
          .filter(([key, value]) => typeof value === 'string' && value !== '')
          .map(([key, value]) => `${key}=${encodeURIComponent(value)}`)
          .join('; ');
        if (cookieString) {
          proxyReq.setHeader('Cookie', cookieString);
          console.log('[Proxy] Forwarding cookies:', cookieString);
        }
      }
      
      if (req.method === 'POST' && req.body && Object.keys(req.body).length > 0) {
        const contentType = proxyReq.getHeader('Content-Type') || 'application/json';
        let bodyData;
        
        if (contentType.includes('application/json')) {
          bodyData = JSON.stringify(req.body);
          proxyReq.setHeader('Content-Length', Buffer.byteLength(bodyData));
          proxyReq.setHeader('Content-Type', 'application/json');
          proxyReq.write(bodyData);
        }
      }
    },
    onProxyRes: function(proxyRes, req, res) {
      // Handle the response
      let responseBody = '';
      proxyRes.on('data', function(chunk) {
        responseBody += chunk;
      });
      
      proxyRes.on('end', function() {
        if (!res.writableEnded) {
          // Copy headers from the proxied response
          Object.keys(proxyRes.headers).forEach(function(key) {
            res.setHeader(key, proxyRes.headers[key]);
          });
          
          // Set proper CORS headers
          res.setHeader('Access-Control-Allow-Origin', req.headers.origin || '*');
          res.setHeader('Access-Control-Allow-Credentials', 'true');
          
          // Check if the response is empty or has an error
          if (proxyRes.statusCode !== 200 || !responseBody || responseBody.includes('error')) {
            // Provide a fallback response with empty pages array instead of an error
            console.log('Manga pages error detected, providing fallback response');
            res.statusCode = 200;
            res.setHeader('Content-Type', 'application/json');
            res.end(JSON.stringify({
              pages: [],
              message: 'No pages found, but you can still access the manga viewer.'
            }));
          } else {
            // Pass through the successful response
            res.statusCode = proxyRes.statusCode;
            res.end(responseBody);
          }
        }
      });
    }
  });
  
  mangaPagesProxy(req, res, next);
});

// Special handling for manga chapters
app.use('/api/v1/manga/chapters', (req, res, next) => {
  // Create a custom middleware to handle manga chapter requests
  const mangaChaptersProxy = createProxyMiddleware({
    ...proxyOptions,
    selfHandleResponse: true, // Handle the response ourselves
    onProxyReq: function(proxyReq, req, res) {
      // Add specific headers for manga API requests
      proxyReq.setHeader('Accept', 'application/json, image/*, */*');
      
      // Copy cookies and handle POST body
      if (req.cookies) {
        const cookieString = Object.entries(req.cookies)
          .filter(([key, value]) => typeof value === 'string' && value !== '')
          .map(([key, value]) => `${key}=${encodeURIComponent(value)}`)
          .join('; ');
        if (cookieString) {
          proxyReq.setHeader('Cookie', cookieString);
          console.log('[Proxy] Forwarding cookies:', cookieString);
        }
      }
      
      if (req.method === 'POST' && req.body && Object.keys(req.body).length > 0) {
        const contentType = proxyReq.getHeader('Content-Type') || 'application/json';
        let bodyData;
        
        if (contentType.includes('application/json')) {
          bodyData = JSON.stringify(req.body);
          proxyReq.setHeader('Content-Length', Buffer.byteLength(bodyData));
          proxyReq.setHeader('Content-Type', 'application/json');
          proxyReq.write(bodyData);
        }
      }
    },
    onProxyRes: function(proxyRes, req, res) {
      // Handle the response
      let responseBody = '';
      proxyRes.on('data', function(chunk) {
        responseBody += chunk;
      });
      
      proxyRes.on('end', function() {
        if (!res.writableEnded) {
          // Copy headers from the proxied response
          Object.keys(proxyRes.headers).forEach(function(key) {
            res.setHeader(key, proxyRes.headers[key]);
          });
          
          // Set proper CORS headers
          res.setHeader('Access-Control-Allow-Origin', req.headers.origin || '*');
          res.setHeader('Access-Control-Allow-Credentials', 'true');
          
          // Check if the response is empty or has an error
          if (proxyRes.statusCode !== 200 || !responseBody || responseBody.includes('error')) {
            // Provide a fallback response with an empty chapters array instead of an error
            console.log('Manga chapters error detected, providing fallback response');
            res.statusCode = 200;
            res.setHeader('Content-Type', 'application/json');
            res.end(JSON.stringify({
              chapters: [],
              message: 'No chapters found, but you can still access the manga viewer.'
            }));
          } else {
            // Pass through the successful response
            res.statusCode = proxyRes.statusCode;
            res.end(responseBody);
          }
        }
      });
    }
  });
  
  mangaChaptersProxy(req, res, next);
});

// Handle other manga endpoints
app.use('/api/v1/manga', createProxyMiddleware({
  ...proxyOptions,
  onProxyReq: function(proxyReq, req, res) {
    // Add specific headers for manga API requests
    proxyReq.setHeader('Accept', 'application/json, image/*, */*');
    
    // Copy cookies and handle POST body
    if (req.cookies) {
      const cookieString = Object.entries(req.cookies)
        .filter(([key, value]) => typeof value === 'string' && value !== '')
        .map(([key, value]) => `${key}=${encodeURIComponent(value)}`)
        .join('; ');
      if (cookieString) {
        proxyReq.setHeader('Cookie', cookieString);
        console.log('[Proxy] Forwarding cookies:', cookieString);
      }
    }
    
    if (req.method === 'POST' && req.body && Object.keys(req.body).length > 0) {
      const contentType = proxyReq.getHeader('Content-Type') || 'application/json';
      let bodyData;
      
      if (contentType.includes('application/json')) {
        bodyData = JSON.stringify(req.body);
        proxyReq.setHeader('Content-Length', Buffer.byteLength(bodyData));
        proxyReq.setHeader('Content-Type', 'application/json');
        proxyReq.write(bodyData);
      }
    }
  }
}));

// Handle OPTIONS requests for CORS preflight
app.options('*', (req, res) => {
  res.header('Access-Control-Allow-Origin', req.headers.origin || '*');
  res.header('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, PATCH, OPTIONS');
  res.header('Access-Control-Allow-Headers', 'X-Requested-With, Content-Type, Authorization');
  res.header('Access-Control-Allow-Credentials', 'true');
  res.status(200).send();
});

// Special handling for video streaming with a simple implementation
app.use('/api/v1/proxy', async (req, res) => {
  // Set CORS headers for every response immediately
  res.setHeader('Access-Control-Allow-Origin', req.headers.origin || '*');
  res.setHeader('Access-Control-Allow-Credentials', 'true');
  res.setHeader('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, PATCH, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Range, X-Requested-With, Content-Type, Authorization, Referer, Origin, Accept, Cookie');

  // Handle OPTIONS preflight requests
  if (req.method === 'OPTIONS') {
    res.status(200).end();
    return;
  }

  // Extract the target URL
  const targetUrl = req.query.url;
  if (!targetUrl) {
    return res.status(400).json({ error: 'Missing URL parameter' });
  }

  // Prepare headers to forward
  const forwardedHeaders = { ...req.headers };

  // If headers are provided via ?headers=, merge/override
  if (req.query.headers) {
    try {
      const extraHeaders = JSON.parse(req.query.headers);
      Object.assign(forwardedHeaders, extraHeaders);
    } catch (e) {
      // Ignore parse errors
    }
  }

  // Remove hop-by-hop headers and headers that should not be forwarded
  delete forwardedHeaders['host'];
  delete forwardedHeaders['connection'];
  delete forwardedHeaders['content-length'];
  delete forwardedHeaders['accept-encoding']; // Let Node handle compression

  // Forward cookies explicitly if present
  if (req.cookies && Object.keys(req.cookies).length > 0) {
    forwardedHeaders['cookie'] = Object.entries(req.cookies)
      .map(([key, value]) => `${key}=${value}`)
      .join('; ');
  }

  // Forward the request using http/https
  const http = targetUrl.startsWith('https') ? require('https') : require('http');
  const options = {
    method: req.method,
    headers: forwardedHeaders,
  };

  const proxyReq = http.request(targetUrl, options, proxyRes => {
    // Copy headers from the proxied response, REMOVING any CORS headers
    const responseHeaders = {};
    Object.entries(proxyRes.headers).forEach(([key, value]) => {
      // Skip any CORS headers from the remote server
      if (!key.toLowerCase().startsWith('access-control-')) {
        responseHeaders[key] = value;
      }
    });
    
    // Copy non-CORS headers to our response
    Object.entries(responseHeaders).forEach(([key, value]) => {
      res.setHeader(key, value);
    });
    
    // SET our own CORS headers (overwrites any that might have been copied)
    res.setHeader('Access-Control-Allow-Origin', req.headers.origin || '*');
    res.setHeader('Access-Control-Allow-Credentials', 'true');
    res.setHeader('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, PATCH, OPTIONS');
    res.setHeader('Access-Control-Allow-Headers', 'Range, X-Requested-With, Content-Type, Authorization, Referer, Origin, Accept, Cookie');
    
    res.statusCode = proxyRes.statusCode;
    proxyRes.pipe(res);
  });

  proxyReq.on('error', err => {
    console.error('Proxy error:', err);
    if (!res.headersSent) {
      res.status(500).json({ error: 'Proxy error', message: err.message });
    }
  });

  // Pipe body for POST/PUT
  if (['POST', 'PUT', 'PATCH'].includes(req.method)) {
    req.pipe(proxyReq);
  } else {
    proxyReq.end();
  }
});

// Special handling for video streaming and proxying external content

// Disable console.log temporarily to suppress WebSocket upgrade messages
const originalConsoleLog = console.log;
console.log = function(message, ...args) {
  // Filter out WebSocket upgrade messages
  if (typeof message === 'string' && 
      (message.includes('Upgrading to WebSocket') || 
       message.includes('[HPM]'))) {
    return;
  }
  originalConsoleLog(message, ...args);
};

// Create a dedicated WebSocket server for the /api/v1/ws endpoint only
const wsProxy = createProxyMiddleware({
  target: BACKEND_URL,
  changeOrigin: true,
  ws: true,
  secure: false,
  // Explicitly set WebSocket to true only for this route
  pathRewrite: { '^/api/v1/ws': '/api/v1/ws' },
  onError: (err, req, res) => {
    console.error('WebSocket proxy error:', err);
    if (!res.headersSent && res.writeHead) {
      res.writeHead(500, {
        'Content-Type': 'application/json'
      });
      res.end(JSON.stringify({ error: 'WebSocket proxy error', message: err.message }));
    }
  },
  // Improved WebSocket handling
  onProxyReqWs: (proxyReq, req, socket, options, head) => {
    // Add a one-time connection established log
    socket.once('open', () => {
      console.error('WebSocket connection established'); // Use error to bypass our log filter
    });
    
    // Handle socket errors
    socket.on('error', (err) => {
      console.error('WebSocket socket error:', err);
    });
    
    // Handle socket close to prevent memory leaks
    socket.on('close', () => {
      console.error('WebSocket connection closed');
    });
  }
});

// Apply WebSocket proxy ONLY to the specific WebSocket path
app.use('/api/v1/ws', wsProxy);

// Handle Next.js events endpoint with a simple response
app.get('/events', (req, res) => {
  res.setHeader('Content-Type', 'text/plain');
  res.status(200).send('OK');
});



// Proxy API requests to the backend
app.use('/api', createProxyMiddleware(proxyOptions));

// Proxy asset requests to the backend
app.use('/assets', createProxyMiddleware(proxyOptions));

// Proxy offline-assets requests to the backend
app.use('/offline-assets', createProxyMiddleware(proxyOptions));

// For all other requests, create a separate proxy with WebSockets DISABLED
const httpProxy = createProxyMiddleware({
  target: BACKEND_URL,
  changeOrigin: true,
  ws: false, // Explicitly disable WebSockets for all other routes
  secure: false,
  followRedirects: true,
  cookieDomainRewrite: { '*': '' },
  cookiePathRewrite: { '*': '/' },
  preserveHeaderKeyCase: true,
  timeout: 60000,
  proxyTimeout: 60000,
  onError: (err, req, res) => {
    console.error('HTTP proxy error:', err);
    if (!res.headersSent) {
      res.writeHead(500, {
        'Content-Type': 'application/json'
      });
      res.end(JSON.stringify({ error: 'Proxy error', message: err.message }));
    }
  }
});

// Create a middleware to inject our script into HTML responses
app.use((req, res, next) => {
  // Store the original write function
  const originalWrite = res.write;
  const originalEnd = res.end;
  
  // Only inject into HTML responses
  res.on('header', () => {
    const contentType = res.getHeader('content-type');
    if (contentType && contentType.includes('text/html')) {
      let buffer = [];
      
      // Override the write function to capture chunks
      res.write = function(chunk, encoding) {
        buffer.push(Buffer.from(chunk, encoding));
        return true;
      };
      
      // Override the end function to inject our script
      res.end = function(chunk, encoding) {
        if (chunk) {
          buffer.push(Buffer.from(chunk, encoding));
        }
        
        // Combine all chunks
        let body = Buffer.concat(buffer).toString('utf8');
        
        // Inject our scripts before the closing body tag
        const scriptTags = `
          <script>
            // Fix for Next.js hydration issues
            window.__NEXT_DATA__ = window.__NEXT_DATA__ || { props: { pageProps: {} } };
            
            // Fix for Next.js hydration issues and WebSocket connections
            try {
              // Ensure Next.js data is properly initialized
              window.__NEXT_DATA__ = window.__NEXT_DATA__ || { 
                props: { pageProps: {} },
                page: window.location.pathname,
                query: {},
                buildId: 'development'
              };
              
              // Fix WebSocket issues by completely bypassing them for Next.js
              const originalWebSocket = window.WebSocket;
              window.WebSocket = function(url, protocols) {
                // For Next.js event WebSockets, create a mock that never fails
                if (url.includes('/events')) {
                  console.log('Creating mock WebSocket for Next.js events:', url);
                  
                  // Create a simple mock object with the necessary properties and methods
                  const mockWs = {};
                  
                  // Add WebSocket properties
                  mockWs.url = url;
                  mockWs.readyState = 1; // OPEN
                  mockWs.protocol = '';
                  mockWs.extensions = '';
                  mockWs.bufferedAmount = 0;
                  mockWs.binaryType = 'blob';
                  mockWs.onopen = null;
                  mockWs.onclose = null;
                  mockWs.onmessage = null;
                  mockWs.onerror = null;
                  
                  // Add methods
                  mockWs.close = function() {
                    if (this.onclose) {
                      this.onclose({ code: 1000, reason: 'Normal closure', wasClean: true });
                    }
                  };
                  
                  mockWs.send = function(data) {
                    console.log('Mock WebSocket send:', data);
                    return true;
                  };
                  
                  // Simulate successful connection
                  setTimeout(function() {
                    if (mockWs.onopen) {
                      mockWs.onopen({ target: mockWs });
                    }
                  }, 0);
                  
                  return mockWs;
                }
              
              // For all other WebSockets, use the original implementation
              // but ensure the URL is absolute
              if (url.startsWith('/')) {
                const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
                url = protocol + '//' + window.location.host + url;
                console.log('Rewrote WebSocket URL to:', url);
              }
              return new originalWebSocket(url, protocols);
            };
            } catch (e) {
              console.error('Error setting up WebSocket fix:', e);
            }
          </script>
          <script src="/kwik-interceptor.js"></script>
        `;
        body = body.replace('</body>', `${scriptTags}</body>`);
        
        // Write the modified body
        originalWrite.call(res, body);
        originalEnd.call(res);
      };
    }
  });
  
  next();
});

// Serve our interceptor script
app.get('/kwik-interceptor.js', (req, res) => {
  res.setHeader('Content-Type', 'application/javascript');
  fs.readFile(path.join('/aeternae/functional/dockers/animechanica', 'kwik-interceptor.js'), (err, data) => {
    if (err) {
      console.error('Error serving kwik-interceptor.js:', err);
      res.status(500).send('Error loading script');
      return;
    }
    res.send(data);
  });
});

// Apply HTTP proxy to all non-WebSocket routes
app.use('/', httpProxy);

const PORT = 3000;
app.listen(PORT, () => {
  console.log(`Proxy server running on http://localhost:${PORT}`);
  console.log(`Forwarding requests to backend at ${BACKEND_URL}`);
});
EOFPROXYJS

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
