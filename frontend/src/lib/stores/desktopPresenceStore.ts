import { writable } from "svelte/store";
import { getBackendUrl } from "$lib/config/runtime";

export interface DesktopPresenceCursor {
  clientId: string;
  userId: string;
  name: string;
  color: string;
  activeModule?: string;
  activeTitle?: string;
  workspaceId: string | null;
  desktopId: string;
  x: number;
  y: number;
  viewportWidth: number;
  viewportHeight: number;
  lastSeen: number;
}

export interface DesktopSpaceUpdateEvent {
  workspaceId: string | null;
  desktopId: string;
  revision?: string;
  action: "updated" | "deleted";
  lastSeen: number;
}

interface PresenceIdentity {
  userId: string;
  name: string;
  color?: string;
  activeModule?: string;
  activeTitle?: string;
}

type PresenceMessage =
  | {
      type: "cursor";
      payload: DesktopPresenceCursor;
    }
  | {
      type: "desktop_space_updated";
      payload: DesktopSpaceUpdateEvent;
    }
  | {
      type: "leave";
      clientId: string;
      workspaceId?: string | null;
      desktopId: string;
    };

type RemotePresenceEvent = {
  type: "cursor" | "join" | "leave" | "heartbeat" | "error" | "desktop_space_updated";
  client_id?: string;
  user_id?: string;
  name?: string;
  color?: string;
  active_module?: string;
  active_title?: string;
  workspace_id?: string;
  desktop_space_id?: string;
  x?: number;
  y?: number;
  viewport_width?: number;
  viewport_height?: number;
  last_seen?: number;
  revision?: string;
  action?: "updated" | "deleted";
};

const CHANNEL_NAME = "businessos:desktop-presence";
const CURSOR_TTL_MS = 8000;
const CURSOR_THROTTLE_MS = 50;
const UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
const MIN_CURSOR_COORDINATE = -100000;
const MAX_CURSOR_COORDINATE = 100000;
const clientId =
  typeof crypto !== "undefined" && "randomUUID" in crypto
    ? crypto.randomUUID()
    : `client-${Date.now()}-${Math.random().toString(16).slice(2)}`;

const cursors = new Map<string, DesktopPresenceCursor>();
const { subscribe, set } = writable<DesktopPresenceCursor[]>([]);
const { subscribe: subscribeFollowedCursor, set: setFollowedCursor } = writable<string | null>(null);

let channel: BroadcastChannel | null = null;
let cleanupTimer: ReturnType<typeof setInterval> | null = null;
let identity: PresenceIdentity = {
  userId: "local-user",
  name: "Teammate",
};
let activeDesktopId = "personal";
let activeWorkspaceId: string | null = null;
let lastPublishAt = 0;
let socket: WebSocket | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let reconnectAttempts = 0;
let followedCursorClientId: string | null = null;
const desktopSpaceUpdateListeners = new Set<(event: DesktopSpaceUpdateEvent) => void>();

function colorFor(value: string) {
  const palette = ["#0ea5e9", "#22c55e", "#f97316", "#a855f7", "#ef4444", "#14b8a6"];
  let hash = 0;
  for (let i = 0; i < value.length; i += 1) {
    hash = (hash * 31 + value.charCodeAt(i)) >>> 0;
  }
  return palette[hash % palette.length];
}

function publish() {
  const now = Date.now();
  const visible = Array.from(cursors.values()).filter(
    (cursor) =>
      cursor.workspaceId === activeWorkspaceId &&
      cursor.desktopId === activeDesktopId &&
      cursor.clientId !== clientId &&
      cursor.userId !== identity.userId &&
      now - cursor.lastSeen <= CURSOR_TTL_MS,
  );
  set(visible);
}

function setFollowedCursorId(cursorId: string | null) {
  followedCursorClientId = cursorId;
  setFollowedCursor(cursorId);
}

function clearFollowedCursorIfClient(remoteClientId: string) {
  if (followedCursorClientId === remoteClientId) {
    setFollowedCursorId(null);
  }
}

function post(message: PresenceMessage) {
  channel?.postMessage(message);
}

function notifyDesktopSpaceUpdated(event: DesktopSpaceUpdateEvent) {
  if (event.workspaceId !== activeWorkspaceId || event.desktopId !== activeDesktopId) return;
  for (const listener of desktopSpaceUpdateListeners) {
    listener(event);
  }
}

function workspaceCanUseRemote(workspaceId: string | null) {
  return Boolean(workspaceId && UUID_PATTERN.test(workspaceId));
}

function desktopCanUseRemote(desktopId: string | null) {
  return Boolean(desktopId && UUID_PATTERN.test(desktopId));
}

function canUseRemotePresence() {
  return workspaceCanUseRemote(activeWorkspaceId) && desktopCanUseRemote(activeDesktopId);
}

function cursorKey(workspaceId: string | null | undefined, desktopId: string, remoteClientId: string) {
  return `${workspaceId || "local"}:${desktopId}:${remoteClientId}`;
}

function validCursorNumber(value: number) {
  return Number.isFinite(value) && value >= MIN_CURSOR_COORDINATE && value <= MAX_CURSOR_COORDINATE;
}

function validViewportNumber(value: number) {
  return Number.isFinite(value) && value >= 0 && value <= MAX_CURSOR_COORDINATE;
}

function resetPublishThrottle() {
  lastPublishAt = 0;
}

function normalizeLocalCursor(cursor: DesktopPresenceCursor) {
  if (!cursor.clientId || !cursor.userId || !cursor.desktopId) return null;
  if (!validCursorNumber(cursor.x) || !validCursorNumber(cursor.y)) return null;
  const viewportWidth = Number(cursor.viewportWidth || 0);
  const viewportHeight = Number(cursor.viewportHeight || 0);
  if (!validViewportNumber(viewportWidth) || !validViewportNumber(viewportHeight)) return null;
  return {
    ...cursor,
    name: cursor.name || "Teammate",
    color: cursor.color || colorFor(cursor.userId || cursor.clientId),
    workspaceId: cursor.workspaceId || null,
    viewportWidth,
    viewportHeight,
    lastSeen: Number(cursor.lastSeen || Date.now()),
  };
}

function normalizeRemoteCursorEvent(data: RemotePresenceEvent, workspaceId: string, desktopId: string) {
  if (!data.client_id) return null;
  const x = data.type === "join" ? Number(data.x || 0) : Number(data.x);
  const y = data.type === "join" ? Number(data.y || 0) : Number(data.y);
  const viewportWidth = data.type === "join" ? Number(data.viewport_width || 0) : Number(data.viewport_width);
  const viewportHeight = data.type === "join" ? Number(data.viewport_height || 0) : Number(data.viewport_height);
  if (!validCursorNumber(x) || !validCursorNumber(y)) return null;
  if (!validViewportNumber(viewportWidth) || !validViewportNumber(viewportHeight)) return null;
  const lastSeen = Number(data.last_seen || Date.now());
  if (!Number.isFinite(lastSeen) || lastSeen <= 0) return null;
  return {
    clientId: data.client_id,
    userId: data.user_id || data.client_id,
    name: data.name || "Teammate",
    color: data.color || colorFor(data.user_id || data.client_id),
    activeModule: data.active_module,
    activeTitle: data.active_title,
    workspaceId,
    desktopId,
    x,
    y,
    viewportWidth,
    viewportHeight,
    lastSeen,
  } satisfies DesktopPresenceCursor;
}

function buildPresenceWebSocketUrl(workspaceId: string, desktopId: string) {
  const backendUrl = getBackendUrl() || window.location.origin;
  const url = new URL(`/api/v1/workspaces/${encodeURIComponent(workspaceId)}/desktop-spaces/${encodeURIComponent(desktopId)}/presence/ws`, backendUrl);
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  return url.toString();
}

function closeRemoteSocket(sendLeave = true) {
  if (reconnectTimer) clearTimeout(reconnectTimer);
  reconnectTimer = null;
  if (socket) {
    if (sendLeave && socket.readyState === WebSocket.OPEN) {
      try {
        socket.send(JSON.stringify({ type: "leave" }));
      } catch {
        /* ignore */
      }
    }
    socket.close();
  }
  socket = null;
}

function connectRemotePresence() {
  if (typeof window === "undefined" || !canUseRemotePresence()) {
    closeRemoteSocket(false);
    return;
  }
  closeRemoteSocket(false);

  const ws = new WebSocket(buildPresenceWebSocketUrl(activeWorkspaceId!, activeDesktopId));
  socket = ws;

  ws.onopen = () => {
    reconnectAttempts = 0;
    ws.send(JSON.stringify({ type: "heartbeat" }));
  };

  ws.onmessage = (event) => {
    let data: RemotePresenceEvent;
    try {
      data = JSON.parse(String(event.data)) as RemotePresenceEvent;
    } catch {
      return;
    }
    if (!data || data.type === "heartbeat" || data.type === "error") return;
    const eventWorkspaceId = data.workspace_id || activeWorkspaceId;
    const eventDesktopId = data.desktop_space_id || activeDesktopId;
    if (!eventWorkspaceId || !eventDesktopId) return;
    if (eventWorkspaceId !== activeWorkspaceId || eventDesktopId !== activeDesktopId) return;
    if (data.type === "desktop_space_updated") {
      notifyDesktopSpaceUpdated({
        workspaceId: eventWorkspaceId,
        desktopId: eventDesktopId,
        revision: data.revision,
        action: data.action === "deleted" ? "deleted" : "updated",
        lastSeen: data.last_seen || Date.now(),
      });
      return;
    }
    if (!data.client_id) return;
    const key = cursorKey(eventWorkspaceId, eventDesktopId, data.client_id);
    if (data.type === "leave") {
      cursors.delete(key);
      clearFollowedCursorIfClient(data.client_id);
      publish();
      return;
    }
    if (data.type !== "cursor" && data.type !== "join") return;
    if ((data.user_id || data.client_id) === identity.userId) return;
    const cursor = normalizeRemoteCursorEvent(data, eventWorkspaceId, eventDesktopId);
    if (!cursor) return;
    cursors.set(key, cursor);
    publish();
  };

  ws.onclose = () => {
    if (socket !== ws || !canUseRemotePresence()) return;
    socket = null;
    reconnectAttempts += 1;
    const delay = Math.min(5000, 400 * reconnectAttempts);
    reconnectTimer = setTimeout(connectRemotePresence, delay);
  };
}

export function startDesktopPresence() {
  if (typeof window === "undefined" || channel) return;

  channel = new BroadcastChannel(CHANNEL_NAME);
  channel.addEventListener("message", (event: MessageEvent<PresenceMessage>) => {
    const message = event.data;
    if (!message || typeof message !== "object") return;

    if (message.type === "cursor") {
      const cursor = normalizeLocalCursor(message.payload);
      if (!cursor || cursor.clientId === clientId) return;
      if (cursor.userId === identity.userId) return;
      cursors.set(cursorKey(cursor.workspaceId, cursor.desktopId, cursor.clientId), cursor);
      publish();
      return;
    }

    if (message.type === "desktop_space_updated") {
      notifyDesktopSpaceUpdated(message.payload);
      return;
    }

    if (message.type === "leave") {
      cursors.delete(cursorKey(message.workspaceId, message.desktopId, message.clientId));
      publish();
    }
  });

  cleanupTimer = setInterval(() => {
    const now = Date.now();
    let changed = false;
    for (const [id, cursor] of cursors.entries()) {
      if (now - cursor.lastSeen > CURSOR_TTL_MS) {
        cursors.delete(id);
        clearFollowedCursorIfClient(cursor.clientId);
        changed = true;
      }
    }
    if (changed) publish();
  }, 1000);
}

export function stopDesktopPresence() {
  post({ type: "leave", clientId, workspaceId: activeWorkspaceId, desktopId: activeDesktopId });
  closeRemoteSocket();
  channel?.close();
  channel = null;
  if (cleanupTimer) clearInterval(cleanupTimer);
  cleanupTimer = null;
  cursors.clear();
  desktopSpaceUpdateListeners.clear();
  setFollowedCursorId(null);
  resetPublishThrottle();
  publish();
}

export function setDesktopPresenceContext(desktopId: string, nextIdentity: PresenceIdentity) {
  const nextDesktopId = desktopId || "personal";
  const desktopChanged = nextDesktopId !== activeDesktopId;
  if (desktopChanged) closeRemoteSocket();
  activeDesktopId = nextDesktopId;
  identity = {
    ...nextIdentity,
    name: nextIdentity.name || "Teammate",
    color: nextIdentity.color || colorFor(nextIdentity.userId || nextIdentity.name || clientId),
  };
  if (desktopChanged) {
    cursors.clear();
    setFollowedCursorId(null);
    resetPublishThrottle();
    if (canUseRemotePresence()) connectRemotePresence();
    else closeRemoteSocket();
  }
  publish();
}

export function setDesktopPresenceWorkspace(workspaceId: string | null) {
  const nextWorkspaceId = workspaceId || null;
  if (nextWorkspaceId === activeWorkspaceId) return;
  closeRemoteSocket();
  activeWorkspaceId = nextWorkspaceId;
  cursors.clear();
  setFollowedCursorId(null);
  resetPublishThrottle();
  publish();
  if (canUseRemotePresence()) connectRemotePresence();
  else closeRemoteSocket();
}

export function connectDesktopPresenceRemote(workspaceId: string | null, desktopId: string) {
  const workspaceChanged = (workspaceId || null) !== activeWorkspaceId;
  const desktopChanged = desktopId !== activeDesktopId;
  if (workspaceChanged || desktopChanged) closeRemoteSocket();
  activeWorkspaceId = workspaceId || null;
  activeDesktopId = desktopId || "personal";
  if (workspaceChanged || desktopChanged) {
    cursors.clear();
    setFollowedCursorId(null);
    resetPublishThrottle();
    publish();
    if (canUseRemotePresence()) connectRemotePresence();
    else closeRemoteSocket();
  }
}

export function publishDesktopCursor(desktopId: string, x: number, y: number, viewportWidth = 0, viewportHeight = 0) {
  if (!channel && !socket) return;
  if (desktopId !== activeDesktopId) return;
  if (!validCursorNumber(x) || !validCursorNumber(y)) return;
  if (!validViewportNumber(viewportWidth) || !validViewportNumber(viewportHeight)) return;
  const now = Date.now();
  if (now - lastPublishAt < CURSOR_THROTTLE_MS) return;
  lastPublishAt = now;

  if (socket?.readyState === WebSocket.OPEN && canUseRemotePresence()) {
    socket.send(JSON.stringify({
      type: "cursor",
      x,
      y,
      viewport_width: viewportWidth,
      viewport_height: viewportHeight,
      active_module: identity.activeModule,
      active_title: identity.activeTitle,
    }));
  }

  post({
    type: "cursor",
    payload: {
      clientId,
      userId: identity.userId,
      name: identity.name,
      color: identity.color || colorFor(identity.userId || clientId),
      activeModule: identity.activeModule,
      activeTitle: identity.activeTitle,
      workspaceId: activeWorkspaceId,
      desktopId,
      x,
      y,
      viewportWidth,
      viewportHeight,
      lastSeen: now,
    },
  });
}

export function publishDesktopSpaceUpdated(desktopId: string, revision?: string) {
  if (desktopId !== activeDesktopId) return;
  post({
    type: "desktop_space_updated",
    payload: {
      workspaceId: activeWorkspaceId,
      desktopId,
      revision,
      action: "updated",
      lastSeen: Date.now(),
    },
  });
}

export function onDesktopSpaceUpdated(listener: (event: DesktopSpaceUpdateEvent) => void) {
  desktopSpaceUpdateListeners.add(listener);
  return () => {
    desktopSpaceUpdateListeners.delete(listener);
  };
}

export function followDesktopCursor(cursorId: string | null) {
  setFollowedCursorId(cursorId || null);
}

export const followedDesktopCursor = {
  subscribe: subscribeFollowedCursor,
};

export const desktopPresenceStore = {
  subscribe,
};
