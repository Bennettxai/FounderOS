/**
 * Shim for `$lib/api/base` that the ported knowledge components import.
 *
 * In BusinessOS this file resolved a multi-backend URL (local OSA / cloud
 * API / electron) and emitted CSRF tokens. Here we hit the engine directly
 * over a Vite proxy in dev, or `VITE_ENGINE_URL` when running inside Tauri.
 *
 * The engine's API router mounts everything under `/api/...`, so the
 * components that were hitting `${base}/optimal/graph` in BusinessOS now
 * hit `/api/optimal/graph` here.
 */

const BASE = import.meta.env.VITE_ENGINE_URL ?? "";

export function getApiBaseUrl(): string {
  // Components concatenate `${base}/optimal/...`. Return `${BASE}/api` so
  // `/api/optimal/graph` is the final URL whether we're in dev (Vite
  // proxies /api → 127.0.0.1:4200) or production (VITE_ENGINE_URL points
  // at the engine host).
  return `${BASE}/api`;
}

/**
 * The engine doesn't require CSRF tokens — it's a localhost/desktop service
 * without cookie auth. Returning an empty string keeps the component call
 * sites identical to the BusinessOS ones so diffs stay minimal.
 */
export function getCSRFToken(): string {
  return "";
}
