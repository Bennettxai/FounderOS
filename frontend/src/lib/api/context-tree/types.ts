// Context Tree API Types

export type EntityType = 'project' | 'node' | 'conversation' | 'user' | 'memories' | 'contexts' | 'artifacts' | 'documents' | 'voice_notes';
export type ContextItemType = 'project' | 'context' | 'memory' | 'document' | 'conversation' | 'task' | 'note';

export interface ContextTreeNode {
  id: string;
  title: string;
  type: ContextItemType;
  summary?: string;
  token_estimate: number;
  is_expanded: boolean;
  is_loaded: boolean;
  children: ContextTreeNode[];
  metadata?: Record<string, unknown>;
}

export interface ContextTree {
  root_node: ContextTreeNode;
  total_items: number;
  last_updated: string;
}

export interface TreeSearchParams {
  query: string;
  search_type?: 'semantic' | 'keyword' | 'hybrid' | 'title' | 'content' | 'browse';
  entity_types?: ContextItemType[];
  entity_type?: EntityType;  // Alternative singular form
  max_results?: number;
  limit?: number;  // Alternative for max_results
  project_scope?: string;
  project_id?: string;  // Alternative for project_scope
  node_scope?: string;
  node_id?: string;  // Alternative for node_scope
}

export interface TreeSearchResult {
  id: string;
  title: string;
  type: ContextItemType;
  summary: string;
  relevance_score: number;
  tree_path: string[];
  token_estimate: number;
  entity_type?: EntityType | 'all';
  snippet?: string;
  token_count?: number;
  created_at?: string;
}

export interface LoadContextItemParams {
  item_id: string;
  item_type: ContextItemType;
  include_related?: boolean;
  max_tokens?: number;
}

export interface LoadedContextItem {
  id: string;
  type: ContextItemType;
  title: string;
  content: string;
  summary?: string;
  token_count: number;
  related_items?: LoadedContextItem[];
  metadata?: Record<string, unknown>;
}

export interface ContextStats {
  total_items: number;
  by_type: Record<ContextItemType, number>;
  total_tokens: number;
  last_updated: string;
}

export interface LoadingRule {
  id: string;
  entity_type: EntityType;
  entity_id: string;
  item_type: ContextItemType;
  priority: number;
  max_items: number;
  max_tokens: number;
  filters?: Record<string, unknown>;
  is_active: boolean;
}

export interface ContextSession {
  id: string;
  user_id: string;
  entity_type: EntityType;
  entity_id: string;
  loaded_items: string[];
  total_tokens: number;
  max_tokens: number;
  started_at: string;
  last_activity_at: string;
  metadata?: Record<string, unknown>;
}

export interface CreateSessionParams {
  entity_type: EntityType;
  entity_id: string;
  max_tokens?: number;
  auto_load?: boolean;
}

export interface UpdateSessionParams {
  loaded_items?: string[];
  total_tokens?: number;
  metadata?: Record<string, unknown>;
}
