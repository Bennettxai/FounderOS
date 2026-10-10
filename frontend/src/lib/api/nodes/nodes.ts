import { request, raw, getApiBaseUrl } from '../base';
import type {
  Node,
  NodeTree,
  NodeDetail,
  NodeActivateResponse,
  CreateNodeData,
  UpdateNodeData,
  NodeLinks,
  NodeLinkCounts,
  LinkedProject,
  LinkedContext,
  LinkedConversation
} from './types';

export async function getNodes(includeArchived = false) {
  const params = includeArchived ? '?include_archived=true' : '';
  return request<Node[]>(`/nodes${params}`);
}

export async function getNodeTree(includeArchived = false) {
  const params = includeArchived ? '?include_archived=true' : '';
  return request<NodeTree[]>(`/nodes/tree${params}`);
}

export async function getActiveNode(): Promise<Node | null> {
  // Use raw.get to avoid console.error spam for expected 404s
  const response = await raw.get('/nodes/active');
  if (response.status === 404) {
    // 404 is expected when no node is active
    return null;
  }
  if (!response.ok) {
    throw new Error(`Failed to get active node (HTTP ${response.status})`);
  }
  return response.json();
}

export async function getNode(id: string) {
  return request<NodeDetail>(`/nodes/${id}`);
}

export async function createNode(data: CreateNodeData) {
  return request<Node>('/nodes', { method: 'POST', body: data });
}

export async function updateNode(id: string, data: UpdateNodeData) {
  return request<Node>(`/nodes/${id}`, { method: 'PATCH', body: data });
}

export async function activateNode(id: string) {
  return request<NodeActivateResponse>(`/nodes/${id}/activate`, { method: 'POST' });
}

export async function deactivateNode(id: string) {
  return request<Node>(`/nodes/${id}/deactivate`, { method: 'POST' });
}

export async function deleteNode(id: string) {
  return request(`/nodes/${id}`, { method: 'DELETE' });
}

export async function getNodeChildren(id: string, includeArchived = false) {
  const params = includeArchived ? '?include_archived=true' : '';
  return request<Node[]>(`/nodes/${id}/children${params}`);
}

export async function reorderNode(id: string, newOrder: number) {
  return request(`/nodes/${id}/reorder?new_order=${newOrder}`, { method: 'POST' });
}

export async function archiveNode(id: string) {
  return request<Node>(`/nodes/${id}/archive`, { method: 'POST' });
}

export async function unarchiveNode(id: string) {
  return request<Node>(`/nodes/${id}/unarchive`, { method: 'POST' });
}

// ===== LINKING API FUNCTIONS =====

// Note: These endpoints may not exist yet on the backend.
// Use raw fetch to avoid console.error spam from base request.
// We differentiate between "not found/not implemented" (return empty) and auth errors (throw).

export async function getNodeLinks(nodeId: string): Promise<NodeLinks> {
  try {
    const response = await raw.get(`/nodes/${nodeId}/links`);
    // Auth errors should be surfaced, not silently swallowed
    if (response.status === 401 || response.status === 403) {
      throw new Error(`Authorization error: ${response.status}`);
    }
    if (!response.ok) {
      // 404 or other errors - endpoint may not exist yet
      return { projects: [], contexts: [], conversations: [] };
    }
    return await response.json();
  } catch (error) {
    // Re-throw auth errors
    if (error instanceof Error && error.message.startsWith('Authorization')) {
      throw error;
    }
    // Network errors, etc. - return empty
    return { projects: [], contexts: [], conversations: [] };
  }
}

export async function getNodeLinkCounts(nodeId: string): Promise<NodeLinkCounts> {
  try {
    const response = await raw.get(`/nodes/${nodeId}/links/counts`);
    // Auth errors should be surfaced
    if (response.status === 401 || response.status === 403) {
      throw new Error(`Authorization error: ${response.status}`);
    }
    if (!response.ok) {
      // 404 or other errors - endpoint may not exist yet
      return { linked_projects_count: 0, linked_contexts_count: 0, linked_conversations_count: 0 };
    }
    return await response.json();
  } catch (error) {
    // Re-throw auth errors
    if (error instanceof Error && error.message.startsWith('Authorization')) {
      throw error;
    }
    // Network errors, etc. - return zeros
    return { linked_projects_count: 0, linked_contexts_count: 0, linked_conversations_count: 0 };
  }
}

// Helper to check for auth errors
function checkAuthError(response: Response): void {
  if (response.status === 401 || response.status === 403) {
    throw new Error(`Authorization error: ${response.status}`);
  }
}

// Project linking
export async function linkNodeProject(nodeId: string, projectId: string): Promise<void> {
  try {
    const response = await raw.post(`/nodes/${nodeId}/links/projects`, { project_id: projectId });
    checkAuthError(response);
  } catch (error) {
    if (error instanceof Error && error.message.startsWith('Authorization')) {
      throw error;
    }
    // Silently fail if endpoint doesn't exist (404)
  }
}

export async function unlinkNodeProject(nodeId: string, projectId: string): Promise<void> {
  try {
    const response = await raw.delete(`/nodes/${nodeId}/links/projects/${projectId}`);
    checkAuthError(response);
  } catch (error) {
    if (error instanceof Error && error.message.startsWith('Authorization')) {
      throw error;
    }
    // Silently fail if endpoint doesn't exist
  }
}

// Context linking
export async function linkNodeContext(nodeId: string, contextId: string): Promise<void> {
  try {
    const response = await raw.post(`/nodes/${nodeId}/links/contexts`, { context_id: contextId });
    checkAuthError(response);
  } catch (error) {
    if (error instanceof Error && error.message.startsWith('Authorization')) {
      throw error;
    }
    // Silently fail if endpoint doesn't exist
  }
}

export async function unlinkNodeContext(nodeId: string, contextId: string): Promise<void> {
  try {
    const response = await raw.delete(`/nodes/${nodeId}/links/contexts/${contextId}`);
    checkAuthError(response);
  } catch (error) {
    if (error instanceof Error && error.message.startsWith('Authorization')) {
      throw error;
    }
    // Silently fail if endpoint doesn't exist
  }
}

// Conversation linking
export async function linkNodeConversation(nodeId: string, conversationId: string): Promise<void> {
  try {
    const response = await raw.post(`/nodes/${nodeId}/links/conversations`, { conversation_id: conversationId });
    checkAuthError(response);
  } catch (error) {
    if (error instanceof Error && error.message.startsWith('Authorization')) {
      throw error;
    }
    // Silently fail if endpoint doesn't exist
  }
}

export async function unlinkNodeConversation(nodeId: string, conversationId: string): Promise<void> {
  try {
    const response = await raw.delete(`/nodes/${nodeId}/links/conversations/${conversationId}`);
    checkAuthError(response);
  } catch (error) {
    if (error instanceof Error && error.message.startsWith('Authorization')) {
      throw error;
    }
    // Silently fail if endpoint doesn't exist
  }
}
