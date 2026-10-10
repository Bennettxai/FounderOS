/**
 * Terminal Store — Svelte 5 runes state management
 * Manages tabs, panes, providers, and config for the terminal app
 */

import type {
  TerminalTab,
  TerminalConfig,
  PaneNode,
  PaneLeaf,
  PaneSplit,
  TerminalProvider,
  PaneMode,
  SplitDirection,
  EnvironmentMode,
  EnvironmentInfo,
} from "./terminalTypes";
import { DEFAULT_CONFIG } from "./terminalTypes";
import {
  saveTerminalLayout,
  saveTerminalConfig,
  restoreTerminalConfig,
  restoreTerminalLayout,
} from "./terminalPersistence";

const MAX_TABS = 20;
const MAX_SPLIT_DEPTH = 4;

function createStore() {
  // Core state
  let tabs = $state<TerminalTab[]>([]);
  let activeTabId = $state<string | null>(null);
  let panes = $state<Record<string, PaneNode>>({});
  let config = $state<TerminalConfig>(restoreTerminalConfig());
  let focusedPaneId = $state<string | null>(null);
  let launchEnvironmentMode = $state<EnvironmentMode>("local");

  // Derived
  const activeTab = $derived(tabs.find((t) => t.id === activeTabId) ?? null);
  const activePane = $derived(
    activeTab ? (panes[activeTab.rootPaneId] ?? null) : null,
  );
  const tabCount = $derived(tabs.length);

  // --- Tab Management ---

  function createTab(
    provider: TerminalProvider = "shell",
    environmentInfo?: EnvironmentInfo,
  ): string | null {
    if (tabs.length >= MAX_TABS) return null;

    const tabId = crypto.randomUUID();
    const paneId = crypto.randomUUID();
    const environment = buildEnvironmentInfo(
      environmentInfo ?? { mode: launchEnvironmentMode },
      tabId,
    );
    // ALL providers use shell mode — agent providers auto-launch their CLI command after connect
    const mode: PaneMode = "shell";

    const leaf: PaneLeaf = {
      type: "leaf",
      id: paneId,
      mode,
      provider,
      sessionId: null,
    };

    const tab: TerminalTab = {
      id: tabId,
      title:
        provider === "shell"
          ? "Shell"
          : provider.charAt(0).toUpperCase() + provider.slice(1),
      provider,
      paneMode: mode,
      rootPaneId: paneId,
      sessionId: null,
      isActive: true,
      environment,
    };

    // Deactivate other tabs
    tabs = tabs.map((t) => ({ ...t, isActive: false }));
    tabs = [...tabs, tab];
    panes = { ...panes, [paneId]: leaf };
    activeTabId = tabId;
    focusedPaneId = paneId;

    persist();
    return tabId;
  }

  function closeTab(tabId: string): void {
    const idx = tabs.findIndex((t) => t.id === tabId);
    if (idx === -1) return;

    const tab = tabs[idx];

    // KILL the PTY sessions in this tab's panes. Closing a tab is the ONLY thing
    // that terminates a local shell - a refresh detaches and reconnects instead.
    const root = panes[tab.rootPaneId];
    if (root) {
      const killApi = (
        globalThis as unknown as {
          window?: {
            electron?: { terminal?: { kill?: (id: string) => void } };
          };
        }
      ).window?.electron?.terminal;
      const walk = (node: PaneNode | undefined): void => {
        if (!node) return;
        if (node.type === "leaf") {
          if (node.sessionId && node.sessionId !== "cloud") {
            try {
              killApi?.kill?.(node.sessionId);
            } catch {
              /* best effort */
            }
          }
        } else {
          walk(node.children[0]);
          walk(node.children[1]);
        }
      };
      walk(root);
    }

    // Remove pane tree
    const newPanes = { ...panes };
    delete newPanes[tab.rootPaneId];
    panes = newPanes;

    // Remove tab
    tabs = tabs.filter((t) => t.id !== tabId);

    // Switch to adjacent tab if we closed the active one
    if (activeTabId === tabId) {
      if (tabs.length > 0) {
        const newIdx = Math.min(idx, tabs.length - 1);
        switchTab(tabs[newIdx].id);
      } else {
        activeTabId = null;
        focusedPaneId = null;
      }
    }

    // Ensure at least one tab exists
    if (tabs.length === 0) {
      createTab("shell");
    }

    persist();
  }

  function switchTab(tabId: string): void {
    tabs = tabs.map((t) => ({ ...t, isActive: t.id === tabId }));
    activeTabId = tabId;

    // Focus root pane of new tab
    const tab = tabs.find((t) => t.id === tabId);
    if (tab) {
      const root = panes[tab.rootPaneId];
      if (root) {
        const firstLeaf = getFirstLeaf(root);
        focusedPaneId = firstLeaf?.id ?? null;
      }
    }

    persist();
  }

  function switchTabByIndex(index: number): void {
    if (index >= 0 && index < tabs.length) {
      switchTab(tabs[index].id);
    }
  }

  function switchTabRelative(delta: number): void {
    if (tabs.length === 0) return;
    const currentIdx = tabs.findIndex((t) => t.id === activeTabId);
    const newIdx =
      (((currentIdx + delta) % tabs.length) + tabs.length) % tabs.length;
    switchTab(tabs[newIdx].id);
  }

  function renameTab(tabId: string, title: string): void {
    tabs = tabs.map((t) => (t.id === tabId ? { ...t, title } : t));
    persist();
  }

  // --- Provider / Mode Switching ---

  function setTabProvider(tabId: string, provider: TerminalProvider): void {
    const mode: PaneMode = "shell";
    tabs = tabs.map((t) =>
      t.id === tabId ? { ...t, provider, paneMode: mode } : t,
    );

    // Update ALL leaves in the pane tree
    const tab = tabs.find((t) => t.id === tabId);
    if (tab) {
      const tree = panes[tab.rootPaneId];
      if (tree) {
        const updated = updateAllLeavesInTree(tree, { provider, mode });
        panes = { ...panes, [tab.rootPaneId]: updated };
      }
    }

    persist();
  }

  function setTabMode(tabId: string, mode: PaneMode): void {
    tabs = tabs.map((t) => (t.id === tabId ? { ...t, paneMode: mode } : t));

    const tab = tabs.find((t) => t.id === tabId);
    if (tab) {
      const tree = panes[tab.rootPaneId];
      if (tree) {
        const updated = updateAllLeavesInTree(tree, { mode });
        panes = { ...panes, [tab.rootPaneId]: updated };
      }
    }

    persist();
  }

  // --- Focus Management ---

  function setFocusedPane(paneId: string): void {
    focusedPaneId = paneId;
  }

  // --- Split Pane Management ---

  function splitPane(paneId: string, direction: SplitDirection): string | null {
    const rootId = findRootForPane(paneId);
    if (!rootId) return null;

    const tree = panes[rootId];
    if (!tree) return null;

    // Check depth limit
    if (getTreeDepth(tree) >= MAX_SPLIT_DEPTH) return null;

    const newPaneId = crypto.randomUUID();
    const splitId = crypto.randomUUID();

    const targetLeaf = findPaneNode(tree, paneId);
    if (!targetLeaf || targetLeaf.type !== "leaf") return null;

    const newLeaf: PaneLeaf = {
      type: "leaf",
      id: newPaneId,
      mode: targetLeaf.mode,
      provider: targetLeaf.provider,
      sessionId: null,
    };

    const splitNode: PaneSplit = {
      type: "split",
      id: splitId,
      direction,
      ratio: 0.5,
      children: [{ ...targetLeaf }, newLeaf],
    };

    const newTree = replacePaneNode(tree, paneId, splitNode);
    panes = { ...panes, [rootId]: newTree };
    focusedPaneId = newPaneId;

    persist();
    return newPaneId;
  }

  function closePaneInSplit(paneId: string): void {
    const rootId = findRootForPane(paneId);
    if (!rootId) return;

    const tree = panes[rootId];
    if (!tree) return;

    // Can't close the only pane
    if (tree.type === "leaf" && tree.id === paneId) return;

    const newTree = removePaneFromTree(tree, paneId);
    if (newTree) {
      panes = { ...panes, [rootId]: newTree };
      // Focus the remaining pane
      const firstLeaf = getFirstLeaf(newTree);
      if (firstLeaf) focusedPaneId = firstLeaf.id;
      persist();
    }
  }

  function resizeSplit(splitId: string, ratio: number): void {
    const clamped = Math.max(0.1, Math.min(0.9, ratio));
    const newPanes: Record<string, PaneNode> = {};
    for (const [key, tree] of Object.entries(panes)) {
      newPanes[key] = updateSplitRatio(tree, splitId, clamped);
    }
    panes = newPanes;
    persist();
  }

  // --- Pane Session ---

  function setPaneSessionId(paneId: string, sessionId: string): void {
    const rootId = findRootForPane(paneId);
    if (!rootId) return;

    const tree = panes[rootId];
    if (!tree) return;

    const updated = updateLeafInTree(tree, paneId, { sessionId });
    panes = { ...panes, [rootId]: updated };
    // Persist immediately: the sessionId is what lets a shell RECONNECT after a
    // refresh. Without this write it lived only in memory and was lost on reload,
    // so reconnect never fired and a fresh shell was spawned every time.
    persist();
  }

  // --- Pane Mode (per-pane, for splits) ---

  function setPaneMode(
    paneId: string,
    mode: PaneMode,
    provider?: TerminalProvider,
  ): void {
    const rootId = findRootForPane(paneId);
    if (!rootId) return;

    const tree = panes[rootId];
    if (!tree) return;

    const updates: Partial<PaneLeaf> = { mode };
    if (provider !== undefined) updates.provider = provider;

    const updated = updateLeafInTree(tree, paneId, updates);
    panes = { ...panes, [rootId]: updated };
    persist();
  }

  // --- Monaco file ---

  function setPaneFile(
    paneId: string,
    filePath: string,
    fileContent: string,
  ): void {
    const rootId = findRootForPane(paneId);
    if (!rootId) return;

    const tree = panes[rootId];
    if (!tree) return;

    const updated = updateLeafInTree(tree, paneId, {
      mode: "monaco",
      filePath,
      fileContent,
    });
    panes = { ...panes, [rootId]: updated };
    persist();
  }

  // --- Config ---

  function updateConfig(partial: Partial<TerminalConfig>): void {
    config = { ...config, ...partial };
    saveTerminalConfig(config);
  }

  // --- Tree Helpers ---

  function findRootForPane(paneId: string): string | null {
    for (const [rootId, tree] of Object.entries(panes)) {
      if (findPaneNode(tree, paneId)) return rootId;
    }
    return null;
  }

  function findPaneNode(node: PaneNode, id: string): PaneNode | null {
    if (node.id === id) return node;
    if (node.type === "split") {
      return (
        findPaneNode(node.children[0], id) || findPaneNode(node.children[1], id)
      );
    }
    return null;
  }

  function getFirstLeaf(node: PaneNode): PaneLeaf | null {
    if (node.type === "leaf") return node;
    return getFirstLeaf(node.children[0]);
  }

  function getTreeDepth(node: PaneNode): number {
    if (node.type === "leaf") return 0;
    return (
      1 +
      Math.max(getTreeDepth(node.children[0]), getTreeDepth(node.children[1]))
    );
  }

  function replacePaneNode(
    node: PaneNode,
    targetId: string,
    replacement: PaneNode,
  ): PaneNode {
    if (node.id === targetId) return replacement;
    if (node.type === "split") {
      return {
        ...node,
        children: [
          replacePaneNode(node.children[0], targetId, replacement),
          replacePaneNode(node.children[1], targetId, replacement),
        ],
      };
    }
    return node;
  }

  function removePaneFromTree(node: PaneNode, paneId: string): PaneNode | null {
    if (node.type === "leaf") return node.id === paneId ? null : node;

    if (node.children[0].type === "leaf" && node.children[0].id === paneId) {
      return node.children[1];
    }
    if (node.children[1].type === "leaf" && node.children[1].id === paneId) {
      return node.children[0];
    }

    const left = removePaneFromTree(node.children[0], paneId);
    const right = removePaneFromTree(node.children[1], paneId);

    if (!left) return right;
    if (!right) return left;

    return { ...node, children: [left, right] };
  }

  function updateSplitRatio(
    node: PaneNode,
    splitId: string,
    ratio: number,
  ): PaneNode {
    if (node.type === "leaf") return node;
    if (node.id === splitId) return { ...node, ratio };
    return {
      ...node,
      children: [
        updateSplitRatio(node.children[0], splitId, ratio),
        updateSplitRatio(node.children[1], splitId, ratio),
      ],
    };
  }

  function updateLeafInTree(
    node: PaneNode,
    leafId: string,
    updates: Partial<PaneLeaf>,
  ): PaneNode {
    if (node.type === "leaf") {
      if (node.id === leafId) return { ...node, ...updates };
      return node;
    }
    return {
      ...node,
      children: [
        updateLeafInTree(node.children[0], leafId, updates),
        updateLeafInTree(node.children[1], leafId, updates),
      ],
    };
  }

  function updateAllLeavesInTree(
    node: PaneNode,
    updates: Partial<PaneLeaf>,
  ): PaneNode {
    if (node.type === "leaf") return { ...node, ...updates };
    return {
      ...node,
      children: [
        updateAllLeavesInTree(node.children[0], updates),
        updateAllLeavesInTree(node.children[1], updates),
      ],
    };
  }

  function collectLeaves(node: PaneNode): PaneLeaf[] {
    if (node.type === "leaf") return [node];
    return [
      ...collectLeaves(node.children[0]),
      ...collectLeaves(node.children[1]),
    ];
  }

  // --- Environment Mode ---

  function setEnvironmentMode(tabId: string, mode: EnvironmentMode): void {
    tabs = tabs.map((tab) =>
      tab.id === tabId
        ? {
            ...tab,
            environment: buildEnvironmentInfo(
              { ...(tab.environment ?? { mode: "local" }), mode },
              tab.id,
            ),
          }
        : tab,
    );
    persist();
  }

  function getEnvironmentInfo(tabId: string): EnvironmentInfo {
    return tabs.find((tab) => tab.id === tabId)?.environment ?? { mode: "local" };
  }

  function updateEnvironmentInfo(
    tabId: string,
    info: Partial<EnvironmentInfo>,
  ): void {
    tabs = tabs.map((tab) =>
      tab.id === tabId
        ? {
            ...tab,
            environment: buildEnvironmentInfo(
              { ...(tab.environment ?? { mode: "local" }), ...info },
              tab.id,
            ),
          }
        : tab,
    );
    persist();
  }

  function setLaunchEnvironmentMode(mode: EnvironmentMode): void {
    launchEnvironmentMode = mode;
  }

  function buildEnvironmentInfo(
    info: EnvironmentInfo,
    tabId: string,
  ): EnvironmentInfo {
    if (info.mode !== "sandbox") {
      return { ...info, mode: info.mode };
    }

    const sandboxId = info.sandboxId ?? `sbx-${tabId.slice(0, 8)}`;
    return {
      ...info,
      mode: "sandbox",
      sandboxId,
      branchName: info.branchName ?? sandboxId,
      changeCount: info.changeCount ?? 0,
      sandboxRemoteState: info.sandboxRemoteState ?? "local-ready",
      miosaAccountSource: info.miosaAccountSource ?? "local",
    };
  }

  // --- Persistence ---

  function persist(): void {
    saveTerminalLayout(tabs, activeTabId, panes);
  }

  // --- Initialize ---

  function init(): void {
    if (tabs.length === 0) {
      const restored = restoreTerminalLayout();
      if (restored) {
        tabs = restored.tabs.map((tab) => ({
          id: tab.id,
          title: tab.title,
          provider: tab.provider as TerminalProvider,
          paneMode: tab.paneMode as PaneMode,
          rootPaneId: tab.rootPaneId,
          sessionId: null,
          isActive: tab.id === restored.activeTabId,
          environment: tab.environment ?? { mode: "local" },
        }));
        panes = restored.panes;
        activeTabId = restored.activeTabId ?? tabs[0]?.id ?? null;
        const activeRootId =
          tabs.find((tab) => tab.id === activeTabId)?.rootPaneId ?? null;
        focusedPaneId = activeRootId
          ? (getFirstLeaf(panes[activeRootId])?.id ?? null)
          : null;
        if (activeTabId) {
          tabs = tabs.map((tab) => ({
            ...tab,
            isActive: tab.id === activeTabId,
          }));
        }
        const shellTab = tabs.find((tab) => tab.provider === "shell");
        if (shellTab) {
          switchTab(shellTab.id);
        } else {
          createTab("shell", { mode: "local" });
        }
        return;
      }
      createTab("shell");
    }
  }

  return {
    get tabs() {
      return tabs;
    },
    get activeTabId() {
      return activeTabId;
    },
    get activeTab() {
      return activeTab;
    },
    get activePane() {
      return activePane;
    },
    get panes() {
      return panes;
    },
    get config() {
      return config;
    },
    get tabCount() {
      return tabCount;
    },
    get focusedPaneId() {
      return focusedPaneId;
    },
    get launchEnvironmentMode() {
      return launchEnvironmentMode;
    },

    createTab,
    closeTab,
    switchTab,
    switchTabByIndex,
    switchTabRelative,
    renameTab,
    setTabProvider,
    setTabMode,
    setFocusedPane,
    splitPane,
    closePaneInSplit,
    resizeSplit,
    setPaneSessionId,
    setPaneMode,
    setPaneFile,
    updateConfig,
    collectLeaves,
    findPaneNode,
    getFirstLeaf,
    setEnvironmentMode,
    getEnvironmentInfo,
    updateEnvironmentInfo,
    setLaunchEnvironmentMode,
    init,

    MAX_TABS,
    MAX_SPLIT_DEPTH,
  };
}

export const terminalStore = createStore();
