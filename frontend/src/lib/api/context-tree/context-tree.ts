import { request } from '../base';
import type {
  EntityType,
  ContextTree,
  TreeSearchParams,
  TreeSearchResult,
  LoadContextItemParams,
  LoadedContextItem,
  ContextStats,
  LoadingRule,
  ContextSession,
  CreateSessionParams,
  UpdateSessionParams
} from './types';

// ============================================
// Context Tree Operations
// ============================================

export async function getContextTree(entityType: EntityType, entityId: string): Promise<ContextTree> {
  return request<ContextTree>(`/context-tree/${entityType}/${entityId}`);
}

export async function searchContextTree(params: TreeSearchParams): Promise<TreeSearchResult[]> {
  return request<TreeSearchResult[]>('/context-tree/search', { method: 'POST', body: params });
}

export async function loadContextItem(params: LoadContextItemParams): Promise<LoadedContextItem> {
  return request<LoadedContextItem>('/context-tree/load', { method: 'POST', body: params });
}

export async function getContextStats(): Promise<ContextStats> {
  return request<ContextStats>('/context-tree/stats');
}

// ============================================
// Loading Rules Operations
// ============================================

export async function getLoadingRules(entityType: EntityType, entityId: string): Promise<LoadingRule[]> {
  return request<LoadingRule[]>(`/context-tree/rules/${entityType}/${entityId}`);
}

// ============================================
// Context Session Operations
// ============================================

export async function createContextSession(params: CreateSessionParams): Promise<ContextSession> {
  return request<ContextSession>('/context-tree/session', { method: 'POST', body: params });
}

export async function getContextSession(sessionId: string): Promise<ContextSession> {
  return request<ContextSession>(`/context-tree/session/${sessionId}`);
}

export async function updateContextSession(sessionId: string, params: UpdateSessionParams): Promise<ContextSession> {
  return request<ContextSession>(`/context-tree/session/${sessionId}`, { method: 'PUT', body: params });
}

export async function endContextSession(sessionId: string): Promise<void> {
  return request(`/context-tree/session/${sessionId}`, { method: 'DELETE' });
}
