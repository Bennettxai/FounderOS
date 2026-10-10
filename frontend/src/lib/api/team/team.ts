import { request } from '../base';
import type {
  TeamMemberListResponse,
  TeamMemberDetailResponse,
  TeamMemberResponse,
  CreateTeamMemberData,
  UpdateTeamMemberData
} from './types';

export async function getTeamMembers(status?: string) {
  const params = status ? `?status_filter=${encodeURIComponent(status)}` : '';
  const raw = await request<Record<string, unknown>>(`/team${params}`);

  // Backend returns paginated format { data: [...], pagination: { total_items } }
  if (raw && typeof raw === 'object' && 'data' in raw && Array.isArray(raw.data)) {
    return raw.data as TeamMemberListResponse[];
  }

  // Fallback: raw array
  if (Array.isArray(raw)) {
    return raw as TeamMemberListResponse[];
  }

  return [] as TeamMemberListResponse[];
}

export async function getTeamMember(id: string) {
  return request<TeamMemberDetailResponse>(`/team/${id}`);
}

export async function createTeamMember(data: CreateTeamMemberData) {
  return request<TeamMemberDetailResponse>(`/team`, { method: 'POST', body: data });
}

export async function updateTeamMember(id: string, data: UpdateTeamMemberData) {
  return request<TeamMemberDetailResponse>(`/team/${id}`, { method: 'PUT', body: data });
}

export async function deleteTeamMember(id: string) {
  return request(`/team/${id}`, { method: 'DELETE' });
}

export async function updateTeamMemberStatus(id: string, status: string) {
  return request<TeamMemberResponse>(`/team/${id}/status?new_status=${encodeURIComponent(status)}`, { method: 'PATCH' });
}

export async function updateTeamMemberCapacity(id: string, capacity: number) {
  return request<TeamMemberResponse>(`/team/${id}/capacity?capacity=${capacity}`, { method: 'PATCH' });
}
