// Window Serialization Store - Save/restore window state, persistence, export/import
// Contains exportConfig, importConfig, getConfigSchema, resetDesktop, and reset.

import { browser } from "$app/environment";
import type { WindowStoreShape, DesktopConfig } from "./desktopTypes";
import {
  buildExportConfig,
  validateImportConfig,
  desktopConfigSchema,
  initialDesktopIcons,
  STORAGE_KEY,
  saveSettings,
} from "./desktopPersistence";

// Creates the serialization methods that operate on the window store
export function createSerializationMethods(
  update: (fn: (state: WindowStoreShape) => WindowStoreShape) => void,
  set: (state: WindowStoreShape) => void,
  subscribe: (fn: (state: WindowStoreShape) => void) => () => void,
  initialState: WindowStoreShape,
) {
  return {
    // Export desktop configuration as JSON
    exportConfig: (): DesktopConfig => {
      let config: DesktopConfig = buildExportConfig(initialState);
      const unsubscribe = subscribe((state) => {
        config = buildExportConfig(state);
      });
      unsubscribe();
      return config;
    },

    // Import desktop configuration from JSON
    importConfig: (
      config: DesktopConfig,
    ): { success: boolean; error?: string } => {
      const validation = validateImportConfig(config);
      if (!validation.valid) {
        return { success: false, error: validation.error };
      }

      try {
        update((state) => {
          // Ensure Finder is in dock
          let dockItems = config.dockPinnedItems;
          if (!dockItems.includes("finder")) {
            dockItems = ["finder", ...dockItems];
          }

          const newState = {
            ...state,
            desktopIcons: config.desktopIcons,
            dockPinnedItems: dockItems,
            hiddenModules: config.hiddenModules || [],
            folders: config.folders || [],
          };
          saveSettings(newState);
          return newState;
        });

        return { success: true };
      } catch {
        return { success: false, error: "Failed to import configuration" };
      }
    },

    // Get JSON schema for desktop config
    getConfigSchema: () => desktopConfigSchema,

    // Reset store (clears saved settings too)
    reset: () => {
      if (browser) {
        localStorage.removeItem(STORAGE_KEY);
      }
      set(initialState);
    },

    // Reset only desktop settings (icons, dock) to defaults
    resetDesktop: () => {
      if (browser) {
        localStorage.removeItem(STORAGE_KEY);
      }
      update((state) => ({
        ...state,
        desktopIcons: initialDesktopIcons,
        dockPinnedItems: initialState.dockPinnedItems,
        hiddenModules: [],
        folders: [],
      }));
    },
  };
}
