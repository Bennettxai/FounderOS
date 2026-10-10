// Memory API Types

export type MemoryType = 'fact' | 'preference' | 'decision' | 'event' | 'learning' | 'context' | 'relationship';

export interface Memory {
  id: string;
  user_id: string;
  title: string;
  summary: string;
  content: string;
  memory_type: MemoryType;
  importance_score: number;
  is_pinned: boolean;
  is_active: boolean;
  tags: string[];
  metadata: Record<string, unknown>;
  source_type: string | null;
  source_id: string | null;
  project_id: string | null;
  node_id: string | null;
  expires_at: string | null;
  access_count: number;
  last_accessed_at: string | null;
  created_at: string;
  updated_at: string;

  // Visualization customization
  icon?: string | null;        // Custom icon (SVG path or emoji)
  color?: string | null;       // Custom bubble color (hex)
  cover_image?: string | null; // Image URL to display inside bubble
  learning_summary?: string;   // Short learning-focused summary for display
}

export interface MemoryListItem {
  id: string;
  title: string;
  summary: string;
  content?: string;
  memory_type: MemoryType;
  importance_score: number;
  is_pinned: boolean;
  is_active: boolean;
  tags: string[];
  project_id: string | null;
  node_id: string | null;
  access_count: number;
  last_accessed_at: string | null;
  created_at: string;
  updated_at: string;
  metadata?: Record<string, unknown>;
}

export interface CreateMemoryData {
  title: string;
  summary?: string;
  content: string;
  memory_type?: MemoryType;
  importance_score?: number;
  tags?: string[];
  metadata?: Record<string, unknown>;
  source_type?: string;
  source_id?: string;
  project_id?: string;
  node_id?: string;
  expires_at?: string;
}

export interface UpdateMemoryData {
  title?: string;
  summary?: string;
  content?: string;
  memory_type?: MemoryType;
  importance_score?: number;
  is_pinned?: boolean;
  is_active?: boolean;
  tags?: string[];
  metadata?: Record<string, unknown>;
  expires_at?: string;
  // Visualization customization
  icon?: string;
  color?: string;
  cover_image?: string;
}

export interface MemoryFilters {
  memory_type?: MemoryType;
  project_id?: string;
  node_id?: string;
  is_pinned?: boolean;
  is_active?: boolean;
  min_importance?: number;
  tags?: string[];
  limit?: number;
  offset?: number;
}

export interface MemorySearchParams {
  query: string;
  memory_type?: MemoryType;
  project_id?: string;
  node_id?: string;
  min_score?: number;
  limit?: number;
}

export interface MemorySearchResult extends MemoryListItem {
  // Returned by different endpoints
  similarity?: number;
  relevance_score?: number;
}

export interface RelevantMemoryParams {
  query: string;
  conversation_id?: string;
  project_id?: string;
  node_id?: string;
  memory_types?: MemoryType[];
  limit?: number;
  min_relevance?: number;
}

export interface MemoryStats {
  total_memories: number;
  active_memories: number;
  pinned_memories: number;
  by_type: Record<MemoryType, number>;
  avg_importance: number;
  total_access_count: number;
}

export interface UserFact {
  id: string;
  user_id: string;
  fact_key: string;
  fact_value: string;
  fact_type: string;
  source_memory_id: string | null;
  confidence_score: number;
  is_active: boolean;
  last_confirmed_at: string | null;
  created_at: string;
  updated_at: string;
  // Additional properties used in UI
  key?: string;
  value?: string;
  type?: string;
  source?: string;
  description?: string;
  is_confirmed?: boolean;
  is_rejected?: boolean;
}

export interface UpdateUserFactData {
  fact_value: string;
  fact_type?: string;
  confidence_score?: number;
}
