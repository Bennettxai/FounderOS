// FounderOS v1 tests/sidebar-layout.test.ts. The operator, 2026-09-23: "allow me to
// collapse the left sidebar with my mouse completely and expand without
// pressing the button". Three shapes: expanded (resizable), the icon rail, and
// hidden (0px; the far-left edge reveals it as an overlay, dragging out docks it).
import { describe, expect, it } from 'vitest';
import { EXPANDED_W, RAIL_KEY, RAIL_W } from './chrome';
import { SIDEBAR, SIDEBAR_KEYS, clampWidth, dragTo, inHotZone, peekShouldEnd, readStoredLayout, shellOffset } from './sidebar-layout';

describe('clampWidth', () => {
	it('keeps a docked sidebar between MIN_W and MAX_W', () => {
		expect(clampWidth(100)).toBe(SIDEBAR.MIN_W);
		expect(clampWidth(300)).toBe(300);
		expect(clampWidth(9000)).toBe(SIDEBAR.MAX_W);
		expect(clampWidth(Number.NaN)).toBe(SIDEBAR.DEFAULT_W);
	});

	it('shares the widths the chrome already uses', () => {
		expect(SIDEBAR.DEFAULT_W).toBe(EXPANDED_W);
		expect(SIDEBAR.RAIL_W).toBe(RAIL_W);
	});
});

describe('dragTo', () => {
	it('past the hide threshold the sidebar goes away completely', () => {
		expect(dragTo(SIDEBAR.HIDE_AT - 1)).toEqual({ hidden: true, width: null });
		expect(dragTo(0)).toEqual({ hidden: true, width: null });
	});

	it('above it, the sidebar is docked at the clamped width', () => {
		expect(dragTo(SIDEBAR.HIDE_AT)).toEqual({ hidden: false, width: SIDEBAR.MIN_W });
		expect(dragTo(300)).toEqual({ hidden: false, width: 300 });
		expect(dragTo(2000)).toEqual({ hidden: false, width: SIDEBAR.MAX_W });
	});

	it('the threshold sits below the minimum width, so a resize never hides by accident', () => {
		expect(SIDEBAR.HIDE_AT).toBeLessThan(SIDEBAR.MIN_W);
	});
});

describe('shellOffset', () => {
	it('hidden gives the page the whole viewport, even while the overlay peeks', () => {
		expect(shellOffset({ hidden: true, rail: false, width: 300 })).toBe(0);
	});
	it('the rail and the docked width', () => {
		expect(shellOffset({ hidden: false, rail: true, width: 300 })).toBe(SIDEBAR.RAIL_W);
		expect(shellOffset({ hidden: false, rail: false, width: 300 })).toBe(300);
	});
});

describe('the far-left hot zone', () => {
	it('is a thin strip at the screen edge', () => {
		expect(SIDEBAR.HOT_ZONE).toBeGreaterThanOrEqual(6);
		expect(SIDEBAR.HOT_ZONE).toBeLessThanOrEqual(8);
		expect(inHotZone(0)).toBe(true);
		expect(inHotZone(SIDEBAR.HOT_ZONE)).toBe(true);
		expect(inHotZone(SIDEBAR.HOT_ZONE + 1)).toBe(false);
		expect(inHotZone(-5)).toBe(false);
	});

	it('the overlay hides once the mouse is clear of it, not the instant it crosses the border', () => {
		expect(peekShouldEnd(250, 232)).toBe(false);
		expect(peekShouldEnd(232 + SIDEBAR.PEEK_EXIT_SLOP + 1, 232)).toBe(true);
		expect(peekShouldEnd(100, 232)).toBe(false);
	});
});

describe('readStoredLayout', () => {
	it('defaults to expanded at the default width', () => {
		expect(readStoredLayout(() => null)).toEqual({ hidden: false, rail: false, width: SIDEBAR.DEFAULT_W });
	});

	it('reads the saved shape; the rail keeps its existing key', () => {
		expect(SIDEBAR_KEYS.rail).toBe(RAIL_KEY);
		const store: Record<string, string> = { [SIDEBAR_KEYS.width]: '300', [SIDEBAR_KEYS.rail]: '1', [SIDEBAR_KEYS.hidden]: '0' };
		expect(readStoredLayout((k) => store[k] ?? null)).toEqual({ hidden: false, rail: true, width: 300 });
		expect(readStoredLayout((k) => ({ ...store, [SIDEBAR_KEYS.hidden]: '1' })[k] ?? null).hidden).toBe(true);
	});

	it('garbage and a throwing storage fall back to the defaults', () => {
		expect(readStoredLayout((k) => (k === SIDEBAR_KEYS.width ? 'wide' : null)).width).toBe(SIDEBAR.DEFAULT_W);
		expect(readStoredLayout((k) => (k === SIDEBAR_KEYS.width ? '5000' : null)).width).toBe(SIDEBAR.DEFAULT_W);
		expect(
			readStoredLayout(() => {
				throw new Error('SecurityError');
			})
		).toEqual({ hidden: false, rail: false, width: SIDEBAR.DEFAULT_W });
	});
});
