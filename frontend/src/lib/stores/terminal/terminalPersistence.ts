/**
 * Terminal Persistence — save/restore tab layout and config to localStorage
 */

import type {
  TerminalTab,
  TerminalConfig,
  PaneNode,
  TerminalProvider,
  PaneMode,
  SplitDirection,
  EnvironmentInfo,
} from "./terminalTypes";
import { DEFAULT_CONFIG, PROVIDER_CONFIGS } from "./terminalTypes";

const STORAGE_KEY = "bos-terminal-state";
const CONFIG_KEY = "bos-terminal-config";

export interface PersistedState {
  tabs: Array<{
    id: string;
    title: string;
    provider: TerminalProvider;
    paneMode: PaneMode;
    rootPaneId: string;
    environment?: EnvironmentInfo;
  }>;
  activeTabId: string | null;
  panes: Record<string, PaneNode>;
}

const VALID_PROVIDERS = new Set<string>(PROVIDER_CONFIGS.map((p) => p.id));
const VALID_MODES = new Set<string>(["shell", "ai", "monaco"]);
const VALID_DIRECTIONS = new Set<string>(["horizontal", "vertical"]);

/**
 * Save terminal layout (tabs + active tab + pane tree structure)
 * Note: WebSocket sessions are NOT persisted — shells reconnect on restore
 */
export function saveTerminalLayout(
  tabs: TerminalTab[],
  activeTabId: string | null,
  panes: Record<string, PaneNode>,
): void {
  try {
    const state = {
      tabs: tabs.map((t) => ({
        id: t.id,
        title: t.title,
        provider: t.provider,
        paneMode: t.paneMode,
        rootPaneId: t.rootPaneId,
        environment: t.environment ?? { mode: "local" },
      })),
      activeTabId,
      panes: serializePanes(panes),
    };
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  } catch {
    // localStorage full or unavailable — silently ignore
  }
}

/**
 * Restore terminal layout from localStorage
 * Returns null if no saved state or invalid data
 */
export function restoreTerminalLayout(): PersistedState | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as {
      tabs?: Array<{
        id?: unknown;
        title?: unknown;
        provider?: unknown;
        paneMode?: unknown;
        rootPaneId?: unknown;
        environment?: unknown;
      }>;
      activeTabId?: unknown;
      panes?: unknown;
    };

    // Validate structure
    if (!parsed || typeof parsed !== "object") return null;
    if (!Array.isArray(parsed.tabs) || parsed.tabs.length === 0) return null;

    // Validate each tab
    const tabs: PersistedState["tabs"] = [];
    for (const tab of parsed.tabs) {
      if (!tab.id || typeof tab.id !== "string") return null;
      if (!tab.rootPaneId || typeof tab.rootPaneId !== "string") return null;
      if (
        typeof tab.provider !== "string" ||
        !VALID_PROVIDERS.has(tab.provider)
      )
        return null;
      if (typeof tab.paneMode !== "string" || !VALID_MODES.has(tab.paneMode))
        return null;
      tabs.push({
        id: tab.id,
        title: typeof tab.title === "string" && tab.title ? tab.title : "Shell",
        provider: tab.provider as TerminalProvider,
        paneMode: tab.paneMode as PaneMode,
        rootPaneId: tab.rootPaneId,
        environment: deserializeEnvironmentInfo(tab.environment),
      });
    }

    // Validate activeTabId references a real tab
    let activeTabId =
      typeof parsed.activeTabId === "string" ? parsed.activeTabId : null;
    if (activeTabId && !tabs.some((t) => t.id === activeTabId)) {
      activeTabId = tabs[0].id;
    }

    // Validate panes exist for each tab
    if (!parsed.panes || typeof parsed.panes !== "object") return null;
    const panes: Record<string, PaneNode> = {};
    const rawPanes = parsed.panes as Record<string, unknown>;
    for (const tab of tabs) {
      const pane = deserializePaneNode(rawPanes[tab.rootPaneId]);
      if (!pane) return null;
      panes[tab.rootPaneId] = pane;
    }

    return { tabs, activeTabId, panes };
  } catch {
    return null;
  }
}

/**
 * Save terminal config (font, theme, cursor)
 */
export function saveTerminalConfig(config: TerminalConfig): void {
  try {
    localStorage.setItem(CONFIG_KEY, JSON.stringify(config));
  } catch {
    // silently ignore
  }
}

/**
 * Restore terminal config from localStorage
 */
export function restoreTerminalConfig(): TerminalConfig {
  try {
    const raw = localStorage.getItem(CONFIG_KEY);
    if (!raw) return DEFAULT_CONFIG;
    const parsed = JSON.parse(raw) as Partial<TerminalConfig>;

    // Validate individual fields
    const config = { ...DEFAULT_CONFIG };

    if (typeof parsed.fontFamily === "string") {
      const portableNerdFonts =
        '"Hack Nerd Font", "JetBrainsMono Nerd Font", "MesloLGS NF", "FiraCode Nerd Font"';
      config.fontFamily = /Nerd Font|MesloLGS NF/.test(parsed.fontFamily)
        ? parsed.fontFamily
        : `${portableNerdFonts}, ${parsed.fontFamily}`;
    }
    if (
      typeof parsed.fontSize === "number" &&
      parsed.fontSize >= 8 &&
      parsed.fontSize <= 32
    ) {
      config.fontSize = parsed.fontSize;
    }
    if (typeof parsed.theme === "string") {
      config.theme = parsed.theme;
    }
    if (
      parsed.cursorStyle === "block" ||
      parsed.cursorStyle === "underline" ||
      parsed.cursorStyle === "bar"
    ) {
      config.cursorStyle = parsed.cursorStyle;
    }
    if (typeof parsed.cursorBlink === "boolean") {
      config.cursorBlink = parsed.cursorBlink;
    }

    return config;
  } catch {
    return DEFAULT_CONFIG;
  }
}

/**
 * Clear all terminal persistence
 */
export function clearTerminalPersistence(): void {
  localStorage.removeItem(STORAGE_KEY);
  localStorage.removeItem(CONFIG_KEY);
}

// Serialize pane tree — strip non-serializable data (service instances, xterm refs)
function serializePanes(
  panes: Record<string, PaneNode>,
): Record<string, unknown> {
  const result: Record<string, unknown> = {};
  for (const [key, node] of Object.entries(panes)) {
    result[key] = serializePaneNode(node);
  }
  return result;
}

function deserializePaneNode(node: unknown): PaneNode | null {
  if (!node || typeof node !== "object") return null;
  const candidate = node as Record<string, unknown>;
  if (typeof candidate.id !== "string") return null;

  if (candidate.type === "leaf") {
    if (
      typeof candidate.mode !== "string" ||
      !VALID_MODES.has(candidate.mode) ||
      typeof candidate.provider !== "string" ||
      !VALID_PROVIDERS.has(candidate.provider)
    ) {
      return null;
    }
    return {
      type: "leaf",
      id: candidate.id,
      mode: candidate.mode as PaneMode,
      provider: candidate.provider as TerminalProvider,
      // Restore the PTY session id so TerminalShell can reconnect to the live
      // shell in the main process instead of spawning a fresh one.
      sessionId:
        typeof candidate.sessionId === "string" ? candidate.sessionId : null,
      filePath:
        typeof candidate.filePath === "string" ? candidate.filePath : undefined,
      fileContent: undefined,
    };
  }

  if (candidate.type === "split") {
    if (
      typeof candidate.direction !== "string" ||
      !VALID_DIRECTIONS.has(candidate.direction) ||
      !Array.isArray(candidate.children) ||
      candidate.children.length !== 2
    ) {
      return null;
    }
    const left = deserializePaneNode(candidate.children[0]);
    const right = deserializePaneNode(candidate.children[1]);
    if (!left || !right) return null;
    const ratio =
      typeof candidate.ratio === "number" && Number.isFinite(candidate.ratio)
        ? Math.max(0.1, Math.min(0.9, candidate.ratio))
        : 0.5;
    return {
      type: "split",
      id: candidate.id,
      direction: candidate.direction as SplitDirection,
      ratio,
      children: [left, right],
    };
  }

  return null;
}

function deserializeEnvironmentInfo(info: unknown): EnvironmentInfo {
  if (!info || typeof info !== "object") return { mode: "local" };
  const candidate = info as Record<string, unknown>;
  const mode =
    candidate.mode === "sandbox" ||
    candidate.mode === "production" ||
    candidate.mode === "local"
      ? candidate.mode
      : "local";

  return {
    mode,
    os: typeof candidate.os === "string" ? candidate.os : undefined,
    containerized:
      typeof candidate.containerized === "boolean"
        ? candidate.containerized
        : undefined,
    agentProcess:
      typeof candidate.agentProcess === "string"
        ? candidate.agentProcess
        : undefined,
    changeCount:
      typeof candidate.changeCount === "number" &&
      Number.isFinite(candidate.changeCount)
        ? candidate.changeCount
        : undefined,
    branchName:
      typeof candidate.branchName === "string"
        ? candidate.branchName
        : undefined,
    sandboxId:
      typeof candidate.sandboxId === "string"
        ? candidate.sandboxId
        : undefined,
    workspacePath:
      typeof candidate.workspacePath === "string"
        ? candidate.workspacePath
        : undefined,
    sandboxRemoteState:
      candidate.sandboxRemoteState === "local-ready" ||
      candidate.sandboxRemoteState === "remote-pending" ||
      candidate.sandboxRemoteState === "remote-attached" ||
      candidate.sandboxRemoteState === "remote-error"
        ? candidate.sandboxRemoteState
        : undefined,
    miosaAccountSource:
      candidate.miosaAccountSource === "local" ||
      candidate.miosaAccountSource === "businessos" ||
      candidate.miosaAccountSource === "user"
        ? candidate.miosaAccountSource
        : undefined,
    miosaSandboxId:
      typeof candidate.miosaSandboxId === "string"
        ? candidate.miosaSandboxId
        : undefined,
    miosaWorkspaceId:
      typeof candidate.miosaWorkspaceId === "string"
        ? candidate.miosaWorkspaceId
        : undefined,
    miosaProjectId:
      typeof candidate.miosaProjectId === "string"
        ? candidate.miosaProjectId
        : undefined,
    miosaPreviewUrl:
      typeof candidate.miosaPreviewUrl === "string"
        ? candidate.miosaPreviewUrl
        : undefined,
    miosaDeploymentUrl:
      typeof candidate.miosaDeploymentUrl === "string"
        ? candidate.miosaDeploymentUrl
        : undefined,
    miosaAttribution:
      deserializeMiosaAttribution(candidate.miosaAttribution),
    miosaError:
      typeof candidate.miosaError === "string"
        ? candidate.miosaError
        : undefined,
  };
}

function deserializeMiosaAttribution(info: unknown): EnvironmentInfo["miosaAttribution"] {
  if (!info || typeof info !== "object") return undefined;
  const candidate = info as Record<string, unknown>;
  return {
    externalWorkspaceId:
      typeof candidate.externalWorkspaceId === "string"
        ? candidate.externalWorkspaceId
        : undefined,
    externalWorkspaceSlug:
      typeof candidate.externalWorkspaceSlug === "string"
        ? candidate.externalWorkspaceSlug
        : undefined,
    externalUserId:
      typeof candidate.externalUserId === "string"
        ? candidate.externalUserId
        : undefined,
    externalProjectId:
      typeof candidate.externalProjectId === "string"
        ? candidate.externalProjectId
        : undefined,
  };
}

function serializePaneNode(node: PaneNode): unknown {
  if (node.type === "leaf") {
    return {
      type: "leaf",
      id: node.id,
      mode: node.mode,
      provider: node.provider,
      // Persist the local PTY session id so the shell can RECONNECT after a
      // reload (the PTY lives in the main process and survives a refresh).
      sessionId: node.sessionId ?? null,
      filePath: node.filePath,
      fileContent: undefined, // don't persist file contents
    };
  }
  return {
    type: "split",
    id: node.id,
    direction: node.direction,
    ratio: node.ratio,
    children: [
      serializePaneNode(node.children[0]),
      serializePaneNode(node.children[1]),
    ],
  };
}
