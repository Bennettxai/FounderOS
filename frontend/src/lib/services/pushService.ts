/**
 * Web Push Notification Service
 *
 * Handles:
 * - Service worker registration
 * - Push subscription management
 * - Permission requests
 */

import { browser } from "$app/environment";
import { goto } from "$app/navigation";
import { writable, get } from "svelte/store";
import { getApiBaseUrl } from "$lib/api/base";

// State
export const pushPermission = writable<NotificationPermission>("default");
export const pushSupported = writable<boolean>(false);
export const pushSubscribed = writable<boolean>(false);
export const pushLoading = writable<boolean>(false);

// Get API base URL - uses centralized config from api/base.ts
function getApiBase(): string {
  if (typeof window === "undefined") return "";
  return getApiBaseUrl();
}

// Check if Web Push is supported
export function checkPushSupport(): boolean {
  if (!browser) return false;

  const supported =
    "serviceWorker" in navigator &&
    "PushManager" in window &&
    "Notification" in window;

  pushSupported.set(supported);
  return supported;
}

// Initialize push notifications
export async function initializePush(): Promise<void> {
  if (!browser || !checkPushSupport()) return;

  // Check current permission
  pushPermission.set(Notification.permission);

  // Register service worker
  try {
    const registration = await navigator.serviceWorker.register("/sw.js", {
      scope: "/",
    });

    // Check existing subscription
    const subscription = await registration.pushManager.getSubscription();
    pushSubscribed.set(!!subscription);
  } catch (err) {
    console.error("[Push] Service worker registration failed:", err);
  }
}

// Request push notification permission
export async function requestPermission(): Promise<NotificationPermission> {
  if (!browser || !checkPushSupport()) return "denied";

  const permission = await Notification.requestPermission();
  pushPermission.set(permission);
  return permission;
}

// Subscribe to push notifications
export async function subscribeToPush(): Promise<boolean> {
  if (!browser || !checkPushSupport()) return false;

  pushLoading.set(true);

  try {
    // Request permission if needed
    if (Notification.permission === "default") {
      const permission = await requestPermission();
      if (permission !== "granted") {
        pushLoading.set(false);
        return false;
      }
    }

    if (Notification.permission !== "granted") {
      pushLoading.set(false);
      return false;
    }

    // Get VAPID public key from server
    const baseUrl = getApiBase();
    const vapidResponse = await fetch(
      `${baseUrl}/notifications/push/vapid-public-key`,
      {
        credentials: "include",
      },
    );

    if (!vapidResponse.ok) {
      console.error("[Push] Failed to get VAPID key");
      pushLoading.set(false);
      return false;
    }

    const { public_key, enabled } = await vapidResponse.json();

    if (!enabled || !public_key) {
      pushLoading.set(false);
      return false;
    }

    // Get service worker registration
    const registration = await navigator.serviceWorker.ready;

    // Subscribe to push
    const subscription = await registration.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: urlBase64ToUint8Array(public_key) as BufferSource,
    });

    // Send subscription to server
    const subscriptionJSON = subscription.toJSON();

    const subscribeResponse = await fetch(
      `${baseUrl}/notifications/push/subscribe`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          endpoint: subscriptionJSON.endpoint,
          p256dh: subscriptionJSON.keys?.p256dh,
          auth: subscriptionJSON.keys?.auth,
          user_agent: navigator.userAgent,
        }),
      },
    );

    if (!subscribeResponse.ok) {
      console.error("[Push] Failed to save subscription on server");
      pushLoading.set(false);
      return false;
    }

    pushSubscribed.set(true);
    pushLoading.set(false);
    return true;
  } catch (err) {
    console.error("[Push] Subscription failed:", err);
    pushLoading.set(false);
    return false;
  }
}

// Unsubscribe from push notifications
export async function unsubscribeFromPush(): Promise<boolean> {
  if (!browser) return false;

  pushLoading.set(true);

  try {
    const registration = await navigator.serviceWorker.ready;
    const subscription = await registration.pushManager.getSubscription();

    if (!subscription) {
      pushSubscribed.set(false);
      pushLoading.set(false);
      return true;
    }

    // Unsubscribe locally
    await subscription.unsubscribe();

    // Remove from server
    const baseUrl = getApiBase();
    await fetch(`${baseUrl}/notifications/push/unsubscribe`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ endpoint: subscription.endpoint }),
    });

    pushSubscribed.set(false);
    pushLoading.set(false);
    return true;
  } catch (err) {
    console.error("[Push] Unsubscribe failed:", err);
    pushLoading.set(false);
    return false;
  }
}

// Send a test push notification
export async function sendTestPush(): Promise<boolean> {
  if (!browser) return false;

  const baseUrl = getApiBase();

  try {
    const response = await fetch(`${baseUrl}/notifications/push/test`, {
      method: "POST",
      credentials: "include",
    });

    return response.ok;
  } catch (err) {
    console.error("[Push] Test push failed:", err);
    return false;
  }
}

// Helper: Convert base64 to Uint8Array for applicationServerKey
function urlBase64ToUint8Array(base64String: string): Uint8Array {
  const padding = "=".repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, "+").replace(/_/g, "/");

  const rawData = window.atob(base64);
  const outputArray = new Uint8Array(rawData.length);

  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i);
  }

  return outputArray;
}

// Listen for messages from service worker
if (browser && "serviceWorker" in navigator) {
  navigator.serviceWorker.addEventListener("message", (event) => {
    if (event.data?.type === "NOTIFICATION_CLICK") {
      // Handle notification click from service worker
      const data = event.data.data;
      if (data?.url) {
        goto(data.url);
      }

      // Dispatch event for app to handle
      window.dispatchEvent(
        new CustomEvent("businessos:push-click", { detail: data }),
      );
    }
  });
}

export const pushService = {
  checkSupport: checkPushSupport,
  initialize: initializePush,
  requestPermission,
  subscribe: subscribeToPush,
  unsubscribe: unsubscribeFromPush,
  sendTest: sendTestPush,

  // Stores
  permission: pushPermission,
  supported: pushSupported,
  subscribed: pushSubscribed,
  loading: pushLoading,
};

export default pushService;
