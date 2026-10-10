// Client-side Optimal Engine connectivity, run from the user's own browser or
// desktop renderer so a LOCAL engine (e.g. http://localhost:4200) is reachable
// regardless of how BusinessOS is running - live web, desktop app, or local dev.
// The cloud backend cannot reach a user's localhost, so these calls deliberately
// originate on the user's machine. On desktop we prefer the Electron main
// process (avoids any mixed-content edge cases); otherwise a direct fetch works
// because the engine now allows cross-origin requests (CORS) with auth.

export interface EngineWorkspace {
  id: string;
  slug: string;
  name: string;
}

export interface EngineTestResult {
  reachable: boolean;
  status?: number;
  message: string;
}

export function isSelectableEngineWorkspace(slug: string): boolean {
  const normalized = slug.trim().toLowerCase();
  if (["default", "inbox", "team", "knowledge-intake"].includes(normalized)) {
    return false;
  }
  return !normalized.startsWith("default-") && !normalized.startsWith("benchmark-");
}

interface ElectronEngine {
  test?: (b: string, k?: string) => Promise<EngineTestResult>;
  workspaces?: (
    b: string,
    k?: string,
  ) => Promise<{ ok: boolean; workspaces: EngineWorkspace[]; message: string }>;
  createWorkspace?: (
    b: string,
    k: string | undefined,
    workspace: { slug: string; name: string; description?: string },
  ) => Promise<{
    ok: boolean;
    workspace?: EngineWorkspace;
    message: string;
  }>;
  status?: () => Promise<{
    running: boolean;
    url: string;
    port: number;
    dataDir: string;
  }>;
  setConfig?: (workspaceId: string, cfg: unknown) => Promise<unknown>;
  memory?: (
    workspaceId: string,
    payload: {
      content: string;
      citation?: string;
      metadata?: Record<string, unknown>;
    },
  ) => Promise<{ ok: boolean; skipped?: boolean; status?: number; message?: string }>;
}

function electronEngine(): ElectronEngine | undefined {
  return (globalThis as unknown as { electron?: { engine?: ElectronEngine } })
    .electron?.engine;
}

function authHeaders(apiKey?: string): Record<string, string> {
  const k = apiKey?.trim();
  return k ? { Authorization: `Bearer ${k}` } : {};
}

function clean(baseUrl: string): string {
  return baseUrl.trim().replace(/\/+$/, "");
}

/** Test reachability of an engine from the user's machine. */
export async function testEngine(
  baseUrl: string,
  apiKey?: string,
): Promise<EngineTestResult> {
  if (!baseUrl?.trim())
    return { reachable: false, message: "No engine URL set" };

  const el = electronEngine();
  if (el?.test) {
    try {
      return await el.test(baseUrl.trim(), apiKey);
    } catch {
      // fall through to direct fetch
    }
  }

  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), 5000);
  try {
    const res = await fetch(clean(baseUrl) + "/api/health", {
      headers: authHeaders(apiKey),
      signal: controller.signal,
    });
    return {
      reachable: res.ok,
      status: res.status,
      message: res.ok ? "Connected" : `Engine returned HTTP ${res.status}`,
    };
  } catch (e) {
    return {
      reachable: false,
      message: e instanceof Error ? e.message : "Engine unreachable",
    };
  } finally {
    clearTimeout(timer);
  }
}

/** List the workspaces that exist inside an engine. */
export async function listEngineWorkspaces(
  baseUrl: string,
  apiKey?: string,
): Promise<EngineWorkspace[]> {
  if (!baseUrl?.trim()) return [];

  const el = electronEngine();
  if (el?.workspaces) {
    try {
      const r = await el.workspaces(baseUrl.trim(), apiKey);
      if (r.ok) return r.workspaces;
    } catch {
      // fall through to direct fetch
    }
  }

  // Archived workspaces are historical records, not normal connection targets.
  // An explicit archive browser can request them later without flooding the
  // normal detector with retired brains.
  const res = await fetch(clean(baseUrl) + "/api/workspaces?status=active", {
    headers: authHeaders(apiKey),
  });
  if (!res.ok) throw new Error(`Engine returned HTTP ${res.status}`);
  const raw: unknown = await res.json();
  const list = Array.isArray(raw)
    ? raw
    : ((raw as { workspaces?: unknown[]; data?: unknown[] }).workspaces ??
      (raw as { data?: unknown[] }).data ??
      []);
  return (list as Record<string, unknown>[])
    .map((w) => ({
      id: String(w.id ?? w.slug ?? ""),
      slug: String(w.slug ?? w.id ?? ""),
      name: String(w.name ?? w.slug ?? w.id ?? ""),
    }))
    .filter((workspace) => isSelectableEngineWorkspace(workspace.slug));
}

/** Ensure one engine workspace exists for a BusinessOS workspace slug. */
export async function ensureEngineWorkspace(
  baseUrl: string,
  apiKey: string | undefined,
  workspace: { slug: string; name: string; description?: string },
): Promise<EngineWorkspace> {
  const slug = workspace.slug.trim().toLowerCase();
  if (!baseUrl?.trim() || !slug || !workspace.name.trim()) {
    throw new Error("Engine URL, workspace slug, and name are required");
  }

  const existing = (await listEngineWorkspaces(baseUrl, apiKey)).find(
    (candidate) => candidate.slug.toLowerCase() === slug,
  );
  if (existing) return existing;

  const input = { ...workspace, slug };
  const el = electronEngine();
  if (el?.createWorkspace) {
    const result = await el.createWorkspace(baseUrl.trim(), apiKey, input);
    if (result.ok && result.workspace) return result.workspace;
    throw new Error(result.message || "Could not create engine workspace");
  }

  const response = await fetch(clean(baseUrl) + "/api/workspaces", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...authHeaders(apiKey),
    },
    body: JSON.stringify(input),
  });
  if (!response.ok) {
    throw new Error(`Engine returned HTTP ${response.status}`);
  }
  const raw = (await response.json()) as Record<string, unknown>;
  return {
    id: String(raw.id ?? raw.slug ?? slug),
    slug: String(raw.slug ?? slug),
    name: String(raw.name ?? workspace.name),
  };
}

// Common addresses a locally-running Optimal Engine listens on (default :4200).
const DEFAULT_LOCAL_ENGINES = [
  "http://localhost:4200",
  "http://127.0.0.1:4200",
];

/**
 * Probe the user's machine for a running Optimal Engine and return the first
 * address that responds. Runs from the browser/renderer so a localhost engine
 * is reachable. Returns null when nothing is found.
 */
export async function detectLocalEngine(
  candidates: string[] = DEFAULT_LOCAL_ENGINES,
): Promise<string | null> {
  const el = electronEngine();
  if (el?.status) {
    try {
      const status = await el.status();
      if (status.running && status.url) return status.url;
    } catch {
      // Fall back to well-known local ports.
    }
  }
  for (const url of candidates) {
    try {
      const r = await testEngine(url);
      if (r.reachable) return url;
    } catch {
      /* try the next candidate */
    }
  }
  return null;
}

/**
 * Mirror a content write into the active workspace's LOCAL engine, from the
 * user's machine. Fire-and-forget and a no-op on web / non-Electron / when the
 * workspace's engine isn't localhost (the cloud backend already writes to
 * reachable engines). Never throws - it must not affect the originating write.
 */
export function mirrorToEngine(
  workspaceId: string,
  payload: {
    content: string;
    citation?: string;
    metadata?: Record<string, unknown>;
  },
): void {
  const el = electronEngine();
  if (!el?.memory || !workspaceId || !payload?.content) return;
  void el
    .memory(workspaceId, payload)
    .then((result) => {
      if (!result?.ok && !result?.skipped) {
        console.warn(
          "[optimal-engine] memory mirror failed",
          result?.status ?? "",
          result?.message ?? "",
        );
      }
    })
    .catch((error) => {
      console.warn(
        "[optimal-engine] memory mirror failed",
        error instanceof Error ? error.message : error,
      );
    });
}
