import { getModuleCatalog } from "$lib/config/workspaceModules";

export interface ModuleWindowDefinition {
  url: string;
  title: string;
}

export const MODULE_WINDOWS: Record<string, ModuleWindowDefinition> = {
  // The operator's full platform opens on FounderOS v1 (his sidebar, with the
  // workspace switcher and the BusinessOS primitives kept in it).
  platform: { url: "/os", title: "Business OS" },
  dashboard: { url: "/dashboard", title: "Dashboard" },
  "founderos-os": { url: "/os", title: "FounderOS" },
  chat: { url: "/chat", title: "Chat" },
  tasks: { url: "/tasks", title: "Tasks" },
  projects: { url: "/projects", title: "Projects" },
  team: { url: "/team", title: "Team" },
  contexts: { url: "/knowledge", title: "Knowledge" },
  nodes: { url: "/nodes", title: "Nodes" },
  daily: { url: "/daily", title: "Daily Log" },
  settings: { url: "/settings", title: "Settings" },
  clients: { url: "/clients", title: "Clients" },
  tables: { url: "/tables", title: "Tables" },
  communication: { url: "/communication", title: "Communications" },
  calendar: { url: "/calendar", title: "Calendar" },
  pages: { url: "/knowledge", title: "Knowledge" },
  knowledge: { url: "/knowledge", title: "Knowledge" },
  sites: { url: "/sites", title: "Sites" },
  "ai-settings": { url: "/settings/ai", title: "AI Settings" },
  integrations: { url: "/integrations", title: "Integrations" },
  help: { url: "/help", title: "Help" },
  agents: { url: "/agents", title: "Agents" },
  crm: { url: "/crm", title: "CRM" },
  notifications: { url: "/notifications", title: "Notifications" },
  profile: { url: "/profile", title: "Profile" },
  "voice-notes": { url: "/voice-notes", title: "Voice Notes" },
  usage: { url: "/usage", title: "Usage" },
  code: { url: "/code", title: "Code" },
  analytics: { url: "/analytics", title: "Analytics" },
  modules: { url: "/settings", title: "Modules" },
  admin: { url: "/admin", title: "Platform Admin" },
  computer: { url: "/computer", title: "Computer" },
  engine: { url: "/engine", title: "Optimal Engine" },
  engines: { url: "/engines", title: "Engines" },
  relationships: { url: "/relationships", title: "Relationships" },
  rhythm: { url: "/rhythm", title: "Rhythm" },
  inbox: { url: "/inbox", title: "Inbox" },
  offers: { url: "/offers", title: "Offers" },
  campaigns: { url: "/campaigns", title: "Campaigns" },
  pipelines: { url: "/pipelines", title: "Pipelines" },
  deliverables: { url: "/deliverables", title: "Deliverables" },
  connectors: { url: "/connectors", title: "Connectors" },
  apps: { url: "/apps", title: "Apps" },
  drive: { url: "/drive", title: "Drive" },
  content: { url: "/content", title: "Content" },
  personas: { url: "/personas", title: "Personas" },
  glossary: { url: "/glossary", title: "Glossary" },
  finance: { url: "/finance", title: "Finance" },
  boards: { url: "/boards", title: "Boards" },
  assets: { url: "/assets", title: "Assets" },
  builders: { url: "/builders", title: "Builders" },
  intelligence: { url: "/intelligence", title: "Intelligence" },
  data: { url: "/data", title: "Data" },
  resources: { url: "/resources", title: "Resources" },
};
// Each FounderOS v1 module opens its own /os page in a window.
for (const m of getModuleCatalog()) {
  if (m.group === "FounderOS" && !MODULE_WINDOWS[m.id]) {
    MODULE_WINDOWS[m.id] = { url: m.href, title: m.label };
  }
}


export function getModuleWindow(module: string): ModuleWindowDefinition | null {
  return MODULE_WINDOWS[module] ?? null;
}
