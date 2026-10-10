import { describe, expect, it } from 'vitest';
import { SEEN_KEY, changedIds, markSeenMap, parseSeenMap, revisionMap, tabBadges } from './deliverables-seen';
import type { DeliverableGroup } from './types';

const g = (items: Array<[string, string]>): DeliverableGroup[] => [
	{
		name: 'Agent files',
		items: items.map(([id, revision]) => ({ id, name: id, kind: 'file', url: null, meta: '', modifiedAt: '', sizeBytes: 1, accessCode: '', title: id, summary: '', revision }))
	}
];

describe('deliverables seen map (FounderOS v1 lib/deliverables-seen.ts)', () => {
	it('shares production’s storage key', () => {
		expect(SEEN_KEY).toBe('founderos-os:deliverables-seen');
	});

	it('maps every row id to its revision', () => {
		expect(revisionMap(g([['a', 'r1'], ['b', 'r2']]))).toEqual({ a: 'r1', b: 'r2' });
	});

	it('a never-looked browser badges nothing', () => {
		expect(changedIds({ a: 'r1' }, null)).toEqual({ added: [], updated: [] });
	});

	it('new ids are added, rewritten ones are updated; a blank (legacy) revision never pings', () => {
		expect(changedIds({ a: 'r1', b: 'r2', c: 'r3', d: 'r4' }, { a: 'r1', b: 'old', d: '' })).toEqual({ added: ['c'], updated: ['b'] });
	});

	it('parses the current map and the legacy string[] store; junk reads as never-looked', () => {
		expect(parseSeenMap(null)).toBeNull();
		expect(parseSeenMap('not json')).toBeNull();
		expect(parseSeenMap('["a",1]')).toEqual({ a: '' });
		expect(parseSeenMap('{"a":"r1","b":2}')).toEqual({ a: 'r1' });
	});

	it('a look adopts current revisions and prunes what is gone, but an empty board leaves the store alone', () => {
		expect(markSeenMap({ gone: 'x', a: 'old' }, { a: 'r1' })).toEqual({ a: 'r1' });
		expect(markSeenMap({ a: 'r1' }, {})).toEqual({ a: 'r1' });
		expect(markSeenMap(null, { a: 'r1' })).toEqual({ a: 'r1' });
	});

	it('tab badges: nothing before hydration, then counts of new and changed', () => {
		expect(tabBadges(false, { a: 'r1' }, {})).toEqual({ unseen: 0, changed: 0 });
		expect(tabBadges(true, { a: 'r1', b: 'r2' }, { a: 'r0' })).toEqual({ unseen: 1, changed: 1 });
	});
});
