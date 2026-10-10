import { describe, expect, it } from 'vitest';
import { fmtShort, formatFollowers, formatPct, tileValue } from './format';

describe('analytics formatting', () => {
	it('unknown reads as an em dash, never 0', () => {
		expect(formatFollowers(null)).toBe('—');
		expect(formatPct(null)).toBe('—');
	});
	it('growth keeps its sign and precision', () => {
		expect(formatPct(2.5)).toBe('+2.50%');
		expect(formatPct(-12.34)).toBe('-12.3%');
	});
	it('tiles: $ for money, bare for followers, a small unit otherwise', () => {
		expect(tileValue(1234, 'usd')).toEqual({ main: '$1,234', small: '' });
		expect(tileValue(9000, 'followers')).toEqual({ main: '9,000', small: '' });
		expect(tileValue(12, 'leads')).toEqual({ main: '12', small: 'leads' });
	});
	it('short axis dates', () => {
		expect(fmtShort('2026-09-24')).toBe('sep 24');
	});
});
