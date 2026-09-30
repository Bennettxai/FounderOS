'use client';

import { useSyncExternalStore } from 'react';
import type { BoardLivePayload } from '@/lib/board-live';

/**
 * The latest live board snapshot, shared in the browser (2026-09-24). BoardLive
 * owns the 4s poll and publishes every snapshot here; the /agents slab hero
 * subscribes, so its numbers move with the board below instead of freezing at
 * first paint. One poll, two readers. Before the first publish (and on the
 * server) readers see the server-rendered payload they were handed.
 */
let latest: BoardLivePayload | null = null;
const listeners = new Set<() => void>();

export function publishBoard(p: BoardLivePayload): void {
  if (p === latest) return;
  latest = p;
  for (const l of listeners) l();
}

const subscribe = (l: () => void) => {
  listeners.add(l);
  return () => {
    listeners.delete(l);
  };
};

export function useBoardSnapshot(initial: BoardLivePayload): BoardLivePayload {
  return useSyncExternalStore(
    subscribe,
    // a snapshot left over from an earlier visit never beats a fresher render
    () => (latest && latest.checkedAt >= initial.checkedAt ? latest : initial),
    () => initial,
  );
}
