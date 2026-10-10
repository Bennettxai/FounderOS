import { describe, expect, test } from 'vitest';
import { activityCounts, chartGeometry, filterActivity, pnl, sortVerdicts, timeAgo, usd } from './view';
import type { TradeActivity, TradingAccountSnapshot } from './types';

// Ported from FounderOS v1 tests/trading-chart.test.ts (chartGeometry) and the
// filterActivity half of tests/trading-view.test.ts. freshness, positionSizes,
// tradingVolume and agentSummary moved to the backend (Go tests).

const snap = (capturedAt: string, accountValueUsd: number): TradingAccountSnapshot => ({
	capturedAt,
	accountId: 'agentic',
	accountLabel: 'Agentic',
	accountValueUsd,
	buyingPowerUsd: 0,
	cashUsd: 0,
	dayPnlUsd: 0,
	totalPnlUsd: 0,
	source: 'robinhood'
});
const trade = (at: string, symbol = 'SPY'): TradeActivity => ({
	id: `t-${at}`,
	at,
	accountId: 'agentic',
	agent: 'Markets Agent',
	action: 'buy',
	symbol,
	quantity: 1,
	priceUsd: 100,
	rationale: '',
	status: 'filled'
});
const SERIES = [snap('2026-08-13T11:00:00.000Z', 600), snap('2026-08-13T12:00:00.000Z', 640), snap('2026-08-13T13:00:00.000Z', 620)];

describe('chartGeometry', () => {
	test('returns null when there is nothing to plot', () => {
		expect(chartGeometry([], [], { w: 100, h: 40 })).toBeNull();
		expect(chartGeometry([snap('2026-08-13T11:00:00.000Z', 600)], [], { w: 100, h: 40 })).toBeNull();
	});
	test('spans the full width oldest-left, newest-right', () => {
		const g = chartGeometry(SERIES, [], { w: 300, h: 100 })!;
		expect(g.points).toHaveLength(3);
		expect(g.points[0].x).toBe(0);
		expect(g.points[2].x).toBe(300);
		expect(g.points[1].y).toBeLessThan(g.points[0].y);
		expect(g.points[1].y).toBeLessThan(g.points[2].y);
	});
	test('keeps every point inside the box, including a dead-flat series', () => {
		const g = chartGeometry([snap('2026-08-13T11:00:00.000Z', 600), snap('2026-08-13T12:00:00.000Z', 600)], [], { w: 200, h: 50 })!;
		for (const p of g.points) {
			expect(Number.isFinite(p.y)).toBe(true);
			expect(p.y).toBeGreaterThanOrEqual(0);
			expect(p.y).toBeLessThanOrEqual(50);
		}
	});
	test('summarizes the move across the window', () => {
		const g = chartGeometry(SERIES, [], { w: 300, h: 100 })!;
		expect(g.firstUsd).toBe(600);
		expect(g.lastUsd).toBe(620);
		expect(g.changeUsd).toBe(20);
		expect(g.changePct).toBeCloseTo(3.33, 2);
	});
	test('pins each trade to the closest point on the line', () => {
		const g = chartGeometry(SERIES, [trade('2026-08-13T12:10:00.000Z')], { w: 300, h: 100 })!;
		expect(g.markers).toHaveLength(1);
		expect(g.markers[0].x).toBe(g.points[1].x);
		expect(g.markers[0].trade.symbol).toBe('SPY');
	});
	test('ignores trades outside the plotted window', () => {
		expect(chartGeometry(SERIES, [trade('2020-01-01T00:00:00.000Z')], { w: 300, h: 100 })!.markers).toHaveLength(0);
	});
});

describe('filterActivity', () => {
	const act = (over: Partial<TradeActivity>): TradeActivity => ({ ...trade('2026-09-18T14:00:00.000Z'), ...over });
	const rows = [
		act({ id: '1', agent: 'Markets Agent' }),
		act({ id: '2', agent: 'the operator (manual)', accountId: 'individual' }),
		act({ id: '3', agent: 'Markets Agent', status: 'rejected' }),
		act({ id: '4', agent: 'the operator (override)', status: 'pending' })
	];
	test('all keeps every row', () => expect(filterActivity(rows, 'all').map((r) => r.id)).toEqual(['1', '2', '3', '4']));
	test('agent is what the Markets Agent did, whatever the outcome', () => expect(filterActivity(rows, 'agent').map((r) => r.id)).toEqual(['1', '3']));
	test('you is every row a human placed, manual or override', () => expect(filterActivity(rows, 'you').map((r) => r.id)).toEqual(['2', '4']));
	test('rejected isolates the orders that never reached a fill', () => expect(filterActivity(rows, 'rejected').map((r) => r.id)).toEqual(['3']));
	test('the chip counts come from the same rules', () => expect(activityCounts(rows)).toEqual({ all: 4, agent: 2, you: 2, rejected: 1 }));
});

describe('formatting', () => {
	test('money, signed P&L with a status tone, and short ages', () => {
		expect(usd(1234.5)).toBe('$1,234.50');
		expect(pnl(3.25)).toEqual({ text: '+$3.25', tone: 'ok' });
		expect(pnl(-1)).toEqual({ text: '-$1.00', tone: 'err' });
		expect(pnl(0)).toEqual({ text: '$0.00', tone: 'muted' });
		const now = Date.parse('2026-09-18T15:00:00.000Z');
		expect(timeAgo('2026-09-18T14:30:00.000Z', now)).toBe('30m');
		expect(timeAgo('2026-09-18T09:00:00.000Z', now)).toBe('6h');
		expect(timeAgo('2026-09-11T15:00:00.000Z', now)).toBe('7d');
	});
	test('verdict chips follow the strategy order, anything new after it alphabetically', () => {
		expect(sortVerdicts(['watch', 'zeta', 'dropped', 'alpha', 'signal'])).toEqual(['signal', 'watch', 'dropped', 'alpha', 'zeta']);
	});
});
