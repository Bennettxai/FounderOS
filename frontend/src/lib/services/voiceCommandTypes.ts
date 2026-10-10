/**
 * Voice Command Types
 *
 * Shared type definitions for the voice command system.
 */

import type { ModuleId } from "$lib/stores/desktop3dStore";

export type VoiceCommand =
  | { type: "enter_edit_mode" }
  | { type: "exit_edit_mode" }
  | { type: "save_layout"; name: string }
  | { type: "load_layout"; name: string }
  | { type: "delete_layout"; name: string }
  | { type: "reset_layout" }
  | { type: "open_layout_manager" }
  | { type: "focus_module"; module: ModuleId }
  | { type: "open_module"; module: ModuleId }
  | { type: "close_module"; module: ModuleId }
  | { type: "close_all_windows" }
  | { type: "minimize_window" }
  | { type: "maximize_window" }
  | { type: "unfocus" }
  | {
      type: "resize_window";
      direction: "wider" | "narrower" | "taller" | "shorter";
    }
  | { type: "switch_view"; view: "orb" | "grid" }
  | { type: "toggle_auto_rotate" }
  | { type: "rotate_left" }
  | { type: "rotate_right" }
  | { type: "stop_rotation" }
  | { type: "rotate_faster" }
  | { type: "rotate_slower" }
  | { type: "zoom_in" }
  | { type: "zoom_out" }
  | { type: "reset_zoom" }
  | { type: "expand_orb" }
  | { type: "contract_orb" }
  | { type: "increase_grid_spacing" }
  | { type: "decrease_grid_spacing" }
  | { type: "more_grid_columns" }
  | { type: "less_grid_columns" }
  | { type: "next_window" }
  | { type: "previous_window" }
  | { type: "help" }
  | { type: "unknown"; text: string };

/** All recognized module IDs for voice navigation */
export const VOICE_MODULE_IDS: ModuleId[] = [
  "dashboard",
  "chat",
  "tasks",
  "projects",
  "team",
  "clients",
  "tables",
  "communication",
  "pages",
  "nodes",
  "daily",
  "settings",
  "terminal",
  "help",
  "agents",
  "crm",
  "integrations",
  "knowledge",
  "notifications",
  "profile",
  "voice-notes",
  "usage",
];
