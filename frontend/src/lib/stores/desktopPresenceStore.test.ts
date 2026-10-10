import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

class FakeBroadcastChannel {
  static instances: FakeBroadcastChannel[] = [];
  listeners: Array<(event: MessageEvent) => void> = [];
  messages: unknown[] = [];

  constructor(public name: string) {
    FakeBroadcastChannel.instances.push(this);
  }

  addEventListener(_type: string, listener: (event: MessageEvent) => void) {
    this.listeners.push(listener);
  }

  postMessage(message: unknown) {
    this.messages.push(message);
  }

  emit(data: unknown) {
    for (const listener of this.listeners) {
      listener({ data } as MessageEvent);
    }
  }

  close() {}
}

class FakeWebSocket {
  static instances: FakeWebSocket[] = [];
  static readonly OPEN = 1;
  readyState = FakeWebSocket.OPEN;
  onopen: (() => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onclose: (() => void) | null = null;
  sent: string[] = [];

  constructor(public url: string) {
    FakeWebSocket.instances.push(this);
  }

  send(message: string) {
    this.sent.push(message);
  }

  close() {
    this.readyState = 3;
  }

  open() {
    this.onopen?.();
  }

  emit(data: unknown) {
    this.onmessage?.({ data: JSON.stringify(data) } as MessageEvent);
  }
}

async function loadStore() {
  vi.resetModules();
  vi.stubGlobal("BroadcastChannel", FakeBroadcastChannel);
  vi.stubGlobal("WebSocket", FakeWebSocket);
  return import("./desktopPresenceStore");
}

describe("desktopPresenceStore", () => {
  beforeEach(() => {
    FakeBroadcastChannel.instances = [];
    FakeWebSocket.instances = [];
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-07-04T12:00:00.000Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("does not show the current user as a remote cursor", async () => {
    const mod = await loadStore();
    const seen: unknown[][] = [];
    const unsubscribe = mod.desktopPresenceStore.subscribe((value) => seen.push(value));

    mod.startDesktopPresence();
    mod.setDesktopPresenceContext("desktop-a", { userId: "user-a", name: "Alice" });
    FakeBroadcastChannel.instances[0].emit({
      type: "cursor",
      payload: {
        clientId: "client-b",
        userId: "user-a",
        name: "Alice",
        color: "#0ea5e9",
        workspaceId: null,
        desktopId: "desktop-a",
        x: 10,
        y: 20,
        lastSeen: Date.now(),
      },
    });

    expect(seen.at(-1)).toEqual([]);
    unsubscribe();
    mod.stopDesktopPresence();
  });

  it("isolates cursors by workspace and desktop", async () => {
    const mod = await loadStore();
    let current: unknown[] = [];
    const unsubscribe = mod.desktopPresenceStore.subscribe((value) => {
      current = value;
    });

    mod.startDesktopPresence();
    mod.setDesktopPresenceWorkspace("11111111-1111-4111-8111-111111111111");
    mod.setDesktopPresenceContext("desktop-a", { userId: "user-a", name: "Alice" });
    FakeBroadcastChannel.instances[0].emit({
      type: "cursor",
      payload: {
        clientId: "client-b",
        userId: "user-b",
        name: "Bob",
        color: "#22c55e",
        workspaceId: "22222222-2222-4222-8222-222222222222",
        desktopId: "desktop-a",
        x: 10,
        y: 20,
        lastSeen: Date.now(),
      },
    });
    expect(current).toEqual([]);

    FakeBroadcastChannel.instances[0].emit({
      type: "cursor",
      payload: {
        clientId: "client-c",
        userId: "user-c",
        name: "Chris",
        color: "#f97316",
        workspaceId: "11111111-1111-4111-8111-111111111111",
        desktopId: "desktop-b",
        x: 30,
        y: 40,
        lastSeen: Date.now(),
      },
    });
    expect(current).toEqual([]);

    FakeBroadcastChannel.instances[0].emit({
      type: "cursor",
      payload: {
        clientId: "client-d",
        userId: "user-d",
        name: "Drew",
        color: "#a855f7",
        workspaceId: "11111111-1111-4111-8111-111111111111",
        desktopId: "desktop-a",
        x: 50,
        y: 60,
        lastSeen: Date.now(),
      },
    });
    expect(current).toHaveLength(1);
    expect(current[0] as Record<string, unknown>).toMatchObject({ clientId: "client-d", name: "Drew" });

    unsubscribe();
    mod.stopDesktopPresence();
  });

  it("carries active tool metadata with cursor presence", async () => {
    const mod = await loadStore();
    let current: unknown[] = [];
    const unsubscribe = mod.desktopPresenceStore.subscribe((value) => {
      current = value;
    });

    mod.startDesktopPresence();
    mod.setDesktopPresenceWorkspace("11111111-1111-4111-8111-111111111111");
    mod.setDesktopPresenceContext("desktop-a", {
      userId: "user-a",
      name: "Alice",
    });
    FakeBroadcastChannel.instances[0].emit({
      type: "cursor",
      payload: {
        clientId: "client-e",
        userId: "user-e",
        name: "Elliot",
        color: "#14b8a6",
        activeModule: "workspace-app-claude",
        activeTitle: "Claude",
        workspaceId: "11111111-1111-4111-8111-111111111111",
        desktopId: "desktop-a",
        x: 50,
        y: 60,
        viewportWidth: 1440,
        viewportHeight: 900,
        lastSeen: Date.now(),
      },
    });

    expect(current).toHaveLength(1);
    expect(current[0] as Record<string, unknown>).toMatchObject({
      clientId: "client-e",
      activeModule: "workspace-app-claude",
      activeTitle: "Claude",
    });

    unsubscribe();
    mod.stopDesktopPresence();
  });

  it("connects remote presence for persisted workspace desktop ids", async () => {
    const mod = await loadStore();
    const workspaceId = "11111111-1111-4111-8111-111111111111";
    const desktopId = "22222222-2222-4222-8222-222222222222";

    mod.startDesktopPresence();
    mod.connectDesktopPresenceRemote(workspaceId, desktopId);

    expect(FakeWebSocket.instances).toHaveLength(1);
    expect(FakeWebSocket.instances[0].url).toContain(
      `/api/v1/workspaces/${workspaceId}/desktop-spaces/${desktopId}/presence/ws`,
    );

    mod.stopDesktopPresence();
  });

  it("does not connect remote presence for local-only desktop ids", async () => {
    const mod = await loadStore();

    mod.startDesktopPresence();
    mod.connectDesktopPresenceRemote("11111111-1111-4111-8111-111111111111", "personal");

    expect(FakeWebSocket.instances).toHaveLength(0);

    mod.stopDesktopPresence();
  });

  it("publishes active window labels and negative infinity canvas coordinates", async () => {
    const mod = await loadStore();
    const workspaceId = "11111111-1111-4111-8111-111111111111";
    const desktopId = "22222222-2222-4222-8222-222222222222";

    mod.startDesktopPresence();
    mod.setDesktopPresenceContext(desktopId, {
      userId: "user-a",
      name: "Alice",
      activeModule: "apps",
      activeTitle: "Apps",
    });
    mod.connectDesktopPresenceRemote(workspaceId, desktopId);
    FakeWebSocket.instances[0].open();
    FakeWebSocket.instances[0].sent = [];

    mod.publishDesktopCursor(desktopId, -120, -80, 20000, 20000);

    expect(JSON.parse(FakeWebSocket.instances[0].sent[0])).toMatchObject({
      type: "cursor",
      x: -120,
      y: -80,
      viewport_width: 20000,
      viewport_height: 20000,
      active_module: "apps",
      active_title: "Apps",
    });
    expect(FakeBroadcastChannel.instances[0].messages.at(-1)).toMatchObject({
      type: "cursor",
      payload: {
        desktopId,
        workspaceId,
        x: -120,
        y: -80,
        viewportWidth: 20000,
        viewportHeight: 20000,
        activeModule: "apps",
        activeTitle: "Apps",
      },
    });

    mod.stopDesktopPresence();
  });

  it("does not publish a cursor for a stale desktop context", async () => {
    const mod = await loadStore();
    const workspaceId = "11111111-1111-4111-8111-111111111111";
    const desktopId = "22222222-2222-4222-8222-222222222222";

    mod.startDesktopPresence();
    mod.connectDesktopPresenceRemote(workspaceId, desktopId);
    FakeWebSocket.instances[0].open();
    FakeWebSocket.instances[0].sent = [];
    FakeBroadcastChannel.instances[0].messages = [];

    mod.publishDesktopCursor("33333333-3333-4333-8333-333333333333", 10, 20, 1440, 900);

    expect(FakeWebSocket.instances[0].sent).toEqual([]);
    expect(FakeBroadcastChannel.instances[0].messages).toEqual([]);

    mod.stopDesktopPresence();
  });

  it("notifies listeners when the backend broadcasts a desktop-space revision", async () => {
    const mod = await loadStore();
    const workspaceId = "11111111-1111-4111-8111-111111111111";
    const desktopId = "22222222-2222-4222-8222-222222222222";
    const updates: unknown[] = [];

    mod.startDesktopPresence();
    mod.connectDesktopPresenceRemote(workspaceId, desktopId);
    const unsubscribe = mod.onDesktopSpaceUpdated((event) => updates.push(event));

    FakeWebSocket.instances[0].emit({
      type: "desktop_space_updated",
      workspace_id: workspaceId,
      desktop_space_id: desktopId,
      revision: "2026-07-04T12:30:00Z",
      last_seen: Date.now(),
    });

    expect(updates).toEqual([
      {
        workspaceId,
        desktopId,
        revision: "2026-07-04T12:30:00Z",
        action: "updated",
        lastSeen: Date.now(),
      },
    ]);

    unsubscribe();
    mod.stopDesktopPresence();
  });

  it("broadcasts local desktop-space revisions across same-browser windows", async () => {
    const mod = await loadStore();
    const workspaceId = "11111111-1111-4111-8111-111111111111";
    const desktopId = "22222222-2222-4222-8222-222222222222";
    const updates: unknown[] = [];

    mod.startDesktopPresence();
    mod.connectDesktopPresenceRemote(workspaceId, desktopId);
    const unsubscribe = mod.onDesktopSpaceUpdated((event) => updates.push(event));

    mod.publishDesktopSpaceUpdated(desktopId, "revision-a");

    expect(FakeBroadcastChannel.instances[0].messages.at(-1)).toMatchObject({
      type: "desktop_space_updated",
      payload: {
        workspaceId,
        desktopId,
        revision: "revision-a",
        action: "updated",
      },
    });

    FakeBroadcastChannel.instances[0].emit(FakeBroadcastChannel.instances[0].messages.at(-1));
    expect(updates).toHaveLength(1);
    expect(updates[0]).toMatchObject({ workspaceId, desktopId, revision: "revision-a" });

    unsubscribe();
    mod.stopDesktopPresence();
  });

  it("notifies listeners when a shared desktop is deleted", async () => {
    const mod = await loadStore();
    const workspaceId = "11111111-1111-4111-8111-111111111111";
    const desktopId = "22222222-2222-4222-8222-222222222222";
    const updates: unknown[] = [];

    mod.startDesktopPresence();
    mod.connectDesktopPresenceRemote(workspaceId, desktopId);
    const unsubscribe = mod.onDesktopSpaceUpdated((event) => updates.push(event));

    FakeWebSocket.instances[0].emit({
      type: "desktop_space_updated",
      workspace_id: workspaceId,
      desktop_space_id: desktopId,
      revision: "delete-a",
      action: "deleted",
      last_seen: Date.now(),
    });

    expect(updates).toHaveLength(1);
    expect(updates[0]).toMatchObject({ workspaceId, desktopId, revision: "delete-a", action: "deleted" });

    unsubscribe();
    mod.stopDesktopPresence();
  });

  it("drops malformed remote cursor packets before they reach the canvas", async () => {
    const mod = await loadStore();
    const workspaceId = "11111111-1111-4111-8111-111111111111";
    const desktopId = "22222222-2222-4222-8222-222222222222";
    let current: unknown[] = [];
    const unsubscribe = mod.desktopPresenceStore.subscribe((value) => {
      current = value;
    });

    mod.startDesktopPresence();
    mod.connectDesktopPresenceRemote(workspaceId, desktopId);

    FakeWebSocket.instances[0].emit({
      type: "cursor",
      client_id: "client-bad",
      user_id: "user-bad",
      name: "Bad Cursor",
      workspace_id: workspaceId,
      desktop_space_id: desktopId,
      x: 200001,
      y: 50,
      viewport_width: 1440,
      viewport_height: 900,
      last_seen: Date.now(),
    });

    expect(current).toEqual([]);

    FakeWebSocket.instances[0].emit({
      type: "cursor",
      client_id: "client-good",
      user_id: "user-good",
      name: "Good Cursor",
      workspace_id: workspaceId,
      desktop_space_id: desktopId,
      x: -100000,
      y: 100000,
      viewport_width: 1440,
      viewport_height: 900,
      last_seen: Date.now(),
    });

    expect(current).toHaveLength(1);
    expect(current[0]).toMatchObject({ clientId: "client-good", x: -100000, y: 100000 });

    unsubscribe();
    mod.stopDesktopPresence();
  });

  it("cleans desktop-space update listeners when presence stops", async () => {
    const mod = await loadStore();
    const workspaceId = "11111111-1111-4111-8111-111111111111";
    const desktopId = "22222222-2222-4222-8222-222222222222";
    const updates: unknown[] = [];

    mod.startDesktopPresence();
    mod.connectDesktopPresenceRemote(workspaceId, desktopId);
    mod.onDesktopSpaceUpdated((event) => updates.push(event));
    mod.stopDesktopPresence();

    mod.startDesktopPresence();
    mod.connectDesktopPresenceRemote(workspaceId, desktopId);
    FakeBroadcastChannel.instances.at(-1)?.emit({
      type: "desktop_space_updated",
      payload: {
        workspaceId,
        desktopId,
        revision: "after-stop",
        action: "updated",
        lastSeen: Date.now(),
      },
    });

    expect(updates).toEqual([]);
    mod.stopDesktopPresence();
  });

  it("allows the first cursor publish immediately after switching desktops", async () => {
    const mod = await loadStore();
    const workspaceId = "11111111-1111-4111-8111-111111111111";
    const desktopA = "22222222-2222-4222-8222-222222222222";
    const desktopB = "33333333-3333-4333-8333-333333333333";

    mod.startDesktopPresence();
    mod.connectDesktopPresenceRemote(workspaceId, desktopA);
    FakeWebSocket.instances[0].open();
    FakeWebSocket.instances[0].sent = [];
    mod.publishDesktopCursor(desktopA, 10, 20, 1440, 900);
    expect(FakeWebSocket.instances[0].sent).toHaveLength(1);

    mod.connectDesktopPresenceRemote(workspaceId, desktopB);
    FakeWebSocket.instances.at(-1)?.open();
    FakeWebSocket.instances.at(-1)!.sent = [];
    FakeBroadcastChannel.instances[0].messages = [];
    mod.publishDesktopCursor(desktopB, 30, 40, 1440, 900);

    expect(FakeWebSocket.instances.at(-1)?.sent).toHaveLength(1);
    expect(FakeBroadcastChannel.instances[0].messages.at(-1)).toMatchObject({
      type: "cursor",
      payload: { desktopId: desktopB, x: 30, y: 40 },
    });

    mod.stopDesktopPresence();
  });

  it("tracks a followed teammate and clears follow state when they leave", async () => {
    const mod = await loadStore();
    const workspaceId = "11111111-1111-4111-8111-111111111111";
    const desktopId = "22222222-2222-4222-8222-222222222222";
    const followed: Array<string | null> = [];
    const unsubscribe = mod.followedDesktopCursor.subscribe((value) => followed.push(value));

    mod.startDesktopPresence();
    mod.connectDesktopPresenceRemote(workspaceId, desktopId);
    mod.followDesktopCursor("client-follow");

    expect(followed.at(-1)).toBe("client-follow");

    FakeWebSocket.instances[0].emit({
      type: "leave",
      client_id: "client-follow",
      user_id: "user-follow",
      workspace_id: workspaceId,
      desktop_space_id: desktopId,
      last_seen: Date.now(),
    });

    expect(followed.at(-1)).toBeNull();

    unsubscribe();
    mod.stopDesktopPresence();
  });
});
