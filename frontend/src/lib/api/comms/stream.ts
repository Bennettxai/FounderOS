// Comms realtime stream.
//
// Single SSE connection to /api/comms/stream multiplexing every realtime
// signal the comms tabs care about: new mail, mail mutations, new channel
// messages, channel metadata changes. Backed by a Svelte store for status
// (connecting / connected / reconnecting / disconnected) plus a typed
// subscriber API per event kind.
//
// Backend contract (Ghost — pending /api/comms/stream).
// Each event arrives as a named SSE event with a JSON payload:
//
//   event: email.received
//   data: <UnifiedEmail>
//
//   event: email.updated
//   data: { id: string; provider: "gmail"|"outlook";
//           changes: Partial<UnifiedEmail> }
//
//   event: message.received
//   data: <CommsMessage>
//
//   event: channel.updated
//   data: { id: string; provider: "slack"|"teams";
//           changes: Partial<CommsChannel> }
//
// Mock / dev hook: any code can dispatch real events on
//   window.__commsStream__: EventTarget
// and the wrapper will route them through the same subscriber API. Use
// `forceMockStream()` from a dev console to swap the live EventSource for
// the mock target without reloading.
import { writable, type Readable } from "svelte/store";
import { browser } from "$app/environment";
import { getApiBaseUrl } from "$lib/api/base";
import type { UnifiedEmail, EmailProvider } from "./types";
import type {
  CommsChannel,
  CommsMessage,
  ChannelProvider,
} from "./channels";

// ---------------------------------------------------------------------------
// Event types — exported so handlers can be typed without re-importing the
// whole module.
// ---------------------------------------------------------------------------

export interface EmailReceivedEvent {
  email: UnifiedEmail;
}

export interface EmailUpdatedEvent {
  id: string;
  provider: EmailProvider;
  changes: Partial<UnifiedEmail>;
}

export interface MessageReceivedEvent {
  message: CommsMessage;
}

export interface ChannelUpdatedEvent {
  id: string;
  provider: ChannelProvider;
  changes: Partial<CommsChannel>;
}

export type EmailReceivedHandler = (event: EmailReceivedEvent) => void;
export type EmailUpdatedHandler = (event: EmailUpdatedEvent) => void;
export type MessageReceivedHandler = (event: MessageReceivedEvent) => void;
export type ChannelUpdatedHandler = (event: ChannelUpdatedEvent) => void;

export type CommsStreamStatus =
  | "idle"
  | "connecting"
  | "connected"
  | "reconnecting"
  | "disconnected";

export interface CommsStreamHandle {
  close(): void;
}

// ---------------------------------------------------------------------------
// Module state. Single connection per page session.
// ---------------------------------------------------------------------------

const statusStore = writable<CommsStreamStatus>("idle");
const lastConnectedStore = writable<number | null>(null);
const lastDisconnectedStore = writable<number | null>(null);

// Public read-only views.
export const commsStreamStatus: Readable<CommsStreamStatus> = {
  subscribe: statusStore.subscribe,
};
export const commsStreamLastConnectedAt: Readable<number | null> = {
  subscribe: lastConnectedStore.subscribe,
};
export const commsStreamLastDisconnectedAt: Readable<number | null> = {
  subscribe: lastDisconnectedStore.subscribe,
};

// ---------------------------------------------------------------------------
// Subscriber registries. Adding a listener is O(1); removing is via the
// returned unsubscribe.
// ---------------------------------------------------------------------------

const emailReceivedHandlers = new Set<EmailReceivedHandler>();
const emailUpdatedHandlers = new Set<EmailUpdatedHandler>();
const messageReceivedHandlers = new Set<MessageReceivedHandler>();
const channelUpdatedHandlers = new Set<ChannelUpdatedHandler>();

export function onEmailReceived(handler: EmailReceivedHandler): () => void {
  emailReceivedHandlers.add(handler);
  return () => emailReceivedHandlers.delete(handler);
}

export function onEmailUpdated(handler: EmailUpdatedHandler): () => void {
  emailUpdatedHandlers.add(handler);
  return () => emailUpdatedHandlers.delete(handler);
}

export function onMessageReceived(
  handler: MessageReceivedHandler,
): () => void {
  messageReceivedHandlers.add(handler);
  return () => messageReceivedHandlers.delete(handler);
}

export function onChannelUpdated(
  handler: ChannelUpdatedHandler,
): () => void {
  channelUpdatedHandlers.add(handler);
  return () => channelUpdatedHandlers.delete(handler);
}

// ---------------------------------------------------------------------------
// Mock / dev hook.
// ---------------------------------------------------------------------------

declare global {
  interface Window {
    __commsStream__?: EventTarget;
  }
}

let mockTarget: EventTarget | null = null;
let mockUnsubscribers: Array<() => void> = [];

function attachMock(target: EventTarget): void {
  detachMock();
  mockTarget = target;
  const wire = <E extends Event>(name: string, handler: (e: E) => void) => {
    const fn = handler as EventListener;
    target.addEventListener(name, fn);
    mockUnsubscribers.push(() => target.removeEventListener(name, fn));
  };
  wire("email.received", (e: MessageEvent) =>
    dispatchEmailReceived(parsePayload(e.data)),
  );
  wire("email.updated", (e: MessageEvent) =>
    dispatchEmailUpdated(parsePayload(e.data)),
  );
  wire("message.received", (e: MessageEvent) =>
    dispatchMessageReceived(parsePayload(e.data)),
  );
  wire("channel.updated", (e: MessageEvent) =>
    dispatchChannelUpdated(parsePayload(e.data)),
  );
  setStatus("connected");
}

function detachMock(): void {
  for (const off of mockUnsubscribers) off();
  mockUnsubscribers = [];
  mockTarget = null;
}

function parsePayload(raw: unknown): unknown {
  if (typeof raw !== "string") return raw;
  try {
    return JSON.parse(raw);
  } catch {
    return null;
  }
}

// ---------------------------------------------------------------------------
// Connection lifecycle.
// ---------------------------------------------------------------------------

const STREAM_PATH = "/comms/stream";
const INITIAL_BACKOFF_MS = 1_000;
const MAX_BACKOFF_MS = 30_000;

let eventSource: EventSource | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let backoffMs = INITIAL_BACKOFF_MS;
let connectionRefcount = 0;

function setStatus(next: CommsStreamStatus): void {
  statusStore.set(next);
  if (next === "connected") lastConnectedStore.set(Date.now());
  if (next === "reconnecting" || next === "disconnected") {
    lastDisconnectedStore.set(Date.now());
  }
}

function clearReconnectTimer(): void {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
}

function dispatchEmailReceived(data: unknown): void {
  if (!data || typeof data !== "object") return;
  const event = isEmailReceivedPayload(data)
    ? data
    : { email: data as UnifiedEmail };
  for (const handler of emailReceivedHandlers) handler(event);
  resetBackoffOnFirstEvent();
}

function dispatchEmailUpdated(data: unknown): void {
  if (!data || typeof data !== "object") return;
  for (const handler of emailUpdatedHandlers) handler(data as EmailUpdatedEvent);
  resetBackoffOnFirstEvent();
}

function dispatchMessageReceived(data: unknown): void {
  if (!data || typeof data !== "object") return;
  const event = isMessageReceivedPayload(data)
    ? data
    : { message: data as CommsMessage };
  for (const handler of messageReceivedHandlers) handler(event);
  resetBackoffOnFirstEvent();
}

function dispatchChannelUpdated(data: unknown): void {
  if (!data || typeof data !== "object") return;
  for (const handler of channelUpdatedHandlers)
    handler(data as ChannelUpdatedEvent);
  resetBackoffOnFirstEvent();
}

function isEmailReceivedPayload(data: object): data is EmailReceivedEvent {
  return "email" in data;
}

function isMessageReceivedPayload(data: object): data is MessageReceivedEvent {
  return "message" in data;
}

function resetBackoffOnFirstEvent(): void {
  if (backoffMs !== INITIAL_BACKOFF_MS) backoffMs = INITIAL_BACKOFF_MS;
}

function openEventSource(): void {
  if (!browser) return;
  const baseUrl = getApiBaseUrl();
  if (!baseUrl) return;
  setStatus("connecting");
  let source: EventSource;
  try {
    source = new EventSource(`${baseUrl}${STREAM_PATH}`, {
      withCredentials: true,
    });
  } catch (err) {
    if (import.meta.env.DEV) console.error("[comms-stream] open failed", err);
    scheduleReconnect();
    return;
  }
  eventSource = source;

  source.onopen = () => {
    backoffMs = INITIAL_BACKOFF_MS;
    setStatus("connected");
  };

  source.onerror = () => {
    // Browsers fire onerror on disconnect, before any reconnect; treat as a
    // hint that we lost the line and own the retry ourselves.
    closeEventSource();
    scheduleReconnect();
  };

  source.addEventListener("email.received", (event) => {
    dispatchEmailReceived(parsePayload((event as MessageEvent).data));
  });
  source.addEventListener("email.updated", (event) => {
    dispatchEmailUpdated(parsePayload((event as MessageEvent).data));
  });
  source.addEventListener("message.received", (event) => {
    dispatchMessageReceived(parsePayload((event as MessageEvent).data));
  });
  source.addEventListener("channel.updated", (event) => {
    dispatchChannelUpdated(parsePayload((event as MessageEvent).data));
  });
}

function closeEventSource(): void {
  if (!eventSource) return;
  try {
    eventSource.close();
  } catch {
    // Already closed.
  }
  eventSource = null;
}

function scheduleReconnect(): void {
  clearReconnectTimer();
  setStatus("reconnecting");
  const delay = backoffMs;
  backoffMs = Math.min(backoffMs * 2, MAX_BACKOFF_MS);
  reconnectTimer = setTimeout(() => {
    if (connectionRefcount <= 0) return; // No subscribers — abort.
    openEventSource();
  }, delay);
}

// ---------------------------------------------------------------------------
// Public connect/close. Reference-counted so multiple mounts collapse to one
// underlying connection — the layout owns the lifecycle, but tests and
// component-level connectors stay safe.
// ---------------------------------------------------------------------------

export function connectCommsStream(): CommsStreamHandle {
  if (!browser) {
    return { close() {} };
  }
  connectionRefcount += 1;

  if (connectionRefcount === 1) {
    if (typeof window !== "undefined" && window.__commsStream__) {
      attachMock(window.__commsStream__);
    } else {
      openEventSource();
    }
  }

  return {
    close() {
      connectionRefcount = Math.max(0, connectionRefcount - 1);
      if (connectionRefcount > 0) return;
      clearReconnectTimer();
      closeEventSource();
      detachMock();
      backoffMs = INITIAL_BACKOFF_MS;
      setStatus("disconnected");
    },
  };
}

// Swap the live source for the dev mock without reloading. Useful from the
// browser console: `__forceCommsMockStream(new EventTarget())`.
export function forceMockStream(target: EventTarget): void {
  closeEventSource();
  clearReconnectTimer();
  attachMock(target);
}

// ---------------------------------------------------------------------------
// Test hooks.
// ---------------------------------------------------------------------------

export function __resetCommsStreamForTest(): void {
  closeEventSource();
  detachMock();
  clearReconnectTimer();
  emailReceivedHandlers.clear();
  emailUpdatedHandlers.clear();
  messageReceivedHandlers.clear();
  channelUpdatedHandlers.clear();
  backoffMs = INITIAL_BACKOFF_MS;
  connectionRefcount = 0;
  setStatus("idle");
}
