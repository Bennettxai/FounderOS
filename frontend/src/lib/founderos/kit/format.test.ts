import { describe, expect, it, vi } from 'vitest';
import { countFrame, dotState, formatCount, insightTicks, meterWidth, prefersReducedMotion, sparkPoints } from './format';

describe('dotState (FounderOS v1 terminal.tsx DOT_FOR)', () => {
	it('maps connector/agent states onto the four LED states', () => {
		expect(dotState('connected')).toBe('ok');
		expect(dotState('active')).toBe('ok');
		expect(dotState('available')).toBe('warn');
		expect(dotState('idle')).toBe('warn');
		expect(dotState('error')).toBe('err');
		expect(dotState('fail')).toBe('err');
		expect(dotState('not_configured')).toBe('off');
		expect(dotState('planned')).toBe('off');
	});
	it('an unknown state is off, never ok', () => {
		expect(dotState('mystery')).toBe('off');
		expect(dotState('')).toBe('off');
	});
});

describe('count-up', () => {
	it('countFrame eases from where the number is to the target and lands exactly', () => {
		expect(countFrame(1000, 1100, 0)).toBe(1000);
		expect(countFrame(1000, 1100, 1)).toBe(1100);
		const mid = countFrame(1000, 1100, 0.5);
		expect(mid).toBeGreaterThan(1000);
		expect(mid).toBeLessThan(1100);
		expect(countFrame(50, 20, 1)).toBe(20);
		expect(countFrame(0, 12.34, 1)).toBe(12.34);
	});
	it('formats land on the text the pages print', () => {
		expect(formatCount('tokens', 68_219_000)).toBe('68.2M');
		expect(formatCount('tokens', 1_500)).toBe('1.5k');
		expect(formatCount('tokens', 2_100_000_000)).toBe('2.1B');
		expect(formatCount('tokens', 12)).toBe('12');
		expect(formatCount('pct', 56)).toBe('56%');
		expect(formatCount('int', 47501)).toBe('47,501');
		expect(formatCount('usd', 1200)).toBe('$1,200');
		expect(formatCount('usdCents', 12.5)).toBe('$12.50');
		expect(formatCount('followers', 9876.4)).toBe('9,876');
	});
	it('reduced motion is read from the media query', () => {
		const spy = vi.spyOn(window, 'matchMedia').mockReturnValue({ matches: true } as MediaQueryList);
		expect(prefersReducedMotion()).toBe(true);
		spy.mockReturnValue({ matches: false } as MediaQueryList);
		expect(prefersReducedMotion()).toBe(false);
		spy.mockRestore();
	});
});

describe('insightTicks', () => {
	it('lights a real fraction of 8 and never overflows', () => {
		expect(insightTicks(0)).toBe(0);
		expect(insightTicks(0.5)).toBe(4);
		expect(insightTicks(3)).toBe(8);
		expect(insightTicks(Number.NaN)).toBe(0);
		expect(insightTicks(null)).toBe(0);
	});
});

describe('meterWidth: honest meters', () => {
	it('a known fraction is clamped to [2%, 100%] so a real zero is still a mark', () => {
		expect(meterWidth(0.5)).toBe(0.5);
		expect(meterWidth(0)).toBe(0.02);
		expect(meterWidth(0.001)).toBe(0.02);
		expect(meterWidth(7)).toBe(1);
	});
	it('unknown (null, undefined, NaN, ±Infinity) is null: no bar at all, not a zero bar', () => {
		expect(meterWidth(null)).toBeNull();
		expect(meterWidth(undefined)).toBeNull();
		expect(meterWidth(Number.NaN)).toBeNull();
		expect(meterWidth(Number.POSITIVE_INFINITY)).toBeNull();
	});
});

describe('sparkPoints (Spark geometry from terminal.tsx)', () => {
	it('fewer than two samples draw nothing', () => {
		expect(sparkPoints([], 72, 22)).toBeNull();
		expect(sparkPoints([3], 72, 22)).toBeNull();
	});
	it('spans the width, min at the floor and max at the top', () => {
		const pts = sparkPoints([0, 10], 72, 22)!;
		expect(pts).toEqual(['0.0,20.0', '72.0,3.0']);
	});
	it('a flat series does not divide by zero', () => {
		expect(sparkPoints([4, 4, 4], 72, 22)).toEqual(['0.0,20.0', '36.0,20.0', '72.0,20.0']);
	});
});
