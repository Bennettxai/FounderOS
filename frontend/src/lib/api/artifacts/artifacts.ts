import { request } from '../base';
import type { Artifact, ArtifactListItem, CreateArtifactData, UpdateArtifactData, ArtifactFilters, ArtifactVersion } from './types';

export async function getArtifacts(filters?: ArtifactFilters) {
  const params = new URLSearchParams();
  if (filters?.type) params.set('type', filters.type);
  if (filters?.conversationId) params.set('conversation_id', filters.conversationId);
  if (filters?.projectId) params.set('project_id', filters.projectId);
  if (filters?.contextId) params.set('context_id', filters.contextId);
  if (filters?.unassignedOnly) params.set('unassigned_only', 'true');
  const query = params.toString();
  return request<ArtifactListItem[]>(`/artifacts${query ? `?${query}` : ''}`);
}

export async function getArtifact(id: string) {
  return request<Artifact>(`/artifacts/${id}`);
}

export async function createArtifact(data: CreateArtifactData) {
  return request<Artifact>('/artifacts', { method: 'POST', body: data });
}

export async function updateArtifact(id: string, data: UpdateArtifactData) {
  return request<Artifact>(`/artifacts/${id}`, { method: 'PATCH', body: data });
}

export async function deleteArtifact(id: string) {
  return request(`/artifacts/${id}`, { method: 'DELETE' });
}

export async function linkArtifact(id: string, data: { project_id?: string; context_id?: string; sync_to_kb?: boolean }) {
  return request<Artifact>(`/artifacts/${id}/link`, { method: 'PATCH', body: data });
}

export async function getArtifactVersions(id: string) {
  return request<ArtifactVersion[]>(`/artifacts/${id}/versions`);
}

export async function restoreArtifactVersion(id: string, version: number) {
  return request<Artifact>(`/artifacts/${id}/restore`, { method: 'POST', body: { version } });
}
