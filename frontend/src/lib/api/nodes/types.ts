// Nodes API Types

export type NodeType = 'business' | 'project' | 'learning' | 'operational';
export type NodeHealth = 'healthy' | 'needs_attention' | 'critical' | 'not_started';

export interface DecisionItem {
  id: string;
  question: string;
  added_at: string;
  decided: boolean;
  decision: string | null;
}

export interface DelegationItem {
  id: string;
  task: string;
  assignee_id: string | null;
  assignee_name: string | null;
  status: string;
}

export interface Node {
  id: string;
  user_id: string;
  parent_id: string | null;
  context_id: string | null;
  name: string;
  type: NodeType;
  health: NodeHealth;
  purpose: string | null;
  current_status: string | null;
  this_week_focus: string[] | null;
  decision_queue: DecisionItem[] | null;
  delegation_ready: DelegationItem[] | null;
  is_active: boolean;
  is_archived: boolean;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

export interface NodeTree extends Node {
  children: NodeTree[];
  children_count: number;
}

export interface NodeDetail extends Node {
  parent_name: string | null;
  children_count: number;
  linked_projects_count: number;
  linked_conversations_count: number;
  linked_artifacts_count: number;
}

export interface NodeActivateResponse {
  node: Node;
  previous_active_id: string | null;
  context_prompt: string | null;
}

export interface CreateNodeData {
  name: string;
  type: NodeType;
  parent_id?: string;
  purpose?: string;
  context_id?: string;
}

export interface UpdateNodeData {
  name?: string;
  type?: NodeType;
  parent_id?: string | null;
  health?: NodeHealth;
  purpose?: string;
  current_status?: string;
  this_week_focus?: string[];
  decision_queue?: DecisionItem[];
  delegation_ready?: DelegationItem[];
  is_active?: boolean;
  is_archived?: boolean;
  sort_order?: number;
  context_id?: string;
}

// ===== LINKING TYPES =====

export interface LinkedProject {
  id: string;
  user_id: string;
  name: string;
  description: string | null;
  status: string;
  priority: string | null;
  start_date: string | null;
  end_date: string | null;
  created_at: string;
  updated_at: string;
  linked_at: string;
}

export interface LinkedContext {
  id: string;
  user_id: string;
  name: string;
  type: string;
  description: string | null;
  content: string | null;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  linked_at: string;
}

export interface LinkedConversation {
  id: string;
  user_id: string;
  title: string | null;
  model: string | null;
  provider: string | null;
  is_archived: boolean;
  created_at: string;
  updated_at: string;
  linked_at: string;
}

export interface NodeLinks {
  projects: LinkedProject[];
  contexts: LinkedContext[];
  conversations: LinkedConversation[];
}

export interface NodeLinkCounts {
  linked_projects_count: number;
  linked_contexts_count: number;
  linked_conversations_count: number;
}
