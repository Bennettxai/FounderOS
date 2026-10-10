/**
 * Module Store — Generated Apps [id] page
 * Manages module installation state: installed flag, loading state,
 * and install error message.
 *
 * Uses the Svelte 5 singleton factory pattern with $state runes.
 */

import { installModule, getModule } from "$lib/api/modules";
import {
  desktop3dStore,
  DYNAMIC_MODULE_COLORS,
} from "$lib/stores/desktop3dStore";
import { windowStore } from "$lib/stores/windowStore";

function createModuleStore() {
  let moduleInstalled = $state(false);
  let isInstallingModule = $state(false);
  let moduleInstallError = $state<string | null>(null);

  return {
    get moduleInstalled() {
      return moduleInstalled;
    },
    set moduleInstalled(v: boolean) {
      moduleInstalled = v;
    },

    get isInstallingModule() {
      return isInstallingModule;
    },
    set isInstallingModule(v: boolean) {
      isInstallingModule = v;
    },

    get moduleInstallError() {
      return moduleInstallError;
    },
    set moduleInstallError(v: string | null) {
      moduleInstallError = v;
    },

    // ── Methods ─────────────────────────────────────────────────────────

    /** Install the module associated with the given module id. */
    async install(moduleId: string) {
      isInstallingModule = true;
      moduleInstallError = null;

      try {
        await installModule(moduleId);

        // Add to 3D desktop + dock
        try {
          const mod = await getModule(moduleId);
          desktop3dStore.addModule({
            id: mod.id,
            title: mod.name,
            icon: mod.icon ?? "box",
            color:
              DYNAMIC_MODULE_COLORS[mod.category] ??
              DYNAMIC_MODULE_COLORS.general,
            category: mod.category,
          });
          windowStore.addToDock(mod.id);
        } catch {
          // Module details fetch failed — still mark as installed
        }

        moduleInstalled = true;
      } catch (err) {
        const msg = err instanceof Error ? err.message : "Install failed";
        if (msg.includes("MANIFEST_INVALID")) {
          moduleInstallError = "The module manifest is invalid.";
        } else if (msg.includes("PROTECTION_VIOLATION")) {
          moduleInstallError = "This module conflicts with a protected module.";
        } else if (msg.includes("SQL_UNSAFE")) {
          moduleInstallError =
            "The module contains unsafe database operations.";
        } else {
          moduleInstallError = msg;
        }
      } finally {
        isInstallingModule = false;
      }
    },

    /** Clear the install error. */
    clearError() {
      moduleInstallError = null;
    },

    /** Reset state when navigating away. */
    reset() {
      moduleInstalled = false;
      isInstallingModule = false;
      moduleInstallError = null;
    },
  };
}

export const moduleStore = createModuleStore();
