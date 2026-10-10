/**
 * Desktop 3D Types
 * All shared type definitions and constants for the 3D desktop system.
 */

// Core modules that are always visible - only modules with actual routes.
export const CORE_MODULES = [
  "dashboard",
  "knowledge",
  "tasks",
  "projects",
  "team",
  "relationships",
  "apps",
  "calendar",
  "inbox",
  "offers",
  "pipelines",
  "settings",
  "help",
] as const;

// Built-in modules (static - these ship with every OS).
// Keep this list aligned to actual SvelteKit routes so 3D panels do not render
// stale 404 windows.
export const BUILTIN_MODULES = [
  "dashboard",
  "admin",
  "agents",
  "analytics",
  "apps",
  "assets",
  "boards",
  "builders",
  "calendar",
  "campaigns",
  "clients",
  "communication",
  "connectors",
  "content",
  "crm",
  "data",
  "deliverables",
  "drive",
  "engines",
  "finance",
  "glossary",
  "help",
  "inbox",
  "intelligence",
  "knowledge",
  "offers",
  "personas",
  "pipelines",
  "projects",
  "relationships",
  "resources",
  "rhythm",
  "settings",
  "sites",
  "tasks",
  "team",
  "notifications",
] as const;

// Keep ALL_MODULES as alias for backwards compatibility
export const ALL_MODULES = BUILTIN_MODULES;

export type BuiltinModuleId = (typeof BUILTIN_MODULES)[number];
// ModuleId now accepts both built-in and dynamic string IDs
export type ModuleId = BuiltinModuleId | (string & {});
export type ViewMode = "orb" | "grid" | "focused";

export interface Window3DState {
  id: string;
  module: ModuleId;
  title: string;
  position: [number, number, number];
  targetPosition: [number, number, number];
  rotation: [number, number, number];
  scale: number;
  targetScale: number;
  opacity: number;
  targetOpacity: number;
  isCore: boolean;
  isOpen: boolean;
  isFocused: boolean;
  lastFocused: number;
  color: string;
  // Window dimensions (resizable)
  width: number;
  height: number;
}

export interface Desktop3DState {
  viewMode: ViewMode;
  windows: Window3DState[];
  focusedWindowId: string | null;
  sphereRadius: number;
  cameraDistance: number; // Camera distance from scene (zoom control)
  gridColumns: number;
  gridSpacing: number;
  autoRotate: boolean;
  animating: boolean;
  // Gesture-based camera rotation
  cameraRotationDelta: { x: number; y: number }; // Delta for gesture-based rotation
  gestureDragging: boolean; // True when actively dragging with gestures
}

// Config shape for dynamically installed/generated modules
export interface DynamicModuleConfig {
  id: string;
  title: string;
  icon?: string;
  color?: string;
  category?: string;
}
