// Service Worker for Yandex Music PWA
const CACHE_NAME = 'yandex-music-pwa-v6';

// Get base path from the service worker's location
// The service worker is registered from the page which has the base path
const getBasePath = () => {
  const scriptURL = self.location.pathname;
  const basePath = scriptURL.substring(0, scriptURL.lastIndexOf('/'));
  return basePath || '';
};

const BASE_PATH = getBasePath();

const urlsToCache = [
  BASE_PATH + '/',
  BASE_PATH + '/index.html',
  BASE_PATH + '/css/styles.css',
  BASE_PATH + '/js/app.js',
  BASE_PATH + '/manifest.json'
].map(url => url.replace('//', '/')); // Clean up double slashes

// Install event - cache static assets
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then((cache) => {
        console.log('Opened cache');
        return cache.addAll(urlsToCache);
      })
  );
  self.skipWaiting();
});

// Activate event - clean up old caches
self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames.map((cacheName) => {
          // Delete all old caches (including v5)
          if (cacheName !== CACHE_NAME) {
            console.log('Deleting old cache:', cacheName);
            return caches.delete(cacheName);
          }
        })
      );
    })
  );
  self.clients.claim();
});

// Fetch event - serve from cache, fallback to network
self.addEventListener('fetch', (event) => {
  const url = event.request.url;
  const isExternal = !url.startsWith(self.location.origin);
  const isImage = event.request.destination === 'image';
  
  // Log all requests for debugging
  console.log('[SW Fetch]', {
    url: url,
    destination: event.request.destination,
    mode: event.request.mode,
    isExternal: isExternal,
    isImage: isImage
  });
  
  // For external images, don't intercept at all - let browser handle naturally
  if (isImage && isExternal) {
    console.log('[SW] Bypassing external image:', url);
    // Don't call event.respondWith() - let the browser handle it completely
    return;
  }
  
  // Skip API calls and audio streams - always fetch from network without caching
  if (url.includes('/api/') || 
      url.includes('.mp3') ||
      url.includes('download-url')) {
    console.log('[SW] Fetching from network (no cache):', url);
    event.respondWith(fetch(event.request));
    return;
  }

  event.respondWith(
    caches.match(event.request)
      .then((response) => {
        // Cache hit - return response
        if (response) {
          console.log('[SW] Cache hit:', url);
          return response;
        }

        console.log('[SW] Cache miss, fetching:', url);
        // Clone the request
        const fetchRequest = event.request.clone();

        return fetch(fetchRequest).then((response) => {
          console.log('[SW] Fetch response:', {
            url: url,
            status: response.status,
            type: response.type
          });
          
          // Check if valid response - only cache successful responses from same origin
          if (!response || response.status !== 200 || (response.type !== 'basic' && response.type !== 'cors')) {
            console.log('[SW] Not caching (invalid or non-2xx):', url);
            return response;
          }

          // Don't cache opaque responses
          if (response.type === 'opaque') {
            console.log('[SW] Not caching opaque response:', url);
            return response;
          }

          // Clone the response
          const responseToCache = response.clone();

          caches.open(CACHE_NAME)
            .then((cache) => {
              console.log('[SW] Caching:', url);
              cache.put(event.request, responseToCache);
            });

          return response;
        });
      })
  );
});
