import { beforeEach, describe, expect, it } from "vitest";
import {
  clearTerminalPersistence,
  restoreTerminalLayout,
  saveTerminalLayout,
} from "./terminalPersistence";
import type { PaneNode, TerminalTab } from "./terminalTypes";

describe("terminal persistence", () => {
  beforeEach(() => {
    clearTerminalPersistence();
  });

  it("restores local PTY session ids so refresh can reconnect shells", () => {
    const tabs: TerminalTab[] = [
      {
        id: "tab-1",
        title: "Shell",
        provider: "shell",
        paneMode: "shell",
        rootPaneId: "pane-1",
        sessionId: null,
        isActive: true,
        environment: { mode: "local" },
      },
    ];
    const panes: Record<string, PaneNode> = {
      "pane-1": {
        type: "leaf",
        id: "pane-1",
        mode: "shell",
        provider: "shell",
        sessionId: "pty-live-1",
      },
    };

    saveTerminalLayout(tabs, "tab-1", panes);

    const restored = restoreTerminalLayout();
    expect(restored?.panes["pane-1"]).toMatchObject({
      type: "leaf",
      sessionId: "pty-live-1",
    });
  });

  it("restores session ids inside split pane trees", () => {
    const tabs: TerminalTab[] = [
      {
        id: "tab-1",
        title: "Shell",
        provider: "shell",
        paneMode: "shell",
        rootPaneId: "root",
        sessionId: null,
        isActive: true,
        environment: { mode: "local" },
      },
    ];
    const panes: Record<string, PaneNode> = {
      root: {
        type: "split",
        id: "root",
        direction: "horizontal",
        ratio: 0.5,
        children: [
          {
            type: "leaf",
            id: "left",
            mode: "shell",
            provider: "shell",
            sessionId: "pty-left",
          },
          {
            type: "leaf",
            id: "right",
            mode: "shell",
            provider: "codex",
            sessionId: "pty-right",
          },
        ],
      },
    };

    saveTerminalLayout(tabs, "tab-1", panes);

    const restored = restoreTerminalLayout();
    const root = restored?.panes.root;
    expect(root).toMatchObject({ type: "split" });
    if (root?.type !== "split") throw new Error("expected split root");
    expect(root.children[0]).toMatchObject({ type: "leaf", sessionId: "pty-left" });
    expect(root.children[1]).toMatchObject({ type: "leaf", sessionId: "pty-right" });
  });
});
