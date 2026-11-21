'use client';

/**
 * PipelineChart — the "Pipeline" widget's design lifted out of
 * components/brand-deals/DealBoard.tsx (`Funnel`, ~2026-08-19) so it can be
 * reused wherever a set of stages/categories deserves the same hero
 * treatment: column headers (42px numeral + label + mono $ line, each a
 * working filter button) above an SVG of hatched, gradient-filled step
 * columns that fade out at the bottom via a mask, with an optional AI-style
 * search bar melting out of the chart base. DealBoard now renders its
 * Pipeline card through this component — see the `Funnel`-shaped usage
 * there for the reference call.
 */

import { useEffect, useState } from 'react';
import { stageBarGeometry } from '@/lib/chart-geometry';

export type PipelineStage = {
  label: string;
  value: number;
  /** Optional dollar figure for the mono line under the numeral. */
  usd?: number;
  /** Text shown next to the value in the tooltip pill over the active column. */
  note?: string;
  /** Dim this column's header (e.g. it doesn't match an active search) without removing it. */
  dim?: boolean;
  /** Stable id for deep links (e.g. an income category key, `/finances?category=`). */
  key?: string;
};

const EASE = 'cubic-bezier(.2,.7,.2,1)';

/** Same ease-out cubic count-up DealBoard's stat numerals use. */
export function useCountUp(target: number, dur = 1400): number {
  const [v, setV] = useState(0);
  useEffect(() => {
    let raf = 0;
    const t0 = performance.now();
    const tick = (t: number) => {
      const p = Math.min(1, (t - t0) / dur);
      const e = 1 - Math.pow(1 - p, 3);
      setV(Math.round(target * e));
      if (p < 1) raf = requestAnimationFrame(tick);
    };
    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
  }, [target, dur]);
  return v;
}

/** The exact caret-blink animation shorthand for a search-bar slot's cursor span. */
export const CARET_ANIMATION = 'pc-caret 1.1s step-end infinite';

function AnimatedNumeral({
  value,
  format,
  compact,
  masked,
}: {
  value: number;
  format: (v: number) => string;
  /** When set, a compact form is rendered too; CSS shows it when the column is narrow (responsive mode). */
  compact?: (v: number) => string;
  masked?: boolean;
}) {
  const v = useCountUp(value);
  const style = masked ? { filter: 'blur(7px)', userSelect: 'none' as const } : undefined;
  if (!compact) return <span style={style}>{format(v)}</span>;
  return (
    <span style={style}>
      <span className="pc-full">{format(v)}</span>
      <span className="pc-compact">{compact(v)}</span>
    </span>
  );
}

function MaybeMasked({ text, masked }: { text: string; masked?: boolean }) {
  return <span style={masked ? { filter: 'blur(7px)', userSelect: 'none' } : undefined}>{text}</span>;
}

export interface PipelineChartProps<T extends PipelineStage> {
  stages: T[];
  /** Index of the stage whose column renders solid/filled and drives the tooltip. */
  active: number;
  onSelect?: (stage: T, index: number) => void;
  /** Formats the big numeral (defaults to a plain rounded integer, like a count). */
  formatValue?: (value: number) => string;
  /** Formats the mono `usd` line (defaults to `$12,345`). */
  formatUsd?: (value: number) => string;
  /** Blur every rendered figure (numeral, mono line, tooltip) — a hide-values mode. */
  maskValues?: boolean;
  /** Renders inside the gradient backdrop-blur wrapper melting out of the chart base. */
  searchSlot?: React.ReactNode;
  numeralSize?: number;
  svgHeight?: number;
  showAxis?: boolean;
  showTooltip?: boolean;
  contentClassName?: string;
  className?: string;
  /**
   * Width-aware header (S10 FR-12): columns share the width with
   * minmax(0,1fr), numerals scale with the chart's container, a column under
   * 112px shows `compactValue` / `compactUsd` (full value in title and
   * aria-label), and under 360px three or more columns wrap 2x2. Off by
   * default; the home finance strip, /finances and /brand-deals opt in.
   */
  responsive?: boolean;
  compactValue?: (value: number) => string;
  compactUsd?: (value: number) => string;
  /** Hover or keyboard focus on a column header. */
  onHover?: (stage: T, index: number) => void;
  /**
   * What the bar heights encode when it is not `value` (e.g. `(s) => s.usd`
   * on /brand-deals, H8). Heights are then strictly proportional through
   * `lib/chart-geometry.ts`, and the y-axis (which would be in `value`
   * units) is dropped. Omitted: today's count-sized bars, unchanged.
   */
  measure?: (stage: T) => number;
}

export function PipelineChart<T extends PipelineStage>({
  stages,
  active,
  onSelect,
  formatValue = (v) => String(Math.round(v)),
  formatUsd = (v) => `$${Math.round(v).toLocaleString('en-US')}`,
  maskValues = false,
  searchSlot,
  numeralSize = 42,
  svgHeight = 220,
  showAxis = true,
  showTooltip = true,
  contentClassName = 'px-6 pt-4',
  className = '',
  responsive = false,
  compactValue,
  compactUsd,
  onHover,
  measure,
}: PipelineChartProps<T>) {
  const max = Math.max(...stages.map((s) => s.value), 1);
  const W = 1000;
  const SVG_H = svgHeight;
  const H = SVG_H + 40;
  const AXIS = showAxis && !measure ? 44 : 0;
  const colW = (W - AXIS) / Math.max(stages.length, 1);
  const barW = colW * 0.62;
  // responsive: caps follow the bar (half its width, at most today's 32)
  const capW = responsive ? Math.min(32, barW * 0.5) : 32;
  const h = (v: number) => (v === 0 ? 10 : 34 + (v / max) * (H - 54));
  // Measured bars: linear in the measure, tallest = today's tallest (H - 20),
  // zero = today's 10-unit stub.
  const measured = measure ? stageBarGeometry(stages.map(measure), H - 20, 10) : null;
  const barH = (i: number) => (measured ? measured[i].h : h(stages[i].value));
  const x0 = (i: number) => AXIS + i * colW + 4;
  const ticks = showAxis && !measure ? [0, 0.25, 0.5, 0.75, 1].map((f) => Math.round(max * f)) : [];
  const activeStage: T | undefined = stages[active];
  const activeTopPx = activeStage ? ((H - barH(active)) / H) * SVG_H : 0;

  return (
    <div className={`${responsive ? 'pc-r ' : ''}${className}`} data-pc>
      <style>{`
        @keyframes pc-grow { from { transform: scaleY(0); } to { transform: scaleY(1); } }
        @keyframes pc-fadein { from { opacity: 0; } to { opacity: 1; } }
        @keyframes pc-rise { from { visibility: hidden; translate: 0 6px; } to { visibility: visible; translate: 0 0; } }
        @media (prefers-reduced-motion: reduce) { [data-pc] .pc-tip { animation: none !important; } }
        @keyframes pc-caret { 0%, 45% { opacity: 1; } 55%, 100% { opacity: 0; } }
      `}</style>
      <div className={contentClassName}>
        {/* column headers: numerals rule the card; every header is a working filter button */}
        <div
          className={responsive ? 'pc-head grid' : 'grid'}
          data-pc-header={responsive ? '' : undefined}
          data-pc-n={responsive ? stages.length : undefined}
          style={{
            gridTemplateColumns: `repeat(${stages.length}, ${responsive ? 'minmax(0, 1fr)' : '1fr'})`,
            marginLeft: `${(AXIS / W) * 100}%`,
            ...(responsive ? ({ '--pc-num': `${numeralSize}px` } as React.CSSProperties) : null),
          }}
        >
          {stages.map((s, i) => {
            const isActive = i === active;
            const label = responsive && !maskValues ? `${s.label}: ${formatValue(s.value)}${s.usd !== undefined ? `, ${formatUsd(s.usd)}` : ''}` : undefined;
            return (
              <button
                key={s.key ?? s.label}
                type="button"
                onClick={() => onSelect?.(s, i)}
                onMouseEnter={onHover ? () => onHover(s, i) : undefined}
                onFocus={onHover ? () => onHover(s, i) : undefined}
                title={label}
                aria-label={label}
                aria-pressed={responsive ? isActive : undefined}
                className={`${responsive ? 'pc-col ' : ''}px-5 pb-3 text-left transition-colors hover:bg-[color-mix(in_oklab,var(--text)_4%,transparent)] ${
                  i > 0 ? 'border-l border-os-border' : ''
                }`}
                style={{ opacity: s.dim && !isActive ? 0.4 : 1 }}
              >
                <div className={`${responsive ? 'pc-label ' : ''}text-[12.5px] ${isActive ? 'text-os-muted' : 'text-os-dim'}`}>{s.label}</div>
                <div
                  className={`${responsive ? 'pc-num whitespace-nowrap ' : ''}mt-0.5 font-semibold leading-none tabular-nums tracking-[-0.03em] ${
                    isActive ? '' : 'text-os-dim'
                  }`}
                  style={responsive ? undefined : { fontSize: numeralSize }}
                >
                  <AnimatedNumeral value={s.value} format={formatValue} compact={responsive ? compactValue : undefined} masked={maskValues} />
                </div>
                {s.usd !== undefined && (
                  <div className={`${responsive ? 'pc-num pc-usd whitespace-nowrap ' : ''}mt-1 font-mono text-[11.5px] tabular-nums ${isActive ? 'text-os-accent' : 'text-os-dim'}`} style={responsive ? { fontSize: 11.5 } : undefined}>
                    {responsive && compactUsd ? (
                      <span style={maskValues ? { filter: 'blur(7px)', userSelect: 'none' } : undefined}>
                        <span className="pc-full">{formatUsd(s.usd)}</span>
                        <span className="pc-compact">{compactUsd(s.usd)}</span>
                      </span>
                    ) : (
                      <MaybeMasked text={formatUsd(s.usd)} masked={maskValues} />
                    )}
                  </div>
                )}
              </button>
            );
          })}
        </div>

        <div
          className="relative"
          style={{
            maskImage: 'linear-gradient(to bottom, black 74%, transparent 100%)',
            WebkitMaskImage: 'linear-gradient(to bottom, black 74%, transparent 100%)',
          }}
        >
          <svg viewBox={`0 0 ${W} ${H}`} className="block w-full" style={{ height: SVG_H }} preserveAspectRatio="none" aria-hidden="true">
            <defs>
              <pattern id="pc-hatch" width="9" height="9" patternUnits="userSpaceOnUse" patternTransform="rotate(45)">
                <rect width="9" height="9" fill="color-mix(in oklab, var(--accent) 14%, transparent)" />
                <rect width="3.5" height="9" fill="color-mix(in oklab, var(--accent) 42%, transparent)" />
              </pattern>
              <linearGradient id="pc-solid" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0" stopColor="var(--accent)" />
                <stop offset="1" stopColor="color-mix(in oklab, var(--accent) 55%, var(--pc-shade))" />
              </linearGradient>
              <linearGradient id="pc-fadeg" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0" stopColor="#fff" stopOpacity="0.95" />
                <stop offset="1" stopColor="#fff" stopOpacity="0.14" />
              </linearGradient>
              <mask id="pc-fade">
                <rect width={W} height={H} fill="url(#pc-fadeg)" />
              </mask>
            </defs>
            {/* y-axis ticks ground the block heights */}
            {showAxis &&
              ticks.map((t, i) => {
                const y = H - h(t) + (t === 0 ? 10 : 0);
                return (
                  <g key={i}>
                    <text x={AXIS - 10} y={y + 3} textAnchor="end" fontSize="11" fill="var(--text-3)" style={{ fontVariantNumeric: 'tabular-nums' }}>
                      {t}
                    </text>
                    <line x1={AXIS} x2={W} y1={y} y2={y} stroke="var(--border)" strokeWidth="0.5" opacity="0.5" />
                  </g>
                );
              })}
            {/* one <g> per stage: block + its short ramp step-face to the next */}
            {stages.map((s, i) => {
              const isActive = i === active;
              const bx = x0(i);
              const top = H - barH(i);
              const next = stages[i + 1];
              const rampEnd = next ? x0(i + 1) : null;
              const nextTop = next ? H - barH(i + 1) : 0;
              const fill = isActive ? 'url(#pc-solid)' : 'url(#pc-hatch)';
              const body = (
                <g key={s.label}>
                  <rect
                    x={bx}
                    y={top}
                    width={barW}
                    height={H - top}
                    rx={isActive ? 3 : 0}
                    fill={fill}
                    style={{ transformBox: 'fill-box', transformOrigin: 'bottom', animation: `pc-grow .8s ${EASE} ${200 + i * 110}ms both` }}
                  />
                  {rampEnd !== null && (
                    <polygon
                      points={`${bx + barW},${top} ${rampEnd},${nextTop} ${rampEnd},${H} ${bx + barW},${H}`}
                      fill={fill}
                      opacity={isActive ? 0.5 : 0.55}
                      style={{ animation: `pc-fadein .9s ${EASE} ${380 + i * 110}ms both` }}
                    />
                  )}
                </g>
              );
              return isActive ? body : <g key={s.label} mask="url(#pc-fade)">{body}</g>;
            })}
            {/* capsule caps floating above each bar */}
            {stages.map((s, i) => (
              <rect
                key={`cap-${s.label}`}
                x={x0(i) + barW / 2 - capW / 2}
                y={H - barH(i) - 12}
                width={capW}
                height="6"
                rx="3"
                fill={i === active ? 'var(--accent)' : 'color-mix(in oklab, var(--text) 25%, transparent)'}
                style={{ animation: `pc-fadein .5s ${EASE} ${700 + i * 90}ms both` }}
              />
            ))}
          </svg>
          {/* tooltip pill anchored just above the active stage's top edge */}
          {showTooltip && activeStage?.note && (
            <div
              className="pc-tip glass-solid pointer-events-none absolute flex items-center gap-2 whitespace-nowrap rounded-full px-3.5 py-1.5 text-[12px]"
              style={(() => {
                // clamp near the edges — a centered pill over the first or
                // last column half-clips outside the card
                const centerPct = ((AXIS + active * colW + 4 + barW / 2) / W) * 100;
                const tx = centerPct < 15 ? '0%' : centerPct > 85 ? '-100%' : '-50%';
                return {
                left: `${centerPct}%`,
                top: Math.max(6, activeTopPx - 40),
                transform: `translateX(${tx})`,
                // a tooltip is a solid raised face: it rises 6px into place, never fades (glass brief §5 B13)
                animation: `pc-rise .6s ${EASE} 1100ms both`,
                };
              })()}
            >
              <span className="font-semibold tabular-nums">
                <MaybeMasked text={formatValue(activeStage.value)} masked={maskValues} />
              </span>
              <span className="text-os-muted">{activeStage.note}</span>
            </div>
          )}
        </div>
      </div>

      {/* AI prompt bar melting out of the chart base — no top edge */}
      {searchSlot && (
        <div className="relative z-[2] mx-6 -mt-12">
          <div
            className="rounded-[16px] p-3 backdrop-blur"
            style={{
              background:
                'linear-gradient(180deg, transparent, color-mix(in oklab, var(--accent) 9%, transparent) 40%, color-mix(in oklab, var(--accent) 12%, transparent))',
            }}
          >
            {searchSlot}
          </div>
        </div>
      )}
    </div>
  );
}
