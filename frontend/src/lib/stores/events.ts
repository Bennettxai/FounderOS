import { writable } from "svelte/store";

export type ModuleEventType =
  | "task:created"
  | "project:created"
  | "client:created"
  | "deal:created"
  | "note:created";

export type ModuleEvent = {
  type: ModuleEventType;
  payload?: { id?: string; name?: string };
  timestamp: number;
};

const { subscribe, set } = writable<ModuleEvent | null>(null);

/** Read-only store — use emitModuleEvent() to publish. */
export const moduleEvents = { subscribe };

/**
 * Publish a module event so subscribed pages can reload their data.
 * Called after a chat slash command successfully creates an entity.
 */
export function emitModuleEvent(
  type: ModuleEventType,
  payload?: ModuleEvent["payload"],
): void {
  set({ type, timestamp: Date.now(), payload });
}
