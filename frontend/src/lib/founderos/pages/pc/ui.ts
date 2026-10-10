/**
 * Shared bits for the pages-comms ports (/os/comms, /os/social*, /os/content*):
 * FounderOS v1's PILL / PILL_ACCENT / chipClass (components/slab.tsx) over the
 * --bn-* tokens, and the relative-time labels those pages print. Square
 * corners, as the Monolith kit draws them.
 */
export const PILL =
	'pc-pill inline-flex items-center gap-1.5 border px-4 py-2 text-[13px] whitespace-nowrap';
export const PILL_ACCENT =
	'pc-pill-accent inline-flex items-center gap-2 border px-4 py-2 text-[13px] whitespace-nowrap';
export const chipClass = (active: boolean) =>
	`pc-filter inline-flex items-center px-4 py-1.5 text-[12.5px] ${active ? 'pc-filter-on font-semibold' : 'border'}`;

/** "now" · "5m" · "3h" · "2d" (the three-pane list and Slack cards). */
export function agoCompact(iso: string, nowMs: number): string {
	const ms = nowMs - Date.parse(iso);
	if (!Number.isFinite(ms)) return '';
	if (ms < 60_000) return 'now';
	const m = Math.floor(ms / 60_000);
	if (m < 60) return `${m}m`;
	const h = Math.floor(m / 60);
	if (h < 24) return `${h}h`;
	return `${Math.floor(h / 24)}d`;
}

/** "3m ago" · "5h ago" · "2d ago" (the morning report). */
export function agoLong(iso: string, nowMs: number): string {
	const ms = nowMs - Date.parse(iso);
	if (!Number.isFinite(ms)) return '';
	const m = Math.floor(ms / 60_000);
	if (m < 60) return `${Math.max(m, 1)}m ago`;
	const h = Math.floor(m / 60);
	return h < 24 ? `${h}h ago` : `${Math.floor(h / 24)}d ago`;
}

/** Rounded "2h"/"3d" from a published-at stamp (social + content posts). */
export function agoRounded(iso: string | null, nowMs: number): string {
	if (!iso) return '';
	const ms = nowMs - new Date(iso).getTime();
	if (!Number.isFinite(ms) || ms < 0) return '';
	const mins = Math.round(ms / 60_000);
	if (mins < 60) return `${Math.max(1, mins)}m`;
	const hrs = Math.round(mins / 60);
	if (hrs < 48) return `${hrs}h`;
	return `${Math.round(hrs / 24)}d`;
}
