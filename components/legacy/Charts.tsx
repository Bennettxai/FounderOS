'use client';

import { useId, type CSSProperties, type ReactNode } from 'react';
import { dotColumns, meterWidth, stepGeometry, type Meter, type SeriesPoint } from './format';

const INK = 'var(--l-text)';

/** One labelled bar whose hatched, glowing fill sweeps in from the left; unknown draws a dashed empty track (FO VolumeMeter). */
export function VolumeMeter({ label, frac, display, hue = INK, delay = 0, compact = false }: Meter & { delay?: number; compact?: boolean }) {
  const w = meterWidth(frac);
  const fill: CSSProperties | undefined =
    w === null
      ? undefined
      : {
          width: `${Math.round(w * 1000) / 10}%`,
          background: `linear-gradient(90deg, transparent 72%, color-mix(in oklab, ${hue} 60%, var(--l-text)) 100%), repeating-linear-gradient(45deg, ${hue}, ${hue} 6px, color-mix(in oklab, ${hue} 45%, transparent) 6px, color-mix(in oklab, ${hue} 45%, transparent) 12px)`,
          boxShadow: `0 0 14px color-mix(in oklab, ${hue} 45%, transparent), inset 0 0 5px color-mix(in oklab, ${hue} 55%, transparent)`,
          animationDelay: `${delay}ms`,
        };
  return (
    <div data-l="VolumeMeter" className={compact ? 'l-meter is-compact' : 'l-meter'}>
      <div className="l-meter-row">
        <span className="l-meter-label">{label}</span>
        {w === null ? <span className="l-meter-value is-unknown">unknown</span> : <span className="l-meter-value">{display}</span>}
      </div>
      <div className="l-meter-track" data-unknown={w === null ? '' : undefined}>
        {w !== null && <div className="l-meter-fill" style={fill} />}
      </div>
    </div>
  );
}

/** The Deal Volume bars, staggered 150 ms, with a centred mono foot (FO MeterStack). */
export function MeterStack({
  meters,
  foot,
  empty = 'Nothing to measure yet.',
  baseDelay = 500,
  dense = false,
}: {
  meters: Meter[];
  foot?: ReactNode;
  empty?: string;
  baseDelay?: number;
  dense?: boolean;
}) {
  return (
    <div data-l="MeterStack" className={dense ? 'l-meters is-dense' : 'l-meters'}>
      <div className="l-meters-list">
        {meters.length === 0 ? (
          <div className="l-caption">{empty}</div>
        ) : (
          meters.map((m, i) => <VolumeMeter key={m.label} {...m} delay={baseDelay + i * 150} />)
        )}
      </div>
      {foot && <div className="l-meters-foot">{foot}</div>}
    </div>
  );
}

/**
 * Stepped line over fading pinstripes that draws itself and pins its peak with
 * a square marker (FO slab-charts StepLine). `hue` is a token expression.
 */
export function StepLine({
  series,
  hue = INK,
  empty = 'No activity in this window.',
  unit = '',
  height = 150,
  format,
}: {
  series: SeriesPoint[];
  hue?: string;
  empty?: string;
  unit?: string;
  height?: number;
  /** How the peak pill prints the count (default: grouped integer). */
  format?: (n: number) => string;
}) {
  const uid = useId().replace(/:/g, '');
  const geo = stepGeometry(series, 600, height);
  if (!geo) return <div data-l="StepLine" className="l-caption l-chart-empty">{empty}</div>;
  const { W, H } = geo;
  const pins = `l-pins-${uid}`;
  const fade = `l-grad-${uid}`;
  const mask = `l-mask-${uid}`;
  const mx = (geo.peakIdx + 0.5) * geo.stepW;
  return (
    <div data-l="StepLine" className="l-stepline">
      <svg viewBox={`0 0 ${W} ${H}`} className="block w-full" aria-hidden>
        <defs>
          <pattern id={pins} width="5" height="8" patternUnits="userSpaceOnUse">
            <rect width="1.2" height="8" style={{ fill: hue }} opacity="0.28" />
          </pattern>
          <linearGradient id={fade} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0" style={{ stopColor: 'var(--l-text)', stopOpacity: 0.9 }} />
            <stop offset="1" style={{ stopColor: 'var(--l-text)', stopOpacity: 0.05 }} />
          </linearGradient>
          <mask id={mask}>
            <rect width={W} height={H} fill={`url(#${fade})`} />
          </mask>
        </defs>
        <path d={geo.area} fill={`url(#${pins})`} mask={`url(#${mask})`} />
        <path data-l-part="line" className="l-draw" d={geo.path} fill="none" strokeWidth={2.5} strokeLinejoin="miter" pathLength={1} style={{ stroke: hue }} />
        <rect className="l-fade" x={mx - 4.5} y={geo.py - 4.5} width={9} height={9} style={{ fill: 'var(--l-bg-2)', stroke: hue, strokeWidth: 2 }} />
      </svg>
      <div
        className="l-peak glass-solid"
        style={{ left: `calc(24px + (100% - 48px) * ${(geo.peakIdx + 0.5) / series.length})`, top: `${(geo.py / H) * 100}%` }}
      >
        <span className="l-peak-n" style={{ color: hue }}>
          {format ? format(geo.peak.count) : geo.peak.count.toLocaleString('en-US')}
          {unit}
        </span>{' '}
        <span className="l-peak-on">on {geo.peak.label}</span>
      </div>
      <div className="l-axis">
        <span>{series[0].label}</span>
        <span>{series[series.length - 1].label}</span>
      </div>
    </div>
  );
}

/** Waffle dot-matrix mini: up to six dots a column, the tallest at the max (FO slab-charts DotMatrix). */
export function DotMatrix({ cols, hue = INK }: { cols: SeriesPoint[]; hue?: string }) {
  const bars = dotColumns(cols);
  return (
    <div data-l="DotMatrix" className="l-dotmatrix">
      {bars.map((c, ci) => (
        <div key={c.label} className="l-dotcol">
          <div className="l-dotstack">
            {Array.from({ length: c.dots }, (_, i) => (
              <span key={i} className="l-fade l-dm-dot" style={{ background: hue, opacity: c.strength, animationDelay: `${900 + ci * 90 + i * 55}ms` }} />
            ))}
            {c.dots === 0 && <span className="l-dm-dot" style={{ background: hue, opacity: 0.12 }} />}
          </div>
          <span className="l-dm-label">{c.label}</span>
        </div>
      ))}
    </div>
  );
}
