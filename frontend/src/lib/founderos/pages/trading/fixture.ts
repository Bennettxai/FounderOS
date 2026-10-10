import type { TradingPayload } from './types';

/** A fed board: two accounts, a wallet, one live order, an analysis, a log. */
export function tradingFixture(over: Partial<TradingPayload> = {}): TradingPayload {
	const agentic = {
		capturedAt: '2026-09-18T14:30:00.000Z',
		accountId: 'agentic',
		accountLabel: 'Agentic',
		accountValueUsd: 600,
		buyingPowerUsd: 480,
		cashUsd: 480,
		dayPnlUsd: 4.5,
		totalPnlUsd: 0,
		source: 'robinhood'
	};
	const individual = { ...agentic, accountId: 'individual', accountLabel: 'Individual', accountValueUsd: 1200, dayPnlUsd: -1.25, cashUsd: 100 };
	return {
		accounts: [individual, agentic],
		history: {
			agentic: [
				{ ...agentic, capturedAt: '2026-09-18T12:00:00.000Z', accountValueUsd: 590 },
				agentic
			],
			individual: [individual]
		},
		snapshot: individual,
		positions: [
			{ capturedAt: agentic.capturedAt, accountId: 'agentic', symbol: 'QQQ', quantity: 0.2, avgCostUsd: 500, marketValueUsd: 120, unrealizedPnlUsd: 2 },
			{ capturedAt: individual.capturedAt, accountId: 'individual', symbol: 'NVDA', quantity: 5, avgCostUsd: 150, marketValueUsd: 900, unrealizedPnlUsd: -12 }
		],
		activity: [
			{ id: 't1', at: '2026-09-18T13:00:00.000Z', accountId: 'agentic', agent: 'Markets Agent', action: 'buy', symbol: 'QQQ', quantity: 0.2, priceUsd: 500, rationale: 'Dip under the 20-day band; index core top-up.', status: 'filled' },
			{ id: 't2', at: '2026-09-18T12:00:00.000Z', accountId: 'individual', agent: 'the operator (manual)', action: 'sell', symbol: 'TSLA', quantity: 1, priceUsd: 250, rationale: '', status: 'rejected' }
		],
		analysis: {
			id: 'a1',
			at: '2026-09-18T14:00:00.000Z',
			accountId: 'agentic',
			agent: 'Markets Agent',
			examined: 8,
			signals: 1,
			notes: 'QQQ pulled back; SPY flat.',
			rows: [
				{ ticker: 'QQQ', score: 0.8, verdict: 'signal', reason: 'below band' },
				{ ticker: 'SPY', score: null, verdict: 'watch', reason: 'flat' }
			]
		},
		openOrders: [
			{ id: 'o1', accountId: 'agentic', symbol: 'SPY', side: 'buy', type: 'market', state: 'queued', quantity: 0, filledQuantity: 0, dollarAmountUsd: 25, limitPriceUsd: null, placedAgent: 'agentic', createdAt: '2026-09-18T14:40:00.000Z' },
			{ id: 'o2', accountId: 'individual', symbol: 'AAPL', side: 'sell', type: 'limit', state: 'confirmed', quantity: 2, filledQuantity: 0, dollarAmountUsd: null, limitPriceUsd: 250, placedAgent: '', createdAt: '2026-09-18T14:10:00.000Z' }
		],
		status: { id: 'robinhood', name: 'Robinhood', state: 'connected', detail: '2 accounts · updated 30 min ago' },
		source: 'robinhood',
		phantom: { address: '68SHabcdefghBST2', sol: 2, usdPerSol: 100, usdValue: 200, fetchedAt: '2026-09-18T15:00:00.000Z' },
		phantomStatus: { state: 'connected', detail: '68SH…BST2 · 2 SOL' },
		limits: {
			limits: { maxNotionalPerTradeUsd: 150, maxPositionPctOfSleeve: 40, maxRiskPctPerTrade: 2.5, maxConcurrentPositions: 6, maxTradesPerDay: 12, minSleeveValueUsd: 480, maxDeployedCapitalUsd: 560, autopilot: false },
			clamped: [],
			source: 'default',
			updatedAt: null
		},
		view: {
			freshness: { state: 'live', label: 'synced 30m ago' },
			volume: {
				headline: 1800,
				chips: [
					{ tone: 'ok', text: '+$3.25 today' },
					{ tone: 'accent', text: '$1,020 in 2 positions' }
				],
				caption: 'brokerage, 2 accounts · $2,000.00 with the wallet',
				meters: [
					{ label: 'Individual · read-only to agents', frac: 0.6, display: '$1,200.00', hue: 'var(--bn-text)' },
					{ label: 'Agentic · agent may trade', frac: 0.3, display: '$600.00', hue: 'var(--bn-accent)' },
					{ label: 'Phantom · 2 SOL · 68SH…BST2', frac: 0.1, display: '$200.00', hue: 'var(--bn-text)' }
				],
				foot: 'only the agentic sleeve can be traded by an agent · $100.00 / SOL'
			},
			sizes: [
				{ label: '<$50', count: 0 },
				{ label: '$50-250', count: 1 },
				{ label: '$250-1k', count: 1 },
				{ label: '$1k+', count: 0 }
			],
			agent: { hasActed: true, tradeCount: 1, lastTrade: null, lastTradeAt: '2026-09-18T13:00:00.000Z', deployedUsd: 120, idleCashUsd: 480, unrealizedPnlUsd: 2 }
		},
		at: '2026-09-18T15:00:00.000Z',
		...over
	};
}
