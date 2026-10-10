// Pedro Documents API Types - Document Processing with RAG

export type DocumentStatus = 'pending' | 'processing' | 'ready' | 'error';
export type ChunkType = 'text' | 'code' | 'heading' | 'list' | 'table';

export interface DocumentMetadata {
  title?: string;
  description?: string;
  tags?: string[];
  project_id?: string;
  node_id?: string;
}

export interface ProcessedDocument {
  id: string;
  user_id: string;
  filename: string;
  original_filename: string;
  file_type: string;
  file_size: number;
  title?: string;
  description?: string;
  status: DocumentStatus;
  chunk_count: number;
  total_tokens: number;
  project_id?: string;
  node_id?: string;
  tags: string[];
  metadata: Record<string, unknown>;
  processed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface DocumentListItem {
  id: string;
  filename: string;
  title?: string;
  file_type: string;
  file_size: number;
  status: DocumentStatus;
  chunk_count: number;
  tags: string[];
  created_at: string;
}

export interface DocumentChunk {
  id: string;
  document_id: string;
  chunk_index: number;
  content: string;
  token_count: number;
  start_char: number;
  end_char: number;
  section_title?: string;
  chunk_type: ChunkType;
  relevance_score?: number;
}

export interface DocumentSearchResult {
  document_id: string;
  document_title: string;
  filename: string;
  chunk_id: string;
  chunk_content: string;
  chunk_index: number;
  relevance_score: number;
  highlights?: string[];
}

export interface DocumentUploadResponse {
  document: ProcessedDocument;
  message: string;
  // Convenience properties - flattened from document
  id?: string;
  display_name?: string;
  original_filename?: string;
  word_count?: number;
}

export interface DocumentSearchParams {
  query: string;
  limit?: number;
  document_ids?: string[];
  project_id?: string;
  min_relevance?: number;
}

export interface RelevantChunksParams {
  query: string;
  document_ids?: string[];
  limit?: number;
  min_relevance?: number;
}
