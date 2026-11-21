'use client';

/**
 * F8 spend report, Classic tree (and Bennett Legacy, which renders the Classic
 * tree under its skin): a bordered card with spend per month stacked by
 * category (chips toggle each category), month over month, top categories (a
 * click lists that category's transactions) and top merchants. Data:
 * lib/spend-report.ts over the uploaded card ledger only.
 */

import { useState, type CSSProperties } from 'react';
import type { SpendCard } from '@/lib/spend-report';

const usd = (n: number, cents = false) => n.toLocaleString('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: cents ? 2 : 0 });
const MASK: CSSProperties = { filter: 'blur(6px)', userSelect: 'none' };
/** Stack colours: the data ramp, then two funnel hues (Classic tokens, every theme). */
const COLORS = ['var(--ramp-1)', 'var(--ramp-3)', 'var(--ramp-4)', 'var(--ramp-2)', 'var(--funnel-s3)', 'var(--funnel-s5)'];
const CARD = 'rounded-lg-t border border-os-border bg-os-surface';

export function Spending({ spend, hide }: { spend: SpendCard; hide: boolean }) {
  const [hidden, setHidden] = useState<string[]>([]);
  const [open, setOpen] = useState<string | null>(null);
  const mask = hide ? MASK : undefined;

  if (spend.state === 'none') {
    return (
      <div className={`${CARD} flex flex-col items-start gap-2 px-6 py-6`} data-feature="spending">
        <h3 className="text-[15px] font-medium text-os-text">Spending</h3>
        <p className="text-[13px] text-os-muted">Upload a statement to see spending.</p>
        <a href="#statement-upload" className="c-btn">Upload a statement</a>
      </div>
    );
  }

  const { report, sample } = spend;
  const first = report.months[0];
  const last = report.months[report.months.length - 1];
  const keys = report.keys;
  const color = (k: string) => COLORS[keys.indexOf(k) % COLORS.length];
  const shownTotal = (m: (typeof report.months)[number]) => keys.reduce((n, k) => n + (hidden.includes(k) ? 0 : m.byCategory[k] ?? 0), 0);
  const max = Math.max(1, ...report.months.map(shownTotal));
  const mom = report.mom;
  const prevLabel = mom ? report.months.find((m) => m.month === mom.prevMonth)?.label ?? mom.prevMonth : '';
  const txns = open ? report.txns.filter((t) => t.category === open) : [];

  return (
    <div className={`${CARD} p-6`} data-feature="spending">
      <div className="mb-1 flex flex-wrap items-baseline gap-x-3 gap-y-1">
        <h3 className="text-[15px] font-medium text-os-text">Spending</h3>
        <span className="text-[12.5px] text-os-muted" style={mask}>
          Card statements · {first.label} to {last.label} · {usd(report.total)}
        </span>
        {sample && (
          <span data-sample-badge="" className="ml-auto rounded-full border border-os-border px-2 py-[1px] text-[11px] text-os-muted" title="Sample data: lib/spend-report.ts sampleSpendRows">
            Sample
          </span>
        )}
      </div>
      <p className="mb-4 text-[12.5px] text-os-muted" style={mask}>
        {usd(last.total)} in {last.label}
        {mom && mom.deltaPct !== null && `, ${mom.deltaPct >= 0 ? 'up' : 'down'} ${Math.abs(mom.deltaPct)}% on ${prevLabel}`}
      </p>

      <div className="mb-3 flex flex-wrap gap-2" role="group" aria-label="Show or hide a category">
        {keys.map((k) => (
          <button
            key={k}
            type="button"
            aria-pressed={!hidden.includes(k)}
            onClick={() => setHidden((h) => (h.includes(k) ? h.filter((x) => x !== k) : [...h, k]))}
            className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-[3px] text-[12px] ${hidden.includes(k) ? 'border-os-border text-os-dim' : 'border-os-border-strong text-os-text'}`}
          >
            <span className="h-[7px] w-[7px] rounded-full" style={{ background: color(k), opacity: hidden.includes(k) ? 0.35 : 1 }} />
            {k}
          </button>
        ))}
      </div>

      <div className="mb-5 flex h-[170px] items-end gap-3" style={mask} role="group" aria-label="Spending per month by category">
        {report.months.map((m) => {
          const h = (shownTotal(m) / max) * 150;
          return (
            <div key={m.month} className="flex min-w-0 flex-1 flex-col items-center gap-1.5" role="img" aria-label={`${m.label}: ${usd(shownTotal(m))}`} title={`${m.label} · ${usd(shownTotal(m))}`}>
              <div className="flex w-full max-w-[44px] flex-col-reverse gap-[2px] overflow-hidden rounded-[6px]" style={{ height: Math.max(h, shownTotal(m) > 0 ? 3 : 0) }}>
                {keys.filter((k) => !hidden.includes(k) && (m.byCategory[k] ?? 0) > 0).map((k) => (
                  <i key={k} style={{ flex: m.byCategory[k], background: color(k) }} />
                ))}
              </div>
              <span className="font-mono text-[11px] text-os-muted">{m.label}</span>
            </div>
          );
        })}
      </div>

      <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
        <section>
          <h4 className="mb-2 text-[12.5px] font-medium text-os-muted">Top categories</h4>
          <ul className="flex flex-col">
            {report.categories.slice(0, 6).map((c) => (
              <li key={c.category}>
                <button
                  type="button"
                  aria-expanded={open === c.category}
                  onClick={() => setOpen((o) => (o === c.category ? null : c.category))}
                  className="hoverable flex w-full items-baseline justify-between gap-3 border-t border-os-border py-2 text-left text-[13px]"
                  title={`See the ${c.category} transactions`}
                >
                  <span className="text-os-text">
                    {c.category} <span className="text-[12px] text-os-muted">· {c.count} · {Math.round(c.share * 100)}%</span>
                  </span>
                  <span className="tabular-nums text-os-text" style={mask}>{usd(c.total)}</span>
                </button>
                {open === c.category && (
                  <ul className="glass-well mb-2 max-h-[220px] overflow-y-auto px-3 py-1" aria-label={`${c.category} transactions`}>
                    {txns.map((t) => (
                      <li key={`${t.date}|${t.description}|${t.usd}`} className="flex items-baseline justify-between gap-3 py-1 text-[12px]">
                        <span className="font-mono text-os-muted">{t.date}</span>
                        <span className="min-w-0 flex-1 truncate text-os-text" title={t.description}>{t.merchant}</span>
                        <span className="tabular-nums text-os-text" style={mask}>{usd(t.usd, true)}</span>
                      </li>
                    ))}
                  </ul>
                )}
              </li>
            ))}
          </ul>
        </section>
        <section>
          <h4 className="mb-2 text-[12.5px] font-medium text-os-muted">Top merchants</h4>
          <ul className="flex flex-col">
            {report.merchants.map((m) => (
              <li key={m.merchant} className="flex items-baseline justify-between gap-3 border-t border-os-border py-2 text-[13px]">
                <span className="text-os-text">
                  {m.merchant} <span className="text-[12px] text-os-muted">· {m.count}</span>
                </span>
                <span className="tabular-nums text-os-text" style={mask}>{usd(m.total)}</span>
              </li>
            ))}
          </ul>
        </section>
      </div>
    </div>
  );
}
