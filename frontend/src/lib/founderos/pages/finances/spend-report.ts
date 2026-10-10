/**
 * Month-to-month expenditure math for /finances (port of FounderOS v1
 * lib/spend-report.ts, lib/cards.ts lanes and bank-statements rangeTotalCents).
 * Pure and client-side: the page hands down the whole ledger once and every
 * month step, pie and statement is recomputed here with no round trip.
 */

export type CardId = 'gold' | 'platinum' | 'blue';
export type CardLane = { id: CardId; label: string; blurb: string };

export const CARD_LANES: CardLane[] = [
	{ id: 'gold', label: 'Gold · Personal', blurb: 'Personal-life spend' },
	{ id: 'platinum', label: 'Platinum · Business', blurb: 'Business general + cohort programmes' },
	{ id: 'blue', label: 'Business Blue · Vantage', blurb: 'Second-entity (Vantage) spend' }
];
export const DEFAULT_CARD: CardId = 'platinum';

export function cardLabel(id: string): string {
	return CARD_LANES.find((c) => c.id === id)?.label ?? id;
}

export type SpendRow = {
	date: string;
	description: string;
	amountCents: number;
	direction: 'in' | 'out';
	category: string;
	card: CardId;
};

export type MonthTotal = { month: string; totalCents: number; byCard: Record<CardId, number> };

export type Subscription = {
	merchant: string;
	card: CardId;
	monthlyCents: number;
	latestCents: number;
	months: string[];
	firstMonth: string;
	lastMonth: string;
	charges: number;
	status: 'active' | 'cancelled';
	category: string;
};

const monthOf = (date: string): string => date.slice(0, 7);

export const monthName = (month: string): string =>
	new Date(`${month}-01T00:00:00Z`).toLocaleDateString('en-US', { month: 'short', year: 'numeric', timeZone: 'UTC' });

export const usd = (cents: number, decimals = false): string =>
	(cents / 100).toLocaleString('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: decimals ? 2 : 0 });

export function prevMonth(month: string): string {
	const [y, m] = month.split('-').map(Number);
	return m === 1 ? `${y - 1}-12` : `${y}-${String(m - 1).padStart(2, '0')}`;
}

/** A stable merchant key: strip Amex `*` refs, order/store numbers, keep the leading words. */
export function normalizeMerchant(description: string): string {
	const raw = description.toUpperCase().trim();
	let s = raw.split('*')[0];
	s = s.replace(/[^A-Z0-9.& ]+/g, ' ');
	s = s.replace(/\b\d[\d.]*\b/g, ' ');
	s = s.replace(/\s{2,}/g, ' ').trim();
	const tokens = s.split(' ').filter(Boolean).slice(0, 3);
	return tokens.length > 0 ? tokens.join(' ') : raw.replace(/\s+/g, ' ').trim() || description;
}

const zeroByCard = (): Record<CardId, number> => ({ gold: 0, platinum: 0, blue: 0 });

export function monthlyTotals(rows: SpendRow[]): MonthTotal[] {
	const byMonth = new Map<string, MonthTotal>();
	for (const r of rows) {
		if (r.direction !== 'out') continue;
		const month = monthOf(r.date);
		const entry = byMonth.get(month) ?? { month, totalCents: 0, byCard: zeroByCard() };
		entry.totalCents += r.amountCents;
		entry.byCard[r.card] = (entry.byCard[r.card] ?? 0) + r.amountCents;
		byMonth.set(month, entry);
	}
	return [...byMonth.values()].sort((a, b) => a.month.localeCompare(b.month));
}

const median = (ns: number[]): number => {
	const s = [...ns].sort((a, b) => a - b);
	return s.length % 2 ? s[(s.length - 1) / 2] : Math.round((s[s.length / 2 - 1] + s[s.length / 2]) / 2);
};

/** Recurring merchants: two or more months on one card, every charge within 25% of the median. */
export function detectSubscriptions(rows: SpendRow[], latestMonth: string): Subscription[] {
	const groups = new Map<string, SpendRow[]>();
	for (const r of rows) {
		if (r.direction !== 'out') continue;
		const key = `${r.card}|${normalizeMerchant(r.description)}`;
		groups.set(key, [...(groups.get(key) ?? []), r]);
	}
	const grace = prevMonth(latestMonth);
	const subs: Subscription[] = [];
	for (const [key, group] of groups) {
		const months = [...new Set(group.map((r) => monthOf(r.date)))].sort();
		if (months.length < 2) continue;
		const amounts = group.map((r) => r.amountCents);
		const typical = median(amounts);
		if (typical <= 0) continue;
		if (amounts.some((a) => Math.abs(a - typical) > typical * 0.25)) continue;
		const sorted = [...group].sort((a, b) => a.date.localeCompare(b.date));
		const lastMonth = months[months.length - 1];
		subs.push({
			merchant: key.slice(key.indexOf('|') + 1),
			card: group[0].card,
			monthlyCents: typical,
			latestCents: sorted[sorted.length - 1].amountCents,
			months,
			firstMonth: months[0],
			lastMonth,
			charges: group.length,
			status: lastMonth >= grace ? 'active' : 'cancelled',
			category: sorted[sorted.length - 1].category
		});
	}
	return subs.sort((a, b) => {
		if (a.status !== b.status) return a.status === 'active' ? -1 : 1;
		return b.monthlyCents - a.monthlyCents || a.merchant.localeCompare(b.merchant);
	});
}

export function topMerchants(rows: SpendRow[], month: string | null, limit = 25) {
	const totals = new Map<string, { merchant: string; amountCents: number; count: number; card: CardId; category: string }>();
	for (const r of rows) {
		if (r.direction !== 'out') continue;
		if (month && monthOf(r.date) !== month) continue;
		const merchant = normalizeMerchant(r.description);
		const entry = totals.get(merchant) ?? { merchant, amountCents: 0, count: 0, card: r.card, category: r.category };
		entry.amountCents += r.amountCents;
		entry.count += 1;
		totals.set(merchant, entry);
	}
	return [...totals.values()].sort((a, b) => b.amountCents - a.amountCents).slice(0, limit);
}

export function spendTotalCents(rows: SpendRow[], month: string | null): number {
	return rows.reduce((sum, r) => (r.direction === 'out' && (!month || monthOf(r.date) === month) ? sum + r.amountCents : sum), 0);
}

export function categoryTotals(rows: SpendRow[], month: string | null): { category: string; totalCents: number }[] {
	const totals = new Map<string, number>();
	for (const r of rows) {
		if (r.direction !== 'out') continue;
		if (month && monthOf(r.date) !== month) continue;
		totals.set(r.category, (totals.get(r.category) ?? 0) + r.amountCents);
	}
	return [...totals.entries()]
		.map(([category, totalCents]) => ({ category, totalCents }))
		.sort((a, b) => b.totalCents - a.totalCents || a.category.localeCompare(b.category));
}

/** Every lane is reported, a silent one as $0 (a missing statement reads as $0, not an absent lane). */
export function cardTotals(rows: SpendRow[], month: string | null): { card: CardId; totalCents: number }[] {
	const totals = new Map<CardId, number>(CARD_LANES.map((c) => [c.id, 0]));
	for (const r of rows) {
		if (r.direction !== 'out') continue;
		if (month && monthOf(r.date) !== month) continue;
		totals.set(r.card, (totals.get(r.card) ?? 0) + r.amountCents);
	}
	return CARD_LANES.map((c) => ({ card: c.id, totalCents: totals.get(c.id) ?? 0 }));
}

/** Where the expenses view points after the ledger changes: a month that just arrived wins. */
export function monthAfterRefresh(current: string | null, previousMonths: string[], months: string[]): string | null {
	const seen = new Set(previousMonths);
	const arrived = months.filter((m) => !seen.has(m));
	if (arrived.length > 0) return arrived[arrived.length - 1];
	if (current && months.includes(current)) return current;
	return months.length > 0 ? months[months.length - 1] : null;
}

// ── bank statement months (lib/bank-statements.ts) ──────────────────────
export type MonthPoint = { month: string; creditsCents: number; debitsCents: number; netCents: number };
export type BusinessSeries = { business: string; months: MonthPoint[] };
export type IncomeRange = 1 | 2 | 3 | 6 | 12 | 'all';

export function rangeTotalCents(months: MonthPoint[], range: IncomeRange): number {
	const slice = range === 'all' ? months : months.slice(-range);
	return slice.reduce((s, m) => s + m.creditsCents, 0);
}

// ── share pie (lib/social-chart.ts pieSlices) ───────────────────────────
export type PieItem = { key: string; label: string; value: number | null };
export type PieSlice = { key: string; label: string; value: number; share: number; startAngle: number; endAngle: number };

export function pieSlices(items: PieItem[]): PieSlice[] {
	const live = items.filter((i): i is PieItem & { value: number } => i.value != null && i.value > 0);
	const total = live.reduce((s, i) => s + i.value, 0);
	if (total <= 0) return [];
	let angle = -90;
	return live.map((i) => {
		const share = i.value / total;
		const startAngle = angle;
		angle += share * 360;
		return { key: i.key, label: i.label, value: i.value, share, startAngle, endAngle: angle };
	});
}

/** A money band ("$0 – $92,989", "– $104,017") glued so a narrow card never
 *  splits it across lines: no-break spaces, plus a word joiner after the dash,
 *  since an en dash is itself a break-after character (UAX #14 BA). */
export function keepRange(s: string): string {
	return s.replace(/ – /g, '\u00a0–\u2060\u00a0').replace(/^– /, '–\u2060\u00a0');
}

// FounderOS v1 globals.css data ramp on the port's tokens: --ramp-1 is brain-2,
// --ramp-4 brain-1 leaning to the accent (white and grey on the Monolith).
export const RAMP_1 = 'var(--bn-brain-2)';
export const RAMP_4 = 'color-mix(in oklab, var(--bn-brain-1) 70%, var(--bn-accent))';
