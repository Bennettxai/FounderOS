// The Deliverables tab as ONE flat feed (FounderOS v1 lib/deliverables-feed.ts,
// 2026-09-24: "What was done would go in Deliverables outside of the
// folders"). Agent files, proposals and every board task that reached `done`
// sit in a single list, newest first. The folder a file came from survives
// only as the row's source tag.
import type { DeliverableGroup, DeliverableItem, PaperclipIssue } from './types';

export const DONE_SOURCE = 'Board · done';

export type FeedRow =
	| { id: string; kind: 'item'; item: DeliverableItem; source: string; at: string | null }
	| { id: string; kind: 'done'; issue: PaperclipIssue; source: string; at: string | null };

const time = (iso: string | null): number => {
	const t = iso ? Date.parse(iso) : NaN;
	return Number.isFinite(t) ? t : -Infinity;
};

export function deliverablesFeed(groups: DeliverableGroup[] | null, issues: PaperclipIssue[]): FeedRow[] {
	const rows: FeedRow[] = [];
	for (const g of groups ?? []) {
		for (const item of g.items) rows.push({ id: item.id, kind: 'item', item, source: g.name, at: item.modifiedAt });
	}
	for (const issue of issues) {
		if (issue.status !== 'done') continue;
		rows.push({ id: `done:${issue.id}`, kind: 'done', issue, source: DONE_SOURCE, at: issue.updatedAt });
	}
	// stable sort: equal times keep folder order
	return rows
		.map((r, n) => ({ r, n }))
		.sort((a, b) => time(b.r.at) - time(a.r.at) || a.n - b.n)
		.map((x) => x.r);
}

export type FeedCounts = { all: number; files: number; links: number; done: number };

export function feedCounts(rows: FeedRow[]): FeedCounts {
	const c: FeedCounts = { all: rows.length, files: 0, links: 0, done: 0 };
	for (const r of rows) {
		if (r.kind === 'done') c.done++;
		else if (r.item.kind === 'link') c.links++;
		else c.files++;
	}
	return c;
}
