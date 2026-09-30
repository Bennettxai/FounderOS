'use client';

import { useEffect, useMemo, useState } from 'react';
import { ChevronLeft, ChevronRight, Maximize2 } from 'lucide-react';
import { cardLabel } from '@/lib/cards';
import { cardTotals, categoryTotals, monthAfterRefresh, spendTotalCents, type SpendRow } from '@/lib/spend-report';
import { SharePie } from '@/components/SharePie';
import { ExpenditureReport } from '@/components/ExpenditureReport';
import { STATEMENT_UPLOADED, type StatementUploadedDetail } from '@/lib/statement-events';
import { VolumeMeter } from '@/components/VolumeMeter';
import { chipClass } from '@/components/slab';

const usd = (cents: number) =>
  (cents / 100).toLocaleString('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 });

const monthName = (month: string): string =>
  new Date(`${month}-01T00:00:00Z`).toLocaleDateString('en-US', { month: 'short', year: 'numeric', timeZone: 'UTC' });

/**
 * Monthly expenses (Alex, 2026-08-26): the pie is per month, not a frozen
 * latest-month snapshot  -  stepping months redraws it from the ledger rows the
 * server handed down, and the full expenditure statement opens over the page.
 * With no statements uploaded it falls back to the declared set fees, honestly
 * labelled.
 */
export function MonthlyExpenses({
  rows,
  months,
  fallback,
  children,
}: {
  rows: SpendRow[];
  months: string[]; // ascending
  /** declared set fees in cents, used only when nothing has been uploaded */
  fallback: { category: string; totalCents: number }[];
  /** the statement uploader, rendered as the third column */
  children?: React.ReactNode;
}) {
  const live = rows.length > 0;
  const [month, setMonth] = useState<string | null>(() => monthAfterRefresh(null, [], months));
  const [open, setOpen] = useState(false);

  // An upload refreshes the server data under us, and React keeps this client
  // state across that refresh  -  so without this the month he just submitted
  // showed up as an unselected chip while the pie kept drawing the old one.
  // Sync during render (no effect): the panel never paints the stale month.
  const [seenMonths, setSeenMonths] = useState<string[]>(months);
  if (seenMonths.length !== months.length || seenMonths.some((m, i) => m !== months[i])) {
    setSeenMonths(months);
    setMonth(monthAfterRefresh(month, seenMonths, months));
  }

  // A statement just uploaded from the panel below: go to the month it covered,
  // even one already in the ledger (a re-upload of a month he is not reading).
  // It is held until the refreshed rows actually carry that month, otherwise
  // the pie would briefly draw an empty one.
  const [pending, setPending] = useState<string | null>(null);
  useEffect(() => {
    const onUploaded = (e: Event) => {
      const uploaded = (e as CustomEvent<StatementUploadedDetail>).detail?.months ?? [];
      if (uploaded.length > 0) setPending(uploaded[uploaded.length - 1]);
    };
    window.addEventListener(STATEMENT_UPLOADED, onUploaded);
    return () => window.removeEventListener(STATEMENT_UPLOADED, onUploaded);
  }, []);
  if (pending && months.includes(pending)) {
    setPending(null);
    if (pending !== month) setMonth(pending);
  }

  const index = month ? months.indexOf(month) : -1;

  const step = (delta: number) => {
    const next = months[index + delta];
    if (next) setMonth(next);
  };

  const cats = useMemo(
    () => (live ? categoryTotals(rows, month) : fallback),
    [live, rows, month, fallback],
  );
  const lanes = useMemo(() => cardTotals(rows, month), [rows, month]);
  // Last six months, plus the selected one when a back-dated statement lands
  // outside that window  -  the chosen chip is always on screen.
  const chips = useMemo(() => {
    const tail = months.slice(-6);
    return month && !tail.includes(month) ? [month, ...tail] : tail;
  }, [months, month]);
  const total = live ? spendTotalCents(rows, month) : fallback.reduce((s, c) => s + c.totalCents, 0);
  const period = live ? (month ? monthName(month) : 'all time') : 'per month';

  return (
    <section className="flex h-full flex-col">
      {/* the slab card head: 19px title, mono total, the way in to the full statement */}
      <div className="flex flex-wrap items-center justify-between gap-3 px-6 pt-5">
        <div className="flex min-w-0 items-baseline gap-2">
          <h2 className="text-[19px] font-semibold tracking-[-0.01em]">Where it goes</h2>
          <span className="font-mono text-[12px] tabular-nums text-os-dim">
            {live ? `${usd(total)} · ${period}` : `${usd(total)} /mo`}
          </span>
        </div>
        <button
          type="button"
          onClick={() => setOpen(true)}
          disabled={!live}
          data-lens="c"
          className="pressable inline-flex items-center gap-1.5 rounded-full border border-os-border px-4 py-2 text-[13px] text-os-muted hover:border-os-border-strong hover:text-os-text disabled:opacity-40"
        >
          <Maximize2 className="h-3 w-3" strokeWidth={1.8} />
          View full expenditure statement
        </button>
      </div>

      {/* month switcher: filter pills, the chosen month solid accent */}
      <div className="mt-4 flex flex-wrap items-center gap-2 px-6">
        <button
          type="button"
          onClick={() => step(-1)}
          disabled={!live || index <= 0}
          aria-label="Previous month"
          className="pressable rounded-ctl border border-os-border p-1.5 text-os-muted hover:border-os-border-strong disabled:opacity-30"
        >
          <ChevronLeft className="h-3.5 w-3.5" strokeWidth={1.8} />
        </button>
        {chips.map((m) => (
          <button key={m} type="button" onClick={() => setMonth(m)} data-lens="c" className={chipClass(m === month)}>
            {monthName(m)}
          </button>
        ))}
        <button
          type="button"
          onClick={() => step(1)}
          disabled={!live || index < 0 || index >= months.length - 1}
          aria-label="Next month"
          className="pressable rounded-ctl border border-os-border p-1.5 text-os-muted hover:border-os-border-strong disabled:opacity-30"
        >
          <ChevronRight className="h-3.5 w-3.5" strokeWidth={1.8} />
        </button>
        {!live && (
          <span className="font-mono text-[10.5px] uppercase tracking-[0.1em] text-os-warn">
            set fees · upload a card statement for real months
          </span>
        )}
      </div>

      <div className="mt-5 grid flex-1 items-stretch gap-5 px-6 pb-6 md:grid-cols-2 2xl:grid-cols-[1.15fr_1fr_0.85fr]">
        {/* where the money goes: share per category, for the chosen month */}
        <SharePie
          items={cats.map((c) => ({ key: c.category, label: c.category, value: c.totalCents }))}
          total={total}
          centerLabel={period}
          format={(cents) => usd(cents)}
          framed={false}
          stacked
          donutPx={190}
          ariaLabel="Monthly expenses by category"
        />

        {/* each category as a Deal Volume bar: its real share of the month's spend.
            Keyed by month so stepping months sweeps them out again. */}
        <div className="flex flex-col justify-between">
          <div className="flex flex-col gap-4">
            {cats.length === 0 ? (
              <p className="py-3 text-center font-mono text-[10.5px] text-os-dim">Nothing spent this month.</p>
            ) : (
              cats.map((c, i) => (
                <VolumeMeter
                  key={`${month ?? 'fees'}-${c.category}`}
                  label={c.category}
                  frac={total > 0 ? c.totalCents / total : 0}
                  display={usd(c.totalCents)}
                  hue="var(--ramp-4)"
                  delay={300 + i * 120}
                />
              ))
            )}
          </div>

          {/* the three card lanes, so the split is visible without opening the report */}
          {live && (
            <div className="mt-4 flex flex-wrap gap-x-4 gap-y-1 border-t border-os-border pt-3">
              {lanes.map((l) => (
                <span key={l.card} className="font-mono text-[10.5px] text-os-dim">
                  {cardLabel(l.card)} <span className="text-os-muted">{usd(l.totalCents)}</span>
                </span>
              ))}
            </div>
          )}
        </div>

        <div className="md:col-span-2 2xl:col-span-1">{children}</div>
      </div>

      {open && (
        <ExpenditureReport rows={rows} months={months} initialMonth={month} onClose={() => setOpen(false)} />
      )}
    </section>
  );
}
