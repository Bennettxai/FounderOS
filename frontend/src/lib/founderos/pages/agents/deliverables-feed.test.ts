import { describe, expect, it } from 'vitest';
import { DONE_SOURCE, deliverablesFeed, feedCounts } from './deliverables-feed';
import type { DeliverableGroup, DeliverableItem, PaperclipIssue } from './types';

const item = (id: string, kind: 'file' | 'link', modifiedAt: string): DeliverableItem => ({
	id,
	name: id,
	kind,
	url: kind === 'link' ? `https://x/${id}` : null,
	meta: '',
	modifiedAt,
	sizeBytes: kind === 'file' ? 10 : null,
	accessCode: '',
	title: id,
	summary: '',
	revision: 'r'
});
const issue = (id: string, status: string, updatedAt: string | null): PaperclipIssue => ({
	id,
	identifier: `BEN-${id}`,
	title: id,
	status,
	assigneeName: null,
	updatedAt
});

describe('deliverables feed (FounderOS v1 lib/deliverables-feed.ts)', () => {
	const groups: DeliverableGroup[] = [
		{ name: 'Proposals', items: [item('p1', 'link', '2026-09-20T00:00:00Z')] },
		{ name: 'Agent files', items: [item('f1', 'file', '2026-09-22T00:00:00Z'), item('f2', 'file', '2026-09-20T00:00:00Z')] }
	];
	const issues = [issue('1', 'done', '2026-09-21T00:00:00Z'), issue('2', 'in_progress', '2026-09-29T00:00:00Z'), issue('3', 'done', null)];

	it('is one flat list, newest first, with finished board tasks folded in', () => {
		const feed = deliverablesFeed(groups, issues);
		expect(feed.map((r) => r.id)).toEqual(['f1', 'done:1', 'p1', 'f2', 'done:3']);
		expect(feed.map((r) => r.source)).toEqual(['Agent files', DONE_SOURCE, 'Proposals', 'Agent files', DONE_SOURCE]);
	});

	it('equal times keep folder order, and no groups is just the done tasks', () => {
		expect(deliverablesFeed(groups, []).slice(1).map((r) => r.id)).toEqual(['p1', 'f2']);
		expect(deliverablesFeed(null, issues).map((r) => r.id)).toEqual(['done:1', 'done:3']);
	});

	it('counts each kind for the chips', () => {
		expect(feedCounts(deliverablesFeed(groups, issues))).toEqual({ all: 5, files: 2, links: 1, done: 2 });
	});
});
