/**
 * The left sidebar's shape as pure functions (FounderOS v1 lib/sidebar-layout.ts;
 * Sidebar.svelte is the DOM half). Three shapes:
 *
 *   expanded: docked, resizable between MIN_W and MAX_W by its right edge
 *   rail:     the docked icon rail (the toggle button)
 *   hidden:   gone completely, 0px, and the page takes the whole viewport
 *
 * Dragging the right edge left past HIDE_AT hides it. While hidden, moving the
 * mouse into the HOT_ZONE strip at the far-left edge reveals it as an overlay
 * that goes away once the mouse is PEEK_EXIT_SLOP clear of it, and dragging
 * out from that edge docks it again. ⌘\ / Ctrl+\ hides or docks it.
 */
import { EXPANDED_W, RAIL_KEY, RAIL_W } from './chrome';

export const SIDEBAR = {
	DEFAULT_W: EXPANDED_W,
	MIN_W: 190,
	MAX_W: 420,
	RAIL_W,
	/** Drag the edge left of this and the sidebar hides completely. */
	HIDE_AT: 120,
	/** The far-left strip that reveals a hidden sidebar. */
	HOT_ZONE: 8,
	/** How far past the overlay's edge the mouse may drift before it hides. */
	PEEK_EXIT_SLOP: 24
} as const;

export const SIDEBAR_KEYS = {
	width: 'founderos-sidebar-w',
	rail: RAIL_KEY,
	hidden: 'founderos-sidebar-hidden'
} as const;

export type SidebarLayout = { hidden: boolean; rail: boolean; width: number };

export function clampWidth(w: number): number {
	if (!Number.isFinite(w)) return SIDEBAR.DEFAULT_W;
	return Math.min(SIDEBAR.MAX_W, Math.max(SIDEBAR.MIN_W, Math.round(w)));
}

/** Where a drag of the right edge to `clientX` leaves the sidebar. */
export function dragTo(clientX: number): { hidden: true; width: null } | { hidden: false; width: number } {
	if (clientX < SIDEBAR.HIDE_AT) return { hidden: true, width: null };
	return { hidden: false, width: clampWidth(clientX) };
}

/** How far the page sits from the left edge. A peeking overlay floats over the
 *  page, so hidden is 0 whether or not it is showing. */
export function shellOffset(l: SidebarLayout): number {
	if (l.hidden) return 0;
	return l.rail ? SIDEBAR.RAIL_W : l.width;
}

export function inHotZone(clientX: number): boolean {
	return clientX >= 0 && clientX <= SIDEBAR.HOT_ZONE;
}

export function peekShouldEnd(clientX: number, overlayWidth: number): boolean {
	return clientX > overlayWidth + SIDEBAR.PEEK_EXIT_SLOP;
}

/** The saved shape, or the defaults for anything missing, malformed or
 *  unreadable (private window, blocked storage). Never throws. */
export function readStoredLayout(get: (key: string) => string | null): SidebarLayout {
	const fallback: SidebarLayout = { hidden: false, rail: false, width: SIDEBAR.DEFAULT_W };
	try {
		const w = Number(get(SIDEBAR_KEYS.width));
		const width = Number.isFinite(w) && w >= SIDEBAR.MIN_W && w <= SIDEBAR.MAX_W ? w : SIDEBAR.DEFAULT_W;
		return { hidden: get(SIDEBAR_KEYS.hidden) === '1', rail: get(SIDEBAR_KEYS.rail) === '1', width };
	} catch {
		return fallback;
	}
}
