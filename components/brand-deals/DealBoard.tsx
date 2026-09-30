'use client';

/**
 * Brand Deals: the GladOS "Deal Journeys" slab (Futurism / Zentra layout,
 * imported 2026-09-17) fed by the Notion Brand Deals Hub. The page floats as
 * one slab, numerals rule, data owns all chroma (one hue per domain), a
 * hatched stepped funnel is the hero with an AI prompt bar melting out of it
 * (wired as a live filter), barber-pole meters, a step-line and a dot-matrix
 * mini, exactly ONE gradient insight card, and a detail drawer.
 *
 * Read-only by design: Notion stays the source of truth (Alex's brand-deal
 * agents and his friend's guest access live there), so every deal deep-links
 * back and nothing here writes. The board rereads the hub on refresh or
 * within the connector's cache window.
 */

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { X, Search, Sparkles, RefreshCw, Lightbulb, ExternalLink, CalendarClock } from 'lucide-react';
import { Badge, Label } from '@/components/terminal';
import { VolumeMeter } from '@/components/VolumeMeter';
import { StepLine, DotMatrix } from '@/components/slab-charts';
import { PipelineChart, useCountUp, CARET_ANIMATION } from '@/components/PipelineChart';
import type { BrandDealsResult } from '@/lib/connectors/brand-deals';
import type { BrandDeal } from '@/lib/schemas';
import {
  activitySeries,
  bucketOf,
  dealUsd,
  dueSoon,
  filterDeals,
  fmtUsd,
  funnelStages,
  parseQuery,
  sizeMatrix,
  timeAgo,
  volume,
  type Filter,
} from '@/lib/brand-deals-view';

const BTN_SOLID =
  'inline-flex items-center gap-1.5 rounded-[9px] bg-os-accent px-3.5 py-1.5 text-[12.5px] font-medium text-os-ink hover:bg-os-accent2 disabled:opacity-50';

// One hue per data domain (anti-drift rule). All of them ride the colorway.
const HUE = {
  accent: 'var(--accent)', // pipeline / hero
  amber: 'var(--warn)', // in talks: needs a reply
  cobalt: 'var(--ramp-1)', // in production
  violet: 'var(--ramp-4)', // S-tier
  activity: 'var(--send-activity)', // Notion edits
};

const GRAIN =
  "url(\"data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='160' height='160'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='2'/%3E%3C/filter%3E%3Crect width='160' height='160' filter='url(%23n)' opacity='0.55'/%3E%3C/svg%3E\")";

const EASE = 'cubic-bezier(.2,.7,.2,1)';

function syncAge(syncedAt: number | null): string | null {
  if (syncedAt === null) return null;
  const m = Math.max(0, Math.floor((Date.now() - syncedAt) / 60_000));
  if (m < 1) return 'synced just now';
  if (m < 60) return `synced ${m}m ago`;
  return `synced ${Math.floor(m / 60)}h ago`;
}

/** Card shell: near-bg surface, soft dual shadow, stagger-in, 1px hover lift. */
function Card({ children, delay = 0, className = '', style }: { children: React.ReactNode; delay?: number; className?: string; style?: React.CSSProperties }) {
  return (
    <div className={`bd-card relative rounded-[12px] border border-os-border bg-os-surface ${className}`} style={{ animation: `bd-rise .6s ${EASE} ${delay}ms backwards`, ...style }}>
      {children}
    </div>
  );
}

/** Card header with a REAL kebab: refresh the data or open the hub. */
function CardHead({ title, onRefresh, hubUrl }: { title: string; onRefresh: () => void; hubUrl: string | null }) {
  const [open, setOpen] = useState(false);
  return (
    <div className="relative flex items-center justify-between px-6 pt-5">
      <h2 className="text-[19px] font-semibold tracking-[-0.01em]">{title}</h2>
      <button
        onClick={() => setOpen((v) => !v)}
        aria-label={`${title} menu`}
        className="pressable grid h-8 w-8 place-items-center rounded-full border border-os-border text-[13px] leading-none text-os-dim hover:border-os-border-strong hover:text-os-text"
      >
        ···
      </button>
      {open && (
        <>
          <div className="fixed inset-0 z-[30]" onClick={() => setOpen(false)} aria-hidden="true" />
          <div className="absolute right-5 top-14 z-[40] w-[210px] overflow-hidden rounded-md-t border border-os-border bg-os-bg2/95 py-1 backdrop-blur">
            <button
              onClick={() => {
                setOpen(false);
                onRefresh();
              }}
              className="pressable flex w-full items-center gap-2.5 px-3.5 py-2 text-left text-[12.5px] text-os-muted hover:bg-[color-mix(in_oklab,var(--text)_6%,transparent)] hover:text-os-text"
            >
              <RefreshCw size={13} strokeWidth={1.7} /> Refresh from Notion
            </button>
            {hubUrl && (
              <a
                href={hubUrl}
                target="_blank"
                rel="noreferrer"
                onClick={() => setOpen(false)}
                className="flex w-full items-center gap-2.5 px-3.5 py-2 text-left text-[12.5px] text-os-muted hover:bg-[color-mix(in_oklab,var(--text)_6%,transparent)] hover:text-os-text"
              >
                <ExternalLink size={13} strokeWidth={1.7} /> Open in Notion
              </a>
            )}
          </div>
        </>
      )}
    </div>
  );
}

function StatNumber({ value }: { value: number }) {
  const v = useCountUp(value);
  return <>{v}</>;
}

function DollarStat({ value }: { value: number }) {
  const v = useCountUp(value);
  return <>{fmtUsd(v)}</>;
}

const FILTER_TO_STAGE_IDX: Record<Filter, number> = { all: 0, 'tier-s': 0, talks: 1, production: 2, paid: 3, declined: 3 };

function statusStyle(d: BrandDeal): React.CSSProperties {
  const b = bucketOf(d.status);
  if (b === 'paid') return { background: 'color-mix(in oklab, var(--ok) 16%, transparent)', color: 'var(--ok)' };
  if (b === 'declined' || b === 'paused') return { background: 'color-mix(in oklab, var(--err) 16%, transparent)', color: 'var(--err)' };
  if (b === 'inbound' || b === 'talking') return { background: 'color-mix(in oklab, var(--warn) 15%, transparent)', color: 'var(--warn)' };
  if (b === 'producing' || b === 'billing') return { background: 'color-mix(in oklab, var(--ramp-1) 15%, transparent)', color: HUE.cobalt };
  return { background: 'color-mix(in oklab, var(--text) 8%, transparent)', color: 'var(--text-2)' };
}

function badgeTone(d: BrandDeal): 'ok' | 'warn' | 'err' | 'default' {
  const b = bucketOf(d.status);
  if (b === 'paid') return 'ok';
  if (b === 'declined' || b === 'paused') return 'err';
  if (b === 'inbound' || b === 'talking') return 'warn';
  return 'default';
}

function Field({ k, v }: { k: string; v: string | null }) {
  return (
    <div className="flex items-baseline justify-between gap-4 border-b border-os-hairline py-2 last:border-0">
      <span className="text-[12px] text-os-dim">{k}</span>
      <span className="text-right text-[12.5px] tabular-nums text-os-text">{v ?? <span className="text-os-dim">not set</span>}</span>
    </div>
  );
}

export function DealBoard({ initial }: { initial: BrandDealsResult }) {
  const [result, setResult] = useState(initial);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [query, setQuery] = useState('');
  const [stageFilter, setStageFilter] = useState<Filter>('all');
  const pending = useRef(false);

  const refresh = useCallback(async () => {
    if (pending.current) return;
    pending.current = true;
    try {
      const r = await fetch('/api/brand-deals');
      if (r.ok) setResult(await r.json());
    } catch {
      /* keep last good */
    } finally {
      pending.current = false;
    }
  }, []);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setSelectedId(null);
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);

  useEffect(() => {
    const t = setInterval(refresh, 60_000);
    return () => clearInterval(t);
  }, [refresh]);

  const { deals, mode, detail, syncedAt } = result;
  const vol = useMemo(() => volume(deals), [deals]);
  const stages = useMemo(() => funnelStages(deals), [deals]);
  const series = useMemo(() => activitySeries(deals, new Date(), 30), [deals]);
  const sizes = useMemo(() => sizeMatrix(deals), [deals]);
  const due = useMemo(() => dueSoon(deals, new Date(), 7), [deals]);
  const filtered = useMemo(() => filterDeals(deals, { filter: stageFilter, query }), [deals, stageFilter, query]);
  const { slash } = parseQuery(query);
  const selected = deals.find((d) => d.id === selectedId) ?? null;
  const editsInWindow = series.reduce((n, s) => n + s.count, 0);
  const largest = deals.reduce((m, d) => Math.max(m, dealUsd(d)), 0);
  const needsYou = due.followUps.length + due.deadlines.length;
  const openCount = vol.counts.talks + vol.counts.production;
  const hubUrl = deals.find((d) => !d.seeded)?.notionUrl ?? null;
  const age = syncAge(syncedAt);

  return (
    <div>
      <style>{`
        @keyframes bd-rise { from { opacity: 0; transform: translateY(10px) scale(.992); } to { opacity: 1; transform: none; } }
        @keyframes bd-fadein { from { opacity: 0; } to { opacity: 1; } }
        @keyframes bd-draw { to { stroke-dashoffset: 0; } }
        @keyframes bd-meter-in { from { width: 0; } }
        @keyframes bd-drift { from { background-position: 0% 0%; } to { background-position: 12% 8%; } }
        .bd-card { transition: transform .15s ease, border-color .15s ease; }
        .bd-card:hover { transform: translateY(-1px); border-color: var(--border-strong); }
      `}</style>

      {/* The slab: the whole view floats as one surface. */}
      <div
        className="rounded-[28px] border border-os-border p-7"
        style={{ background: 'var(--bg-2)', boxShadow: '0 1px 2px rgba(0,0,0,.4), 0 24px 70px -18px rgba(0,0,0,.6)', animation: `bd-rise .7s ${EASE} backwards` }}
      >
        {/* Title row */}
        <div className="mb-7 flex flex-wrap items-end justify-between gap-5">
          <div>
            <div className="mb-2 font-mono text-[11px] uppercase tracking-[0.32em] text-os-dim">// sponsorships · notion brand deals hub</div>
            <h1 className="text-[46px] font-semibold leading-none tracking-[-0.035em]">Brand Deals</h1>
            <div className="mt-3 font-mono text-[11px] text-os-dim">
              {fmtUsd(vol.openUsd)} open pipeline · {deals.length} deal{deals.length === 1 ? '' : 's'} ·{' '}
              {mode === 'live' ? `${deals.length} Notion rows${age ? ` · ${age}` : ''}` : detail}
            </div>
          </div>
          <div className="flex items-center gap-2.5">
            {mode === 'live' ? (
              <Badge tone="ok">live · Notion</Badge>
            ) : mode === 'seeded' ? (
              <Badge tone="warn">seeded · connect Notion</Badge>
            ) : (
              <Badge tone="err">Notion error</Badge>
            )}
            <span className="rounded-full border border-os-border px-4 py-2 text-[13px] text-os-muted">read-only · refreshes 60s</span>
            {hubUrl && (
              <a
                href={hubUrl}
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-2 rounded-full border border-[var(--accent-line)] bg-[var(--accent-soft)] px-4 py-2 text-[13px] text-os-accent transition-transform hover:-translate-y-[1px]"
              >
                Open in Notion <ExternalLink size={12} strokeWidth={1.8} />
              </a>
            )}
            <button
              onClick={refresh}
              aria-label="Refresh"
              className="pressable grid h-10 w-10 place-items-center rounded-full border border-os-border text-os-muted hover:border-os-border-strong hover:text-os-text"
            >
              <RefreshCw size={15} strokeWidth={1.7} />
            </button>
          </div>
        </div>

        {/* Hero row: funnel + volume card */}
        <div className="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
          <Card delay={120} className="pb-4">
            <CardHead title="Pipeline" onRefresh={refresh} hubUrl={hubUrl} />
            <PipelineChart
              stages={stages}
              active={FILTER_TO_STAGE_IDX[stageFilter]}
              onSelect={(stage) => setStageFilter(stage.filter)}
              searchSlot={
                <>
                  <div className="mb-2 flex items-center gap-2 px-1 text-[13px] text-os-muted">
                    <Sparkles size={14} strokeWidth={1.7} className="text-os-accent" />
                    What are you looking for?
                  </div>
                  <label className="flex items-center gap-2 rounded-[12px] border border-os-border bg-os-bg px-3.5 py-2.5">
                    <Search size={14} strokeWidth={1.7} className="shrink-0 text-os-dim" />
                    <input
                      value={query}
                      onChange={(e) => setQuery(e.target.value)}
                      placeholder="Filter deals: a brand, a contact, a channel, or /talks /production /paid /declined /s"
                      className="w-full bg-transparent text-[13.5px] outline-none placeholder:text-os-dim"
                    />
                    {slash && (
                      <span
                        className="shrink-0 rounded-md border px-2 py-0.5 font-mono text-[11.5px]"
                        style={{ borderColor: 'color-mix(in oklab, var(--warn) 45%, transparent)', background: 'color-mix(in oklab, var(--warn) 12%, transparent)', color: 'var(--warn)' }}
                      >
                        {slash}
                      </span>
                    )}
                    <span className="h-[15px] w-[1.5px] shrink-0 bg-os-accent" style={{ animation: CARET_ANIMATION }} />
                  </label>
                </>
              }
            />
          </Card>

          <Card delay={220} className="flex flex-col">
            <CardHead title="Deal Volume" onRefresh={refresh} hubUrl={hubUrl} />
            <div className="flex flex-1 flex-col px-6 pb-6">
              <div className="mt-3 flex flex-wrap items-center gap-2.5">
                <span className="text-[50px] font-semibold leading-none tracking-[-0.035em] tabular-nums">
                  <DollarStat value={vol.openUsd} />
                </span>
                {vol.paidUsd > 0 && (
                  <span className="inline-flex items-center gap-1.5 rounded-full border border-os-border px-2.5 py-1 font-mono text-[11.5px] tabular-nums text-os-muted">
                    <span className="h-1.5 w-1.5 rounded-full bg-os-ok" /> {fmtUsd(vol.paidUsd)} paid
                  </span>
                )}
                {vol.declinedUsd > 0 && (
                  <span className="inline-flex items-center gap-1.5 rounded-full border border-os-border px-2.5 py-1 font-mono text-[11.5px] tabular-nums text-os-muted">
                    <span className="h-1.5 w-1.5 rounded-full bg-os-err" /> {fmtUsd(vol.declinedUsd)} declined
                  </span>
                )}
              </div>
              <div className="mb-5 mt-2 text-[13px] text-os-dim">
                open pipeline across {openCount} deal{openCount === 1 ? '' : 's'} · {vol.quotedDeals} of {deals.length} priced
              </div>
              <div className="flex flex-1 flex-col justify-around gap-6 border-t border-os-border pt-5">
                <VolumeMeter label={`In talks (${vol.counts.talks})`} frac={vol.openUsd > 0 ? vol.talksUsd / vol.openUsd : 0} display={fmtUsd(vol.talksUsd)} hue={HUE.amber} delay={500} />
                <VolumeMeter label={`In production (${vol.counts.production})`} frac={vol.openUsd > 0 ? vol.productionUsd / vol.openUsd : 0} display={fmtUsd(vol.productionUsd)} hue={HUE.cobalt} delay={650} />
                <VolumeMeter
                  label={`Paid vs declined (${vol.counts.paid}/${vol.counts.declined})`}
                  frac={vol.paidUsd + vol.declinedUsd > 0 ? vol.paidUsd / (vol.paidUsd + vol.declinedUsd) : 0}
                  display={`${fmtUsd(vol.paidUsd)} / ${fmtUsd(vol.declinedUsd)}`}
                  hue={HUE.accent}
                  delay={800}
                />
              </div>
              <div className="mt-5 border-t border-os-border pt-3 text-center font-mono text-[10.5px] tracking-[0.1em] text-os-dim">
                open = talks + production · {vol.counts.tierS} S-tier
              </div>
            </div>
          </Card>
        </div>

        {/* Second row: activity line + sizes + THE gradient card */}
        <div className="mt-6 grid grid-cols-[1fr_1fr_1fr] gap-6 max-[1200px]:grid-cols-1">
          <Card delay={320}>
            <CardHead title="Deal Activity" onRefresh={refresh} hubUrl={hubUrl} />
            <div className="px-6 pt-3">
              <span className="text-[30px] font-semibold tabular-nums tracking-[-0.03em]">
                <StatNumber value={editsInWindow} />
              </span>
              <span className="ml-2 text-[13px] text-os-dim">Notion edits, last 30 days</span>
            </div>
            <StepLine series={series} hue={HUE.activity} empty="No edits in this window." />
          </Card>

          <Card delay={420}>
            <CardHead title="Deal Sizes" onRefresh={refresh} hubUrl={hubUrl} />
            <div className="flex items-end justify-between gap-4 px-6 pb-6 pt-3">
              <div>
                <div className="text-[30px] font-semibold tabular-nums tracking-[-0.03em]">
                  <StatNumber value={vol.quotedDeals} />
                </div>
                <div className="mt-1 text-[13px] text-os-dim">priced deals</div>
                <div className="mt-4 rounded-full border border-os-border px-3 py-1 text-[12px] text-os-muted">
                  Largest: <span className="font-semibold tabular-nums">{fmtUsd(largest)}</span>
                </div>
              </div>
              <DotMatrix cols={sizes} hue={HUE.cobalt} />
            </div>
          </Card>

          {/* the ONE gradient insight card */}
          <Card
            delay={520}
            className="overflow-hidden !border-transparent"
            style={{
              background: [
                'radial-gradient(120% 90% at 85% 8%, color-mix(in oklab, var(--tile-glow-a) 55%, transparent), transparent 60%)',
                'radial-gradient(130% 110% at 12% 92%, color-mix(in oklab, var(--tile-glow-b) 55%, transparent), transparent 62%)',
                'radial-gradient(110% 110% at 55% 55%, color-mix(in oklab, var(--tile-glow-c) 45%, transparent), transparent 70%)',
                'var(--insight-base)',
              ].join(', '),
              backgroundSize: '160% 160%',
              animation: `bd-rise .6s ${EASE} 520ms backwards, bd-drift 14s ease-in-out 1s infinite alternate`,
            }}
          >
            <div className="pointer-events-none absolute inset-0" style={{ backgroundImage: GRAIN, mixBlendMode: 'overlay', opacity: 0.85 }} />
            <div className="pointer-events-none absolute -right-14 -top-16 h-56 w-56 rotate-[24deg] rounded-[36px] border border-white/25 bg-white/5" style={{ backdropFilter: 'blur(3px)' }} />
            <div className="relative flex h-full flex-col px-6 py-5 text-white">
              <span className="inline-flex w-fit items-center gap-1.5 rounded-full bg-white/15 px-3 py-1 text-[12px] backdrop-blur">
                <Lightbulb size={13} strokeWidth={1.7} /> Needs you this week
              </span>
              <div className="mt-4 text-[64px] font-semibold leading-none tracking-[-0.03em] tabular-nums">
                <StatNumber value={needsYou} />
              </div>
              <div className="mt-2 text-[16px] font-semibold leading-snug">
                {due.followUps.length} follow-up{due.followUps.length === 1 ? '' : 's'} due · {due.deadlines.length} deadline{due.deadlines.length === 1 ? '' : 's'} inside 7 days.
              </div>
              <div className="mt-1.5 text-[12.5px] leading-relaxed text-white/75">
                {[...due.followUps, ...due.deadlines]
                  .slice(0, 3)
                  .map((d) => d.brand)
                  .join(' · ') || 'Nothing due. The pipeline is waiting on brands, not on you.'}
              </div>
              <div className="mt-auto flex gap-1.5 pt-4">
                {Array.from({ length: 8 }, (_, i) => (
                  <span key={i} className="h-[3px] flex-1 rounded-full" style={{ background: i < Math.round((needsYou / Math.max(openCount, 1)) * 8) ? '#fff' : 'rgba(255,255,255,.25)' }} />
                ))}
              </div>
            </div>
          </Card>
        </div>

        {/* Deal list */}
        <Card delay={620} className="mt-6">
          <div className="flex flex-wrap items-center justify-between gap-4 px-6 pt-5">
            <h2 className="text-[19px] font-semibold tracking-[-0.01em]">
              Deals{' '}
              <span className="ml-1 font-mono text-[12px] font-normal tabular-nums text-os-dim">
                {filtered.length} of {deals.length}
              </span>
            </h2>
            <div className="flex flex-wrap items-center gap-2">
              {(
                [
                  ['all', `All ${vol.counts.all}`],
                  ['talks', `In talks ${vol.counts.talks}`],
                  ['production', `In production ${vol.counts.production}`],
                  ['paid', `Paid ${vol.counts.paid}`],
                  ['declined', `Declined ${vol.counts.declined}`],
                  ['tier-s', `S-tier ${vol.counts.tierS}`],
                ] as Array<[Filter, string]>
              ).map(([key, label]) => (
                <button
                  key={key}
                  onClick={() => setStageFilter(key)}
                  className={`pressable rounded-full px-4 py-1.5 text-[12.5px] ${
                    stageFilter === key ? 'bg-os-accent font-semibold text-os-ink' : 'border border-os-border text-os-muted hover:text-os-text'
                  }`}
                >
                  {label}
                </button>
              ))}
            </div>
          </div>
          <div className="mt-4 border-t border-os-border">
            {filtered.map((d) => {
              const usd = dealUsd(d);
              return (
                <button
                  key={d.id}
                  onClick={() => setSelectedId(d.id)}
                  className="pressable grid w-full grid-cols-[220px_1fr_auto_auto_auto] items-center gap-5 border-b border-os-border px-6 py-3.5 text-left last:border-0 hover:bg-[color-mix(in_oklab,var(--text)_4%,transparent)] max-[1000px]:grid-cols-[1fr_auto]"
                >
                  <div className="min-w-0">
                    <div className="truncate text-[13.5px] font-medium">{d.brand}</div>
                    <div className="truncate font-mono text-[11px] text-os-dim">{d.contactName ?? d.contactEmail ?? d.source ?? 'no contact yet'}</div>
                  </div>
                  <div className="min-w-0 max-[1000px]:hidden">
                    <div className="truncate text-[12.5px] text-os-muted">{[d.videoType, d.mainChannel].filter(Boolean).join(' on ') || d.status}</div>
                  </div>
                  <div className="flex items-center gap-2">
                    {usd > 0 && (
                      <span
                        className="rounded-full px-2.5 py-0.5 font-mono text-[10.5px] tabular-nums tracking-[0.04em]"
                        style={{ background: 'var(--accent-soft)', color: 'var(--accent)' }}
                        title={d.amountAgreedUsd != null ? 'amount agreed' : d.dealValueUsd != null ? 'deal value' : 'brand budget'}
                      >
                        {fmtUsd(usd)}
                      </span>
                    )}
                    {(d.tier ?? '').toUpperCase() === 'S' && (
                      <span className="rounded-full px-2.5 py-0.5 font-mono text-[10.5px] uppercase tracking-[0.12em]" style={{ background: 'color-mix(in oklab, var(--ramp-4) 18%, transparent)', color: HUE.violet }}>
                        S-tier
                      </span>
                    )}
                    <span className="rounded-full px-2.5 py-0.5 font-mono text-[10.5px] uppercase tracking-[0.12em]" style={statusStyle(d)}>
                      {d.status}
                    </span>
                  </div>
                  <span className="inline-flex w-[72px] items-center gap-1.5 font-mono text-[11px] tabular-nums text-os-dim max-[1000px]:hidden">
                    {d.followUpDate && (
                      <>
                        <CalendarClock size={12} strokeWidth={1.7} />
                        {d.followUpDate.slice(5)}
                      </>
                    )}
                  </span>
                  <span className="w-[64px] text-right font-mono text-[11px] text-os-dim max-[1000px]:hidden">{timeAgo(d.lastEdited)}</span>
                </button>
              );
            })}
            {filtered.length === 0 && <div className="px-6 py-8 text-center text-[12.5px] text-os-dim">Nothing matches that filter.</div>}
          </div>
        </Card>
      </div>

      {/* Detail drawer */}
      {selected && <div className="fixed inset-0 z-[60] bg-black/45 backdrop-blur-[2px]" onClick={() => setSelectedId(null)} aria-hidden="true" />}
      {selected && (
        <aside className="fixed right-0 top-0 z-[70] flex h-full w-[460px] max-w-[92vw] flex-col border-l border-os-border bg-os-bg2/95 backdrop-blur">
          <div className="flex items-start justify-between gap-4 border-b border-os-border px-6 py-5">
            <div className="min-w-0">
              <div className="truncate text-[17px] font-semibold">{selected.brand}</div>
              <div className="mt-0.5 truncate font-mono text-[11.5px] text-os-dim">{selected.contactEmail ?? selected.contactName ?? 'no contact on the row'}</div>
              <div className="mt-2.5 flex flex-wrap items-center gap-2">
                {selected.tier && <Badge tone={(selected.tier ?? '').toUpperCase() === 'S' ? 'accent' : 'default'}>tier {selected.tier}</Badge>}
                <Badge tone={badgeTone(selected)}>{selected.status}</Badge>
                {selected.paidInFull && <Badge tone="ok">paid in full</Badge>}
                {selected.seeded && <Badge tone="warn">example row</Badge>}
              </div>
            </div>
            <button
              onClick={() => setSelectedId(null)}
              aria-label="Close"
              className="pressable grid h-[30px] w-[30px] shrink-0 place-items-center rounded-sm-t border border-os-border text-os-muted hover:border-os-border-strong hover:text-os-text"
            >
              <X size={15} strokeWidth={1.8} />
            </button>
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto px-6 py-5">
            <section className="mb-6">
              <a href={selected.notionUrl} target="_blank" rel="noreferrer" className={BTN_SOLID}>
                <ExternalLink size={13} strokeWidth={1.8} /> Open in Notion
              </a>
              <p className="mt-2.5 text-[11px] text-os-dim">Notion is the source of truth. Edit the deal there; this board rereads it on refresh or within 20 minutes.</p>
            </section>
            <section className="mb-6">
              <Label rule>Money</Label>
              <div className="mt-2">
                <Field k="Amount agreed" v={selected.amountAgreedUsd != null ? fmtUsd(selected.amountAgreedUsd) : null} />
                <Field k="Deal value" v={selected.dealValueUsd != null ? fmtUsd(selected.dealValueUsd) : null} />
                <Field k="Brand budget" v={selected.budgetUsd != null ? fmtUsd(selected.budgetUsd) : null} />
                <Field k="Suggested rate" v={selected.suggestedRateUsd != null ? fmtUsd(selected.suggestedRateUsd) : null} />
              </div>
            </section>
            <section className="mb-6">
              <Label rule>Dates</Label>
              <div className="mt-2">
                <Field k="Follow-up" v={selected.followUpDate} />
                <Field k="Deadline" v={selected.deadline} />
                <Field k="Last edited" v={`${timeAgo(selected.lastEdited)} (${selected.lastEdited.slice(0, 10)})`} />
              </div>
            </section>
            <section>
              <Label rule>Fit</Label>
              <div className="mt-2">
                <Field k="Channel" v={selected.mainChannel} />
                <Field k="Format" v={selected.videoType} />
                <Field k="Source" v={selected.source} />
                <Field k="ICP fit" v={selected.icpFit} />
              </div>
            </section>
          </div>
        </aside>
      )}
    </div>
  );
}
