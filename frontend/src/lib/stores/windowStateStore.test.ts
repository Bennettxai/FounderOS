import { beforeEach, describe, expect, it } from "vitest";
import { get } from "svelte/store";
import { windowStore } from "./windowStateStore";
import { desktopSettings } from "./desktopThemeStore";
import { loadSavedSettings, STORAGE_KEY } from "./desktopPersistence";

describe("windowStore desktop spaces", () => {
  beforeEach(() => {
    localStorage.clear();
    desktopSettings.reset();
    windowStore.reset();
  });

  it("keeps all desktop spaces registered when switching between them", () => {
    const personalId = get(windowStore).activeDesktopId;

    windowStore.createDesktopSpace("Client Work");
    const clientDesktopId = get(windowStore).activeDesktopId;

    expect(get(windowStore).desktopSpaces.map((space) => space.id)).toEqual([
      personalId,
      clientDesktopId,
    ]);

    windowStore.switchDesktopSpace(personalId);
    expect(get(windowStore).activeDesktopId).toBe(personalId);
    expect(get(windowStore).desktopSpaces.map((space) => space.id)).toEqual([
      personalId,
      clientDesktopId,
    ]);

    windowStore.switchDesktopSpace(clientDesktopId);
    expect(get(windowStore).activeDesktopId).toBe(clientDesktopId);
    expect(get(windowStore).desktopSpaces.map((space) => space.id)).toEqual([
      personalId,
      clientDesktopId,
    ]);
  });

  it("starts new desktops empty so users choose their modules", () => {
    windowStore.createDesktopSpace("Client Work");
    const state = get(windowStore);

    expect(state.desktopIcons).toEqual([]);
    expect(state.dockPinnedItems).toEqual(["finder"]);
    expect(state.hiddenModules).toContain("workspace-app-*");
    expect(state.windows).toEqual([]);
  });

  it("can create a workspace desktop with a stable server-ready id", () => {
    const id = "11111111-1111-4111-8111-111111111111";

    windowStore.createDesktopSpace("Team Room", { id, kind: "workspace" });
    const state = get(windowStore);
    const active = state.desktopSpaces.find((space) => space.id === id);

    expect(state.activeDesktopId).toBe(id);
    expect(active).toMatchObject({
      id,
      name: "Team Room",
      kind: "workspace",
      desktopIcons: [],
      dockPinnedItems: ["finder"],
    });
  });

  it("can upgrade an old local canvas desktop id to a workspace-safe id", () => {
    windowStore.createDesktopSpace("Infinity Desktop", {
      id: "local-infinity",
      kind: "workspace",
    });
    windowStore.openWindow("terminal", { x: -500, y: 220 });
    windowStore.addDesktopIcon("apps", "Apps");

    const nextId = "22222222-2222-4222-8222-222222222222";
    windowStore.rekeyDesktopSpace("local-infinity", nextId);

    const state = get(windowStore);
    const upgraded = state.desktopSpaces.find((space) => space.id === nextId);

    expect(state.activeDesktopId).toBe(nextId);
    expect(state.desktopSpaces.some((space) => space.id === "local-infinity")).toBe(false);
    expect(upgraded).toMatchObject({
      id: nextId,
      name: "Infinity Desktop",
      kind: "workspace",
    });
    expect(state.windows.find((win) => win.module === "terminal")).toMatchObject({
      x: -500,
      y: 220,
    });
    expect(state.desktopIcons.find((icon) => icon.module === "apps")).toBeTruthy();
  });

  it("keeps intentionally empty desktops empty when settings reload", () => {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({
        version: "1.2.0",
        activeDesktopId: "client-empty",
        desktopSpaces: [
          {
            id: "client-empty",
            name: "Client Empty",
            kind: "personal",
            desktopIcons: [],
            dockPinnedItems: ["finder"],
            hiddenModules: ["workspace-app-*"],
            folders: [],
            windows: [],
            windowOrder: [],
            focusedWindowId: null,
            createdAt: new Date(0).toISOString(),
            updatedAt: new Date(0).toISOString(),
          },
        ],
      }),
    );

    const saved = loadSavedSettings(["finder"]);

    expect(saved.activeDesktopId).toBe("client-empty");
    expect(saved.desktopIcons).toEqual([]);
    expect(saved.desktopSpaces?.[0]?.desktopIcons).toEqual([]);
    expect(saved.dockPinnedItems).toEqual(["finder"]);
    expect(saved.hiddenModules).toEqual(["workspace-app-*"]);
  });

  it("does not repopulate an intentionally empty desktop from workspace app discovery", () => {
    windowStore.createDesktopSpace("Client Work");

    windowStore.syncWorkspaceApps([
      {
        id: "calculator",
        name: "Calculator",
        url: "https://example.com/calculator",
        launch_mode: "iframe",
        show_on_desktop: true,
        show_in_dock: true,
      },
    ]);

    const state = get(windowStore);
    expect(state.desktopIcons).toEqual([]);
    expect(state.dockPinnedItems).toEqual(["finder"]);
  });

  it("places an explicitly added workspace app on an empty desktop and dock", () => {
    windowStore.createDesktopSpace("Client Work");

    windowStore.placeWorkspaceApp({
      id: "claude",
      name: "Claude",
      url: "https://claude.ai",
      launch_mode: "browser",
      logo_url: "/app-logos/claude.svg",
      color: "#111827",
      show_on_desktop: true,
      show_in_dock: true,
    });

    const state = get(windowStore);
    const icon = state.desktopIcons.find((item) => item.module === "workspace-app-claude");
    expect(icon).toMatchObject({
      label: "Claude",
      appUrl: "https://claude.ai",
      launchMode: "browser",
    });
    expect(state.dockPinnedItems).toContain("workspace-app-claude");
    expect(state.hiddenModules).not.toContain("workspace-app-*");
  });

  it("removes workspace app icon, dock item, and open window when app disappears from sync", () => {
    windowStore.placeWorkspaceApp({
      id: "claude",
      name: "Claude",
      url: "https://claude.ai",
      launch_mode: "iframe",
      show_on_desktop: true,
      show_in_dock: true,
    });
    windowStore.openWindow("workspace-app-claude", {
      title: "Claude",
      data: { url: "https://claude.ai", launchMode: "iframe" },
    });

    windowStore.syncWorkspaceApps([]);

    const state = get(windowStore);
    expect(state.desktopIcons.some((icon) => icon.module === "workspace-app-claude")).toBe(false);
    expect(state.dockPinnedItems).not.toContain("workspace-app-claude");
    expect(state.windows.some((win) => win.module === "workspace-app-claude")).toBe(false);
  });

  it("keeps removed workspace apps removed across sync and reload", () => {
    windowStore.syncWorkspaceApps([
      {
        id: "calculator",
        name: "Calculator",
        url: "https://example.com/calculator",
        launch_mode: "iframe",
        show_on_desktop: true,
        show_in_dock: true,
      },
    ]);
    windowStore.removeDesktopIconByModule("workspace-app-calculator");

    windowStore.syncWorkspaceApps([
      {
        id: "calculator",
        name: "Calculator",
        url: "https://example.com/calculator",
        launch_mode: "iframe",
        show_on_desktop: true,
        show_in_dock: true,
      },
    ]);

    const state = get(windowStore);
    expect(state.desktopIcons.some((icon) => icon.module === "workspace-app-calculator")).toBe(false);
    expect(state.dockPinnedItems).not.toContain("workspace-app-calculator");
    expect(state.hiddenModules).toContain("workspace-app-calculator");

    const saved = loadSavedSettings(["finder"]);
    expect(saved.desktopIcons?.some((icon) => icon.module === "workspace-app-calculator")).toBe(false);
    expect(saved.dockPinnedItems).not.toContain("workspace-app-calculator");
    expect(saved.hiddenModules).toContain("workspace-app-calculator");
  });

  it("persists pixel icon positions used by resize clamping", () => {
    windowStore.updateIconPosition("icon-knowledge", 320, 180);

    const state = get(windowStore);
    const icon = state.desktopIcons.find((item) => item.id === "icon-knowledge");
    expect(icon).toMatchObject({ x: 320, y: 180, positionMode: "pixel" });

    const saved = loadSavedSettings(["finder"]);
    const savedIcon = saved.desktopIcons?.find((item) => item.id === "icon-knowledge");
    expect(savedIcon).toMatchObject({ x: 320, y: 180, positionMode: "pixel" });
  });

  it("duplicates the current desktop layout and appearance", () => {
    desktopSettings.setBackground("dark-mode");
    windowStore.addDesktopIcon("knowledge", "Knowledge");
    windowStore.openWindow("knowledge");

    const id = "22222222-2222-4222-8222-222222222222";
    windowStore.duplicateDesktopSpace("Knowledge Copy", { id, kind: "team" });
    const state = get(windowStore);

    expect(state.activeDesktopId).toBe(id);
    expect(state.desktopSpaces.find((space) => space.id === id)?.kind).toBe("team");
    expect(state.desktopIcons.map((icon) => icon.module)).toContain("knowledge");
    expect(state.windows.map((window) => window.module)).toContain("knowledge");
    expect(get(desktopSettings).backgroundId).toBe("dark-mode");
    expect(
      state.desktopSpaces.find((space) => space.id === state.activeDesktopId)?.desktopSettings,
    ).toMatchObject({ backgroundId: "dark-mode" });
  });

  it("renames the current desktop and persists the new name", () => {
    windowStore.createDesktopSpace("Client Work");
    const desktopId = get(windowStore).activeDesktopId;

    windowStore.renameDesktopSpace(desktopId, "Client Delivery");

    const state = get(windowStore);
    expect(state.desktopSpaces.find((space) => space.id === desktopId)?.name).toBe("Client Delivery");
    expect(loadSavedSettings(["finder"]).desktopSpaces?.find((space) => space.id === desktopId)?.name).toBe(
      "Client Delivery",
    );
  });

  it("deletes the current desktop and switches to a remaining desktop", () => {
    const personalId = get(windowStore).activeDesktopId;
    windowStore.createDesktopSpace("Client Work");
    const clientId = get(windowStore).activeDesktopId;
    windowStore.openWindow("knowledge");

    windowStore.deleteDesktopSpace(clientId);

    const state = get(windowStore);
    expect(state.activeDesktopId).toBe(personalId);
    expect(state.desktopSpaces.map((space) => space.id)).toEqual([personalId]);
    expect(state.windows).toEqual([]);
    expect(loadSavedSettings(["finder"]).desktopSpaces?.map((space) => space.id)).toEqual([personalId]);
  });

  it("keeps separate backgrounds for each desktop when switching", () => {
    const personalId = get(windowStore).activeDesktopId;
    desktopSettings.setBackground("classic-gray");
    windowStore.updateActiveDesktopSettings(get(desktopSettings));

    windowStore.createDesktopSpace("Studio", { kind: "personal" });
    const studioId = get(windowStore).activeDesktopId;
    desktopSettings.setBackground("dark-mode");
    windowStore.updateActiveDesktopSettings(get(desktopSettings));

    windowStore.switchDesktopSpace(personalId);
    expect(get(desktopSettings).backgroundId).toBe("classic-gray");

    windowStore.switchDesktopSpace(studioId);
    expect(get(desktopSettings).backgroundId).toBe("dark-mode");
  });

  it("applies a remote workspace desktop update to the active desktop", () => {
    const id = "33333333-3333-4333-8333-333333333333";
    windowStore.createDesktopSpace("Shared Room", { id, kind: "workspace" });

    windowStore.applyRemoteDesktopSpace({
      id,
      name: "Shared Room",
      kind: "workspace",
      desktopIcons: [],
      dockPinnedItems: ["finder"],
      hiddenModules: [],
      folders: [],
      windows: [
        {
          id: "remote-note",
          module: "canvas-note-remote",
          title: "Note",
          x: 420,
          y: 260,
          width: 320,
          height: 240,
          minWidth: 220,
          minHeight: 160,
          minimized: false,
          maximized: false,
          data: { text: "Shared update" },
        },
      ],
      windowOrder: ["remote-note"],
      focusedWindowId: "remote-note",
      createdAt: new Date(0).toISOString(),
      updatedAt: new Date().toISOString(),
    });

    const state = get(windowStore);
    expect(state.windows).toHaveLength(1);
    expect(state.windows[0]).toMatchObject({
      id: "remote-note",
      data: { text: "Shared update" },
    });
  });

  it("persists window data updates for canvas notes", () => {
    windowStore.openWindow("canvas-note-test", {
      title: "Note",
      data: { text: "" },
    });
    const note = get(windowStore).windows.find((w) => w.module === "canvas-note-test");
    expect(note).toBeTruthy();

    windowStore.updateWindowData(note!.id, { text: "Remember this" });

    const state = get(windowStore);
    expect(state.windows.find((w) => w.id === note!.id)?.data).toMatchObject({
      text: "Remember this",
    });
    expect(loadSavedSettings(["finder"]).windows?.find((w) => w.id === note!.id)?.data).toMatchObject({
      text: "Remember this",
    });
  });

  it("uses compact sticky-note defaults for unique canvas note windows", () => {
    windowStore.openWindow("canvas-note-123", {
      title: "Note",
      data: { text: "" },
    });

    const note = get(windowStore).windows.find((w) => w.module === "canvas-note-123");
    expect(note).toMatchObject({
      title: "Note",
      width: 320,
      height: 240,
      minWidth: 220,
      minHeight: 160,
    });
  });

  it("keeps module visibility independent per desktop", () => {
    const personalId = get(windowStore).activeDesktopId;
    windowStore.createDesktopSpace("Build Room", { kind: "workspace" });
    const buildId = get(windowStore).activeDesktopId;

    windowStore.removeDesktopIconByModule("knowledge");
    expect(get(windowStore).desktopIcons.some((icon) => icon.module === "knowledge")).toBe(false);
    expect(get(windowStore).hiddenModules).toContain("knowledge");

    windowStore.switchDesktopSpace(personalId);
    expect(get(windowStore).desktopIcons.some((icon) => icon.module === "knowledge")).toBe(true);
    expect(get(windowStore).hiddenModules).not.toContain("knowledge");

    windowStore.switchDesktopSpace(buildId);
    expect(get(windowStore).desktopIcons.some((icon) => icon.module === "knowledge")).toBe(false);
    expect(get(windowStore).hiddenModules).toContain("knowledge");
  });

  it("can explicitly add a workspace app onto an otherwise empty desktop", () => {
    windowStore.createDesktopSpace("Client Work", { kind: "workspace" });
    windowStore.syncWorkspaceApps([
      {
        id: "perplexity",
        name: "Perplexity",
        url: "https://perplexity.ai",
        launch_mode: "browser",
        show_on_desktop: true,
        show_in_dock: true,
        logo_url: "https://example.com/perplexity.png",
      },
    ]);

    expect(get(windowStore).desktopIcons).toEqual([]);
    expect(get(windowStore).dockPinnedItems).toEqual(["finder"]);

    windowStore.addDesktopIcon("workspace-app-perplexity", "Perplexity");
    windowStore.addToDock("workspace-app-perplexity");

    const state = get(windowStore);
    expect(state.desktopIcons.find((icon) => icon.module === "workspace-app-perplexity")).toMatchObject({
      label: "Perplexity",
      appUrl: "https://perplexity.ai",
      launchMode: "browser",
    });
    expect(state.dockPinnedItems).toContain("workspace-app-perplexity");
    expect(state.hiddenModules).not.toContain("workspace-app-perplexity");
    expect(state.hiddenModules).not.toContain("workspace-app-*");
  });

  it("clamps windows and pixel icons into the visible desktop after resize", () => {
    windowStore.openWindow("knowledge");
    const win = get(windowStore).windows.find((window) => window.module === "knowledge");
    expect(win).toBeTruthy();

    windowStore.updateWindowBounds(win!.id, 1900, 1200, 1000, 700);
    windowStore.updateIconPosition("icon-knowledge", 1800, 1200);

    windowStore.clampActiveDesktopToViewport(900, 620);

    const state = get(windowStore);
    const clampedWindow = state.windows.find((window) => window.id === win!.id);
    const clampedIcon = state.desktopIcons.find((icon) => icon.id === "icon-knowledge");
    expect(clampedWindow).toMatchObject({
      x: 0,
      y: 0,
      width: 900,
      height: 620,
    });
    expect(clampedIcon).toMatchObject({
      x: 810,
      y: 516,
      positionMode: "pixel",
    });
    expect(loadSavedSettings(["finder"]).windows?.find((window) => window.id === win!.id)).toMatchObject({
      x: 0,
      y: 0,
      width: 900,
      height: 620,
    });
  });

  it("moves colliding pixel icons to distinct visible slots after resize", () => {
    windowStore.createDesktopSpace("Resize Test");
    for (const module of ["dashboard", "knowledge", "agents"]) {
      windowStore.addDesktopIcon(module, module);
    }
    for (const icon of get(windowStore).desktopIcons) {
      windowStore.updateIconPosition(icon.id, 1200, 900);
    }

    windowStore.clampActiveDesktopToViewport(520, 420, {
      iconWidth: 90,
      iconHeight: 104,
    });

    const icons = get(windowStore).desktopIcons;
    const positions = icons.map((icon) => `${icon.x}:${icon.y}`);
    expect(new Set(positions).size).toBe(icons.length);
    expect(icons.every((icon) => icon.x >= 0 && icon.x <= 430)).toBe(true);
    expect(icons.every((icon) => icon.y >= 0 && icon.y <= 316)).toBe(true);
    for (const [index, icon] of icons.entries()) {
      for (const other of icons.slice(index + 1)) {
        expect(Math.abs(icon.x - other.x) >= 98 || Math.abs(icon.y - other.y) >= 112).toBe(true);
      }
    }
  });

  it("preserves a manually placed icon's relative area through resize", () => {
    windowStore.updateIconPosition("icon-knowledge", 320, 180, {
      width: 1000,
      height: 700,
      iconWidth: 90,
      iconHeight: 104,
    });

    windowStore.clampActiveDesktopToViewport(500, 350, {
      iconWidth: 90,
      iconHeight: 104,
    });
    const compact = get(windowStore).desktopIcons.find((icon) => icon.id === "icon-knowledge");
    expect(compact?.x).toBeCloseTo((320 / 910) * 410, 4);
    expect(compact?.y).toBeCloseTo((180 / 596) * 246, 4);

    windowStore.clampActiveDesktopToViewport(1000, 700, {
      iconWidth: 90,
      iconHeight: 104,
    });
    const restored = get(windowStore).desktopIcons.find((icon) => icon.id === "icon-knowledge");
    expect(restored?.x).toBeCloseTo(320, 4);
    expect(restored?.y).toBeCloseTo(180, 4);
    expect(loadSavedSettings(["finder"]).desktopIcons?.find((icon) => icon.id === "icon-knowledge"))
      .toMatchObject({ relativeX: 320 / 910, relativeY: 180 / 596 });
  });

  it("arranges every visible icon into unique rows and columns inside the viewport", () => {
    windowStore.createDesktopSpace("Arrange Test");
    for (const module of ["dashboard", "knowledge", "agents", "projects", "tasks", "inbox"]) {
      windowStore.addDesktopIcon(module, module);
    }
    for (const icon of get(windowStore).desktopIcons) {
      windowStore.updateIconPosition(icon.id, 880, 100);
    }

    windowStore.arrangeActiveDesktopIcons(520, 420, {
      iconWidth: 90,
      iconHeight: 104,
      gap: 12,
      padding: 16,
    });

    const visibleIcons = get(windowStore).desktopIcons.filter(
      (icon) => !icon.folderId || icon.type === "folder",
    );
    const positions = visibleIcons.map((icon) => `${icon.x}:${icon.y}`);

    expect(new Set(positions).size).toBe(visibleIcons.length);
    expect(visibleIcons.every((icon) => icon.positionMode === "pixel")).toBe(true);
    expect(visibleIcons.every((icon) => icon.x >= 16 && icon.x <= 414)).toBe(true);
    expect(visibleIcons.every((icon) => icon.y >= 16 && icon.y <= 300)).toBe(true);
  });

  it("allows canvas desktops to keep offscreen positions while normal desktops are clamped", () => {
    windowStore.createDesktopSpace("Infinity Desktop", { kind: "workspace" });
    windowStore.openWindow("knowledge");
    const win = get(windowStore).windows.find((window) => window.module === "knowledge");
    windowStore.updateWindowPosition(win!.id, -4200, -1400);
    windowStore.addDesktopIcon("apps", "Apps");
    windowStore.updateIconPosition("icon-apps", -2400, -900);

    windowStore.clampActiveDesktopToViewport(900, 620, { canvas: true });

    const state = get(windowStore);
    expect(state.windows.find((window) => window.id === win!.id)).toMatchObject({ x: -4200, y: -1400 });
    expect(state.desktopIcons.find((icon) => icon.id === "icon-apps")).toMatchObject({
      x: -2400,
      y: -900,
    });
  });

  it("sanitizes restored desktop state before windows render", () => {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({
        version: "1.2.0",
        activeDesktopId: "personal",
        desktopIcons: [
          {
            id: "icon-knowledge",
            module: "knowledge",
            label: "Wrong label",
            x: Number.POSITIVE_INFINITY,
            y: 900000,
          },
        ],
        dockPinnedItems: ["finder"],
        hiddenModules: [],
        windows: [
          {
            id: "broken",
            module: "knowledge",
            title: "Broken",
            x: Number.POSITIVE_INFINITY,
            y: 0,
            width: 500,
            height: 500,
          },
          {
            id: "huge",
            module: "terminal",
            title: "Terminal",
            x: 900000,
            y: -900000,
            width: 100000,
            height: 100000,
            minWidth: 10,
            minHeight: 10,
            previousBounds: {
              x: -900000,
              y: 900000,
              width: 100000,
              height: 100000,
            },
          },
        ],
      }),
    );

    const saved = loadSavedSettings(["finder"]);

    expect(saved.desktopIcons?.find((icon) => icon.id === "icon-knowledge")).toMatchObject({
      label: "Knowledge",
      x: 0,
      y: 50000,
    });
    expect(saved.windows).toHaveLength(1);
    expect(saved.windows?.[0]).toMatchObject({
      id: "huge",
      x: 50000,
      y: -50000,
      width: 2200,
      height: 1600,
      minWidth: 240,
      minHeight: 160,
      previousBounds: {
        x: -50000,
        y: 50000,
        width: 2200,
        height: 1600,
      },
    });
  });
});
