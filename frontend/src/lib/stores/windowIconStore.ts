// Window Icon Store - Desktop icon configs, icon positions, folder management
// Contains icon selection, position updates, folder CRUD, and icon customization.

import type {
  WindowStoreShape,
  WindowState,
  DesktopIcon,
  DesktopFolder,
  CustomIconConfig,
} from "./desktopTypes";
import { saveSettings } from "./desktopPersistence";
import { moduleDefaults } from "./windowModuleStore";

// Creates the icon and folder methods that operate on the window store
export function createIconMethods(
  update: (fn: (state: WindowStoreShape) => WindowStoreShape) => void,
  subscribe: (fn: (state: WindowStoreShape) => void) => () => void,
) {
  return {
    // Select desktop icon
    selectIcon: (iconId: string, additive: boolean = false) => {
      update((state) => ({
        ...state,
        selectedIconIds: additive
          ? state.selectedIconIds.includes(iconId)
            ? state.selectedIconIds.filter((id) => id !== iconId)
            : [...state.selectedIconIds, iconId]
          : [iconId],
      }));
    },

    // Clear icon selection
    clearIconSelection: () => {
      update((state) => ({
        ...state,
        selectedIconIds: [],
      }));
    },

    // Set selected icons (for lasso selection)
    setSelectedIcons: (iconIds: string[]) => {
      update((state) => ({
        ...state,
        selectedIconIds: iconIds,
      }));
    },

    // Update desktop icon position
    updateIconPosition: (
      iconId: string,
      x: number,
      y: number,
      viewport?: { width: number; height: number; iconWidth?: number; iconHeight?: number },
    ) => {
      update((state) => {
        const maxX = viewport ? Math.max(0, viewport.width - (viewport.iconWidth ?? 90)) : 0;
        const maxY = viewport ? Math.max(0, viewport.height - (viewport.iconHeight ?? 104)) : 0;
        const newState = {
          ...state,
          desktopIcons: state.desktopIcons.map((icon) =>
            icon.id === iconId
              ? {
                  ...icon,
                  x,
                  y,
                  positionMode: "pixel" as const,
                  ...(viewport
                    ? {
                        relativeX: maxX > 0 ? Math.min(1, Math.max(0, x / maxX)) : 0,
                        relativeY: maxY > 0 ? Math.min(1, Math.max(0, y / maxY)) : 0,
                      }
                    : {}),
                }
              : icon,
          ),
        };
        saveSettings(newState);
        return newState;
      });
    },

    // Update icon customization (Lucide icon or custom SVG)
    updateIconCustomization: (
      iconId: string,
      customIcon: CustomIconConfig | undefined,
    ) => {
      update((state) => {
        const newState = {
          ...state,
          desktopIcons: state.desktopIcons.map((icon) =>
            icon.id === iconId ? { ...icon, customIcon } : icon,
          ),
        };
        saveSettings(newState);
        return newState;
      });
    },

    // Reset icon to default appearance
    resetIconCustomization: (iconId: string) => {
      update((state) => {
        const newState = {
          ...state,
          desktopIcons: state.desktopIcons.map((icon) =>
            icon.id === iconId ? { ...icon, customIcon: undefined } : icon,
          ),
        };
        saveSettings(newState);
        return newState;
      });
    },

    // Add item to dock
    addToDock: (module: string) => {
      update((state) => {
        if (state.dockPinnedItems.includes(module)) return state;
        const newState = {
          ...state,
          hiddenModules: state.hiddenModules.filter((hiddenModule) => hiddenModule !== module),
          dockPinnedItems: [...state.dockPinnedItems, module],
        };
        saveSettings(newState);
        return newState;
      });
    },

    // Remove item from dock
    removeFromDock: (module: string) => {
      update((state) => {
        const newState = {
          ...state,
          hiddenModules: [...new Set([...state.hiddenModules, module])],
          dockPinnedItems: state.dockPinnedItems.filter((m) => m !== module),
        };
        saveSettings(newState);
        return newState;
      });
    },

    // Create a new folder
    createFolder: (
      name: string,
      x: number,
      y: number,
      color: string = "#3B82F6",
    ) => {
      update((state) => {
        const folderId = `folder-${Date.now()}`;
        const iconId = `icon-${folderId}`;

        const newFolder: DesktopFolder = {
          id: folderId,
          name,
          color,
          iconIds: [],
        };

        const newIcon: DesktopIcon = {
          id: iconId,
          module: "folder",
          label: name,
          x,
          y,
          type: "folder",
          folderId,
          folderColor: color,
        };

        const newState = {
          ...state,
          folders: [...state.folders, newFolder],
          desktopIcons: [...state.desktopIcons, newIcon],
          hiddenModules: state.hiddenModules.filter((module) => module !== "folder"),
        };
        saveSettings(newState);
        return newState;
      });
    },

    // Rename a folder
    renameFolder: (folderId: string, newName: string) => {
      update((state) => {
        const newState = {
          ...state,
          folders: state.folders.map((f) =>
            f.id === folderId ? { ...f, name: newName } : f,
          ),
          desktopIcons: state.desktopIcons.map((icon) =>
            icon.folderId === folderId ? { ...icon, label: newName } : icon,
          ),
        };
        saveSettings(newState);
        return newState;
      });
    },

    // Change folder color
    setFolderColor: (folderId: string, color: string) => {
      update((state) => {
        const newState = {
          ...state,
          folders: state.folders.map((f) =>
            f.id === folderId ? { ...f, color } : f,
          ),
          desktopIcons: state.desktopIcons.map((icon) =>
            icon.folderId === folderId && icon.type === "folder"
              ? { ...icon, folderColor: color }
              : icon,
          ),
        };
        saveSettings(newState);
        return newState;
      });
    },

    // Delete a folder (moves icons back to desktop)
    deleteFolder: (folderId: string) => {
      update((state) => {
        const folder = state.folders.find((f) => f.id === folderId);

        // Move icons out of folder back to desktop
        let updatedIcons = state.desktopIcons.map((icon) => {
          if (folder?.iconIds.includes(icon.id)) {
            return { ...icon, folderId: undefined };
          }
          return icon;
        });

        // Remove the folder icon
        updatedIcons = updatedIcons.filter(
          (icon) => !(icon.type === "folder" && icon.folderId === folderId),
        );

        const newState = {
          ...state,
          folders: state.folders.filter((f) => f.id !== folderId),
          desktopIcons: updatedIcons,
        };
        saveSettings(newState);
        return newState;
      });
    },

    // Move an icon into a folder
    moveIconToFolder: (iconId: string, folderId: string) => {
      update((state) => {
        const newState = {
          ...state,
          folders: state.folders.map((f) => {
            if (f.id === folderId) {
              // Add icon to this folder
              return {
                ...f,
                iconIds: f.iconIds.includes(iconId)
                  ? f.iconIds
                  : [...f.iconIds, iconId],
              };
            }
            // Remove icon from other folders
            return {
              ...f,
              iconIds: f.iconIds.filter((id) => id !== iconId),
            };
          }),
          desktopIcons: state.desktopIcons.map((icon) =>
            icon.id === iconId ? { ...icon, folderId } : icon,
          ),
        };
        saveSettings(newState);
        return newState;
      });
    },

    // Remove an icon from its folder (back to desktop)
    removeIconFromFolder: (iconId: string) => {
      update((state) => {
        const newState = {
          ...state,
          folders: state.folders.map((f) => ({
            ...f,
            iconIds: f.iconIds.filter((id) => id !== iconId),
          })),
          desktopIcons: state.desktopIcons.map((icon) =>
            icon.id === iconId ? { ...icon, folderId: undefined } : icon,
          ),
        };
        saveSettings(newState);
        return newState;
      });
    },

    // Get folder by ID
    getFolder: (folderId: string): DesktopFolder | undefined => {
      let result: DesktopFolder | undefined;
      const unsubscribe = subscribe((state) => {
        result = state.folders.find((f) => f.id === folderId);
      });
      unsubscribe();
      return result;
    },

    // Get icons in a folder
    getIconsInFolder: (folderId: string): DesktopIcon[] => {
      let result: DesktopIcon[] = [];
      const unsubscribe = subscribe((state) => {
        result = state.desktopIcons.filter(
          (icon) => icon.folderId === folderId && icon.type !== "folder",
        );
      });
      unsubscribe();
      return result;
    },

    // Open folder window
    openFolder: (folderId: string) => {
      update((state) => {
        const folder = state.folders.find((f) => f.id === folderId);
        if (!folder) return state;

        // Check if folder window already exists
        const existingWindow = state.windows.find(
          (w) => w.module === `folder-${folderId}` && !w.minimized,
        );
        if (existingWindow) {
          return {
            ...state,
            focusedWindowId: existingWindow.id,
            windowOrder: [
              ...state.windowOrder.filter((id) => id !== existingWindow.id),
              existingWindow.id,
            ],
          };
        }

        const defaults = moduleDefaults.folder;
        const id = `folder-${folderId}-${Date.now()}`;

        const baseX = 150 + Math.random() * 100;
        const baseY = 80 + Math.random() * 50;

        const newWindow: WindowState = {
          id,
          module: `folder-${folderId}`,
          title: folder.name,
          x: baseX,
          y: baseY,
          width: defaults.width,
          height: defaults.height,
          minWidth: defaults.minWidth,
          minHeight: defaults.minHeight,
          minimized: false,
          maximized: false,
        };

        return {
          ...state,
          windows: [...state.windows, newWindow],
          focusedWindowId: id,
          windowOrder: [...state.windowOrder, id],
        };
      });
    },
  };
}
