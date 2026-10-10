// Pedro Documents API - Document Processing with RAG
import { request, getApiBaseUrl } from '../base';
import type {
  ProcessedDocument,
  DocumentListItem,
  DocumentChunk,
  DocumentSearchResult,
  DocumentUploadResponse,
  DocumentMetadata,
  DocumentSearchParams,
  RelevantChunksParams
} from './types';

/**
 * Upload and process a document for RAG
 */
export async function uploadDocument(
  file: File,
  metadata?: DocumentMetadata
): Promise<DocumentUploadResponse> {
  const formData = new FormData();
  formData.append('file', file);

  if (metadata) {
    if (metadata.title) formData.append('title', metadata.title);
    if (metadata.description) formData.append('description', metadata.description);
    if (metadata.tags) formData.append('tags', JSON.stringify(metadata.tags));
    if (metadata.project_id) formData.append('project_id', metadata.project_id);
    if (metadata.node_id) formData.append('node_id', metadata.node_id);
  }

  const response = await fetch(`${getApiBaseUrl()}/documents`, {
    method: 'POST',
    body: formData,
    credentials: 'include'
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ detail: 'Upload failed' }));
    throw new Error(error.detail || error.message || 'Upload failed');
  }

  return response.json();
}

/**
 * List all uploaded documents
 */
export async function listDocuments(): Promise<DocumentListItem[]> {
  return request<DocumentListItem[]>('/documents');
}

/**
 * Search documents using semantic search
 */
export async function searchDocuments(
  params: DocumentSearchParams
): Promise<DocumentSearchResult[]> {
  return request<DocumentSearchResult[]>('/documents/search', {
    method: 'POST',
    body: JSON.stringify(params)
  });
}

/**
 * Get relevant chunks for a query (for RAG context injection)
 */
export async function getRelevantChunks(
  params: RelevantChunksParams
): Promise<DocumentChunk[]> {
  return request<DocumentChunk[]>('/documents/chunks', {
    method: 'POST',
    body: JSON.stringify(params)
  });
}

/**
 * Get a specific document by ID
 */
export async function getDocument(id: string): Promise<ProcessedDocument> {
  return request<ProcessedDocument>(`/documents/${id}`);
}

/**
 * Delete a document
 */
export async function deleteDocument(id: string): Promise<void> {
  await request<void>(`/documents/${id}`, {
    method: 'DELETE'
  });
}

/**
 * Reprocess a document (regenerate chunks and embeddings)
 */
export async function reprocessDocument(id: string): Promise<ProcessedDocument> {
  return request<ProcessedDocument>(`/documents/${id}/reprocess`, {
    method: 'POST'
  });
}

/**
 * Get document content (full text)
 */
export async function getDocumentContent(id: string): Promise<{ content: string }> {
  return request<{ content: string }>(`/documents/${id}/content`);
}
