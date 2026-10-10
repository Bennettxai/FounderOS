// Desktop Persistence - localStorage save/load and JSON config export/import
import { browser } from "$app/environment";
import type {
  WindowStoreShape,
  DesktopIcon,
  DesktopConfig,
  DesktopFolder,
  DesktopSpace,
  WindowState,
} from "./desktopTypes";

// Storage key for persisting desktop settings
export const STORAGE_KEY = "businessos_desktop_settings";
export const DEFAULT_DESKTOP_ID = "personal";

// Desktop config schema version for backwards compatibility
export const CONFIG_VERSION = "1.2.0";

const MAX_RESTORED_WINDOWS_PER_DESKTOP = 24;
const MAX_RESTORED_ICONS_PER_DESKTOP = 96;
const MAX_WINDOW_WIDTH = 2200;
const MAX_WINDOW_HEIGHT = 1600;
const MIN_WINDOW_WIDTH = 240;
const MIN_WINDOW_HEIGHT = 160;
const CANVAS_COORDINATE_LIMIT = 50000;

// Initial desktop icon positions (right side, top to bottom)
export const initialDesktopIcons: DesktopIcon[] = [
  { id: "icon-platform", module: "platform", label: "Business OS", x: 0, y: 0 }, // Top left - full platform
  { id: "icon-founderos-os", module: "founderos-os", label: "FounderOS", x: 0, y: 1 }, // The operator's ported operator console (/os)
  { id: "icon-command", module: "dashboard", label: "Command", x: -1, y: 0 },
  {
    id: "icon-knowledge",
    module: "knowledge",
    label: "Knowledge",
    x: -1,
    y: 1,
  },
  { id: "icon-agents", module: "agents", label: "Agents", x: -1, y: 2 },
  { id: "icon-projects", module: "projects", label: "Projects", x: -1, y: 3 },
  { id: "icon-tasks", module: "tasks", label: "Tasks", x: -1, y: 4 },
  {
    id: "icon-pipelines",
    module: "pipelines",
    label: "Pipelines",
    x: -1,
    y: 5,
  },
  { id: "icon-offers", module: "offers", label: "Offers", x: -1, y: 6 },
  {
    id: "icon-deliverables",
    module: "deliverables",
    label: "Deliverables",
    x: -1,
    y: 7,
  },
  {
    id: "icon-connectors",
    module: "connectors",
    label: "Connectors",
    x: -1,
    y: 8,
  },
  { id: "icon-terminal", module: "terminal", label: "Terminal", x: -2, y: 0 },
  { id: "icon-inbox", module: "inbox", label: "Inbox", x: -2, y: 1 },
  { id: "icon-calendar", module: "calendar", label: "Calendar", x: -2, y: 2 },
  {
    id: "icon-relationships",
    module: "relationships",
    label: "Relationships",
    x: -2,
    y: 3,
  },
  { id: "icon-rhythm", module: "rhythm", label: "Rhythm", x: -2, y: 4 },
  {
    id: "icon-growth",
    module: "campaigns",
    label: "Campaigns",
    x: -2,
    y: 5,
  },
  { id: "icon-sites", module: "sites", label: "Sites", x: -2, y: 6 },
  { id: "icon-apps", module: "apps", label: "Apps", x: -2, y: 7 },
  { id: "icon-drive", module: "drive", label: "Drive", x: -2, y: 8 },
  { id: "icon-boards", module: "boards", label: "Boards", x: -3, y: 0 },
  { id: "icon-team", module: "team", label: "Team", x: -3, y: 1 },
  { id: "icon-trash", module: "trash", label: "Trash", x: -1, y: -1 }, // Bottom right
];

type SanitizeDesktopIconOptions = {
  allowEmpty?: boolean;
  mergeDefaults?: boolean;
};

function sanitizeDesktopIcons(
  desktopIcons: DesktopIcon[] | undefined,
  options: SanitizeDesktopIconOptions = {},
): DesktopIcon[] {
  const allowEmpty = options.allowEmpty ?? false;
  const mergeDefaults = options.mergeDefaults ?? true;
  let icons = Array.isArray(desktopIcons)
    ? desktopIcons
    : initialDesktopIcons;

  if (icons.length === 0 && !allowEmpty) {
    icons = initialDesktopIcons;
  }

  const savedIconIds = new Set(icons.map((i: DesktopIcon) => i.id));
  const newIcons = mergeDefaults
    ? initialDesktopIcons.filter((icon) => !savedIconIds.has(icon.id))
    : [];
  if (newIcons.length > 0) icons = [...icons, ...newIcons];

  const knownModules = new Set(initialDesktopIcons.map((i) => i.module));
  icons = icons.filter(
    (icon: DesktopIcon) =>
      knownModules.has(icon.module) ||
      icon.module === "finder" ||
      icon.module.startsWith("workspace-app-"),
  );

  const defaultLabels = new Map(initialDesktopIcons.map((i) => [i.id, i.label]));
  return icons.slice(0, MAX_RESTORED_ICONS_PER_DESKTOP).map((icon: DesktopIcon) => {
    const defaultLabel = defaultLabels.get(icon.id);
    const x = Number.isFinite(icon.x)
      ? clampNumber(icon.x, -CANVAS_COORDINATE_LIMIT, CANVAS_COORDINATE_LIMIT)
      : 0;
    const y = Number.isFinite(icon.y)
      ? clampNumber(icon.y, -CANVAS_COORDINATE_LIMIT, CANVAS_COORDINATE_LIMIT)
      : 0;
    if (defaultLabel && icon.label !== defaultLabel) {
      return { ...icon, label: defaultLabel, x, y };
    }
    return { ...icon, x, y };
  });
}

function clampNumber(value: number, min: number, max: number) {
  return Math.max(min, Math.min(value, max));
}

function sanitizePreviousBounds(bounds: WindowState["previousBounds"]): WindowState["previousBounds"] {
  if (!bounds) return undefined;
  const { x, y, width, height } = bounds;
  if (![x, y, width, height].every(Number.isFinite)) return undefined;
  return {
    x: clampNumber(x, -CANVAS_COORDINATE_LIMIT, CANVAS_COORDINATE_LIMIT),
    y: clampNumber(y, -CANVAS_COORDINATE_LIMIT, CANVAS_COORDINATE_LIMIT),
    width: clampNumber(width, MIN_WINDOW_WIDTH, MAX_WINDOW_WIDTH),
    height: clampNumber(height, MIN_WINDOW_HEIGHT, MAX_WINDOW_HEIGHT),
  };
}

function validWindows(windows: unknown): WindowState[] {
  if (!Array.isArray(windows)) return [];
  return windows
    .slice(0, MAX_RESTORED_WINDOWS_PER_DESKTOP)
    .flatMap((w): WindowState[] => {
      const item = w as Partial<WindowState>;
      if (
        typeof item.id !== "string" ||
        typeof item.module !== "string" ||
        typeof item.title !== "string" ||
        typeof item.x !== "number" ||
        typeof item.y !== "number" ||
        typeof item.width !== "number" ||
        typeof item.height !== "number" ||
        ![item.x, item.y, item.width, item.height].every(Number.isFinite)
      ) {
        return [];
      }

      return [{
        id: item.id,
        module: item.module,
        title: item.title,
        x: clampNumber(item.x, -CANVAS_COORDINATE_LIMIT, CANVAS_COORDINATE_LIMIT),
        y: clampNumber(item.y, -CANVAS_COORDINATE_LIMIT, CANVAS_COORDINATE_LIMIT),
        width: clampNumber(item.width, MIN_WINDOW_WIDTH, MAX_WINDOW_WIDTH),
        height: clampNumber(item.height, MIN_WINDOW_HEIGHT, MAX_WINDOW_HEIGHT),
        minWidth: clampNumber(item.minWidth ?? 400, MIN_WINDOW_WIDTH, MAX_WINDOW_WIDTH),
        minHeight: clampNumber(item.minHeight ?? 300, MIN_WINDOW_HEIGHT, MAX_WINDOW_HEIGHT),
        minimized: Boolean(item.minimized),
        maximized: Boolean(item.maximized),
        snapped: item.snapped ?? null,
        previousBounds: sanitizePreviousBounds(item.previousBounds),
        data: typeof item.data === "object" && item.data !== null ? item.data : undefined,
      }];
    });
}

function validFolders(folders: unknown): DesktopFolder[] {
  return Array.isArray(folders) ? folders as DesktopFolder[] : [];
}

function validHiddenModules(hiddenModules: unknown): string[] {
  return Array.isArray(hiddenModules)
    ? [...new Set(hiddenModules.filter((module): module is string => typeof module === "string"))]
    : [];
}

function createDesktopSpace(
  id: string,
  name: string,
  initialDockPinnedItems: string[],
  source?: Partial<DesktopSpace>,
): DesktopSpace {
  const now = new Date().toISOString();
  const windows = validWindows(source?.windows ?? []);
  const windowIds = new Set(windows.map((w) => w.id));

  return {
    id,
    name,
    kind: source?.kind ?? "personal",
    desktopSettings: source?.desktopSettings,
    desktopIcons: sanitizeDesktopIcons(source?.desktopIcons, {
      allowEmpty: Array.isArray(source?.desktopIcons),
      mergeDefaults: !Array.isArray(source?.desktopIcons),
    }),
    dockPinnedItems: Array.isArray(source?.dockPinnedItems) && source.dockPinnedItems.length > 0
      ? source.dockPinnedItems
      : initialDockPinnedItems,
    hiddenModules: validHiddenModules(source?.hiddenModules),
    folders: validFolders(source?.folders),
    windows,
    windowOrder: Array.isArray(source?.windowOrder)
      ? source.windowOrder.filter((id) => windowIds.has(id))
      : windows.map((w) => w.id),
    focusedWindowId:
      typeof source?.focusedWindowId === "string" && windowIds.has(source.focusedWindowId)
        ? source.focusedWindowId
        : null,
    createdAt: source?.createdAt ?? now,
    updatedAt: now,
  };
}

function snapshotActiveSpace(state: WindowStoreShape): DesktopSpace {
  const existing = state.desktopSpaces.find((space) => space.id === state.activeDesktopId);
  const now = new Date().toISOString();
  return {
    id: state.activeDesktopId || DEFAULT_DESKTOP_ID,
    name: existing?.name || "Personal",
    kind: existing?.kind || "personal",
    desktopSettings: existing?.desktopSettings,
    desktopIcons: state.desktopIcons,
    dockPinnedItems: state.dockPinnedItems,
    hiddenModules: state.hiddenModules,
    folders: state.folders,
    windows: state.windows,
    windowOrder: state.windowOrder,
    focusedWindowId: state.focusedWindowId,
    createdAt: existing?.createdAt || now,
    updatedAt: now,
  };
}

// Load saved desktop settings from localStorage
export function loadSavedSettings(
  initialDockPinnedItems: string[],
): Partial<WindowStoreShape> {
  if (!browser) return {};

  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved) {
      const parsed = JSON.parse(saved);
      const activeDesktopId =
        typeof parsed.activeDesktopId === "string" ? parsed.activeDesktopId : DEFAULT_DESKTOP_ID;
      const legacySpace = createDesktopSpace(DEFAULT_DESKTOP_ID, "Personal", initialDockPinnedItems, {
        desktopIcons: parsed.desktopIcons,
        dockPinnedItems: parsed.dockPinnedItems,
        hiddenModules: parsed.hiddenModules,
        folders: parsed.folders,
        windows: parsed.windows,
        windowOrder: parsed.windowOrder,
        focusedWindowId: parsed.focusedWindowId,
      });
      const desktopSpaces =
        Array.isArray(parsed.desktopSpaces) && parsed.desktopSpaces.length > 0
          ? parsed.desktopSpaces.map((space: Partial<DesktopSpace>) =>
              createDesktopSpace(
                typeof space.id === "string" ? space.id : `desktop-${Date.now()}`,
                typeof space.name === "string" ? space.name : "Desktop",
                initialDockPinnedItems,
                space,
              ),
            )
          : [legacySpace];
      const activeSpace =
        desktopSpaces.find((space: DesktopSpace) => space.id === activeDesktopId) ||
        desktopSpaces[0] ||
        legacySpace;

      let desktopIcons = activeSpace.desktopIcons;
      const dockPinnedItems =
        activeSpace.dockPinnedItems.length > 0 ? activeSpace.dockPinnedItems : initialDockPinnedItems;

      desktopIcons = sanitizeDesktopIcons(desktopIcons, {
        allowEmpty: Array.isArray(activeSpace.desktopIcons),
        mergeDefaults: false,
      });

      return {
        activeDesktopId: activeSpace.id,
        desktopSpaces,
        desktopIcons,
        dockPinnedItems,
        hiddenModules: activeSpace.hiddenModules || [],
        folders: activeSpace.folders,
        windows: activeSpace.windows,
        windowOrder: activeSpace.windowOrder,
        focusedWindowId: activeSpace.focusedWindowId,
      };
    }
  } catch (e) {
    console.error("Failed to load desktop settings:", e);
  }
  return {};
}

// Save desktop settings to localStorage
export function saveSettings(state: WindowStoreShape) {
  if (!browser) return;

  try {
    const config = {
      version: CONFIG_VERSION,
      activeDesktopId: state.activeDesktopId,
      desktopSpaces: [
        ...state.desktopSpaces.filter((space) => space.id !== state.activeDesktopId),
        snapshotActiveSpace(state),
      ],
      desktopIcons: state.desktopIcons,
      dockPinnedItems: state.dockPinnedItems,
      hiddenModules: state.hiddenModules,
      folders: state.folders,
      windows: state.windows,
      windowOrder: state.windowOrder,
      focusedWindowId: state.focusedWindowId,
    };
    localStorage.setItem(STORAGE_KEY, JSON.stringify(config));
  } catch (e) {
    console.error("Failed to save desktop settings:", e);
  }
}

// Build an exportable DesktopConfig snapshot from current state
export function buildExportConfig(state: WindowStoreShape): DesktopConfig {
  return {
    version: CONFIG_VERSION,
    exportedAt: new Date().toISOString(),
    activeDesktopId: state.activeDesktopId,
    desktopSpaces: state.desktopSpaces,
    desktopIcons: state.desktopIcons,
    dockPinnedItems: state.dockPinnedItems,
    hiddenModules: state.hiddenModules,
    folders: state.folders,
  };
}

// Validate an imported DesktopConfig object
export function validateImportConfig(
  config: unknown,
): { valid: true; data: DesktopConfig } | { valid: false; error: string } {
  if (!config || typeof config !== "object") {
    return { valid: false, error: "Invalid configuration format" };
  }
  const c = config as Record<string, unknown>;
  if (!Array.isArray(c.desktopIcons)) {
    return { valid: false, error: "Missing or invalid desktopIcons" };
  }
  if (!Array.isArray(c.dockPinnedItems)) {
    return { valid: false, error: "Missing or invalid dockPinnedItems" };
  }
  // Validate each icon has required fields
  for (const icon of c.desktopIcons as Record<string, unknown>[]) {
    if (
      !icon.id ||
      !icon.module ||
      !icon.label ||
      typeof icon.x !== "number" ||
      typeof icon.y !== "number"
    ) {
      return { valid: false, error: "Invalid icon structure" };
    }
  }
  return { valid: true, data: c as unknown as DesktopConfig };
}

// JSON schema for desktop config (for documentation/tooling)
export const desktopConfigSchema = {
  $schema: "http://json-schema.org/draft-07/schema#",
  title: "BusinessOS Desktop Configuration",
  type: "object",
  required: ["version", "desktopIcons", "dockPinnedItems"],
  properties: {
    version: { type: "string", description: "Config version" },
    exportedAt: {
      type: "string",
      format: "date-time",
      description: "Export timestamp",
    },
    desktopIcons: {
      type: "array",
      items: {
        type: "object",
        required: ["id", "module", "label", "x", "y"],
        properties: {
          id: { type: "string" },
          module: { type: "string" },
          label: { type: "string" },
          x: { type: "number" },
          y: { type: "number" },
          type: { type: "string", enum: ["app", "folder"] },
          folderId: { type: "string" },
          folderColor: { type: "string" },
          appUrl: { type: "string" },
          launchMode: { type: "string", enum: ["iframe", "browser", "external"] },
        },
      },
    },
    dockPinnedItems: {
      type: "array",
      items: { type: "string" },
    },
    folders: {
      type: "array",
      items: {
        type: "object",
        required: ["id", "name", "color", "iconIds"],
        properties: {
          id: { type: "string" },
          name: { type: "string" },
          color: { type: "string" },
          iconIds: { type: "array", items: { type: "string" } },
        },
      },
    },
  },
};
