import { writable } from 'svelte/store';
import type { CalendarEvent } from '$lib/api/calendar';
import * as calendarApi from '$lib/api/calendar';

interface CalendarState {
  events: CalendarEvent[];
  loading: boolean;
  syncing: boolean;
  error: string | null;
}

function createCalendarStore() {
  const { subscribe, update } = writable<CalendarState>({
    events: [],
    loading: false,
    syncing: false,
    error: null
  });

  return {
    subscribe,

    async loadEvents(filters?: { start?: string; end?: string; meetingType?: string; contextId?: string }) {
      update((s) => ({ ...s, loading: true, error: null }));
      try {
        const events = await calendarApi.getCalendarEvents(filters as any);
        update((s) => ({ ...s, events, loading: false }));
        return events;
      } catch (err) {
        console.error('Failed to load calendar events:', err);
        update((s) => ({ ...s, loading: false, error: err instanceof Error ? err.message : 'Failed to load events' }));
        return [] as CalendarEvent[];
      }
    },

    async syncCalendar() {
      update((s) => ({ ...s, syncing: true, error: null }));
      try {
        await calendarApi.syncCalendar();
        update((s) => ({ ...s, syncing: false }));
        return true;
      } catch (err) {
        console.error('Failed to sync calendar:', err);
        update((s) => ({ ...s, syncing: false, error: err instanceof Error ? err.message : 'Sync failed' }));
        return false;
      }
    },

    async loadTodayEvents() {
      update((s) => ({ ...s, loading: true, error: null }));
      try {
        const events = await calendarApi.getTodayEvents();
        update((s) => ({ ...s, events, loading: false }));
        return events;
      } catch (err) {
        console.error('Failed to load today events:', err);
        update((s) => ({ ...s, loading: false, error: err instanceof Error ? err.message : 'Failed to load today events' }));
        return [] as CalendarEvent[];
      }
    },

    async loadUpcoming(limit = 5) {
      update((s) => ({ ...s, loading: true, error: null }));
      try {
        const events = await calendarApi.getUpcomingEvents(limit);
        update((s) => ({ ...s, loading: false }));
        return events;
      } catch (err) {
        console.error('Failed to load upcoming events:', err);
        update((s) => ({ ...s, loading: false, error: err instanceof Error ? err.message : 'Failed to load upcoming events' }));
        return [] as CalendarEvent[];
      }
    },

    clearError() {
      update((s) => ({ ...s, error: null }));
    }
  };
}

export const calendar = createCalendarStore();
