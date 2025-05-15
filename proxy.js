const express = require('express');
const { createProxyMiddleware } = require('http-proxy-middleware');
const cookieParser = require('cookie-parser');
const bodyParser = require('body-parser');
const path = require('path');
const fs = require('fs');
const app = express();

// Set up the backend URL
const BACKEND_URL = 'http://localhost:3000';

// Add middleware for parsing cookies and request bodies
app.use(cookieParser());
app.use(bodyParser.json({ limit: '50mb' }));
app.use(bodyParser.urlencoded({ extended: true, limit: '50mb' }));

// Configure proxy options with better error handling
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
      // Set CORS headers
      res.setHeader('Access-Control-Allow-Origin', req.headers.origin || '*');
      res.setHeader('Access-Control-Allow-Credentials', 'true');
      res.setHeader('Access-Control-Allow-Methods', 'GET, OPTIONS');
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
        return cookie
          .replace(/Domain=[^;]+;/i, '')
          .replace(/SameSite=[^;]+;/i, 'SameSite=Lax;');
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
        .map(([key, value]) => `${key}=${value}`)
        .join('; ');
      if (cookieString) {
        proxyReq.setHeader('Cookie', cookieString);
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
    res.writeHead(500, {
      'Content-Type': 'application/json'
    });
    res.end(JSON.stringify({ error: 'Proxy error', message: err.message }));
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
          .map(([key, value]) => `${key}=${value}`)
          .join('; ');
        if (cookieString) {
          proxyReq.setHeader('Cookie', cookieString);
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
          .map(([key, value]) => `${key}=${value}`)
          .join('; ');
        if (cookieString) {
          proxyReq.setHeader('Cookie', cookieString);
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
        .map(([key, value]) => `${key}=${value}`)
        .join('; ');
      if (cookieString) {
        proxyReq.setHeader('Cookie', cookieString);
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

// Proxy API requests to the backend
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

// Add a special route for handling session issues
app.use('/api/v1/auth', (req, res, next) => {
  // Add special handling for authentication routes
  const proxyWithAuth = createProxyMiddleware({
    ...proxyOptions,
    onProxyRes: function(proxyRes, req, res) {
      // Preserve cookies and fix session issues
      const proxyCookies = proxyRes.headers['set-cookie'];
      if (proxyCookies) {
        const cookies = Array.isArray(proxyCookies) ? proxyCookies : [proxyCookies];
        proxyRes.headers['set-cookie'] = cookies.map(cookie => {
          return cookie
            .replace(/Domain=[^;]+;/i, '')
            .replace(/SameSite=[^;]+;/i, 'SameSite=Lax;')
            .replace(/Secure;/i, '');
        });
      }
      
      // Handle CORS headers
      proxyRes.headers['Access-Control-Allow-Origin'] = req.headers.origin || '*';
      proxyRes.headers['Access-Control-Allow-Credentials'] = 'true';
    }
  });
  
  proxyWithAuth(req, res, next);
});

// Special handling for video streaming with a simple implementation
app.use('/api/v1/proxy', async (req, res) => {
  // Set CORS headers for every response
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
    // Copy all headers from the proxied response
    Object.entries(proxyRes.headers).forEach(([key, value]) => {
      res.setHeader(key, value);
    });
    // Set CORS headers again in case they were overwritten
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
    if (!res.headersSent) {
      res.writeHead(500, {
        'Content-Type': 'application/json'
      });
      res.end(JSON.stringify({ error: 'WebSocket proxy error', message: err.message }));
    }
  },
  // Minimal WebSocket handling to prevent loops
  onProxyReqWs: (proxyReq, req, socket, options, head) => {
    // Add a one-time connection established log
    socket.once('open', () => {
      console.error('WebSocket connection established'); // Use error to bypass our log filter
    });
    
    // Handle socket errors
    socket.on('error', (err) => {
      console.error('WebSocket socket error:', err);
    });
  }
});

// Apply WebSocket proxy ONLY to the specific WebSocket path
app.use('/api/v1/ws', wsProxy);

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

// Apply HTTP proxy to all non-WebSocket routes
app.use('/', httpProxy);

const PORT = 3000;
app.listen(PORT, () => {
  console.log(`Proxy server running on http://localhost:${PORT}`);
  console.log(`Forwarding requests to backend at ${BACKEND_URL}`);
});