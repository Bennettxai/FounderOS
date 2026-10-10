import { request } from '../base';
import type {
  Context,
  ContextListItem,
  CreateContextData,
  UpdateContextData,
  BlocksUpdateData,
  ShareResponse,
  AggregateContextRequest,
  AggregateContextResponse
} from './types';

export async function getContexts(filters?: { type?: string; includeArchived?: boolean; templatesOnly?: boolean; parentId?: string; search?: string }) {
  const params = new URLSearchParams();
  if (filters?.type) params.set('type_filter', filters.type);
  if (filters?.includeArchived) params.set('include_archived', 'true');
  if (filters?.templatesOnly) params.set('templates_only', 'true');
  if (filters?.parentId) params.set('parent_id', filters.parentId);
  if (filters?.search) params.set('search', filters.search);
  const query = params.toString();
  return request<ContextListItem[]>(`/contexts${query ? `?${query}` : ''}`);
}

export async function getContext(id: string) {
  return request<Context>(`/contexts/${id}`);
}

export async function createContext(data: CreateContextData) {
  return request<Context>('/contexts', { method: 'POST', body: data });
}

export async function updateContext(id: string, data: UpdateContextData) {
  return request<Context>(`/contexts/${id}`, { method: 'PUT', body: data });
}

export async function updateContextBlocks(id: string, data: BlocksUpdateData) {
  return request<Context>(`/contexts/${id}/blocks`, { method: 'PATCH', body: data });
}

export async function enableContextSharing(id: string) {
  return request<ShareResponse>(`/contexts/${id}/share`, { method: 'POST' });
}

export async function disableContextSharing(id: string) {
  return request(`/contexts/${id}/share`, { method: 'DELETE' });
}

export async function getPublicContext(shareId: string) {
  return request<Context>(`/contexts/public/${shareId}`);
}

export async function duplicateContext(id: string) {
  return request<Context>(`/contexts/${id}/duplicate`, { method: 'POST' });
}

export async function archiveContext(id: string) {
  return request(`/contexts/${id}/archive`, { method: 'PATCH' });
}

export async function unarchiveContext(id: string) {
  return request(`/contexts/${id}/unarchive`, { method: 'PATCH' });
}

export async function deleteContext(id: string) {
  return request(`/contexts/${id}`, { method: 'DELETE' });
}

export async function aggregateContext(data: AggregateContextRequest) {
  return request<AggregateContextResponse>('/contexts/aggregate', { method: 'POST', body: data });
}
