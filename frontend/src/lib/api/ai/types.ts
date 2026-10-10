
export interface LLMProvider {
  id: string;
  name: string;
  type: 'local' | 'cloud';
  description: string;
  configured: boolean;
  base_url?: string;
  key_source?: 'platform' | 'byok' | 'both' | 'local' | 'none' | string;
  platform_configured?: boolean;
  byok_configured?: boolean;
}

export interface LLMModel {
  id: string;
  name: string;
  provider: string;
  description?: string;
  size?: string;
  family?: string;
}

export interface AIProvidersResponse {
  providers: LLMProvider[];
  active_provider: string;
  default_model: string;
}

export interface AllModelsResponse {
  models: LLMModel[];
  active_provider: string;
  default_model: string;
}

export interface LocalModelsResponse {
  models: LLMModel[];
  provider: string;
  base_url: string;
}

export interface RecommendedModel {
  name: string;
  description: string;
  ram_required: string;
  speed: string;
  quality: string;
}

export interface AISystemInfo {
  total_ram_gb: number;
  available_ram_gb: number;
  platform: string;
  has_gpu: boolean;
  gpu_name?: string;
  recommended_models: RecommendedModel[];
}

export interface AgentInfo {
  id: string;
  name: string;
  description: string;
  prompt: string;
  category: 'general' | 'specialist' | 'system';
}

export interface WarmupResponse {
  status: string;
  model: string;
  provider: string;
  message: string;
}

// MCP Tools Types
export interface Tool {
  name: string;
  description: string;
  input_schema: Record<string, unknown>;
  source: 'builtin' | 'custom';
}

// Discriminated union for ToolResponse - prevents illegal state where both result and error are set
export type ToolResponse =
  | { success: true; result: string; error?: never }
  | { success: false; result?: never; error: string };

// Custom Agents Types
export interface CustomAgent {
  id: string;
  user_id: string;
  name: string;
  display_name: string;
  description?: string;
  avatar?: string;
  system_prompt: string;
  model_preference?: string;
  temperature?: number;
  max_tokens?: number;
  capabilities?: string[];
  tools_enabled?: string[];
  context_sources?: string[];
  thinking_enabled?: boolean;
  streaming_enabled?: boolean;
  apply_personalization?: boolean;
  welcome_message?: string;
  suggested_prompts?: string[];
  category?: string;
  is_active?: boolean;
  is_public?: boolean;
  is_featured?: boolean;
  times_used?: number;
  usage_count?: number;  // Alias for times_used (used in some components)
  created_at: string;
  updated_at: string;
}

export interface CustomAgentsResponse {
  agents: CustomAgent[];
}

// Agent Preset Types
export interface AgentPreset {
  id: string;
  name: string;
  display_name: string;
  description: string;
  category: string;
  avatar?: string;
  system_prompt: string;
  model_preference?: string;
  temperature?: number;
  capabilities?: string[];
  tools_enabled?: string[];
  is_featured?: boolean;
  copy_count?: number;
  created_at: string;
}

export interface AgentPresetsResponse {
  presets: AgentPreset[];
}

// Test/Sandbox Types
export interface AgentTestRequest {
  message: string;
  model?: string;  // Override agent's model
  temperature?: number;  // Override temperature
}

export interface AgentTestResponse {
  response: string;
  tokens_used?: number;
  duration_ms?: number;
  model_used?: string;
}

export interface SandboxTestRequest {
  system_prompt: string;
  test_message: string;  // Backend expects "test_message"
  message?: string;       // Alias for convenience
  model?: string;
  temperature?: number;
}

// Validation Types
export interface AgentValidationResult {
  valid: boolean;
  errors?: string[];
  warnings?: string[];
}

// Filter Types
export interface AgentFilters {
  category?: string;
  search?: string;
  is_active?: boolean;
}

// Create/Update Types
export type CreateAgentData = Omit<CustomAgent, 'id' | 'user_id' | 'created_at' | 'updated_at' | 'times_used'>;
export type UpdateAgentData = Partial<CreateAgentData>;

// Thinking / Chain-of-Thought Types
export type StepType =
  | 'understand'      // Understanding the question
  | 'analyze'         // Analyzing context
  | 'plan'            // Planning approach
  | 'reason'          // Core reasoning
  | 'evaluate'        // Evaluating options
  | 'conclude'        // Drawing conclusion
  | 'verify';         // Verifying answer

export interface ThinkingStep {
  step_number: number;
  step_type: StepType;
  content: string;
  duration_ms: number;
}

export interface ThinkingTrace {
  id: string;
  message_id: string;
  conversation_id: string;
  steps: ThinkingStep[];
  model_used: string;
  template_id?: string;
  duration_ms: number;
  token_count: number;
  created_at: string;
}

export interface TemplateStep {
  name: string;
  prompt: string;
  required: boolean;
}

export interface ReasoningTemplate {
  id: string;
  name: string;
  description: string;
  steps: TemplateStep[];
  is_default: boolean;
  is_active?: boolean;
  usage_count?: number;
  created_at: string;
  updated_at?: string;
}

export interface ThinkingSettings {
  enabled: boolean;
  show_thinking_by_default: boolean;
  default_template_id?: string;
  save_traces: boolean;
  max_steps: number;
}

export interface ReasoningTemplatesResponse {
  templates: ReasoningTemplate[];
}

export type CreateTemplateData = Omit<ReasoningTemplate, 'id' | 'created_at' | 'updated_at' | 'usage_count' | 'is_default'>;
export type UpdateTemplateData = Partial<CreateTemplateData>;
