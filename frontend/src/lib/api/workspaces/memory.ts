import { request } from '../base';
import type { Memory, MemoryListItem, CreateMemoryData, UpdateMemoryData } from '../memory/types';

// ============================================
// Workspace Memory Types
// ============================================

export type MemoryVisibility = 'workspace' | 'private' | 'shared';

export interface WorkspaceMemory extends Memory {
  workspace_id: string;
  visibility: MemoryVisibility;
  shared_with_user_ids: string[] | null;
  created_by: string;
}

export interface WorkspaceMemoryListItem extends MemoryListItem {
  workspace_id: string;
  visibility: MemoryVisibility;
  shared_with_user_ids: string[] | null;
  created_by: string;
}

export interface CreateWorkspaceMemoryData extends CreateMemoryData {
  workspace_id: string;
  visibility?: MemoryVisibility;
  shared_with_user_ids?: string[];
}

export interface UpdateWorkspaceMemoryData extends UpdateMemoryData {
  visibility?: MemoryVisibility;
}

export interface WorkspaceMemoryFilters {
  memory_type?: string;
  visibility?: MemoryVisibility;
  is_pinned?: boolean;
  is_active?: boolean;
  min_importance?: number;
  tags?: string[];
  limit?: number;
  offset?: number;
}

export interface ShareMemoryData {
  user_ids: string[];
}

export interface UnshareMemoryData {
  user_ids: string[];
}

// ============================================
// Workspace Memory API Functions
// ============================================

/**
 * Create a new memory in a workspace
 */
export async function createWorkspaceMemory(
  workspaceId: string,
  data: CreateWorkspaceMemoryData
): Promise<WorkspaceMemory> {
  return request<WorkspaceMemory>(`/workspaces/${workspaceId}/memories`, {
    method: 'POST',
    body: data,
  });
}

/**
 * List all workspace-level memories (visibility = 'workspace')
 */
export async function listWorkspaceMemories(
  workspaceId: string,
  filters?: WorkspaceMemoryFilters
): Promise<WorkspaceMemoryListItem[]> {
  const params = new URLSearchParams();
  if (filters?.memory_type) params.set('memory_type', filters.memory_type);
  if (filters?.visibility) params.set('visibility', filters.visibility);
  if (filters?.is_pinned !== undefined) params.set('is_pinned', String(filters.is_pinned));
  if (filters?.is_active !== undefined) params.set('is_active', String(filters.is_active));
  if (filters?.min_importance !== undefined)
    params.set('min_importance', String(filters.min_importance));
  if (filters?.tags?.length) params.set('tags', filters.tags.join(','));
  if (filters?.limit) params.set('limit', String(filters.limit));
  if (filters?.offset) params.set('offset', String(filters.offset));

  const query = params.toString();
  const data = await request<{ memories: WorkspaceMemoryListItem[] }>(
    `/workspaces/${workspaceId}/memories${query ? `?${query}` : ''}`
  );
  return data.memories;
}

/**
 * List private memories (visibility = 'private')
 */
export async function listPrivateMemories(
  workspaceId: string,
  filters?: WorkspaceMemoryFilters
): Promise<WorkspaceMemoryListItem[]> {
  const params = new URLSearchParams();
  if (filters?.memory_type) params.set('memory_type', filters.memory_type);
  if (filters?.is_pinned !== undefined) params.set('is_pinned', String(filters.is_pinned));
  if (filters?.is_active !== undefined) params.set('is_active', String(filters.is_active));
  if (filters?.min_importance !== undefined)
    params.set('min_importance', String(filters.min_importance));
  if (filters?.tags?.length) params.set('tags', filters.tags.join(','));
  if (filters?.limit) params.set('limit', String(filters.limit));
  if (filters?.offset) params.set('offset', String(filters.offset));

  const query = params.toString();
  const data = await request<{ memories: WorkspaceMemoryListItem[] }>(
    `/workspaces/${workspaceId}/memories/private${query ? `?${query}` : ''}`
  );
  return data.memories;
}

/**
 * List all accessible memories (workspace + private + shared)
 */
export async function listAccessibleMemories(
  workspaceId: string,
  filters?: WorkspaceMemoryFilters
): Promise<WorkspaceMemoryListItem[]> {
  const params = new URLSearchParams();
  if (filters?.memory_type) params.set('memory_type', filters.memory_type);
  if (filters?.visibility) params.set('visibility', filters.visibility);
  if (filters?.is_pinned !== undefined) params.set('is_pinned', String(filters.is_pinned));
  if (filters?.is_active !== undefined) params.set('is_active', String(filters.is_active));
  if (filters?.min_importance !== undefined)
    params.set('min_importance', String(filters.min_importance));
  if (filters?.tags?.length) params.set('tags', filters.tags.join(','));
  if (filters?.limit) params.set('limit', String(filters.limit));
  if (filters?.offset) params.set('offset', String(filters.offset));

  const query = params.toString();
  const data = await request<{ memories: WorkspaceMemoryListItem[] }>(
    `/workspaces/${workspaceId}/memories/accessible${query ? `?${query}` : ''}`
  );
  return data.memories;
}

/**
 * Get a specific workspace memory
 */
export async function getWorkspaceMemory(
  workspaceId: string,
  memoryId: string
): Promise<WorkspaceMemory> {
  return request<WorkspaceMemory>(`/workspaces/${workspaceId}/memories/${memoryId}`);
}

/**
 * Update a workspace memory
 */
export async function updateWorkspaceMemory(
  workspaceId: string,
  memoryId: string,
  data: UpdateWorkspaceMemoryData
): Promise<WorkspaceMemory> {
  return request<WorkspaceMemory>(`/workspaces/${workspaceId}/memories/${memoryId}`, {
    method: 'PUT',
    body: data,
  });
}

/**
 * Delete a workspace memory
 */
export async function deleteWorkspaceMemory(workspaceId: string, memoryId: string): Promise<void> {
  return request<void>(`/workspaces/${workspaceId}/memories/${memoryId}`, {
    method: 'DELETE',
  });
}

/**
 * Share a memory with specific users
 */
export async function shareMemory(
  workspaceId: string,
  memoryId: string,
  data: ShareMemoryData
): Promise<WorkspaceMemory> {
  return request<WorkspaceMemory>(`/workspaces/${workspaceId}/memories/${memoryId}/share`, {
    method: 'POST',
    body: data,
  });
}

/**
 * Unshare a memory from specific users
 */
export async function unshareMemory(
  workspaceId: string,
  memoryId: string,
  data: UnshareMemoryData
): Promise<WorkspaceMemory> {
  return request<WorkspaceMemory>(`/workspaces/${workspaceId}/memories/${memoryId}/share`, {
    method: 'DELETE',
    body: data,
  });
}

/**
 * Pin/unpin a workspace memory
 */
export async function pinWorkspaceMemory(
  workspaceId: string,
  memoryId: string,
  pinned: boolean
): Promise<WorkspaceMemory> {
  return request<WorkspaceMemory>(`/workspaces/${workspaceId}/memories/${memoryId}/pin`, {
    method: 'POST',
    body: { is_pinned: pinned },
  });
}
