import { createEngine, type EngineContext } from "@miosa/optimal-engine";
import { browser } from "$app/environment";

// Cached per-workspace Optimal Engine connection. The connection is configured
// in Settings > Optimal Engine (stored server-side on the workspace) and mirrored
// here in localStorage so getEngine() can build the client synchronously.
const CACHE_KEY = "businessos_engine_config";

export interface CachedEngineConfig {
  enabled: boolean;
  base_url: string;
  api_key: string;
  workspace: string;
}

let instance: EngineContext | null = null;

function electronEngine():
  | {
      setConfig?: (
        workspaceId: string,
        cfg: CachedEngineConfig,
      ) => Promise<unknown>;
    }
  | undefined {
  return (globalThis as unknown as {
    electron?: {
      engine?: {
        setConfig?: (
          workspaceId: string,
          cfg: CachedEngineConfig,
        ) => Promise<unknown>;
      };
    };
  }).electron?.engine;
}

function isLocalEngineUrl(baseUrl: string): boolean {
  try {
    const host = new URL(baseUrl).hostname.toLowerCase();
    return host === "localhost" || host === "127.0.0.1" || host === "::1";
  } catch {
    return false;
  }
}

export function getCachedEngineConfig(): CachedEngineConfig | null {
  if (!browser) return null;
  try {
    const raw = localStorage.getItem(CACHE_KEY);
    return raw ? (JSON.parse(raw) as CachedEngineConfig) : null;
  } catch {
    return null;
  }
}

export function setCachedEngineConfig(cfg: CachedEngineConfig | null): void {
  if (!browser) return;
  if (cfg) localStorage.setItem(CACHE_KEY, JSON.stringify(cfg));
  else localStorage.removeItem(CACHE_KEY);
  resetEngine();
}

// Fallback base URL when no workspace connection is configured.
function defaultBaseUrl(): string {
  if (!browser) return "http://localhost:4200";
  const isDev =
    window.location.hostname === "localhost" ||
    window.location.hostname === "127.0.0.1";
  if (isDev) return "http://localhost:4200";
  return `${window.location.origin}/api/v1/optimal-engine`;
}

export function getEngine(): EngineContext {
  if (!instance) {
    const cfg = getCachedEngineConfig();
    if (cfg && cfg.enabled && cfg.base_url) {
      instance = createEngine({
        baseUrl: cfg.base_url,
        apiKey: cfg.api_key || undefined,
        workspace: cfg.workspace || "default",
      });
    } else {
      instance = createEngine({
        baseUrl: defaultBaseUrl(),
        workspace: "default",
      });
    }
  }
  return instance;
}

export function resetEngine(): void {
  instance = null;
}

// Fetch the workspace's engine connection from the backend, cache it, and
// rebuild the client. Call after the active workspace is known. The API client
// is imported lazily to keep this module light for callers that only read.
export async function syncEngineConfig(workspaceId: string): Promise<void> {
  if (!browser || !workspaceId) return;
  try {
    const { getEngineConfig } = await import("$lib/api/workspace-admin");
    const cfg = await getEngineConfig(workspaceId);
    const cached = {
      enabled: cfg.enabled ?? false,
      base_url: cfg.base_url ?? "",
      api_key: cfg.api_key ?? "",
      workspace: cfg.workspace ?? "",
    };
    setCachedEngineConfig(cached);

    // Backend settings are canonical, but localhost engines must also be
    // known to Electron because local writes happen from the user's machine.
    // This keeps admin/migration-created configs working, not only configs
    // saved manually through the Settings panel.
    if (cached.enabled && cached.base_url && isLocalEngineUrl(cached.base_url)) {
      void electronEngine()?.setConfig?.(workspaceId, cached).catch(() => {});
    }
  } catch {
    // Engine not configured or unreachable — keep the fallback.
  }
}
