import { writable, derived } from "svelte/store";
import { browser } from "$app/environment";
import { request, requestPaginated, getCSRFToken } from "$lib/api/base";
import type { PaginationMeta } from "$lib/api/base";
import type { AgentType, AgentStatus } from "$lib/types/agent";
import type { SandboxContainer } from "$lib/types/sandbox";
import {
  deploySandbox,
  stopSandbox as stopSandboxApi,
  restartSandbox as restartSandboxApi,
  removeSandbox as removeSandboxApi,
} from "$lib/api/sandbox";

export type AppStatus = "generating" | "generated" | "deployed" | "failed";
export type BuildPhase = "planning" | "generation" | "testing" | "deployment";
export type LogLevel = "info" | "warn" | "error" | "debug";

export interface AgentState {
  type: AgentType;
  status: AgentStatus;
  progress: number;
  message: string;
  started_at?: string;
  completed_at?: string;
}

export interface TimelineEvent {
  id: string;
  phase: BuildPhase;
  agent_type?: AgentType;
  event: string;
  timestamp: string;
  duration_ms?: number;
}

export interface GenerationLog {
  id: string;
  level: LogLevel;
  agent_type?: AgentType;
  message: string;
  timestamp: string;
  details?: Record<string, unknown>;
}

export interface GenerationState {
  app_id: string;
  current_phase: BuildPhase;
  agent_statuses: Map<AgentType, AgentState>;
  timeline: TimelineEvent[];
  logs: GenerationLog[];
  started_at: string;
  completed_at?: string;
}

export interface GeneratedApp {
  id: string;
  app_name: string;
  description: string;
  status: AppStatus;
  progress: number;
  build_phase?: BuildPhase;
  status_message?: string;
  error_message?: string | null;
  generated_at: string;
  deployed_at?: string | null;
  user_id: string;
  workspace_id: string;
  custom_config?: {
    description?: string;
    category?: string;
    keywords?: string[];
  };
  custom_icon?: string;
  sandbox?: SandboxContainer;
  /** Set when BUILD mode saves the app as an installable module */
  module_id?: string | null;
}

export const DEFAULT_PAGE_SIZE = 20;

interface GeneratedAppsStore {
  apps: GeneratedApp[];
  loading: boolean;
  error: string | null;
  filter: AppStatus | "all";
  searchQuery: string;
  activeGenerations: Map<string, GenerationState>;
  pagination: PaginationMeta;
  currentPage: number;
  pageSize: number;
}

function createInitialPagination(): PaginationMeta {
  return {
    page: 1,
    page_size: DEFAULT_PAGE_SIZE,
    total_items: 0,
    total_pages: 0,
    has_more: false,
  };
}

function createInitialState(): GeneratedAppsStore {
  return {
    apps: [],
    loading: false,
    error: null,
    filter: "all",
    searchQuery: "",
    activeGenerations: new Map(),
    pagination: createInitialPagination(),
    currentPage: 1,
    pageSize: DEFAULT_PAGE_SIZE,
  };
}

const initialState: GeneratedAppsStore = createInitialState();

// SSE connections map to track active streams
const sseConnections = new Map<string, EventSource>();

// Track reconnection attempts per app
const reconnectionAttempts = new Map<string, number>();
const MAX_RECONNECT_ATTEMPTS = 5;
const RECONNECT_DELAY = 2000; // 2s base delay
const FETCH_TIMEOUT = 30000; // 30s timeout

function createGeneratedAppsStore() {
  const { subscribe, set, update } = writable<GeneratedAppsStore>(initialState);

  // Filtered apps derived store
  const filteredApps = derived({ subscribe }, ($store) => {
    let filtered = $store.apps;

    // Apply status filter
    if ($store.filter !== "all") {
      filtered = filtered.filter((app) => app.status === $store.filter);
    }

    // Apply search query
    if ($store.searchQuery.trim()) {
      const query = $store.searchQuery.toLowerCase();
      filtered = filtered.filter(
        (app) =>
          app.app_name.toLowerCase().includes(query) ||
          app.description?.toLowerCase().includes(query),
      );
    }

    // Sort by date (newest first)
    return filtered.sort(
      (a, b) =>
        new Date(b.generated_at).getTime() - new Date(a.generated_at).getTime(),
    );
  });

  // Helper: Fetch with timeout support
  async function fetchWithTimeout(
    url: string,
    options: RequestInit = {},
  ): Promise<Response> {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), FETCH_TIMEOUT);

    try {
      const response = await fetch(url, {
        ...options,
        credentials: "include",
        signal: controller.signal,
      });
      return response;
    } finally {
      clearTimeout(timeout);
    }
  }

  // Helper: Fetch with exponential backoff retry
  async function fetchWithRetry(
    url: string,
    options: RequestInit = {},
    maxRetries: number = 3,
  ): Promise<Response> {
    for (let i = 0; i < maxRetries; i++) {
      try {
        const response = await fetchWithTimeout(url, options);
        if (!response.ok && i < maxRetries - 1) {
          // Retry on non-OK responses (except on last attempt)
          await new Promise((resolve) =>
            setTimeout(resolve, 1000 * Math.pow(2, i)),
          );
          continue;
        }
        return response;
      } catch (error) {
        if (i === maxRetries - 1) throw error;
        // Exponential backoff: 1s, 2s, 4s
        await new Promise((resolve) =>
          setTimeout(resolve, 1000 * Math.pow(2, i)),
        );
      }
    }
    throw new Error("Max retries exceeded");
  }

  async function fetchApps(page?: number, pageSize?: number) {
    if (!browser) return;

    update((state) => ({ ...state, loading: true, error: null }));

    try {
      // Read current pagination state when no explicit params provided
      let resolvedPage = page;
      let resolvedPageSize = pageSize;
      if (resolvedPage === undefined || resolvedPageSize === undefined) {
        const unsubscribe = { page: 1, pageSize: DEFAULT_PAGE_SIZE };
        // snapshot current state
        subscribe((s) => {
          unsubscribe.page = s.currentPage;
          unsubscribe.pageSize = s.pageSize;
        })();
        if (resolvedPage === undefined) resolvedPage = unsubscribe.page;
        if (resolvedPageSize === undefined)
          resolvedPageSize = unsubscribe.pageSize;
      }

      const result = await requestPaginated<GeneratedApp>(
        "/osa/module-instances",
        { page: resolvedPage, page_size: resolvedPageSize },
        "apps",
      );

      update((state) => ({
        ...state,
        apps: result.data,
        pagination: result.pagination,
        currentPage: result.pagination.page,
        pageSize: result.pagination.page_size,
        loading: false,
      }));
    } catch (err) {
      update((state) => ({
        ...state,
        loading: false,
        error: err instanceof Error ? err.message : "Failed to fetch apps",
      }));
    }
  }

  async function goToPage(page: number) {
    let currentPageSize = DEFAULT_PAGE_SIZE;
    subscribe((s) => {
      currentPageSize = s.pageSize;
    })();
    await fetchApps(page, currentPageSize);
  }

  async function setPageSize(pageSize: number) {
    // Reset to page 1 when page size changes
    await fetchApps(1, pageSize);
  }

  async function getAppById(appId: string): Promise<GeneratedApp | null> {
    if (!browser) return null;

    try {
      const data = await request<{ app: GeneratedApp }>(
        `/osa/module-instances/${appId}`,
      );
      return data.app;
    } catch (err) {
      console.error("Error fetching app:", err);
      return null;
    }
  }

  async function deployApp(appId: string): Promise<void> {
    if (!browser) return;

    try {
      // Use the sandbox API client for deployment (aligned with backend 2026-01-27)
      await deploySandbox(appId);

      // Refresh apps list
      await fetchApps();
    } catch (err) {
      throw new Error(
        err instanceof Error ? err.message : "Failed to deploy app",
      );
    }
  }

  async function deleteApp(appId: string): Promise<void> {
    if (!browser) return;

    // Save state before optimistic update (for rollback)
    let appToDelete: GeneratedApp | undefined;
    update((state) => {
      appToDelete = state.apps.find((a) => a.id === appId);
      return state;
    });

    // Optimistic update - remove immediately
    update((state) => ({
      ...state,
      apps: state.apps.filter((app) => app.id !== appId),
      error: null,
    }));

    try {
      // Use retry logic for delete with CSRF token
      const deleteHeaders: Record<string, string> = {};
      const csrfToken = getCSRFToken();
      if (csrfToken) deleteHeaders["X-CSRF-Token"] = csrfToken;

      const response = await fetchWithRetry(
        `/api/osa/module-instances/${appId}`,
        {
          method: "DELETE",
          headers: deleteHeaders,
        },
      );

      if (!response.ok) {
        throw new Error("Delete failed");
      }
    } catch (err) {
      // ROLLBACK on error - restore the deleted app
      if (appToDelete) {
        update((state) => ({
          ...state,
          apps: [...state.apps, appToDelete as GeneratedApp].sort(
            (a, b) =>
              new Date(b.generated_at).getTime() -
              new Date(a.generated_at).getTime(),
          ),
          error: "Failed to delete app. Please try again.",
        }));
      }
      throw new Error(
        err instanceof Error ? err.message : "Failed to delete app",
      );
    }
  }

  async function updateApp(
    appId: string,
    updates: Partial<GeneratedApp>,
  ): Promise<void> {
    if (!browser) return;

    try {
      // Use retry logic for update - PATCH method aligned with backend (2026-01-27)
      const patchHeaders: Record<string, string> = {
        "Content-Type": "application/json",
      };
      const patchCsrf = getCSRFToken();
      if (patchCsrf) patchHeaders["X-CSRF-Token"] = patchCsrf;

      const response = await fetchWithRetry(
        `/api/osa/module-instances/${appId}`,
        {
          method: "PATCH",
          headers: patchHeaders,
          body: JSON.stringify(updates),
        },
      );

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || "Failed to update app");
      }

      // Update local state
      update((state) => ({
        ...state,
        apps: state.apps.map((app) =>
          app.id === appId ? { ...app, ...updates } : app,
        ),
      }));
    } catch (err) {
      throw new Error(
        err instanceof Error ? err.message : "Failed to update app",
      );
    }
  }

  function subscribeToAppProgress(queueItemId: string): void {
    if (!browser) return;

    // Close existing connection if any
    const existing = sseConnections.get(queueItemId);
    if (existing) {
      existing.close();
      sseConnections.delete(queueItemId);
    }

    // Reset reconnection attempts
    reconnectionAttempts.set(queueItemId, 0);

    // Create new SSE connection (canonical queue-based route)
    const eventSource = new EventSource(
      `/api/osa/module-instances/generate/${queueItemId}/stream`,
    );

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);

        // Reset reconnection counter on successful message
        reconnectionAttempts.set(queueItemId, 0);

        // Update app in store by matching queue item or app id
        update((state) => ({
          ...state,
          apps: state.apps.map((app) =>
            app.id === queueItemId
              ? {
                  ...app,
                  progress: data.progress || app.progress,
                  build_phase: data.phase || app.build_phase,
                  status_message: data.message || app.status_message,
                  status: data.status || app.status,
                }
              : app,
          ),
        }));

        // If complete or failed, close connection
        if (data.status === "generated" || data.status === "failed") {
          eventSource.close();
          sseConnections.delete(queueItemId);
          reconnectionAttempts.delete(queueItemId);
        }
      } catch (err) {
        console.error("Error parsing SSE message:", err);
      }
    };

    eventSource.onerror = (error) => {
      console.error("SSE connection error:", error);
      eventSource.close();
      sseConnections.delete(queueItemId);

      // Attempt reconnection with exponential backoff
      const attempts = reconnectionAttempts.get(queueItemId) || 0;
      if (attempts < MAX_RECONNECT_ATTEMPTS) {
        const delay = RECONNECT_DELAY * Math.pow(2, attempts);
        if (import.meta.env.DEV) {
          console.log(
            `Reconnecting to SSE for queue item ${queueItemId} in ${delay}ms (attempt ${attempts + 1}/${MAX_RECONNECT_ATTEMPTS})`,
          );
        }
        setTimeout(() => {
          reconnectionAttempts.set(queueItemId, attempts + 1);
          subscribeToAppProgress(queueItemId); // Reconnect
        }, delay);
      } else {
        console.error(
          `Max reconnection attempts (${MAX_RECONNECT_ATTEMPTS}) reached for queue item ${queueItemId}`,
        );
        reconnectionAttempts.delete(queueItemId);
        // Update app with error status
        update((state) => ({
          ...state,
          apps: state.apps.map((app) =>
            app.id === queueItemId
              ? {
                  ...app,
                  status: "failed" as AppStatus,
                  error_message: "Connection lost. Please refresh.",
                }
              : app,
          ),
        }));
      }
    };

    sseConnections.set(queueItemId, eventSource);
  }

  function unsubscribeFromAppProgress(queueItemId: string): void {
    const eventSource = sseConnections.get(queueItemId);
    if (eventSource) {
      eventSource.close();
      sseConnections.delete(queueItemId);
    }
  }

  function setFilter(filter: AppStatus | "all"): void {
    update((state) => ({ ...state, filter }));
  }

  function setSearchQuery(query: string): void {
    update((state) => ({ ...state, searchQuery: query }));
  }

  function startGeneration(appId: string): void {
    update((state) => {
      const newGenerations = new Map(state.activeGenerations);
      newGenerations.set(appId, {
        app_id: appId,
        current_phase: "planning",
        agent_statuses: new Map(),
        timeline: [],
        logs: [],
        started_at: new Date().toISOString(),
      });
      return { ...state, activeGenerations: newGenerations };
    });
  }

  function updateAgentStatus(
    appId: string,
    agentType: AgentType,
    agentState: Partial<AgentState>,
  ): void {
    update((state) => {
      const generation = state.activeGenerations.get(appId);
      if (!generation) return state;

      const existing = generation.agent_statuses.get(agentType) || {
        type: agentType,
        status: "pending" as AgentStatus,
        progress: 0,
        message: "",
      };

      generation.agent_statuses.set(agentType, { ...existing, ...agentState });
      const newGenerations = new Map(state.activeGenerations);
      newGenerations.set(appId, generation);
      return { ...state, activeGenerations: newGenerations };
    });
  }

  function updateGenerationPhase(appId: string, phase: BuildPhase): void {
    update((state) => {
      const generation = state.activeGenerations.get(appId);
      if (!generation) return state;

      generation.current_phase = phase;
      generation.timeline.push({
        id: crypto.randomUUID(),
        phase,
        event: `Started ${phase} phase`,
        timestamp: new Date().toISOString(),
      });

      const newGenerations = new Map(state.activeGenerations);
      newGenerations.set(appId, generation);
      return { ...state, activeGenerations: newGenerations };
    });
  }

  function addGenerationLog(
    appId: string,
    log: Omit<GenerationLog, "id" | "timestamp">,
  ): void {
    update((state) => {
      const generation = state.activeGenerations.get(appId);
      if (!generation) return state;

      generation.logs.push({
        ...log,
        id: crypto.randomUUID(),
        timestamp: new Date().toISOString(),
      });

      const newGenerations = new Map(state.activeGenerations);
      newGenerations.set(appId, generation);
      return { ...state, activeGenerations: newGenerations };
    });
  }

  function completeGeneration(appId: string): void {
    update((state) => {
      const generation = state.activeGenerations.get(appId);
      if (!generation) return state;

      generation.completed_at = new Date().toISOString();
      const newGenerations = new Map(state.activeGenerations);
      newGenerations.set(appId, generation);
      return { ...state, activeGenerations: newGenerations };
    });
  }

  function clearGeneration(appId: string): void {
    update((state) => {
      const newGenerations = new Map(state.activeGenerations);
      newGenerations.delete(appId);
      return { ...state, activeGenerations: newGenerations };
    });
  }

  function getGenerationState(appId: string): GenerationState | undefined {
    let result: GenerationState | undefined;
    subscribe((state) => {
      result = state.activeGenerations.get(appId);
    })();
    return result;
  }

  // Sandbox lifecycle methods (aligned with backend cd8e4b49)
  async function startSandbox(appId: string): Promise<void> {
    if (!browser) return;
    try {
      await restartSandboxApi(appId);
      await fetchApps();
    } catch (err) {
      throw new Error(
        err instanceof Error ? err.message : "Failed to start sandbox",
      );
    }
  }

  async function stopSandbox(appId: string): Promise<void> {
    if (!browser) return;
    try {
      await stopSandboxApi(appId);
      await fetchApps();
    } catch (err) {
      throw new Error(
        err instanceof Error ? err.message : "Failed to stop sandbox",
      );
    }
  }

  async function removeSandbox(appId: string): Promise<void> {
    if (!browser) return;
    try {
      await removeSandboxApi(appId);
      // Also remove sandbox info from local state
      update((state) => ({
        ...state,
        apps: state.apps.map((app) =>
          app.id === appId ? { ...app, sandbox: undefined } : app,
        ),
      }));
      await fetchApps();
    } catch (err) {
      throw new Error(
        err instanceof Error ? err.message : "Failed to remove sandbox",
      );
    }
  }

  if (browser) {
    window.addEventListener("beforeunload", () => {
      sseConnections.forEach((eventSource) => eventSource.close());
      sseConnections.clear();
    });
  }

  return {
    subscribe,
    filteredApps: { subscribe: filteredApps.subscribe },
    fetchApps,
    goToPage,
    setPageSize,
    getAppById,
    deployApp,
    deleteApp,
    updateApp,
    startSandbox,
    stopSandbox,
    removeSandbox,
    subscribeToAppProgress,
    unsubscribeFromAppProgress,
    setFilter,
    setSearchQuery,
    startGeneration,
    updateAgentStatus,
    updateGenerationPhase,
    addGenerationLog,
    completeGeneration,
    clearGeneration,
    getGenerationState,
    reset: () => set(createInitialState()),
  };
}

export const generatedAppsStore = createGeneratedAppsStore();
