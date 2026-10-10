/**
 * Desktop 3D Module Registry
 * Static lookup tables for module metadata and dynamic module colors.
 * These are not reactive — they are plain records mutated only by addModule/removeModule.
 */

import { getModuleCatalog } from "$lib/config/workspaceModules";

// Module metadata for built-in modules
export const MODULE_INFO: Record<
  string,
  { title: string; color: string; icon: string }
> = {
  dashboard: { title: "Dashboard", color: "#1E88E5", icon: "grid" },
  "founderos-os": { title: "FounderOS", color: "#0a0a0a", icon: "grid" },
  admin: { title: "Admin", color: "#F59E0B", icon: "shield" },
  agents: { title: "Agents", color: "#4F46E5", icon: "bot" },
  analytics: { title: "Analytics", color: "#0EA5E9", icon: "bar-chart" },
  apps: { title: "Apps", color: "#2563EB", icon: "layout-grid" },
  assets: { title: "Assets", color: "#7C3AED", icon: "image" },
  boards: { title: "Boards", color: "#2563EB", icon: "columns" },
  builders: { title: "Builders", color: "#8B5CF6", icon: "hammer" },
  calendar: { title: "Calendar", color: "#EF4444", icon: "calendar" },
  campaigns: { title: "Campaigns", color: "#F97316", icon: "megaphone" },
  clients: { title: "Clients", color: "#0891B2", icon: "briefcase" },
  communication: {
    title: "Communications",
    color: "#3B82F6",
    icon: "message-square",
  },
  connectors: { title: "Connectors", color: "#F59E0B", icon: "link" },
  content: { title: "Content", color: "#14B8A6", icon: "file-text" },
  crm: { title: "CRM", color: "#0D9488", icon: "trending-up" },
  data: { title: "Data", color: "#64748B", icon: "database" },
  deliverables: {
    title: "Deliverables",
    color: "#10B981",
    icon: "clipboard-check",
  },
  drive: { title: "Drive", color: "#2563EB", icon: "folder" },
  engines: { title: "Engines", color: "#7C3AED", icon: "cpu" },
  finance: { title: "Finance", color: "#10B981", icon: "wallet" },
  glossary: { title: "Glossary", color: "#6366F1", icon: "book-a" },
  help: { title: "Help", color: "#607D8B", icon: "help-circle" },
  inbox: { title: "Inbox", color: "#0891B2", icon: "inbox" },
  intelligence: { title: "Intelligence", color: "#7C3AED", icon: "sparkles" },
  knowledge: { title: "Knowledge", color: "#FF6F00", icon: "book-open" },
  offers: { title: "Offers", color: "#FB923C", icon: "gift" },
  personas: { title: "Personas", color: "#EC4899", icon: "user-round" },
  pipelines: { title: "Pipelines", color: "#EAB308", icon: "workflow" },
  projects: { title: "Projects", color: "#8E24AA", icon: "folder" },
  relationships: { title: "Relationships", color: "#06B6D4", icon: "users" },
  resources: { title: "Resources", color: "#059669", icon: "library" },
  rhythm: { title: "Rhythm", color: "#F43F5E", icon: "activity" },
  settings: { title: "Settings", color: "#78909C", icon: "settings" },
  sites: { title: "Sites", color: "#0EA5E9", icon: "globe" },
  tasks: { title: "Tasks", color: "#FB8C00", icon: "check-square" },
  team: { title: "Team", color: "#00ACC1", icon: "users" },
  notifications: { title: "Notifications", color: "#D32F2F", icon: "bell" },
};
// The operator's modules (sidebar group "FounderOS"): black tiles, one per page.
for (const m of getModuleCatalog()) {
  if (m.group === "FounderOS" && !MODULE_INFO[m.id]) {
    MODULE_INFO[m.id] = { title: m.label, color: "#0a0a0a", icon: "grid" };
  }
}


// Category-based colors for dynamically added modules
export const DYNAMIC_MODULE_COLORS: Record<string, string> = {
  finance: "#10b981",
  communication: "#3b82f6",
  productivity: "#a855f7",
  analytics: "#f97316",
  ecommerce: "#ec4899",
  crm: "#06b6d4",
  hr: "#6366f1",
  inventory: "#f59e0b",
  marketing: "#f43f5e",
  project: "#14b8a6",
  general: "#8B5CF6",
};
