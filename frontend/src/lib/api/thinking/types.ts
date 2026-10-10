// Thinking API Types
// Matches backend Go structs from internal/database/sqlc/models.go and internal/handlers/thinking.go

// ============================================================================
// THINKING TYPES (ENUMS)
// ============================================================================

export type ThinkingType =
  | 'analysis'
  | 'planning'
  | 'reflection'
  | 'tool_use'
  | 'reasoning'
  | 'evaluation';

// ============================================================================
// THINKING TRACE
// ============================================================================

export interface ThinkingStep {
  step_number: number;
  step_type: 'understand' | 'analyze' | 'plan' | 'reason' | 'evaluate' | 'conclude' | 'verify';
  content: string;
  duration_ms?: number;
}

export interface ThinkingTrace {
  id: string;
  user_id: string;
  conversation_id: string;
  message_id: string;
  thinking_content: string;
  thinking_type?: ThinkingType | null;
  step_number?: number | null;
  started_at: string;
  completed_at?: string | null;
  duration_ms?: number | null;
  thinking_tokens?: number | null;
  model_used?: string | null;
  reasoning_template_id?: string | null;
  metadata?: Record<string, unknown> | null;
  created_at: string;
}

export interface ThinkingTracesResponse {
  traces: ThinkingTrace[];
}

// ============================================================================
// REASONING TEMPLATES
// ============================================================================

export type StepType = 'exploration' | 'analysis' | 'conclusion' | 'reflection';

export interface ReasoningStep {
  id?: string;
  order: number;
  type: StepType;
  prompt: string;
}

export interface ReasoningTemplate {
  id: string;
  user_id: string;
  name: string;
  description?: string | null;
  steps: ReasoningStep[];
  system_prompt?: string | null;
  thinking_instruction?: string | null;
  output_format?: string | null;
  show_thinking?: boolean | null;
  save_thinking?: boolean | null;
  max_thinking_tokens?: number | null;
  times_used?: number | null;
  is_default?: boolean | null;
  created_at: string;
  updated_at: string;
}

export interface ReasoningTemplatesResponse {
  templates: ReasoningTemplate[];
}

// ============================================================================
// THINKING SETTINGS
// ============================================================================

export interface ThinkingSettings {
  enabled: boolean;
  show_in_ui: boolean;
  save_traces: boolean;
  default_template_id?: string | null;
  max_tokens: number;
}

// ============================================================================
// REQUEST/RESPONSE TYPES
// ============================================================================

export interface CreateTemplateData {
  name: string;
  description?: string;
  steps: Omit<ReasoningStep, 'id'>[];
  system_prompt?: string;
  thinking_instruction?: string;
  output_format?: string;
  show_thinking?: boolean;
  save_thinking?: boolean;
  max_thinking_tokens?: number;
  is_default?: boolean;
}

export interface UpdateTemplateData {
  name?: string;
  description?: string;
  steps?: Omit<ReasoningStep, 'id'>[];
  system_prompt?: string;
  thinking_instruction?: string;
  output_format?: string;
  show_thinking?: boolean;
  save_thinking?: boolean;
  max_thinking_tokens?: number;
}

export interface UpdateSettingsData {
  enabled: boolean;
  show_in_ui: boolean;
  save_traces: boolean;
  default_template_id?: string | null;
  max_tokens: number;
}

// ============================================================================
// STATISTICS
// ============================================================================

export interface ThinkingStats {
  total_traces: number;
  total_tokens: number;
  avg_duration_ms: number;
}
