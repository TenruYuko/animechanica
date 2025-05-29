// This script will be injected into the qBittorrent WebUI to allow iframe embedding
// and to ensure proper communication with Seanime
window.addEventListener('DOMContentLoaded', function() {
  // Remove X-Frame-Options header if present
  if (window.parent !== window) {
    // We're in an iframe
    console.log('qBittorrent is running in an iframe, enabling cross-frame communication');
    
    // Set up message passing to parent frame if needed
    window.addEventListener('message', function(event) {
      // Handle messages from parent frame
      if (event.data && event.data.type === 'seanime-qbittorrent-ping') {
        window.parent.postMessage({ type: 'seanime-qbittorrent-pong' }, '*');
      }
    });
  }
});
