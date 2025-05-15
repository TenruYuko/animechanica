// Video proxy handler for Seanime
const fs = require('fs');
const path = require('path');
const os = require('os');
const crypto = require('crypto');
const { execSync, spawn } = require('child_process');

// Create a proxy handler function
function createProxyHandler() {
  return function(req, res) {
    console.log('Video proxy request:', req.url);
    
    // Extract the target URL from the query parameters
    const targetUrl = req.query.url;
    if (!targetUrl) {
      return res.status(400).json({ error: 'Missing URL parameter' });
    }
    
    console.log('Proxying video request to:', targetUrl);
    
    // Parse custom headers if provided
    let customHeaders = {};
    try {
      if (req.query.headers) {
        customHeaders = JSON.parse(req.query.headers);
      }
    } catch (e) {
      console.error('Error parsing custom headers:', e);
    }
    
    // Set CORS headers
    res.setHeader('Access-Control-Allow-Origin', '*');
    res.setHeader('Access-Control-Allow-Methods', 'GET, OPTIONS');
    res.setHeader('Access-Control-Allow-Headers', 'Content-Type');
    res.setHeader('Access-Control-Allow-Credentials', 'true');
    
    // Create a temporary directory for downloads if it doesn't exist
    const tempDir = path.join(os.tmpdir(), 'seanime-proxy');
    if (!fs.existsSync(tempDir)) {
      fs.mkdirSync(tempDir, { recursive: true });
    }
    
    // Generate a unique filename based on the URL
    const urlHash = crypto.createHash('md5').update(targetUrl).digest('hex');
    const tempFilePath = path.join(tempDir, `${urlHash}`);
    
    // Prepare headers for curl
    const headers = { ...customHeaders };
    
    if (targetUrl.endsWith('.m3u8')) {
      // For m3u8 files, download with curl and process
      console.log('Downloading m3u8 playlist...');
      
      try {
        // Build curl command with headers
        const headerArgs = Object.entries(headers)
          .map(([key, value]) => `-H "${key}: ${value}"`)
          .join(' ');
        
        // Download the m3u8 file
        execSync(`curl -s -k ${headerArgs} "${targetUrl}" -o "${tempFilePath}.m3u8"`);
        
        // Read the file content
        const fileContent = fs.readFileSync(`${tempFilePath}.m3u8`, 'utf8');
        
        // Process the file to convert relative URLs to absolute
        const lines = fileContent.split('\n');
        const processedLines = lines.map(line => {
          // Skip comments and empty lines
          if (line.startsWith('#') || !line.trim()) {
            return line;
          }
          
          // If the line is not already an absolute URL, make it absolute
          if (!line.startsWith('http')) {
            const base = targetUrl.substring(0, targetUrl.lastIndexOf('/') + 1);
            return base + line;
          }
          
          return line;
        });
        
        // Write the processed content back to the file
        fs.writeFileSync(`${tempFilePath}.processed.m3u8`, processedLines.join('\n'));
        
        // Set appropriate content type
        res.setHeader('Content-Type', 'application/vnd.apple.mpegurl');
        
        // Stream the file to the client
        console.log('Sending modified HLS playlist');
        const fileStream = fs.createReadStream(`${tempFilePath}.processed.m3u8`);
        fileStream.pipe(res);
        
        // Clean up the file when done
        fileStream.on('end', () => {
          try {
            fs.unlinkSync(`${tempFilePath}.m3u8`);
            fs.unlinkSync(`${tempFilePath}.processed.m3u8`);
          } catch (e) {
            console.error('Error cleaning up temporary files:', e);
          }
        });
      } catch (error) {
        console.error('Error downloading or processing m3u8:', error);
        if (!res.headersSent) {
          return res.status(500).json({ error: 'Error processing m3u8', message: error.message });
        }
      }
    } else {
      // For TS segments and other files, use a direct approach with curl
      console.log(`Streaming video segment: ${targetUrl}`);
      
      // Set appropriate content type based on file extension
      if (targetUrl.endsWith('.ts')) {
        res.setHeader('Content-Type', 'video/mp2t');
      } else if (targetUrl.endsWith('.mp4')) {
        res.setHeader('Content-Type', 'video/mp4');
      } else {
        res.setHeader('Content-Type', 'application/octet-stream');
      }
      
      // Create a curl process that outputs to stdout
      const curl = spawn('curl', [
        '-s', '-k', '-L', // silent, insecure, follow redirects
        ...Object.entries(headers).flatMap(([key, value]) => ['-H', `${key}: ${value}`]),
        targetUrl
      ], { stdio: ['ignore', 'pipe', 'pipe'] });
      
      // Handle errors
      curl.on('error', (error) => {
        console.error(`Curl process error: ${error.message}`);
        if (!res.headersSent) {
          res.status(500).json({ error: 'Error streaming content', message: error.message });
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
          res.status(500).json({ error: 'Failed to stream content', code, stderr: stderrOutput });
        }
      });
      
      // Pipe the curl output directly to the response
      curl.stdout.pipe(res);
      
      // Handle client disconnect
      req.on('close', () => {
        curl.kill();
      });
    }
  };
}

module.exports = createProxyHandler;
