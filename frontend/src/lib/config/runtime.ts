export const API_VERSION = "v1";

export const LOCAL_BACKEND_URL =
  import.meta.env.VITE_LOCAL_BACKEND_URL || "http://localhost:8801";
export const LOCAL_OSA_URL = "http://localhost:18080";
// Must be a *.businessos.dev host: the session cookie is Domain=.businessos.dev,
// so it only attaches to those hosts (not run.app, not the unrouted
// api.businessos.dev). app.businessos.dev's Cloudflare Pages function proxies
// /api/* to the Cloud Run backend.
export const PRODUCTION_BACKEND_URL =
  import.meta.env.VITE_BACKEND_URL || "https://app.businessos.dev";

function isBrowser(): boolean {
  return typeof window !== "undefined";
}

export function isLocalRuntimeLocation(
  protocol: string,
  hostname: string,
): boolean {
  if (protocol === "app:") return false;

  return hostname === "localhost" || hostname === "127.0.0.1";
}

export function resolveElectronApiBaseUrl(
  mode: string | null,
  cloudUrl: string | null,
  protocol: string,
  hostname: string,
  version: string | null,
): string {
  const suffix = version ? `/api/${version}` : "/api";

  if (mode === "local") return `${LOCAL_OSA_URL}${suffix}`;
  if (isLocalRuntimeLocation(protocol, hostname)) return version ? suffix : "/api";
  if (mode === "cloud" && cloudUrl) return `${cloudUrl}${suffix}`;
  return `${PRODUCTION_BACKEND_URL}${suffix}`;
}

function isLocalhost(): boolean {
  if (!isBrowser()) return false;

  // The PACKAGED desktop app serves the renderer from `app://localhost/`, whose
  // hostname is literally "localhost" - but that is the bundled-asset scheme, NOT
  // a real local dev server. Treating it as localhost pointed every API call
  // (including Google sign-in) at http://localhost:8801, which broke the app for
  // anyone who wasn't running a local backend. Only the http(s) dev server on
  // localhost is real localhost; the `app:` scheme means packaged -> use cloud.
  return isLocalRuntimeLocation(
    window.location.protocol,
    window.location.hostname,
  );
}

function isElectron(): boolean {
  return isBrowser() && "electron" in window;
}

function getStoredMode(): string | null {
  if (!isBrowser()) return null;
  return localStorage.getItem("businessos_mode");
}

function getStoredCloudUrl(): string | null {
  if (!isBrowser()) return null;
  return localStorage.getItem("businessos_cloud_url");
}

export function getDefaultCloudBackendUrl(): string {
  return isLocalhost() ? LOCAL_BACKEND_URL : PRODUCTION_BACKEND_URL;
}

export function getBackendUrl(): string {
  if (!isBrowser()) return "";

  if (isElectron()) {
    const mode = getStoredMode();
    const cloudUrl = getStoredCloudUrl() || getDefaultCloudBackendUrl();

    if (mode === "local") return LOCAL_OSA_URL;
    return cloudUrl;
  }

  if (import.meta.env.VITE_BACKEND_URL) {
    return import.meta.env.VITE_BACKEND_URL;
  }

  return getDefaultCloudBackendUrl();
}

export function getApiBaseUrl(version = API_VERSION): string {
  if (!isBrowser()) {
    return import.meta.env.VITE_API_URL || `/api/${version}`;
  }

  if (isElectron()) {
    const mode = getStoredMode();
    return resolveElectronApiBaseUrl(
      mode,
      getStoredCloudUrl(),
      window.location.protocol,
      window.location.hostname,
      version,
    );
  }

  // Web (browser, non-Electron): always same-origin. In production a Cloudflare
  // Pages Function proxies /api/* to the Cloud Run backend so the session cookie
  // is first-party; in dev Vite proxies /api/* to the local backend. The browser
  // only ever talks to its own origin, so cross-site cookie blocking never applies.
  return `/api/${version}`;
}

export function getLegacyApiBaseUrl(): string {
  if (!isBrowser()) {
    return import.meta.env.VITE_API_URL || "/api";
  }

  if (isElectron()) {
    return resolveElectronApiBaseUrl(
      getStoredMode(),
      getStoredCloudUrl(),
      window.location.protocol,
      window.location.hostname,
      null,
    );
  }

  // Web (browser, non-Electron): same-origin (see getApiBaseUrl).
  return "/api";
}
