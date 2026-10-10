import {
  DEFAULT_MODULE_PROFILE,
  getProfileModuleIds,
  resolveWorkspaceModuleProfile,
  type ModuleProfile,
} from "./workspaceModuleProfiles";

export {
  DEFAULT_MODULE_PROFILE,
  getWorkspaceModuleProfileOptions,
  resolveWorkspaceModuleProfile,
  type ModuleProfile,
} from "./workspaceModuleProfiles";

export interface WorkspaceModuleDefinition {
  id: string;
  href: string;
  label: string;
  icon: string;
  group: "Operate" | "FounderOS" | "Business" | "Growth" | "Content" | "Build" | "Manage";
  desktopId?: string;
  adminOnly?: boolean;
}

export interface WorkspaceModuleGroup {
  label: WorkspaceModuleDefinition["group"];
  items: WorkspaceModuleDefinition[];
}

export interface WorkspaceCustomModuleDefinition {
  id: string;
  key: string;
  name: string;
  icon?: string;
  sidebar_group: string;
  sidebar_order: number;
  share_scope: "workspace" | "organization";
}

export interface WorkspaceSidebarItem {
  id: string;
  href: string;
  label: string;
  icon: string;
  adminOnly?: boolean;
  shared?: boolean;
}

export interface WorkspaceSidebarGroup {
  label: string;
  items: WorkspaceSidebarItem[];
}

interface SidebarLayoutSection {
  label: string;
  builtin?: string[];
  custom?: string[];
}

const MODULES: WorkspaceModuleDefinition[] = [
  { id: "dashboard", href: "/dashboard", label: "Command", icon: "command", group: "Operate" },
  { id: "agents", href: "/agents", label: "Agents", icon: "agents", group: "Operate" },
  { id: "knowledge", href: "/knowledge", label: "Knowledge", icon: "knowledge", group: "Operate" },
  { id: "glossary", href: "/glossary", label: "Glossary", icon: "glossary", group: "Operate" },
  { id: "intelligence", href: "/intelligence", label: "Intelligence", icon: "intelligence", group: "Operate" },
  { id: "inbox", href: "/inbox", label: "Inbox", icon: "inbox", group: "Operate" },
  { id: "calendar", href: "/calendar", label: "Calendar", icon: "calendar", group: "Operate" },
  { id: "communications", href: "/communication", label: "Communications", icon: "communication", group: "Operate", desktopId: "communication" },
  // The operator's own modules, ported from FounderOS v1's sidebar (lib/nav.ts) in
  // its order; the pages live under src/routes/(founderos)/os. Each is a 3D OS
  // tile, switched on or off per workspace like any other module.
  { id: "founderos-os", href: "/os", label: "Home", icon: "home", group: "FounderOS" },
  ...([
    ["comms", "Comms", "communication"], ["funnel", "Funnel", "funnel"],
    ["workflows", "Workflows", "workflows"], ["social", "Social", "social"],
    ["content", "Content", "clapperboard"], ["brand-deals", "Brand Deals", "handshake"],
    ["finances", "Finances", "finance"], ["trading", "Trading", "trading"],
    ["adpilot", "AdPilot", "crosshair"],
    ["agents", "Agents", "agents"], ["chats", "Chats", "communication"],
    ["tasks", "Tasks", "tasks"], ["skills", "Skills", "intelligence"],
    ["org", "Org Chart", "org"], ["blueprint", "Blueprint", "blueprint"],
    ["brain", "Brain", "brain"], ["doctor", "Doctor", "doctor"],
    ["integrations", "Connections", "connectors"], ["usage", "Usage", "usage"],
    ["roadmap", "Roadmap", "rhythm"], ["analytics", "Analytics", "analytics"],
    ["reference", "Reference Model", "boards"], ["personas", "Personas", "layers"],
  ] as const).map(([page, label, icon]): WorkspaceModuleDefinition => ({
    id: `founderos-${page}`, href: `/os/${page}`, label, icon, group: "FounderOS",
  })),
  { id: "relationships", href: "/relationships", label: "Relationships", icon: "relationships", group: "Business" },
  { id: "clients", href: "/clients", label: "Clients", icon: "clients", group: "Business" },
  { id: "crm", href: "/crm", label: "CRM", icon: "crm", group: "Business" },
  { id: "projects", href: "/projects", label: "Projects", icon: "projects", group: "Business" },
  { id: "tasks", href: "/tasks", label: "Tasks", icon: "tasks", group: "Business" },
  { id: "rhythm", href: "/rhythm", label: "Rhythm", icon: "rhythm", group: "Business" },
  { id: "pipelines", href: "/pipelines", label: "Pipelines", icon: "pipelines", group: "Business" },
  { id: "offers", href: "/offers", label: "Offers", icon: "offers", group: "Business" },
  { id: "campaigns", href: "/campaigns", label: "Campaigns", icon: "campaigns", group: "Growth" },
  { id: "sites", href: "/sites", label: "Sites", icon: "sites", group: "Growth" },
  { id: "personas", href: "/personas", label: "Personas", icon: "personas", group: "Growth" },
  { id: "my-content", href: "/content?scope=my", label: "My Content", icon: "content", group: "Content", desktopId: "content" },
  { id: "client-content", href: "/content?scope=clients", label: "Client Content", icon: "relationships", group: "Content", desktopId: "content" },
  { id: "apps", href: "/apps", label: "Apps", icon: "apps", group: "Build" },
  { id: "boards", href: "/boards", label: "Boards", icon: "boards", group: "Build" },
  { id: "assets", href: "/assets", label: "Assets", icon: "assets", group: "Build" },
  { id: "deliverables", href: "/deliverables", label: "Deliverables", icon: "deliverables", group: "Build" },
  { id: "engines", href: "/engines", label: "Engines", icon: "engines", group: "Build" },
  { id: "builders", href: "/builders", label: "Builders", icon: "builders", group: "Build" },
  { id: "drive", href: "/drive", label: "Drive", icon: "drive", group: "Build" },
  { id: "finance", href: "/finance", label: "Finance", icon: "finance", group: "Manage" },
  { id: "analytics", href: "/analytics", label: "Analytics", icon: "analytics", group: "Manage" },
  { id: "data", href: "/data", label: "Data", icon: "data", group: "Manage" },
  { id: "team", href: "/team", label: "Team", icon: "team", group: "Manage" },
  { id: "connectors", href: "/connectors", label: "Connectors", icon: "connectors", group: "Manage" },
  { id: "admin", href: "/admin", label: "Admin", icon: "admin", group: "Manage", adminOnly: true },
  { id: "resources", href: "/resources", label: "Resources", icon: "resources", group: "Manage" },
  { id: "notifications", href: "/notifications", label: "Notifications", icon: "notifications", group: "Manage" },
  { id: "help", href: "/help", label: "Help Center", icon: "help", group: "Manage" },
];

const PRIMITIVE_MODULES = [
  "dashboard", "agents", "knowledge", "glossary", "inbox", "calendar",
  "communications", "relationships", "projects", "tasks", "rhythm", "team",
  "connectors", "help",
];

// The operator's modules are on for every workspace: they read the owner's data,
// not the workspace's.
PRIMITIVE_MODULES.push(...MODULES.filter((m) => m.group === "FounderOS").map(({ id }) => id));

const MODULE_IDS = new Set(MODULES.map(({ id }) => id));
const LEGACY_MODULE_IDS: Record<string, string> = {
  command: "dashboard",
  communication: "communications",
  content: "my-content",
};

export function getEnabledModuleIds(settings: Record<string, unknown>): string[] {
  const configured = settings.enabled_builtin_modules;
  const requested = Array.isArray(configured)
    ? configured
    : getProfileModuleIds(
        resolveWorkspaceModuleProfile(settings),
        MODULES.map(({ id }) => id),
        PRIMITIVE_MODULES,
      );

  return [...new Set(requested)]
    .filter((id): id is string => typeof id === "string")
    .map((id) => LEGACY_MODULE_IDS[id] ?? id)
    .filter((id) => MODULE_IDS.has(id));
}

export function getModuleCatalog(): WorkspaceModuleDefinition[] {
  return MODULES.map((module) => ({ ...module }));
}

export function getDesktopModuleIds(settings: Record<string, unknown>): string[] {
  const enabled = new Set(getEnabledModuleIds(settings));

  return [...new Set(
    MODULES
      .filter((module) => enabled.has(module.id))
      .map((module) => module.desktopId ?? module.id),
  )];
}

export function getModuleGroups(settings: Record<string, unknown>): WorkspaceModuleGroup[] {
  const enabled = new Set(getEnabledModuleIds(settings));
  const groups: WorkspaceModuleDefinition["group"][] = [
    "Operate", "FounderOS", "Business", "Growth", "Content", "Build", "Manage",
  ];

  return groups
    .map((label) => ({
      label,
      items: MODULES.filter((module) => module.group === label && enabled.has(module.id)),
    }))
    .filter((group) => group.items.length > 0);
}

function parseSidebarLayout(settings: Record<string, unknown>): SidebarLayoutSection[] {
  if (!Array.isArray(settings.sidebar_layout)) return [];

  return settings.sidebar_layout.flatMap((value) => {
    if (!value || typeof value !== "object") return [];
    const section = value as Record<string, unknown>;
    if (typeof section.label !== "string" || !section.label.trim()) return [];

    const strings = (candidate: unknown): string[] =>
      Array.isArray(candidate)
        ? candidate.filter((item): item is string => typeof item === "string")
        : [];

    return [{
      label: section.label.trim(),
      builtin: strings(section.builtin),
      custom: strings(section.custom),
    }];
  });
}

function customSidebarItem(module: WorkspaceCustomModuleDefinition): WorkspaceSidebarItem {
  return {
    id: module.key,
    href: `/modules/${module.id}`,
    label: module.name,
    icon: module.icon ?? "box",
    shared: module.share_scope === "organization",
  };
}

export function getWorkspaceSidebarGroups(
  settings: Record<string, unknown>,
  customModules: WorkspaceCustomModuleDefinition[],
): WorkspaceSidebarGroup[] {
  const enabled = new Set(getEnabledModuleIds(settings));
  const builtins = new Map(MODULES.map((module) => [module.id, module]));
  const custom = new Map(customModules.map((module) => [module.key, module]));
  const layout = parseSidebarLayout(settings);

  if (layout.length > 0) {
    return layout.flatMap((section) => {
      const builtinItems = (section.builtin ?? []).flatMap((id) => {
        const normalized = LEGACY_MODULE_IDS[id] ?? id;
        const module = builtins.get(normalized);
        if (!module || !enabled.has(normalized)) return [];
        return [{ ...module }];
      });
      const customItems = (section.custom ?? []).flatMap((key) => {
        const module = custom.get(key);
        return module ? [customSidebarItem(module)] : [];
      });
      const items = [...builtinItems, ...customItems];
      return items.length > 0 ? [{ label: section.label, items }] : [];
    });
  }

  const builtinGroups: WorkspaceSidebarGroup[] = getModuleGroups(settings);
  const customGroups = new Map<string, WorkspaceSidebarItem[]>();
  for (const module of [...customModules].sort((a, b) => a.sidebar_order - b.sidebar_order)) {
    const items = customGroups.get(module.sidebar_group) ?? [];
    items.push(customSidebarItem(module));
    customGroups.set(module.sidebar_group, items);
  }

  return [
    ...builtinGroups,
    ...[...customGroups].map(([label, items]) => ({ label, items })),
  ];
}
