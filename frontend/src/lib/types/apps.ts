/**
 * App Types for BusinessOS Apps Page
 * Based on APPS_PAGE_SPEC.md
 */

export type AppStatus =
  | "active"
  | "draft"
  | "generating"
  | "error"
  | "archived";

export interface App {
  id: string;
  name: string;
  description: string;
  icon?: string; // Lucide icon name or emoji
  status: AppStatus;
  version: number;
  versionCount: number;
  isPinned: boolean;
  createdAt: string;
  updatedAt: string;
  templateId?: string; // If created from template
  errorMessage?: string; // If status is 'error'
  generationProgress?: number; // 0-100 if status is 'generating'
}

export interface AppTemplate {
  id: string;
  name: string;
  description: string;
  icon: string;
  category: string;
}

// Common app templates for empty state
// IDs must match UUIDs from app_templates table in database
export const APP_TEMPLATES: AppTemplate[] = [
  {
    id: "ce320e5a-6159-480a-a793-233299f815a3", // lead_manager
    name: "CRM",
    description: "Track clients and deals",
    icon: "Users",
    category: "crm",
  },
  {
    id: "403a2edc-75b0-4533-ae0e-82d765e3d9f3", // project_kanban
    name: "Tasks",
    description: "Manage your to-dos",
    icon: "CheckSquare",
    category: "productivity",
  },
  {
    id: "7fd6e7b4-1ffa-4e54-b252-28503414b61d", // invoice_generator
    name: "Invoices",
    description: "Create and track invoices",
    icon: "Receipt",
    category: "finance",
  },
  {
    id: "d475b9cb-a3c3-4a23-90b5-4e6272ee354a", // analytics_dashboard
    name: "Analytics",
    description: "Track your metrics",
    icon: "BarChart3",
    category: "analytics",
  },
];
