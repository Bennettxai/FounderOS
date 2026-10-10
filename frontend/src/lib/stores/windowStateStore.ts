// Window State Store - Window lifecycle, positioning, z-order, focus, minimize/maximize
// This is the primary store that composes all domain-specific methods.
// It exports windowStore, focusedWindow, visibleWindows, minimizedWindows, openModules.

import { writable, derived, get } from "svelte/store";
import { soundStore } from "./soundStore";
import { defaultDesktopSettings, desktopSettings } from "./desktopThemeStore";
import type { DesktopSettings } from "./desktopThemeStore";
import {
  DEFAULT_DESKTOP_ID,
  loadSavedSettings,
  saveSettings,
  initialDesktopIcons,
} from "./desktopPersistence";
import type { DesktopSpace, SnapZone, WindowState, WindowStoreShape } from "./desktopTypes";
import {
  moduleDefaults,
  createModuleMethods,
  getWorkspaceAppIconConfig,
  getWorkspaceAppMetadata,
} from "./windowModuleStore";
import { createSnapMethods } from "./windowSnapStore";
import { createIconMethods } from "./windowIconStore";
import { createSerializationMethods } from "./windowSerializationStore";

const initialState: WindowStoreShape = {
  windows: [],
  focusedWindowId: null,
  windowOrder: [],
  dockPinnedItems: [
    "finder",
    "dashboard",
    "knowledge",
    "agents",
    "projects",
    "tasks",
    "pipelines",
    "drive",
  ],
  desktopIcons: initialDesktopIcons,
  hiddenModules: [],
  selectedIconIds: [],
  folders: [],
  activeDesktopId: DEFAULT_DESKTOP_ID,
  desktopSpaces: [
    {
      id: DEFAULT_DESKTOP_ID,
      name: "Personal",
      kind: "personal",
      desktopIcons: initialDesktopIcons,
      dockPinnedItems: [
        "finder",
        "dashboard",
        "knowledge",
        "agents",
        "projects",
        "tasks",
        "pipelines",
        "drive",
      ],
      hiddenModules: [],
      folders: [],
      windows: [],
      windowOrder: [],
      focusedWindowId: null,
      desktopSettings: defaultDesktopSettings as unknown as Record<string, unknown>,
      createdAt: new Date(0).toISOString(),
      updatedAt: new Date(0).toISOString(),
    },
  ],
};

function createDesktopSpaceId() {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `desktop-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

type CreateDesktopSpaceOptions = {
  id?: string;
  kind?: DesktopSpace["kind"];
};

type OpenWindowOptions = {
  title?: string;
  data?: Record<string, unknown>;
  x?: number;
  y?: number;
};

type ClampActiveDesktopOptions = {
  canvas?: boolean;
  iconWidth?: number;
  iconHeight?: number;
};

type ArrangeActiveDesktopIconOptions = {
  iconWidth?: number;
  iconHeight?: number;
  gap?: number;
  padding?: number;
  originX?: number;
  originY?: number;
};

function unhideModule(hiddenModules: string[], module: string): string[] {
  return hiddenModules.filter((hiddenModule) => {
    if (hiddenModule === module) return false;
    if (module.startsWith("workspace-app-") && hiddenModule === "workspace-app-*") return false;
    return true;
  });
}

function clampValue(value: number, min: number, max: number) {
  return Math.max(min, Math.min(value, max));
}

function createWindowStore() {
  // Merge initial state with any saved settings
  const savedSettings = loadSavedSettings(initialState.dockPinnedItems);
  const mergedInitial: WindowStoreShape = {
    ...initialState,
    ...savedSettings,
  };

  const { subscribe, set, update } = writable<WindowStoreShape>(mergedInitial);

  let cascadeOffset = 0;
  let initialized = false;

  const OPEN_WINDOWS_KEY = "businessos_open_windows";
  type PersistedOpenWindow = {
    module: string;
    title: string;
    x: number;
    y: number;
    width: number;
    height: number;
    minWidth?: number;
    minHeight?: number;
    minimized: boolean;
    maximized: boolean;
    snapped?: SnapZone;
    previousBounds?: WindowState["previousBounds"];
    data?: Record<string, unknown>;
  };

  function snapshotDesktopSpace(state: WindowStoreShape): DesktopSpace {
    const existing = state.desktopSpaces.find((space) => space.id === state.activeDesktopId);
    const now = new Date().toISOString();
    return {
      id: state.activeDesktopId || DEFAULT_DESKTOP_ID,
      name: existing?.name || "Personal",
      kind: existing?.kind || "personal",
      desktopSettings: get(desktopSettings) as unknown as Record<string, unknown>,
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

  function applyDesktopSpace(state: WindowStoreShape, space: DesktopSpace): WindowStoreShape {
    const windowIds = new Set(space.windows.map((w) => w.id));
    return {
      ...state,
      activeDesktopId: space.id,
      desktopIcons: space.desktopIcons,
      dockPinnedItems: space.dockPinnedItems,
      hiddenModules: space.hiddenModules || [],
      folders: space.folders,
      windows: space.windows,
      windowOrder: space.windowOrder.filter((id) => windowIds.has(id)),
      focusedWindowId:
        space.focusedWindowId && windowIds.has(space.focusedWindowId)
          ? space.focusedWindowId
          : null,
      selectedIconIds: [],
    };
  }

  function applyDesktopSettingsForSpace(space: DesktopSpace) {
    desktopSettings.apply((space.desktopSettings || defaultDesktopSettings) as Partial<DesktopSettings>);
  }

  function saveWholeState(state: WindowStoreShape) {
    saveSettings(state);
    persistOpenWindows(state.windows);
  }

  // Persist open windows to localStorage
  function persistOpenWindows(windows: WindowState[]) {
    if (typeof window === "undefined") return;
    try {
      const toSave: PersistedOpenWindow[] = windows.map((w) => ({
        module: w.module,
        title: w.title,
        x: w.x,
        y: w.y,
        width: w.width,
        height: w.height,
        minWidth: w.minWidth,
        minHeight: w.minHeight,
        minimized: w.minimized,
        maximized: w.maximized,
        snapped: w.snapped,
        previousBounds: w.previousBounds,
        data: w.data,
      }));
      localStorage.setItem(OPEN_WINDOWS_KEY, JSON.stringify(toSave));
    } catch {
      /* ignore */
    }
  }

  // Restore open windows from localStorage
  function restoreOpenWindows(): WindowState[] {
    if (typeof window === "undefined") return [];
    try {
      const raw = localStorage.getItem(OPEN_WINDOWS_KEY);
      if (!raw) return [];
      const saved = JSON.parse(raw) as PersistedOpenWindow[];
      if (!Array.isArray(saved)) return [];
      return saved
        .filter(
          (w) =>
            w &&
            typeof w.module === "string" &&
            typeof w.title === "string" &&
            typeof w.x === "number" &&
            typeof w.y === "number" &&
            typeof w.width === "number" &&
            typeof w.height === "number",
        )
        .map((w, i) => ({
          id: `${w.module}-restored-${i}`,
          module: w.module,
          title: w.title,
          x: w.x,
          y: w.y,
          width: w.width,
          height: w.height,
          minWidth: w.minWidth ?? 400,
          minHeight: w.minHeight ?? 300,
          minimized: Boolean(w.minimized),
          maximized: Boolean(w.maximized),
          snapped: w.snapped,
          previousBounds: w.previousBounds,
          data: w.data,
        }));
    } catch {
      return [];
    }
  }

  // Compose domain-specific method groups
  const snapMethods = createSnapMethods(update);
  const iconMethods = createIconMethods(update, subscribe);
  const moduleMethods = createModuleMethods(update, persistOpenWindows);
  const serializationMethods = createSerializationMethods(
    update,
    set,
    subscribe,
    initialState,
  );

  return {
    subscribe,

    // Initialize store on client side (call this in onMount)
    initialize: () => {
      if (initialized || typeof window === "undefined") return;
      initialized = true;

      const saved = loadSavedSettings(initialState.dockPinnedItems);

      update((state) => {
        // Get dock items from saved settings or current state
        let dockItems =
          Object.keys(saved).length > 0 && saved.dockPinnedItems
            ? saved.dockPinnedItems
            : state.dockPinnedItems;

        // Filter out stale dock items that don't match any known module
        // This prevents ghost entries (e.g. "app-store") from persisting in localStorage
        const knownModules = new Set(Object.keys(moduleDefaults));
        dockItems = dockItems.filter(
          (item) => knownModules.has(item) || item.startsWith("workspace-app-"),
        );

        // ALWAYS ensure Finder is at the beginning of dock
        if (!dockItems.includes("finder")) {
          dockItems = ["finder", ...dockItems];
        } else if (dockItems[0] !== "finder") {
          // Move finder to first position
          dockItems = [
            "finder",
            ...dockItems.filter((item) => item !== "finder"),
          ];
        }

        // Get desktop icons - use saved if available, otherwise keep current state (initial)
        // Filter out stale osa-app-* icons - they are re-registered dynamically
        // by deployedAppsStore.startDiscovery() from the live backend API
        const rawDesktopIcons =
          Object.keys(saved).length > 0 && Array.isArray(saved.desktopIcons)
            ? saved.desktopIcons
            : state.desktopIcons;
        const desktopIcons = rawDesktopIcons.filter(
          (icon) => !icon.module.startsWith("osa-app-"),
        );

        // Get folders
        const folders =
          Object.keys(saved).length > 0 && saved.folders
            ? saved.folders
            : state.folders;

        // Restore previously open windows from localStorage
        const restoredWindows = (Array.isArray(saved.windows)
          ? saved.windows
          : restoreOpenWindows()).filter(
          (w) =>
            knownModules.has(w.module) ||
            w.module.startsWith("folder-") ||
            w.module.startsWith("osa-app-") ||
            w.module.startsWith("workspace-app-"),
        );

        // Merge: keep any already-open windows + add restored ones (no duplicates)
        const existingModules = new Set(state.windows.map((w) => w.module));
        const windows = [
          ...state.windows,
          ...restoredWindows.filter((w) => !existingModules.has(w.module)),
        ];
        const windowIds = new Set(windows.map((w) => w.id));
        const windowOrder = windows.map((w) => w.id);

        const activeSpace = (saved.desktopSpaces || state.desktopSpaces).find(
          (space) => space.id === (saved.activeDesktopId || state.activeDesktopId),
        );
        if (activeSpace) {
          applyDesktopSettingsForSpace(activeSpace);
        }

        const newState = {
          ...state,
          activeDesktopId: saved.activeDesktopId || state.activeDesktopId,
          desktopSpaces: saved.desktopSpaces || state.desktopSpaces,
          dockPinnedItems: dockItems,
          desktopIcons,
          folders,
          windows,
          windowOrder,
        };

        // Save the updated settings
        saveSettings(newState);

        return newState;
      });
    },

    // Open a new window for a module
    openWindow: (
      module: string,
      options?: string | OpenWindowOptions,
    ) => {
      // Handle legacy string parameter (custom title)
      const customTitle =
        typeof options === "string" ? options : options?.title;
      const windowData =
        typeof options === "object" ? options?.data : undefined;
      const requestedX = typeof options === "object" ? options?.x : undefined;
      const requestedY = typeof options === "object" ? options?.y : undefined;

      update((state) => {
        // Check if window already exists for this module
        const existingWindow = state.windows.find(
          (w) => w.module === module && !w.minimized,
        );
        if (existingWindow) {
          // Update data if provided and focus window
          const updatedWindows = windowData
            ? state.windows.map((w) =>
                w.id === existingWindow.id
                  ? {
                      ...w,
                      x: typeof requestedX === "number" ? requestedX : w.x,
                      y: typeof requestedY === "number" ? requestedY : w.y,
                      data: { ...w.data, ...windowData },
                    }
                  : w,
              )
            : typeof requestedX === "number" || typeof requestedY === "number"
              ? state.windows.map((w) =>
                  w.id === existingWindow.id
                    ? {
                        ...w,
                        x: typeof requestedX === "number" ? requestedX : w.x,
                        y: typeof requestedY === "number" ? requestedY : w.y,
                      }
                    : w,
                )
            : state.windows;
          const newState = {
            ...state,
            windows: updatedWindows,
            focusedWindowId: existingWindow.id,
            windowOrder: [
              ...state.windowOrder.filter((id) => id !== existingWindow.id),
              existingWindow.id,
            ],
          };
          saveWholeState(newState);
          return {
            ...newState,
          };
        }

        // Check if minimized window exists
        const minimizedWindow = state.windows.find(
          (w) => w.module === module && w.minimized,
        );
        if (minimizedWindow) {
          // Restore it and update data if provided
          const newState = {
            ...state,
            windows: state.windows.map((w) =>
              w.id === minimizedWindow.id
                ? {
                    ...w,
                    x: typeof requestedX === "number" ? requestedX : w.x,
                    y: typeof requestedY === "number" ? requestedY : w.y,
                    minimized: false,
                    data: windowData ? { ...w.data, ...windowData } : w.data,
                  }
                : w,
            ),
            focusedWindowId: minimizedWindow.id,
            windowOrder: [
              ...state.windowOrder.filter((id) => id !== minimizedWindow.id),
              minimizedWindow.id,
            ],
          };
          saveWholeState(newState);
          return newState;
        }

        const defaults = moduleDefaults[module] || (module.startsWith("canvas-note") ? moduleDefaults["canvas-note"] : undefined) || {
          title: module,
          width: 1100,
          height: 750,
          minWidth: 400,
          minHeight: 300,
        };
        const id = `${module}-${Date.now()}`;

        // Center the window on screen with slight cascade offset
        const screenW =
          typeof window !== "undefined" ? window.innerWidth : 1440;
        const screenH =
          typeof window !== "undefined" ? window.innerHeight : 900;
        const baseX = Math.max(
          40,
          (screenW - defaults.width) / 2 + cascadeOffset * 25,
        );
        const baseY = Math.max(
          30,
          (screenH - defaults.height) / 2 + cascadeOffset * 25,
        );
        cascadeOffset = (cascadeOffset + 1) % 8;

        const newWindow: WindowState = {
          id,
          module,
          title: customTitle || defaults.title,
          x: typeof requestedX === "number" ? requestedX : baseX,
          y: typeof requestedY === "number" ? requestedY : baseY,
          width: defaults.width,
          height: defaults.height,
          minWidth: defaults.minWidth,
          minHeight: defaults.minHeight,
          minimized: false,
          maximized: false,
          data: windowData,
        };

        // Play window open sound
        soundStore.playSound("windowOpen");

        const newWindows = [...state.windows, newWindow];
        const newState = {
          ...state,
          windows: newWindows,
          focusedWindowId: id,
          windowOrder: [...state.windowOrder, id],
        };
        saveWholeState(newState);
        return newState;
      });
    },

    // Close a window
    closeWindow: (windowId: string) => {
      // Play window close sound
      soundStore.playSound("windowClose");
      update((state) => {
        const newWindows = state.windows.filter((w) => w.id !== windowId);
        const newOrder = state.windowOrder.filter((id) => id !== windowId);
        const newFocused =
          state.focusedWindowId === windowId
            ? newOrder.length > 0
              ? newOrder[newOrder.length - 1]
              : null
            : state.focusedWindowId;

        const newState = {
          ...state,
          windows: newWindows,
          windowOrder: newOrder,
          focusedWindowId: newFocused,
        };
        saveWholeState(newState);
        return newState;
      });
    },

    // Minimize a window
    minimizeWindow: (windowId: string) => {
      // Play minimize sound
      soundStore.playSound("windowMinimize");
      update((state) => {
        const newWindows = state.windows.map((w) =>
          w.id === windowId ? { ...w, minimized: true } : w,
        );
        const newOrder = state.windowOrder.filter((id) => id !== windowId);
        const newFocused =
          state.focusedWindowId === windowId
            ? newOrder.length > 0
              ? newOrder[newOrder.length - 1]
              : null
            : state.focusedWindowId;
        const newState = {
          ...state,
          windows: newWindows,
          windowOrder: newOrder,
          focusedWindowId: newFocused,
        };
        saveWholeState(newState);
        return newState;
      });
    },

    // Restore a minimized window
    restoreWindow: (windowId: string) => {
      update((state) => {
        const newWindows = state.windows.map((w) =>
          w.id === windowId ? { ...w, minimized: false } : w,
        );
        const newState = {
          ...state,
          windows: newWindows,
          focusedWindowId: windowId,
          windowOrder: [
            ...state.windowOrder.filter((id) => id !== windowId),
            windowId,
          ],
        };
        saveWholeState(newState);
        return newState;
      });
    },

    // Toggle maximize state
    toggleMaximize: (windowId: string) => {
      // Play maximize sound
      soundStore.playSound("windowMaximize");
      update((state) => {
        const newWindows = state.windows.map((w) => {
          if (w.id !== windowId) return w;

          if (w.maximized) {
            // Restore to previous bounds
            return {
              ...w,
              maximized: false,
              snapped: null,
              x: w.previousBounds?.x ?? w.x,
              y: w.previousBounds?.y ?? w.y,
              width: w.previousBounds?.width ?? w.width,
              height: w.previousBounds?.height ?? w.height,
              previousBounds: undefined,
            };
          } else {
            // Store current bounds and maximize
            return {
              ...w,
              maximized: true,
              snapped: null,
              previousBounds: {
                x: w.x,
                y: w.y,
                width: w.width,
                height: w.height,
              },
            };
          }
        });
        const newState = {
          ...state,
          windows: newWindows,
        };
        saveWholeState(newState);
        return newState;
      });
    },

    // Focus a window (bring to front)
    focusWindow: (windowId: string) => {
      update((state) => {
        const newState = {
          ...state,
          focusedWindowId: windowId,
          windowOrder: [
            ...state.windowOrder.filter((id) => id !== windowId),
            windowId,
          ],
        };
        saveSettings(newState);
        return newState;
      });
    },

    // Update window position
    updateWindowPosition: (windowId: string, x: number, y: number) => {
      update((state) => {
        const newWindows = state.windows.map((w) =>
          w.id === windowId ? { ...w, x, y, maximized: false } : w,
        );
        const newState = {
          ...state,
          windows: newWindows,
        };
        saveWholeState(newState);
        return newState;
      });
    },

    // Update window size
    updateWindowSize: (windowId: string, width: number, height: number) => {
      update((state) => {
        const newWindows = state.windows.map((w) =>
          w.id === windowId
            ? {
                ...w,
                width: Math.max(width, w.minWidth),
                height: Math.max(height, w.minHeight),
                maximized: false,
              }
            : w,
        );
        const newState = {
          ...state,
          windows: newWindows,
        };
        saveWholeState(newState);
        return newState;
      });
    },

    // Update window bounds (position and size)
    updateWindowBounds: (
      windowId: string,
      x: number,
      y: number,
      width: number,
      height: number,
    ) => {
      update((state) => {
        const newWindows = state.windows.map((w) =>
          w.id === windowId
            ? {
                ...w,
                x,
                y,
                width: Math.max(width, w.minWidth),
                height: Math.max(height, w.minHeight),
                maximized: false,
              }
            : w,
        );
        const newState = {
          ...state,
          windows: newWindows,
        };
        saveWholeState(newState);
        return newState;
      });
    },

    createDesktopSpace: (name?: string, options: CreateDesktopSpaceOptions = {}) => {
      update((state) => {
        const now = new Date().toISOString();
        const current = snapshotDesktopSpace(state);
        const id = options.id || createDesktopSpaceId();
        const nextSpace: DesktopSpace = {
          id,
          name: name?.trim() || `Desktop ${state.desktopSpaces.length + 1}`,
          kind: options.kind || "personal",
          desktopSettings: defaultDesktopSettings as unknown as Record<string, unknown>,
          desktopIcons: [],
          dockPinnedItems: ["finder"],
          hiddenModules: ["workspace-app-*"],
          folders: [],
          windows: [],
          windowOrder: [],
          focusedWindowId: null,
          createdAt: now,
          updatedAt: now,
        };
        const spaces = [
          ...state.desktopSpaces.filter((space) => space.id !== state.activeDesktopId),
          current,
          nextSpace,
        ];
        const newState = applyDesktopSpace({ ...state, desktopSpaces: spaces }, nextSpace);
        applyDesktopSettingsForSpace(nextSpace);
        saveWholeState(newState);
        return newState;
      });
    },

    addDesktopIcon: (module: string, label: string) => {
      update((state) => {
        if (state.desktopIcons.some((icon) => icon.module === module)) return state;

        const workspaceApp = getWorkspaceAppMetadata(module);
        const sameColumn = state.desktopIcons.filter((icon) => icon.x === -1);
        const nextY =
          sameColumn.length > 0
            ? Math.max(...sameColumn.map((icon) => icon.y)) + 1
            : 0;
        const newState = {
          ...state,
          hiddenModules: unhideModule(state.hiddenModules, module),
          desktopIcons: [
            ...state.desktopIcons,
            {
              id: `icon-${module}`,
              module,
              label: workspaceApp?.name || label,
              x: -1,
              y: nextY,
              type: "app" as const,
              appUrl: workspaceApp?.url,
              launchMode: workspaceApp?.launch_mode,
              customIcon: workspaceApp
                ? getWorkspaceAppIconConfig(workspaceApp)
                : undefined,
            },
          ],
        };
        saveSettings(newState);
        return newState;
      });
    },

    removeDesktopIconByModule: (module: string) => {
      update((state) => {
        const newState = {
          ...state,
          hiddenModules: [...new Set([...state.hiddenModules, module])],
          selectedIconIds: state.selectedIconIds.filter((id) => {
            const selected = state.desktopIcons.find((icon) => icon.id === id);
            return selected?.module !== module;
          }),
          desktopIcons: state.desktopIcons.filter((icon) => icon.module !== module),
        };
        saveSettings(newState);
        return newState;
      });
    },

    clampActiveDesktopToViewport: (
      viewportWidth: number,
      viewportHeight: number,
      options: ClampActiveDesktopOptions = {},
    ) => {
      update((state) => {
        if (options.canvas) return state;
        if (viewportWidth <= 0 || viewportHeight <= 0) return state;

        const iconWidth = options.iconWidth ?? 90;
        const iconHeight = options.iconHeight ?? 104;
        const maxIconX = Math.max(0, viewportWidth - iconWidth);
        const maxIconY = Math.max(0, viewportHeight - iconHeight);

        const windows = state.windows.map((w) => {
          const width = Math.min(Math.max(1, w.width), viewportWidth);
          const height = Math.min(Math.max(1, w.height), viewportHeight);
          return {
            ...w,
            x: clampValue(w.x, 0, Math.max(0, viewportWidth - width)),
            y: clampValue(w.y, 0, Math.max(0, viewportHeight - height)),
            width,
            height,
            maximized: false,
          };
        });

        const occupiedIconPositions: Array<{ x: number; y: number }> = [];
        const slotStepX = iconWidth + 12;
        const slotStepY = iconHeight + 12;
        const availableSlots: Array<{ x: number; y: number }> = [];
        for (let x = maxIconX; x >= 0; x -= slotStepX) {
          for (let y = 0; y <= maxIconY; y += slotStepY) {
            availableSlots.push({ x: Math.max(0, x), y: Math.min(maxIconY, y) });
          }
        }

        const desktopIcons = state.desktopIcons.map((icon) => {
          if (icon.positionMode !== "pixel") return icon;
          const desired = {
            x: clampValue(icon.relativeX === undefined ? icon.x : icon.relativeX * maxIconX, 0, maxIconX),
            y: clampValue(icon.relativeY === undefined ? icon.y : icon.relativeY * maxIconY, 0, maxIconY),
          };
          let position = desired;
          const overlapsOccupied = (candidate: { x: number; y: number }) =>
            occupiedIconPositions.some(
              (occupied) =>
                Math.abs(candidate.x - occupied.x) < iconWidth + 8 &&
                Math.abs(candidate.y - occupied.y) < iconHeight + 8,
            );
          if (overlapsOccupied(desired)) {
            position = availableSlots
              .filter((slot) => !overlapsOccupied(slot))
              .sort((a, b) => {
                const distanceA = Math.hypot(a.x - desired.x, a.y - desired.y);
                const distanceB = Math.hypot(b.x - desired.x, b.y - desired.y);
                return distanceA - distanceB;
              })[0] ?? desired;
          }
          occupiedIconPositions.push(position);
          return {
            ...icon,
            x: position.x,
            y: position.y,
            positionMode: "pixel" as const,
          };
        });

        const newState = {
          ...state,
          windows,
          desktopIcons,
        };
        saveWholeState(newState);
        return newState;
      });
    },

    arrangeActiveDesktopIcons: (
      viewportWidth: number,
      viewportHeight: number,
      options: ArrangeActiveDesktopIconOptions = {},
    ) => {
      update((state) => {
        if (viewportWidth <= 0 || viewportHeight <= 0) return state;

        const iconWidth = options.iconWidth ?? 90;
        const iconHeight = options.iconHeight ?? 104;
        const gap = options.gap ?? 12;
        const padding = options.padding ?? 16;
        const originX = options.originX ?? 0;
        const originY = options.originY ?? 0;
        const rowStep = iconHeight + gap;
        const columnStep = iconWidth + gap;
        const rows = Math.max(
          1,
          Math.floor((viewportHeight - padding * 2 + gap) / rowStep),
        );
        const visibleIconIds = new Set(
          state.desktopIcons
            .filter((icon) => !icon.folderId || icon.type === "folder")
            .map((icon) => icon.id),
        );
        let visibleIndex = 0;

        const desktopIcons = state.desktopIcons.map((icon) => {
          if (!visibleIconIds.has(icon.id)) return icon;
          const row = visibleIndex % rows;
          const column = Math.floor(visibleIndex / rows);
          visibleIndex += 1;

          const x = Math.max(
            originX + padding,
            originX + viewportWidth - padding - iconWidth - column * columnStep,
          );
          const y = originY + padding + row * rowStep;
          const maxX = Math.max(0, viewportWidth - iconWidth);
          const maxY = Math.max(0, viewportHeight - iconHeight);
          return {
            ...icon,
            x,
            y,
            relativeX: maxX > 0 ? clampValue((x - originX) / maxX, 0, 1) : 0,
            relativeY: maxY > 0 ? clampValue((y - originY) / maxY, 0, 1) : 0,
            positionMode: "pixel" as const,
          };
        });

        const newState = { ...state, desktopIcons };
        saveWholeState(newState);
        return newState;
      });
    },

    duplicateDesktopSpace: (name?: string, options: CreateDesktopSpaceOptions = {}) => {
      update((state) => {
        const now = new Date().toISOString();
        const current = snapshotDesktopSpace(state);
        const id = options.id || createDesktopSpaceId();
        const clonedWindows = state.windows.map((w, index) => ({
          ...w,
          id: `${w.module}-${Date.now()}-${index}`,
        }));
        const oldToNew = new Map(state.windows.map((w, index) => [w.id, clonedWindows[index].id]));
        const nextSpace: DesktopSpace = {
          ...current,
          id,
          name: name?.trim() || `${current.name} Copy`,
          kind: options.kind || current.kind,
          windows: clonedWindows,
          windowOrder: state.windowOrder.map((id) => oldToNew.get(id)).filter(Boolean) as string[],
          focusedWindowId: state.focusedWindowId ? oldToNew.get(state.focusedWindowId) || null : null,
          createdAt: now,
          updatedAt: now,
        };
        const spaces = [
          ...state.desktopSpaces.filter((space) => space.id !== state.activeDesktopId),
          current,
          nextSpace,
        ];
        const newState = applyDesktopSpace({ ...state, desktopSpaces: spaces }, nextSpace);
        applyDesktopSettingsForSpace(nextSpace);
        saveWholeState(newState);
        return newState;
      });
    },

    switchDesktopSpace: (desktopId: string) => {
      update((state) => {
        if (desktopId === state.activeDesktopId) return state;
        const target = state.desktopSpaces.find((space) => space.id === desktopId);
        if (!target) return state;
        const current = snapshotDesktopSpace(state);
        const spaces = state.desktopSpaces.map((space) =>
          space.id === state.activeDesktopId ? current : space,
        );
        const newState = applyDesktopSpace({ ...state, desktopSpaces: spaces }, target);
        applyDesktopSettingsForSpace(target);
        saveWholeState(newState);
        return newState;
      });
    },

    renameDesktopSpace: (desktopId: string, name: string) => {
      const trimmed = name.trim();
      if (!trimmed) return;
      update((state) => {
        const current = snapshotDesktopSpace(state);
        const spaces = state.desktopSpaces.map((space) => {
          const source = space.id === state.activeDesktopId ? current : space;
          return source.id === desktopId
            ? { ...source, name: trimmed, updatedAt: new Date().toISOString() }
            : source;
        });
        const newState = {
          ...state,
          desktopSpaces: spaces,
        };
        saveSettings(newState);
        return newState;
      });
    },

    rekeyDesktopSpace: (desktopId: string, nextDesktopId: string) => {
      if (!desktopId || !nextDesktopId || desktopId === nextDesktopId) return;
      update((state) => {
        if (state.desktopSpaces.some((space) => space.id === nextDesktopId)) return state;
        const current = snapshotDesktopSpace(state);
        const spaces = state.desktopSpaces.map((space) => {
          const source = space.id === state.activeDesktopId ? current : space;
          return source.id === desktopId
            ? { ...source, id: nextDesktopId, updatedAt: new Date().toISOString() }
            : source;
        });
        const target = spaces.find((space) => space.id === nextDesktopId);
        if (!target) return state;
        const nextState = {
          ...state,
          desktopSpaces: spaces,
          activeDesktopId: state.activeDesktopId === desktopId ? nextDesktopId : state.activeDesktopId,
        };
        const applied = state.activeDesktopId === desktopId
          ? applyDesktopSpace(nextState, target)
          : nextState;
        saveWholeState(applied);
        return applied;
      });
    },

    deleteDesktopSpace: (desktopId: string) => {
      update((state) => {
        if (state.desktopSpaces.length <= 1) return state;
        const current = snapshotDesktopSpace(state);
        const spaces = state.desktopSpaces
          .map((space) => (space.id === state.activeDesktopId ? current : space))
          .filter((space) => space.id !== desktopId);
        const target = desktopId === state.activeDesktopId ? spaces[0] : current;
        const newState = applyDesktopSpace({ ...state, desktopSpaces: spaces }, target);
        applyDesktopSettingsForSpace(target);
        saveWholeState(newState);
        return newState;
      });
    },

    updateActiveDesktopSettings: (settings: Partial<DesktopSettings>) => {
      update((state) => {
        const current = {
          ...snapshotDesktopSpace(state),
          desktopSettings: settings as Record<string, unknown>,
        };
        const spaces = state.desktopSpaces.some((space) => space.id === state.activeDesktopId)
          ? state.desktopSpaces.map((space) => (space.id === state.activeDesktopId ? current : space))
          : [...state.desktopSpaces, current];
        const newState = {
          ...state,
          desktopSpaces: spaces,
        };
        saveSettings(newState);
        return newState;
      });
    },

    mergeDesktopSpaces: (spaces: DesktopSpace[]) => {
      update((state) => {
        if (!spaces.length) return state;
        const byId = new Map(state.desktopSpaces.map((space) => [space.id, space]));
        for (const space of spaces) {
          byId.set(space.id, {
            ...space,
            desktopIcons: Array.isArray(space.desktopIcons) ? space.desktopIcons : [],
            dockPinnedItems: Array.isArray(space.dockPinnedItems) ? space.dockPinnedItems : ["finder"],
            hiddenModules: Array.isArray(space.hiddenModules) ? space.hiddenModules : [],
            folders: Array.isArray(space.folders) ? space.folders : [],
            windows: Array.isArray(space.windows) ? space.windows : [],
            windowOrder: Array.isArray(space.windowOrder) ? space.windowOrder : [],
            focusedWindowId: space.focusedWindowId || null,
          });
        }
        const activeStillExists = byId.has(state.activeDesktopId);
        const target = activeStillExists ? byId.get(state.activeDesktopId) : spaces[0];
        const newState = {
          ...state,
          desktopSpaces: Array.from(byId.values()),
          activeDesktopId: activeStillExists ? state.activeDesktopId : spaces[0].id,
        };
        if (target) {
          applyDesktopSettingsForSpace(target);
        }
        saveSettings(newState);
        return activeStillExists ? newState : applyDesktopSpace(newState, spaces[0]);
      });
    },

    applyRemoteDesktopSpace: (space: DesktopSpace) => {
      update((state) => {
        const normalized: DesktopSpace = {
          ...space,
          desktopIcons: Array.isArray(space.desktopIcons) ? space.desktopIcons : [],
          dockPinnedItems: Array.isArray(space.dockPinnedItems) ? space.dockPinnedItems : ["finder"],
          hiddenModules: Array.isArray(space.hiddenModules) ? space.hiddenModules : [],
          folders: Array.isArray(space.folders) ? space.folders : [],
          windows: Array.isArray(space.windows) ? space.windows : [],
          windowOrder: Array.isArray(space.windowOrder) ? space.windowOrder : [],
          focusedWindowId: space.focusedWindowId || null,
        };
        const spaces = state.desktopSpaces.some((item) => item.id === normalized.id)
          ? state.desktopSpaces.map((item) => (item.id === normalized.id ? normalized : item))
          : [...state.desktopSpaces, normalized];
        const nextState = { ...state, desktopSpaces: spaces };
        if (state.activeDesktopId !== normalized.id) {
          saveSettings(nextState);
          return nextState;
        }
        const applied = applyDesktopSpace(nextState, normalized);
        applyDesktopSettingsForSpace(normalized);
        saveWholeState(applied);
        return applied;
      });
    },

    updateWindowData: (windowId: string, data: Record<string, unknown>) => {
      update((state) => {
        const newWindows = state.windows.map((w) =>
          w.id === windowId ? { ...w, data: { ...w.data, ...data } } : w,
        );
        const newState = { ...state, windows: newWindows };
        saveWholeState(newState);
        return newState;
      });
    },

    // Cycle to next window (for Cmd+`)
    cycleWindows: () => {
      update((state) => {
        const visibleWindows = state.windows.filter((w) => !w.minimized);
        if (visibleWindows.length < 2) return state;

        const visibleOrder = state.windowOrder.filter((id) =>
          state.windows.find((w) => w.id === id && !w.minimized),
        );

        if (visibleOrder.length < 2) return state;

        const currentVisibleIndex = visibleOrder.indexOf(
          state.focusedWindowId || "",
        );
        const nextIndex = (currentVisibleIndex + 1) % visibleOrder.length;
        const nextWindowId = visibleOrder[nextIndex];

        return {
          ...state,
          focusedWindowId: nextWindowId,
          windowOrder: [
            ...state.windowOrder.filter((id) => id !== nextWindowId),
            nextWindowId,
          ],
        };
      });
    },

    // Get windows for a specific module (for dock indicator)
    getWindowsForModule: (module: string) => {
      let result: WindowState[] = [];
      const unsubscribe = subscribe((state) => {
        result = state.windows.filter((w) => w.module === module);
      });
      unsubscribe();
      return result;
    },

    // --- Snap methods (from windowSnapStore) ---
    ...snapMethods,

    // --- Icon and folder methods (from windowIconStore) ---
    ...iconMethods,

    // --- Module registry methods (from windowModuleStore) ---
    ...moduleMethods,

    // --- Serialization methods (from windowSerializationStore) ---
    ...serializationMethods,
  };
}

export const windowStore = createWindowStore();

// Derived stores
export const focusedWindow = derived(
  windowStore,
  ($store) =>
    $store.windows.find((w) => w.id === $store.focusedWindowId) || null,
);

export const visibleWindows = derived(windowStore, ($store) =>
  $store.windows.filter((w) => !w.minimized),
);

export const minimizedWindows = derived(windowStore, ($store) =>
  $store.windows.filter((w) => w.minimized),
);

export const openModules = derived(windowStore, ($store) => [
  ...new Set($store.windows.map((w) => w.module)),
]);
