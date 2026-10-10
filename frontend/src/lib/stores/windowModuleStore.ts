// Window Module Store - Module configurations and module registry
// Contains moduleDefaults (the known module dimension/title configs) and
// the registerDeployedApp / unregisterDeployedApp methods for dynamic OSA apps.

import type {
  WindowStoreShape,
  WindowState,
  DesktopIcon,
  DesktopSpace,
  CustomIconConfig,
} from "./desktopTypes";
import { saveSettings } from "./desktopPersistence";

// Default module configurations
export const moduleDefaults: Record<
  string,
  {
    title: string;
    width: number;
    height: number;
    minWidth: number;
    minHeight: number;
  }
> = {
  platform: {
    title: "Business OS",
    width: 1200,
    height: 800,
    minWidth: 800,
    minHeight: 600,
  },
  dashboard: {
    title: "Command",
    width: 1000,
    height: 700,
    minWidth: 600,
    minHeight: 400,
  },
  chat: {
    title: "Chat",
    width: 900,
    height: 650,
    minWidth: 400,
    minHeight: 300,
  },
  agents: {
    title: "Agents",
    width: 1000,
    height: 700,
    minWidth: 600,
    minHeight: 420,
  },
  knowledge: {
    title: "Knowledge",
    width: 1050,
    height: 720,
    minWidth: 650,
    minHeight: 450,
  },
  intelligence: {
    title: "Intelligence",
    width: 1000,
    height: 700,
    minWidth: 600,
    minHeight: 420,
  },
  inbox: {
    title: "Inbox",
    width: 900,
    height: 650,
    minWidth: 500,
    minHeight: 380,
  },
  calendar: {
    title: "Calendar",
    width: 1000,
    height: 700,
    minWidth: 650,
    minHeight: 450,
  },
  relationships: {
    title: "Relationships",
    width: 1000,
    height: 700,
    minWidth: 600,
    minHeight: 420,
  },
  tasks: {
    title: "Tasks",
    width: 800,
    height: 600,
    minWidth: 400,
    minHeight: 300,
  },
  projects: {
    title: "Projects",
    width: 900,
    height: 650,
    minWidth: 500,
    minHeight: 400,
  },
  team: {
    title: "Team",
    width: 850,
    height: 600,
    minWidth: 400,
    minHeight: 300,
  },
  clients: {
    title: "Clients",
    width: 1000,
    height: 700,
    minWidth: 600,
    minHeight: 400,
  },
  rhythm: {
    title: "Rhythm",
    width: 900,
    height: 650,
    minWidth: 500,
    minHeight: 380,
  },
  pipelines: {
    title: "Pipelines",
    width: 1000,
    height: 700,
    minWidth: 600,
    minHeight: 420,
  },
  offers: {
    title: "Offers",
    width: 950,
    height: 680,
    minWidth: 560,
    minHeight: 400,
  },
  campaigns: {
    title: "Campaigns",
    width: 1000,
    height: 700,
    minWidth: 600,
    minHeight: 420,
  },
  sites: {
    title: "Sites",
    width: 1050,
    height: 740,
    minWidth: 650,
    minHeight: 450,
  },
  personas: {
    title: "Personas",
    width: 950,
    height: 680,
    minWidth: 560,
    minHeight: 400,
  },
  content: {
    title: "ContentOS",
    width: 1000,
    height: 700,
    minWidth: 600,
    minHeight: 420,
  },
  apps: {
    title: "Apps",
    width: 1050,
    height: 740,
    minWidth: 650,
    minHeight: 450,
  },
  "canvas-note": {
    title: "Note",
    width: 320,
    height: 240,
    minWidth: 220,
    minHeight: 160,
  },
  assets: {
    title: "Assets",
    width: 950,
    height: 680,
    minWidth: 560,
    minHeight: 400,
  },
  deliverables: {
    title: "Deliverables",
    width: 1000,
    height: 700,
    minWidth: 600,
    minHeight: 420,
  },
  engines: {
    title: "Engines",
    width: 1100,
    height: 760,
    minWidth: 700,
    minHeight: 480,
  },
  builders: {
    title: "Builders",
    width: 1050,
    height: 740,
    minWidth: 650,
    minHeight: 450,
  },
  drive: {
    title: "Drive",
    width: 950,
    height: 680,
    minWidth: 560,
    minHeight: 400,
  },
  finance: {
    title: "Finance",
    width: 950,
    height: 680,
    minWidth: 560,
    minHeight: 400,
  },
  analytics: {
    title: "Analytics",
    width: 1050,
    height: 740,
    minWidth: 650,
    minHeight: 450,
  },
  data: {
    title: "Data",
    width: 1050,
    height: 740,
    minWidth: 650,
    minHeight: 450,
  },
  tables: {
    title: "Tables",
    width: 1100,
    height: 750,
    minWidth: 700,
    minHeight: 500,
  },
  pages: {
    title: "Pages",
    width: 900,
    height: 650,
    minWidth: 500,
    minHeight: 400,
  },
  contexts: {
    title: "Pages",
    width: 900,
    height: 650,
    minWidth: 500,
    minHeight: 400,
  }, // Legacy alias
  nodes: {
    title: "Nodes",
    width: 1000,
    height: 700,
    minWidth: 600,
    minHeight: 400,
  },
  daily: {
    title: "Daily Log",
    width: 700,
    height: 550,
    minWidth: 350,
    minHeight: 300,
  },
  settings: {
    title: "Settings",
    width: 700,
    height: 550,
    minWidth: 400,
    minHeight: 350,
  },
  communication: {
    title: "Communication",
    width: 1000,
    height: 700,
    minWidth: 600,
    minHeight: 450,
  },
  "ai-settings": {
    title: "AI Settings",
    width: 800,
    height: 600,
    minWidth: 500,
    minHeight: 400,
  },
  integrations: {
    title: "Integrations",
    width: 950,
    height: 700,
    minWidth: 600,
    minHeight: 500,
  },
  connectors: {
    title: "Connectors",
    width: 950,
    height: 700,
    minWidth: 600,
    minHeight: 500,
  },
  resources: {
    title: "Resources",
    width: 900,
    height: 650,
    minWidth: 500,
    minHeight: 380,
  },
  notifications: {
    title: "Notifications",
    width: 760,
    height: 560,
    minWidth: 420,
    minHeight: 340,
  },
  trash: {
    title: "Trash",
    width: 600,
    height: 450,
    minWidth: 300,
    minHeight: 250,
  },
  terminal: {
    title: "Terminal",
    width: 700,
    height: 500,
    minWidth: 400,
    minHeight: 300,
  },
  "desktop-settings": {
    title: "Desktop Settings",
    width: 550,
    height: 500,
    minWidth: 450,
    minHeight: 400,
  },
  admin: {
    title: "Platform Admin",
    width: 1100,
    height: 750,
    minWidth: 800,
    minHeight: 500,
  },
  computer: {
    title: "Computer",
    width: 1100,
    height: 800,
    minWidth: 700,
    minHeight: 500,
  },
  engine: {
    title: "Optimal Engine",
    width: 1200,
    height: 800,
    minWidth: 800,
    minHeight: 500,
  },
  folder: {
    title: "Folder",
    width: 600,
    height: 450,
    minWidth: 300,
    minHeight: 250,
  },
  files: {
    title: "Files",
    width: 900,
    height: 600,
    minWidth: 500,
    minHeight: 400,
  },
  finder: {
    title: "Finder",
    width: 900,
    height: 600,
    minWidth: 500,
    minHeight: 400,
  },
  help: {
    title: "Help",
    width: 900,
    height: 650,
    minWidth: 600,
    minHeight: 450,
  },
};

// Category-specific colors for deployed OSA apps
export const categoryColors: Record<string, { fg: string; bg: string }> = {
  finance: { fg: "#10b981", bg: "#d1fae5" },
  communication: { fg: "#3b82f6", bg: "#dbeafe" },
  productivity: { fg: "#a855f7", bg: "#f3e8ff" },
  analytics: { fg: "#f97316", bg: "#fed7aa" },
  ecommerce: { fg: "#ec4899", bg: "#fce7f3" },
  crm: { fg: "#06b6d4", bg: "#cffafe" },
  hr: { fg: "#6366f1", bg: "#e0e7ff" },
  inventory: { fg: "#f59e0b", bg: "#fef3c7" },
  marketing: { fg: "#f43f5e", bg: "#ffe4e6" },
  project: { fg: "#14b8a6", bg: "#ccfbf1" },
  general: { fg: "#8B5CF6", bg: "#F3E8FF" },
};

function workspaceAppIconConfig(app: {
  logo_url?: string;
  color?: string;
}): CustomIconConfig {
  if (app.logo_url) {
    return {
      type: "image",
      imageUrl: app.logo_url,
      backgroundColor: "#ffffff",
    };
  }

  return {
    type: "lucide",
    lucideName: "AppWindow",
    foregroundColor: app.color || "#111827",
    backgroundColor: "#ffffff",
  };
}

function shouldReplaceWorkspaceAppIcon(icon?: CustomIconConfig): boolean {
  return !icon || (icon.type === "lucide" && icon.lucideName === "AppWindow");
}

function isModuleHidden(module: string, hiddenModules: string[] = []): boolean {
  return hiddenModules.includes(module) ||
    (module.startsWith("workspace-app-") && hiddenModules.includes("workspace-app-*"));
}

const workspaceAppMetadataByModule = new Map<
  string,
  {
    name: string;
    url: string;
    launch_mode: "iframe" | "browser" | "external";
    logo_url?: string;
    color?: string;
  }
>();

export function getWorkspaceAppMetadata(module: string) {
  return workspaceAppMetadataByModule.get(module);
}

export function listWorkspaceAppMetadata() {
  return Array.from(workspaceAppMetadataByModule.entries()).map(([module, metadata]) => ({
    module,
    ...metadata,
  }));
}

export function getWorkspaceAppIconConfig(app: {
  logo_url?: string;
  color?: string;
}): CustomIconConfig {
  return workspaceAppIconConfig(app);
}

function pruneWorkspaceAppsFromSpace(
  space: DesktopSpace,
  visibleModuleIds: Set<string>,
  appByModule: Map<string, { name: string; url: string; launch_mode: "iframe" | "browser" | "external"; logo_url?: string; color?: string; show_in_dock?: boolean }>,
): DesktopSpace {
  const explicitIconModules = new Set(
    space.desktopIcons
      .filter((icon) => icon.module.startsWith("workspace-app-"))
      .map((icon) => icon.module),
  );
  const desktopIcons = space.desktopIcons
    .filter(
      (icon) =>
        !icon.module.startsWith("workspace-app-") ||
        (visibleModuleIds.has(icon.module) &&
          (explicitIconModules.has(icon.module) ||
            !isModuleHidden(icon.module, space.hiddenModules))),
    )
    .map((icon) => {
      const app = appByModule.get(icon.module);
      if (!app) return icon;
      return {
        ...icon,
        label: app.name,
        appUrl: app.url,
        launchMode: app.launch_mode,
        customIcon: shouldReplaceWorkspaceAppIcon(icon.customIcon)
          ? workspaceAppIconConfig(app)
          : icon.customIcon,
      };
    });

  const dockModules = new Set(
    [...appByModule.entries()]
      .filter(([, app]) => app.show_in_dock)
      .map(([module]) => module),
  );
  const dockPinnedItems = [
    ...space.dockPinnedItems.filter(
      (item) =>
        !item.startsWith("workspace-app-") ||
        (visibleModuleIds.has(item) &&
          dockModules.has(item) &&
          (explicitIconModules.has(item) || !isModuleHidden(item, space.hiddenModules))),
    ),
    ...[...dockModules].filter(
      (module) =>
        !space.dockPinnedItems.includes(module) &&
        space.desktopIcons.length > 0 &&
        !isModuleHidden(module, space.hiddenModules),
    ),
  ];

  const windows = space.windows
    .filter(
      (w) =>
        !w.module.startsWith("workspace-app-") ||
        (visibleModuleIds.has(w.module) &&
          (explicitIconModules.has(w.module) || !isModuleHidden(w.module, space.hiddenModules))),
    )
    .map((w) => {
      const app = appByModule.get(w.module);
      if (!app) return w;
      return {
        ...w,
        title: app.name,
        data: {
          ...w.data,
          url: app.url,
          launchMode: app.launch_mode,
        },
      };
    });
  const windowIds = new Set(windows.map((w) => w.id));

  return {
    ...space,
    desktopIcons,
    dockPinnedItems,
    hiddenModules: space.hiddenModules || [],
    windows,
    windowOrder: space.windowOrder.filter((id) => windowIds.has(id)),
    focusedWindowId:
      space.focusedWindowId && windowIds.has(space.focusedWindowId)
        ? space.focusedWindowId
        : space.windowOrder.filter((id) => windowIds.has(id)).at(-1) ?? null,
    updatedAt: new Date().toISOString(),
  };
}

// Creates the module registry methods that operate on the window store
export function createModuleMethods(
  update: (fn: (state: WindowStoreShape) => WindowStoreShape) => void,
  persistOpenWindows?: (windows: WindowState[]) => void,
) {
  return {
    syncWorkspaceApps: (
      apps: Array<{
        id: string;
        name: string;
        url: string;
        launch_mode: "iframe" | "browser" | "external";
        icon?: string;
        color?: string;
        logo_url?: string;
        show_on_desktop: boolean;
        show_in_dock?: boolean;
      }>,
    ) => {
      update((state) => {
        for (const app of apps) {
          const moduleId = `workspace-app-${app.id}`;
          workspaceAppMetadataByModule.set(moduleId, {
            name: app.name,
            url: app.url,
            launch_mode: app.launch_mode,
            logo_url: app.logo_url,
            color: app.color,
          });
          moduleDefaults[moduleId] = {
            title: app.name,
            width: 1100,
            height: 760,
            minWidth: 520,
            minHeight: 360,
          };
        }

        const visibleApps = apps.filter((app) => app.show_on_desktop);
        const dockApps = apps.filter((app) => app.show_on_desktop && app.show_in_dock);
        const visibleModuleIds = new Set(
          visibleApps.map((app) => `workspace-app-${app.id}`),
        );
        const dockModuleIds = new Set(dockApps.map((app) => `workspace-app-${app.id}`));

        let desktopIcons = state.desktopIcons.filter(
          (icon) =>
            !icon.module.startsWith("workspace-app-") ||
            (visibleModuleIds.has(icon.module) ||
              (!isModuleHidden(icon.module, state.hiddenModules) && visibleModuleIds.has(icon.module))),
        );
        const explicitIconModules = new Set(
          desktopIcons
            .filter((icon) => icon.module.startsWith("workspace-app-"))
            .map((icon) => icon.module),
        );

        const existingByModule = new Map(
          desktopIcons.map((icon) => [icon.module, icon]),
        );

        for (const app of visibleApps) {
          const moduleId = `workspace-app-${app.id}`;

          const existing = existingByModule.get(moduleId);
          if (existing) {
            desktopIcons = desktopIcons.map((icon) =>
              icon.module === moduleId
                ? {
                    ...icon,
                    label: app.name,
                    appUrl: app.url,
                    launchMode: app.launch_mode,
                    customIcon: shouldReplaceWorkspaceAppIcon(icon.customIcon)
                      ? workspaceAppIconConfig(app)
                      : icon.customIcon,
                  }
                : icon,
            );
            continue;
          }

          const shouldAutoAddToDesktop =
            state.desktopIcons.length > 0 && !isModuleHidden(moduleId, state.hiddenModules);
          if (!shouldAutoAddToDesktop) continue;

          const rightSideIcons = desktopIcons.filter((icon) => icon.x === -1);
          const nextY =
            rightSideIcons.length > 0
              ? Math.max(...rightSideIcons.map((icon) => icon.y)) + 1
              : 0;

          desktopIcons = [
            ...desktopIcons,
            {
              id: `icon-${moduleId}`,
              module: moduleId,
              label: app.name,
              x: -1,
              y: nextY,
              type: "app",
              appUrl: app.url,
              launchMode: app.launch_mode,
              customIcon: workspaceAppIconConfig(app),
            },
          ];
        }

        const appByModule = new Map(
          visibleApps.map((app) => [`workspace-app-${app.id}`, app]),
        );
        const dockPinnedItems = [
          ...state.dockPinnedItems.filter(
            (item) =>
              !item.startsWith("workspace-app-") ||
              (visibleModuleIds.has(item) &&
                dockModuleIds.has(item) &&
                (explicitIconModules.has(item) || !isModuleHidden(item, state.hiddenModules))),
          ),
          ...[...dockModuleIds].filter(
            (module) =>
              !state.dockPinnedItems.includes(module) &&
              state.desktopIcons.length > 0 &&
              !isModuleHidden(module, state.hiddenModules),
          ),
        ];
        const windows = state.windows
          .filter(
            (w) =>
              !w.module.startsWith("workspace-app-") ||
              (visibleModuleIds.has(w.module) &&
                (explicitIconModules.has(w.module) || !isModuleHidden(w.module, state.hiddenModules))),
          )
          .map((w) => {
            const app = appByModule.get(w.module);
            if (!app) return w;
            return {
              ...w,
              title: app.name,
              data: {
                ...w.data,
                url: app.url,
                launchMode: app.launch_mode,
              },
            };
          });
        const windowIds = new Set(windows.map((w) => w.id));
        const desktopSpaces = state.desktopSpaces.map((space) => {
          if (space.id === state.activeDesktopId) {
            return {
              ...space,
              desktopIcons,
              dockPinnedItems,
              hiddenModules: state.hiddenModules,
              windows,
              windowOrder: state.windowOrder.filter((id) => windowIds.has(id)),
              focusedWindowId: windowIds.has(state.focusedWindowId ?? "")
                ? state.focusedWindowId
                : state.windowOrder.filter((id) => windowIds.has(id)).at(-1) ?? null,
              updatedAt: new Date().toISOString(),
            };
          }
          return pruneWorkspaceAppsFromSpace(space, visibleModuleIds, appByModule);
        });

        const newState = {
          ...state,
          desktopIcons,
          dockPinnedItems,
          hiddenModules: state.hiddenModules,
          desktopSpaces,
          windows,
          windowOrder: state.windowOrder.filter((id) => windowIds.has(id)),
          focusedWindowId: windowIds.has(state.focusedWindowId ?? "")
            ? state.focusedWindowId
            : state.windowOrder.filter((id) => windowIds.has(id)).at(-1) ?? null,
        };

        saveSettings(newState);
        persistOpenWindows?.(windows);
        return newState;
      });
    },

    placeWorkspaceApp: (app: {
      id: string;
      name: string;
      url: string;
      launch_mode: "iframe" | "browser" | "external";
      icon?: string;
      color?: string;
      logo_url?: string;
      show_on_desktop: boolean;
      show_in_dock?: boolean;
    }) => {
      update((state) => {
        const moduleId = `workspace-app-${app.id}`;
        workspaceAppMetadataByModule.set(moduleId, {
          name: app.name,
          url: app.url,
          launch_mode: app.launch_mode,
          logo_url: app.logo_url,
          color: app.color,
        });
        moduleDefaults[moduleId] = {
          title: app.name,
          width: 1100,
          height: 760,
          minWidth: 520,
          minHeight: 360,
        };

        const baseIcons = app.show_on_desktop
          ? state.desktopIcons
          : state.desktopIcons.filter((icon) => icon.module !== moduleId);
        const existing = baseIcons.find((icon) => icon.module === moduleId);
        const sameColumn = baseIcons.filter((icon) => icon.x === -1 && icon.module !== moduleId);
        const nextY =
          sameColumn.length > 0
            ? Math.max(...sameColumn.map((icon) => icon.y)) + 1
            : 0;
        const nextIcon: DesktopIcon = existing
          ? {
              ...existing,
              label: app.name,
              appUrl: app.url,
              launchMode: app.launch_mode,
              customIcon: shouldReplaceWorkspaceAppIcon(existing.customIcon)
                ? workspaceAppIconConfig(app)
                : existing.customIcon,
            }
          : {
              id: `icon-${moduleId}`,
              module: moduleId,
              label: app.name,
              x: -1,
              y: nextY,
              type: "app",
              appUrl: app.url,
              launchMode: app.launch_mode,
              customIcon: workspaceAppIconConfig(app),
            };
        const desktopIcons = app.show_on_desktop
          ? existing
            ? baseIcons.map((icon) => (icon.module === moduleId ? nextIcon : icon))
            : [...baseIcons, nextIcon]
          : baseIcons;
        const dockPinnedItems = app.show_on_desktop && app.show_in_dock
          ? [...state.dockPinnedItems.filter((item) => item !== moduleId), moduleId]
          : state.dockPinnedItems.filter((item) => item !== moduleId);
        const windows = state.windows.map((w) =>
          w.module === moduleId
            ? {
                ...w,
                title: app.name,
                data: {
                  ...w.data,
                  url: app.url,
                  launchMode: app.launch_mode,
                },
              }
            : w,
        );
        const newState = {
          ...state,
          desktopIcons,
          dockPinnedItems,
          hiddenModules: app.show_on_desktop
            ? state.hiddenModules.filter(
                (hiddenModule) => hiddenModule !== moduleId && hiddenModule !== "workspace-app-*",
              )
            : [...new Set([...state.hiddenModules, moduleId])],
          windows,
        };
        saveSettings(newState);
        persistOpenWindows?.(windows);
        return newState;
      });
    },

    // Register a deployed OSA app dynamically
    registerDeployedApp: (app: {
      id: string;
      name: string;
      url: string;
      port: number;
      metadata?: {
        name: string;
        description: string;
        category: string;
        icon: string;
        keywords: string[];
      };
    }) => {
      update((state) => {
        const moduleId = `osa-app-${app.id}`;

        // Check if already registered
        if (moduleDefaults[moduleId]) {
          return state;
        }

        // Use metadata name or fallback to app name
        const displayName = app.metadata?.name || app.name;

        // Add to moduleDefaults
        moduleDefaults[moduleId] = {
          title: displayName,
          width: 1000,
          height: 700,
          minWidth: 600,
          minHeight: 400,
        };

        // Check if desktop icon already exists
        const iconExists = state.desktopIcons.some(
          (icon) => icon.module === moduleId,
        );
        if (iconExists) {
          return state;
        }

        // Find next available position on the right side
        const rightSideIcons = state.desktopIcons.filter(
          (icon) => icon.x === -1,
        );
        const nextY =
          rightSideIcons.length > 0
            ? Math.max(...rightSideIcons.map((icon) => icon.y)) + 1
            : 0;

        const category = app.metadata?.category || "general";
        const colors = categoryColors[category] || categoryColors.general;

        // Create desktop icon with metadata
        const newIcon: DesktopIcon = {
          id: `icon-${moduleId}`,
          module: moduleId,
          label: displayName,
          x: -1, // Right side
          y: nextY,
          type: "app",
          customIcon: {
            type: "lucide",
            lucideName: app.metadata?.icon || "AppWindow",
            foregroundColor: colors.fg,
            backgroundColor: colors.bg,
          },
        };

        const newState = {
          ...state,
          hiddenModules: state.hiddenModules.filter((module) => module !== moduleId),
          desktopIcons: [...state.desktopIcons, newIcon],
        };

        // Save to localStorage
        saveSettings(newState);

        if (import.meta.env.DEV) {
          console.log(
            `[windowStore] Registered OSA app: ${displayName} (${moduleId})`,
          );
        }

        return newState;
      });
    },

    // Unregister a deployed app when it stops
    unregisterDeployedApp: (appId: string) => {
      const moduleId = `osa-app-${appId}`;

      update((state) => {
        // Remove from moduleDefaults
        delete moduleDefaults[moduleId];

        // Remove desktop icon
        const newState = {
          ...state,
          hiddenModules: [...new Set([...state.hiddenModules, moduleId])],
          desktopIcons: state.desktopIcons.filter(
            (icon) => icon.module !== moduleId,
          ),
          // Close any open windows for this app
          windows: state.windows.filter((w) => w.module !== moduleId),
          windowOrder: state.windowOrder.filter((id) => {
            const win = state.windows.find((w) => w.id === id);
            return win?.module !== moduleId;
          }),
        };

        saveSettings(newState);
        persistOpenWindows?.(newState.windows);

        if (import.meta.env.DEV)
          console.log(`[windowStore] Unregistered OSA app: ${moduleId}`);

        return newState;
      });
    },
  };
}
