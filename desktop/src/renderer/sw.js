// Service Worker for Web Push Notifications
// This runs in the background and receives push events even when the app is closed

// Bump this on every release so the service worker drops the old cached app and
// users get the new bundle instead of a stale one (the "looks not deployed" bug).
const CACHE_NAME = "businessos-v1.0.1-2";

// Install event - cache essential assets
self.addEventListener("install", (event) => {
  console.log("[ServiceWorker] Install");
  self.skipWaiting();
});

// Activate event - clean up old caches
self.addEventListener("activate", (event) => {
  console.log("[ServiceWorker] Activate");
  event.waitUntil(self.clients.claim());
});

// Push event - receive push notification from server
self.addEventListener("push", (event) => {
  console.log("[ServiceWorker] Push received");

  let data = {
    title: "BusinessOS",
    body: "You have a new notification",
    icon: "/icon-192.png",
    badge: "/badge-72.png",
    tag: "default",
    data: {},
  };

  if (event.data) {
    try {
      const payload = event.data.json();
      data = { ...data, ...payload };
    } catch (e) {
      console.error("[ServiceWorker] Failed to parse push data:", e);
      data.body = event.data.text();
    }
  }

  const options = {
    body: data.body,
    icon: data.icon || "/icon-192.png",
    badge: data.badge || "/badge-72.png",
    tag: data.tag || "default",
    data: data.data || {},
    vibrate: data.vibrate || [100, 50, 100],
    requireInteraction: data.priority === "high" || data.priority === "urgent",
    actions: data.actions || [],
  };

  event.waitUntil(self.registration.showNotification(data.title, options));
});

// Notification click event - handle user interaction
self.addEventListener("notificationclick", (event) => {
  console.log("[ServiceWorker] Notification click:", event.action);

  event.notification.close();

  // Get URL to open
  let url = "/";
  if (event.notification.data && event.notification.data.url) {
    url = event.notification.data.url;
  }

  // Handle action buttons
  if (event.action) {
    switch (event.action) {
      case "view":
        // Default behavior - open the URL
        break;
      case "dismiss":
        // Just close, don't open anything
        return;
      case "mark-read":
        // Mark as read via API (fire and forget)
        if (
          event.notification.data &&
          event.notification.data.notification_id
        ) {
          fetch(
            `/api/notifications/${event.notification.data.notification_id}/read`,
            {
              method: "POST",
              credentials: "include",
            },
          ).catch(() => {});
        }
        return;
      default:
        break;
    }
  }

  // Open or focus the app
  event.waitUntil(
    self.clients
      .matchAll({ type: "window", includeUncontrolled: true })
      .then((clients) => {
        // Check if app is already open
        for (const client of clients) {
          if (client.url.includes(self.location.origin) && "focus" in client) {
            client.postMessage({
              type: "NOTIFICATION_CLICK",
              data: event.notification.data,
            });
            return client.focus();
          }
        }
        // Open new window
        if (self.clients.openWindow) {
          return self.clients.openWindow(url);
        }
      }),
  );
});

// Notification close event
self.addEventListener("notificationclose", (event) => {
  console.log("[ServiceWorker] Notification closed");
});

// Message from main app
self.addEventListener("message", (event) => {
  console.log("[ServiceWorker] Message received:", event.data);

  if (event.data && event.data.type === "SKIP_WAITING") {
    self.skipWaiting();
  }
});
