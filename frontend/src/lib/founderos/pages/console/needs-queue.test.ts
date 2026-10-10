import { describe, expect, it } from 'vitest';
import type { Classified, DeliverablesBody } from '../agents/types';
import {
	PRIMARY,
	SNOOZE_MS,
	ago,
	isSnoozed,
	markManyOpened,
	markOpened,
	parseOpenedMap,
	parseSnoozeMap,
	pruneOpened,
	pruneSnoozed,
	queueView,
	snoozeItem,
	subject,
	unseenStateOf,
	unsnooze
} from './needs-queue';

const NOW = Date.parse('2026-09-30T12:00:00Z');

function item(id: string, over: Partial<Classified> = {}): Classified {
	return {
		id,
		name: `${id}.md`,
		kind: 'file',
		url: null,
		meta: 'f7d1ec4a',
		modifiedAt: '2026-09-22T12:00:00.000Z',
		sizeBytes: 10,
		accessCode: '',
		title: `Title ${id}`,
		summary: '',
		revision: `file|${id}|10`,
		ask: 'draft',
		needsYou: true,
		deadline: null,
		overdue: false,
		label: 'draft to approve',
		action: '',
		person: false,
		glyph: '!',
		glyphTone: 'dim',
		why: 'a draft is waiting to go out',
		...over
	};
}

function body(open: Classified[], files = 200, decided = 24): DeliverablesBody {
	const fileItems = Array.from({ length: files }, (_, i) => ({ ...item(`f${i}`), kind: 'file' as const }));
	return {
		groups: [
			{ name: 'Vantage proposals', items: [{ ...item('proposal:x'), kind: 'link' }] },
			{ name: 'Agent files', items: fileItems }
		],
		decisions: [],
		dir: '/x',
		available: true,
		reason: '',
		needsYou: {
			open,
			decided: Array.from({ length: decided }, (_, i) => ({
				item: item(`d${i}`),
				decision: { id: `d${i}`, decision: 'dismissed' as const, decidedAt: '', decidedRevision: '', note: '' }
			}))
		}
	};
}

describe('needs you: the words (NeedsYouList.tsx)', () => {
	it('the white button says what approve means for each kind', () => {
		expect(PRIMARY.draft.label).toBe('publish');
		expect(PRIMARY.decision.label).toBe('go ahead');
		expect(PRIMARY.staged.label).toBe('send it');
		expect(PRIMARY.gate.label).toBe('unblock');
		expect(PRIMARY.request.label).toBe('approve');
	});
	it('says how long ago with "ago", like the queue', () => {
		expect(ago('2026-09-30T11:15:00Z', NOW)).toBe('45m ago');
		expect(ago('2026-09-30T07:00:00Z', NOW)).toBe('5h ago');
		expect(ago('2026-09-22T12:00:00Z', NOW)).toBe('8d ago');
		expect(ago('nope', NOW)).toBe('');
	});
	it('strips the agent bookkeeping from a filename', () => {
		expect(subject('STAGED-reply-to-sam-DELIVER-BEFORE-1700Z.md')).toBe('reply to sam');
		expect(subject('launch_copy-2026-09-20T0300Z.json')).toBe('launch copy');
	});
});

describe('needs you: snooze is a two-hour hold on his own view', () => {
	it('holds, wakes and expires', () => {
		const m = snoozeItem({}, 'a', NOW);
		expect(m.a).toBe(NOW + SNOOZE_MS);
		expect(isSnoozed(m, 'a', NOW + 1000)).toBe(true);
		expect(isSnoozed(m, 'a', NOW + SNOOZE_MS + 1)).toBe(false);
		expect(unsnooze(m, 'a')).toEqual({});
		expect(pruneSnoozed({ a: NOW - 1, b: NOW + 5 }, NOW)).toEqual({ b: NOW + 5 });
	});
	it('reads a stored map defensively', () => {
		expect(parseSnoozeMap(null)).toEqual({});
		expect(parseSnoozeMap('nope')).toEqual({});
		expect(parseSnoozeMap('[1]')).toEqual({});
		expect(parseSnoozeMap('{"a":5,"b":"x"}')).toEqual({ a: 5 });
	});
});

describe('needs you: the unseen dot', () => {
	it('is new until opened, updated when rewritten after, and silent before the store exists', () => {
		expect(unseenStateOf(null, 'a', 'r1')).toBeNull();
		expect(unseenStateOf({}, 'a', 'r1')).toBe('new');
		expect(unseenStateOf({ a: 'r1' }, 'a', 'r1')).toBeNull();
		expect(unseenStateOf({ a: 'r1' }, 'a', 'r2')).toBe('updated');
		expect(markOpened({}, 'a', 'r1')).toEqual({ a: 'r1' });
		expect(markManyOpened({ a: 'r0' }, [{ id: 'a', revision: 'r1' }, { id: 'b', revision: 'r2' }])).toEqual({ a: 'r1', b: 'r2' });
		expect(pruneOpened({ a: '1', b: '2' }, ['b'])).toEqual({ b: '2' });
		expect(pruneOpened({ a: '1' }, [])).toEqual({ a: '1' });
		expect(parseOpenedMap(null)).toBeNull();
		expect(parseOpenedMap('{"a":"r","b":3}')).toEqual({ a: 'r' });
	});
});

describe('needs you: the queue', () => {
	it('counts his queue against every agent file, and what he handled', () => {
		const v = queueView(body([item('a'), item('b')]), { local: {}, snoozed: {}, opened: null, now: NOW })!;
		expect(v.files).toBe(200);
		expect(v.queue.map((c) => c.id)).toEqual(['a', 'b']);
		expect(v.handled).toBe(24);
		expect(v.held).toBe(0);
	});
	it('keeps the Conductor ranking the bridge sent', () => {
		const v = queueView(body([item('z', { overdue: true }), item('a', { person: true })]), { local: {}, snoozed: {}, opened: null, now: NOW })!;
		expect(v.queue.map((c) => c.id)).toEqual(['z', 'a']);
		expect(v.overdue).toBe(1);
	});
	it('a row he just decided leaves at once and counts as handled', () => {
		const v = queueView(body([item('a'), item('b')]), { local: { a: 'dismissed' }, snoozed: {}, opened: null, now: NOW })!;
		expect(v.queue.map((c) => c.id)).toEqual(['b']);
		expect(v.handled).toBe(25);
	});
	it('a snoozed row is held, not handled', () => {
		const v = queueView(body([item('a'), item('b')]), { local: {}, snoozed: { a: NOW + 1000 }, opened: null, now: NOW })!;
		expect(v.queue.map((c) => c.id)).toEqual(['b']);
		expect(v.held).toBe(1);
		expect(v.handled).toBe(24);
	});
	it('counts unread rows once the opened store exists', () => {
		const v = queueView(body([item('a'), item('b')]), { local: {}, snoozed: {}, opened: { a: 'file|a|10' }, now: NOW })!;
		expect(v.unread).toBe(1);
	});
	it('no payload yet is no queue, not an empty one', () => {
		expect(queueView(null, { local: {}, snoozed: {}, opened: null, now: NOW })).toBeNull();
	});
});
