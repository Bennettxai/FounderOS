// Custom Modules System Types

export type ModuleCategory =
  | "productivity"
  | "communication"
  | "finance"
  | "analytics"
  | "automation"
  | "integration"
  | "utilities"
  | "custom";

export type ModuleActionType = "function" | "api" | "workflow";

export type ModuleVisibility = "private" | "workspace" | "public";

export type SharePermission = "view" | "install" | "modify" | "reshare";

export interface ModuleAction {
  name: string;
  description: string;
  type: ModuleActionType;
  parameters: Record<string, unknown>;
  returns: Record<string, unknown>;
}

export interface ModuleManifest {
  name: string;
  version: string;
  description: string;
  author: string;
  category: ModuleCategory;
  icon?: string;
  actions: ModuleAction[];
  config_schema?: Record<string, unknown>;
  dependencies?: string[];
  permissions?: string[];
}

export interface CustomModule {
  id: string;
  workspace_id: string;
  creator_id: string;
  name: string;
  slug?: string;
  description: string;
  category: ModuleCategory;
  icon: string | null;
  manifest: ModuleManifest;
  config: Record<string, unknown> | null;
  config_schema: Record<string, unknown> | null;
  share_scope?: "workspace" | "organization";
  visibility: ModuleVisibility;
  is_active: boolean;
  version: string;
  install_count: number;
  star_count: number;
  created_at: string;
  updated_at: string;
  creator_name?: string;
}

export interface ModuleVersion {
  id: string;
  module_id: string;
  version: string;
  manifest: ModuleManifest;
  changelog: string | null;
  created_at: string;
}

export interface ModuleInstallation {
  id: string;
  module_id: string;
  workspace_id: string;
  user_id: string;
  config: Record<string, unknown> | null;
  is_active: boolean;
  installed_at: string;
  updated_at: string;
}

export interface ModuleShare {
  id: string;
  module_id: string;
  shared_by_user_id: string;
  shared_with_user_id: string | null;
  shared_with_workspace_id: string | null;
  permissions: SharePermission[];
  created_at: string;
}

export interface ModuleFilters {
  category: ModuleCategory | null;
  search: string;
  sort: "popular" | "newest" | "name" | "installs";
  visibility: ModuleVisibility | null;
}

export interface CreateModuleData {
  name: string;
  description: string;
  category: ModuleCategory;
  icon?: string;
  manifest: ModuleManifest;
  config?: Record<string, unknown>;
  config_schema?: Record<string, unknown>;
  visibility?: ModuleVisibility;
}

export interface UpdateModuleData {
  name?: string;
  description?: string;
  category?: ModuleCategory;
  icon?: string;
  manifest?: ModuleManifest;
  config_schema?: Record<string, unknown>;
  visibility?: ModuleVisibility;
  is_active?: boolean;
}

export interface ShareModuleData {
  shared_with_user_id?: string;
  shared_with_workspace_id?: string;
  permissions: SharePermission[];
}

export interface ModuleExportData {
  module: CustomModule;
  versions: ModuleVersion[];
  metadata: {
    exported_at: string;
    exported_by: string;
    version: string;
  };
}

// UI-specific types
export interface ModuleCardProps {
  module: CustomModule;
  onClick?: () => void;
}

export interface ActionBuilderItem {
  id: string;
  name: string;
  description: string;
  type: ModuleActionType;
  parameters: Record<string, unknown>;
  returns: Record<string, unknown>;
}

export const categoryLabels: Record<ModuleCategory, string> = {
  productivity: "Productivity",
  communication: "Communication",
  finance: "Finance",
  analytics: "Analytics",
  automation: "Automation",
  integration: "Integration",
  utilities: "Utilities",
  custom: "Custom",
};

export const categoryHexColors: Record<ModuleCategory, string> = {
  productivity: "#3b82f6",
  communication: "#a855f7",
  finance: "#10b981",
  analytics: "#f97316",
  automation: "#ec4899",
  integration: "#6366f1",
  utilities: "#6b7280",
  custom: "#06b6d4",
};
