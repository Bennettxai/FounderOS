// Organization API client — the account level (Organization → Workspace → Team).
// Org membership is account-wide; workspace access stays granular.
import { request } from "./base";

export interface Organization {
  id: string;
  name: string;
  slug: string;
  logo_url?: string;
  role?: string;
  member_count?: number;
  workspace_count?: number;
}

export interface OrgMember {
  user_id: string;
  role: string;
  status: string;
  joined_at?: string | null;
  name?: string | null;
  email?: string | null;
  image?: string | null;
}

export interface OrgInvite {
  id: string;
  email: string;
  role: string;
  status: string;
  expires_at: string;
  created_at: string;
}

export type OrgRole = "owner" | "admin" | "member";

export async function listMyOrganizations(): Promise<Organization[]> {
  const res = await request<{ organizations: Organization[] }>(
    "/organizations",
    { skipCache: true },
  );
  return res.organizations ?? [];
}

export function createOrganization({
  name,
}: {
  name: string;
}): Promise<Organization> {
  return request<Organization>("/organizations", {
    method: "POST",
    body: { name },
  });
}

export function getOrganization(id: string): Promise<Organization> {
  return request<Organization>(`/organizations/${id}`, { skipCache: true });
}

export function updateOrganization(
  id: string,
  patch: { name?: string; logo_url?: string },
): Promise<unknown> {
  return request(`/organizations/${id}`, { method: "PUT", body: patch });
}

export async function listOrgMembers(id: string): Promise<OrgMember[]> {
  const res = await request<{ members: OrgMember[] }>(
    `/organizations/${id}/members`,
    { skipCache: true },
  );
  return res.members ?? [];
}

export function updateOrgMemberRole(
  id: string,
  userId: string,
  role: OrgRole,
): Promise<unknown> {
  return request(`/organizations/${id}/members/${userId}`, {
    method: "PUT",
    body: { role },
  });
}

export function removeOrgMember(id: string, userId: string): Promise<unknown> {
  return request(`/organizations/${id}/members/${userId}`, {
    method: "DELETE",
  });
}

export async function listOrgInvites(id: string): Promise<OrgInvite[]> {
  const res = await request<{ invites: OrgInvite[] }>(
    `/organizations/${id}/invites`,
    { skipCache: true },
  );
  return res.invites ?? [];
}

export function createOrgInvite(
  id: string,
  email: string,
  role: "admin" | "member",
): Promise<OrgInvite> {
  return request<OrgInvite>(`/organizations/${id}/invites`, {
    method: "POST",
    body: { email, role },
  });
}

export function revokeOrgInvite(
  id: string,
  inviteId: string,
): Promise<unknown> {
  return request(`/organizations/${id}/invites/${inviteId}`, {
    method: "DELETE",
  });
}

export async function listOrgWorkspaces(
  id: string,
): Promise<{ id: string; name: string; slug: string }[]> {
  const res = await request<{
    workspaces: { id: string; name: string; slug: string }[];
  }>(`/organizations/${id}/workspaces`, { skipCache: true });
  return res.workspaces ?? [];
}

export function validateOrgInvite(token: string): Promise<{
  valid: boolean;
  organization_name?: string;
  role?: string;
  error?: string;
}> {
  return request("/organizations/invites/validate", {
    method: "POST",
    body: { token },
  });
}

export function acceptOrgInvite(
  token: string,
): Promise<{ organization_id: string; role: string }> {
  return request("/organizations/invites/accept", {
    method: "POST",
    body: { token },
  });
}
