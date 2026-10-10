// windowStore.ts — barrel re-export for backwards compatibility
// All existing imports like `import { something } from '$lib/stores/windowStore'`
// continue working without modification.
//
// Domain files:
//   windowStateStore.ts      — windowStore, derived stores, core lifecycle methods
//   windowSnapStore.ts       — snapWindow logic
//   windowIconStore.ts       — icon/folder methods
//   windowModuleStore.ts     — moduleDefaults, registerDeployedApp/unregisterDeployedApp
//   windowSerializationStore.ts — exportConfig, importConfig, reset, resetDesktop

// Re-export types (consumers import types from here)
export type {
  WindowState,
  SnapZone,
  DesktopSpace,
  DesktopIcon,
  DesktopFolder,
  CustomIconConfig,
  DesktopConfig,
} from "./desktopTypes";

// Re-export the primary store and all derived stores
export {
  windowStore,
  focusedWindow,
  visibleWindows,
  minimizedWindows,
  openModules,
} from "./windowStateStore";
