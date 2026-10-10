// FounderOS v1 lib/roadmap.ts + the pure half of components/RoadmapBoard.tsx.
import type { BadgeTone } from '$lib/founderos/kit';
import type { RoadmapItem, RoadmapStatus } from './types';

export type QuarterGroup = { quarter: string; items: RoadmapItem[] };

export function groupRoadmapByQuarter(items: RoadmapItem[]): QuarterGroup[] {
	const byQuarter = new Map<string, RoadmapItem[]>();
	for (const item of items) {
		const bucket = byQuarter.get(item.quarter) ?? [];
		bucket.push(item);
		byQuarter.set(item.quarter, bucket);
	}
	// '2026-Q1' sorts chronologically as a plain string.
	return [...byQuarter.entries()]
		.sort(([a], [b]) => a.localeCompare(b))
		.map(([quarter, quarterItems]) => ({ quarter, items: quarterItems }));
}

export const STATUS_BADGE: Record<RoadmapStatus, { tone: BadgeTone; ghost: boolean; label: string }> = {
	done: { tone: 'ok', ghost: false, label: 'Done' },
	now: { tone: 'accent', ghost: false, label: 'Now' },
	next: { tone: 'warn', ghost: false, label: 'Next' },
	later: { tone: 'default', ghost: true, label: 'Later' }
};

export const FILTERS = ['All', 'Done', 'Now', 'Next', 'Later'] as const;
export type Filter = (typeof FILTERS)[number];

/** The rows a phase owns: its progress bar is done/total of exactly these. */
export function owned(board: RoadmapItem[], phaseId: string): RoadmapItem[] {
	return board.filter((i) => i.phaseId === phaseId);
}

/** done/total of a phase's rows as a whole percent; no rows reads 0, never NaN. */
export function pctOf(board: RoadmapItem[], phaseId: string): number {
	const rows = owned(board, phaseId);
	if (rows.length === 0) return 0;
	return Math.round((rows.filter((i) => i.status === 'done').length / rows.length) * 100);
}

/** The selected phase narrows the quarters, then the status chip filters on top. */
export function visibleRows(board: RoadmapItem[], phaseId: string | null, filter: Filter): RoadmapItem[] {
	return board.filter(
		(i) =>
			(phaseId === null || i.phaseId === phaseId) &&
			(filter === 'All' || i.status === (filter.toLowerCase() as RoadmapStatus))
	);
}

/** '2026-Q3' reads '2026 · Q3' on the quarter header. */
export const quarterLabel = (q: string) => q.replace('-', ' · ');
