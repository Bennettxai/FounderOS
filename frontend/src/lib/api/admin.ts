// Platform admin API client (superadmin only).
//
// Wraps /api/v1/admin/* - the platform-owner control plane. Every user, every
// workspace, every organization across the whole platform. Gated server-side by
// platform_role = 'superadmin'; a non-superadmin gets 403 from the backend.

import { request, requestPaginated } from "./base";
import type { PaginatedResponse } from "./base";

// ── Types (mirror the Go structs in platform_admin.go) ──────────────────────

export interface AdminDashboard {
  users: { total: number; online: number; new_today: number };
  workspaces: { total: number; active: number };
  computers: { total: number; running: number; hibernated: number };
  revenue: {
    mrr_cents: number | null;
    active_subscriptions: number;
    status: string;
  };
  system: {
    miosa_status: string;
    compute_utilization: number | null;
    status: string;
  };
}

export interface AdminUser {
  id: string;
  name: string;
  email: string;
  platform_role: string;
  created_at: string;
  workspace_id: string | null;
  workspace_name: string | null;
  plan: string | null;
}

export interface AdminWorkspace {
  id: string;
  name: string;
  slug: string;
  plan_type: string | null;
  created_at: string;
  owner_email: string | null;
  owner_name: string | null;
  member_count: number;
}

export interface AdminWorkspaceMember {
  user_id: string;
  email: string;
  name: string;
  role: string | null;
  status: string | null;
  joined_at: string;
}

export interface AdminWorkspaceDetail extends AdminWorkspace {
  members: AdminWorkspaceMember[];
}

export interface AdminOrg {
  id: string;
  name: string;
  slug: string;
  plan: string | null;
  owner_email: string | null;
  owner_name: string | null;
  member_count: number;
  workspace_count: number;
  created_at: string;
}

export interface AdminOrgMember {
  user_id: string;
  email: string;
  name: string;
  role: string;
  status: string | null;
  joined_at: string;
}

export interface AdminOrgWorkspace {
  id: string;
  name: string;
  slug: string;
  member_count: number;
}

export interface AdminOrgDetail extends AdminOrg {
  members: AdminOrgMember[];
  workspaces: AdminOrgWorkspace[];
}

export interface AdminSettings {
  allowed_origins: string[];
  environment: string;
  deployment_mode: string;
  ai_provider: string;
  default_model: string;
  osa_enabled: boolean;
  carrier_enabled: boolean;
  nats_enabled: boolean;
}

export interface AdminMiosaTenantStatus {
  configured: boolean;
  key_prefix?: string;
  capacity_provider: "local" | "businessos" | "user";
  tenant?: {
    id: string;
    name: string;
    plan: string;
    status: string;
    limits?: Record<string, number>;
    usage?: Record<string, number>;
  };
  credits?: {
    balance: number;
    expires_at?: string;
  };
  credit_usage?: {
    period_start: string;
    period_end: string;
    compute_credits: number;
    ai_credits: number;
    total_credits: number;
  };
  usage_rollup: Array<{
    external_user_id?: string;
    external_project_id?: string;
    workspace_id?: string;
    sandbox_seconds: number;
    computer_seconds: number;
    storage_gb_hours: number;
    credit_cents: number;
  }>;
  recent_sandboxes: Array<{
    id: string;
    workspace_id: string;
    workspace_name: string;
    user_email: string;
    miosa_sandbox_id: string;
    miosa_workspace_id?: string;
    external_workspace_id: string;
    external_user_id: string;
    terminal_session_id?: string;
    status: string;
    preview_url?: string;
    created_at: string;
  }>;
  external_workspaces: Array<{
    id: string;
    name: string;
    slug: string;
    organization_id?: string;
    organization_name?: string;
    plan_type: string;
    owner_email?: string;
    member_count: number;
    miosa_workspace_id?: string;
    external_workspace_id?: string;
    external_user_id?: string;
    miosa_status?: string;
    sandbox_enabled: boolean;
    computer_enabled: boolean;
    desktop_enabled: boolean;
  }>;
  error?: string;
}

export interface AdminMiosaQuotaInput {
  max_sandboxes?: number | null;
  max_concurrent?: number | null;
  max_storage_gb?: number | null;
  max_credit_cents?: number | null;
}

export interface AdminMiosaEntitlementInput {
  sandbox_enabled?: boolean;
  computer_enabled?: boolean;
  desktop_enabled?: boolean;
}

export interface AdminMiosaWorkspaceLink {
  workspace_id: string;
  miosa_workspace_id: string;
  external_workspace_id: string;
  external_user_id: string;
  status: string;
  last_synced_at?: string;
}

export interface AdminEngineStatus {
  id: string;
  name: string;
  slug: string;
  owner_email: string | null;
  owner_name: string | null;
  member_count: number;
  engine_configured: boolean;
  engine_enabled: boolean;
  engine_host: string | null;
}

export interface AdminConnection {
  id: string;
  name: string;
  email: string;
  google_email: string | null;
  google_connected: boolean;
  slack_connected: boolean;
  notion_connected: boolean;
  other_connections: number;
}

// ── Reads ───────────────────────────────────────────────────────────────────

// Walk every page of a paginated admin list. The list endpoints cap page_size
// server-side, so satisfying "see ALL" (not just the first 100 rows) requires
// following has_more until the list is exhausted. A hard page cap guards against
// any accidental unbounded loop if the backend ever reports has_more incorrectly.
async function fetchAllPages<T>(
  fetchPage: (page: number, pageSize: number) => Promise<PaginatedResponse<T>>,
  pageSize = 200,
): Promise<{ data: T[]; total: number }> {
  let page = 1;
  let all: T[] = [];
  let total = 0;
  for (let i = 0; i < 500; i++) {
    const res = await fetchPage(page, pageSize);
    all = all.concat(res.data ?? []);
    total = res.pagination?.total_items ?? all.length;
    if (!res.pagination?.has_more) break;
    page += 1;
  }
  return { data: all, total };
}

export const adminApi = {
  dashboard: () =>
    request<AdminDashboard>("/admin/dashboard", { skipCache: true }),

  listUsers: (page = 1, pageSize = 100) =>
    requestPaginated<AdminUser>("/admin/users", { page, page_size: pageSize }),

  listWorkspaces: (page = 1, pageSize = 100) =>
    requestPaginated<AdminWorkspace>("/admin/workspaces", {
      page,
      page_size: pageSize,
    }),

  getWorkspace: (id: string) =>
    request<AdminWorkspaceDetail>(`/admin/workspaces/${id}`, {
      skipCache: true,
    }),

  listOrganizations: (page = 1, pageSize = 100) =>
    requestPaginated<AdminOrg>("/admin/organizations", {
      page,
      page_size: pageSize,
    }),

  getOrganization: (id: string) =>
    request<AdminOrgDetail>(`/admin/organizations/${id}`, { skipCache: true }),

  settings: () =>
    request<AdminSettings>("/admin/settings", { skipCache: true }),

  miosaTenant: () =>
    request<AdminMiosaTenantStatus>("/admin/miosa-tenant", {
      skipCache: true,
    }),

  saveMiosaTenantKey: (apiKey: string) =>
    request<{ status: string; key_prefix: string }>("/admin/miosa-tenant/key", {
      method: "POST",
      body: { api_key: apiKey },
    }),

  ensureMiosaExternalWorkspace: (externalWorkspaceId: string) =>
    request<AdminMiosaWorkspaceLink>(
      `/admin/miosa-tenant/external-workspaces/${externalWorkspaceId}/link`,
      { method: "POST" },
    ),

  setMiosaExternalWorkspaceEntitlement: (
    externalWorkspaceId: string,
    entitlement: AdminMiosaEntitlementInput,
  ) =>
    request<Record<string, unknown>>(
      `/admin/miosa-tenant/external-workspaces/${externalWorkspaceId}/entitlement`,
      {
        method: "POST",
        body: entitlement,
      },
    ),

  setMiosaExternalWorkspaceQuota: (
    externalWorkspaceId: string,
    quota: AdminMiosaQuotaInput,
  ) =>
    request<Record<string, unknown>>(
      `/admin/miosa-tenant/external-workspaces/${externalWorkspaceId}/quota`,
      {
        method: "POST",
        body: quota,
      },
    ),

  listEngineStatus: (page = 1, pageSize = 100) =>
    requestPaginated<AdminEngineStatus>("/admin/engine-status", {
      page,
      page_size: pageSize,
    }),

  // ── Read-all (paginate to completion) ────────────────────────────────────────
  // Use these where the UI must reflect the entire platform, not just page one.

  listAllUsers: () =>
    fetchAllPages<AdminUser>((page, page_size) =>
      requestPaginated<AdminUser>("/admin/users", { page, page_size }),
    ),

  listAllWorkspaces: () =>
    fetchAllPages<AdminWorkspace>((page, page_size) =>
      requestPaginated<AdminWorkspace>("/admin/workspaces", {
        page,
        page_size,
      }),
    ),

  listAllOrganizations: () =>
    fetchAllPages<AdminOrg>((page, page_size) =>
      requestPaginated<AdminOrg>("/admin/organizations", { page, page_size }),
    ),

  listAllEngineStatus: () =>
    fetchAllPages<AdminEngineStatus>((page, page_size) =>
      requestPaginated<AdminEngineStatus>("/admin/engine-status", {
        page,
        page_size,
      }),
    ),

  listAllConnections: () =>
    fetchAllPages<AdminConnection>((page, page_size) =>
      requestPaginated<AdminConnection>("/admin/connections", {
        page,
        page_size,
      }),
    ),

  // ── Writes ─────────────────────────────────────────────────────────────────

  sendEngineReminder: (workspaceId: string) =>
    request<{ status: string }>(
      `/admin/workspaces/${workspaceId}/engine-reminder`,
      { method: "POST" },
    ),

  setUserRole: (userId: string, role: string) =>
    request<{ status: string }>(`/admin/users/${userId}/role`, {
      method: "POST",
      body: { role },
    }),

  setUserPlan: (userId: string, plan: string) =>
    request<{ status: string }>(`/admin/users/${userId}/plan`, {
      method: "POST",
      body: { plan },
    }),
};

export type { PaginatedResponse };
