'use client';

/**
 * /finances client shell. The server page (page.tsx) does every data fetch
 * (Stripe, FanBasis, the local ledger/bank stores); this component owns the
 * two pieces of state that need the browser — the
 * hero search box and the hide-values toggle — and renders everything that
 * either one touches. Search is a pure client-side filter over data the
 * server already sent down (see ./search.ts); hide-values persists to
 * localStorage, guarded, default visible.
 */

import { useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import { ArrowDownLeft, ArrowUpRight, Scale, Landmark, Send, Eye, EyeOff, Sparkles, Search as SearchIcon, X } from 'lucide-react';
import { PageHeader } from '@/components/PageHeader';
import { SetupLink } from '@/components/SetupLink';
import { SharePie } from '@/components/SharePie';
import { StatementUploader } from '@/components/StatementUploader';
import { BusinessIncomeChart } from '@/components/BusinessIncomeChart';
import { Badge, Label, SectionHead } from '@/components/terminal';
import { PipelineChart, CARET_ANIMATION, type PipelineStage } from '@/components/PipelineChart';
import { matchesQuery, type SearchableTxn } from './search';
import { formatCompact, formatUsdCompact } from '@/lib/format-number';
import type { BusinessSeries } from '@/lib/bank-statements';
import { processorPeriodLabel, processorSetupHref, type ExpensesSource } from '@/lib/finances';
import type { SpendCard } from '@/lib/spend-report';
import { Spending } from './Spending';

const HIDE_VALUES_KEY = 'finances:hide-values';
const MASK_STYLE: React.CSSProperties = { filter: 'blur(6px)', userSelect: 'none' };

const usd = (n: number, cents = false) =>
  n.toLocaleString('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: cents ? 2 : 0 });

function ago(unix: number): string {
  const mins = Math.round((Date.now() - unix * 1000) / 60_000);
  if (mins < 60) return `${Math.max(0, mins)}m`;
  const hours = Math.round(mins / 60);
  if (hours < 48) return `${hours}h`;
  return `${Math.round(hours / 24)}d`;
}

/** Blurs its children when hide-values is on — the one masking treatment
    used everywhere on this page, including inside components we don't own
    (SharePie, BusinessIncomeChart) via a wrapping span/div. */
function Money({ children, hide, block }: { children: React.ReactNode; hide: boolean; block?: boolean }) {
  const style = hide ? MASK_STYLE : undefined;
  return block ? <div style={style}>{children}</div> : <span style={style}>{children}</span>;
}

export type FinancesAccount = {
  id: string;
  processor: string;
  label: string;
  configured: boolean;
  live: boolean;
  income: number | null;
};

export type FinancesClientProps = {
  headlineTotal: number;
  headlinePeriod: string;
  incomeMtd: number;
  netMonthly: number;
  expenses: number;
  expensesSource: ExpensesSource;
  monthLabel: string | null;
  categories: { key: string; label: string; usd: number }[];
  accounts: FinancesAccount[];
  byCategory: { category: string; total: number }[];
  bankSeries: BusinessSeries[];
  wiseOut: { amountCents: number; currency: string; status: string; created: string | number; reference?: string }[] | null;
  recent: { amount: number; currency: string; description: string; created: number }[];
  stripeLive: boolean;
  available: number;
  pending: number;
  liveCount: number;
  maxAccount: number;
  maxCategory: number;
  /** F8 spend report, decided on the server. */
  spend?: SpendCard;
};

export function FinancesClient({
  headlineTotal,
  headlinePeriod,
  incomeMtd,
  netMonthly,
  expenses,
  expensesSource,
  monthLabel,
  categories,
  accounts,
  byCategory,
  bankSeries,
  wiseOut,
  recent,
  stripeLive,
  available,
  pending,
  liveCount,
  maxAccount,
  maxCategory,
  spend,
}: FinancesClientProps) {
  const [hideValues, setHideValues] = useState(false);
  const [hydrated, setHydrated] = useState(false);
  const [query, setQuery] = useState('');

  // Hide-values persists across visits — default visible, guarded so a
  // blocked/unavailable localStorage never breaks the page.
  useEffect(() => {
    try {
      setHideValues(localStorage.getItem(HIDE_VALUES_KEY) === '1');
    } catch {
      /* private mode / blocked storage — stay visible */
    }
    setHydrated(true);
  }, []);
  useEffect(() => {
    if (!hydrated) return;
    try {
      localStorage.setItem(HIDE_VALUES_KEY, hideValues ? '1' : '0');
    } catch {
      /* nothing to persist to — the toggle still works for this session */
    }
  }, [hideValues, hydrated]);

  const categoryTotal = categories.reduce((s, c) => s + c.usd, 0);
  // ?category=<key> (from the home Finances strip) lights that column on
  // arrival; an unknown key falls back to the largest category (S10 FR-12).
  const searchParams = useSearchParams();
  const [heroActive, setHeroActive] = useState(() => {
    const wanted = searchParams?.get('category');
    const hit = wanted ? categories.findIndex((c) => c.key === wanted) : -1;
    return hit >= 0 ? hit : categories.reduce((best, c, i) => (c.usd > categories[best]?.usd ? i : best), 0);
  });
  const activeQuery = query.trim().length > 0;

  const heroStages: PipelineStage[] = useMemo(
    () =>
      categories.map((c) => ({
        key: c.key,
        label: c.label,
        value: c.usd,
        usd: c.usd,
        note: `${categoryTotal > 0 ? Math.round((c.usd / categoryTotal) * 100) : 0}% of ${headlinePeriod} revenue`,
        dim: activeQuery && !matchesQuery({ payee: c.label, category: c.label, amountUsd: c.usd }, query),
      })),
    [categories, categoryTotal, headlinePeriod, activeQuery, query],
  );

  const filteredRecent = useMemo(
    () =>
      recent.filter((c) =>
        matchesQuery({ payee: c.description, description: c.description, amountUsd: c.amount / 100 }, query),
      ),
    [recent, query],
  );
  const filteredWise = useMemo(
    () =>
      (wiseOut ?? []).filter((t) =>
        matchesQuery(
          { payee: t.reference ?? t.status, description: t.reference ?? t.status, amountUsd: t.amountCents / 100 },
          query,
        ),
      ),
    [wiseOut, query],
  );

  const accountSearchable = (a: FinancesAccount): SearchableTxn => ({
    payee: a.label,
    description: a.processor,
    amountUsd: a.income ?? 0,
  });
  const categorySearchable = (c: { category: string; total: number }): SearchableTxn => ({
    payee: c.category,
    category: c.category,
    amountUsd: c.total,
  });

  return (
    <div>
      <PageHeader
        eyebrow="Every processor, one view"
        title="Finances"
        right={
          <div className="flex flex-col items-end gap-2">
            <button
              type="button"
              onClick={() => setHideValues((v) => !v)}
              aria-pressed={hideValues}
              aria-label={hideValues ? 'Show values' : 'Hide values'}
              className="inline-flex items-center gap-1.5 rounded-full border border-os-border px-2.5 py-1 text-[11.5px] text-os-muted transition-colors hover:border-os-border-strong hover:text-os-text"
            >
              {hideValues ? <EyeOff size={13} strokeWidth={1.7} /> : <Eye size={13} strokeWidth={1.7} />}
              {hideValues ? 'Hidden' : 'Visible'}
            </button>
            <div className="flex items-baseline gap-2">
              <span className="text-[28px] font-normal leading-none tracking-[-0.01em] tabular-nums text-os-text">
                <Money hide={hideValues}>{usd(headlineTotal)}</Money>
              </span>
              <span className="text-[12px] text-os-dim">Revenue · {headlinePeriod}</span>
            </div>
            <Badge tone={netMonthly >= 0 ? 'ok' : 'err'}>
              <Money hide={hideValues}>
                {netMonthly >= 0 ? '+' : '−'}
                {usd(Math.abs(netMonthly))} net {headlinePeriod}
              </Money>
            </Badge>
          </div>
        }
      />

      {/* Hero — income by category, the Pipeline design, with the search bar
          melting out of its base. Typing filters everything below. */}
      <section className="mb-6">
        <div className="rounded-lg-t border border-os-border bg-os-surface pb-4">
          <div className="flex items-center justify-between px-6 pt-5">
            <h2 className="text-[19px] font-semibold tracking-[-0.01em]">Income by category</h2>
            <span className="text-[12.5px] text-os-dim">
              {headlinePeriod} · {categories.length} categories
            </span>
          </div>
          {categoryTotal > 0 ? (
            <PipelineChart
              stages={heroStages}
              active={heroActive}
              onSelect={(_, i) => setHeroActive(i)}
              formatValue={(v) => Math.round(v).toLocaleString('en-US')}
              responsive
              compactValue={formatCompact}
              compactUsd={formatUsdCompact}
              maskValues={hideValues}
              svgHeight={220}
              showAxis={false}
              searchSlot={
                <>
                  <div className="mb-2 flex items-center gap-2 px-1 text-[13px] text-os-muted">
                    <Sparkles size={14} strokeWidth={1.7} className="text-os-accent" />
                    What are you looking for?
                  </div>
                  <label className="flex items-center gap-2 rounded-[12px] border border-os-border bg-os-bg px-3.5 py-2.5">
                    <SearchIcon size={14} strokeWidth={1.7} className="shrink-0 text-os-dim" />
                    <input
                      value={query}
                      onChange={(e) => setQuery(e.target.value)}
                      placeholder="Search transactions"
                      className="w-full bg-transparent text-[13.5px] outline-none placeholder:text-os-dim"
                    />
                    {query && (
                      <button
                        type="button"
                        onClick={() => setQuery('')}
                        aria-label="Clear search"
                        className="shrink-0 text-os-dim transition-colors hover:text-os-text"
                      >
                        <X size={13} strokeWidth={1.8} />
                      </button>
                    )}
                    <span className="h-[15px] w-[1.5px] shrink-0 bg-os-accent" style={{ animation: CARET_ANIMATION }} />
                  </label>
                </>
              }
            />
          ) : (
            // No processor is connected/keyed yet, so there is nothing honest
            // to chart — an honest connect state instead of a faked or
            // all-zero graph.
            <div className="flex flex-col items-center gap-3 px-6 py-10 text-center">
              <p className="text-[13px] text-os-muted">No processor income configured yet.</p>
              <SetupLink href="/integrations" hint="connect a payment processor to see income by category" />
            </div>
          )}
        </div>
      </section>

      {/* Summary tiles — slim single-line rows so the page opens condensed */}
      <section className="mb-5 grid grid-cols-1 gap-3.5 sm:grid-cols-2 xl:grid-cols-4">
        <div className="flex flex-col gap-2 rounded-lg-t border border-os-border bg-os-surface px-4 py-3.5">
          <div className="flex items-center justify-between gap-3">
            <Label>Income · {headlinePeriod}</Label>
            <ArrowDownLeft className="h-3.5 w-3.5 text-os-dim" strokeWidth={1.6} />
          </div>
          <div className="flex items-baseline justify-between gap-3">
            <span className="text-[30px] font-normal leading-none tracking-[-0.01em] tabular-nums">
              <Money hide={hideValues}>{usd(incomeMtd)}</Money>
            </span>
            <span className="min-w-0 truncate text-[12.5px] text-os-muted">
              {liveCount}/{accounts.length} live
            </span>
          </div>
        </div>

        <div className="flex flex-col gap-2 rounded-lg-t border border-os-border bg-os-surface px-4 py-3.5">
          <div className="flex items-center justify-between gap-3">
            <Label>Expenses · /mo</Label>
            <ArrowUpRight className="h-3.5 w-3.5 text-os-dim" strokeWidth={1.6} />
          </div>
          <div className="flex items-baseline justify-between gap-3">
            <span className="text-[30px] font-normal leading-none tracking-[-0.01em] tabular-nums">
              {expensesSource === 'none' ? '—' : <Money hide={hideValues}>{usd(expenses)}</Money>}
            </span>
            {expensesSource === 'none' ? (
              <a
                href="#statement-upload"
                data-expenses-empty
                title="Upload a card or bank statement below to see monthly spend"
                className="min-w-0 truncate text-[12.5px] text-os-accent hover:underline"
              >
                No statements yet
              </a>
            ) : (
              <span className="flex min-w-0 items-center gap-2 truncate text-[12.5px] text-os-muted">
                {expensesSource === 'uploaded' && <span className="dot ok" />}
                {expensesSource === 'uploaded' ? (monthLabel ? `Uploaded · ${monthLabel}` : 'Uploaded') : 'per month'}
              </span>
            )}
          </div>
        </div>

        <div className="flex flex-col gap-2 rounded-lg-t border border-os-border bg-os-surface px-4 py-3.5">
          <div className="flex items-center justify-between gap-3">
            <Label>Net · {headlinePeriod}</Label>
            <Scale className="h-3.5 w-3.5 text-os-dim" strokeWidth={1.6} />
          </div>
          <div className="flex items-baseline justify-between gap-3">
            <span className="text-[30px] font-normal leading-none tracking-[-0.01em] tabular-nums">
              <Money hide={hideValues}>
                {netMonthly >= 0 ? '' : '−'}
                {usd(Math.abs(netMonthly))}
              </Money>
            </span>
            <span className="min-w-0 truncate text-[12.5px] text-os-muted">
              {expensesSource === 'none' ? 'no expenses uploaded' : 'in − out'}
            </span>
          </div>
        </div>

        <div className="flex flex-col gap-2 rounded-lg-t border border-os-border bg-os-surface px-4 py-3.5">
          <div className="flex items-center justify-between gap-3">
            <Label>Stripe balance</Label>
            <Landmark className="h-3.5 w-3.5 text-os-dim" strokeWidth={1.6} />
          </div>
          <div className="flex items-baseline justify-between gap-3">
            <span className="text-[30px] font-normal leading-none tracking-[-0.01em] tabular-nums">
              {stripeLive ? <Money hide={hideValues}>{usd(available, true)}</Money> : '—'}
            </span>
            <span className="min-w-0 truncate text-[12.5px] text-os-muted">
              {stripeLive ? (
                <Money hide={hideValues}>{usd(pending, true)} pending</Money>
              ) : (
                <SetupLink href={processorSetupHref('stripe')} label="Connect Stripe" hint="STRIPE_SECRET_KEY" />
              )}
            </span>
          </div>
        </div>
      </section>

      {/* Income by business — from uploaded bank statements */}
      {bankSeries.length > 0 && (
        <section className="mb-5">
          <SectionHead label="Income · by business" count="bank deposits" />
          <div className="grid grid-cols-1 gap-5 lg:grid-cols-2" style={hideValues ? MASK_STYLE : undefined}>
            {bankSeries.map((s) => (
              <BusinessIncomeChart key={s.business} series={s} />
            ))}
          </div>
        </section>
      )}

      {/* Monthly expenses by category */}
      <section className="mb-5">
        <SectionHead
          label="Monthly expenses · by category"
          count={
            expensesSource === 'none'
              ? 'none uploaded'
              : expensesSource === 'uploaded' && monthLabel
                ? `${usd(expenses)} · ${monthLabel}`
                : `${usd(expenses)} /mo`
          }
        />
        <div className="grid grid-cols-1 items-stretch gap-5 lg:grid-cols-[1.15fr_1fr_0.85fr]">
          {expensesSource === 'none' ? (
            // H9: no ledger and not Filming. Show where the figures come from,
            // never seeded sample rows dressed as real spend.
            <div
              data-expenses-empty
              className="flex flex-col items-center justify-center gap-2 rounded-lg-t border border-os-border bg-os-surface px-6 py-10 text-center lg:col-span-2"
            >
              <p className="text-[14px] text-os-text">Upload a card CSV to see spend by category</p>
              <p className="max-w-[380px] text-[12.5px] text-os-muted">
                Nothing has been uploaded yet, so there are no expense figures on this page.
              </p>
            </div>
          ) : (
          <>
          <div style={hideValues ? MASK_STYLE : undefined}>
            <SharePie
              items={byCategory.map((c) => ({ key: c.category, label: c.category, value: Math.round(c.total * 100) }))}
              total={Math.round(expenses * 100)}
              centerLabel={expensesSource === 'uploaded' && monthLabel ? monthLabel : 'per month'}
              format={(cents) => usd(cents / 100)}
              donutPx={190}
              ariaLabel="Monthly expenses by category"
            />
          </div>

          <div className="rounded-lg-t border border-os-border bg-os-surface p-6">
            <div className="flex flex-col gap-3.5">
              {byCategory.map((c) => {
                const dim = activeQuery && !matchesQuery(categorySearchable(c), query);
                return (
                  <div key={c.category} style={{ opacity: dim ? 0.4 : 1 }}>
                    <div className="mb-1 flex items-baseline justify-between gap-3 text-[12.5px]">
                      <span className="text-os-muted">{c.category}</span>
                      <span className="text-os-text tabular-nums">
                        <Money hide={hideValues}>{usd(c.total)}</Money>
                      </span>
                    </div>
                    <div className="h-1.5 overflow-hidden rounded-sm-t bg-os-surface2">
                      <div
                        className="h-full"
                        style={{ width: `${(c.total / maxCategory) * 100}%`, background: 'var(--ramp-1)' }}
                      />
                    </div>
                  </div>
                );
              })}
              {byCategory.length === 0 && (
                <p className="py-4 text-center text-[12.5px] text-os-dim">Nothing matches that filter.</p>
              )}
            </div>
          </div>
          </>
          )}

          {/* Statement ingestion: upload a CSV or PDF to fill this section */}
          <div id="statement-upload" className="h-full scroll-mt-24">
            <StatementUploader />
          </div>
        </div>
      </section>

      {/* Spending (FounderOS merge F8): by category per month, top merchants, month over month */}
      {spend && (
        <section className="mb-5">
          <Spending spend={spend} hide={hideValues} />
        </section>
      )}

      <section className="mb-5">
        <SectionHead label="Income · by processor" count={`${liveCount}/${accounts.length} live`} />
        <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 xl:grid-cols-3">
          {accounts.map((a) => {
            const dim = activeQuery && !matchesQuery(accountSearchable(a), query);
            return (
              <div
                key={a.id}
                className="hoverable rounded-lg-t border border-os-border bg-os-surface px-5 py-3"
                style={{ opacity: dim ? 0.4 : 1 }}
              >
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="text-[14px] font-medium">{a.label}</div>
                    <div className="mt-0.5 font-mono text-[11px] text-os-dim">{a.processor}</div>
                  </div>
                  {a.live ? (
                    <Badge>
                      <span className="dot ok pulse" /> Live
                    </Badge>
                  ) : a.configured ? (
                    <Badge>
                      <span className="dot warn" /> Key set
                    </Badge>
                  ) : (
                    <SetupLink href={processorSetupHref(a.id)} label="Connect" hint={`connect ${a.processor} to pull its income`} />
                  )}
                </div>
                <div className="mt-3 flex items-baseline gap-2.5">
                  <span className="text-[20px] font-normal tracking-[-0.01em] tabular-nums">
                    {a.income != null ? <Money hide={hideValues}>{usd(a.income)}</Money> : '—'}
                  </span>
                  <span className="text-[12px] text-os-dim">
                    {processorPeriodLabel(a, headlinePeriod)}
                  </span>
                </div>
                <div className="mt-3 h-1 overflow-hidden rounded-sm-t bg-os-surface2">
                  <div
                    className="h-full"
                    style={{
                      width: `${a.income != null ? (a.income / maxAccount) * 100 : 0}%`,
                      background: 'var(--ramp-1)',
                    }}
                  />
                </div>
              </div>
            );
          })}
        </div>
      </section>

      {/* Outgoing transfers — Wise (hidden entirely until a Wise key lands) */}
      {wiseOut && (
        <section className="mb-5">
          <SectionHead label="Outgoing · Wise" count={`${filteredWise.length} transfer${filteredWise.length === 1 ? '' : 's'}`} />
          {filteredWise.length === 0 ? (
            <div className="rounded-lg-t border border-os-border bg-os-surface px-5 py-3 text-[12.5px] text-os-muted">
              {activeQuery ? 'Nothing matches that search.' : 'Wise connected · no recent outgoing transfers'}
            </div>
          ) : (
            <ul className="space-y-2.5">
              {filteredWise.map((t, i) => (
                <li
                  key={`${t.created}-${i}`}
                  className="hoverable flex items-center gap-5 rounded-lg-t border border-os-border bg-os-surface px-5 py-3"
                >
                  <Send className="h-[15px] w-[15px] shrink-0 text-os-dim" strokeWidth={1.6} />
                  <span className="text-[15px] tabular-nums text-os-err">
                    <Money hide={hideValues}>
                      −{(t.amountCents / 100).toLocaleString('en-US', { style: 'currency', currency: t.currency })}
                    </Money>
                  </span>
                  <span className="min-w-0 flex-1 truncate text-[13.5px] text-os-muted">{t.reference ?? t.status}</span>
                  <span className="shrink-0 text-[12.5px] text-os-dim">{t.status}</span>
                </li>
              ))}
            </ul>
          )}
        </section>
      )}

      {/* Recent income — real Stripe charges */}
      {stripeLive && recent.length > 0 && (
        <section>
          <SectionHead label="Recent income" count="Stripe · live" />
          {filteredRecent.length === 0 ? (
            <div className="rounded-lg-t border border-os-border bg-os-surface px-5 py-3 text-[12.5px] text-os-muted">
              Nothing matches that search.
            </div>
          ) : (
            <ul className="space-y-2.5">
              {filteredRecent.map((c, i) => (
                <li
                  key={`${c.created}-${i}`}
                  className="hoverable flex items-center gap-5 rounded-lg-t border border-os-border bg-os-surface px-5 py-3"
                >
                  <span className="text-[15px] tabular-nums text-os-ok">
                    <Money hide={hideValues}>+{usd(c.amount / 100, true)}</Money>
                  </span>
                  <span className="min-w-0 flex-1 truncate text-[13.5px] text-os-muted">{c.description}</span>
                  <span className="shrink-0 font-mono text-[12.5px] text-os-dim">{ago(c.created)}</span>
                </li>
              ))}
            </ul>
          )}
        </section>
      )}
    </div>
  );
}
