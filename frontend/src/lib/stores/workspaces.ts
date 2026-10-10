import { writable, derived, get } from "svelte/store";
import type {
  Workspace,
  WorkspaceRole,
  WorkspaceMember,
  UserWorkspaceProfile,
  UserRoleContext,
} from "$lib/api/workspaces";
import {
  getWorkspaces,
  getWorkspace,
  createWorkspace as apiCreateWorkspace,
  getWorkspaceRoles,
  getWorkspaceMembers,
  getWorkspaceProfile,
  getUserRoleContext,
} from "$lib/api/workspaces";
import { fetchWorkspaces as fetchKnowledgeWorkspaces } from "$lib/kb/client";
import {
  detectLocalEngine,
  ensureEngineWorkspace,
} from "$lib/optimal-engine/connect";
import { updateEngineConfig } from "$lib/api/workspace-admin";

// ============================================================================
// STATE
// ============================================================================

/**
 * All workspaces the user has access to
 */
export const workspaces = writable<Workspace[]>([]);

/**
 * Currently selected workspace
 * This is the main state that drives workspace-aware features
 */
export const currentWorkspace = writable<Workspace | null>(null);

/**
 * Roles in the current workspace
 */
export const currentWorkspaceRoles = writable<WorkspaceRole[]>([]);

/**
 * Members in the current workspace
 */
export const currentWorkspaceMembers = writable<WorkspaceMember[]>([]);

/**
 * Current user's profile in the current workspace
 */
export const currentWorkspaceProfile = writable<UserWorkspaceProfile | null>(
  null,
);

/**
 * Current user's role context (permissions, etc)
 */
export const currentUserRoleContext = writable<UserRoleContext | null>(null);

/**
 * Loading states
 */
export const workspaceLoading = writable({
  workspaces: false,
  switching: false,
  roles: false,
  members: false,
  profile: false,
});

/**
 * Error state
 */
export const workspaceError = writable<string | null>(null);

// ============================================================================
// DERIVED STORES
// ============================================================================

/**
 * Get the current workspace ID (convenience)
 */
export const currentWorkspaceId = derived(
  currentWorkspace,
  ($currentWorkspace) => $currentWorkspace?.id ?? null,
);

/**
 * Check if user has a specific permission in current workspace
 */
export const hasPermission = derived(
  currentUserRoleContext,
  ($context) =>
    (resource: string, permission: string): boolean => {
      if (!$context) return false;
      return !!$context.permissions?.[resource]?.[permission];
    },
);

/**
 * Check if user is at least a certain hierarchy level
 */
export const isAtLeastLevel = derived(
  currentUserRoleContext,
  ($context) =>
    (level: number): boolean => {
      if (!$context) return false;
      return $context.hierarchy_level <= level; // Lower number = higher authority
    },
);

/**
 * Get current user's role name
 */
export const currentUserRole = derived(
  currentUserRoleContext,
  ($context) => $context?.role_name ?? null,
);

function workspaceFromKnowledgeWorkspace(workspace: {
  slug: string;
  name: string;
}): Workspace {
  const now = new Date().toISOString();

  return {
    id: `kb:${workspace.slug}`,
    name: workspace.name,
    slug: workspace.slug,
    description: "Knowledge workspace",
    logo_url: null,
    plan_type: "free",
    max_members: 1,
    max_projects: 0,
    max_storage_gb: 1,
    settings: { source: "knowledge" },
    owner_id: "local",
    created_at: now,
    updated_at: now,
  };
}

async function getAvailableWorkspaces(): Promise<Workspace[]> {
  try {
    const apiWorkspaces = (await getWorkspaces()) ?? [];

    // BusinessOS workspaces are the operating environments shown in the
    // switcher. Optimal Engine also contains internal knowledge scopes,
    // benchmarks, and system workspaces. Merging those lists made test fixtures
    // look like duplicate companies and allowed an engine scope to be promoted
    // accidentally just by clicking it. When the BusinessOS backend is healthy,
    // it is the canonical source for this navigation surface.
    return apiWorkspaces;
  } catch (error) {
    if (import.meta.env.DEV) {
      console.warn(
        "[Workspaces] Workspace API unavailable, falling back to knowledge workspaces",
        error,
      );
    }
  }

  // Preserve local-first recovery when the relational BusinessOS backend is
  // genuinely unavailable. These browse-only scopes are not mixed into a
  // healthy BusinessOS workspace list.
  const knowledgeWorkspaces = await fetchKnowledgeWorkspaces();
  return knowledgeWorkspaces.map(workspaceFromKnowledgeWorkspace);
}

function clearCurrentWorkspaceContext(): void {
  currentWorkspace.set(null);
  currentWorkspaceRoles.set([]);
  currentWorkspaceMembers.set([]);
  currentWorkspaceProfile.set(null);
  currentUserRoleContext.set(null);
}

function isKnowledgeWorkspace(workspace: Workspace | undefined): boolean {
  return workspace?.id.startsWith("kb:") === true;
}

function setLightweightWorkspaceContext(workspace: Workspace): void {
  currentWorkspace.set(workspace);
  currentWorkspaceRoles.set([]);
  currentWorkspaceMembers.set([]);
  currentWorkspaceProfile.set(null);
  currentUserRoleContext.set({
    user_id: "local",
    workspace_id: workspace.id,
    role_name: "Owner",
    role_display_name: "Owner",
    hierarchy_level: 0,
    permissions: {},
    title: null,
    department: null,
    expertise_areas: null,
  });
  localStorage.setItem("businessos_current_workspace_id", workspace.id);
}

// ============================================================================
// ACTIONS
// ============================================================================

/**
 * Initialize workspace state - load all workspaces
 */
export async function initializeWorkspaces(): Promise<void> {
  workspaceLoading.update((s) => ({ ...s, workspaces: true }));
  workspaceError.set(null);

  try {
    const allWorkspaces = await getAvailableWorkspaces();
    if (import.meta.env.DEV) {
      console.log(
        `[Workspaces] Loaded ${allWorkspaces.length} workspaces:`,
        allWorkspaces,
      );
    }
    workspaces.set(allWorkspaces);

    // If no workspace is selected and we have workspaces, restore the SAVED one
    // (the user's last selection) and only fall back to the first workspace when
    // there is no valid saved selection. Selecting allWorkspaces[0] unconditionally
    // (and writing it to localStorage via switchWorkspace) clobbered the saved
    // selection, so every refresh dumped the user on a "base" workspace.
    const current = get(currentWorkspace);
    if (!current && allWorkspaces.length > 0) {
      const savedId = localStorage.getItem("businessos_current_workspace_id");
      const savedWorkspace = allWorkspaces.find((w) => w.id === savedId);
      const target =
        savedId && savedWorkspace
          ? savedWorkspace.id
          : defaultWorkspaceId(allWorkspaces);
      if (import.meta.env.DEV) {
        console.log(
          `[Workspaces] Selecting workspace on init: ${target} (saved=${savedId ?? "none"})`,
        );
      }
      await switchWorkspace(target);
    } else if (current) {
      if (!allWorkspaces.some((workspace) => workspace.id === current.id)) {
        const savedId = localStorage.getItem("businessos_current_workspace_id");
        if (savedId === current.id) {
          localStorage.removeItem("businessos_current_workspace_id");
        }
        clearCurrentWorkspaceContext();
        await switchWorkspace(allWorkspaces[0].id);
        return;
      }
      if (import.meta.env.DEV) {
        console.log(
          `[Workspaces] Current workspace already set: ${current.name} (${current.id})`,
        );
      }
    } else if (allWorkspaces.length === 0) {
      clearCurrentWorkspaceContext();
      if (import.meta.env.DEV) {
        console.debug("[Workspaces] No workspaces available");
      }
    }
  } catch (error) {
    console.error("[Workspaces] Failed to load workspaces:", error);
    workspaceError.set(
      error instanceof Error ? error.message : "Failed to load workspaces",
    );

    workspaces.set([]);
    clearCurrentWorkspaceContext();
  } finally {
    workspaceLoading.update((s) => ({ ...s, workspaces: false }));
  }
}

/**
 * Switch to a different workspace
 * This loads all workspace-specific data
 */
/**
 * Switch to a workspace, retrying transient failures (rate limit, server
 * error, timeout) after a short jittered pause: a 3D desktop boots dozens of
 * embedded tiles that restore the workspace at the same moment.
 */
export async function switchWorkspace(workspaceId: string): Promise<void> {
  for (let attempt = 0; ; attempt++) {
    await switchWorkspaceOnce(workspaceId);
    if (get(currentWorkspace)?.id === workspaceId) return;
    const err = get(workspaceError);
    if (attempt >= 3 || !err || !isTransientError(new Error(err))) return;
    await new Promise((r) => setTimeout(r, 120 * (attempt + 1) + Math.random() * 200));
  }
}

/** Rate limit, server error, timeout or network failure (lib/api/base.ts doFetch messages). */
function isTransientError(error: unknown): boolean {
  const msg = error instanceof Error ? error.message : String(error);
  return /\(HTTP (429|5\d\d)\)/.test(msg) || /timeout|Failed to fetch|NetworkError|Load failed/i.test(msg);
}

async function switchWorkspaceOnce(workspaceId: string): Promise<void> {
  workspaceLoading.update((s) => ({ ...s, switching: true }));
  workspaceError.set(null);

  try {
    const cachedWorkspace = get(workspaces).find((w) => w.id === workspaceId);
    if (!cachedWorkspace) {
      localStorage.removeItem("businessos_current_workspace_id");
      clearCurrentWorkspaceContext();
      throw new Error("Workspace is no longer available to this account");
    }

    if (cachedWorkspace && isKnowledgeWorkspace(cachedWorkspace)) {
      // Promote an engine (local-knowledge) workspace into a REAL BusinessOS
      // workspace the first time it's opened, so its modules (glossary, settings,
      // sync) work and its knowledge is scoped to it. Reuse an existing real
      // workspace with the same slug if one was already created. Once promoted,
      // the kb: duplicate is hidden (dedup by slug) and we operate on the real one.
      let real = get(workspaces).find(
        (w) => !isKnowledgeWorkspace(w) && w.slug === cachedWorkspace.slug,
      );
      if (!real) {
        try {
          real = await apiCreateWorkspace({
            name: cachedWorkspace.name,
            slug: cachedWorkspace.slug,
          });
        } catch {
          // Likely already exists (slug taken) — fall through to re-fetch.
        }
        const refreshed = await getAvailableWorkspaces();
        workspaces.set(refreshed);
        real =
          real ??
          refreshed.find(
            (w) => !isKnowledgeWorkspace(w) && w.slug === cachedWorkspace.slug,
          );
      }
      if (real && !isKnowledgeWorkspace(real)) {
        if (import.meta.env.DEV) {
          console.log(
            `[Workspaces] Promoted engine workspace ${cachedWorkspace.slug} to real workspace ${real.id}`,
          );
        }
        await switchWorkspace(real.id);
        return;
      }
      // Couldn't promote — fall back to browse-only context so the app still works.
      setLightweightWorkspaceContext(cachedWorkspace);
      localStorage.setItem("businessos_current_workspace_id", workspaceId);
      return;
    }

    // Load workspace details
    const workspace = await getWorkspace(workspaceId);
    // Persist the request scope before publishing the workspace to subscribers.
    // Workspace-aware pages react immediately to currentWorkspace changes and
    // the shared API client reads this value when building X-Workspace-ID.
    localStorage.setItem("businessos_current_workspace_id", workspaceId);
    currentWorkspace.set(workspace);

    // Load workspace-specific data in parallel
    const [roles, members, profile, roleContext] = await Promise.all([
      getWorkspaceRoles(workspaceId),
      getWorkspaceMembers(workspaceId),
      getWorkspaceProfile(workspaceId).catch(() => null), // Profile might not exist yet
      getUserRoleContext(workspaceId),
    ]);

    currentWorkspaceRoles.set(roles);
    currentWorkspaceMembers.set(members);
    currentWorkspaceProfile.set(profile);
    currentUserRoleContext.set(roleContext);

    if (import.meta.env.DEV) {
      console.log(
        `[Workspaces] Switched to workspace: ${workspace.name} (${workspace.slug})`,
      );
    }
    if (import.meta.env.DEV) {
      console.log(
        `[Workspaces] User role: ${roleContext.role_display_name} (Level ${roleContext.hierarchy_level})`,
      );
    }
  } catch (error) {
    console.error("[Workspaces] Failed to switch workspace:", error);
    workspaceError.set(
      error instanceof Error ? error.message : "Failed to switch workspace",
    );
    // A transient failure (rate limit, server error, timeout, offline) says
    // nothing about access: keep the saved choice. Many desktop tiles restore
    // the workspace at once and some get 429; forgetting it there wiped the
    // selection for every window.
    if (!isTransientError(error)) {
      if (localStorage.getItem("businessos_current_workspace_id") === workspaceId) {
        localStorage.removeItem("businessos_current_workspace_id");
      }
      if (get(currentWorkspace)?.id === workspaceId) {
        clearCurrentWorkspaceContext();
      }
    }
    // Do not re-throw — callers (layout, loadSavedWorkspace) should not blank the app
    // on a stale or invalid workspace ID. The error store carries the failure signal.
  } finally {
    workspaceLoading.update((s) => ({ ...s, switching: false }));
  }
}

/**
 * Refresh current workspace data
 */
export async function refreshCurrentWorkspace(): Promise<void> {
  const current = get(currentWorkspace);
  if (!current) return;
  await switchWorkspace(current.id);
}

/**
 * Create a new workspace, refresh the list, and switch to it.
 * Persists the new workspace ID to localStorage via switchWorkspace.
 */
export async function createWorkspace(data: {
  name: string;
  slug?: string;
  organization_id?: string;
}): Promise<Workspace> {
  workspaceLoading.update((s) => ({ ...s, switching: true }));
  workspaceError.set(null);

  try {
    const created = await apiCreateWorkspace(data);

    // Desktop/local-first onboarding: bind the new BusinessOS workspace to a
    // same-slug workspace in the running Optimal Engine. This is best effort so
    // cloud/web workspace creation still succeeds when no local engine exists.
    try {
      const baseUrl = await detectLocalEngine();
      if (baseUrl) {
        await ensureEngineWorkspace(baseUrl, undefined, {
          slug: created.slug,
          name: created.name,
          description: created.description ?? undefined,
        });
        await updateEngineConfig(created.id, {
          enabled: true,
          base_url: baseUrl,
          api_key: "",
          workspace: created.slug,
        });
        const el = (
          globalThis as unknown as {
            electron?: {
              engine?: {
                setConfig?: (workspaceId: string, config: unknown) => Promise<unknown>;
              };
            };
          }
        ).electron;
        await el?.engine?.setConfig?.(created.id, {
          enabled: true,
          base_url: baseUrl,
          api_key: "",
          workspace: created.slug,
        });
      }
    } catch (engineError) {
      console.warn(
        "[Workspaces] Workspace created, but local engine provisioning failed:",
        engineError,
      );
    }

    // Refresh the full list so the new workspace appears in the switcher.
    const allWorkspaces = await getAvailableWorkspaces();
    workspaces.set(allWorkspaces);

    // Switch to the newly created workspace (also writes localStorage).
    await switchWorkspace(created.id);

    if (import.meta.env.DEV) {
      console.log(
        `[Workspaces] Created and switched to: ${created.name} (${created.id})`,
      );
    }

    return created;
  } catch (error) {
    console.error("[Workspaces] Failed to create workspace:", error);
    workspaceError.set(
      error instanceof Error ? error.message : "Failed to create workspace",
    );
    throw error; // Re-throw so the UI form can display the error.
  } finally {
    workspaceLoading.update((s) => ({ ...s, switching: false }));
  }
}

/**
 * Load saved workspace from localStorage.
 * - Always loads the workspace list first (if not already loaded).
 * - If a saved ID exists, attempts to switch to it; on failure falls back to
 *   the first available workspace without throwing.
 * - If no workspaces exist at all, sets currentWorkspace=null gracefully.
 */
/** With no saved choice, open FounderOS's HQ (the command-center workspace,
 *  slug "founderos") when the account has one, else the first in API order. */
function defaultWorkspaceId(all: Workspace[]): string {
  return (all.find((w) => w.slug === "founderos") ?? all[0]).id;
}

export async function loadSavedWorkspace(): Promise<void> {
  // Load the list if it hasn't been populated yet.
  if (get(workspaces).length === 0) {
    await initializeWorkspaces();
  }

  const savedId = localStorage.getItem("businessos_current_workspace_id");
  const allWorkspaces = get(workspaces);

  if (allWorkspaces.length === 0) {
    // No workspaces available — set clean null state, do not throw.
    clearCurrentWorkspaceContext();
    if (import.meta.env.DEV) {
      console.debug("[Workspaces] loadSavedWorkspace: no workspaces available");
    }
    return;
  }

  if (savedId) {
    const exists = allWorkspaces.some((w) => w.id === savedId);
    if (exists) {
      // switchWorkspace no longer throws — error surfaces via workspaceError store.
      await switchWorkspace(savedId);
      if (get(currentWorkspace)?.id === savedId) return;
      // Still failing transiently: keep the saved choice rather than quietly
      // moving the user to another workspace.
      const lastError = get(workspaceError);
      if (lastError && isTransientError(new Error(lastError))) return;
      const fallback = allWorkspaces.find((w) => w.id !== savedId);
      if (fallback) {
        await switchWorkspace(fallback.id);
        return;
      }
    }
    // Saved ID no longer valid (workspace deleted/access removed) — clear it.
    localStorage.removeItem("businessos_current_workspace_id");
    if (import.meta.env.DEV) {
      console.warn(
        `[Workspaces] Saved workspace ${savedId} not found in list, falling back to first`,
      );
    }
  }

  // Default to HQ when there is one, else the first workspace in the list.
  await switchWorkspace(defaultWorkspaceId(allWorkspaces));
}

/**
 * Clear workspace state (for logout, etc)
 */
export function clearWorkspaceState(): void {
  workspaces.set([]);
  currentWorkspace.set(null);
  currentWorkspaceRoles.set([]);
  currentWorkspaceMembers.set([]);
  currentWorkspaceProfile.set(null);
  currentUserRoleContext.set(null);
  workspaceError.set(null);
  localStorage.removeItem("businessos_current_workspace_id");
}

// Desktop (Electron) only: keep the local-first sync engine scoped to the active
// workspace, so shared (BusinessOSSync) records push/pull for the right team
// workspace. Fires immediately with the current value and on every change.
if (typeof window !== "undefined" && "electron" in window) {
  currentWorkspace.subscribe((ws) => {
    const electron = (
      window as unknown as {
        electron?: { sync?: { setWorkspace?: (id: string | null) => void } };
      }
    ).electron;
    electron?.sync?.setWorkspace?.(ws?.id ?? null);
  });
}
