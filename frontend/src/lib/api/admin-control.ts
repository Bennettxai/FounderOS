// Platform admin control API — write operations for the superadmin control plane.
//
// Exposes renameOrganization, deleteOrganization, workspace edits, member management,
// and user suspend/unsuspend. All routes live under /admin (gated by platform_role =
// 'superadmin' server-side; non-superadmin receives 403).

import { request } from "./base";

// ── Response shapes ──────────────────────────────────────────────────────────

export interface StatusResponse {
  status: string;
}

// ── adminControl ─────────────────────────────────────────────────────────────

export const adminControl = {
  // Organizations

  renameOrganization: (id: string, name: string) =>
    request<StatusResponse>(`/admin/organizations/${id}`, {
      method: "PUT",
      body: { name },
    }),

  deleteOrganization: (id: string) =>
    request<StatusResponse>(`/admin/organizations/${id}`, {
      method: "DELETE",
    }),

  // Workspaces

  updateWorkspace: (
    id: string,
    fields: { name?: string; plan_type?: string },
  ) =>
    request<StatusResponse>(`/admin/workspaces/${id}`, {
      method: "PUT",
      body: fields,
    }),

  deleteWorkspace: (id: string) =>
    request<StatusResponse>(`/admin/workspaces/${id}`, {
      method: "DELETE",
    }),

  // Workspace members

  addWorkspaceMember: (id: string, email: string, role: string) =>
    request<StatusResponse>(`/admin/workspaces/${id}/members`, {
      method: "POST",
      body: { email, role },
    }),

  setWorkspaceMemberRole: (id: string, userId: string, role: string) =>
    request<StatusResponse>(`/admin/workspaces/${id}/members/${userId}`, {
      method: "PUT",
      body: { role },
    }),

  removeWorkspaceMember: (id: string, userId: string) =>
    request<StatusResponse>(`/admin/workspaces/${id}/members/${userId}`, {
      method: "DELETE",
    }),

  // Users

  suspendUser: (id: string) =>
    request<StatusResponse>(`/admin/users/${id}/suspend`, {
      method: "POST",
    }),

  unsuspendUser: (id: string) =>
    request<StatusResponse>(`/admin/users/${id}/unsuspend`, {
      method: "POST",
    }),
};
