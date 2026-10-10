/**
 * Platform detection utilities for BusinessOS
 * Detects whether running in Electron vs Web browser
 */

import { browser } from "$app/environment";

// Type definition for the Electron API exposed via preload
export interface ElectronAPI {
  getVersion: () => Promise<string>;
  getPlatform: () => Promise<{
    platform: string;
    arch: string;
    isPackaged: boolean;
  }>;
  backend: {
    getStatus: () => Promise<{
      running: boolean;
      port: number;
      url: string;
    }>;
    getUrl: () => Promise<string>;
    restart: () => Promise<boolean>;
  };
  network: {
    getStatus: () => Promise<{ online: boolean }>;
  };
  sync: {
    getStatus: () => Promise<{
      status: string;
      lastSync: string;
      pendingChanges: number;
    }>;
    trigger: () => Promise<boolean>;
  };
  updates: {
    getInfo?: () => Promise<{
      currentVersion: string;
      channel: string;
      isPackaged: boolean;
      owner: string;
      repo: string;
      minimumSupportedVersion: string | null;
      supported: boolean;
    }>;
    check: () => Promise<{
      ok: boolean;
      available: boolean;
      version: string | null;
      currentVersion: string;
      releaseDate?: string;
      releaseNotes?: unknown;
      error?: string;
      message?: string;
    }>;
    download: () => Promise<boolean>;
    install: () => Promise<boolean>;
  };
  shell: {
    openExternal: (url: string) => Promise<void>;
    openPath: (path: string) => Promise<void>;
  };
  dialog: {
    showOpen: (
      options: Record<string, unknown>,
    ) => Promise<{ canceled: boolean; filePaths: string[] }>;
    showSave: (
      options: Record<string, unknown>,
    ) => Promise<{ canceled: boolean; filePath?: string }>;
    showMessage: (
      options: Record<string, unknown>,
    ) => Promise<{ response: number }>;
  };
  window: {
    getState: () => Promise<Record<string, unknown>>;
    setState: (state: Record<string, unknown>) => void;
  };
  on: (channel: string, callback: (...args: unknown[]) => void) => () => void;
  once: (channel: string, callback: (...args: unknown[]) => void) => void;
}

// Declare the global electron object
declare global {
  interface Window {
    electron?: ElectronAPI;
  }
}

/**
 * Check if running in Electron environment
 * Uses multiple detection methods for reliability
 */
export function isElectron(): boolean {
  if (!browser) return false;
  if (typeof window === "undefined") return false;

  // Method 1: Check for preload-exposed API
  if (typeof window.electron !== "undefined") return true;

  // Method 2: Check user agent for Electron
  if (navigator.userAgent.toLowerCase().includes("electron")) return true;

  // Method 3: Check for Electron-specific process object
  const win = window as Window & { process?: { type?: string } };
  if (win.process?.type === "renderer") return true;

  return false;
}

/**
 * OAuth deep-link return target for the desktop app, or null on the web.
 *
 * In Electron the packaged renderer runs on app://localhost and cannot receive
 * an http redirect from the external browser where OAuth consent happens, so we
 * ask the backend to bounce back through a businessos:// deep link that the main
 * process routes into the connectors page. On the web we return null and let the
 * backend use its default same-origin redirect.
 */
export function oauthRedirectUri(): string | null {
  return isElectron() ? "businessos://oauth" : null;
}

/**
 * Append the desktop OAuth deep-link redirect_uri to an auth-initiation path
 * when running in Electron. No-op on the web.
 */
export function withOAuthRedirect(path: string): string {
  const uri = oauthRedirectUri();
  if (!uri) return path;
  const sep = path.includes("?") ? "&" : "?";
  return `${path}${sep}redirect_uri=${encodeURIComponent(uri)}`;
}

/**
 * Check if running on macOS (for traffic light handling)
 */
export function isMacOS(): boolean {
  if (!browser) return false;
  return navigator.platform.toLowerCase().includes("mac");
}

/**
 * Check if running in a web browser (not Electron)
 */
export function isWebBrowser(): boolean {
  if (!browser) return false;
  return !isElectron();
}

/**
 * Get the Electron API if available
 * Returns undefined in web browser context
 */
export function getElectronAPI(): ElectronAPI | undefined {
  if (!browser) return undefined;
  return window.electron;
}

/**
 * Get the app version (Electron or from env/package.json)
 */
export async function getAppVersion(): Promise<string> {
  if (isElectron()) {
    const api = getElectronAPI();
    if (api) {
      return api.getVersion();
    }
  }
  // Fallback for web
  return "1.0.0";
}

/**
 * Get platform info
 */
export async function getPlatformInfo(): Promise<{
  platform: string;
  arch: string;
  isElectron: boolean;
  isPackaged: boolean;
}> {
  if (isElectron()) {
    const api = getElectronAPI();
    if (api) {
      const info = await api.getPlatform();
      return {
        ...info,
        isElectron: true,
      };
    }
  }

  // Web browser fallback
  const platform = browser ? navigator.platform.toLowerCase() : "unknown";
  return {
    platform: platform.includes("win")
      ? "win32"
      : platform.includes("mac")
        ? "darwin"
        : "linux",
    arch: "unknown",
    isElectron: false,
    isPackaged: false,
  };
}

/**
 * Get the API base URL
 * In Electron: uses the local backend sidecar
 * In Web: uses the configured API URL (relative or absolute)
 */
export async function getApiBaseUrl(): Promise<string> {
  if (isElectron()) {
    const api = getElectronAPI();
    if (api) {
      try {
        return await api.backend.getUrl();
      } catch {
        // Fallback to default
        return "http://localhost:18080"; // Electron backend fallback
      }
    }
  }

  // Web browser - use relative URL or environment variable
  return "/api";
}

/**
 * Open an external URL in the default browser
 * Works in both Electron and web
 */
export async function openExternal(url: string): Promise<void> {
  if (isElectron()) {
    const api = getElectronAPI();
    if (api) {
      await api.shell.openExternal(url);
      return;
    }
  }

  // Web browser fallback
  if (browser) {
    window.open(url, "_blank", "noopener,noreferrer");
  }
}

/**
 * Check online status
 */
export async function isOnline(): Promise<boolean> {
  if (isElectron()) {
    const api = getElectronAPI();
    if (api) {
      const status = await api.network.getStatus();
      return status.online;
    }
  }

  // Web browser fallback
  if (browser) {
    return navigator.onLine;
  }

  return true;
}

/**
 * Subscribe to navigation events from Electron menu
 * Returns unsubscribe function
 */
export function onNavigate(callback: (path: string) => void): () => void {
  if (!isElectron()) return () => {};

  const api = getElectronAPI();
  if (!api) return () => {};

  return api.on("navigate", (path) => {
    if (typeof path === "string") {
      callback(path);
    }
  });
}

/**
 * Subscribe to keyboard shortcuts from Electron menu
 * Returns unsubscribe function
 */
export function onShortcut(callback: (shortcut: string) => void): () => void {
  if (!isElectron()) return () => {};

  const api = getElectronAPI();
  if (!api) return () => {};

  return api.on("shortcut", (shortcut) => {
    if (typeof shortcut === "string") {
      callback(shortcut);
    }
  });
}
