import { request } from "../base";
import type {
  Workspace,
  WorkspaceRole,
  WorkspaceMember,
  UserWorkspaceProfile,
  CreateWorkspaceData,
  UpdateWorkspaceData,
  UserRoleContext,
  WorkspaceInvite,
  CreateInviteData,
  UpdateMemberRoleData,
  CreateRoleData,
  UpdateRoleData,
} from "./types";

/**
 * Get all workspaces for the current user
 */
export async function getWorkspaces(): Promise<Workspace[]> {
  const response = await request<{ workspaces: Workspace[] }>("/workspaces");
  return response?.workspaces ?? [];
}

/**
 * Get a specific workspace by ID
 */
export async function getWorkspace(id: string): Promise<Workspace> {
  return request<Workspace>(`/workspaces/${id}`);
}

/**
 * Create a new workspace
 */
export async function createWorkspace(
  data: CreateWorkspaceData,
): Promise<Workspace> {
  return request<Workspace>("/workspaces", {
    method: "POST",
    body: data,
  });
}

/**
 * Update a workspace
 */
export async function updateWorkspace(
  id: string,
  data: UpdateWorkspaceData,
): Promise<Workspace> {
  return request<Workspace>(`/workspaces/${id}`, {
    method: "PUT",
    body: data,
  });
}

/**
 * Delete a workspace
 */
export async function deleteWorkspace(id: string): Promise<void> {
  return request<void>(`/workspaces/${id}`, {
    method: "DELETE",
  });
}

/**
 * Get all members of a workspace
 */
export async function getWorkspaceMembers(
  workspaceId: string,
): Promise<WorkspaceMember[]> {
  const response = await request<WorkspaceMember[] | { members: WorkspaceMember[] }>(
    `/workspaces/${workspaceId}/members`,
  );
  return Array.isArray(response) ? response : response?.members ?? [];
}

/**
 * Get all roles in a workspace
 */
export async function getWorkspaceRoles(
  workspaceId: string,
): Promise<WorkspaceRole[]> {
  return request<WorkspaceRole[]>(`/workspaces/${workspaceId}/roles`);
}

/**
 * Get the current user's profile in a workspace
 */
export async function getWorkspaceProfile(
  workspaceId: string,
): Promise<UserWorkspaceProfile> {
  return request<UserWorkspaceProfile>(`/workspaces/${workspaceId}/profile`);
}

/**
 * Update the current user's profile in a workspace
 */
export async function updateWorkspaceProfile(
  workspaceId: string,
  data: Partial<UserWorkspaceProfile>,
): Promise<UserWorkspaceProfile> {
  return request<UserWorkspaceProfile>(`/workspaces/${workspaceId}/profile`, {
    method: "PUT",
    body: data,
  });
}

/**
 * Get the current user's role context in a workspace
 * This returns the full role information including permissions
 */
export async function getUserRoleContext(
  workspaceId: string,
): Promise<UserRoleContext> {
  return request<UserRoleContext>(`/workspaces/${workspaceId}/role-context`);
}

/**
 * Get all invitations for a workspace
 */
export async function getWorkspaceInvites(
  workspaceId: string,
): Promise<WorkspaceInvite[]> {
  return request<WorkspaceInvite[]>(`/workspaces/${workspaceId}/invites`);
}

/**
 * Create a new workspace invitation
 */
export async function createWorkspaceInvite(
  workspaceId: string,
  data: CreateInviteData,
): Promise<WorkspaceInvite> {
  return request<WorkspaceInvite>(`/workspaces/${workspaceId}/invites`, {
    method: "POST",
    body: data,
  });
}

/**
 * Revoke a workspace invitation
 */
export async function revokeWorkspaceInvite(
  workspaceId: string,
  inviteId: string,
): Promise<void> {
  return request<void>(`/workspaces/${workspaceId}/invites/${inviteId}`, {
    method: "DELETE",
  });
}

/**
 * Update a workspace member's role
 */
export async function updateWorkspaceMemberRole(
  workspaceId: string,
  memberId: string,
  data: UpdateMemberRoleData,
): Promise<WorkspaceMember> {
  return request<WorkspaceMember>(
    `/workspaces/${workspaceId}/members/${memberId}`,
    {
      method: "PUT",
      body: data,
    },
  );
}

/**
 * Remove a member from a workspace
 */
export async function removeWorkspaceMember(
  workspaceId: string,
  memberId: string,
): Promise<void> {
  return request<void>(`/workspaces/${workspaceId}/members/${memberId}`, {
    method: "DELETE",
  });
}

/**
 * Create a custom role in a workspace
 */
export async function createWorkspaceRole(
  workspaceId: string,
  data: CreateRoleData,
): Promise<WorkspaceRole> {
  return request<WorkspaceRole>(`/workspaces/${workspaceId}/roles`, {
    method: "POST",
    body: data,
  });
}

/**
 * Update a custom role in a workspace
 */
export async function updateWorkspaceRole(
  workspaceId: string,
  roleId: string,
  data: UpdateRoleData,
): Promise<WorkspaceRole> {
  return request<WorkspaceRole>(`/workspaces/${workspaceId}/roles/${roleId}`, {
    method: "PUT",
    body: data,
  });
}

/**
 * Delete a custom role from a workspace
 */
export async function deleteWorkspaceRole(
  workspaceId: string,
  roleId: string,
): Promise<void> {
  return request<void>(`/workspaces/${workspaceId}/roles/${roleId}`, {
    method: "DELETE",
  });
}

/**
 * Accept a workspace invitation by token
 * This is a public endpoint - user must be logged in but doesn't need workspace access
 */
export async function acceptWorkspaceInvite(
  token: string,
): Promise<{ message: string; workspace_id?: string; role?: string }> {
  return request<{ message: string; workspace_id?: string; role?: string }>(
    "/workspaces/invites/accept",
    {
      method: "POST",
      body: { token },
    },
  );
}
