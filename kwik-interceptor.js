// This script intercepts fetch and XMLHttpRequest calls to kwik.si and kwikie.ru
// and redirects them through our proxy with proper headers

(function() {
  console.log('Kwik.si and Kwikie.ru interceptor loaded');
  
  // Store original fetch
  const originalFetch = window.fetch;
  
  // Override fetch
  window.fetch = function(resource, options) {
    let url;
    
    // Handle both string URLs and Request objects
    if (typeof resource === 'string') {
      url = resource;
    } else if (resource instanceof Request) {
      url = resource.url;
    } else {
      return originalFetch(resource, options);
    }
    
    // Check if this is a kwik.si or kwikie.ru URL
    if (url.includes('kwik.si') || url.includes('kwikie.ru')) {
      console.log('Intercepted kwik/kwikie request:', url);
      
      // Create proxy URL
      const proxyUrl = `/api/v1/proxy?url=${encodeURIComponent(url)}`;
      
      // Create new options with proper headers if needed
      const newOptions = options || {};
      if (!newOptions.headers) {
        newOptions.headers = {};
      }
      
      // Add referer header for kwik.si
      newOptions.headers['Referer'] = 'https://kwik.si';
      
      console.log('Redirecting through proxy:', proxyUrl);
      
      // Call original fetch with proxied URL
      if (typeof resource === 'string') {
        return originalFetch(proxyUrl, newOptions);
      } else {
        const newRequest = new Request(proxyUrl, {
          method: resource.method,
          headers: newOptions.headers,
          body: resource.body,
          mode: 'cors',
          credentials: resource.credentials,
          cache: resource.cache,
          redirect: resource.redirect,
          referrer: 'https://kwik.si',
          integrity: resource.integrity
        });
        return originalFetch(newRequest);
      }
    }
    
    // Not a kwik URL, proceed normally
    return originalFetch(resource, options);
  };
  
  // XMLHttpRequest interceptor
  const originalXHROpen = XMLHttpRequest.prototype.open;
  
  XMLHttpRequest.prototype.open = function(method, url, async, user, password) {
    // Check if this is a kwik.si or kwikie.ru URL
    if (typeof url === 'string' && (url.includes('kwik.si') || url.includes('kwikie.ru'))) {
      console.log('Intercepted XHR kwik/kwikie request:', url);
      
      // Create proxy URL
      const proxyUrl = `/api/v1/proxy?url=${encodeURIComponent(url)}`;
      console.log('Redirecting XHR through proxy:', proxyUrl);
      
      // Call original with proxied URL
      return originalXHROpen.call(this, method, proxyUrl, async, user, password);
    }
    
    // Not a kwik URL, proceed normally
    return originalXHROpen.call(this, method, url, async, user, password);
  };
  
  // Also intercept media elements
  const interceptMedia = () => {
    // This function finds all video/audio elements and rewrites their sources
    const mediaElements = document.querySelectorAll('video, audio');
    
    mediaElements.forEach(element => {
      // Check the current src
      if (element.src && (element.src.includes('kwik.si') || element.src.includes('kwikie.ru'))) {
        console.log('Intercepted media element src:', element.src);
        const proxyUrl = `/api/v1/proxy?url=${encodeURIComponent(element.src)}`;
        console.log('Rewriting media src to:', proxyUrl);
        element.src = proxyUrl;
      }
      
      // Also check source elements
      const sources = element.querySelectorAll('source');
      sources.forEach(source => {
        if (source.src && (source.src.includes('kwik.si') || source.src.includes('kwikie.ru'))) {
          console.log('Intercepted source element src:', source.src);
          const proxyUrl = `/api/v1/proxy?url=${encodeURIComponent(source.src)}`;
          console.log('Rewriting source src to:', proxyUrl);
          source.src = proxyUrl;
        }
      });
    });
  };
  
  // Run media interception periodically and on DOM changes
  setInterval(interceptMedia, 1000);
  
  // Set up a MutationObserver to detect DOM changes
  const observer = new MutationObserver(mutations => {
    for (const mutation of mutations) {
      if (mutation.type === 'childList' && mutation.addedNodes.length) {
        interceptMedia();
      }
    }
  });
  
  // Start observing
  observer.observe(document.body, { 
    childList: true,
    subtree: true
  });
  
  console.log('Kwik.si and Kwikie.ru interceptor setup complete');
})();
