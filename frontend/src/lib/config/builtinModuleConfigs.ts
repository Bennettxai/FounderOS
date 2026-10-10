/**
 * Built-in module registry.
 *
 * Maps the known built-in module slugs to their metadata and config schemas.
 * Any ID that matches a key here is treated as a built-in module (not a UUID).
 */

export type SchemaFieldType = "string" | "boolean" | "number" | "select";

export interface SchemaField {
  key: string;
  label: string;
  type: SchemaFieldType;
  description?: string;
  default?: string | boolean | number;
  options?: Array<{ label: string; value: string }>;
  required?: boolean;
}

export interface BuiltinModuleConfig {
  id: string;
  name: string;
  description: string;
  category: string;
  icon: string;
  version: string;
  configSchema: SchemaField[];
}

const BUILTIN_CONFIGS: Record<string, BuiltinModuleConfig> = {
  dashboard: {
    id: "dashboard",
    name: "Dashboard",
    description: "Main overview — widgets, quick access, and OptimalOS nodes.",
    category: "productivity",
    icon: "layout-dashboard",
    version: "1.0.0",
    configSchema: [
      {
        key: "default_view",
        label: "Default View",
        type: "select",
        description: "Which view loads when you open the dashboard.",
        default: "overview",
        options: [
          { label: "Overview", value: "overview" },
          { label: "Compact", value: "compact" },
          { label: "Focus", value: "focus" },
        ],
      },
      {
        key: "show_quick_actions",
        label: "Show Quick Actions",
        type: "boolean",
        description: "Display the quick action toolbar at the top.",
        default: true,
      },
      {
        key: "refresh_interval",
        label: "Refresh Interval (seconds)",
        type: "number",
        description:
          "How often to refresh dashboard data. Set to 0 to disable.",
        default: 60,
      },
    ],
  },
  chat: {
    id: "chat",
    name: "Chat",
    description: "AI conversation with streaming, agents, and slash commands.",
    category: "communication",
    icon: "message-square",
    version: "1.0.0",
    configSchema: [
      {
        key: "default_model",
        label: "Default AI Model",
        type: "select",
        description: "Which AI model to use by default for new conversations.",
        default: "claude-3-5-sonnet",
        options: [
          { label: "Claude 3.5 Sonnet", value: "claude-3-5-sonnet" },
          { label: "Claude 3 Haiku", value: "claude-3-haiku" },
          { label: "GPT-4o", value: "gpt-4o" },
          { label: "Local (Ollama)", value: "ollama" },
        ],
      },
      {
        key: "streaming",
        label: "Enable Streaming",
        type: "boolean",
        description: "Stream AI responses as they are generated.",
        default: true,
      },
      {
        key: "system_prompt",
        label: "System Prompt Override",
        type: "string",
        description:
          "Optional custom system prompt prepended to all conversations.",
        default: "",
      },
    ],
  },
  tasks: {
    id: "tasks",
    name: "Tasks",
    description:
      "Task management with list and board views, statuses, and assignees.",
    category: "productivity",
    icon: "check-square",
    version: "1.0.0",
    configSchema: [
      {
        key: "default_view",
        label: "Default View",
        type: "select",
        description: "Start in list or board view.",
        default: "list",
        options: [
          { label: "List", value: "list" },
          { label: "Board", value: "board" },
        ],
      },
      {
        key: "show_completed",
        label: "Show Completed Tasks",
        type: "boolean",
        description: "Include completed tasks in the default view.",
        default: false,
      },
      {
        key: "group_by",
        label: "Group By",
        type: "select",
        description: "Default grouping for tasks.",
        default: "status",
        options: [
          { label: "Status", value: "status" },
          { label: "Assignee", value: "assignee" },
          { label: "Priority", value: "priority" },
          { label: "Project", value: "project" },
        ],
      },
    ],
  },
  projects: {
    id: "projects",
    name: "Projects",
    description:
      "Project tracking with tasks, clients, team members, and files.",
    category: "productivity",
    icon: "folder",
    version: "1.0.0",
    configSchema: [
      {
        key: "default_view",
        label: "Default View",
        type: "select",
        description: "Start in grid or list view.",
        default: "grid",
        options: [
          { label: "Grid", value: "grid" },
          { label: "List", value: "list" },
        ],
      },
      {
        key: "show_archived",
        label: "Show Archived Projects",
        type: "boolean",
        description: "Include archived projects in the project list.",
        default: false,
      },
    ],
  },
  team: {
    id: "team",
    name: "Team",
    description: "Team roster — members, roles, capacity, and skills.",
    category: "productivity",
    icon: "users",
    version: "1.0.0",
    configSchema: [
      {
        key: "show_capacity",
        label: "Show Capacity Bars",
        type: "boolean",
        description:
          "Display workload capacity indicators for each team member.",
        default: true,
      },
    ],
  },
  clients: {
    id: "clients",
    name: "Clients",
    description: "Client CRM — companies, contacts, and interactions.",
    category: "finance",
    icon: "briefcase",
    version: "1.0.0",
    configSchema: [
      {
        key: "default_view",
        label: "Default View",
        type: "select",
        description: "Start in cards or table view.",
        default: "cards",
        options: [
          { label: "Cards", value: "cards" },
          { label: "Table", value: "table" },
        ],
      },
    ],
  },
  nodes: {
    id: "nodes",
    name: "Nodes",
    description: "Business graph nodes — OptimalOS knowledge base.",
    category: "analytics",
    icon: "git-branch",
    version: "1.0.0",
    configSchema: [
      {
        key: "auto_sync",
        label: "Auto-Sync with OptimalOS",
        type: "boolean",
        description: "Automatically pull updates from the OptimalOS engine.",
        default: true,
      },
      {
        key: "sync_interval",
        label: "Sync Interval (minutes)",
        type: "number",
        description:
          "How often to sync with the OptimalOS engine when auto-sync is on.",
        default: 5,
      },
    ],
  },
  daily: {
    id: "daily",
    name: "Daily Log",
    description: "Date-based journal with AI extraction.",
    category: "productivity",
    icon: "calendar",
    version: "1.0.0",
    configSchema: [
      {
        key: "ai_extraction",
        label: "AI Extraction",
        type: "boolean",
        description:
          "Automatically extract tasks and decisions from daily entries.",
        default: true,
      },
      {
        key: "default_template",
        label: "Default Template",
        type: "select",
        description:
          "Which template to pre-fill when creating a new daily log.",
        default: "standard",
        options: [
          { label: "Standard", value: "standard" },
          { label: "Minimal", value: "minimal" },
          { label: "Deep Work", value: "deep-work" },
        ],
      },
    ],
  },
  pages: {
    id: "pages",
    name: "Pages",
    description: "Block-based document editor.",
    category: "productivity",
    icon: "file-text",
    version: "1.0.0",
    configSchema: [
      {
        key: "autosave_interval",
        label: "Autosave Interval (seconds)",
        type: "number",
        description: "How frequently to autosave pages. Set to 0 to disable.",
        default: 30,
      },
      {
        key: "default_font",
        label: "Default Font",
        type: "select",
        description: "Default body font for new pages.",
        default: "inter",
        options: [
          { label: "Inter", value: "inter" },
          { label: "Mono", value: "mono" },
          { label: "Serif", value: "serif" },
        ],
      },
    ],
  },
  tables: {
    id: "tables",
    name: "Tables",
    description: "Airtable-like structured data workspace.",
    category: "analytics",
    icon: "table",
    version: "1.0.0",
    configSchema: [
      {
        key: "row_height",
        label: "Default Row Height",
        type: "select",
        description: "How tall rows appear by default.",
        default: "medium",
        options: [
          { label: "Compact", value: "compact" },
          { label: "Medium", value: "medium" },
          { label: "Tall", value: "tall" },
        ],
      },
    ],
  },
  crm: {
    id: "crm",
    name: "CRM",
    description: "Full CRM — companies, deals, and pipelines.",
    category: "finance",
    icon: "trending-up",
    version: "1.0.0",
    configSchema: [
      {
        key: "currency",
        label: "Currency Symbol",
        type: "string",
        description: "Symbol displayed next to deal values.",
        default: "$",
      },
      {
        key: "default_pipeline",
        label: "Default Pipeline View",
        type: "select",
        description: "Which pipeline to show on open.",
        default: "all",
        options: [
          { label: "All Pipelines", value: "all" },
          { label: "My Deals", value: "mine" },
        ],
      },
    ],
  },
};

export function isBuiltinModule(id: string): boolean {
  return id in BUILTIN_CONFIGS;
}

export function getBuiltinConfig(id: string): BuiltinModuleConfig | null {
  return BUILTIN_CONFIGS[id] ?? null;
}

export function getAllBuiltinIds(): string[] {
  return Object.keys(BUILTIN_CONFIGS);
}

/** Returns an array of all built-in module configs in registry order. */
export function getAllBuiltinModules(): BuiltinModuleConfig[] {
  return Object.values(BUILTIN_CONFIGS);
}
