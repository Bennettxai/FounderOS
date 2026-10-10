/**
 * Generated Apps Store Types
 * Shared interfaces for the generated-apps/[id] page stores.
 */

import type { OSAFile, FileTreeNode } from "$lib/components/osa/types";
import type { Version } from "$lib/types/versions";
import type { BackendVersionInfo } from "$lib/api/versions/types";
import type { GeneratedApp } from "$lib/stores/generatedAppsStore";

export type {
  OSAFile,
  FileTreeNode,
  Version,
  BackendVersionInfo,
  GeneratedApp,
};

// Re-export tab type for the editor workspace
export type ActiveTab = "preview" | "code" | "terminal";
