import { ExternalLink, Send, Upload } from 'lucide-react';
import { configuredProcessors, monthToDateIncome, stripeMtdForKey, stripeSnapshot, wiseOutgoing, paykitMonthToDateIncome } from '@/lib/connectors/payments';
import {
  incomeAccounts,
  totalIncome,
  totalIncomeUpper,
  totalExpenses,
  expensesByCategory,
  net,
  DECLARED_EXPENSES,
} from '@/lib/finances';
import type { IncomeBand } from '@/lib/finances';
import { openLedger } from '@/lib/ledger';
import { openPaykitHistory, type PaykitHistory } from '@/lib/paykit-history';
import type { SpendRow } from '@/lib/spend-report';
import { openBankStore } from '@/lib/bank';
import { businessSeries } from '@/lib/bank-statements';
import { StatementUploader } from '@/components/StatementUploader';
import { MonthlyExpenses } from '@/components/MonthlyExpenses';
import { BusinessIncomeChart } from '@/components/BusinessIncomeChart';
import { Slab, SlabTitle, SlabCard, BigStat, Chip, MeterStack, InsightCard, PILL, PILL_ACCENT } from '@/components/slab';
import { StepLine, DotMatrix } from '@/components/slab-charts';
import { moneyVolume, spendSeries, chargeSizes } from '@/lib/finances-volume';

export const dynamic = 'force-dynamic';

const usd = (n: number, cents = false) =>
  n.toLocaleString('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: cents ? 2 : 0 });

function ago(unix: number): string {
  const mins = Math.round((Date.now() - unix * 1000) / 60_000);
  if (mins < 60) return `${Math.max(0, mins)}m`;
  const hours = Math.round(mins / 60);
  if (hours < 48) return `${hours}h`;
  return `${Math.round(hours / 24)}d`;
}

export default async function FinancesPage() {
  const stripeKeyed = configuredProcessors(process.env).some((p) => p.id === 'stripe' && p.configured);

  // Stripe is only "live" when the API actually answers  -  a present-but-invalid
  // key (or a server env missing it) stays honest pending, never a fake live.
  let stripeLive = false;
  let mtdUsd: number | null = null;
  let available = 0;
  let pending = 0;
  let recent: { amount: number; currency: string; description: string; created: number }[] = [];
  if (stripeKeyed) {
    const [mtd, snap] = await Promise.all([
      monthToDateIncome().catch(() => null),
      stripeSnapshot().catch(() => null),
    ]);
    if (snap) {
      stripeLive = true;
      available = (snap.available[0]?.amount ?? 0) / 100;
      pending = (snap.pending[0]?.amount ?? 0) / 100;
      recent = snap.recentCharges;
    }
    mtdUsd = mtd ? mtd.amountCents / 100 : null;
  }

  // Which processors have keys (honest config), so non-Stripe cards show
  // "key set · pull pending" vs "connect →" rather than a misleading live badge.
  const configuredMap = Object.fromEntries(configuredProcessors(process.env).map((p) => [p.id, p.configured]));
  // Live month-to-date income per account (null when unkeyed): PayKit via
  // its customers API, Vantage's own Stripe via the charges API.
  // The PayKit pull is handed a snapshot store, so this render both READS
  // /customers and keeps it. A month bracketed by two stored snapshots comes
  // back exact rather than bounded. See lib/paykit-history.ts and OS-658.
  let fbHistory: PaykitHistory | null = null;
  try {
    fbHistory = openPaykitHistory();
  } catch {
    fbHistory = null; // No writable data dir  -  the band below still renders.
  }
  let fbAa: Awaited<ReturnType<typeof paykitMonthToDateIncome>> = null;
  let stripeMer: Awaited<ReturnType<typeof stripeMtdForKey>> = null;
  try {
    [fbAa, stripeMer] = await Promise.all([
      paykitMonthToDateIncome(process.env.PAYKIT_LC_KEY, undefined, fbHistory ?? undefined).catch(() => null),
      stripeMtdForKey(process.env.STRIPE_VANTAGE_KEY).catch(() => null),
    ]);
  } finally {
    fbHistory?.close();
  }
  // A PayKit month the snapshots do not reach back before stays a BAND, not a
  // number: the API alone cannot split a repeat buyer's lifetime spend across
  // months, so the card shows "floor - ceiling" rather than the old confident
  // (and, for six months, inflated) single figure. See OS-655.
  const liveIncomeUsd: Record<string, number | IncomeBand> = {};
  if (fbAa != null) liveIncomeUsd['paykit-lc'] = fbAa;
  if (stripeMer != null) liveIncomeUsd['stripe-vantage'] = stripeMer.amountCents / 100;
  const accounts = incomeAccounts({ connected: stripeLive, mtdUsd }, configuredMap, liveIncomeUsd);
  // Outgoing Wise transfers  -  null (no Wise key) hides the section entirely.
  const wiseOut = await wiseOutgoing(process.env).catch(() => null);
  const incomeMtd = totalIncome(accounts);
  // Expenses from the uploaded statement ledger when present; the DECLARED set
  // fees otherwise (Marco CSM  -  subscriptions arrive via statement upload).
  // Every out-row goes to the client so the pie and the full expenditure
  // statement can be recomputed per month without another round trip.
  let ledgerRows: SpendRow[] = [];
  let ledgerMonths: string[] = [];
  let ledgerSpend: { category: string; total: number }[] = [];
  let ledgerMonth: string | null = null;
  try {
    const ledger = openLedger();
    ledgerRows = ledger.allRows();
    ledgerMonths = ledger.monthsAscending();
    ledgerSpend = ledger.monthly();
    ledgerMonth = ledger.latestMonth();
    ledger.close();
  } catch {
    ledgerRows = [];
    ledgerMonths = [];
    ledgerSpend = [];
  }
  const expensesLive = ledgerSpend.length > 0;
  // Per-business income from uploaded bank statements (Vantage, General Ops…).
  let bankSeries: ReturnType<typeof businessSeries> = [];
  try {
    const bank = openBankStore();
    bankSeries = businessSeries(bank.all());
    bank.close();
  } catch {
    bankSeries = [];
  }
  // "2026-06" → "Jun 2026" for an honest period label on the uploaded figures.
  const monthLabel = ledgerMonth
    ? new Date(`${ledgerMonth}-01T00:00:00Z`).toLocaleDateString('en-US', { month: 'short', year: 'numeric', timeZone: 'UTC' })
    : null;
  const expenses = expensesLive ? ledgerSpend.reduce((s, c) => s + c.total, 0) : totalExpenses(DECLARED_EXPENSES);
  const netMonthly = net(incomeMtd, expenses);
  // Income is month to date, so only this month's statement (or the monthly
  // set fees) can be netted against it.
  const now = new Date();
  const thisMonth = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
  const statementIsThisMonth = ledgerMonth === thisMonth;
  const netComparable = !expensesLive || statementIsThisMonth;
  const liveCount = accounts.filter((a) => a.live).length;
  const incomeMtdUpper = totalIncomeUpper(accounts);
  // The Brand Deals shape (2026-09-24): the page's own money as a count-up
  // headline, dot chips, honest share meters and the one insight card.
  const vol = moneyVolume({ accounts, expenses, expensesLive, monthLabel, statementIsThisMonth });
  const spend = spendSeries(ledgerRows);
  const sizes = chargeSizes(recent);
  const largestCharge = recent.reduce((m, c) => Math.max(m, c.amount), 0) / 100;
  const latestSpend = spend.at(-1) ?? null;

  return (
    <Slab>
      <SlabTitle
        eyebrow="money · every processor, one view"
        title="Finances"
        meta={
          <>
            {usd(incomeMtd)}
            {incomeMtdUpper > incomeMtd ? ` - ${usd(incomeMtdUpper)}` : ''} in this month · {usd(expenses)} out
            {expensesLive ? ` (${monthLabel} statement)` : ' (set fees)'} · {liveCount}/{accounts.length} processors live
            {stripeLive ? ` · Stripe balance ${usd(available, true)}, ${usd(pending, true)} pending` : ' · Stripe balance needs a live key'}
          </>
        }
        right={
          <>
            {netComparable && (
              <Chip tone={netMonthly >= 0 ? 'ok' : 'err'}>
                {netMonthly >= 0 ? '+' : '−'}
                {usd(Math.abs(netMonthly))} net /mo
              </Chip>
            )}
            <a href="#statements" className={PILL}>
              <Upload size={13} strokeWidth={1.7} /> Upload statement
            </a>
            <a href="https://dashboard.stripe.com/" target="_blank" rel="noreferrer" className={PILL_ACCENT}>
              Open Stripe <ExternalLink size={12} strokeWidth={1.8} />
            </a>
          </>
        }
      />

      {/* Hero row: where the money goes + the Money Volume card */}
      <div className="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
        <SlabCard i={1} className="flex flex-col">
          {/* Monthly expenses by category, month by month (client-side) */}
          <MonthlyExpenses
            rows={ledgerRows}
            months={ledgerMonths}
            fallback={expensesByCategory(DECLARED_EXPENSES).map((c) => ({
              category: c.category,
              totalCents: Math.round(c.total * 100),
            }))}
          >
            {/* Statement ingestion: pick a card lane, drop a CSV or PDF */}
            <div id="statements" className="h-full scroll-mt-24">
              <StatementUploader />
            </div>
          </MonthlyExpenses>
        </SlabCard>

        <SlabCard i={2} title="Money Volume" sub="month to date" className="flex flex-col">
          <div className="flex flex-1 flex-col px-6 pb-6 pt-3">
            <BigStat
              value={vol.headline}
              kind="usd"
              unit={vol.upper != null ? `- ${usd(vol.upper)}` : undefined}
              chips={vol.chips}
              caption={vol.caption}
            />
            <MeterStack meters={vol.meters} foot={vol.foot} empty="no processors wired yet" />
          </div>
        </SlabCard>
      </div>

      {/* Second row: spend by month, charge sizes, THE gradient card */}
      <div className="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
        <SlabCard i={3} title="Spend by month" sub={expensesLive ? 'card statements' : 'no statements yet'}>
          <div className="px-6 pt-3">
            <BigStat
              size={30}
              value={latestSpend?.count ?? 0}
              kind="usd"
              caption={latestSpend ? `spent in ${latestSpend.label}, across ${spend.length} uploaded month${spend.length === 1 ? '' : 's'}` : 'upload a card statement to chart spend'}
            />
          </div>
          <StepLine series={spend} hue="var(--ramp-4)" unit=" USD" empty="No card statements uploaded yet." />
        </SlabCard>

        <SlabCard i={4} title="Charge sizes" sub={stripeLive ? 'Stripe · recent' : 'Stripe not live'}>
          <div className="flex items-end justify-between gap-4 px-6 pb-6 pt-3">
            <div>
              <BigStat size={30} value={recent.length} caption={`recent charge${recent.length === 1 ? '' : 's'}`} />
              <div className="mt-4 w-fit rounded-full border border-os-border px-3 py-1 text-[12px] text-os-muted">
                Largest: <span className="font-semibold tabular-nums">{stripeLive && recent.length > 0 ? usd(largestCharge, true) : ' - '}</span>
              </div>
            </div>
            <DotMatrix cols={sizes} hue="var(--ramp-1)" />
          </div>
        </SlabCard>

        {/* the ONE gradient insight card: what was kept */}
        <InsightCard
          i={5}
          badge="Kept this month"
          display={vol.insight.display}
          headline={vol.insight.headline}
          body={vol.insight.body}
          frac={vol.insight.frac}
        />
      </div>

      {/* Income by business  -  from uploaded bank statements, with range chips */}
      {bankSeries.length > 0 && (
        <SlabCard i={6} title="Income · by business" sub="bank deposits" className="mt-6">
          <div className="grid gap-4 px-6 pb-6 pt-4 lg:grid-cols-2">
            {bankSeries.map((s) => (
              <BusinessIncomeChart key={s.business} series={s} />
            ))}
          </div>
        </SlabCard>
      )}

      <SlabCard i={7} title="Income · by processor" sub={`${liveCount}/${accounts.length} live`} className="mt-6">
        <div className="mt-4 border-t border-os-border">
          {accounts.map((a) => (
            <div
              key={a.id}
              data-lens="r"
              className="pressable is-row grid grid-cols-[minmax(0,1fr)_auto_auto] items-center gap-5 border-b border-os-hairline px-6 py-3.5 last:border-0 max-[700px]:grid-cols-[minmax(0,1fr)_auto]"
            >
              <div className="min-w-0">
                <div className="truncate text-[13.5px] font-medium">{a.label}</div>
                <div className="truncate font-mono text-[11px] text-os-dim">
                  {a.processor}
                  {/* A bounded month says so out loud rather than printing one
                      confident number the source cannot actually support. */}
                  {a.unsplittableCustomers > 0
                    ? ` · ${a.unsplittableCustomers} repeat ${a.unsplittableCustomers === 1 ? 'customer' : 'customers'} · split unavailable`
                    : ''}
                </div>
              </div>
              <div className="text-right">
                <div className="font-mono text-[15px] font-semibold tabular-nums">
                  {a.income != null ? usd(a.income) : ' - '}
                  {a.incomeUpper != null ? <span className="text-os-dim"> - {usd(a.incomeUpper)}</span> : null}
                </div>
                <div className="font-mono text-[10.5px] text-os-dim">
                  {a.live ? 'this month' : a.configured ? 'pull pending' : 'awaiting key'}
                </div>
              </div>
              <span
                className="rounded-full px-2.5 py-0.5 font-mono text-[10.5px] uppercase tracking-[0.12em] max-[700px]:hidden"
                style={
                  a.live
                    ? { background: 'color-mix(in oklab, var(--ok) 16%, transparent)', color: 'var(--ok)' }
                    : a.configured
                      ? { background: 'color-mix(in oklab, var(--warn) 15%, transparent)', color: 'var(--warn)' }
                      : { background: 'color-mix(in oklab, var(--text) 8%, transparent)', color: 'var(--text-2)' }
                }
              >
                {a.live ? 'live' : a.configured ? 'key set' : 'connect →'}
              </span>
            </div>
          ))}
        </div>
      </SlabCard>

      {/* Recent income  -  real Stripe charges */}
      {stripeLive && recent.length > 0 && (
        <SlabCard i={8} title="Recent income" sub="Stripe · live" className="mt-6">
          <ul className="mt-4 border-t border-os-border">
            {recent.map((c, i) => (
              <li
                key={`${c.created}-${i}`}
                data-lens="r"
                className="pressable is-row flex items-center gap-4 border-b border-os-hairline px-6 py-3.5 last:border-0"
              >
                <span
                  className="shrink-0 rounded-full px-2.5 py-0.5 font-mono text-[11px] font-semibold tabular-nums"
                  style={{ background: 'color-mix(in oklab, var(--ok) 16%, transparent)', color: 'var(--ok)' }}
                >
                  +{usd(c.amount / 100, true)}
                </span>
                <span className="min-w-0 flex-1 truncate text-[13px] text-os-muted">{c.description}</span>
                <span className="shrink-0 font-mono text-[11px] text-os-dim">{ago(c.created)}</span>
              </li>
            ))}
          </ul>
        </SlabCard>
      )}

      {/* Outgoing transfers  -  Wise (hidden entirely until a Wise key lands) */}
      {wiseOut && (
        <SlabCard
          i={9}
          title="Outgoing · Wise"
          sub={`${wiseOut.length} transfer${wiseOut.length === 1 ? '' : 's'}`}
          className="mt-6"
        >
          {wiseOut.length === 0 ? (
            <div className="px-6 py-7 text-center font-mono text-[11.5px] text-os-dim">Wise connected · no recent outgoing transfers</div>
          ) : (
            <ul className="mt-4 border-t border-os-border">
              {wiseOut.map((t, i) => (
                <li
                  key={`${t.created}-${i}`}
                  data-lens="r"
                  className="pressable is-row flex items-center gap-4 border-b border-os-hairline px-6 py-3.5 last:border-0"
                >
                  <Send className="h-[15px] w-[15px] shrink-0 text-os-err" strokeWidth={1.8} />
                  <span
                    className="shrink-0 rounded-full px-2.5 py-0.5 font-mono text-[11px] font-semibold tabular-nums"
                    style={{ background: 'color-mix(in oklab, var(--err) 16%, transparent)', color: 'var(--err)' }}
                  >
                    −{(t.amountCents / 100).toLocaleString('en-US', { style: 'currency', currency: t.currency })}
                  </span>
                  <span className="min-w-0 flex-1 truncate text-[13px] text-os-muted">{t.reference ?? t.status}</span>
                  <span className="shrink-0 font-mono text-[11px] text-os-dim">{t.status}</span>
                </li>
              ))}
            </ul>
          )}
        </SlabCard>
      )}
    </Slab>
  );
}
