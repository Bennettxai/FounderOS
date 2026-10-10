// App Profiles API Types - Codebase Analysis

export type FrameworkType = 'react' | 'vue' | 'svelte' | 'angular' | 'nextjs' | 'express' | 'gin' | 'django' | 'rails' | 'unknown';
export type LanguageType = 'typescript' | 'javascript' | 'go' | 'python' | 'ruby' | 'java' | 'rust' | 'unknown';

export interface ApplicationProfile {
  id: string;
  user_id: string;
  name: string;
  path: string;
  description?: string;
  app_type: string;
  primary_language: LanguageType;
  frameworks: string[];
  tech_stack: TechStackInfo;
  structure: ProjectStructure;
  components: ComponentInfo[];
  endpoints: EndpointInfo[];
  modules: ModuleInfo[];
  dependencies: DependencyInfo[];
  analysis_status: 'pending' | 'analyzing' | 'complete' | 'error';
  last_analyzed_at?: string;
  file_count: number;
  line_count: number;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface TechStackInfo {
  languages: Array<{
    name: string;
    percentage: number;
    file_count: number;
  }>;
  frameworks: Array<{
    name: string;
    version?: string;
    category: string;
  }>;
  tools: Array<{
    name: string;
    purpose: string;
  }>;
  databases: string[];
  cloud_services: string[];
}

export interface ProjectStructure {
  root_path: string;
  directories: Array<{
    path: string;
    purpose: string;
    file_count: number;
  }>;
  key_files: Array<{
    path: string;
    type: string;
    description: string;
  }>;
  patterns: string[];
}

export interface ComponentInfo {
  name: string;
  type: string;
  path: string;
  dependencies: string[];
  exports: string[];
  description?: string;
}

export interface EndpointInfo {
  method: string;
  path: string;
  handler: string;
  file: string;
  description?: string;
  parameters?: Array<{
    name: string;
    type: string;
    required: boolean;
  }>;
}

export interface ModuleInfo {
  name: string;
  path: string;
  type: string;
  exports: string[];
  imports: string[];
  line_count: number;
}

export interface DependencyInfo {
  name: string;
  version: string;
  type: 'production' | 'development';
  category: string;
}

export interface ProfileListItem {
  id: string;
  name: string;
  path: string;
  app_type: string;
  primary_language: LanguageType;
  frameworks: string[];
  analysis_status: string;
  last_analyzed_at?: string;
  created_at: string;
}

export interface AnalyzeCodebaseInput {
  path: string;
  name?: string;
  description?: string;
  include_patterns?: string[];
  exclude_patterns?: string[];
}
