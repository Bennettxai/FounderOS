import { describe, expect, test } from 'vitest';
import { moneyVolume, chargeSizes, spendSeries } from '@/lib/finances-volume';
import type { IncomeAccount } from '@/lib/finances';
import type { SpendRow } from '@/lib/spend-report';

/**
 * /finances in the Brand Deals look (Alex, 2026-09-24: "the current metrics
 * displayed on each page should be displayed very similarly to how they are in
 * the Brand Deals tab"). The page's own numbers, shaped like Deal Volume: a
 * count-up headline, dot chips, a stack of meters that are honest fractions of
 * a real whole, and the one insight card. Nothing here invents money.
 */

const acct = (over: Partial<IncomeAccount>): IncomeAccount => ({
  id: 'stripe',
  processor: 'Stripe',
  label: 'Stripe · Launchpad Cohort',
  configured: true,
  live: true,
  income: 0,
  incomeUpper: null,
  unsplittableCustomers: 0,
  ...over,
});

const row = (date: string, dollars: number, over: Partial<SpendRow> = {}): SpendRow => ({
  date,
  description: 'x',
  amountCents: Math.round(dollars * 100),
  direction: 'out',
  category: 'Software',
  card: 'platinum',
  ...over,
});

describe('moneyVolume', () => {
  const accounts = [
    acct({ id: 'stripe', income: 6000 }),
    acct({ id: 'stripe-vantage', label: 'Stripe · Vantage', income: 2000 }),
    acct({ id: 'paykit-lc', processor: 'PayKit', label: 'PayKit · Launchpad Cohort', live: false, configured: true, income: null }),
  ];
  const v = moneyVolume({ accounts, expenses: 2000, expensesLive: true, monthLabel: 'Aug 2026', statementIsThisMonth: true });

  test('the headline is month-to-date income, with money out and net as dot chips', () => {
    expect(v.headline).toBe(8000);
    expect(v.upper).toBeNull();
    expect(v.chips).toEqual([
      { tone: 'err', text: '$2,000 out' },
      { tone: 'ok', text: '+$6,000 net' },
    ]);
    expect(v.caption).toBe('income this month · 2/3 processors live');
  });

  test('one meter per processor, each its real share of income, then spend as a share of income', () => {
    expect(v.meters.map((m) => m.label)).toEqual([
      'Stripe · Launchpad Cohort',
      'Stripe · Vantage',
      'PayKit · Launchpad Cohort',
      'Spent of income · Aug 2026',
    ]);
    expect(v.meters[0].frac).toBeCloseTo(0.75);
    expect(v.meters[1].frac).toBeCloseTo(0.25);
    expect(v.meters[0].display).toBe('$6,000');
    // a keyed account that is not pulling reads pending, never $0
    expect(v.meters[2]).toMatchObject({ frac: 0, display: 'pull pending' });
    expect(v.meters[3]).toMatchObject({ display: '25%', hue: 'var(--warn)' });
    expect(v.meters[3].frac).toBeCloseTo(0.25);
    expect(v.foot).toBe('expenses from the uploaded Aug 2026 statement');
  });

  test('the insight card is what was kept, as a fraction of what came in', () => {
    expect(v.insight.display).toBe('+$6,000');
    expect(v.insight.frac).toBeCloseTo(0.75);
    expect(v.insight.headline).toBe('kept of $8,000 in this month.');
  });

  test('a bounded month shows its floor as the headline and carries the ceiling', () => {
    const b = moneyVolume({
      accounts: [acct({ id: 'paykit-lc', income: 1000, incomeUpper: 1500, unsplittableCustomers: 2 })],
      expenses: 0,
      expensesLive: false,
      monthLabel: null,
    });
    expect(b.headline).toBe(1000);
    expect(b.upper).toBe(1500);
    expect(b.meters[0].display).toBe('$1,000 - $1,500');
  });

  test('spending more than came in turns the spend meter red and the net chip red, capped at a full bar', () => {
    const r = moneyVolume({ accounts: [acct({ income: 1000 })], expenses: 1500, expensesLive: false, monthLabel: null });
    expect(r.meters.at(-1)).toMatchObject({ frac: 1, display: '150%', hue: 'var(--err)', label: 'Set fees of income' });
    expect(r.chips[1]).toEqual({ tone: 'err', text: '−$500 net' });
    expect(r.insight.frac).toBe(0);
    expect(r.insight.headline).toBe('more out than in this month.');
    expect(r.foot).toBe('expenses are declared set fees · upload a card statement for real months');
  });

  test('no income at all: empty meters and honest copy, not fabricated fills', () => {
    const e = moneyVolume({
      accounts: [acct({ live: false, configured: false, income: null })],
      expenses: 1200,
      expensesLive: false,
      monthLabel: null,
    });
    expect(e.headline).toBe(0);
    expect(e.meters.every((m) => m.frac === 0)).toBe(true);
    expect(e.meters[0].display).toBe('awaiting key');
    expect(e.meters.at(-1)!.display).toBe('no income yet');
    expect(e.insight.frac).toBe(0);
  });
});

describe('moneyVolume never divides one month by another (review 2026-09-24)', () => {
  // $500 in so far this month against a big August statement read "1600%"
  // and "−$7,500 kept this month": two different months in one ratio.
  const stale = moneyVolume({ accounts: [acct({ income: 500 })], expenses: 8000, expensesLive: true, monthLabel: 'Aug 2026', statementIsThisMonth: false });

  test('the spend meter is empty and says the statement is for another month', () => {
    const m = stale.meters.at(-1)!;
    expect(m.frac).toBe(0);
    expect(m.display).toBe('no statement this month');
    expect(m.label).toBe('Spent of income · Aug 2026');
  });

  test('no net chip and no kept-this-month number across months', () => {
    expect(stale.chips).toEqual([{ tone: 'err', text: '$8,000 out · Aug 2026' }]);
    expect(stale.insight.display).toBe(' - ');
    expect(stale.insight.frac).toBe(0);
    expect(stale.insight.headline).toMatch(/this month's statement/);
  });

  test('set fees are monthly, so they still compare with this month', () => {
    const fees = moneyVolume({ accounts: [acct({ income: 500 })], expenses: 1200, expensesLive: false, monthLabel: null, statementIsThisMonth: false });
    expect(fees.chips.map((c) => c.text)).toEqual(['$1,200 out', '−$700 net']);
  });
});

describe('spendSeries', () => {
  test('spend per ledger month, oldest first, in whole dollars; income rows ignored', () => {
    const s = spendSeries([
      row('2026-07-03', 100.4),
      row('2026-07-20', 50),
      row('2026-08-02', 900),
      row('2026-08-05', 5000, { direction: 'in' }),
    ]);
    expect(s).toEqual([
      { label: 'Jul 2026', count: 150 },
      { label: 'Aug 2026', count: 900 },
    ]);
  });
  test('an empty ledger is an empty series', () => {
    expect(spendSeries([])).toEqual([]);
  });
});

describe('chargeSizes', () => {
  test('recent charges bucketed by size, every bucket present', () => {
    const cols = chargeSizes([{ amount: 5_000 }, { amount: 20_000 }, { amount: 30_000 }, { amount: 150_000 }, { amount: 500_000 }]);
    expect(cols).toEqual([
      { label: '<$100', count: 1 },
      { label: '$100-500', count: 2 },
      { label: '$500-2k', count: 1 },
      { label: '$2k+', count: 1 },
    ]);
  });
  test('no charges keeps the columns, all empty', () => {
    expect(chargeSizes([]).every((c) => c.count === 0)).toBe(true);
    expect(chargeSizes([])).toHaveLength(4);
  });
});
