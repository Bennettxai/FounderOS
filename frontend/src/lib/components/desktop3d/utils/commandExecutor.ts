/**
 * Command Executor
 *
 * Translates a parsed VoiceCommand into concrete store / service calls.
 * Pure function — all side effects happen through the injected stores.
 */

import type { VoiceCommand } from "$lib/services/voiceCommands";
import type { desktop3dStore as Desktop3DStoreType } from "$lib/stores/desktop3dStore";
import type { desktop3dLayoutStore as Desktop3DLayoutStoreType } from "$lib/stores/desktop3dLayoutStore";
import type { OSAVoiceService } from "$lib/services/osaVoice";
import { get } from "svelte/store";

interface CommandDeps {
  desktop3dStore: typeof Desktop3DStoreType;
  desktop3dLayoutStore: typeof Desktop3DLayoutStoreType;
  osaVoiceService: OSAVoiceService;
  /** Callback invoked when a command wants to open the layout manager UI */
  openLayoutManager: () => void;
  /** Reactive store of currently open windows — used for module lookups */
  openWindows: Parameters<typeof get>[0];
  /** Reactive store of the full desktop3d state — used for focusedWindowId */
  desktop3dState: Parameters<typeof get>[0];
}

/**
 * Execute the action described by `command` using the injected dependencies.
 *
 * Does NOT handle the 'unknown' type — callers should route that to the
 * conversation handler before calling this function.
 */
export function executeCommandAction(
  command: VoiceCommand,
  deps: CommandDeps,
): void {
  const {
    desktop3dStore,
    desktop3dLayoutStore,
    osaVoiceService,
    openLayoutManager,
    openWindows,
    desktop3dState,
  } = deps;

  const windows = get(openWindows) as Array<{ id: string; module: string }>;
  const state = get(desktop3dState) as { focusedWindowId: string | null };

  switch (command.type) {
    case "enter_edit_mode":
      desktop3dLayoutStore.enterEditMode();
      break;

    case "exit_edit_mode":
      desktop3dLayoutStore.exitEditMode();
      break;

    case "save_layout":
      desktop3dLayoutStore.saveLayout(command.name);
      break;

    case "load_layout": {
      const layouts = get(desktop3dLayoutStore).layouts as Array<{
        id: string;
        name: string;
      }>;
      const layout = layouts.find(
        (l) => l.name.toLowerCase() === command.name.toLowerCase(),
      );
      if (layout) {
        desktop3dLayoutStore.loadLayout(layout.id);
      } else {
        console.warn("[commandExecutor] Layout not found:", command.name);
        osaVoiceService.speak(
          `I couldn't find a layout called ${command.name}`,
        );
      }
      break;
    }

    case "delete_layout": {
      const layouts = get(desktop3dLayoutStore).layouts as Array<{
        id: string;
        name: string;
      }>;
      const deleteLayout = layouts.find(
        (l) => l.name.toLowerCase() === command.name.toLowerCase(),
      );
      if (deleteLayout) {
        desktop3dLayoutStore.deleteLayout(deleteLayout.id);
      }
      break;
    }

    case "open_layout_manager":
      openLayoutManager();
      break;

    case "reset_layout":
      desktop3dLayoutStore.resetToDefault();
      break;

    case "focus_module": {
      const win = windows.find((w) => w.module === command.module);
      if (win) {
        desktop3dStore.focusWindow(win.id);
      } else {
        desktop3dStore.openWindow(command.module);
      }
      break;
    }

    case "open_module": {
      desktop3dStore.openWindow(command.module);
      break;
    }

    case "close_module": {
      const closeWin = windows.find((w) => w.module === command.module);
      if (closeWin) {
        desktop3dStore.closeWindow(closeWin.id);
      } else {
        console.warn(
          `[commandExecutor] Window "${command.module}" not found (not open)`,
        );
      }
      break;
    }

    case "close_all_windows":
      desktop3dStore.closeAllWindows();
      break;

    case "minimize_window":
      desktop3dStore.unfocusWindow();
      break;

    case "maximize_window":
      if (state.focusedWindowId) {
        desktop3dStore.resizeFocusedWindow(200, 150);
      } else if (windows.length > 0) {
        desktop3dStore.focusWindow(windows[0].id);
      }
      break;

    case "switch_view":
      if (command.view === "orb") {
        desktop3dStore.setViewMode("orb");
      } else {
        desktop3dStore.setViewMode("grid");
      }
      break;

    case "toggle_auto_rotate":
      desktop3dStore.toggleAutoRotate();
      break;

    case "rotate_left":
      desktop3dStore.setAutoRotate(false);
      // TODO: Implement manual rotation control
      break;

    case "rotate_right":
      desktop3dStore.setAutoRotate(false);
      // TODO: Implement manual rotation control
      break;

    case "stop_rotation":
      desktop3dStore.setAutoRotate(false);
      break;

    case "rotate_faster":
      desktop3dStore.adjustRotationSpeed(0.2);
      break;

    case "rotate_slower":
      desktop3dStore.adjustRotationSpeed(-0.2);
      break;

    case "zoom_in":
      desktop3dStore.adjustCameraDistance(-1.0);
      break;

    case "zoom_out":
      desktop3dStore.adjustCameraDistance(1.0);
      break;

    case "reset_zoom":
      desktop3dStore.resetCameraDistance();
      break;

    case "expand_orb":
      desktop3dStore.adjustSphereRadius(3.0);
      break;

    case "contract_orb":
      desktop3dStore.adjustSphereRadius(-3.0);
      break;

    case "increase_grid_spacing":
      desktop3dStore.adjustGridSpacing(10);
      break;

    case "decrease_grid_spacing":
      desktop3dStore.adjustGridSpacing(-10);
      break;

    case "more_grid_columns":
      desktop3dStore.adjustGridColumns(1);
      break;

    case "less_grid_columns":
      desktop3dStore.adjustGridColumns(-1);
      break;

    case "unfocus":
      desktop3dStore.unfocusWindow();
      break;

    case "resize_window": {
      const deltaMap: Record<string, [number, number]> = {
        wider: [100, 0],
        narrower: [-100, 0],
        taller: [0, 100],
        shorter: [0, -100],
      };
      const [widthDelta, heightDelta] = deltaMap[command.direction];
      desktop3dStore.resizeFocusedWindow(widthDelta, heightDelta);
      break;
    }

    case "next_window":
      desktop3dStore.focusNext();
      break;

    case "previous_window":
      desktop3dStore.focusPrevious();
      break;

    case "help":
      desktop3dStore.openWindow("help");
      desktop3dStore.focusWindow("help");
      break;

    case "unknown":
      // Callers should handle 'unknown' before calling this function.
      // Log a warning and do nothing.
      console.warn(
        '[commandExecutor] executeCommandAction called with type "unknown" — route to conversation handler instead',
      );
      break;
  }
}
