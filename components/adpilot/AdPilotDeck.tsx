'use client';

import { useCallback, useEffect, useMemo, useRef, useState, type CSSProperties, type KeyboardEvent } from 'react';
import dynamic from 'next/dynamic';
import { useSearchParams } from 'next/navigation';
import { ArrowDownRight, ArrowUpRight, ChevronLeft, ChevronRight, Search, X } from 'lucide-react';
import type { Campaign, DailyPoint } from '@/lib/adpilot';
import {
  aggregateAudience,
  aggregateCampaigns,
  aggregateDaily,
  aggregateGeo,
  alignDaily,
  campaignMetrics,
  deliveryAlpha,
  isStale,
  periodDays,
  relativeAgo,
  rovingIndex,
  type CampaignMetrics,
} from '@/lib/adpilot';
import { SetupLink } from '@/components/SetupLink';
import { setupHrefFor } from '@/lib/integrations-catalog';
import { patchUrlParams, prefersReducedMotion } from '@/components/adpilot/url';
import { cityBreakdown, cityKey, type CityBreakdown } from '@/lib/adpilot-globe';
import { angleBetween } from '@/components/adpilot/projection';
import { pushEscape } from '@/lib/escape-stack';

/**
 * The AdPilot command deck: status chip → globe (naked over the page grid,
 * glowing stat widgets floating on the right) → snap-scrolling campaign rail
 * (built for N campaigns) → five composed widgets: Spend, Delivery
 * (dot-matrix, same visual language as the globe), ROAS ring, Results,
 * Audience. One shared selection drives everything and lives in the URL
 * (`?campaign=`), so reload, back/forward and deep links keep it. Every
 * colour is a token (`.ap-*` classes in app/globals.css); the layout follows
 * the deck's own width through container queries (SPEC X4).
 */

const Globe = dynamic(() => import('@/components/adpilot/Globe').then((m) => m.Globe), {
  ssr: false,
  loading: () => <div className="h-[560px] w-full" />,
});

const nf = new Intl.NumberFormat('en-US');
const money = (v: number) => `$${nf.format(Math.round(v))}`;
const money2 = (v: number | null) => (v === null ? '—' : `$${v.toFixed(2)}`);
const times = (v: number | null) => (v === null ? '—' : `${v.toFixed(2)}x`);
const pct = (v: number | null) => (v === null ? '—' : `${(v * 100).toFixed(2)}%`);
const alpha = (a: number) => ({ '--ap-a': a }) as CSSProperties;

/** The rail shows a type-to-filter affordance once campaigns outgrow it. */
const RAIL_FILTER_THRESHOLD = 10;
/** Deck width at which the floaters sit beside the globe (matches globals.css). */
const WIDE_DECK = 960;
const NARROW_DECK = 640;

function useElementWidth<T extends HTMLElement>() {
  const ref = useRef<T>(null);
  const [width, setWidth] = useState<number | null>(null);
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const measure = () => setWidth(el.clientWidth);
    measure();
    const ro = new ResizeObserver(measure);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);
  return [ref, width] as const;
}

export function AdPilotDeck({
  campaigns,
  syncedAt,
  now,
  filming,
}: {
  campaigns: Campaign[];
  syncedAt: string | null;
  /** Server render time, so the relative "synced" text matches on hydration. */
  now: number;
  /** The campaigns file is the staged Filming one: label it, never dress it as real. */
  filming: boolean;
}) {
  const params = useSearchParams();
  const urlCampaign = params.get('campaign');
  const known = urlCampaign !== null && campaigns.some((c) => c.id === urlCampaign);
  const selected = known ? (urlCampaign as string) : 'all';

  // A stale or unknown ?campaign= falls back to all and leaves the URL (S01 §8 edge 7).
  useEffect(() => {
    if (urlCampaign !== null && !known) patchUrlParams({ campaign: null });
  }, [urlCampaign, known]);

  const select = useCallback((id: string) => {
    patchUrlParams({ campaign: id === 'all' ? null : id });
  }, []);
  const toggle = useCallback((id: string) => select(selected === id ? 'all' : id), [select, selected]);

  const active = useMemo(
    () => (selected === 'all' ? campaigns : campaigns.filter((c) => c.id === selected)),
    [campaigns, selected],
  );
  const empty = campaigns.length === 0;

  const metrics = useMemo(
    () => (active.length === 1 ? campaignMetrics(active[0]) : aggregateCampaigns(active)),
    [active],
  );
  const geo = useMemo(() => aggregateGeo(active), [active]);

  // The focused city (S11 F-G3): its card sits in the floater column. It stays
  // open across campaign switches while the city is in the selection (Q-AP-8)
  // and clears when it is not (S11 §8 edge 5).
  const [cityFocus, setCityFocus] = useState<string | null>(null);
  const focusInGeo = cityFocus !== null && geo.some((g) => cityKey(g) === cityFocus);
  useEffect(() => {
    if (cityFocus !== null && !focusInGeo) setCityFocus(null);
  }, [cityFocus, focusInGeo]);
  const city = useMemo(() => {
    if (!cityFocus || !focusInGeo) return null;
    const [name, country] = cityFocus.split('|');
    return cityBreakdown(active, name, country);
  }, [active, cityFocus, focusInGeo]);
  const cardRef = useRef<HTMLDivElement>(null);
  const closeCity = useCallback(() => {
    const inCard = !!cardRef.current && cardRef.current.contains(document.activeElement);
    setCityFocus(null);
    // Closing from inside the card returns focus to the globe (S11 §6).
    if (inCard) document.querySelector<HTMLElement>('[data-globe]')?.focus();
  }, []);
  useEffect(() => {
    if (!cityFocus) return;
    return pushEscape(() => closeCity(), { id: 'adpilot-city' });
  }, [cityFocus, closeCity]);
  // On a narrow deck the card drops below the globe: bring it into view by the
  // smallest scroll (a no-op when it is already visible, as on a wide deck).
  useEffect(() => {
    if (!cityFocus) return;
    cardRef.current?.scrollIntoView({ block: 'nearest', behavior: prefersReducedMotion() ? 'auto' : 'smooth' });
  }, [cityFocus]);
  const selectedName = selected === 'all' ? null : campaigns.find((c) => c.id === selected)?.name ?? null;
  const audience = useMemo(() => (active.length === 1 ? active[0].audience : aggregateAudience(active)), [active]);
  const daily = useMemo(() => aggregateDaily(active), [active]);
  const days = useMemo(() => periodDays(active), [active]);
  const windowLabel = days ? `${days} day${days === 1 ? '' : 's'}` : null;

  const [deckRef, deckWidth] = useElementWidth<HTMLDivElement>();
  const [vw, setVw] = useState<number | null>(null);
  useEffect(() => {
    const on = () => setVw(window.innerWidth);
    on();
    window.addEventListener('resize', on);
    return () => window.removeEventListener('resize', on);
  }, []);
  // ≥ WIDE: today's 560px hero. Narrower: clamp(320px, 60vw, 560px), stepped
  // by 40px so a window drag does not rebuild the globe on every pixel.
  const globeHeight =
    deckWidth === null || deckWidth >= WIDE_DECK || vw === null
      ? 560
      : Math.max(320, Math.min(560, Math.round((vw * 0.6) / 40) * 40));

  const resultsRef = useRef<HTMLDivElement>(null);
  const [ringResults, setRingResults] = useState(false);
  const showResults = () => {
    const el = resultsRef.current;
    if (!el) return;
    el.scrollIntoView({ behavior: prefersReducedMotion() ? 'auto' : 'smooth', block: 'center' });
    setRingResults(true);
    window.setTimeout(() => setRingResults(false), 1200);
  };

  return (
    <div ref={deckRef} className="ap-deck">
      <StatusChip campaigns={campaigns} syncedAt={syncedAt} now={now} filming={filming} />

      {/* The globe: no window; glowing stat widgets float on the right */}
      <div className="ap-globe-stage relative">
        <div aria-hidden className="ap-globe-glow" />
        {!empty && (
          <div className="ap-floaters">
            <StatFloat
              label={windowLabel ? `Total leads · ${windowLabel}` : 'Total leads'}
              value={nf.format(metrics.leads)}
              onClick={showResults}
            />
            <StatFloat label="Booked calls" value={nf.format(metrics.bookings)} onClick={showResults} />
            {city && (
              <CityCard
                city={city}
                nearby={geo.filter((g) => cityKey(g) !== city.key && angleBetween(g, city) < NEARBY_RAD)}
                selected={selected}
                onToggle={toggle}
                onClose={closeCity}
                cardRef={cardRef}
              />
            )}
          </div>
        )}
        <Globe
          geo={geo}
          height={globeHeight}
          focus={focusInGeo ? cityFocus : null}
          onFocus={setCityFocus}
          emptyLabel={selectedName ? `No lead locations for ${selectedName}` : 'No lead locations yet'}
        />
      </div>

      {empty ? (
        <div className="ap-panel mx-auto mt-4 max-w-xl px-4 py-6 text-center">
          <p className="text-[13px] text-os-muted">No ad account connected.</p>
          <p className="mt-1 text-[12px] text-os-dim">
            Campaigns arrive automatically once a Meta ad account is connected. Nothing here is added by hand.
          </p>
          <div className="mt-3 flex justify-center">
            <SetupLink href={setupHrefFor('meta-ads')} hint="Connect a Meta ad account" label="Connect Meta Ads" />
          </div>
        </div>
      ) : (
        <>
          <CampaignRail campaigns={campaigns} selected={selected} onSelect={select} narrow={deckWidth !== null && deckWidth < NARROW_DECK} />

          {/* The metric deck: five composed widgets, room to breathe */}
          <div className="ap-deck-grid">
            <div className="ap-cell-spend">
              <HeroSpend
                campaigns={campaigns}
                active={active}
                metrics={metrics}
                daily={daily}
                selected={selected}
                windowLabel={windowLabel}
                onToggle={toggle}
              />
            </div>
            <div className="ap-cell-delivery">
              <DeliveryMatrix campaigns={campaigns} selected={selected} onToggle={toggle} />
            </div>
            <div className="ap-cell-roas">
              <RoasRing metrics={metrics} />
            </div>
            <div className="ap-cell-results">
              <ResultsPanel metrics={metrics} panelRef={resultsRef} ring={ringResults} />
            </div>
            <div className="ap-cell-audience">
              <AudiencePanel audience={audience} />
            </div>
          </div>
        </>
      )}
    </div>
  );
}

/* ------------------------------- status chip ------------------------------- */

function StatusChip({
  campaigns,
  syncedAt,
  now,
  filming,
}: {
  campaigns: Campaign[];
  syncedAt: string | null;
  now: number;
  filming: boolean;
}) {
  const empty = campaigns.length === 0;
  const liveCount = campaigns.filter((c) => c.status === 'active').length;
  const stale = isStale(syncedAt, now);
  const dot = empty || liveCount === 0 ? 'bg-os-dim' : stale ? 'bg-os-warn' : 'ap-solid';
  return (
    <div className="mb-6 flex justify-center">
      <div
        className="ap-chip flex flex-wrap items-center justify-center gap-x-2 gap-y-1 px-5 py-2"
        title={syncedAt ? `Last ad-account sync ${syncedAt}` : undefined}
      >
        <span aria-hidden className={`h-2 w-2 rounded-full ${dot}`} />
        {empty ? (
          <>
            <span className="text-[13px] text-os-text">No ad account connected</span>
            <SetupLink href={setupHrefFor('meta-ads')} hint="Connect a Meta ad account" />
          </>
        ) : (
          <span className="text-[13px] text-os-text">
            {liveCount} active campaign{liveCount === 1 ? '' : 's'}
            {syncedAt && (
              <span className="text-os-muted"> · synced {relativeAgo(syncedAt, now)}{stale ? ' (stale)' : ''}</span>
            )}
          </span>
        )}
        {filming && !empty && (
          <span className="rounded-full border border-os-border px-2 py-[1px] text-[10.5px] text-os-dim" title="Staged Filming data, not a live ad account">
            sample data
          </span>
        )}
      </div>
    </div>
  );
}

/** The glowing floaters beside the globe: a click shows the Results breakdown. */
function StatFloat({ label, value, onClick }: { label: string; value: string; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={`${label}: ${value}. Show results breakdown`}
      title="Show results breakdown"
      data-ap-floater
      className="ap-frost ap-lift block w-full px-4 py-3 text-left"
    >
      <div className="relative text-[11.5px] text-os-dim">{label}</div>
      <div className="relative mt-0.5 text-[24px] font-light tabular-nums text-os-text">{value}</div>
    </button>
  );
}

/** Pins closer than this on the sphere (about 6px at R 230) are "also here" on the City card (S11 §8 edge 1). */
const NEARBY_RAD = 0.026;

/**
 * The City card (S11 F-G3): a focused city's totals over the current
 * selection and one row per campaign that has it; a row selects (or, when
 * it is the selection, releases) that campaign, like the rail.
 */
function CityCard({
  city,
  nearby,
  selected,
  onToggle,
  onClose,
  cardRef,
}: {
  city: CityBreakdown;
  nearby: { city: string }[];
  selected: string;
  onToggle: (id: string) => void;
  onClose: () => void;
  cardRef: React.RefObject<HTMLDivElement>;
}) {
  const rate = city.bookingRate === null ? '—' : `${(city.bookingRate * 100).toFixed(1)}%`;
  const share = city.share === null ? '—' : `${(city.share * 100).toFixed(1)}%`;
  const facts: [string, string, string][] = [
    ['Leads', nf.format(city.leads), 'Leads from this city in the current selection'],
    ['Booked', nf.format(city.bookings), 'Booked calls from this city in the current selection'],
    ['Booking rate', rate, 'Booked calls divided by leads'],
    ['Share', share, 'This city’s share of all leads in the current selection'],
  ];
  return (
    <div ref={cardRef} data-ap-floater role="region" aria-label={`City: ${city.city}, ${city.country}`} className="ap-frost ap-city-card px-4 py-3">
      <div className="relative flex items-start justify-between gap-2">
        <div className="min-w-0">
          <div className="truncate text-[12.5px] text-os-text">{city.city}</div>
          <div className="truncate text-[11px] text-os-dim">{city.country}</div>
        </div>
        <button
          type="button"
          onClick={onClose}
          aria-label="Close city card"
          title="Close (Esc)"
          className="-mr-1.5 -mt-0.5 shrink-0 rounded-full p-1 text-os-dim hover:text-os-text"
        >
          <X className="h-3.5 w-3.5" strokeWidth={1.8} />
        </button>
      </div>
      <dl className="relative mt-2 grid grid-cols-2 gap-x-3 gap-y-1.5">
        {facts.map(([label, value, title]) => (
          <div key={label} title={title}>
            <dt className="text-[11px] text-os-dim">{label}</dt>
            <dd className="text-[24px] font-light leading-tight tabular-nums text-os-text">{value}</dd>
          </div>
        ))}
      </dl>
      {city.byCampaign.length > 0 && (
        <div className="ap-city-by ap-hair-soft relative mt-2.5 border-t pt-2">
          <div className="text-[11px] text-os-dim">By campaign</div>
          <div className="ap-city-rows -mx-1.5 mt-1 flex flex-col gap-0.5">
            {city.byCampaign.map((r) => (
              <button
                key={r.id}
                type="button"
                aria-pressed={selected === r.id}
                onClick={() => onToggle(r.id)}
                title={selected === r.id ? 'Show all campaigns' : `Show ${r.name}`}
                className="ap-row block w-full px-1.5 py-1 text-left"
              >
                <span className="block truncate text-[12px] text-os-muted">{r.name}</span>
                <span className="block text-[11px] tabular-nums text-os-dim">
                  {nf.format(r.leads)} leads · {nf.format(r.bookings)} booked
                </span>
              </button>
            ))}
          </div>
        </div>
      )}
      {nearby.length > 0 && (
        <div className="relative mt-1.5 text-[11px] text-os-dim">Also here: {nearby.map((n) => n.city).join(', ')}</div>
      )}
    </div>
  );
}

/* ------------------------------ campaign rail ------------------------------ */

function CampaignRail({
  campaigns,
  selected,
  onSelect,
  narrow,
}: {
  campaigns: Campaign[];
  selected: string;
  onSelect: (id: string) => void;
  narrow: boolean;
}) {
  const [filter, setFilter] = useState('');
  const scrollRef = useRef<HTMLDivElement>(null);
  const railRef = useRef<HTMLDivElement>(null);
  const [edges, setEdges] = useState({ left: false, right: false });
  const showFilter = campaigns.length > RAIL_FILTER_THRESHOLD;
  const q = filter.trim().toLowerCase();
  const ordered = [...campaigns]
    .sort((a, b) => Number(b.status === 'active') - Number(a.status === 'active'))
    // The selected campaign stays visible even when the filter would hide it.
    .filter((c) => !q || c.id === selected || c.name.toLowerCase().includes(q));
  const ids = ['all', ...ordered.map((c) => c.id)];
  const [focusId, setFocusId] = useState<string>(selected);
  const tabStop = ids.includes(focusId) ? focusId : ids.includes(selected) ? selected : 'all';

  const measure = useCallback(() => {
    const el = scrollRef.current;
    if (!el) return;
    const left = el.scrollLeft > 2;
    const right = el.scrollLeft + el.clientWidth < el.scrollWidth - 2;
    setEdges((e) => (e.left === left && e.right === right ? e : { left, right }));
  }, []);
  useEffect(() => {
    measure();
    const el = scrollRef.current;
    if (!el) return;
    const ro = new ResizeObserver(measure);
    ro.observe(el);
    return () => ro.disconnect();
  }, [measure, ordered.length]);

  const scrollBy = (dir: 1 | -1) => {
    const el = scrollRef.current;
    if (!el) return;
    el.scrollBy({ left: dir * el.clientWidth * 0.8, behavior: prefersReducedMotion() ? 'auto' : 'smooth' });
  };

  const onKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    const target = (e.target as HTMLElement).closest<HTMLElement>('[data-pill]');
    if (!target) return;
    const i = ids.indexOf(target.dataset.pill ?? '');
    const next = rovingIndex(e.key, i, ids.length);
    if (next === null || e.key === 'ArrowUp' || e.key === 'ArrowDown') return;
    e.preventDefault();
    const id = ids[next];
    setFocusId(id);
    const el = railRef.current?.querySelector<HTMLElement>(`[data-pill="${CSS.escape(id)}"]`);
    el?.focus();
    el?.scrollIntoView({ block: 'nearest', inline: 'nearest' });
  };

  const pill = (id: string, label: string, live: boolean | null) => {
    const isActive = selected === id;
    const paused = live === false;
    return (
      <button
        key={id}
        type="button"
        data-pill={id}
        aria-pressed={isActive}
        tabIndex={tabStop === id ? 0 : -1}
        onFocus={() => setFocusId(id)}
        onClick={() => onSelect(id)}
        title={paused ? `${label} · paused` : label}
        className={`ap-pill relative flex shrink-0 snap-start items-center gap-2 rounded-full border px-3.5 py-2 text-[12.5px] transition-colors ${
          isActive ? 'text-os-text' : 'text-os-dim hover:text-os-muted'
        }`}
      >
        {live !== null && <span aria-hidden className={`h-1.5 w-1.5 rounded-full ${live ? 'ap-solid' : 'bg-os-dim'}`} />}
        <span className="max-w-[220px] truncate">{label}</span>
        {paused && <span className="sr-only">(paused)</span>}
      </button>
    );
  };

  const fade = edges.left && edges.right ? 'both' : edges.left ? 'left' : edges.right ? 'right' : undefined;
  const showLeft = narrow || edges.left;
  const showRight = narrow || edges.right;

  return (
    <div ref={railRef} role="group" aria-label="Campaigns" onKeyDown={onKeyDown} className="ap-panel mt-6 flex items-center gap-2 p-2">
      {pill('all', 'All campaigns', null)}
      <div aria-hidden className="ap-rule h-6 w-px shrink-0" />
      {showLeft && (
        <button
          type="button"
          tabIndex={-1}
          onClick={() => scrollBy(-1)}
          disabled={!edges.left}
          aria-label="Scroll campaigns left"
          title={edges.left ? 'Scroll campaigns left' : 'Already at the start'}
          className="shrink-0 rounded-full p-1 text-os-dim hover:text-os-text disabled:opacity-30"
        >
          <ChevronLeft className="h-4 w-4" strokeWidth={1.8} />
        </button>
      )}
      <div ref={scrollRef} onScroll={measure} data-fade={fade} className="ap-rail-scroll flex min-w-0 flex-1 snap-x gap-2 overflow-x-auto">
        {ordered.map((c) => pill(c.id, c.name, c.status === 'active'))}
        {ordered.length === 0 && <span className="self-center px-2 text-[12px] text-os-dim">No campaign matches “{filter.trim()}”.</span>}
      </div>
      {showRight && (
        <button
          type="button"
          tabIndex={-1}
          onClick={() => scrollBy(1)}
          disabled={!edges.right}
          aria-label="Scroll campaigns right"
          title={edges.right ? 'Scroll campaigns right' : 'Already at the end'}
          className="shrink-0 rounded-full p-1 text-os-dim hover:text-os-text disabled:opacity-30"
        >
          <ChevronRight className="h-4 w-4" strokeWidth={1.8} />
        </button>
      )}
      {showFilter && (
        <div className="flex shrink-0 items-center gap-1.5 pl-1">
          <Search aria-hidden className="h-3.5 w-3.5 text-os-dim" strokeWidth={1.7} />
          <input
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Escape' && filter) {
                e.preventDefault();
                setFilter('');
              }
            }}
            aria-label="Filter campaigns"
            placeholder="filter"
            className="w-24 bg-transparent text-[12px] text-os-text outline-none placeholder:text-os-dim"
          />
        </div>
      )}
    </div>
  );
}

/* ------------------------------- hero spend -------------------------------- */

function windowDelta(daily: DailyPoint[], pick: (d: DailyPoint) => number): number | null {
  if (daily.length < 14) return null;
  const last7 = daily.slice(-7).reduce((s, d) => s + pick(d), 0);
  const prior7 = daily.slice(-14, -7).reduce((s, d) => s + pick(d), 0);
  if (prior7 <= 0) return null;
  return (last7 - prior7) / prior7;
}

function DeltaPill({ label, delta }: { label: string; delta: number | null }) {
  if (delta === null) return null;
  const up = delta >= 0;
  const Icon = up ? ArrowUpRight : ArrowDownRight;
  return (
    <span
      className="ap-hair ap-inset flex items-center gap-1 rounded-full border px-2.5 py-1"
      title={`${label} over the last 7 days against the 7 days before`}
    >
      <Icon aria-hidden className={`h-3 w-3 ${up ? 'text-[var(--accent)]' : 'text-os-dim'}`} strokeWidth={1.8} />
      <span className="text-[11px] tabular-nums text-os-muted">
        {up ? '+' : ''}
        {(delta * 100).toFixed(0)}% {label} · 7d
      </span>
    </span>
  );
}

function HeroSpend({
  campaigns,
  active,
  metrics,
  daily,
  selected,
  windowLabel,
  onToggle,
}: {
  campaigns: Campaign[];
  active: Campaign[];
  metrics: CampaignMetrics;
  daily: DailyPoint[];
  selected: string;
  windowLabel: string | null;
  onToggle: (id: string) => void;
}) {
  const maxSpend = Math.max(...campaigns.map((c) => c.spend), 1);
  return (
    <div className="ap-panel gl-on h-full p-5">
      <div className="text-[12.5px] text-os-dim">{windowLabel ? `Ad spend · ${windowLabel}` : 'Ad spend'}</div>
      <div className="mt-1 text-[36px] font-light tabular-nums leading-none text-os-text">{money(metrics.spend)}</div>
      <div className="mt-2.5 flex flex-wrap gap-1.5">
        <DeltaPill label="spend" delta={windowDelta(daily, (d) => d.spend)} />
        <DeltaPill label="leads" delta={windowDelta(daily, (d) => d.leads)} />
      </div>
      <div className="-mx-2 mt-4 flex flex-col gap-1">
        {campaigns.map((c) => {
          const dimmed = selected !== 'all' && !active.some((a) => a.id === c.id);
          return (
            <button
              key={c.id}
              type="button"
              aria-pressed={selected === c.id}
              onClick={() => onToggle(c.id)}
              title={selected === c.id ? `Show all campaigns` : `Show ${c.name}`}
              className={`ap-row block w-full px-2 py-1 text-left ${dimmed ? 'opacity-35' : ''}`}
            >
              <span className="flex items-baseline justify-between gap-2">
                <span className="truncate text-[12.5px] text-os-muted">{c.name}</span>
                <span className="text-[12px] tabular-nums text-os-text">{money(c.spend)}</span>
              </span>
              <span className="ap-inset mt-1.5 block h-[7px] overflow-hidden rounded-full">
                <span className="ap-hatch block h-full rounded-full" style={{ width: `${(c.spend / maxSpend) * 100}%` }} />
              </span>
            </button>
          );
        })}
      </div>
    </div>
  );
}

/* ----------------------------- delivery matrix ----------------------------- */

type Hover = { col: number; row: number; x: number; y: number };

/**
 * Spend per day as a dot matrix: one row per campaign, one dot per date,
 * intensity = that day's share of the campaign's peak. Rows line up by date
 * (alignDaily), and hovering a dot lights its date in every row so
 * cross-campaign pacing reads at a glance.
 */
function DeliveryMatrix({
  campaigns,
  selected,
  onToggle,
}: {
  campaigns: Campaign[];
  selected: string;
  onToggle: (id: string) => void;
}) {
  const withDaily = useMemo(() => campaigns.filter((c) => (c.daily?.length ?? 0) > 0), [campaigns]);
  const { dates, rows } = useMemo(() => alignDaily(withDaily), [withDaily]);
  const panelRef = useRef<HTMLDivElement>(null);
  const [hover, setHover] = useState<Hover | null>(null);
  const timer = useRef<number | null>(null);
  useEffect(() => () => {
    if (timer.current) window.clearTimeout(timer.current);
  }, []);

  if (rows.length === 0) {
    return (
      <div className="ap-panel flex h-full items-center justify-center p-5">
        <span className="text-[12px] text-os-dim">Daily delivery arrives with the ad-account sync.</span>
      </div>
    );
  }

  const peakAll = rows
    .flatMap((r) => r.cells)
    .reduce<DailyPoint>((best, d) => (d && d.spend > best.spend ? d : best), { date: '', spend: 0, leads: 0, bookings: 0 });

  const enter = (e: React.MouseEvent<HTMLSpanElement>, row: number, col: number) => {
    const panel = panelRef.current;
    if (!panel) return;
    const p = panel.getBoundingClientRect();
    const r = e.currentTarget.getBoundingClientRect();
    const next = { col, row, x: r.left + r.width / 2 - p.left, y: r.top - p.top };
    if (timer.current) window.clearTimeout(timer.current);
    timer.current = window.setTimeout(() => setHover(next), hover ? 0 : 80);
  };
  const leave = () => {
    if (timer.current) window.clearTimeout(timer.current);
    timer.current = null;
    setHover(null);
  };
  const hovered = hover ? rows[hover.row]?.cells[hover.col] : null;
  const panelW = panelRef.current?.clientWidth ?? 0;

  return (
    <div ref={panelRef} className="ap-panel relative h-full p-5" onMouseLeave={leave}>
      <div className="flex items-baseline justify-between">
        <div className="text-[12.5px] text-os-dim">Delivery · spend per day</div>
        <div className="text-[11.5px] tabular-nums text-os-dim">
          {dates[0]?.slice(5)} — {dates[dates.length - 1]?.slice(5)}
        </div>
      </div>
      <div className="mt-3.5 flex flex-col gap-3">
        {rows.map(({ item: c, cells }, ri) => {
          const dimmed = selected !== 'all' && selected !== c.id;
          const rowPeak = Math.max(...cells.map((d) => d?.spend ?? 0), 1);
          return (
            <div key={c.id} className={dimmed ? 'opacity-30' : ''}>
              <button
                type="button"
                aria-pressed={selected === c.id}
                onClick={() => onToggle(c.id)}
                title={selected === c.id ? 'Show all campaigns' : `Show ${c.name}`}
                className="ap-row -mx-1.5 mb-1 flex w-[calc(100%+0.75rem)] items-baseline justify-between gap-2 px-1.5 py-0.5 text-left"
              >
                <span className="truncate text-[12.5px] text-os-muted">{c.name}</span>
                <span className="shrink-0 text-[11px] tabular-nums text-os-dim">peak {money(rowPeak)}/d</span>
              </button>
              <div
                role="img"
                aria-label={`${c.name}: spend per day, ${dates.length} days, peak ${money(rowPeak)} a day`}
                className="flex justify-between gap-[3px]"
              >
                {cells.map((d, ci) => {
                  const a = d ? deliveryAlpha(d.spend / rowPeak) : 0;
                  return (
                    <span
                      key={dates[ci]}
                      onMouseEnter={d ? (e) => enter(e, ri, ci) : undefined}
                      data-hot={hover?.col === ci ? '' : undefined}
                      className={`ap-dot aspect-square w-full max-w-[13px] rounded-full ${a > 0 ? 'ap-fill' : 'ap-track'}`}
                      style={a > 0 ? alpha(a) : undefined}
                    />
                  );
                })}
              </div>
            </div>
          );
        })}
      </div>
      <div className="ap-hair-soft mt-4 flex items-center justify-between border-t pt-3">
        <span className="text-[11.5px] text-os-dim">
          Peak day {peakAll.date.slice(5)} · {money(peakAll.spend)}
        </span>
        <span className="flex items-center gap-1.5 text-[11px] text-os-dim">
          quiet
          {[0.15, 0.45, 0.7, 1].map((t) => (
            <span key={t} aria-hidden className="ap-fill h-[9px] w-[9px] rounded-full" style={alpha(deliveryAlpha(t))} />
          ))}
          heavy
        </span>
      </div>
      {hover && hovered && (
        <div
          role="tooltip"
          className="ap-hovercard px-2.5 py-1.5 text-[11.5px]"
          style={{
            left: Math.min(Math.max(hover.x, 70), Math.max(70, panelW - 70)),
            top: hover.y - 6,
            transform: 'translate(-50%, -100%)',
          }}
        >
          <div className="tabular-nums text-os-text">{hovered.date}</div>
          <div className="tabular-nums text-os-muted">
            {money(hovered.spend)} · {nf.format(hovered.leads)} lead{hovered.leads === 1 ? '' : 's'}
          </div>
        </div>
      )}
    </div>
  );
}

/* -------------------------------- ROAS ring -------------------------------- */

function RoasRing({ metrics }: { metrics: CampaignMetrics }) {
  const roas = metrics.roas;
  const CAP = 4;
  const R = 56;
  const C = 2 * Math.PI * R;
  const frac = roas === null ? 0 : Math.min(roas / CAP, 1);
  const tickAngle = (1 / CAP) * 360 - 90;
  return (
    <div className="ap-panel flex h-full flex-col items-center justify-center p-5">
      <div className="self-start text-[12.5px] text-os-dim">ROAS · revenue vs spend</div>
      <div className="relative mt-1">
        <svg width="150" height="150" viewBox="0 0 150 150" role="img" aria-label={`ROAS ${times(roas)}`}>
          <title>
            {`Return on ad spend ${times(roas)}. Ring scale 0 to ${CAP}x; the tick marks 1.00x, where revenue equals spend.`}
          </title>
          <circle cx="75" cy="75" r={R} fill="none" className="ap-stroke-track" strokeWidth="10" />
          {frac > 0 && (
            <circle
              cx="75"
              cy="75"
              r={R}
              fill="none"
              className="ap-stroke-signal"
              strokeWidth="10"
              strokeLinecap="round"
              strokeDasharray={`${(frac * C).toFixed(1)} ${C.toFixed(1)}`}
              transform="rotate(-90 75 75)"
            />
          )}
          <line x1="75" y1="11" x2="75" y2="21" className="ap-stroke-tick" strokeWidth="2" transform={`rotate(${tickAngle + 90} 75 75)`} />
        </svg>
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <span className={`text-[27px] font-light tabular-nums ${roas !== null && roas >= 1 ? 'text-[var(--accent)]' : 'text-os-text'}`}>
            {times(roas)}
          </span>
          <span className="text-[10.5px] text-os-dim">break-even 1.00x</span>
        </div>
      </div>
      <div className="mt-1.5 flex flex-wrap justify-center gap-x-4 text-[11.5px] tabular-nums text-os-dim">
        <span>rev {money(metrics.revenue)}</span>
        <span>spend {money(metrics.spend)}</span>
      </div>
      <div className="mt-0.5 text-[10.5px] text-os-dim">scale 0 to {CAP}x</div>
    </div>
  );
}

/* --------------------------------- results --------------------------------- */

function ResultsPanel({
  metrics,
  panelRef,
  ring,
}: {
  metrics: CampaignMetrics;
  panelRef: React.RefObject<HTMLDivElement>;
  ring: boolean;
}) {
  const items: [string, string][] = [
    ['Leads', nf.format(metrics.leads)],
    ['Bookings', nf.format(metrics.bookings)],
    ['Cost / lead', money2(metrics.cpl)],
    ['Cost / booking', money2(metrics.costPerBooking)],
    ['Cost / result', money2(metrics.costPerResult)],
    ['CTR', pct(metrics.ctr)],
  ];
  return (
    <div ref={panelRef} id="ap-results" data-ring={ring ? '1' : undefined} className="ap-panel ap-results ap-ring h-full p-5">
      <div className="text-[12.5px] text-os-dim">Results</div>
      <div className="ap-results-grid mt-3">
        {items.map(([label, value]) => (
          <div key={label} className="flex items-center gap-2.5">
            <span aria-hidden className="ap-solid h-8 w-[3px] shrink-0 rounded-full opacity-70" />
            <div className="min-w-0">
              <div className="whitespace-nowrap text-[11.5px] text-os-dim">{label}</div>
              <div className="text-[20px] font-light tabular-nums text-os-text">{value}</div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

/* --------------------------------- audience -------------------------------- */

function AudiencePanel({ audience }: { audience: Campaign['audience'] }) {
  const groups: [string, Record<string, number>][] = [
    ['Age', audience.age],
    ['Gender', audience.gender],
    ['Placements', audience.placements],
  ];
  const SEGMENTS = 18;
  return (
    <div className="ap-panel ap-audience h-full p-5">
      <div className="text-[12.5px] text-os-dim">Audience</div>
      <div className="ap-audience-grid mt-3">
        {groups.map(([title, data]) => (
          <div key={title} className={title === 'Age' ? 'ap-audience-age' : ''}>
            <div className="text-[11px] text-os-dim">{title}</div>
            <div className="ap-seg-rows mt-1.5">
              {Object.entries(data).map(([key, value]) => {
                const filled = Math.round((value / 100) * SEGMENTS);
                return (
                  <div key={key} className="contents">
                    <span className="truncate text-[12px] text-os-muted" title={key}>
                      {key}
                    </span>
                    <span role="img" aria-label={`${key} ${value.toFixed(0)}%`} className="flex gap-[2.5px]">
                      {Array.from({ length: SEGMENTS }, (_, i) => (
                        <span key={i} className={`h-[10px] flex-1 rounded-[2px] ${i < filled ? 'ap-fill' : 'ap-track'}`} />
                      ))}
                    </span>
                    <span className="text-right text-[11px] tabular-nums text-os-dim">{value.toFixed(0)}%</span>
                  </div>
                );
              })}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
