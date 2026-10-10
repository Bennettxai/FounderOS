import { describe, expect, test } from 'vitest';
import {
	assignLanes,
	buildCommsSources,
	entryKey,
	formatDuration,
	hourBounds,
	isPlaudSample,
	itemsForSource,
	removeItem,
	replyTarget,
	weekLayout,
	type CalEvent,
	type DigestEntry,
	type Lane,
	type SlackCard,
	type SlackChannel
} from './model';

// Ported from FounderOS v1 tests/comms-panes.test.ts, calendar-layout.test.ts,
// digest-reads.test.ts and recordings.test.ts.

const item = (id: string, ts: string, unread = 0, extra: Record<string, unknown> = {}) => ({
	id,
	sender: `sender-${id}`,
	preview: `preview ${id}`,
	ts,
	unread,
	...extra
});

const lanes: Lane[] = [
	{
		id: 'inbox-1',
		name: 'Ops',
		source: 'email',
		state: 'connected',
		detail: 'ops inbox',
		items: [item('a', '2026-09-07T10:00:00.000Z', 1, { replyTo: 'a@x.com' }), item('b', '2026-09-07T08:00:00.000Z', 1)],
		unread: 2
	},
	{ id: 'inbox-2', name: 'Personal', source: 'email', state: 'error', detail: 'auth failed', items: [], unread: 0 },
	{ id: 'whatsapp', name: 'WhatsApp', source: 'whatsapp', state: 'connected', detail: 'live', items: [item('w1', '2026-09-07T09:00:00.000Z', 1)], unread: 1 }
];
const slackCards: SlackCard[] = [
	{ id: 'c1', name: 'Acme', channel: '#acme', lastText: 'hi', lastTs: '2026-09-07T09:30:00.000Z', unread: 3, heat: 'hot', waiting: 'you', live: true },
	{ id: 'c2', name: 'Globex', channel: null, lastText: '', lastTs: null, unread: 0, heat: null, waiting: 'none', live: false }
];
const channels: SlackChannel[] = [
	{ id: 'ch1', name: 'general', isMember: true, isPrivate: false, members: 8, topic: '' },
	{ id: 'ch2', name: 'ops', isMember: true, isPrivate: true, members: 3, topic: 'ops' }
];

describe('buildCommsSources', () => {
	const sources = buildCommsSources(lanes, slackCards, channels);
	test('rail order: All, each lane, slack clients, slack channels', () => {
		expect(sources.map((s) => s.id)).toEqual(['all', 'inbox-1', 'inbox-2', 'whatsapp', 'slack-clients', 'slack-channels']);
	});
	test('All aggregates the message lanes; slack keeps its own counts', () => {
		expect(sources[0]).toMatchObject({ unread: 3, count: 3 });
		expect(sources.find((s) => s.id === 'slack-clients')).toMatchObject({ unread: 3, count: 2 });
		expect(sources.find((s) => s.id === 'slack-channels')).toMatchObject({ unread: 0, count: 2 });
	});
	test('lane sources carry their connector state', () => {
		expect(sources.find((s) => s.id === 'inbox-2')!.state).toBe('error');
	});
});

describe('itemsForSource / removeItem', () => {
	test('All interleaves every lane newest first, tagged with its lane', () => {
		const rows = itemsForSource(lanes, 'all');
		expect(rows.map((r) => r.id)).toEqual(['a', 'w1', 'b']);
		expect(rows[1].laneName).toBe('WhatsApp');
	});
	test('one lane keeps its own order; an unknown id is empty', () => {
		expect(itemsForSource(lanes, 'inbox-1').map((r) => r.id)).toEqual(['a', 'b']);
		expect(itemsForSource(lanes, 'nope')).toEqual([]);
	});
	test('removal is pure: it recounts unread and leaves the undo state untouched', () => {
		const next = removeItem(lanes, 'a');
		expect(next[0].items.map((i) => i.id)).toEqual(['b']);
		expect(next[0].unread).toBe(1);
		expect(lanes[0].items).toHaveLength(2);
		expect(next[2]).toBe(lanes[2]);
	});
});

describe('calendar layout', () => {
	test('non-overlapping events take the full width; a cluster splits into lanes', () => {
		expect(assignLanes([{ startMin: 540, endMin: 600 }, { startMin: 600, endMin: 660 }])).toEqual([
			{ lane: 0, lanes: 1 },
			{ lane: 0, lanes: 1 }
		]);
		expect(assignLanes([{ startMin: 540, endMin: 660 }, { startMin: 570, endMin: 600 }, { startMin: 600, endMin: 630 }])).toEqual([
			{ lane: 0, lanes: 2 },
			{ lane: 1, lanes: 2 },
			{ lane: 1, lanes: 2 }
		]);
	});
	test('the hour window covers the events with an 8-hour minimum', () => {
		expect(hourBounds([])).toEqual({ loHour: 8, hiHour: 18 });
		expect(hourBounds([{ startMin: 22 * 60, endMin: 23 * 60 + 30 }])).toEqual({ loHour: 16, hiHour: 24 });
		expect(hourBounds([{ startMin: 6 * 60, endMin: 20 * 60 }])).toEqual({ loHour: 6, hiHour: 20 });
	});
	test('weekLayout buckets events into seven local days from today and splits all-day', () => {
		const now = new Date(2026, 8, 24, 15, 0);
		const at = (d: number, h: number) => new Date(2026, 8, d, h).toISOString();
		const ev = (id: string, start: string, allDay = false): CalEvent => ({
			id,
			account: 'Ops',
			color: '#fff',
			title: id,
			start,
			end: null,
			allDay,
			location: null,
			joinUrl: null
		});
		const w = weekLayout([ev('today', at(24, 10)), ev('fri', at(25, 11)), ev('offsite', at(26, 0), true), ev('next week', at(31, 9))], now.toISOString());
		expect(w.columns).toHaveLength(7);
		expect(w.timed[0].map((p) => p.ev.id)).toEqual(['today']);
		expect(w.timed[0][0]).toMatchObject({ startMin: 600, endMin: 630 });
		expect(w.timed[1].map((p) => p.ev.id)).toEqual(['fri']);
		expect(w.allDay[2].map((e) => e.id)).toEqual(['offsite']);
		expect(w.timed.flat().some((p) => p.ev.id === 'next week')).toBe(false);
	});
});

const entry = (over: Partial<DigestEntry> = {}): DigestEntry => ({
	tier: 'people',
	rank: 2,
	reason: 'family',
	source: 'email',
	sender: 'someone@example.com',
	title: 'Personal — someone',
	preview: 'hello there',
	ts: '2026-08-18T09:00:00.000Z',
	...over
});

describe('entryKey / replyTarget', () => {
	test('the key is channel + sender + time, never the tier', () => {
		expect(entryKey(entry())).toBe('email|someone@example.com|2026-08-18T09:00:00.000Z');
		expect(entryKey(entry({ tier: 'call', rank: 0, reason: 'x' }))).toBe(entryKey(entry()));
		expect(entryKey(entry({ source: 'whatsapp' }))).not.toBe(entryKey(entry()));
	});
	test('email replies, WhatsApp deep-links, a group gets no fake link, slack points at the channel', () => {
		expect(replyTarget(entry({ replyTo: 'a@b.com' }))).toEqual({ kind: 'email', href: 'mailto:a@b.com', label: 'Reply' });
		expect(replyTarget(entry({ source: 'whatsapp', sender: 'Mom', replyTo: '447700900000' })).href).toBe('https://wa.me/447700900000');
		expect(replyTarget(entry({ source: 'whatsapp', sender: 'Cohort (18)' })).href).toBeNull();
		expect(replyTarget(entry({ source: 'slack', replyTo: '#general' })).href).toBe('slack://channel?team=&id=general');
	});
});

describe('recordings format', () => {
	test('durations read like a human', () => {
		expect(formatDuration(null)).toBe('');
		expect(formatDuration(0)).toBe('<1m');
		expect(formatDuration(4)).toBe('4m');
		expect(formatDuration(81)).toBe('1h21m');
		expect(formatDuration(120)).toBe('2h');
	});
	test("Plaud's bundled samples are recognised by title", () => {
		expect(isPlaudSample('  Welcome to PLAUD.AI ')).toBe(true);
		expect(isPlaudSample('Site walk')).toBe(false);
	});
});
