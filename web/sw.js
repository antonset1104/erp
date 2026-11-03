// GoERP Offline-First Service Worker (PWA Shell & POS Offline Sync Cache)
const CACHE_NAME = 'goerp-v1.43.0-cache';
const SHELL_ASSETS = [
  '/',
  '/web/index.html',
  '/web/app.js',
  '/web/manifest.json'
];

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => {
      console.log('[GoERP ServiceWorker] Caching App Shell for offline availability');
      return cache.addAll(SHELL_ASSETS).catch((err) => {
        console.warn('[GoERP ServiceWorker] Pre-cache warning:', err);
      });
    })
  );
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((keys) => {
      return Promise.all(
        keys.map((key) => {
          if (key !== CACHE_NAME) {
            console.log('[GoERP ServiceWorker] Removing outdated cache:', key);
            return caches.delete(key);
          }
        })
      );
    })
  );
  self.clients.claim();
});

self.addEventListener('fetch', (event) => {
  const url = new URL(event.request.url);

  // Network-first for dynamic API routes, with offline mock fallback if disconnected
  if (url.pathname.startsWith('/api/')) {
    event.respondWith(
      fetch(event.request).catch(() => {
        return new Response(
          JSON.stringify({
            offline: true,
            status: 'offline_buffered',
            message: 'Perangkat sedang luring (offline). Transaksi akan disimpan ke antrean lokal dan disinkronkan saat terhubung kembali.'
          }),
          { headers: { 'Content-Type': 'application/json' } }
        );
      })
    );
    return;
  }

  // Cache-first with network fallback for static shell assets
  event.respondWith(
    caches.match(event.request).then((cachedResponse) => {
      if (cachedResponse) {
        return cachedResponse;
      }
      return fetch(event.request).then((networkResponse) => {
        if (networkResponse && networkResponse.status === 200 && event.request.method === 'GET') {
          const responseToCache = networkResponse.clone();
          caches.open(CACHE_NAME).then((cache) => {
            cache.put(event.request, responseToCache);
          });
        }
        return networkResponse;
      }).catch(() => {
        // Return cached index.html for navigation fallback
        if (event.request.mode === 'navigate') {
          return caches.match('/web/index.html');
        }
      });
    })
  );
});

// Background sync handler
self.addEventListener('sync', (event) => {
  if (event.tag === 'sync-offline-pos') {
    event.waitUntil(
      self.clients.matchAll().then((clients) => {
        clients.forEach((client) => {
          client.postMessage({ type: 'TRIGGER_OFFLINE_POS_SYNC' });
        });
      })
    );
  }
});
