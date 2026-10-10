// GET /api/founderos/pages/finances (internal/founderos/pages/finances Payload).
import type { Meter, SeriesPoint, StatChip } from '$lib/founderos/kit';
import type { BusinessSeries, CardLane, SpendRow } from './spend-report';

export type IncomeAccount = {
	id: string;
	processor: string;
	label: string;
	configured: boolean;
	live: boolean;
	/** month-to-date USD, the proven floor; null = pending, never a fake $0 */
	income: number | null;
	incomeUpper: number | null;
	unsplittableCustomers: number;
};

export type RecentCharge = { amount: number; currency: string; description: string; created: number };
export type WiseTransfer = { amountCents: number; currency: string; status: string; created: string | number; reference?: string | null };

export type FinancesIncome = {
	stripe: { keyed: boolean; live: boolean; mtdUsd: number | null; availableUsd: number; pendingUsd: number; recentCharges: RecentCharge[] };
	processors: { id: string; name: string; configured: boolean }[] | null;
	accounts: IncomeAccount[];
	totalUsd: number;
	totalUpperUsd: number;
	hasUnsplittable: boolean;
	liveCount: number;
	/** null hides the section (no Wise token) */
	wiseOutgoing: WiseTransfer[] | null;
	wiseError?: string;
};

export type MoneyVolume = {
	headline: number;
	upper: number | null;
	chips: StatChip[];
	caption: string;
	meters: Meter[];
	foot: string;
	insight: { display: string; headline: string; body: string; frac: number | null };
};

export type FinancesPayload = {
	generatedAt: string;
	thisMonth: string;
	income: FinancesIncome;
	incomeError?: string;
	ledger: { rows: SpendRow[]; months: string[]; latestMonth: string | null; byCategory: { category: string; total: number }[]; error?: string };
	bank: { series: BusinessSeries[]; error?: string };
	cardLanes: CardLane[];
	fallback: { category: string; totalCents: number }[];
	expenses: number;
	expensesLive: boolean;
	monthLabel: string | null;
	statementIsThisMonth: boolean;
	netComparable: boolean;
	netMonthly: number;
	volume: MoneyVolume;
	spend: SeriesPoint[];
	chargeSizes: SeriesPoint[];
	largestChargeUsd: number | null;
};

export type StatementUploadResult = {
	inserted: number;
	parsed: number;
	card: string;
	uploadedMonths: string[];
	months: string[];
};

export type BankUploadResult = { summary: { business: string; month: string; creditsCents: number } };
