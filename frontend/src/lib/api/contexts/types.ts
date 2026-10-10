export type ContextType = 'person' | 'business' | 'project' | 'custom' | 'document';

export interface Block {
  id: string;
  type: string;
  content: string | null;
  properties?: Record<string, unknown>;
  children?: Block[];
}

export interface PropertySchema {
  name: string;
  type: 'text' | 'select' | 'multi_select' | 'date' | 'person' | 'relation' | 'number' | 'checkbox' | 'url' | 'email';
  options?: string[];
  relation_type?: 'context' | 'project' | 'client';
}

export interface Context {
  id: string;
  name: string;
  type: ContextType;
  content: string | null;
  structured_data: Record<string, unknown> | null;
  system_prompt_template: string | null;
  blocks: Block[] | null;
  cover_image: string | null;
  icon: string | null;
  parent_id: string | null;
  is_template: boolean;
  is_archived: boolean;
  last_edited_at: string | null;
  word_count: number;
  is_public: boolean;
  share_id: string | null;
  property_schema: PropertySchema[] | null;
  properties: Record<string, unknown> | null;
  client_id: string | null;
  created_at: string;
  updated_at: string;
}

export interface ContextListItem {
  id: string;
  name: string;
  type: ContextType;
  icon: string | null;
  cover_image: string | null;
  parent_id: string | null;
  is_template: boolean;
  is_archived: boolean;
  word_count: string | number;
  property_schema: PropertySchema[] | null;
  properties: Record<string, unknown> | null;
  client_id: string | null;
  updated_at: string;
}

export interface CreateContextData {
  name: string;
  type?: ContextType;
  content?: string;
  structured_data?: Record<string, unknown>;
  system_prompt_template?: string;
  blocks?: Block[];
  cover_image?: string;
  icon?: string;
  parent_id?: string;
  is_template?: boolean;
  property_schema?: PropertySchema[];
  properties?: Record<string, unknown>;
  client_id?: string;
}

export interface UpdateContextData {
  name?: string;
  type?: ContextType;
  content?: string;
  structured_data?: Record<string, unknown>;
  system_prompt_template?: string;
  blocks?: Block[];
  cover_image?: string;
  icon?: string;
  parent_id?: string | null;
  is_template?: boolean;
  is_archived?: boolean;
  is_public?: boolean;
  property_schema?: PropertySchema[];
  properties?: Record<string, unknown>;
  client_id?: string | null;
}

export interface BlocksUpdateData {
  blocks: Block[];
  word_count?: number;
}

export interface ShareResponse {
  share_id: string;
  is_public: boolean;
  share_url: string;
}

export interface AggregateContextRequest {
  context_ids?: string[];
  project_ids?: string[];
  node_ids?: string[];
  include_children?: boolean;
  include_artifacts?: boolean;
  include_tasks?: boolean;
  max_depth?: number;
}

export interface AggregatedContextItem {
  source_type: string;
  source_id: string;
  source_name: string;
  content: string;
  metadata?: Record<string, unknown>;
}

export interface AggregateContextResponse {
  items: AggregatedContextItem[];
  total_items: number;
  total_characters: number;
  formatted_context: string;
}
