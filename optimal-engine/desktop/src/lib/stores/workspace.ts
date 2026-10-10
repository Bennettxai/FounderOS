/**
 * Active organization + workspace context.
 *
 * Persisted in localStorage so refreshes keep your selection.
 * Loaded once on app start; mutated by the header dropdowns.
 */

import { writable, derived, get } from "svelte/store";
import {
  listOrganizations,
  listWorkspaces,
  type Organization,
  type Workspace,
} from "$lib/api";

const ORG_KEY = "oe-active-org";
const WS_KEY = "oe-active-workspace";

function readLS(key: string): string | null {
  if (typeof localStorage === "undefined") return null;
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function writeLS(key: string, value: string | null) {
  if (typeof localStorage === "undefined") return;
  try {
    if (value === null) localStorage.removeItem(key);
    else localStorage.setItem(key, value);
  } catch {
    /* ignore */
  }
}

export const organizations = writable<Organization[]>([]);
export const workspaces = writable<Workspace[]>([]);

// id of the active org (= tenant_id), nullable until first load resolves.
export const activeOrgId = writable<string | null>(readLS(ORG_KEY));

// id of the active workspace, nullable until first load.
export const activeWorkspaceId = writable<string | null>(readLS(WS_KEY));

// Derived convenience structs
export const activeOrg = derived(
  [organizations, activeOrgId],
  ([$orgs, $id]) => $orgs.find((o) => o.id === $id) ?? null,
);

export const activeWorkspace = derived(
  [workspaces, activeWorkspaceId],
  ([$ws, $id]) => $ws.find((w) => w.id === $id) ?? null,
);

// Persist changes back to localStorage
activeOrgId.subscribe((id) => writeLS(ORG_KEY, id));
activeWorkspaceId.subscribe((id) => writeLS(WS_KEY, id));

/**
 * Loads orgs + workspaces; selects defaults if nothing is persisted yet.
 * Idempotent — safe to call on every app boot.
 */
export async function bootstrap(): Promise<void> {
  try {
    const [{ organizations: orgs }, _] = [await listOrganizations(), null];
    organizations.set(orgs);

    let orgId = get(activeOrgId);
    if (!orgId || !orgs.find((o) => o.id === orgId)) {
      orgId = orgs[0]?.id ?? null;
      activeOrgId.set(orgId);
    }

    if (orgId) {
      const { workspaces: ws } = await listWorkspaces(orgId);
      workspaces.set(ws);

      let wsId = get(activeWorkspaceId);
      if (!wsId || !ws.find((w) => w.id === wsId)) {
        wsId = ws[0]?.id ?? null;
        activeWorkspaceId.set(wsId);
      }
    }
  } catch (e) {
    // Engine down — leave stores empty. Header will render nothing
    // and the engine-status chip will surface the failure.
    console.error("[workspace.bootstrap]", e);
  }
}

/**
 * Refresh the workspace list for the active org. Call after creating /
 * archiving a workspace to keep the dropdown in sync.
 */
export async function refreshWorkspaces(): Promise<void> {
  const orgId = get(activeOrgId);
  if (!orgId) return;
  const { workspaces: ws } = await listWorkspaces(orgId);
  workspaces.set(ws);
}

export function setActiveWorkspace(id: string) {
  activeWorkspaceId.set(id);
}

export function setActiveOrg(id: string) {
  activeOrgId.set(id);
  // Switching orgs invalidates the active workspace; bootstrap will reseed.
  activeWorkspaceId.set(null);
  workspaces.set([]);
  void bootstrap();
}
