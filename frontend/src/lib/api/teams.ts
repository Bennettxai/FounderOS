// Teams API client — teams live inside a workspace; members get moved between them.
import { request } from "./base";

export interface Team {
  id: string;
  name: string;
  description?: string;
  color?: string;
  member_count?: number;
}

export interface TeamMember {
  user_id: string;
  role: string;
  name?: string | null;
  email?: string | null;
  image?: string | null;
}

export async function listTeams(workspaceId: string): Promise<Team[]> {
  const res = await request<{ teams: Team[] }>(
    `/workspaces/${workspaceId}/teams`,
    { skipCache: true },
  );
  return res.teams ?? [];
}

export function createTeam(
  workspaceId: string,
  data: { name: string; description?: string; color?: string },
): Promise<Team> {
  return request<Team>(`/workspaces/${workspaceId}/teams`, {
    method: "POST",
    body: data,
  });
}

export function updateTeam(
  teamId: string,
  patch: { name?: string; description?: string; color?: string },
): Promise<unknown> {
  return request(`/teams/${teamId}`, { method: "PUT", body: patch });
}

export function deleteTeam(teamId: string): Promise<unknown> {
  return request(`/teams/${teamId}`, { method: "DELETE" });
}

export async function listTeamMembers(teamId: string): Promise<TeamMember[]> {
  const res = await request<{ members: TeamMember[] }>(
    `/teams/${teamId}/members`,
    { skipCache: true },
  );
  return res.members ?? [];
}

export function addTeamMember(
  teamId: string,
  userId: string,
  role = "member",
): Promise<unknown> {
  return request(`/teams/${teamId}/members`, {
    method: "POST",
    body: { user_id: userId, role },
  });
}

export function removeTeamMember(
  teamId: string,
  userId: string,
): Promise<unknown> {
  return request(`/teams/${teamId}/members/${userId}`, { method: "DELETE" });
}

export function moveTeamMember(
  fromTeamId: string,
  userId: string,
  toTeamId: string,
): Promise<unknown> {
  return request(`/teams/${fromTeamId}/members/${userId}/move`, {
    method: "POST",
    body: { to_team_id: toTeamId },
  });
}
