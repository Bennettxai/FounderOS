/**
 * Desktop 3D Store
 * Manages state for the experimental 3D desktop view.
 * Supports both built-in modules AND dynamically installed/generated modules.
 *
 * Types and constants live in ./desktop3d/types.ts
 * Module metadata lives in ./desktop3d/moduleRegistry.ts
 * Standalone camera store (Svelte 5 runes) lives in ./desktop3d/cameraStore.svelte.ts
 */
import { writable, derived } from "svelte/store";

// ── Re-exports from sub-modules (backward-compatible) ─────────────────────────
export {
  CORE_MODULES,
  BUILTIN_MODULES,
  ALL_MODULES,
  type BuiltinModuleId,
  type ModuleId,
  type ViewMode,
  type Window3DState,
  type Desktop3DState,
  type DynamicModuleConfig,
} from "./desktop3d/types";

export { MODULE_INFO, DYNAMIC_MODULE_COLORS } from "./desktop3d/moduleRegistry";

export { cameraStore } from "./desktop3d/cameraStore.svelte";

// ── Local imports needed by store logic ───────────────────────────────────────
import {
  CORE_MODULES,
  BUILTIN_MODULES,
  ALL_MODULES,
  type ModuleId,
  type ViewMode,
  type Window3DState,
  type Desktop3DState,
} from "./desktop3d/types";

import { MODULE_INFO, DYNAMIC_MODULE_COLORS } from "./desktop3d/moduleRegistry";

// ── Default state ──────────────────────────────────────────────────────────────

const defaultState: Desktop3DState = {
  viewMode: "orb",
  windows: [],
  focusedWindowId: null,
  sphereRadius: 120, // Expanded radius for more spacing (was 95, now 120 for 22 modules)
  cameraDistance: 400, // Default camera distance from scene (zoom control)
  gridColumns: 4,
  gridSpacing: 130,
  autoRotate: true,
  animating: false,
  cameraRotationDelta: { x: 0, y: 0 }, // Gesture rotation delta
  gestureDragging: false, // Gesture dragging state
};

// ── Ring-based sphere layout ───────────────────────────────────────────────────
// Structured layers like a geodesic dome.
// Dynamically creates 3-5 rings based on module count.
// Distributes modules evenly across rings to avoid overcrowding.

interface RingConfig {
  y: number; // Height on sphere (-1 to 1)
  count: number; // Windows in this ring
  startIndex: number; // Global start index
}

function getRingLayout(total: number): RingConfig[] {
  // 1-3 modules: single ring at center
  if (total <= 3) {
    return [{ y: 0, count: total, startIndex: 0 }];
  }

  // 4-6 modules: 2 rings (top, bottom)
  if (total <= 6) {
    const top = Math.floor(total / 2);
    const bottom = total - top;
    return [
      { y: 0.5, count: top, startIndex: 0 },
      { y: -0.5, count: bottom, startIndex: top },
    ];
  }

  // 7-12 modules: 3 rings (top, middle, bottom)
  if (total <= 12) {
    const middle = Math.ceil(total / 2);
    const remaining = total - middle;
    const top = Math.ceil(remaining / 2);
    const bottom = remaining - top;
    return [
      { y: 0.6, count: top, startIndex: 0 },
      { y: 0, count: middle, startIndex: top },
      { y: -0.6, count: bottom, startIndex: top + middle },
    ];
  }

  // 13-18 modules: 4 rings (better distribution)
  if (total <= 18) {
    const perRing = Math.ceil(total / 4);
    const top = Math.min(perRing, total);
    const upperMid = Math.min(perRing, total - top);
    const lowerMid = Math.min(perRing, total - top - upperMid);
    const bottom = total - top - upperMid - lowerMid;
    return [
      { y: 0.65, count: top, startIndex: 0 },
      { y: 0.22, count: upperMid, startIndex: top },
      { y: -0.22, count: lowerMid, startIndex: top + upperMid },
      { y: -0.65, count: bottom, startIndex: top + upperMid + lowerMid },
    ];
  }

  // 19+ modules: 5 rings (maximum distribution)
  const perRing = Math.ceil(total / 5);
  const top = Math.min(perRing, total);
  const upperMid = Math.min(perRing, total - top);
  const middle = Math.min(perRing, total - top - upperMid);
  const lowerMid = Math.min(perRing, total - top - upperMid - middle);
  const bottom = total - top - upperMid - middle - lowerMid;
  return [
    { y: 0.7, count: top, startIndex: 0 },
    { y: 0.35, count: upperMid, startIndex: top },
    { y: 0, count: middle, startIndex: top + upperMid },
    { y: -0.35, count: lowerMid, startIndex: top + upperMid + middle },
    { y: -0.7, count: bottom, startIndex: top + upperMid + middle + lowerMid },
  ];
}

function getPositionInRing(
  indexInRing: number,
  ringCount: number,
  ringY: number,
  radius: number,
  ringIndex: number, // Which ring (0, 1, 2) for offset
): [number, number, number] {
  // Offset each ring's starting angle so they don't align vertically
  const ringOffset = (ringIndex * Math.PI) / 3; // 60 degree offset per ring

  // Calculate angle for this position in the ring
  const angle = ringOffset + (indexInRing / ringCount) * Math.PI * 2;

  // Y position from ring config
  const y = ringY * radius;

  // Radius at this height (smaller at poles, larger at equator)
  // Use a gentler curve so top/bottom rings aren't too small
  const heightFactor = Math.abs(ringY);
  const radiusAtY = Math.sqrt(1 - heightFactor * heightFactor * 0.5) * radius;

  const x = Math.cos(angle) * radiusAtY;
  const z = Math.sin(angle) * radiusAtY;

  return [x, y, z];
}

// Calculate position using structured ring layout
function calculateOrbPosition(
  index: number,
  total: number,
  radius: number,
  moduleId: string,
): [number, number, number] {
  const rings = getRingLayout(total);

  // Find which ring this index belongs to
  for (let ringIndex = 0; ringIndex < rings.length; ringIndex++) {
    const ring = rings[ringIndex];
    if (index < ring.startIndex + ring.count) {
      const indexInRing = index - ring.startIndex;
      return getPositionInRing(
        indexInRing,
        ring.count,
        ring.y,
        radius,
        ringIndex,
      );
    }
  }

  // Fallback to last ring
  const lastRing = rings[rings.length - 1];
  return getPositionInRing(
    0,
    lastRing.count,
    lastRing.y,
    radius,
    rings.length - 1,
  );
}

// Calculate grid position
function calculateGridPosition(
  index: number,
  total: number,
  columns: number,
  spacing: number,
): [number, number, number] {
  const rows = Math.ceil(total / columns);
  const col = index % columns;
  const row = Math.floor(index / columns);

  const offsetX = ((Math.min(total, columns) - 1) * spacing) / 2;
  const offsetY = ((rows - 1) * spacing) / 2;

  const x = col * spacing - offsetX;
  const y = row * -spacing + offsetY;
  const z = 0;

  return [x, y, z];
}

// ── Store factory ──────────────────────────────────────────────────────────────

function createDesktop3DStore() {
  const { subscribe, set, update } = writable<Desktop3DState>(defaultState);
  let initializedModules: readonly string[] = ALL_MODULES;

  return {
    subscribe,

    // Initialize with the active workspace's enabled modules.
    initialize: (modules: readonly string[] = ALL_MODULES) => {
      initializedModules = [...new Set(modules)].filter(
        (module) => MODULE_INFO[module] !== undefined,
      );

      update((state) => {
        const windows: Window3DState[] = initializedModules.map((module, index) => {
          const info = MODULE_INFO[module];
          const position = calculateOrbPosition(
            index,
            initializedModules.length,
            state.sphereRadius,
            module,
          );

          return {
            id: `window-${module}`,
            module,
            title: info.title,
            position,
            targetPosition: position,
            rotation: [0, 0, 0],
            scale: 1,
            targetScale: 1,
            opacity: 1,
            targetOpacity: 1,
            isCore: (CORE_MODULES as readonly string[]).includes(module),
            isOpen: true,
            isFocused: false,
            lastFocused: Date.now(),
            color: info.color,
            width: 1300,
            height: 900,
          };
        });

        return { ...state, windows };
      });
    },

    // Recalculate all positions based on current view mode
    recalculatePositions: () => {
      update((state) => {
        const openWindows = state.windows.filter((w) => w.isOpen);
        const windows = state.windows.map((window) => {
          if (!window.isOpen) return window;

          const index = openWindows.findIndex((w) => w.id === window.id);
          let newPosition: [number, number, number];

          if (state.viewMode === "grid") {
            newPosition = calculateGridPosition(
              index,
              openWindows.length,
              state.gridColumns,
              state.gridSpacing,
            );
          } else {
            newPosition = calculateOrbPosition(
              index,
              openWindows.length,
              state.sphereRadius,
              window.module,
            );
          }

          // Update BOTH position and targetPosition so windows actually move
          return {
            ...window,
            position: newPosition,
            targetPosition: newPosition,
          };
        });

        return { ...state, windows };
      });
    },

    // Set view mode
    setViewMode: (mode: ViewMode) => {
      update((state) => {
        if (mode === state.viewMode) return state;

        // If going to focused mode without a focused window, don't change
        if (mode === "focused" && !state.focusedWindowId) return state;

        // Re-enable auto-rotate when leaving focused mode
        const autoRotate = mode !== "focused";

        const newState = {
          ...state,
          viewMode: mode,
          autoRotate,
          animating: true,
        };

        // Recalculate positions after state update
        setTimeout(() => {
          desktop3dStore.recalculatePositions();
          setTimeout(() => {
            update((s) => ({ ...s, animating: false }));
          }, 500);
        }, 0);

        return newState;
      });
    },

    // Toggle between orb and grid
    toggleViewMode: () => {
      update((state) => {
        const newMode = state.viewMode === "orb" ? "grid" : "orb";
        return { ...state, viewMode: newMode };
      });
      desktop3dStore.recalculatePositions();
    },

    // Focus on a window
    focusWindow: (windowId: string) => {
      update((state) => {
        const windows = state.windows.map((w) => ({
          ...w,
          isFocused: w.id === windowId,
          targetOpacity: w.id === windowId ? 1 : 0.3,
          targetScale: w.id === windowId ? 1.5 : 0.8,
          lastFocused: w.id === windowId ? Date.now() : w.lastFocused,
        }));

        return {
          ...state,
          windows,
          focusedWindowId: windowId,
          viewMode: "focused",
          autoRotate: false,
        };
      });
    },

    // Unfocus current window
    unfocusWindow: () => {
      update((state) => {
        const windows = state.windows.map((w) => ({
          ...w,
          isFocused: false,
          targetOpacity: 1,
          targetScale: 1,
        }));

        return {
          ...state,
          windows,
          focusedWindowId: null,
          viewMode: "orb",
          autoRotate: true, // Re-enable auto-rotate when exiting focus mode
        };
      });
      desktop3dStore.recalculatePositions();
    },

    // Open a module window
    openWindow: (module: ModuleId) => {
      update((state) => {
        // Check if already open
        const existing = state.windows.find((w) => w.module === module);

        if (existing?.isOpen) {
          // Window already open - focus it instead
          // Don't call nested store method — manipulate state directly
          const windows = state.windows.map((w) => ({
            ...w,
            isFocused: w.id === existing.id,
            targetOpacity: w.id === existing.id ? 1 : 0.3,
            targetScale: w.id === existing.id ? 1.5 : 0.8,
            lastFocused: w.id === existing.id ? Date.now() : w.lastFocused,
          }));

          return {
            ...state,
            windows,
            focusedWindowId: existing.id,
            viewMode: "focused",
            autoRotate: false,
          };
        }

        const info = MODULE_INFO[module];
        const openWindows = state.windows.filter((w) => w.isOpen);
        const position = calculateOrbPosition(
          openWindows.length,
          openWindows.length + 1,
          state.sphereRadius,
          module,
        );

        if (existing) {
          // Reopen existing window (was closed before)
          const windows = state.windows.map((w) =>
            w.module === module
              ? {
                  ...w,
                  isOpen: true,
                  position,
                  targetPosition: position,
                  lastFocused: Date.now(),
                }
              : w,
          );
          return { ...state, windows };
        }

        // Create new window (never existed before)
        const newWindow: Window3DState = {
          id: `window-${module}-${Date.now()}`,
          module,
          title: info.title,
          position,
          targetPosition: position,
          rotation: [0, 0, 0],
          scale: 1,
          targetScale: 1,
          opacity: 1,
          targetOpacity: 1,
          isCore: (CORE_MODULES as readonly string[]).includes(module),
          isOpen: true,
          isFocused: false,
          lastFocused: Date.now(),
          color: info.color,
          width: 800, // Default window width
          height: 600, // Default window height
        };

        return { ...state, windows: [...state.windows, newWindow] };
      });

      desktop3dStore.recalculatePositions();
    },

    // Close a window (core modules just minimize)
    closeWindow: (windowId: string) => {
      update((state) => {
        const window = state.windows.find((w) => w.id === windowId);
        if (!window) return state;

        // Core modules can't be closed
        if (window.isCore) return state;

        const windows = state.windows.map((w) =>
          w.id === windowId ? { ...w, isOpen: false } : w,
        );

        return {
          ...state,
          windows,
          focusedWindowId:
            state.focusedWindowId === windowId ? null : state.focusedWindowId,
        };
      });
      desktop3dStore.recalculatePositions();
    },

    // Toggle auto-rotate
    toggleAutoRotate: () => {
      update((state) => ({ ...state, autoRotate: !state.autoRotate }));
    },

    // Set auto-rotate
    setAutoRotate: (enabled: boolean) => {
      update((state) => ({ ...state, autoRotate: enabled }));
    },

    // Reset to default state
    reset: () => {
      set(defaultState);
      desktop3dStore.initialize(initializedModules);
    },

    // Adjust sphere radius (zoom in/out)
    adjustSphereRadius: (delta: number) => {
      update((state) => {
        const newRadius = Math.max(
          80,
          Math.min(180, state.sphereRadius + delta * 10),
        );

        if (newRadius === state.sphereRadius) {
          if (import.meta.env.DEV)
            console.log("[Desktop3D Store] Sphere radius at limit:", newRadius);
          return state;
        }

        const updatedState = { ...state, sphereRadius: newRadius };

        // Recalculate positions after radius change
        setTimeout(() => {
          desktop3dStore.recalculatePositions();
        }, 0);

        return updatedState;
      });
    },

    // Adjust camera distance (TRUE zoom — moves camera closer/farther)
    // Range: 200 (very close) to 800 (very far)
    adjustCameraDistance: (delta: number) => {
      update((state) => {
        const newDistance = Math.max(
          200,
          Math.min(800, state.cameraDistance + delta * 20),
        );

        if (newDistance === state.cameraDistance) {
          if (import.meta.env.DEV) {
            console.log(
              "[Desktop3D Store] Camera distance at limit:",
              newDistance,
            );
          }
          return state;
        }

        // Reduced logging — only log significant changes (> 5 units)
        if (Math.abs(delta) > 5) {
          if (import.meta.env.DEV) {
            console.log(
              `[Desktop3D Store] Camera distance: ${state.cameraDistance.toFixed(1)} → ${newDistance.toFixed(1)}`,
            );
          }
        }

        return { ...state, cameraDistance: newDistance };
      });
    },

    // Reset camera distance to default
    resetCameraDistance: () => {
      update((state) => {
        if (import.meta.env.DEV) {
          console.log(
            "[Desktop3D Store] Resetting camera distance to default (400)",
          );
        }
        return { ...state, cameraDistance: 400 };
      });
    },

    // Navigate to next window (when focused)
    focusNext: () => {
      update((state) => {
        if (!state.focusedWindowId) return state;

        const openWins = state.windows.filter((w) => w.isOpen);
        const currentIndex = openWins.findIndex(
          (w) => w.id === state.focusedWindowId,
        );
        const nextIndex = (currentIndex + 1) % openWins.length;
        const nextWindow = openWins[nextIndex];

        if (!nextWindow) return state;

        const windows = state.windows.map((w) => ({
          ...w,
          isFocused: w.id === nextWindow.id,
          targetOpacity: w.id === nextWindow.id ? 1 : 0.3,
          targetScale: w.id === nextWindow.id ? 1.5 : 0.8,
          lastFocused: w.id === nextWindow.id ? Date.now() : w.lastFocused,
        }));

        return { ...state, windows, focusedWindowId: nextWindow.id };
      });
    },

    // Navigate to previous window (when focused)
    focusPrevious: () => {
      update((state) => {
        if (!state.focusedWindowId) return state;

        const openWins = state.windows.filter((w) => w.isOpen);
        const currentIndex = openWins.findIndex(
          (w) => w.id === state.focusedWindowId,
        );
        const prevIndex =
          (currentIndex - 1 + openWins.length) % openWins.length;
        const prevWindow = openWins[prevIndex];

        if (!prevWindow) return state;

        const windows = state.windows.map((w) => ({
          ...w,
          isFocused: w.id === prevWindow.id,
          targetOpacity: w.id === prevWindow.id ? 1 : 0.3,
          targetScale: w.id === prevWindow.id ? 1.5 : 0.8,
          lastFocused: w.id === prevWindow.id ? Date.now() : w.lastFocused,
        }));

        return { ...state, windows, focusedWindowId: prevWindow.id };
      });
    },

    // Resize focused window
    resizeFocusedWindow: (widthDelta: number, heightDelta: number) => {
      update((state) => {
        if (!state.focusedWindowId) return state;

        const windows = state.windows.map((w) => {
          if (w.id !== state.focusedWindowId) return w;
          return {
            ...w,
            width: Math.max(800, Math.min(1600, w.width + widthDelta)),
            height: Math.max(500, Math.min(1100, w.height + heightDelta)),
          };
        });

        return { ...state, windows };
      });
    },

    // Update window position (used by layout system)
    updateWindowPosition: (
      moduleId: ModuleId,
      position: { x: number; y: number; z: number },
      rotation?: { x: number; y: number; z: number },
      scale?: number,
    ) => {
      update((state) => {
        const windows = state.windows.map((w) => {
          if (w.module === moduleId) {
            // Convert position object to tuple
            const positionTuple: [number, number, number] = [
              position.x,
              position.y,
              position.z,
            ];

            // Convert rotation object to tuple (or use existing)
            const rotationTuple: [number, number, number] = rotation
              ? [rotation.x, rotation.y, rotation.z]
              : w.rotation;

            return {
              ...w,
              position: positionTuple,
              targetPosition: positionTuple,
              rotation: rotationTuple,
              scale: scale !== undefined ? scale : w.scale,
              targetScale: scale !== undefined ? scale : w.targetScale,
            };
          }
          return w;
        });

        return { ...state, windows };
      });
    },

    // Close all windows (except core modules)
    closeAllWindows: () => {
      update((state) => {
        const windows = state.windows.map((w) => ({
          ...w,
          isOpen: w.isCore ? w.isOpen : false,
        }));

        return { ...state, windows, focusedWindowId: null };
      });
      desktop3dStore.recalculatePositions();
    },

    // Adjust rotation speed via gesture delta
    // Send { x: 0, y: 0 } when hand stops moving to clear the delta
    adjustRotationSpeed: (deltaX: number, deltaY: number = 0) => {
      update((state) => ({
        ...state,
        cameraRotationDelta: { x: deltaX, y: deltaY },
        gestureDragging: deltaX !== 0 || deltaY !== 0,
      }));
    },

    // Adjust grid spacing
    adjustGridSpacing: (delta: number) => {
      update((state) => {
        const newSpacing = Math.max(
          80,
          Math.min(200, state.gridSpacing + delta),
        );

        if (newSpacing === state.gridSpacing) {
          if (import.meta.env.DEV)
            console.log("[Desktop3D Store] Grid spacing at limit:", newSpacing);
          return state;
        }

        if (import.meta.env.DEV) {
          console.log(
            `[Desktop3D Store] Adjusting grid spacing: ${state.gridSpacing} → ${newSpacing}`,
          );
        }

        const updatedState = { ...state, gridSpacing: newSpacing };

        // Recalculate positions after spacing change
        setTimeout(() => {
          desktop3dStore.recalculatePositions();
        }, 0);

        return updatedState;
      });
    },

    // Adjust grid columns
    adjustGridColumns: (delta: number) => {
      update((state) => {
        const newColumns = Math.max(2, Math.min(8, state.gridColumns + delta));

        if (newColumns === state.gridColumns) {
          if (import.meta.env.DEV)
            console.log("[Desktop3D Store] Grid columns at limit:", newColumns);
          return state;
        }

        if (import.meta.env.DEV) {
          console.log(
            `[Desktop3D Store] Adjusting grid columns: ${state.gridColumns} → ${newColumns}`,
          );
        }

        const updatedState = { ...state, gridColumns: newColumns };

        // Recalculate positions after column change
        setTimeout(() => {
          desktop3dStore.recalculatePositions();
        }, 0);

        return updatedState;
      });
    },

    // Add a dynamically installed/generated module to the 3D desktop
    addModule: (config: {
      id: string;
      title: string;
      icon?: string;
      color?: string;
      category?: string;
    }) => {
      // Register in MODULE_INFO so openWindow can find it
      if (!MODULE_INFO[config.id]) {
        const color =
          config.color ||
          DYNAMIC_MODULE_COLORS[config.category || "general"] ||
          DYNAMIC_MODULE_COLORS.general;
        MODULE_INFO[config.id] = {
          title: config.title,
          color,
          icon: config.icon || "box",
        };
      }

      update((state) => {
        // Don't add if already exists
        if (state.windows.some((w) => w.module === config.id)) {
          return state;
        }

        const info = MODULE_INFO[config.id];
        const openWindows = state.windows.filter((w) => w.isOpen);
        const position = calculateOrbPosition(
          openWindows.length,
          openWindows.length + 1,
          state.sphereRadius,
          config.id,
        );

        const newWindow: Window3DState = {
          id: `window-${config.id}-${Date.now()}`,
          module: config.id,
          title: info.title,
          position,
          targetPosition: position,
          rotation: [0, 0, 0],
          scale: 1,
          targetScale: 1,
          opacity: 1,
          targetOpacity: 1,
          isCore: false,
          isOpen: true,
          isFocused: false,
          lastFocused: Date.now(),
          color: info.color,
          width: 1000,
          height: 700,
        };

        if (import.meta.env.DEV) {
          console.log(
            `[Desktop3D Store] Added dynamic module: ${config.title} (${config.id})`,
          );
        }

        return { ...state, windows: [...state.windows, newWindow] };
      });

      desktop3dStore.recalculatePositions();
    },

    // Remove a dynamically installed module from the 3D desktop
    removeModule: (moduleId: string) => {
      update((state) => {
        // Don't allow removing core modules
        if ((CORE_MODULES as readonly string[]).includes(moduleId)) {
          console.warn(
            `[Desktop3D Store] Cannot remove core module: ${moduleId}`,
          );
          return state;
        }

        const windows = state.windows.filter((w) => w.module !== moduleId);

        if (import.meta.env.DEV)
          console.log(`[Desktop3D Store] Removed dynamic module: ${moduleId}`);

        return {
          ...state,
          windows,
          focusedWindowId:
            state.focusedWindowId &&
            state.windows.find((w) => w.id === state.focusedWindowId)
              ?.module === moduleId
              ? null
              : state.focusedWindowId,
        };
      });

      // Clean up MODULE_INFO for dynamic modules (keep built-in)
      if (!(BUILTIN_MODULES as readonly string[]).includes(moduleId)) {
        delete MODULE_INFO[moduleId];
      }

      desktop3dStore.recalculatePositions();
    },
  };
}

export const desktop3dStore = createDesktop3DStore();

// Derived store for open windows only
export const openWindows = derived(desktop3dStore, ($store) =>
  $store.windows.filter((w) => w.isOpen),
);

// Derived store for focused window
export const focusedWindow = derived(desktop3dStore, ($store) =>
  $store.windows.find((w) => w.id === $store.focusedWindowId),
);
