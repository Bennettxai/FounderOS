import type { IncomeAccount } from '@/lib/finances';
import { totalIncome, totalIncomeUpper } from '@/lib/finances';
import { monthlyTotals, type SpendRow } from '@/lib/spend-report';

/**
 * /finances in the Brand Deals look (Alex, 2026-09-24). The page's own
 * numbers shaped like Deal Volume: a month-to-date income headline, money out
 * and net as dot chips, one meter per processor (its real share of income),
 * spend as a share of income, and the one insight card (what was kept).
 *
 * Pure: the page gathers the money, this decides how it reads, and
 * tests/finances-volume.test.ts pins it. An account with no pull reads
 * pending, never $0; no income means empty meters, never a fabricated fill.
 */

export type MoneyMeter = { label: string; frac: number; display: string; hue: string };
export type MoneyChip = { tone: 'ok' | 'warn' | 'err' | 'accent'; text: string };
export type SizePoint = { label: string; count: number };

export type MoneyVolume = {
  /** Month-to-date income, the PROVEN floor. */
  headline: number;
  /** The ceiling when a source could only bound its month; null when exact. */
  upper: number | null;
  chips: MoneyChip[];
  caption: string;
  meters: MoneyMeter[];
  foot: string;
  insight: { display: string; headline: string; body: string; frac: number };
};

const usd = (n: number) => n.toLocaleString('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 });
const signed = (n: number) => `${n < 0 ? '−' : '+'}${usd(Math.abs(n))}`;
const clamp01 = (n: number) => (Number.isFinite(n) ? Math.max(0, Math.min(1, n)) : 0);

// One hue per income source, all on the colorway ramp.
const SOURCE_HUES = ['var(--ramp-1)', 'var(--ramp-2)', 'var(--ramp-3)', 'var(--ramp-4)'];

export function moneyVolume(x: {
  accounts: IncomeAccount[];
  /** USD for the month the expense figure covers. */
  expenses: number;
  /** true when expenses come from an uploaded statement, false for set fees. */
  expensesLive: boolean;
  /** "Aug 2026" for the uploaded month, null when the figure is set fees. */
  monthLabel: string | null;
  /** The uploaded statement covers the current month. Income is month to
      date, so a statement from any other month cannot be divided into it.
      Unset means not proven, so no cross-month ratio is drawn. */
  statementIsThisMonth?: boolean;
}): MoneyVolume {
  const income = totalIncome(x.accounts);
  const ceiling = totalIncomeUpper(x.accounts);
  const net = income - x.expenses;
  // Set fees are a monthly figure, so they compare with this month's income;
  // a statement only does when it IS this month's.
  const comparable = !x.expensesLive || x.statementIsThisMonth === true;
  const live = x.accounts.filter((a) => a.live).length;

  const sources: MoneyMeter[] = x.accounts.map((a, i) => ({
    label: a.label,
    frac: income > 0 ? clamp01((a.income ?? 0) / income) : 0,
    display:
      a.income == null
        ? a.configured
          ? 'pull pending'
          : 'awaiting key'
        : a.incomeUpper != null
          ? `${usd(a.income)} - ${usd(a.incomeUpper)}`
          : usd(a.income),
    hue: SOURCE_HUES[i % SOURCE_HUES.length],
  }));

  const spentRatio = income > 0 ? x.expenses / income : 0;
  const spend: MoneyMeter = {
    label: x.expensesLive ? `Spent of income${x.monthLabel ? ` · ${x.monthLabel}` : ''}` : 'Set fees of income',
    frac: comparable ? clamp01(spentRatio) : 0,
    display: !comparable ? 'no statement this month' : income > 0 ? `${Math.round(spentRatio * 100)}%` : 'no income yet',
    hue: comparable && x.expenses > income ? 'var(--err)' : 'var(--warn)',
  };

  return {
    headline: income,
    upper: ceiling > income ? ceiling : null,
    chips: comparable
      ? [
          { tone: 'err', text: `${usd(x.expenses)} out` },
          { tone: net >= 0 ? 'ok' : 'err', text: `${signed(net)} net` },
        ]
      : [{ tone: 'err', text: `${usd(x.expenses)} out · ${x.monthLabel ?? 'latest statement'}` }],
    caption: `income this month · ${live}/${x.accounts.length} processors live`,
    meters: [...sources, spend],
    foot: x.expensesLive
      ? `expenses from the uploaded ${x.monthLabel ?? 'latest'} statement`
      : 'expenses are declared set fees · upload a card statement for real months',
    insight: !comparable
      ? {
          display: ' - ',
          headline: "Upload this month's statement to see what was kept.",
          body: `${usd(income)} in so far this month; the latest statement is ${x.monthLabel ?? 'from another month'}.`,
          frac: 0,
        }
      : {
      display: signed(net),
      headline: net >= 0 ? `kept of ${usd(income)} in this month.` : 'more out than in this month.',
      body: `${usd(income)} in − ${usd(x.expenses)} out${x.expensesLive ? `, spend from the ${x.monthLabel ?? 'latest'} statement` : ', spend is the declared set fees'}.`,
      frac: income > 0 ? clamp01(net / income) : 0,
    },
  };
}

const monthName = (month: string): string =>
  new Date(`${month}-01T00:00:00Z`).toLocaleDateString('en-US', { month: 'short', year: 'numeric', timeZone: 'UTC' });

/** Spend per ledger month, oldest first, in whole dollars, for the step line. */
export function spendSeries(rows: SpendRow[]): SizePoint[] {
  return monthlyTotals(rows).map((m) => ({ label: monthName(m.month), count: Math.round(m.totalCents / 100) }));
}

const SIZE_BUCKETS: Array<{ label: string; maxCents: number }> = [
  { label: '<$100', maxCents: 10_000 },
  { label: '$100-500', maxCents: 50_000 },
  { label: '$500-2k', maxCents: 200_000 },
  { label: '$2k+', maxCents: Infinity },
];

/** Recent charges by size (Stripe cents), every bucket present so the dot
    matrix keeps its columns on a quiet month. */
export function chargeSizes(charges: Array<{ amount: number }>): SizePoint[] {
  return SIZE_BUCKETS.map((b, i) => {
    const min = i === 0 ? -Infinity : SIZE_BUCKETS[i - 1].maxCents;
    return { label: b.label, count: charges.filter((c) => c.amount >= min && c.amount < b.maxCents).length };
  });
}
