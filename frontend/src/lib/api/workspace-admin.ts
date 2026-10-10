// Workspace administration API client.
//
// Wraps the /api/v1/workspaces/* admin endpoints used by the Settings module:
// workspace details, members, invites, roles, and the per-workspace Optimal
// Engine connection. Every call goes through `request<T>()` which handles the
// API base URL, the session cookie, CSRF, and the X-Workspace-ID header.

import { request } from "./base";

// ── Types (mirror the Go service structs) ───────────────────────────────────

export interface Workspace {
  id: string;
  name: string;
  slug: string;
  description?: string | null;
  logo_url?: string | null;
  plan_type: string;
  max_members?: number;
  max_projects?: number;
  settings?: Record<string, unknown>;
  owner_id: string;
  created_at: string;
  updated_at: string;
}

export interface WorkspaceMember {
  id: string;
  workspace_id: string;
  user_id: string;
  role: string;
  status: string;
  joined_at?: string | null;
  invited_at?: string | null;
  // Enriched from the user table.
  name?: string | null;
  email?: string | null;
  image?: string | null;
}

export interface WorkspaceRole {
  id: string;
  workspace_id: string;
  name: string;
  display_name: string;
  description?: string | null;
  color?: string | null;
  icon?: string | null;
  permissions: Record<string, unknown>;
  is_system: boolean;
  is_default: boolean;
  hierarchy_level: number;
}

export interface WorkspaceInvite {
  id: string;
  workspace_id: string;
  email: string;
  role: string;
  invited_by: string;
  token: string;
  status: string; // pending | accepted | expired | revoked
  expires_at: string;
  accepted_at?: string | null;
  created_at: string;
}

export interface EngineConfig {
  enabled: boolean;
  base_url: string;
  api_key?: string;
  has_api_key?: boolean;
  workspace: string; // engine-side workspace slug
  updated_at?: string;
  updated_by?: string;
}

export interface EngineTestResult {
  reachable: boolean;
  status?: number;
  message: string;
  base_url?: string;
}

export type WorkspaceRoleName =
  "owner" | "admin" | "manager" | "member" | "viewer" | "guest";

// ── Workspaces ──────────────────────────────────────────────────────────────

export async function listWorkspaces(): Promise<Workspace[]> {
  const res = await request<{ workspaces: Workspace[] }>("/workspaces", {
    skipCache: true,
  });
  return res.workspaces ?? [];
}

export function getWorkspace(id: string): Promise<Workspace> {
  return request<Workspace>(`/workspaces/${id}`, { skipCache: true });
}

export function updateWorkspace(
  id: string,
  patch: { name?: string; description?: string; logo_url?: string },
): Promise<Workspace> {
  return request<Workspace>(`/workspaces/${id}`, {
    method: "PUT",
    body: patch,
  });
}

// ── Members ─────────────────────────────────────────────────────────────────

export async function listMembers(
  workspaceId: string,
): Promise<WorkspaceMember[]> {
  const res = await request<{ members: WorkspaceMember[] }>(
    `/workspaces/${workspaceId}/members`,
    { skipCache: true },
  );
  return res.members ?? [];
}

export function updateMemberRole(
  workspaceId: string,
  userId: string,
  role: WorkspaceRoleName,
): Promise<WorkspaceMember> {
  return request<WorkspaceMember>(
    `/workspaces/${workspaceId}/members/${userId}`,
    { method: "PUT", body: { role } },
  );
}

export function removeMember(
  workspaceId: string,
  userId: string,
): Promise<{ message: string }> {
  return request<{ message: string }>(
    `/workspaces/${workspaceId}/members/${userId}`,
    { method: "DELETE" },
  );
}

// ── Invites ─────────────────────────────────────────────────────────────────

export async function listInvites(
  workspaceId: string,
): Promise<WorkspaceInvite[]> {
  const res = await request<{ invites: WorkspaceInvite[] }>(
    `/workspaces/${workspaceId}/invites`,
    { skipCache: true },
  );
  return res.invites ?? [];
}

export function createInvite(
  workspaceId: string,
  email: string,
  role: WorkspaceRoleName,
): Promise<WorkspaceInvite> {
  return request<WorkspaceInvite>(`/workspaces/${workspaceId}/invites`, {
    method: "POST",
    body: { email, role },
  });
}

export function revokeInvite(
  workspaceId: string,
  inviteId: string,
): Promise<{ message: string }> {
  return request<{ message: string }>(
    `/workspaces/${workspaceId}/invites/${inviteId}`,
    { method: "DELETE" },
  );
}

// ── Roles ───────────────────────────────────────────────────────────────────

export async function listRoles(workspaceId: string): Promise<WorkspaceRole[]> {
  const res = await request<{ roles: WorkspaceRole[] }>(
    `/workspaces/${workspaceId}/roles`,
    { skipCache: true },
  );
  return res.roles ?? [];
}

// ── Optimal Engine connection ───────────────────────────────────────────────

export async function getEngineConfig(
  workspaceId: string,
): Promise<EngineConfig> {
  const res = await request<{ engine: EngineConfig }>(
    `/workspaces/${workspaceId}/engine`,
    { skipCache: true },
  );
  return res.engine;
}

export async function updateEngineConfig(
  workspaceId: string,
  cfg: Pick<EngineConfig, "enabled" | "base_url" | "api_key" | "workspace">,
): Promise<EngineConfig> {
  const res = await request<{ engine: EngineConfig }>(
    `/workspaces/${workspaceId}/engine`,
    { method: "PUT", body: cfg },
  );
  return res.engine;
}

export function testEngineConnection(
  workspaceId: string,
  cfg?: { base_url?: string; api_key?: string },
): Promise<EngineTestResult> {
  return request<EngineTestResult>(`/workspaces/${workspaceId}/engine/test`, {
    method: "POST",
    body: cfg ?? {},
    timeout: 12_000,
  });
}

// ── Profile ─────────────────────────────────────────────────────────────────

export function updateProfileName(name: string): Promise<unknown> {
  return request("/profile", { method: "PUT", body: { name } });
}

// ── Cloud sync ──────────────────────────────────────────────────────────────

export interface CloudSyncReport {
  workspace: string;
  members: number;
  invites: number;
  tables: Record<string, number>;
  skipped_users: string[];
}

// Owner/admin: push the active workspace + members + invites + scoped data to
// the shared cloud DB so teammates can work in it live (real invite links).
export function syncWorkspaceToCloud(): Promise<CloudSyncReport> {
  return request<CloudSyncReport>(`/workspace/sync-to-cloud`, {
    method: "POST",
    timeout: 30_000,
  });
}

// ── Sync policies (local-first; sync a whole module or not) ──────────────────

export type SyncMode = "local" | "workspace";

export interface SyncPolicy {
  module: string;
  label: string;
  sync_mode: SyncMode;
}

export interface SyncPolicies {
  policies: SyncPolicy[];
  auto_sync_all: boolean;
}

export async function getSyncPolicies(
  workspaceId: string,
): Promise<SyncPolicies> {
  return request<SyncPolicies>(`/workspaces/${workspaceId}/sync-policies`, {
    skipCache: true,
  });
}

// Flip a single module between local-only and synced-to-workspace.
export function setSyncPolicy(
  workspaceId: string,
  module: string,
  syncMode: SyncMode,
): Promise<unknown> {
  return request(`/workspaces/${workspaceId}/sync-policies`, {
    method: "PUT",
    body: { module, sync_mode: syncMode },
  });
}

// Flip every module at once ("auto-sync everything" master switch).
export function setAllSync(
  workspaceId: string,
  mode: SyncMode,
): Promise<unknown> {
  return request(`/workspaces/${workspaceId}/sync-policies`, {
    method: "PUT",
    body: { all: mode },
  });
}
