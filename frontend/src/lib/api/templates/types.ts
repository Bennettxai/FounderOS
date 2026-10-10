// App Template Types

export type TemplateCategory =
  | 'crm'
  | 'project_management'
  | 'hr'
  | 'finance'
  | 'marketing'
  | 'operations'
  | 'custom';

export type BusinessType =
  | 'startup'
  | 'small_business'
  | 'enterprise'
  | 'agency'
  | 'consulting'
  | 'ecommerce'
  | 'saas'
  | 'nonprofit';

export type TeamSize = 'solo' | 'small' | 'medium' | 'large';

export type StackType = 'svelte' | 'go' | 'fullstack';

export interface AppTemplate {
  id: string;
  name: string;
  description: string;
  category: TemplateCategory;
  icon_url: string | null;
  preview_image_url: string | null;
  business_types: BusinessType[];
  team_sizes: TeamSize[];
  features: string[];
  is_premium: boolean;
  popularity_score: number;
  created_at: string;
  updated_at: string;
}

export interface AppTemplateRecommendation {
  template: AppTemplate;
  match_score: number;
  reasoning: string[];
}

export interface ListTemplatesParams {
  category?: TemplateCategory;
  business_type?: BusinessType;
  team_size?: TeamSize;
  search?: string;
  sort?: 'popular' | 'newest' | 'name';
  limit?: number;
  offset?: number;
}

export interface ListTemplatesResponse {
  templates: AppTemplate[];
  total: number;
}

// Built-in template config schema
export interface ConfigField {
  type: 'string' | 'number' | 'boolean' | 'select';
  label: string;
  description?: string;
  default: string;
  required: boolean;
  options?: string[];
}

// Built-in template info (returned by /api/app-templates/builtin)
export interface BuiltInTemplateInfo {
  id: string;
  name: string;
  description: string;
  category: string;
  stack_type: StackType;
  config_schema: Record<string, ConfigField>;
  file_count: number;
}

// Generated file from template
export interface GeneratedFile {
  path: string;
  content: string;
  size: number;
}

// Generation result (returned when generation completes synchronously or via SSE)
export interface GenerationResult {
  app_id: string;
  app_name: string;
  template_id: string;
  template_name: string;
  workspace_id: string;
  files: GeneratedFile[];
  total_files: number;
  status: string;
  version_number: string;
  generated_at: string;
}

// Response from POST /app-templates/:id/generate — returns queue_item_id for async tracking
export interface GenerateFromTemplateResponse {
  queue_item_id?: string;
  message: string;
  result?: GenerationResult;
}

// Generate request
export interface GenerateFromTemplateRequest {
  workspace_id: string;
  app_name: string;
  config?: Record<string, string | number | boolean>;
}
