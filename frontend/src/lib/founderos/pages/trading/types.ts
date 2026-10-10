/**
 * GET /api/founderos/pages/trading: FounderOS v1 TradingPayload (lib/trading-view.ts)
 * plus the wallet's status, the limits the agent runs under, and the view the
 * backend computes (internal/founderos/pages/trading). Unknown is null, never 0.
 */
import type { Meter, SeriesPoint, StatChip } from '$lib/founderos/kit';

export const AGENTIC_ID = 'agentic';

export type TradingAccountSnapshot = {
	capturedAt: string;
	accountId: string;
	accountLabel: string;
	accountValueUsd: number;
	buyingPowerUsd: number;
	cashUsd: number;
	dayPnlUsd: number;
	totalPnlUsd: number;
	source: string;
};

export type TradingPosition = {
	capturedAt: string;
	accountId: string;
	symbol: string;
	quantity: number;
	avgCostUsd: number;
	marketValueUsd: number;
	unrealizedPnlUsd: number;
};

export type TradeActivity = {
	id: string;
	at: string;
	accountId: string;
	agent: string;
	action: 'buy' | 'sell';
	symbol: string;
	quantity: number;
	priceUsd: number;
	rationale: string;
	status: 'filled' | 'pending' | 'cancelled' | 'rejected';
};

export type TradeAnalysisRow = { ticker: string; score: number | null; verdict: string; reason: string };

export type TradeAnalysis = {
	id: string;
	at: string;
	accountId: string;
	agent: string;
	examined: number;
	signals: number;
	notes: string;
	rows: TradeAnalysisRow[];
};

export type TradingOrder = {
	id: string;
	accountId: string;
	symbol: string;
	side: 'buy' | 'sell';
	type: string;
	state: string;
	quantity: number;
	filledQuantity: number;
	dollarAmountUsd: number | null;
	limitPriceUsd: number | null;
	placedAgent: string;
	createdAt: string;
};

export type PhantomView = { address: string; sol: number; usdPerSol: number | null; usdValue: number | null; fetchedAt: string };

export type TradingLimits = {
	maxNotionalPerTradeUsd: number;
	maxPositionPctOfSleeve: number;
	maxRiskPctPerTrade: number;
	maxConcurrentPositions: number;
	maxTradesPerDay: number;
	minSleeveValueUsd: number;
	maxDeployedCapitalUsd: number;
	autopilot: boolean;
};

export type Freshness = { state: 'none' | 'seeded' | 'live' | 'stale'; label: string };

export type AgentSummary = {
	hasActed: boolean;
	tradeCount: number;
	lastTrade: TradeActivity | null;
	lastTradeAt: string | null;
	deployedUsd: number;
	idleCashUsd: number;
	unrealizedPnlUsd: number;
};

export type TradingVolume = { headline: number | null; chips: StatChip[]; caption: string; meters: Meter[]; foot: string };

export type TradingPayload = {
	accounts: TradingAccountSnapshot[];
	history: Record<string, TradingAccountSnapshot[]>;
	snapshot: TradingAccountSnapshot | null;
	positions: TradingPosition[];
	activity: TradeActivity[];
	analysis: TradeAnalysis | null;
	openOrders: TradingOrder[];
	status: { id: string; name: string; state: string; detail: string };
	source: string | null;
	phantom: PhantomView | null;
	phantomStatus: { state: string; detail: string };
	limits: { limits: TradingLimits; clamped: string[]; source: 'stored' | 'default'; updatedAt: string | null };
	view: { freshness: Freshness; volume: TradingVolume; sizes: SeriesPoint[]; agent: AgentSummary };
	at: string;
};
