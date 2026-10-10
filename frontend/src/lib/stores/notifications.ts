/**
 * Notification Store - Real-time notifications via SSE
 *
 * Handles:
 * - SSE connection to /api/notifications/stream
 * - Reactive notification state
 * - Read state management
 * - Reconnection with exponential backoff
 */

import { writable, derived, get } from "svelte/store";
import { browser } from "$app/environment";
import { soundStore } from "./soundStore";
import { getApiBaseUrl } from "$lib/api/base";

// Types
export interface Notification {
  id: string;
  user_id: string;
  workspace_id?: string;
  type: string;
  title: string;
  body?: string;
  entity_type?: string;
  entity_id?: string;
  sender_id?: string;
  sender_name?: string;
  sender_avatar_url?: string;
  is_read: boolean;
  read_at?: string;
  batch_id?: string;
  batch_count?: number;
  channels_sent?: string[];
  priority: "low" | "normal" | "high" | "urgent";
  metadata?: Record<string, unknown>;
  created_at: string;
}

export interface NotificationPreferences {
  user_id: string;
  email_enabled: boolean;
  push_enabled: boolean;
  in_app_enabled: boolean;
  type_settings?: Record<
    string,
    { in_app?: boolean; push?: boolean; email?: boolean }
  >;
  quiet_hours_enabled: boolean;
  quiet_hours_start?: string;
  quiet_hours_end?: string;
  quiet_hours_timezone?: string;
  email_digest_enabled: boolean;
  email_digest_time?: string;
  email_digest_timezone?: string;
}

// State
const notifications = writable<Notification[]>([]);
const unreadCount = writable<number>(0);
const isConnected = writable<boolean>(false);
const connectionError = writable<string | null>(null);

// SSE connection state
let eventSource: EventSource | null = null;
let reconnectAttempts = 0;
let reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
const MAX_RECONNECT_ATTEMPTS = 10;
const BASE_RECONNECT_DELAY = 1000;

// Get API base URL - uses centralized config from api/base.ts
function getApiBase(): string {
  if (typeof window === "undefined") return "";
  return getApiBaseUrl();
}

// Read the CSRF token from the csrf_token cookie (mirrors auth-client pattern)
function getCSRFToken(): string | null {
  if (typeof document === "undefined") return null;
  for (const cookie of document.cookie.split(";")) {
    const trimmed = cookie.trim();
    const equalsIndex = trimmed.indexOf("=");
    if (equalsIndex === -1) continue;
    if (trimmed.substring(0, equalsIndex) === "csrf_token") {
      return trimmed.substring(equalsIndex + 1);
    }
  }
  return null;
}

// Build headers with CSRF token for state-changing requests
function csrfHeaders(base: HeadersInit = {}): HeadersInit {
  const token = getCSRFToken();
  if (token) {
    return { ...base, "X-CSRF-Token": token };
  }
  return base;
}

// Connect to SSE stream
function connect() {
  if (!browser) return;

  const baseUrl = getApiBase();
  if (!baseUrl) return;

  // Close existing connection
  disconnect();

  try {
    eventSource = new EventSource(`${baseUrl}/notifications/stream`, {
      withCredentials: true,
    });

    eventSource.onopen = () => {
      if (import.meta.env.DEV) console.log("[Notifications] SSE connected");
      isConnected.set(true);
      connectionError.set(null);
      reconnectAttempts = 0;
    };

    eventSource.addEventListener("connected", (event) => {
      if (import.meta.env.DEV) console.log("[Notifications] Connection confirmed:", event.data);
    });

    eventSource.addEventListener("notification", (event) => {
      try {
        const notification = JSON.parse(event.data) as Notification;
        handleNewNotification(notification);
      } catch (err) {
        console.error("[Notifications] Failed to parse notification:", err);
      }
    });

    eventSource.addEventListener("read_sync", (event) => {
      try {
        const data = JSON.parse(event.data);
        handleReadSync(data);
      } catch (err) {
        console.error("[Notifications] Failed to parse read_sync:", err);
      }
    });

    eventSource.onerror = (error) => {
      console.error("[Notifications] SSE error:", error);
      isConnected.set(false);
      connectionError.set("Connection lost");
      scheduleReconnect();
    };
  } catch (err) {
    console.error("[Notifications] Failed to create EventSource:", err);
    connectionError.set("Failed to connect");
    scheduleReconnect();
  }
}

function disconnect() {
  if (eventSource) {
    eventSource.close();
    eventSource = null;
  }
  isConnected.set(false);
  if (reconnectTimeout) {
    clearTimeout(reconnectTimeout);
    reconnectTimeout = null;
  }
}

function scheduleReconnect() {
  if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
    console.error("[Notifications] Max reconnection attempts reached");
    connectionError.set("Unable to connect. Please refresh the page.");
    return;
  }

  const delay = Math.min(
    BASE_RECONNECT_DELAY * Math.pow(2, reconnectAttempts),
    30000,
  );
  reconnectAttempts++;

  if (import.meta.env.DEV) {
    console.log(
      `[Notifications] Reconnecting in ${delay}ms (attempt ${reconnectAttempts})`,
    );
  }

  reconnectTimeout = setTimeout(() => {
    connect();
  }, delay);
}

// Handle new notification
function handleNewNotification(notification: Notification) {
  notifications.update((current) => {
    // Avoid duplicates
    if (current.some((n) => n.id === notification.id)) {
      return current;
    }
    return [notification, ...current];
  });

  if (!notification.is_read) {
    unreadCount.update((n) => n + 1);
  }

  // Play sound for new notifications
  soundStore.playSound("notification");

  // Dispatch custom event for toast/UI
  if (browser) {
    window.dispatchEvent(
      new CustomEvent("businessos:notification", {
        detail: notification,
      }),
    );
  }
}

// Handle read sync from other tabs/devices
function handleReadSync(data: {
  read_ids?: string[];
  read_all?: boolean;
  read_at: string;
}) {
  if (data.read_all) {
    notifications.update((current) =>
      current.map((n) => ({ ...n, is_read: true, read_at: data.read_at })),
    );
    unreadCount.set(0);
  } else if (data.read_ids) {
    const readSet = new Set(data.read_ids);
    notifications.update((current) =>
      current.map((n) =>
        readSet.has(n.id) ? { ...n, is_read: true, read_at: data.read_at } : n,
      ),
    );
    // Recalculate unread count
    notifications.subscribe((current) => {
      unreadCount.set(current.filter((n) => !n.is_read).length);
    })();
  }
}

// API functions
async function fetchNotifications(limit = 20, offset = 0): Promise<void> {
  const baseUrl = getApiBase();
  try {
    const response = await fetch(
      `${baseUrl}/notifications?limit=${limit}&offset=${offset}`,
      { credentials: "include" },
    );
    if (response.ok) {
      const data = await response.json();
      const list = Array.isArray(data.data)
        ? data.data
        : Array.isArray(data.notifications)
          ? data.notifications
          : [];
      notifications.set(list);
      // Also update unread count from response if available
      if (typeof data.unread_count === "number") {
        unreadCount.set(data.unread_count);
      }
    }
  } catch (err) {
    console.error("[Notifications] Failed to fetch:", err);
  }
}

async function fetchUnreadCount(): Promise<void> {
  const baseUrl = getApiBase();
  try {
    const response = await fetch(`${baseUrl}/notifications/unread-count`, {
      credentials: "include",
    });
    if (response.ok) {
      const data = await response.json();
      unreadCount.set(data.count || 0);
    }
  } catch (err) {
    console.error("[Notifications] Failed to fetch unread count:", err);
  }
}

async function markAsRead(id: string): Promise<boolean> {
  const baseUrl = getApiBase();
  try {
    const response = await fetch(`${baseUrl}/notifications/${id}/read`, {
      method: "POST",
      headers: csrfHeaders(),
      credentials: "include",
    });
    if (response.ok) {
      notifications.update((current) =>
        current.map((n) =>
          n.id === id
            ? { ...n, is_read: true, read_at: new Date().toISOString() }
            : n,
        ),
      );
      unreadCount.update((n) => Math.max(0, n - 1));
      return true;
    }
  } catch (err) {
    console.error("[Notifications] Failed to mark as read:", err);
  }
  return false;
}

async function markMultipleAsRead(ids: string[]): Promise<boolean> {
  const baseUrl = getApiBase();
  try {
    const response = await fetch(`${baseUrl}/notifications/read`, {
      method: "POST",
      headers: csrfHeaders({ "Content-Type": "application/json" }),
      credentials: "include",
      body: JSON.stringify({ ids }),
    });
    if (response.ok) {
      const readSet = new Set(ids);
      notifications.update((current) =>
        current.map((n) =>
          readSet.has(n.id)
            ? { ...n, is_read: true, read_at: new Date().toISOString() }
            : n,
        ),
      );
      unreadCount.update((n) => Math.max(0, n - ids.length));
      return true;
    }
  } catch (err) {
    console.error("[Notifications] Failed to mark multiple as read:", err);
  }
  return false;
}

async function markAllAsRead(): Promise<boolean> {
  const baseUrl = getApiBase();
  try {
    const response = await fetch(`${baseUrl}/notifications/read-all`, {
      method: "POST",
      headers: csrfHeaders(),
      credentials: "include",
    });
    if (response.ok) {
      notifications.update((current) =>
        current.map((n) => ({
          ...n,
          is_read: true,
          read_at: new Date().toISOString(),
        })),
      );
      unreadCount.set(0);
      return true;
    }
  } catch (err) {
    console.error("[Notifications] Failed to mark all as read:", err);
  }
  return false;
}

async function deleteNotification(id: string): Promise<boolean> {
  const baseUrl = getApiBase();
  try {
    const response = await fetch(`${baseUrl}/notifications/${id}`, {
      method: "DELETE",
      headers: csrfHeaders(),
      credentials: "include",
    });
    if (response.ok) {
      const current = get(notifications);
      const notif = current.find((n) => n.id === id);
      notifications.update((list) => list.filter((n) => n.id !== id));
      if (notif && !notif.is_read) {
        unreadCount.update((n) => Math.max(0, n - 1));
      }
      return true;
    }
  } catch (err) {
    console.error("[Notifications] Failed to delete:", err);
  }
  return false;
}

async function getPreferences(): Promise<NotificationPreferences | null> {
  const baseUrl = getApiBase();
  try {
    const response = await fetch(`${baseUrl}/notifications/preferences`, {
      credentials: "include",
    });
    if (response.ok) {
      return await response.json();
    }
  } catch (err) {
    console.error("[Notifications] Failed to fetch preferences:", err);
  }
  return null;
}

async function updatePreferences(
  prefs: Partial<NotificationPreferences>,
): Promise<NotificationPreferences | null> {
  const baseUrl = getApiBase();
  try {
    const response = await fetch(`${baseUrl}/notifications/preferences`, {
      method: "PUT",
      headers: csrfHeaders({ "Content-Type": "application/json" }),
      credentials: "include",
      body: JSON.stringify(prefs),
    });
    if (response.ok) {
      return await response.json();
    }
  } catch (err) {
    console.error("[Notifications] Failed to update preferences:", err);
  }
  return null;
}

// Derived stores
const hasUnread = derived(unreadCount, ($count) => $count > 0);
const recentNotifications = derived(notifications, ($notifications) =>
  $notifications.slice(0, 10),
);

// Initialize on import (browser only)
function initialize() {
  if (!browser) return;

  // Fetch initial data
  fetchNotifications();
  fetchUnreadCount();

  // Connect to SSE
  connect();

  // Reconnect when visibility changes (tab comes back into focus)
  document.addEventListener("visibilitychange", () => {
    if (document.visibilityState === "visible" && !get(isConnected)) {
      reconnectAttempts = 0;
      connect();
    }
  });
}

// Export store and functions
export const notificationStore = {
  // Stores
  notifications,
  unreadCount,
  isConnected,
  connectionError,
  hasUnread,
  recentNotifications,

  // Connection
  connect,
  disconnect,
  initialize,

  // API
  fetchNotifications,
  fetchUnreadCount,
  markAsRead,
  markMultipleAsRead,
  markAllAsRead,
  deleteNotification,
  getPreferences,
  updatePreferences,
};

export default notificationStore;
