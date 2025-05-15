// Simple proxy server for Seanime using only built-in modules
const http = require('http');
const url = require('url');
const { spawn } = require('child_process');

// Create HTTP server
const server = http.createServer((req, res) => {
  // Only handle GET requests to /api/v1/proxy
  const parsedUrl = url.parse(req.url, true);
  
  if (req.method === 'GET' && parsedUrl.pathname === '/api/v1/proxy') {
    // Extract the target URL from the query parameters
    const targetUrl = parsedUrl.query.url;
    if (!targetUrl) {
      res.writeHead(400, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: 'Missing URL parameter' }));
      return;
    }
    
    console.log(`Proxying video request to: ${targetUrl}`);
    
    // Set CORS headers
    res.setHeader('Access-Control-Allow-Origin', '*');
    res.setHeader('Access-Control-Allow-Methods', 'GET, OPTIONS');
    res.setHeader('Access-Control-Allow-Headers', 'Content-Type');
    res.setHeader('Access-Control-Allow-Credentials', 'true');
    
    // Parse custom headers if provided
    let headers = [];
    try {
      if (parsedUrl.query.headers) {
        const customHeaders = JSON.parse(parsedUrl.query.headers);
        headers = Object.entries(customHeaders).map(([key, value]) => ['-H', `${key}: ${value}`]).flat();
      }
    } catch (e) {
      console.error('Error parsing custom headers:', e);
    }
    
    // Set appropriate content type based on file extension
    if (targetUrl.endsWith('.ts')) {
      res.setHeader('Content-Type', 'video/mp2t');
    } else if (targetUrl.endsWith('.m3u8')) {
      res.setHeader('Content-Type', 'application/vnd.apple.mpegurl');
    } else if (targetUrl.endsWith('.mp4')) {
      res.setHeader('Content-Type', 'video/mp4');
    } else {
      res.setHeader('Content-Type', 'application/octet-stream');
    }
    
    // Create a curl process that outputs to stdout
    const curl = spawn('curl', [
      '-s', '-k', '-L',  // silent, insecure, follow redirects
      ...headers,
      targetUrl
    ], { stdio: ['ignore', 'pipe', 'pipe'] });
    
    // Handle errors
    curl.on('error', (error) => {
      console.error(`Curl process error: ${error.message}`);
      if (!res.headersSent) {
        res.writeHead(500, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ error: 'Error streaming content', message: error.message }));
      }
    });
    
    // Collect stderr output
    let stderrOutput = '';
    curl.stderr.on('data', (data) => {
      stderrOutput += data.toString();
    });
    
    // Handle process exit
    curl.on('exit', (code) => {
      if (code !== 0 && !res.headersSent) {
        console.error(`Curl failed with code ${code}: ${stderrOutput}`);
        res.writeHead(500, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ error: 'Failed to stream content', code, stderr: stderrOutput }));
      }
    });
    
    // Pipe the curl output directly to the response
    curl.stdout.pipe(res);
    
    // Handle client disconnect
    req.on('close', () => {
      curl.kill();
    });
  } else {
    // For any other request, return 404
    res.writeHead(404, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ error: 'Not found' }));
  }
});

// Start the server
const PORT = 3000;
server.listen(PORT, () => {
  console.log(`Proxy server listening at http://localhost:${PORT}`);
});
