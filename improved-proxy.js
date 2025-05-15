// This is an improved version of the proxy code designed to fix CORS issues
// with kwikie.ru and handle connection timeouts properly

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
    timeout: isKwikieUrl ? 60000 : 30000, // Longer timeout for kwikie.ru requests (60 seconds)
    rejectUnauthorized: false, // Accept self-signed certificates for testing
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
      
      // Always set our CORS headers
      responseHeaders['access-control-allow-origin'] = req.headers.origin || '*';
      responseHeaders['access-control-allow-credentials'] = 'true';
      
      // Handle range headers for video streaming
      if (proxyRes.headers['content-range']) {
        responseHeaders['accept-ranges'] = 'bytes';
      }
      
      // Set status code and headers
      res.writeHead(proxyRes.statusCode, responseHeaders);
      
      // Handle timeout and connection errors for kwikie.ru
      if (isKwikieResponse) {
        let dataReceived = false;
        
        // Handle data events
        proxyRes.on('data', chunk => {
          dataReceived = true;
        });
        
        // Handle end event
        proxyRes.on('end', () => {
          if (!dataReceived && proxyRes.statusCode === 200) {
            console.error('Warning: No data received from kwikie.ru despite 200 status');
          }
        });
      }
      
      // Pipe the response data to the client
      proxyRes.pipe(res);
    });

    // Add timeout handling to detect stalled connections
    proxyReq.setTimeout(isKwikieUrl ? 60000 : 30000, () => {
      console.error(`Request timeout for ${targetUrl}`);
      proxyReq.destroy();
      if (!res.headersSent) {
        res.status(504).json({ error: 'Gateway Timeout', message: 'Request timed out' });
      }
    });

    proxyReq.on('error', err => {
      console.error('Proxy error:', err);
      
      // Special retry logic for kwikie.ru if it's a timeout
      if (isKwikieUrl && (err.code === 'ETIMEDOUT' || err.code === 'ECONNABORTED' || err.code === 'ECONNRESET')) {
        console.log('kwikie.ru connection error, may retry on client side');
      }
      
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
