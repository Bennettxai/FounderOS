import { describe, expect, it } from 'vitest';
import type { CalendarEvent } from '$lib/api';
import {
	getEventDisplayStart,
	getEventsForDate,
	getTimedEventSegment,
	filterEventsBySelectedMembers
} from './calendarUtils';

function event(overrides: Partial<CalendarEvent> = {}): CalendarEvent {
	return {
		id: 'event-1',
		user_id: 'user-1',
		google_event_id: 'google-1',
		calendar_id: 'primary',
		title: 'All-day event',
		description: null,
		start_time: '2026-07-20T00:00:00Z',
		end_time: '2026-07-21T00:00:00Z',
		all_day: true,
		location: null,
		attendees: [],
		status: 'confirmed',
		visibility: null,
		html_link: null,
		source: 'google',
		meeting_type: 'other',
		context_id: null,
		project_id: null,
		client_id: null,
		recording_url: null,
		meeting_link: null,
		external_links: [],
		meeting_notes: null,
		meeting_summary: null,
		action_items: [],
		synced_at: null,
		created_at: '2026-07-20T00:00:00Z',
		updated_at: '2026-07-20T00:00:00Z',
		...overrides
	};
}

describe('calendar event date semantics', () => {
	it('keeps an all-day event on its source calendar date', () => {
		const allDay = event();
		const displayStart = getEventDisplayStart(allDay);

		expect(displayStart.getFullYear()).toBe(2026);
		expect(displayStart.getMonth()).toBe(6);
		expect(displayStart.getDate()).toBe(20);
		expect(getEventsForDate([allDay], new Date(2026, 6, 20))).toEqual([allDay]);
		expect(getEventsForDate([allDay], new Date(2026, 6, 19))).toEqual([]);
	});

	it('shows a multi-day all-day event on every covered date and treats the end as exclusive', () => {
		const multiDay = event({ end_time: '2026-07-23T00:00:00Z' });

		expect(getEventsForDate([multiDay], new Date(2026, 6, 20))).toEqual([multiDay]);
		expect(getEventsForDate([multiDay], new Date(2026, 6, 21))).toEqual([multiDay]);
		expect(getEventsForDate([multiDay], new Date(2026, 6, 22))).toEqual([multiDay]);
		expect(getEventsForDate([multiDay], new Date(2026, 6, 23))).toEqual([]);
	});

	it('keeps timed events on their local display date', () => {
		const timed = event({
			all_day: false,
			start_time: '2026-07-20T15:00:00Z',
			end_time: '2026-07-20T16:00:00Z'
		});
		const localStart = new Date(timed.start_time);

		expect(
			getEventsForDate(
				[timed],
				new Date(localStart.getFullYear(), localStart.getMonth(), localStart.getDate())
			)
		).toEqual([timed]);
	});

	it('splits an overnight event across both local calendar days', () => {
		const localStart = new Date(2026, 6, 20, 22, 30);
		const localEnd = new Date(2026, 6, 21, 6, 30);
		const overnight = event({
			title: 'Sleep',
			all_day: false,
			start_time: localStart.toISOString(),
			end_time: localEnd.toISOString()
		});
		const startDay = new Date(localStart.getFullYear(), localStart.getMonth(), localStart.getDate());
		const endDay = new Date(localEnd.getFullYear(), localEnd.getMonth(), localEnd.getDate());

		expect(getEventsForDate([overnight], startDay)).toEqual([overnight]);
		expect(getEventsForDate([overnight], endDay)).toEqual([overnight]);

		const firstSegment = getTimedEventSegment(overnight, startDay);
		const secondSegment = getTimedEventSegment(overnight, endDay);
		expect(firstSegment?.startMinute).toBe(localStart.getHours() * 60 + localStart.getMinutes());
		expect(firstSegment?.endMinute).toBe(1440);
		expect(secondSegment?.startMinute).toBe(0);
		expect(secondSegment?.endMinute).toBe(localEnd.getHours() * 60 + localEnd.getMinutes());
	});
});

describe('calendar event colors', () => {
	it('uses the exact imported Google color when available', async () => {
		const { getEventColor } = await import('./calendarUtils');
		expect(getEventColor(event({ color_hex: '#7986cb' }))).toBe('#7986cb');
	});

	it('uses the Google fallback instead of flattening unclassified events to gray', async () => {
		const { getEventColor } = await import('./calendarUtils');
		expect(getEventColor(event())).toBe('#34a853');
	});
});

describe('calendar member filters', () => {
	it('shows a blank calendar when every workspace calendar is switched off', () => {
		expect(filterEventsBySelectedMembers([event()], new Set(), true)).toEqual([]);
	});

	it('keeps personal events visible when the workspace has no member picker', () => {
		expect(filterEventsBySelectedMembers([event()], new Set(), false)).toHaveLength(1);
	});
});
