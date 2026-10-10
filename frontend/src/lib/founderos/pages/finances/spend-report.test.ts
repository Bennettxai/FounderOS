import { describe, expect, it } from 'vitest';
import {
	CARD_LANES,
	cardLabel,
	cardTotals,
	categoryTotals,
	detectSubscriptions,
	keepRange,
	monthAfterRefresh,
	monthlyTotals,
	normalizeMerchant,
	prevMonth,
	rangeTotalCents,
	spendTotalCents,
	topMerchants,
	type SpendRow
} from './spend-report';

// Port of FounderOS v1 tests/spend-report.test.ts (+ rangeTotalCents from
// tests/bank-statements.test.ts): the client-side math behind the month
// stepper, the expenditure statement and the business income cards.

const out = (date: string, description: string, dollars: number, card: SpendRow['card'] = 'platinum', category = 'Software'): SpendRow => ({
	date,
	description,
	amountCents: Math.round(dollars * 100),
	direction: 'out',
	category,
	card
});

describe('card lanes', () => {
	it('are the three cards, labelled', () => {
		expect(CARD_LANES.map((c) => c.id)).toEqual(['gold', 'platinum', 'blue']);
		expect(cardLabel('blue')).toBe('Business Blue · Vantage');
	});
});

describe('normalizeMerchant', () => {
	it('collapses the same merchant written with different reference noise', () => {
		expect(normalizeMerchant('AMAZON.COM*RT4G2  SEATTLE WA')).toBe(normalizeMerchant('AMAZON.COM*9K12P SEATTLE WA'));
	});
	it('strips store / order numbers so recurring charges group', () => {
		expect(normalizeMerchant('ANTHROPIC 8829174 SAN FRANCISCO')).toBe(normalizeMerchant('ANTHROPIC 4410023 SAN FRANCISCO'));
	});
	it('never returns an empty label', () => {
		expect(normalizeMerchant('####')).not.toBe('');
	});
});

describe('prevMonth', () => {
	it('rolls the year backwards over January', () => {
		expect(prevMonth('2026-01')).toBe('2025-12');
		expect(prevMonth('2026-08')).toBe('2026-07');
	});
});

describe('monthlyTotals', () => {
	it('sums out-rows per month ascending, split per card, ignoring income', () => {
		const rows: SpendRow[] = [
			out('2026-06-02', 'AWS', 100, 'platinum'),
			out('2026-06-20', 'VERCEL', 50, 'blue'),
			out('2026-07-04', 'AWS', 120, 'platinum'),
			{ ...out('2026-07-05', 'CLIENT PAYMENT', 5000), direction: 'in', category: 'Income' }
		];
		expect(monthlyTotals(rows)).toEqual([
			{ month: '2026-06', totalCents: 15000, byCard: { gold: 0, platinum: 10000, blue: 5000 } },
			{ month: '2026-07', totalCents: 12000, byCard: { gold: 0, platinum: 12000, blue: 0 } }
		]);
	});
});

describe('detectSubscriptions', () => {
	const rows: SpendRow[] = [
		out('2026-05-03', 'ANTHROPIC CLAUDE', 20),
		out('2026-06-03', 'ANTHROPIC CLAUDE', 20),
		out('2026-07-03', 'ANTHROPIC CLAUDE', 20),
		out('2026-04-11', 'HIGGSFIELD AI', 49),
		out('2026-05-11', 'HIGGSFIELD AI', 49),
		out('2026-07-14', 'DELTA AIR LINES', 612.4, 'platinum', 'Travel')
	];

	it('finds merchants charged in two or more months at a steady amount', () => {
		const subs = detectSubscriptions(rows, '2026-07');
		expect(subs.map((s) => s.merchant)).toContain(normalizeMerchant('ANTHROPIC CLAUDE'));
		expect(subs.map((s) => s.merchant)).not.toContain(normalizeMerchant('DELTA AIR LINES'));
	});

	it('marks a subscription cancelled once it stops charging', () => {
		const subs = detectSubscriptions(rows, '2026-07');
		const higgs = subs.find((s) => s.merchant === normalizeMerchant('HIGGSFIELD AI'));
		expect(higgs?.status).toBe('cancelled');
		expect(higgs?.lastMonth).toBe('2026-05');
		const anthropic = subs.find((s) => s.merchant === normalizeMerchant('ANTHROPIC CLAUDE'));
		expect(anthropic).toMatchObject({ status: 'active', monthlyCents: 2000, charges: 3, firstMonth: '2026-05' });
	});

	it('keeps the same merchant on two cards as two separate subscriptions', () => {
		const subs = detectSubscriptions(
			[out('2026-06-03', 'NOTION LABS', 10, 'platinum'), out('2026-07-03', 'NOTION LABS', 10, 'platinum'), out('2026-06-03', 'NOTION LABS', 40, 'blue'), out('2026-07-03', 'NOTION LABS', 40, 'blue')],
			'2026-07'
		);
		expect(subs).toHaveLength(2);
		expect(new Set(subs.map((s) => s.card))).toEqual(new Set(['platinum', 'blue']));
	});

	it('sorts active subscriptions first', () => {
		expect(detectSubscriptions(rows, '2026-07')[0].status).toBe('active');
	});
});

describe('topMerchants', () => {
	it('ranks a month’s spend by merchant', () => {
		const top = topMerchants([out('2026-07-01', 'AWS', 100), out('2026-07-09', 'AWS', 40), out('2026-07-02', 'VERCEL', 90), out('2026-06-02', 'AWS', 900)], '2026-07');
		expect(top[0]).toMatchObject({ merchant: 'AWS', amountCents: 14000, count: 2 });
		expect(top[1]).toMatchObject({ merchant: 'VERCEL', amountCents: 9000, count: 1 });
	});
});

describe('categoryTotals / cardTotals / spendTotalCents', () => {
	const rows: SpendRow[] = [
		out('2026-06-05', 'AWS', 50, 'platinum', 'Infrastructure'),
		out('2026-07-05', 'AWS', 60, 'platinum', 'Infrastructure'),
		out('2026-07-06', 'NOTION', 40, 'blue', 'Software'),
		{ date: '2026-07-07', description: 'CLIENT', amountCents: 900000, direction: 'in', category: 'Income', card: 'blue' }
	];
	it('scopes category totals to a month, income excluded', () => {
		expect(categoryTotals(rows, '2026-07')).toEqual([
			{ category: 'Infrastructure', totalCents: 6000 },
			{ category: 'Software', totalCents: 4000 }
		]);
		expect(categoryTotals(rows, null)[0]).toEqual({ category: 'Infrastructure', totalCents: 11000 });
	});
	it('reports every lane for a month, silent lanes included as zero', () => {
		expect(cardTotals(rows, '2026-06')).toEqual([
			{ card: 'gold', totalCents: 0 },
			{ card: 'platinum', totalCents: 5000 },
			{ card: 'blue', totalCents: 0 }
		]);
	});
	it('totals a month or all of it', () => {
		expect(spendTotalCents(rows, '2026-07')).toBe(10000);
		expect(spendTotalCents(rows, null)).toBe(15000);
	});
});

describe('monthAfterRefresh', () => {
	it('jumps to the month that just arrived', () => {
		expect(monthAfterRefresh('2026-06', ['2026-05', '2026-06'], ['2026-05', '2026-06', '2026-07'])).toBe('2026-07');
	});
	it('lands on a back-dated statement too', () => {
		expect(monthAfterRefresh('2026-07', ['2026-06', '2026-07'], ['2026-03', '2026-06', '2026-07'])).toBe('2026-03');
	});
	it('takes the newest of several months uploaded at once', () => {
		expect(monthAfterRefresh(null, [], ['2026-05', '2026-06', '2026-07'])).toBe('2026-07');
	});
	it('keeps the month he is reading when no new month arrived', () => {
		expect(monthAfterRefresh('2026-06', ['2026-06', '2026-07'], ['2026-06', '2026-07'])).toBe('2026-06');
	});
	it('falls back to the newest month when the current one is gone', () => {
		expect(monthAfterRefresh('2026-04', ['2026-04'], ['2026-06', '2026-07'])).toBe('2026-07');
	});
	it('is null on an empty ledger', () => {
		expect(monthAfterRefresh(null, [], [])).toBeNull();
	});
});

describe('rangeTotalCents', () => {
	const months = [1000, 2000, 3000, 4000].map((c, i) => ({ month: `2026-0${i + 1}`, creditsCents: c, debitsCents: 0, netCents: c }));
	it('sums income over the last N months; "all" sums everything', () => {
		expect(rangeTotalCents(months, 1)).toBe(4000);
		expect(rangeTotalCents(months, 3)).toBe(9000);
		expect(rangeTotalCents(months, 'all')).toBe(10000);
		expect(rangeTotalCents(months, 12)).toBe(10000);
	});
});

describe('keepRange', () => {
	// A no-break space alone is not enough: an en dash is a break-after
	// character (UAX #14 BA), so a word joiner glues the dash to what follows.
	it('glues a money band together so it never breaks across lines', () => {
		expect(keepRange('$0 – $92,989')).toBe('$0\u00a0–\u2060\u00a0$92,989');
		expect(keepRange('– $104,017')).toBe('–\u2060\u00a0$104,017');
		expect(keepRange('$9,528')).toBe('$9,528');
	});
});
