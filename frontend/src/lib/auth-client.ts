import { writable, get } from "svelte/store";
import { getCSRFToken, initCSRF } from "$lib/api/base";
import {
  LOCAL_BACKEND_URL,
  PRODUCTION_BACKEND_URL,
  getDefaultCloudBackendUrl,
} from "$lib/config/runtime";
import { openExternal } from "$lib/utils/platform";

function isElectronRuntime(): boolean {
  return typeof window !== "undefined" && "electron" in window;
}

function isDevelopmentRenderer(): boolean {
  return (
    typeof window !== "undefined" &&
    window.location.protocol !== "app:" &&
    (window.location.hostname === "localhost" ||
      window.location.hostname === "127.0.0.1")
  );
}

// App mode store - 'cloud' or 'local'
export const appMode = writable<"cloud" | "local" | null>(null);
export const cloudServerUrl = writable<string>("");

// Initialize mode from localStorage
if (typeof window !== "undefined") {
  let savedMode = localStorage.getItem("businessos_mode") as
    "cloud" | "local" | null;

  // Always use the correct backend URL for the current environment
  // Never trust stale localStorage values from old deployments
  const correctUrl = getDefaultCloudBackendUrl();
  let savedUrl = correctUrl;
  localStorage.setItem("businessos_cloud_url", correctUrl);

  if (!savedMode) {
    savedMode = "cloud";
    localStorage.setItem("businessos_mode", "cloud");
  }

  appMode.set(savedMode);
  cloudServerUrl.set(savedUrl);
}

// Save mode to localStorage
export function setAppMode(mode: "cloud" | "local", serverUrl?: string) {
  appMode.set(mode);
  localStorage.setItem("businessos_mode", mode);
  if (mode === "cloud" && serverUrl) {
    cloudServerUrl.set(serverUrl);
    localStorage.setItem("businessos_cloud_url", serverUrl);
  }
  // Reload to apply new settings
  window.location.reload();
}

// Helper: resolve base URL for auth API calls.
// In web dev mode (not Electron), return "" so fetch uses relative URLs
// routed through the Vite proxy. This keeps cookies on the same origin
// (localhost:5173) so SvelteKit server-side code can read them.
function getAuthBase(serverUrl?: string): string {
  if (serverUrl) return serverUrl;
  // Electron development renders from localhost. Route session/email requests
  // through Vite so they remain same-origin; Vite forwards the localhost
  // session cookie installed by the deep-link handler to the cloud backend.
  // Google OAuth itself still uses the absolute cloud URL below.
  if (isElectronRuntime() && isDevelopmentRenderer()) return "";
  // Web (dev OR prod): same-origin, so every auth call (sign-in, sign-up,
  // getSession) goes through the Vite proxy (dev) or the Cloudflare Pages proxy
  // (prod) and the session cookie is first-party on this host. Using an absolute
  // cloud URL here set the cookie on a different host than the session check read
  // from → infinite redirect back to login.
  if (!isElectronRuntime()) return "";
  return get(cloudServerUrl); // Electron talks to the cloud backend directly.
}

// Google profile photos are served through the authenticated backend instead of
// directly from googleusercontent.com. This keeps avatar rendering reliable in
// Electron and keeps the browser's network policy in one place.
export function getUserAvatarUrl(image?: string | null): string {
  if (!image) return "";
  const base = getAuthBase();
  if (image.startsWith("/")) return `${base}${image}`;
  return `${base}/api/auth/avatar`;
}

// Helper function to add CSRF token to headers
async function addCSRFHeaders(headers: HeadersInit = {}): Promise<HeadersInit> {
  let csrfToken = getCSRFToken();
  if (!csrfToken) {
    await initCSRF();
    csrfToken = getCSRFToken();
  }

  if (csrfToken) {
    return {
      ...headers,
      "X-CSRF-Token": csrfToken,
    };
  }
  return headers;
}

// Google OAuth - initiate OAuth flow
// Returns true on success, false on failure
// Request a password-reset email. Returns ok=true on success (the backend
// always succeeds to avoid leaking which emails have accounts).
export async function requestPasswordReset(
  email: string,
): Promise<{ ok: boolean; message?: string; error?: string }> {
  const base = getAuthBase();
  try {
    const res = await fetch(`${base}/api/auth/forget-password`, {
      method: "POST",
      credentials: "include",
      headers: await addCSRFHeaders({ "Content-Type": "application/json" }),
      body: JSON.stringify({ email }),
    });
    const data = await res.json().catch(() => ({}));
    if (res.ok) return { ok: true, message: data.message };
    return { ok: false, error: data.error || "Could not send reset email" };
  } catch (e) {
    return {
      ok: false,
      error: e instanceof Error ? e.message : "Network error",
    };
  }
}

// Complete a password reset with the emailed token.
export async function resetPasswordWithToken(
  token: string,
  newPassword: string,
): Promise<{ ok: boolean; message?: string; error?: string }> {
  const base = getAuthBase();
  try {
    const res = await fetch(`${base}/api/auth/reset-password`, {
      method: "POST",
      credentials: "include",
      headers: await addCSRFHeaders({ "Content-Type": "application/json" }),
      body: JSON.stringify({ token, newPassword }),
    });
    const data = await res.json().catch(() => ({}));
    if (res.ok) return { ok: true, message: data.message };
    return { ok: false, error: data.error || "Could not reset password" };
  } catch (e) {
    return {
      ok: false,
      error: e instanceof Error ? e.message : "Network error",
    };
  }
}

export function initiateGoogleOAuth(serverUrl?: string): boolean {
  // Web: same-origin ("") so the request goes through the Cloudflare Pages
  // proxy to the backend (first-party cookies). Electron: use the absolute
  // cloud backend URL since it talks to Cloud Run directly.
  const baseUrl = isElectronRuntime()
    ? serverUrl || PRODUCTION_BACKEND_URL
    : "";
  if (isElectronRuntime() && !baseUrl) {
    return false;
  }

  // Starting a fresh sign-in — drop any prior logged-out flag so the post-OAuth
  // reload lands authenticated instead of being forced back to the login screen.
  clearLoggedOut();

  // Electron: redirect back via the registered businessos:// deep link. The
  // system browser can't open app://, and cookies set in the browser don't
  // reach the Electron session, so the backend hands the session token back on
  // this deep link and the main process installs it (see open-url handler).
  // Web: stay same-origin so the cookie is first-party.
  const redirectTarget = isElectronRuntime()
    ? "businessos://auth/callback"
    : window.location.origin + "/auth/callback";
  const redirectUrl = encodeURIComponent(redirectTarget);
  const authUrl = `${baseUrl}/api/auth/google?redirect=${redirectUrl}`;

  if (isElectronRuntime()) {
    // Use Electron's shell to open in system browser
    void openExternal(authUrl);
  } else {
    // Standard web redirect
    window.location.href = authUrl;
  }
  return true;
}

// Email/Password Sign Up
export async function signUpWithEmail(
  email: string,
  password: string,
  name: string,
  serverUrl?: string,
) {
  const baseUrl = getAuthBase(serverUrl);

  try {
    const response = await fetch(`${baseUrl}/api/auth/sign-up/email`, {
      method: "POST",
      headers: await addCSRFHeaders({ "Content-Type": "application/json" }),
      credentials: "include",
      body: JSON.stringify({ email, password, name }),
    });

    const data = await response.json();

    if (!response.ok) {
      return { error: { message: data.error || "Sign up failed" } };
    }

    return { data };
  } catch (err) {
    return { error: { message: (err as Error).message || "Network error" } };
  }
}

// Email/Password Sign In
export async function signInWithEmail(
  email: string,
  password: string,
  serverUrl?: string,
) {
  const baseUrl = getAuthBase(serverUrl);
  const url = `${baseUrl}/api/auth/sign-in/email`;
  const headers = await addCSRFHeaders({ "Content-Type": "application/json" });
  const bodyPayload = { email, password };

  try {
    const response = await fetch(url, {
      method: "POST",
      headers,
      credentials: "include",
      body: JSON.stringify(bodyPayload),
    });

    const data = await response.json();

    if (!response.ok) {
      return { error: { message: data.error || "Sign in failed" } };
    }

    clearLoggedOut();
    return { data };
  } catch (err) {
    return { error: { message: (err as Error).message || "Network error" } };
  }
}

// Get current session from server
export async function getSession(serverUrl?: string) {
  if (isLoggedOut()) {
    return { data: null, error: "Logged out" };
  }

  const baseUrl = getAuthBase(serverUrl);

  try {
    const response = await fetch(`${baseUrl}/api/auth/session`, {
      method: "GET",
      credentials: "include",
    });

    if (!response.ok) {
      return { data: null, error: "Not authenticated" };
    }

    // Guard against non-JSON responses (e.g., HTML error pages)
    const contentType = response.headers.get("content-type") || "";
    if (!contentType.includes("application/json")) {
      return { data: null, error: "Invalid response from server" };
    }

    const data = await response.json();
    return { data, error: null };
  } catch (err) {
    return { data: null, error: (err as Error).message || "Network error" };
  }
}

// Check if user needs onboarding
// Returns { needsOnboarding: true/false, hasSession: true/false }
export async function checkOnboardingStatus(serverUrl?: string): Promise<{
  needsOnboarding: boolean;
  hasSession: boolean;
  error?: string;
}> {
  const baseUrl = getAuthBase(serverUrl);

  try {
    const response = await fetch(`${baseUrl}/api/v1/onboarding/status`, {
      method: "GET",
      credentials: "include",
    });

    if (!response.ok) {
      // If 401, user isn't authenticated
      if (response.status === 401) {
        return {
          needsOnboarding: false,
          hasSession: false,
          error: "Not authenticated",
        };
      }
      return {
        needsOnboarding: false,
        hasSession: false,
        error: "Failed to check onboarding status",
      };
    }

    const data = await response.json();
    return {
      needsOnboarding: data.needs_onboarding ?? false,
      hasSession: data.has_session ?? false,
    };
  } catch (err) {
    return {
      needsOnboarding: false,
      hasSession: false,
      error: (err as Error).message || "Network error",
    };
  }
}

// Sign out
// Returns { success: true } or { success: false, error: string }
export async function signOutFromServer(
  serverUrl?: string,
): Promise<{ success: boolean; error?: string }> {
  const baseUrl = getAuthBase(serverUrl);

  try {
    const response = await fetch(`${baseUrl}/api/auth/logout`, {
      method: "POST",
      credentials: "include",
      headers: await addCSRFHeaders(),
    });

    if (!response.ok) {
      return { success: false, error: `Server returned ${response.status}` };
    }
  } catch (err) {
    const errorMessage = err instanceof Error ? err.message : "Network error";
    return { success: false, error: errorMessage };
  }

  return { success: true };
}

// For Local mode: Create a local-only session
// IMPORTANT: This is a synthetic session for Electron local mode only.
// It does NOT represent actual server authentication.
// The `isLocalMode` flag distinguishes this from real authenticated sessions.
const localSession = writable({
  isPending: false,
  isLocalMode: true, // Flag to indicate this is a synthetic local session
  data: {
    user: {
      id: "local-user",
      email: "local@businessos.app",
      name: "Local User",
      image: undefined as string | undefined,
      platform_role: "user",
    },
    session: {
      id: "local-session",
    },
  },
  error: null,
});

// Local mode flag checked at session init (no console logging needed)

// For when mode is not yet selected - return a "pending" state
const pendingSession = writable({
  isPending: true,
  data: null,
  error: null,
});

// After an explicit logout we set a flag so the synthetic local-mode session
// (which always carries a user) stops reporting one until the user signs in
// again. This is what makes "Log out" actually land on the login screen.
const loggedOutSession = writable({
  isPending: false,
  data: null,
  error: null,
});
export function isLoggedOut(): boolean {
  return (
    typeof window !== "undefined" &&
    localStorage.getItem("businessos_logged_out") === "1"
  );
}
function markLoggedOut() {
  if (typeof window !== "undefined")
    localStorage.setItem("businessos_logged_out", "1");
}
function clearLoggedOut() {
  if (typeof window !== "undefined")
    localStorage.removeItem("businessos_logged_out");
}

// Get the base URL for auth
function getBaseURL(): string {
  if (typeof window === "undefined") return "http://localhost:5174";

  const mode = get(appMode);
  const serverUrl = get(cloudServerUrl);

  // Cloud mode with server URL
  if (mode === "cloud" && serverUrl) {
    return serverUrl;
  }

  // Local mode in Electron - use local backend
  if (isElectronRuntime()) {
    return LOCAL_BACKEND_URL;
  }

  // Web app - use current origin
  return window.location.origin;
}

// Cloud session store - fetched from server
const cloudSession = writable<{
  isPending: boolean;
  data: {
    user: {
      id: string;
      email: string;
      name: string;
      image?: string;
      createdAt?: string;
      platform_role?: string;
    };
    session: { id: string };
  } | null;
  error: string | null;
}>({
  isPending: true,
  data: null,
  error: null,
});

// Fetch cloud session on init (only in cloud mode)
async function initCloudSession() {
  const mode = get(appMode);
  if (mode !== "cloud") return;

  cloudSession.set({ isPending: true, data: null, error: null });

  try {
    const result = await getSession();
    if (result.data?.user) {
      cloudSession.set({ isPending: false, data: result.data, error: null });
    } else {
      cloudSession.set({
        isPending: false,
        data: null,
        error: result.error || null,
      });
    }
  } catch (err) {
    cloudSession.set({
      isPending: false,
      data: null,
      error: (err as Error).message,
    });
  }
}

// Clear session data (call this when receiving 401 from API)
export function clearSession() {
  cloudSession.set({ isPending: false, data: null, error: "Session expired" });
}

// Re-check session from server
export async function refreshSession() {
  await initCloudSession();
}

// Initialize cloud session when mode changes to cloud
if (typeof window !== "undefined") {
  appMode.subscribe((mode) => {
    if (mode === "cloud") {
      initCloudSession();
    }
  });
}

// Local mode auth functions (for compatibility)
const localSignIn = {
  email: async ({ email, password }: { email: string; password: string }) => {
    return signInWithEmail(email, password);
  },
  social: async () => {
    clearLoggedOut();
    return { data: get(localSession).data, error: null };
  },
};
const localSignUp = {
  email: async ({
    email,
    password,
    name,
  }: {
    email: string;
    password: string;
    name: string;
  }) => {
    return signUpWithEmail(email, password, name);
  },
};
// Cloud mode auth functions
const cloudSignIn = {
  email: async ({ email, password }: { email: string; password: string }) => {
    const result = await signInWithEmail(email, password);
    if (result.data) {
      await initCloudSession();
    }
    return result;
  },
  social: async () => {
    initiateGoogleOAuth();
    return { data: null, error: null };
  },
};
const cloudSignUp = {
  email: async ({
    email,
    password,
    name,
  }: {
    email: string;
    password: string;
    name: string;
  }) => {
    const result = await signUpWithEmail(email, password, name);
    if (result.data) {
      await initCloudSession();
    }
    return result;
  },
};
// Export auth functions — each method checks mode at call time, not at import time.
// This avoids the frozen-IIFE problem where mode was captured once on module load.
export const signIn = {
  email: async (params: { email: string; password: string }) => {
    const mode = typeof window !== "undefined" ? get(appMode) : null;
    if (isElectronRuntime() && mode === "local") return localSignIn.email(params);
    return cloudSignIn.email(params);
  },
  social: async () => {
    const mode = typeof window !== "undefined" ? get(appMode) : null;
    if (isElectronRuntime() && mode === "local") return localSignIn.social();
    return cloudSignIn.social();
  },
};

export const signUp = {
  email: async (params: { email: string; password: string; name: string }) => {
    const mode = typeof window !== "undefined" ? get(appMode) : null;
    if (isElectronRuntime() && mode === "local") return localSignUp.email(params);
    return cloudSignUp.email(params);
  },
};

export const signOut = async () => {
  // 1. Best-effort: kill the server-side session + clear its cookie header.
  try {
    await signOutFromServer();
  } catch {
    /* network error — still proceed to clear client state */
  }
  // 2. Electron: clear the cookie persisted in the app's own partition, which
  //    is flushed to disk and would otherwise survive a reload and silently
  //    re-authenticate the user.
  try {
    const el =
      typeof window !== "undefined"
        ? (
            window as unknown as {
              electron?: { auth?: { clearSession?: () => Promise<unknown> } };
            }
          ).electron
        : undefined;
    if (isElectronRuntime() && el?.auth?.clearSession) {
      await el.auth.clearSession();
    }
  } catch {
    /* cookie clear failed — the flag below still forces the login screen */
  }
  // 3. Clear client session state + mark logged out so the synthetic local
  //    session stops reporting a user.
  cloudSession.set({ isPending: false, data: null, error: null });
  markLoggedOut();
  // 4. Land on the login screen and require a fresh sign-in.
  if (typeof window !== "undefined") {
    window.location.href = "/login";
  }
  return {};
};

export const useSession = () => {
  const mode = typeof window !== "undefined" ? get(appMode) : null;
  // After an explicit logout, report no user until the user signs in again —
  // otherwise the synthetic local session would keep them "logged in".
  if (isLoggedOut()) return loggedOutSession;
  // In Electron with no mode selected, return pending session
  if (isElectronRuntime() && mode === null) return pendingSession;
  // In local mode, return local session
  if (isElectronRuntime() && mode === "local") return localSession;
  // In cloud mode or web, use cloud session
  return cloudSession;
};
