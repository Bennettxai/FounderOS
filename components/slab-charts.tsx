import { useId } from 'react';

/**
 * Brand Deals' two mini charts, lent to the whole OS (2026-09-24): the
 * stepped line over fading pinstripes that draws itself and pins its peak,
 * and the waffle dot-matrix. Pure SVG/markup on global keyframes (os-draw,
 * os-fade), so server pages can use them without shipping a client island.
 */

const EASE = 'cubic-bezier(.2,.7,.2,1)';

export type SeriesPoint = { label: string; count: number };

export function StepLine({
  series,
  hue,
  empty = 'No activity in this window.',
  unit = '',
}: {
  series: SeriesPoint[];
  hue: string;
  empty?: string;
  /** Appended to the peak callout's number ("12 runs on Sep 3"). */
  unit?: string;
}) {
  const uid = useId().replace(/:/g, '');
  const total = series.reduce((n, s) => n + s.count, 0);
  if (series.length === 0 || total === 0) return <div className="px-6 py-8 text-[12px] text-os-dim">{empty}</div>;
  const W = 600;
  const H = 150;
  const max = Math.max(...series.map((b) => b.count), 1);
  const stepW = W / series.length;
  const y = (c: number) => H - 14 - (c / max) * (H - 60);
  let path = `M 0 ${y(series[0].count)}`;
  series.forEach((_, i) => {
    path += ` H ${(i + 1) * stepW}`;
    if (i < series.length - 1) path += ` V ${y(series[i + 1].count)}`;
  });
  const peakIdx = series.reduce((bi, b, i) => (b.count > series[bi].count ? i : bi), 0);
  const area = `${path} V ${H} H 0 Z`;
  const pins = `pins-${uid}`;
  const fade = `fade-${uid}`;
  const mask = `mask-${uid}`;
  const peak = series[peakIdx];
  return (
    <div className="relative px-6 pb-5">
      <svg viewBox={`0 0 ${W} ${H}`} className="block w-full" aria-hidden="true">
        <defs>
          <pattern id={pins} width="5" height="8" patternUnits="userSpaceOnUse">
            <rect width="1.2" height="8" fill={hue} opacity="0.28" />
          </pattern>
          <linearGradient id={fade} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0" stopColor="#fff" stopOpacity="0.9" />
            <stop offset="1" stopColor="#fff" stopOpacity="0.05" />
          </linearGradient>
          <mask id={mask}>
            <rect width={W} height={H} fill={`url(#${fade})`} />
          </mask>
        </defs>
        <path d={area} fill={`url(#${pins})`} mask={`url(#${mask})`} />
        <path
          d={path}
          fill="none"
          stroke={hue}
          strokeWidth="3"
          strokeLinejoin="round"
          pathLength={1}
          style={{ strokeDasharray: 1, strokeDashoffset: 1, animation: `os-draw 1.6s ${EASE} .5s both` }}
        />
        <circle cx={(peakIdx + 0.5) * stepW} cy={y(peak.count)} r="4.5" fill={hue} style={{ animation: `os-fade .4s ${EASE} 1.9s both` }} />
      </svg>
      <div
        className="pointer-events-none absolute whitespace-nowrap rounded-full border border-os-border px-2.5 py-1 text-[11px] backdrop-blur"
        style={{
          left: `calc(24px + (100% - 48px) * ${(peakIdx + 0.5) / series.length})`,
          top: `${(y(peak.count) / H) * 100}%`,
          transform: 'translate(-50%, -140%)',
          background: 'color-mix(in oklab, var(--bg) 75%, transparent)',
          animation: `os-fade .5s ${EASE} 2s both`,
        }}
      >
        <span className="font-semibold tabular-nums" style={{ color: hue }}>
          {peak.count.toLocaleString('en-US')}
          {unit}
        </span>{' '}
        <span className="text-os-muted">on {peak.label}</span>
      </div>
      <div className="mt-1 flex justify-between font-mono text-[10.5px] text-os-dim">
        <span>{series[0].label}</span>
        <span>{series[series.length - 1].label}</span>
      </div>
    </div>
  );
}

/** Waffle dot-matrix mini: up to six dots a column, the tallest at the max. */
export function DotMatrix({ cols, hue }: { cols: SeriesPoint[]; hue: string }) {
  const max = Math.max(...cols.map((c) => c.count), 1);
  return (
    <div className="flex items-end gap-3">
      {cols.map((c, ci) => {
        const dots = Math.max(c.count === 0 ? 0 : 1, Math.round((c.count / max) * 6));
        const strength = c.count === max ? 1 : c.count >= max * 0.6 ? 0.55 : 0.25;
        return (
          <div key={c.label} className="flex flex-col items-center gap-1.5">
            <div className="flex flex-col-reverse gap-[3px]">
              {Array.from({ length: dots }, (_, i) => (
                <span
                  key={i}
                  data-dot
                  className="block h-[7px] w-[7px] rounded-full"
                  style={{ background: hue, opacity: strength, animation: `os-fade .3s ${EASE} ${900 + ci * 90 + i * 55}ms both` }}
                />
              ))}
              {dots === 0 && <span className="block h-[7px] w-[7px] rounded-full" style={{ background: hue, opacity: 0.12 }} />}
            </div>
            <span className="whitespace-nowrap font-mono text-[10px] text-os-dim">{c.label}</span>
          </div>
        );
      })}
    </div>
  );
}
