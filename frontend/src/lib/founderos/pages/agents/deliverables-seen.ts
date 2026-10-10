/**
 * The Deliverables tab badges (FounderOS v1 lib/deliverables-seen.ts,
 * useDeliverables.ts): green = arrived since he last looked at the tab, red =
 * was already here and has since been rewritten ("a red ping update when
 * proposals get updated from their initial state", 2026-08-20).
 *
 * It tracks a revision per id, in this browser (no per-user server state
 * exists), and anything unreadable reads as never-looked, which can only ever
 * suppress a badge, never invent one. The per-row dot is a separate store
 * (needs-queue's OPENED_KEY): looking at the tab clears these badges, and
 * must not clear every dot with them.
 */
import type { DeliverableGroup } from './types';

export const SEEN_KEY = 'founderos-os:deliverables-seen';

/** id → revision */
export type SeenMap = Record<string, string>;

export function revisionMap(groups: DeliverableGroup[]): SeenMap {
	const out: SeenMap = {};
	for (const g of groups) for (const i of g.items ?? []) out[i.id] = i.revision;
	return out;
}

/** Ids that are new, and ids that were already here but have since changed. */
export function changedIds(current: SeenMap, seen: SeenMap | null): { added: string[]; updated: string[] } {
	if (seen === null) return { added: [], updated: [] };
	const added: string[] = [];
	const updated: string[] = [];
	for (const [id, rev] of Object.entries(current)) {
		if (!(id in seen)) added.push(id);
		else if (seen[id] && seen[id] !== rev) updated.push(id);
	}
	return { added, updated };
}

/** Accepts both the current id→revision object and the legacy `string[]`. */
export function parseSeenMap(raw: string | null): SeenMap | null {
	if (raw === null) return null;
	let value: unknown;
	try {
		value = JSON.parse(raw);
	} catch {
		return null;
	}
	if (Array.isArray(value)) {
		const out: SeenMap = {};
		for (const v of value) if (typeof v === 'string') out[v] = '';
		return out;
	}
	if (typeof value !== 'object' || value === null) return null;
	const out: SeenMap = {};
	for (const [k, v] of Object.entries(value as Record<string, unknown>)) if (typeof v === 'string') out[k] = v;
	return out;
}

/** The store after a look: adopt current revisions, prune what is gone. An
 *  empty board is treated as a failed fetch and leaves the store alone. */
export function markSeenMap(seen: SeenMap | null, current: SeenMap): SeenMap {
	const merged: SeenMap = { ...(seen ?? {}), ...current };
	if (Object.keys(current).length === 0) return merged;
	const out: SeenMap = {};
	for (const id of Object.keys(current)) out[id] = merged[id];
	return out;
}

/** Before hydration the store is unknown: counting then would flash a badge
 *  on every page load and take it away again. */
export function tabBadges(hydrated: boolean, current: SeenMap, seen: SeenMap | null): { unseen: number; changed: number } {
	if (!hydrated) return { unseen: 0, changed: 0 };
	const { added, updated } = changedIds(current, seen);
	return { unseen: added.length, changed: updated.length };
}
