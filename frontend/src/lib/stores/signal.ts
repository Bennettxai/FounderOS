import { writable, derived } from "svelte/store";
import type { SignalGenre, OsaMode } from "$lib/api/signal/types";

export interface SignalEvent {
  mode: OsaMode;
  genre: SignalGenre | null;
  docType: string | null;
  weight: number | null;
  confidence: number | null;
  timestamp: number;
}

interface SignalState {
  history: SignalEvent[];
  maxHistory: number;
}

function createSignalStore() {
  const { subscribe, update } = writable<SignalState>({
    history: [],
    maxHistory: 100,
  });

  return {
    subscribe,

    /** Record a new signal classification event */
    record(event: Omit<SignalEvent, "timestamp">) {
      update((s) => {
        const newHistory = [...s.history, { ...event, timestamp: Date.now() }];
        // Keep only the most recent N events
        if (newHistory.length > s.maxHistory) {
          newHistory.splice(0, newHistory.length - s.maxHistory);
        }
        return { ...s, history: newHistory };
      });
    },

    /** Clear signal history */
    clear() {
      update((s) => ({ ...s, history: [] }));
    },
  };
}

export const signalStore = createSignalStore();

/** Derived: genre distribution from history */
export const genreDistribution = derived(signalStore, ($s) => {
  const counts = new Map<string, number>();
  for (const event of $s.history) {
    if (event.genre) {
      counts.set(event.genre, (counts.get(event.genre) ?? 0) + 1);
    }
  }
  const total = $s.history.length || 1;
  return Array.from(counts.entries()).map(([genre, count]) => ({
    genre: genre as SignalGenre,
    count,
    percentage: Math.round((count / total) * 100),
  }));
});

/** Derived: mode distribution from history */
export const modeDistribution = derived(signalStore, ($s) => {
  const counts = new Map<string, number>();
  for (const event of $s.history) {
    counts.set(event.mode, (counts.get(event.mode) ?? 0) + 1);
  }
  const total = $s.history.length || 1;
  return Array.from(counts.entries()).map(([mode, count]) => ({
    mode: mode as OsaMode,
    count,
    percentage: Math.round((count / total) * 100),
  }));
});
