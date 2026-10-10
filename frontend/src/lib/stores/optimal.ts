import { writable } from "svelte/store";
import { browser } from "$app/environment";
import { getApiBaseUrl, getCSRFToken } from "$lib/api/base";

// ─── Types ───────────────────────────────────────────────────────────────────

export interface OptimalNode {
  slug: string;
  name: string;
  type: string;
  context_md: string;
  signal_md: string;
  signal_count: number;
  has_signals: boolean;
}

export interface FileEntry {
  name: string;
  path: string;
  is_dir: boolean;
  size: number;
  children?: FileEntry[];
}

export interface SelectedFile {
  content: string;
  path: string;
  slug: string;
}

export interface OptimalSignalFile {
  slug: string;
  date: string;
  name: string;
  path: string;
  content: string;
}

export interface OptimalSearchResult {
  path: string;
  abstract: string;
  score: number;
}

export interface RhythmDay {
  date: string;
  content: string;
}

export interface TeamMember {
  name: string;
  role: string;
  nodes: string[];
  channel: string;
  status: "active" | "inactive" | "exited";
}

export interface RevenueStream {
  name: string;
  model: string;
  amount: number;
  target: number;
  node: string;
}

export interface RevenueData {
  streams: RevenueStream[];
  total_mrr: number;
}

export interface ProjectEntry {
  name: string;
  node: string;
  path: string;
  files: string[];
}

export interface WeeklyPlan {
  week_of: string;
  content: string;
  non_negotiables: string[];
}

export interface DashboardData {
  nodes: { slug: string; name: string; type: string; signal_count: number }[];
  command_center: {
    content: string;
    blockers: string[];
    non_negotiables: string[];
  };
  revenue: { total_mrr: number; stream_count: number };
  stats: {
    context_count: number;
    signal_count: number;
    entity_count: number;
    edge_count: number;
  };
}

export interface HealthCheck {
  status: "healthy" | "degraded" | "error";
  checks: { name: string; status: string; message: string }[];
}

// ─── Store State ─────────────────────────────────────────────────────────────

interface OptimalState {
  nodes: OptimalNode[];
  selectedNode: OptimalNode | null;
  selectedNodeSignals: OptimalSignalFile[];
  todayRhythm: RhythmDay | null;
  searchResults: OptimalSearchResult[];
  fileTree: FileEntry[];
  selectedFile: SelectedFile | null;
  loading: boolean;
  error: string | null;
  team: TeamMember[];
  revenue: RevenueData | null;
  projects: ProjectEntry[];
  weeklyPlan: WeeklyPlan | null;
  dashboard: DashboardData | null;
  health: HealthCheck | null;
}

// ─── Store ───────────────────────────────────────────────────────────────────

function createOptimalStore() {
  const initialState: OptimalState = {
    nodes: [],
    selectedNode: null,
    selectedNodeSignals: [],
    todayRhythm: null,
    searchResults: [],
    fileTree: [],
    selectedFile: null,
    loading: false,
    error: null,
    team: [],
    revenue: null,
    projects: [],
    weeklyPlan: null,
    dashboard: null,
    health: null,
  };

  const { subscribe, update } = writable<OptimalState>(initialState);

  function buildHeaders(includeContentType = false): Record<string, string> {
    const headers: Record<string, string> = {};
    const csrfToken = getCSRFToken();
    if (csrfToken) headers["X-CSRF-Token"] = csrfToken;
    if (includeContentType) headers["Content-Type"] = "application/json";
    return headers;
  }

  return {
    subscribe,

    /** Fetch all OptimalOS nodes — GET /api/optimal/nodes */
    async loadNodes(): Promise<void> {
      if (!browser) return;

      update((s) => ({ ...s, loading: true, error: null }));

      try {
        const res = await fetch(`${getApiBaseUrl()}/optimal/nodes`, {
          method: "GET",
          headers: buildHeaders(),
          credentials: "include",
          signal: AbortSignal.timeout(8000),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        const data: { nodes?: OptimalNode[] } = await res.json();
        const nodes = Array.isArray(data.nodes) ? data.nodes : [];

        update((s) => ({ ...s, nodes, loading: false }));
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to load nodes";
        update((s) => ({ ...s, loading: false, error: message }));
      }
    },

    /**
     * Select a node by slug — GET /api/optimal/nodes/:slug
     * Also loads signals for the selected node.
     */
    async selectNode(slug: string): Promise<void> {
      if (!browser) return;

      update((s) => ({
        ...s,
        loading: true,
        error: null,
        selectedNode: null,
        selectedNodeSignals: [],
      }));

      try {
        const res = await fetch(
          `${getApiBaseUrl()}/optimal/nodes/${encodeURIComponent(slug)}`,
          {
            method: "GET",
            headers: buildHeaders(),
            credentials: "include",
            signal: AbortSignal.timeout(8000),
          },
        );

        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        const selectedNode: OptimalNode = await res.json();

        update((s) => ({ ...s, selectedNode, loading: false }));
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to load node";
        update((s) => ({ ...s, loading: false, error: message }));
        return;
      }

      // Load signals in parallel after node is resolved
      await this.loadNodeSignals(slug);
    },

    /** Load signal files for a node — GET /api/optimal/nodes/:slug/signals */
    async loadNodeSignals(slug: string): Promise<void> {
      if (!browser) return;

      try {
        const res = await fetch(
          `${getApiBaseUrl()}/optimal/nodes/${encodeURIComponent(slug)}/signals`,
          {
            method: "GET",
            headers: buildHeaders(),
            credentials: "include",
            signal: AbortSignal.timeout(8000),
          },
        );

        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        const data: { signals?: OptimalSignalFile[] } = await res.json();
        const signals = Array.isArray(data.signals) ? data.signals : [];

        update((s) => ({ ...s, selectedNodeSignals: signals }));
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to load signals";
        update((s) => ({ ...s, error: message }));
      }
    },

    /** Load today's rhythm file — GET /api/optimal/rhythm/today */
    async loadTodayRhythm(): Promise<void> {
      if (!browser) return;

      try {
        const res = await fetch(`${getApiBaseUrl()}/optimal/rhythm/today`, {
          method: "GET",
          headers: buildHeaders(),
          credentials: "include",
          signal: AbortSignal.timeout(5000),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        const todayRhythm: RhythmDay = await res.json();

        update((s) => ({ ...s, todayRhythm }));
      } catch {
        // Today's rhythm is optional — silently ignore missing/unavailable
      }
    },

    /** Search the OptimalOS context engine — POST /api/optimal/search */
    async search(query: string, limit = 10): Promise<void> {
      if (!browser || !query.trim()) return;

      update((s) => ({ ...s, loading: true, error: null, searchResults: [] }));

      try {
        const res = await fetch(`${getApiBaseUrl()}/optimal/search`, {
          method: "POST",
          headers: buildHeaders(true),
          credentials: "include",
          signal: AbortSignal.timeout(10000),
          body: JSON.stringify({ query: query.trim(), limit }),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        const data: { results?: OptimalSearchResult[] } = await res.json();
        const searchResults = Array.isArray(data.results) ? data.results : [];

        update((s) => ({ ...s, searchResults, loading: false }));
      } catch (err) {
        const message = err instanceof Error ? err.message : "Search failed";
        update((s) => ({ ...s, loading: false, error: message }));
      }
    },

    /** Ingest a signal into the OptimalOS engine — POST /api/optimal/ingest */
    async ingest(text: string, genre?: string): Promise<void> {
      if (!browser || !text.trim()) return;

      update((s) => ({ ...s, loading: true, error: null }));

      try {
        const body: Record<string, string> = { text: text.trim() };
        if (genre) body.genre = genre;

        const res = await fetch(`${getApiBaseUrl()}/optimal/ingest`, {
          method: "POST",
          headers: buildHeaders(true),
          credentials: "include",
          signal: AbortSignal.timeout(15000),
          body: JSON.stringify(body),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        update((s) => ({ ...s, loading: false }));
      } catch (err) {
        const message = err instanceof Error ? err.message : "Ingest failed";
        update((s) => ({ ...s, loading: false, error: message }));
      }
    },

    /** Load file tree for a node — GET /api/optimal/nodes/:slug/files */
    async loadFileTree(slug: string): Promise<void> {
      if (!browser) return;

      update((s) => ({
        ...s,
        loading: true,
        error: null,
        fileTree: [],
        selectedFile: null,
      }));

      try {
        const res = await fetch(
          `${getApiBaseUrl()}/optimal/nodes/${encodeURIComponent(slug)}/files`,
          {
            method: "GET",
            headers: buildHeaders(),
            credentials: "include",
            signal: AbortSignal.timeout(8000),
          },
        );

        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        const data: { files?: FileEntry[] } = await res.json();
        const fileTree = Array.isArray(data.files) ? data.files : [];

        update((s) => ({ ...s, fileTree, loading: false }));
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to load file tree";
        update((s) => ({ ...s, loading: false, error: message }));
      }
    },

    /** Load a single file — GET /api/optimal/nodes/:slug/file/*filepath */
    async loadFile(slug: string, filePath: string): Promise<void> {
      if (!browser) return;

      update((s) => ({ ...s, loading: true, error: null }));

      try {
        // Strip the node slug prefix from the path if present
        // (file tree returns "04-ai-masters/community/file.md" but API expects "community/file.md")
        let cleanPath = filePath;
        if (cleanPath.startsWith(slug + "/")) {
          cleanPath = cleanPath.slice(slug.length + 1);
        }
        const encoded = cleanPath
          .split("/")
          .map((seg) => encodeURIComponent(seg))
          .join("/");

        const res = await fetch(
          `${getApiBaseUrl()}/optimal/nodes/${encodeURIComponent(slug)}/file/${encoded}`,
          {
            method: "GET",
            headers: buildHeaders(),
            credentials: "include",
            signal: AbortSignal.timeout(8000),
          },
        );

        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        const selectedFile: SelectedFile = await res.json();

        update((s) => ({ ...s, selectedFile, loading: false }));
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to load file";
        update((s) => ({ ...s, loading: false, error: message }));
      }
    },

    /** Load dashboard summary — GET /api/optimal/dashboard (optional, silently skips on 404) */
    async loadDashboard(): Promise<void> {
      if (!browser) return;

      try {
        const res = await fetch(`${getApiBaseUrl()}/optimal/dashboard`, {
          method: "GET",
          headers: buildHeaders(),
          credentials: "include",
          signal: AbortSignal.timeout(8000),
        });

        // 404 = backend endpoint not yet deployed; silently skip
        if (res.status === 404) return;
        if (!res.ok) return; // other errors are also non-fatal for this optional section

        const data: DashboardData = await res.json();
        update((s) => ({ ...s, dashboard: data }));
      } catch {
        // Dashboard data is optional — silently ignore network errors
      }
    },

    /** Load team members — GET /api/optimal/team */
    async loadTeam(): Promise<void> {
      if (!browser) return;

      update((s) => ({ ...s, loading: true, error: null }));

      try {
        const res = await fetch(`${getApiBaseUrl()}/optimal/team`, {
          method: "GET",
          headers: buildHeaders(),
          credentials: "include",
          signal: AbortSignal.timeout(8000),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        const data: { members?: TeamMember[] } = await res.json();
        const team = Array.isArray(data.members) ? data.members : [];

        update((s) => ({ ...s, team, loading: false }));
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to load team";
        update((s) => ({ ...s, loading: false, error: message }));
      }
    },

    /** Load revenue data — GET /api/optimal/revenue */
    async loadRevenue(): Promise<void> {
      if (!browser) return;

      update((s) => ({ ...s, loading: true, error: null }));

      try {
        const res = await fetch(`${getApiBaseUrl()}/optimal/revenue`, {
          method: "GET",
          headers: buildHeaders(),
          credentials: "include",
          signal: AbortSignal.timeout(8000),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        const data: RevenueData = await res.json();

        update((s) => ({ ...s, revenue: data, loading: false }));
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to load revenue";
        update((s) => ({ ...s, loading: false, error: message }));
      }
    },

    /** Load projects — GET /api/optimal/projects */
    async loadProjects(): Promise<void> {
      if (!browser) return;

      update((s) => ({ ...s, loading: true, error: null }));

      try {
        const res = await fetch(`${getApiBaseUrl()}/optimal/projects`, {
          method: "GET",
          headers: buildHeaders(),
          credentials: "include",
          signal: AbortSignal.timeout(8000),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        const data: { projects?: ProjectEntry[] } = await res.json();
        const projects = Array.isArray(data.projects) ? data.projects : [];

        update((s) => ({ ...s, projects, loading: false }));
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to load projects";
        update((s) => ({ ...s, loading: false, error: message }));
      }
    },

    /** Load weekly rhythm plan — GET /api/optimal/rhythm/weekly */
    async loadWeeklyRhythm(): Promise<void> {
      if (!browser) return;

      update((s) => ({ ...s, loading: true, error: null }));

      try {
        const res = await fetch(`${getApiBaseUrl()}/optimal/rhythm/weekly`, {
          method: "GET",
          headers: buildHeaders(),
          credentials: "include",
          signal: AbortSignal.timeout(8000),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        const weeklyPlan: WeeklyPlan = await res.json();

        update((s) => ({ ...s, weeklyPlan, loading: false }));
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to load weekly rhythm";
        update((s) => ({ ...s, loading: false, error: message }));
      }
    },

    /** Load engine health check — GET /api/optimal/health */
    async loadHealth(): Promise<void> {
      if (!browser) return;

      update((s) => ({ ...s, loading: true, error: null }));

      try {
        const res = await fetch(`${getApiBaseUrl()}/optimal/health`, {
          method: "GET",
          headers: buildHeaders(),
          credentials: "include",
          signal: AbortSignal.timeout(5000),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        const health: HealthCheck = await res.json();

        update((s) => ({ ...s, health, loading: false }));
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to load health";
        update((s) => ({ ...s, loading: false, error: message }));
      }
    },

    /** Save file content — PUT /api/optimal/nodes/:slug/file/*filepath */
    async saveFile(
      slug: string,
      filePath: string,
      content: string,
    ): Promise<boolean> {
      if (!browser) return false;

      try {
        let cleanPath = filePath;
        if (cleanPath.startsWith(slug + "/")) {
          cleanPath = cleanPath.slice(slug.length + 1);
        }
        const encoded = cleanPath
          .split("/")
          .map((seg) => encodeURIComponent(seg))
          .join("/");

        const res = await fetch(
          `${getApiBaseUrl()}/optimal/nodes/${encodeURIComponent(slug)}/file/${encoded}`,
          {
            method: "PUT",
            headers: buildHeaders(true),
            credentials: "include",
            signal: AbortSignal.timeout(15000),
            body: JSON.stringify({ content }),
          },
        );

        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return true;
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to save file";
        update((s) => ({ ...s, error: message }));
        return false;
      }
    },

    /** Create a new file — POST /api/optimal/nodes/:slug/file */
    async createFile(
      slug: string,
      path: string,
      content: string,
    ): Promise<boolean> {
      if (!browser) return false;

      try {
        const res = await fetch(
          `${getApiBaseUrl()}/optimal/nodes/${encodeURIComponent(slug)}/file`,
          {
            method: "POST",
            headers: buildHeaders(true),
            credentials: "include",
            signal: AbortSignal.timeout(15000),
            body: JSON.stringify({ path, content }),
          },
        );

        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return true;
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to create file";
        update((s) => ({ ...s, error: message }));
        return false;
      }
    },

    /** Load system diagnostic data — GET /api/optimal/diagnose */
    async loadDiagnose(): Promise<any> {
      if (!browser) return null;

      try {
        const res = await fetch(`${getApiBaseUrl()}/optimal/diagnose`, {
          method: "GET",
          headers: buildHeaders(),
          credentials: "include",
          signal: AbortSignal.timeout(10000),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return await res.json();
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to load diagnose";
        update((s) => ({ ...s, error: message }));
        return null;
      }
    },

    /** Persist a memory signal — POST /api/optimal/remember */
    async remember(
      content: string,
      category?: string,
      source?: string,
    ): Promise<any> {
      if (!browser) return null;

      try {
        const body: Record<string, string> = { content };
        if (category) body.category = category;
        if (source) body.source = source;

        const res = await fetch(`${getApiBaseUrl()}/optimal/remember`, {
          method: "POST",
          headers: buildHeaders(true),
          credentials: "include",
          signal: AbortSignal.timeout(15000),
          body: JSON.stringify(body),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return await res.json();
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to remember";
        update((s) => ({ ...s, error: message }));
        return null;
      }
    },

    /** Trigger a rethink on a topic — POST /api/optimal/rethink */
    async rethink(topic: string): Promise<any> {
      if (!browser) return null;

      try {
        const res = await fetch(`${getApiBaseUrl()}/optimal/rethink`, {
          method: "POST",
          headers: buildHeaders(true),
          credentials: "include",
          signal: AbortSignal.timeout(30000),
          body: JSON.stringify({ topic }),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return await res.json();
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to rethink";
        update((s) => ({ ...s, error: message }));
        return null;
      }
    },

    /** Load escalations — GET /api/optimal/escalations */
    async loadEscalations(): Promise<any> {
      if (!browser) return null;

      try {
        const res = await fetch(`${getApiBaseUrl()}/optimal/escalations`, {
          method: "GET",
          headers: buildHeaders(),
          credentials: "include",
          signal: AbortSignal.timeout(8000),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return await res.json();
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to load escalations";
        update((s) => ({ ...s, error: message }));
        return null;
      }
    },

    /** Reweave context for a topic — POST /api/optimal/reweave */
    async reweave(topic: string, maxDays?: number): Promise<any> {
      if (!browser) return null;

      try {
        const body: Record<string, string | number> = { topic };
        if (maxDays !== undefined) body.max_days = maxDays;

        const res = await fetch(`${getApiBaseUrl()}/optimal/reweave`, {
          method: "POST",
          headers: buildHeaders(true),
          credentials: "include",
          signal: AbortSignal.timeout(30000),
          body: JSON.stringify(body),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return await res.json();
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to reweave";
        update((s) => ({ ...s, error: message }));
        return null;
      }
    },

    /** Verify L0 integrity — POST /api/optimal/verify */
    async verifyL0(sampleSize?: number): Promise<any> {
      if (!browser) return null;

      try {
        const body: Record<string, number> = {};
        if (sampleSize !== undefined) body.sample_size = sampleSize;

        const res = await fetch(`${getApiBaseUrl()}/optimal/verify`, {
          method: "POST",
          headers: buildHeaders(true),
          credentials: "include",
          signal: AbortSignal.timeout(30000),
          body: JSON.stringify(body),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return await res.json();
      } catch (err) {
        const message = err instanceof Error ? err.message : "Failed to verify";
        update((s) => ({ ...s, error: message }));
        return null;
      }
    },

    /** Compose a signal — POST /api/optimal/compose */
    async compose(
      content: string,
      receiver?: string,
      genre?: string,
    ): Promise<any> {
      if (!browser) return null;

      try {
        const body: Record<string, string> = { content };
        if (receiver) body.receiver = receiver;
        if (genre) body.genre = genre;

        const res = await fetch(`${getApiBaseUrl()}/optimal/compose`, {
          method: "POST",
          headers: buildHeaders(true),
          credentials: "include",
          signal: AbortSignal.timeout(15000),
          body: JSON.stringify(body),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return await res.json();
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to compose";
        update((s) => ({ ...s, error: message }));
        return null;
      }
    },

    /** Clear the currently selected file */
    clearSelectedFile(): void {
      update((s) => ({ ...s, selectedFile: null }));
    },

    /** Clear search results */
    clearSearch(): void {
      update((s) => ({ ...s, searchResults: [] }));
    },

    /** Clear selected node and its signals */
    clearSelectedNode(): void {
      update((s) => ({
        ...s,
        selectedNode: null,
        selectedNodeSignals: [],
      }));
    },

    /** Dismiss the current error */
    clearError(): void {
      update((s) => ({ ...s, error: null }));
    },
  };
}

export const optimalStore = createOptimalStore();
