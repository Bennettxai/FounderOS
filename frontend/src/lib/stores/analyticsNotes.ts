/**
 * Analytics Notes Store
 *
 * Manages user notes attached to chart data points on the Analytics page.
 * Notes are persisted to localStorage and surfaced in the Daily Log page.
 *
 * Pattern: writable store with factory function + localStorage persistence.
 * Uses `browser` guard so SSR does not touch localStorage.
 */

import { writable, get } from "svelte/store";
import { browser } from "$app/environment";

// ── Types ─────────────────────────────────────────────────────────────────────

export interface AnalyticsNote {
  /** Unique identifier generated via crypto.randomUUID() */
  id: string;
  /** ISO date string (YYYY-MM-DD) — the date of the data point this note belongs to */
  date: string;
  /** ISO timestamp of when the note was created */
  createdAt: string;
  /** ISO timestamp of when the note was last updated */
  updatedAt: string;
  /** The note body text */
  content: string;
  /** Snapshot of the metric values at the time of the note */
  metricContext: {
    requests: number;
    tokens: number;
    cost: number;
    /** Formatted date label shown on the chart, e.g. "Mar 15" */
    label: string;
  };
}

// ── Storage helpers ────────────────────────────────────────────────────────────

const STORAGE_KEY = "bos-analytics-notes";

function loadNotes(): AnalyticsNote[] {
  if (!browser) return [];
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? (parsed as AnalyticsNote[]) : [];
  } catch {
    // Corrupted or non-JSON value — start fresh
    return [];
  }
}

function persistNotes(notes: AnalyticsNote[]): void {
  if (!browser) return;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(notes));
  } catch {
    // Storage quota exceeded or private-browsing restriction — silently skip
  }
}

// ── Store factory ──────────────────────────────────────────────────────────────

function createAnalyticsNotesStore() {
  const store = writable<AnalyticsNote[]>(loadNotes());

  // Persist every mutation to localStorage automatically
  store.subscribe((notes) => {
    persistNotes(notes);
  });

  // ── Mutations ────────────────────────────────────────────────────────────

  /**
   * Create a new note for a specific data-point date and return it.
   */
  function addNote(
    date: string,
    content: string,
    metricContext: AnalyticsNote["metricContext"],
  ): AnalyticsNote {
    const now = new Date().toISOString();
    const note: AnalyticsNote = {
      id: crypto.randomUUID(),
      date,
      createdAt: now,
      updatedAt: now,
      content,
      metricContext,
    };
    store.update((notes) => [note, ...notes]);
    return note;
  }

  /**
   * Update the text content of an existing note by id.
   * No-ops silently when the id is not found.
   */
  function updateNote(id: string, content: string): void {
    store.update((notes) =>
      notes.map((note) =>
        note.id === id
          ? { ...note, content, updatedAt: new Date().toISOString() }
          : note,
      ),
    );
  }

  /**
   * Remove a note by id. No-ops silently when the id is not found.
   */
  function deleteNote(id: string): void {
    store.update((notes) => notes.filter((note) => note.id !== id));
  }

  // ── Reads ────────────────────────────────────────────────────────────────

  /**
   * Return all notes whose `date` field matches the given YYYY-MM-DD string.
   * Reads the current snapshot synchronously — suitable for calling inside
   * Svelte reactive statements or event handlers.
   */
  function getNotesForDate(date: string): AnalyticsNote[] {
    return get(store).filter((note) => note.date === date);
  }

  /**
   * Return a snapshot of every note currently in the store.
   */
  function getAllNotes(): AnalyticsNote[] {
    return get(store);
  }

  return {
    subscribe: store.subscribe,
    addNote,
    updateNote,
    deleteNote,
    getNotesForDate,
    getAllNotes,
  };
}

// ── Singleton export ───────────────────────────────────────────────────────────

/** Singleton analytics notes store — shared across the entire app. */
export const analyticsNotes = createAnalyticsNotesStore();
