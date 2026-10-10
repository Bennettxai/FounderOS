/**
 * Home's Needs you queue (FounderOS v1 components/NeedsYouList.tsx,
 * useDeliverables.ts, lib/deliverable-snooze.ts, lib/deliverables-opened.ts).
 *
 * The bridge classifies and ranks the queue (GET /pages/board/deliverables →
 * needsYou.open: people first, then deadlines) and partitions out what he has
 * already decided. This module only layers his own view on top: a decision he
 * just made (optimistic, until the reload lands), a two-hour snooze, and the
 * per-row unseen dot. Snooze and the dot live in this browser, as in prod.
 */
import type { Classified, DeliverablesBody } from '../agents/types';

export type Ask = 'staged' | 'decision' | 'gate' | 'request' | 'draft' | 'done' | 'output';

/** What the white button says, per kind. "Approve" is not a verb he uses. */
export const PRIMARY: Record<Ask, { label: string; busy: string; done: string }> = {
	staged: { label: 'send it', busy: 'sending', done: 'sent' },
	decision: { label: 'go ahead', busy: 'recording', done: 'called' },
	gate: { label: 'unblock', busy: 'opening', done: 'open' },
	request: { label: 'approve', busy: 'recording', done: 'approved' },
	draft: { label: 'publish', busy: 'publishing', done: 'published' },
	done: { label: 'ok', busy: 'saving', done: 'ok' },
	output: { label: 'ok', busy: 'saving', done: 'ok' }
};

export const primaryOf = (ask: string) => PRIMARY[(ask in PRIMARY ? ask : 'request') as Ask];

export function ago(iso: string, now: number = Date.now()): string {
	const ms = now - Date.parse(iso);
	if (!Number.isFinite(ms)) return '';
	const m = Math.floor(ms / 60000);
	if (m < 60) return `${m}m ago`;
	const h = Math.floor(m / 60);
	return h < 24 ? `${h}h ago` : `${Math.floor(h / 24)}d ago`;
}

/** Strip the agent's bookkeeping so the row reads as a subject, not a filename. */
export function subject(name: string): string {
	return name
		.replace(/\.[a-z0-9]+$/i, '')
		.replace(/^STAGED-/i, '')
		.replace(/-?DELIVER-BEFORE-\d{4}Z/i, '')
		.replace(/-?\d{4}-\d{2}-\d{2}(T\d{4}Z)?$/i, '')
		.replace(/-(COMMENT|PATCH)$/i, '')
		.replace(/[-_]+/g, ' ')
		.trim();
}

// ---- snooze: a two-hour hold on HIS view, never a decision agents see ----

export const SNOOZE_KEY = 'founderos-os:deliverables-snoozed';
export const SNOOZE_MS = 2 * 60 * 60 * 1000;
export type SnoozeMap = Record<string, number>;

export const snoozeItem = (map: SnoozeMap, id: string, now = Date.now()): SnoozeMap => ({ ...map, [id]: now + SNOOZE_MS });

export function unsnooze(map: SnoozeMap, id: string): SnoozeMap {
	const rest = { ...map };
	delete rest[id];
	return rest;
}

export function isSnoozed(map: SnoozeMap, id: string, now = Date.now()): boolean {
	const until = map[id];
	return typeof until === 'number' && until > now;
}

export function pruneSnoozed(map: SnoozeMap, now = Date.now()): SnoozeMap {
	const out: SnoozeMap = {};
	for (const [id, until] of Object.entries(map)) if (until > now) out[id] = until;
	return out;
}

function parseRecord(raw: string | null): Record<string, unknown> | null {
	if (raw === null) return null;
	try {
		const v: unknown = JSON.parse(raw);
		return typeof v === 'object' && v !== null && !Array.isArray(v) ? (v as Record<string, unknown>) : null;
	} catch {
		return null;
	}
}

export function parseSnoozeMap(raw: string | null): SnoozeMap {
	const out: SnoozeMap = {};
	for (const [k, v] of Object.entries(parseRecord(raw) ?? {})) if (typeof v === 'number' && Number.isFinite(v)) out[k] = v;
	return out;
}

// ---- the unseen dot: new since he last opened anything, or rewritten since ----

export const OPENED_KEY = 'founderos-os:deliverables-opened';
export type OpenedMap = Record<string, string>;
export type UnseenState = 'new' | 'updated' | null;

export function unseenStateOf(opened: OpenedMap | null, id: string, revision: string): UnseenState {
	if (opened === null) return null;
	if (!(id in opened)) return 'new';
	return opened[id] === revision ? null : 'updated';
}

export const markOpened = (opened: OpenedMap, id: string, revision: string): OpenedMap => ({ ...opened, [id]: revision });

export function markManyOpened(opened: OpenedMap, items: { id: string; revision: string }[]): OpenedMap {
	const next = { ...opened };
	for (const i of items) next[i.id] = i.revision;
	return next;
}

export function pruneOpened(opened: OpenedMap, presentIds: string[]): OpenedMap {
	if (presentIds.length === 0) return opened;
	const present = new Set(presentIds);
	const out: OpenedMap = {};
	for (const [id, rev] of Object.entries(opened)) if (present.has(id)) out[id] = rev;
	return out;
}

export function parseOpenedMap(raw: string | null): OpenedMap | null {
	const rec = parseRecord(raw);
	if (rec === null) return null;
	const out: OpenedMap = {};
	for (const [k, v] of Object.entries(rec)) if (typeof v === 'string') out[k] = v;
	return out;
}

/** localStorage, fail-soft: private mode or blocked storage is "never looked". */
export function readStore(key: string): string | null {
	try {
		return window.localStorage.getItem(key);
	} catch {
		return null;
	}
}
export function writeStore(key: string, value: unknown): void {
	try {
		window.localStorage.setItem(key, JSON.stringify(value));
	} catch {
		/* the state just lasts this session */
	}
}

// ---- the queue as he sees it ----

export type QueueView = {
	queue: Classified[];
	/** every agent file on the board: the "of M" */
	files: number;
	held: number;
	overdue: number;
	unread: number;
	handled: number;
};

export function queueView(
	body: DeliverablesBody | null,
	s: { local: Record<string, 'approved' | 'dismissed'>; snoozed: SnoozeMap; opened: OpenedMap | null; now: number }
): QueueView | null {
	if (body === null) return null;
	const files = (body.groups ?? []).flatMap((g) => g.items ?? []).filter((i) => i.kind === 'file').length;
	const open = body.needsYou?.open ?? [];
	const ranked = open.filter((c) => !(c.id in s.local));
	const queue = ranked.filter((c) => !isSnoozed(s.snoozed, c.id, s.now));
	return {
		queue,
		files,
		held: ranked.length - queue.length,
		overdue: queue.filter((c) => c.overdue).length,
		unread: queue.filter((c) => unseenStateOf(s.opened, c.id, c.revision) !== null).length,
		handled: (body.needsYou?.decided?.length ?? 0) + (open.length - ranked.length)
	};
}
