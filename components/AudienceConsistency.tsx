'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import { Maximize2, X } from 'lucide-react';
import {
  bucketPostingByDay,
  dateAxis,
  forwardFillCarried,
  platformsPresent,
  type PostDay,
  type PostingActivityDay,
} from '@/lib/posting-activity';
import { channelColor } from '@/lib/series';
import { useMeltEnd, useMeltPhase } from '@/lib/use-melt';
import { Layer } from '@/components/g/Overlays';
import { channelsNetChange, rangeWindow } from '@/lib/social-range';

type Range = 7 | 30 | 60 | 'all';
const RANGES: Range[] = [7, 30, 60, 'all'];
const RANGE_LABEL: Record<string, string> = { '7': '7d', '30': '30d', '60': '60d', all: 'All' };

type AudPoint = { date: string; value: number };

// Stack/legend order (a Facebook entry the server palette doesn't track sorts
// last). Colours come from the one channel -> series map (lib/series.ts, SPEC
// C19), the same slots /social's pop-outs, /analytics and Home use.
const PREFERRED = ['instagram', 'tiktok', 'twitter', 'youtube', 'linkedin', 'facebook'];
const LABELS: Record<string, string> = {
  instagram: 'Instagram',
  tiktok: 'TikTok',
  twitter: 'Twitter / X',
  youtube: 'YouTube',
  linkedin: 'LinkedIn',
  facebook: 'Facebook',
};
const colorFor = (p: string): string => channelColor(p);
const labelFor = (p: string): string => LABELS[p] ?? p.charAt(0).toUpperCase() + p.slice(1);

function fmtDay(d: string): string {
  return new Date(`${d}T00:00:00Z`).toLocaleDateString('en-US', { month: 'short', day: 'numeric', timeZone: 'UTC' });
}

function RangeChips({ value, onChange }: { value: Range; onChange: (r: Range) => void }) {
  return (
    <div className="flex gap-1">
      {RANGES.map((r) => (
        <button
          key={String(r)}
          onClick={() => onChange(r)}
          className={`rounded-sm-t border px-2 py-0.5 text-[11.5px] transition-colors ${
            value === r
              ? 'border-[var(--accent-line)] bg-[var(--accent-soft)] text-os-accent'
              : 'border-os-border text-os-dim hover:bg-os-surface2 hover:text-os-muted'
          }`}
        >
          {RANGE_LABEL[String(r)]}
        </button>
      ))}
    </div>
  );
}

/** The two charts + shared hover crosshair/tooltip. `axis` drives BOTH the
    audience line and the stacked posting bars, so one cursor reads across them.
    Hover a day → see the audience value and exactly which platforms posted. */
function ChartPair({
  axis,
  audienceVals,
  carried,
  from,
  days,
  platforms,
  audClass,
  postClass,
}: {
  axis: string[];
  audienceVals: (number | null)[];
  /** Per axis day: value carried from an earlier point (dashed). */
  carried: boolean[];
  /** Per axis day: the date the value came from. */
  from: (string | null)[];
  days: PostingActivityDay[];
  platforms: string[];
  audClass: string;
  postClass: string;
}) {
  const [hover, setHover] = useState<number | null>(null);
  const n = axis.length;

  const W = 760;
  const HA = 200; // audience viewBox height
  const HP = 120; // posting viewBox height
  const PAD = 12;
  const xAt = (i: number) => (n > 1 ? (i / (n - 1)) * W : W / 2);

  // audience scale (only over known values)
  const known = audienceVals.filter((v): v is number => v != null);
  const aMax = known.length ? Math.max(...known) : 1;
  const aMin = known.length ? Math.min(...known) : 0;
  const yA = (v: number) => HA - PAD - ((v - aMin) / Math.max(1, aMax - aMin)) * (HA - 2 * PAD);
  const audIdx = audienceVals.flatMap((v, i) => (v == null ? [] : [i]));
  const audPts = audIdx.map((i) => [xAt(i), yA(audienceVals[i]!)] as [number, number]);
  const audLine = audPts.map((p, i) => `${i ? 'L' : 'M'}${p[0].toFixed(1)},${p[1].toFixed(1)}`).join(' ');
  // Segments ending on a carried day (a gap or the tail after the last sync)
  // draw dashed; segments ending on a real point draw solid.
  const segPath = (wantCarried: boolean) =>
    audIdx
      .slice(1)
      .map((i, k) =>
        carried[i] === wantCarried
          ? `M${audPts[k][0].toFixed(1)},${audPts[k][1].toFixed(1)} L${audPts[k + 1][0].toFixed(1)},${audPts[k + 1][1].toFixed(1)}`
          : '',
      )
      .join(' ');
  const solidLine = segPath(false);
  const carriedLine = segPath(true);

  // posting scale
  const dayMax = Math.max(1, ...days.map((d) => d.total));
  const slot = n > 0 ? W / n : W;
  const barW = Math.max(1.5, Math.min(16, slot * 0.7));
  const innerHP = HP - 8;

  const onMove = (e: React.MouseEvent<HTMLDivElement>) => {
    const rect = e.currentTarget.getBoundingClientRect();
    const frac = (e.clientX - rect.left) / rect.width;
    setHover(Math.min(n - 1, Math.max(0, Math.round(frac * (n - 1)))));
  };

  const overlay = (
    <div className="absolute inset-0 cursor-crosshair" onMouseMove={onMove} onMouseLeave={() => setHover(null)} />
  );

  const crosshair = (h: number) =>
    hover != null ? <line x1={xAt(hover)} x2={xAt(hover)} y1={0} y2={h} stroke="var(--accent)" strokeWidth="1" strokeOpacity="0.5" /> : null;

  const hoverDay = hover != null ? days[hover] : null;
  const tooltipLeft = hover != null && n > 1 ? (hover / (n - 1)) * 100 : 50;
  const hoverEntries = hoverDay ? platforms.filter((p) => (hoverDay.counts[p] ?? 0) > 0) : [];

  return (
    <div className="relative">
      {/* audience */}
      <div className={audClass}>
        <svg viewBox={`0 0 ${W} ${HA}`} preserveAspectRatio="none" className="block h-full w-full">
          {[0.33, 0.66].map((g) => (
            <line key={g} x1="0" x2={W} y1={HA - PAD - g * (HA - 2 * PAD)} y2={HA - PAD - g * (HA - 2 * PAD)} stroke="var(--border)" strokeDasharray="2 4" />
          ))}
          {/* flat low-opacity area fill — the Spark idiom; gradients retired with Console */}
          {audPts.length > 1 && (
            <path d={`${audLine} L${audPts[audPts.length - 1][0]},${HA} L${audPts[0][0]},${HA} Z`} fill="var(--accent)" opacity="0.08" />
          )}
          {solidLine && <path d={solidLine} fill="none" stroke="var(--accent)" strokeWidth="2" strokeLinejoin="round" strokeLinecap="round" />}
          {carriedLine && (
            <path d={carriedLine} fill="none" stroke="var(--accent)" strokeWidth="2" strokeDasharray="4 5" strokeOpacity="0.6" data-carried="" />
          )}
          {crosshair(HA)}
          {hover != null && audienceVals[hover] != null && (
            <circle cx={xAt(hover)} cy={yA(audienceVals[hover]!)} r="3.5" fill="var(--accent)" stroke="var(--surface)" strokeWidth="2" />
          )}
          {audPts.length > 0 && hover == null && (
            <circle cx={audPts[audPts.length - 1][0]} cy={audPts[audPts.length - 1][1]} r="3.5" fill="var(--accent)" stroke="var(--surface)" strokeWidth="2" />
          )}
        </svg>
      </div>

      <div className="mt-1.5 flex justify-between font-mono text-[11px] text-os-dim">
        <span>{axis[0] ? fmtDay(axis[0]) : ''}</span>
        <span>today</span>
      </div>

      <div className="my-3.5 h-px bg-os-border" />

      <div className="mb-2 flex items-center justify-between gap-4">
        <span className="text-[13px] font-medium text-os-muted">Posting consistency</span>
        <span className="text-[12.5px] text-os-muted tabular-nums">
          {days.reduce((s, d) => s + d.total, 0)} posts · {platforms.length} platforms
        </span>
      </div>

      {/* posting (stacked per platform) */}
      <div className={postClass}>
        <svg viewBox={`0 0 ${W} ${HP}`} preserveAspectRatio="none" className="block h-full w-full">
          {[0.25, 0.5, 0.75].map((g) => (
            <line key={g} x1="0" x2={W} y1={g * HP} y2={g * HP} stroke="var(--border)" strokeDasharray="2 4" />
          ))}
          {days.map((d, i) => {
            const cx = xAt(i) - barW / 2;
            if (d.total === 0) return <rect key={d.date} x={cx} y={HP - 2} width={barW} height={2} rx="1" fill="var(--accent)" opacity={0.14} />;
            let yCursor = HP;
            return (
              <g key={d.date} opacity={hover == null || hover === i ? 1 : 0.4}>
                {platforms.map((p) => {
                  const c = d.counts[p] ?? 0;
                  if (c === 0) return null;
                  const h = (c / dayMax) * innerHP;
                  yCursor -= h;
                  return <rect key={p} x={cx} y={yCursor} width={barW} height={Math.max(1, h - 0.6)} rx="1" fill={colorFor(p)} />;
                })}
              </g>
            );
          })}
          {crosshair(HP)}
        </svg>
      </div>

      <div className="mt-1.5 flex justify-between font-mono text-[11px] text-os-dim">
        <span>{axis[0] ? fmtDay(axis[0]) : ''}</span>
        <span>today</span>
      </div>

      {/* shared hover surface (spans both charts) + floating tooltip */}
      {overlay}
      {hover != null && (
        <div
          className="pointer-events-none absolute top-0 z-10 -translate-x-1/2"
          style={{ left: `${Math.min(88, Math.max(12, tooltipLeft))}%` }}
        >
          <div className="c-pop min-w-[150px] px-2.5 py-3">
            <div className="mb-1 flex items-center justify-between gap-4 font-mono text-[11.5px] text-os-dim">
              <span>{axis[hover] ? fmtDay(axis[hover]) : ''}</span>
              <span className="text-os-muted">
                {audienceVals[hover] != null ? audienceVals[hover]!.toLocaleString('en-US') : '—'} aud
              </span>
            </div>
            {carried[hover] && from[hover] && (
              <div className="mb-1 text-[11.5px] text-os-dim">carried from {fmtDay(from[hover]!)}</div>
            )}
            {hoverEntries.length === 0 ? (
              <div className="text-[11.5px] text-os-dim">No posts</div>
            ) : (
              <div className="flex flex-col gap-0.5">
                {hoverEntries.map((p) => (
                  <div key={p} className="flex items-center gap-2.5 text-[12px] text-os-text">
                    <span className="h-2 w-2 shrink-0 rounded-full" style={{ background: colorFor(p) }} />
                    <span className="flex-1">{labelFor(p)}</span>
                    {(hoverDay!.counts[p] ?? 0) > 1 && <span className="text-os-dim">×{hoverDay!.counts[p]}</span>}
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

function PlatformLegend({ platforms, days }: { platforms: string[]; days: PostingActivityDay[] }) {
  const totals = useMemo(() => {
    const t: Record<string, number> = {};
    for (const d of days) for (const p of platforms) t[p] = (t[p] ?? 0) + (d.counts[p] ?? 0);
    return t;
  }, [platforms, days]);
  if (platforms.length === 0) return null;
  return (
    <div className="mt-4 flex flex-wrap gap-x-5 gap-y-2.5">
      {platforms.map((p) => (
        <div key={p} className="flex items-center gap-2.5 text-[12.5px]">
          <span className="h-[13px] w-[13px] rounded-[4px]" style={{ background: colorFor(p) }} />
          <span className="text-os-muted">{labelFor(p)}</span>
          <span className="text-os-dim tabular-nums">{totals[p] ?? 0}</span>
        </div>
      ))}
    </div>
  );
}

/** Left-column charts: combined audience line + per-platform posting bars on one
    shared 7/30/60/all toggle and a shared hover cursor. The expand control opens
    a fullscreen view of the same two charts. */
export function AudienceConsistency({
  audience,
  channels,
  postDays,
  today,
  aside,
}: {
  audience: AudPoint[];
  /** Each audience channel's own points. "+N net" sums the channels that
      reach back to the window start, the same helper as the Audience growth
      tile (lib/social-range.ts), so the two always agree. */
  channels?: AudPoint[][];
  postDays: PostDay[];
  today: string;
  /** Rendered as a right-hand column sharing the card with the charts. */
  aside?: React.ReactNode;
}) {
  const [range, setRange] = useState<Range>(30);
  const [expanded, setExpanded] = useState(false);
  // Collapse melts the expanded face (lib/use-melt): `expanded` goes false at once, the face lingers for the melt.
  const collapse = () => setExpanded(false);
  const faceRef = useRef<HTMLDivElement>(null);
  const melt = useMeltPhase(expanded);
  const onMeltEnd = useMeltEnd(faceRef, melt.melting, melt.end);

  const { axis, audienceVals, carried, from, days, platforms, net, stale, lastSynced } = useMemo(() => {
    // One window for the line, the bars and "+N net" (lib/social-range.ts):
    // it ends on the newest audience point and the axis runs on to today, so
    // a stale sync shows as a dashed carried tail instead of a blank chart.
    const sorted = [...audience].sort((a, b) => (a.date < b.date ? -1 : a.date > b.date ? 1 : 0));
    const dates = [...sorted.map((a) => a.date), ...postDays.map((p) => p.date)].sort();
    const window = rangeWindow(range, sorted.at(-1)?.date ?? null, today, dates[0] ?? null);
    const start = window.start;
    const end = window.axisEnd;
    const axis = dateAxis(start, end);
    const filled = forwardFillCarried(sorted, axis);
    const inRangePosts = postDays.filter((p) => p.date >= start && p.date <= end);
    const days = bucketPostingByDay(inRangePosts, axis);
    const platforms = platformsPresent(inRangePosts, PREFERRED);
    const net = channelsNetChange(channels ?? [sorted], range, today)?.net ?? null;
    return {
      axis,
      audienceVals: filled.values,
      carried: filled.carried,
      from: filled.from,
      days,
      platforms,
      net,
      stale: window.stale,
      lastSynced: sorted.at(-1)?.date ?? null,
    };
  }, [audience, channels, postDays, today, range]);

  useEffect(() => {
    if (!expanded) return;
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && setExpanded(false);
    window.addEventListener('keydown', onKey);
    const prev = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      window.removeEventListener('keydown', onKey);
      document.body.style.overflow = prev;
    };
  }, [expanded]);

  // Same helper as the Audience growth tile (H18): unknown is absent, not 0.
  const netLabel =
    net == null ? null : (
      <span className={`font-mono text-[12.5px] ${net >= 0 ? 'text-os-ok' : 'text-os-err'}`}>
        {net >= 0 ? '▲' : '▼'} {net >= 0 ? '+' : ''}
        {net.toLocaleString('en-US')} net
      </span>
    );
  const staleNote =
    stale && lastSynced ? (
      <p className="mt-2 text-[12px] text-os-dim" data-stale-note="">
        Last synced {fmtDay(lastSynced)}. The dashed line carries that value forward.
      </p>
    ) : null;

  return (
    <div className="rounded-lg-t border border-os-border bg-os-surface px-5 py-[18px]">
      {/* the charts share the card with the aside (audience pie) — the pie
          rides shotgun on the right, not slammed underneath */}
      <div className={aside ? 'grid grid-cols-1 gap-6 lg:grid-cols-[1.55fr_1fr]' : ''}>
        <div className="flex min-w-0 flex-col">
          <div className="mb-2 flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
            <span className="text-[13px] font-medium text-os-muted">Combined audience</span>
            <div className="flex flex-wrap items-center gap-3.5">
              {netLabel}
              <RangeChips value={range} onChange={setRange} />
              <button
                onClick={() => setExpanded(true)}
                aria-label="Expand to fullscreen"
                title="Fullscreen"
                className="flex h-[22px] w-[22px] items-center justify-center rounded-sm-t border border-os-border text-os-dim transition-colors hover:bg-os-surface2 hover:text-os-text"
              >
                <Maximize2 className="h-3 w-3" />
              </button>
            </div>
          </div>

          <ChartPair
            axis={axis}
            audienceVals={audienceVals}
            carried={carried}
            from={from}
            days={days}
            platforms={platforms}
            audClass="min-h-[140px] grow-[1.6] basis-[140px]"
            postClass="min-h-[84px] grow basis-[84px]"
          />
          {staleNote}

          <PlatformLegend platforms={platforms} days={days} />
        </div>

        {aside && (
          <div className="flex min-w-0 flex-col justify-center max-lg:mt-5 max-lg:border-t max-lg:border-os-border max-lg:pt-4 lg:border-l lg:border-os-border lg:pl-6">
            {aside}
          </div>
        )}
      </div>

      {/* Portalled to <body>: this card frosts, and a frosted ancestor traps a fixed layer. */}
      {melt.mounted && (
        <Layer>
          <div className="fixed inset-0 z-[120] flex items-center justify-center p-4 sm:p-6" data-melting={melt.melting ? '' : undefined}>
            {/* The scrim is the face's sibling (glass brief §3.2): it frosts and dims the page; the face frosts on its own and melts on close. */}
            <div className="gl-scrim" aria-hidden onClick={collapse} />
            <div
              ref={faceRef}
              onAnimationEnd={onMeltEnd}
              className={`c-pop glass-face--strong gl-materialise relative flex max-h-[94vh] w-[min(1240px,96vw)] flex-col overflow-hidden${melt.melting ? ' is-melting' : ''}`}
            >
              <div className="flex flex-wrap items-center justify-between gap-4 border-b border-os-border px-5 py-3.5">
                <div>
                  <div className="text-[15px] font-medium">Combined audience &amp; posting consistency</div>
                  <div className="mt-0.5 text-[12.5px] text-os-muted">Social</div>
                </div>
                <div className="flex items-center gap-4">
                  <RangeChips value={range} onChange={setRange} />
                  <button
                    onClick={collapse}
                    aria-label="Close"
                    className="flex h-7 w-7 items-center justify-center rounded-sm-t border border-os-border text-os-dim transition-colors hover:bg-os-surface2 hover:text-os-text"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>
              </div>

              <div className="flex-1 overflow-y-auto px-5 py-4">
                <div className="mb-2 flex items-center justify-between">
                  <span className="text-[13px] font-medium text-os-muted">Combined audience</span>
                  {netLabel}
                </div>
                <ChartPair
                  axis={axis}
                  audienceVals={audienceVals}
                  carried={carried}
                  from={from}
                  days={days}
                  platforms={platforms}
                  audClass="h-[clamp(240px,40vh,420px)]"
                  postClass="h-[clamp(160px,26vh,300px)]"
                />
                {staleNote}
                <PlatformLegend platforms={platforms} days={days} />
              </div>
            </div>
          </div>
        </Layer>
      )}
    </div>
  );
}
