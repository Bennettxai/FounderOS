'use client';

/**
 * Funnel dashboard — one-page dashboard study (stepped hatched funnel, meters,
 * step-line, dot matrices, one gradient insight card, explore bar) re-dressed
 * in Console tokens and Geist. Everything on screen is computed from the
 * journeys handed in by the server page; nothing is decorative data, and
 * every control does something: view, stage, period, grain, comparison, lane
 * and the open lead all live in the URL so a copied link reproduces the view.
 */
import './funnel.css';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useEffect, useMemo, useRef, useState, useTransition } from 'react';
import { CHANNEL_GLYPHS, DECAY_DAYS, FUNNEL_STAGES, decayFactor, funnelSummary, journeyMeta } from '@/lib/funnel';
import type { FunnelJourney, FunnelStage } from '@/lib/schemas';
import { SetupLink } from '@/components/SetupLink';
import { formatDelta } from '@/lib/delta';
import type { ChannelsCard } from '@/lib/funnel-channels';
import { FunnelChannels } from './FunnelChannels';
import { FunnelLinks } from './FunnelLinks';

export type DashboardConnector = {
  id: string;
  name: string;
  ok: boolean;
  detail: string;
  href: string;
  count?: number;
};

import { LANES, type Compare, type DashboardInitial, type Grain, type Lane, type Period, type View } from './state';

export type DashboardProps = {
  journeys: FunnelJourney[];
  archived: FunnelJourney[];
  nowIso: string;
  source: { kind: 'live' | 'db' | 'sample'; label: string };
  connectors: DashboardConnector[];
  pushNow: string[];
  saveNow: string[];
  initial: DashboardInitial;
  /** Blank install with no CRM: offer the badged sample set as an opt-in preview. */
  sampleOffer?: boolean;
  /** F7 channel analytics, decided on the server. */
  channels?: ChannelsCard;
  /** F6 link studio: whether Trakyo is keyed (the card then reads /api/trakyo/links). */
  trakyo?: { configured: boolean; setupHref: string };
};

const DAY = 86_400_000;
const PERIODS: { id: Period; label: string }[] = [
  { id: 30, label: 'Last 30 days' },
  { id: 90, label: 'Last 90 days' },
  { id: 365, label: 'Last 12 months' },
  { id: 0, label: 'All time' },
];
const WEEKDAYS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];
const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
const LANE_LABEL: Record<Lane, string> = { organic: 'Organic', ads: 'Ads', crm: 'CRM inbound' };
const LANE_VAR: Record<Lane, string> = {
  organic: 'var(--fc-meter-organic)',
  ads: 'var(--fc-meter-ads)',
  crm: 'var(--fc-meter-crm)',
};
const STAGE_VAR = (i: number) => `var(--funnel-s${i})`;

const usd = (n: number) => `$${Math.round(n).toLocaleString('en-US')}`;
const num = (n: number) => n.toLocaleString('en-US');
const dateOf = (s: string) => new Date(`${s}T00:00:00Z`).getTime();
const firstAt = (j: FunnelJourney) => j.touches[0]?.at ?? j.createdAt;
const convertedAt = (j: FunnelJourney) =>
  j.touches.find((t) => t.stage === 'converted')?.at ?? j.touches[j.touches.length - 1]?.at ?? j.createdAt;
const laneOf = (j: FunnelJourney): Lane => {
  const t = j.touches[0];
  if (t?.source === 'attio' || t?.source === 'ghl') return 'crm';
  if (t?.channel === 'ads') return 'ads';
  return 'organic';
};
const weekday = (s: string) => (new Date(`${s}T00:00:00Z`).getUTCDay() + 6) % 7;
const fmtDay = (ms: number, withYear = false) => {
  const d = new Date(ms);
  return `${MONTHS[d.getUTCMonth()]} ${d.getUTCDate()}${withYear ? ` ${d.getUTCFullYear()}` : ''}`;
};
const fmtDate = (s: string) => fmtDay(dateOf(s), true);
const stageLabel = (id: FunnelStage) => FUNNEL_STAGES.find((s) => s.id === id)?.label ?? id;
const stageIndex = (id: FunnelStage) => FUNNEL_STAGES.findIndex((s) => s.id === id);

/* ---------- motion hooks ---------- */

function useReducedMotion() {
  const [reduced, setReduced] = useState(false);
  useEffect(() => {
    const mq = window.matchMedia('(prefers-reduced-motion: reduce)');
    setReduced(mq.matches);
    const on = () => setReduced(mq.matches);
    mq.addEventListener('change', on);
    return () => mq.removeEventListener('change', on);
  }, []);
  return reduced;
}

/** Counts up from the last shown value to `target` on the study's cubic ease. */
function useCountUp(target: number, delay = 500, dur = 1400): number {
  const [v, setV] = useState(0);
  const shown = useRef(0);
  useEffect(() => {
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
      shown.current = target;
      setV(target);
      return;
    }
    const from = shown.current;
    const t0 = performance.now() + delay;
    let raf = 0;
    const tick = (t: number) => {
      const p = Math.min(1, Math.max(0, (t - t0) / dur));
      const e = 1 - Math.pow(1 - p, 3);
      const next = Math.round(from + (target - from) * e);
      shown.current = next;
      setV(next);
      if (p < 1) raf = requestAnimationFrame(tick);
    };
    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
  }, [target, delay, dur]);
  return v;
}

/** Closes a popover on outside mousedown or Escape. */
function useDismiss(open: boolean, ref: React.RefObject<HTMLElement>, close: () => void) {
  useEffect(() => {
    if (!open) return;
    const down = (e: MouseEvent) => {
      if (!ref.current?.contains(e.target as Node)) close();
    };
    const key = (e: KeyboardEvent) => {
      if (e.key === 'Escape') close();
    };
    window.addEventListener('mousedown', down);
    window.addEventListener('keydown', key);
    return () => {
      window.removeEventListener('mousedown', down);
      window.removeEventListener('keydown', key);
    };
  }, [open, ref, close]);
}

/* ---------- icons (inline stroke, Lucide language) ---------- */

const I = {
  refresh: (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 12a9 9 0 1 1-2.64-6.36" />
      <polyline points="21 3 21 9 15 9" />
    </svg>
  ),
  external: (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round">
      <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" />
      <polyline points="15 3 21 3 21 9" />
      <line x1="10" y1="14" x2="21" y2="3" />
    </svg>
  ),
  link: (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round">
      <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" />
      <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" />
    </svg>
  ),
  check: (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round">
      <polyline points="20 6 9 17 4 12" />
    </svg>
  ),
  close: (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <line x1="18" y1="6" x2="6" y2="18" />
      <line x1="6" y1="6" x2="18" y2="18" />
    </svg>
  ),
  calendar: (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round">
      <rect x="3" y="4" width="18" height="18" rx="2" />
      <line x1="16" y1="2" x2="16" y2="6" />
      <line x1="8" y1="2" x2="8" y2="6" />
      <line x1="3" y1="10" x2="21" y2="10" />
    </svg>
  ),
  spark: (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="currentColor" stroke="none">
      <path d="M12 3l1.9 5.6 5.6 1.9-5.6 1.9L12 18l-1.9-5.6L4.5 10.5l5.6-1.9z" />
    </svg>
  ),
  bulb: (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round">
      <path d="M9 18h6" />
      <path d="M10 22h4" />
      <path d="M15.09 14c.18-.98.65-1.74 1.41-2.5A4.65 4.65 0 0 0 18 8 6 6 0 0 0 6 8c0 1 .23 2.23 1.5 3.5.76.76 1.23 1.52 1.41 2.5" />
    </svg>
  ),
};

/* ---------- small chrome ---------- */

function Menu<T extends string | number>({
  value,
  options,
  onChange,
  extra,
}: {
  value: T;
  options: { id: T; label: string; hint?: string }[];
  onChange: (v: T) => void;
  extra?: { label: string; run: () => void }[];
}) {
  return (
    <div className="fc-menu" role="menu">
      {extra?.map((x) => (
        <button
          key={x.label}
          type="button"
          role="menuitem"
          onClick={(e) => {
            e.stopPropagation();
            x.run();
          }}
        >
          {x.label}
        </button>
      ))}
      {extra && extra.length > 0 && <div className="sep" />}
      {options.map((o) => (
        <button
          key={String(o.id)}
          type="button"
          role="menuitemradio"
          aria-checked={o.id === value}
          className={o.id === value ? 'on' : ''}
          onClick={(e) => {
            e.stopPropagation();
            onChange(o.id);
          }}
        >
          <span>
            {o.label}
            {o.hint && <span className="hint">{o.hint}</span>}
          </span>
          {o.id === value && I.check}
        </button>
      ))}
    </div>
  );
}

function Kebab({
  period,
  onPeriod,
  onJourneys,
  label,
}: {
  period: Period;
  onPeriod: (p: Period) => void;
  onJourneys: () => void;
  label: string;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  useDismiss(open, ref, () => setOpen(false));
  return (
    <div className="fc-kebab-wrap" ref={ref}>
      <button
        type="button"
        className="fc-kebab"
        aria-label={`${label} options`}
        aria-expanded={open}
        onClick={() => setOpen((o) => !o)}
      >
        ···
      </button>
      {open && (
        <Menu
          value={period}
          options={PERIODS}
          onChange={(p) => {
            onPeriod(p);
            setOpen(false);
          }}
          extra={[
            {
              label: 'See these journeys',
              run: () => {
                setOpen(false);
                onJourneys();
              },
            },
          ]}
        />
      )}
    </div>
  );
}

function Dropdown<T extends string | number>({
  value,
  options,
  icon,
  onChange,
  label,
}: {
  value: T;
  options: { id: T; label: string; hint?: string }[];
  icon?: React.ReactNode;
  onChange: (v: T) => void;
  label: string;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  useDismiss(open, ref, () => setOpen(false));
  const current = options.find((o) => o.id === value)?.label ?? '';
  return (
    <div
      ref={ref}
      className="fc-pill"
      role="button"
      tabIndex={0}
      aria-haspopup="menu"
      aria-expanded={open}
      aria-label={label}
      onClick={() => setOpen((o) => !o)}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          setOpen((o) => !o);
        }
      }}
    >
      {icon}
      {current} <span className="chev">▾</span>
      {open && (
        <Menu
          value={value}
          options={options}
          onChange={(v) => {
            onChange(v);
            setOpen(false);
          }}
        />
      )}
    </div>
  );
}

/** Period-over-period delta through lib/delta.ts (H12, C41): a percentage
 *  only while it means something; above 999% or off a zero base it reads as
 *  the absolute change ("+$352K vs previous"). Full values in the title. */
function Delta({ cur, prev, money = false, title }: { cur: number; prev: number; money?: boolean; title: string }) {
  const d = formatDelta(cur, prev, { money });
  if (!d) return null;
  const cls = d.direction === 'up' ? 'up' : d.direction === 'down' ? 'down' : 'flat';
  const arrow = d.direction === 'up' ? '▲' : d.direction === 'down' ? '▼' : '—';
  // the arrow carries the sign for percentages; absolute deltas keep theirs
  const text = d.kind === 'pct' ? d.text.replace(/^[+-]/, '') : d.text;
  return (
    <span className="fc-delta" data-delta-kind={d.kind} title={`${title}: ${d.title}`}>
      <span className={cls}>{arrow}</span> {text}
    </span>
  );
}

function Count({ n, delay = 500, money = false }: { n: number; delay?: number; money?: boolean }) {
  const v = useCountUp(n, delay);
  return <>{money ? usd(v) : num(v)}</>;
}

/* ---------- hero: stepped hatched funnel ---------- */

function StepFunnel({
  stages,
  active,
  onSelect,
  openValue,
}: {
  stages: ReturnType<typeof funnelSummary>['stages'];
  active: FunnelStage;
  onSelect: (s: FunnelStage) => void;
  openValue: number;
}) {
  const W = 1000;
  const H = 260;
  const max = Math.max(1, stages[0]?.total ?? 1);
  const bars = stages.map((s, i) => {
    const h = Math.max(s.total > 0 ? 6 : 0, (s.total / max) * 246);
    return { ...s, i, x: 52 + i * 200, y: H - h, h };
  });
  const ai = Math.max(
    0,
    stages.findIndex((s) => s.stage === active),
  );
  const a = bars[ai];
  const ticks = [max, Math.round(max * 0.75), Math.round(max * 0.5), Math.round(max * 0.25)];
  const conv = a?.conversionFromPrev;
  const drop = conv === null || conv === undefined ? null : Math.round(100 - conv);
  return (
    <div className="fc-funnel">
      <div className="fc-funnel-labels">
        {bars.map((b) => (
          <div
            key={b.stage}
            className={`fc-fl${b.stage === active ? ' active' : ''}`}
            onClick={() => onSelect(b.stage)}
            role="button"
            tabIndex={0}
            aria-pressed={b.stage === active}
            title={`Focus ${stageLabel(b.stage).toLowerCase()}`}
            onKeyDown={(e) => e.key === 'Enter' && onSelect(b.stage)}
          >
            <div className="name">{stageLabel(b.stage)}</div>
            <div className="val">
              <Count n={b.total} delay={300 + b.i * 120} />
            </div>
          </div>
        ))}
      </div>
      <div className="fc-funnel-stage">
        <div className="fc-yaxis">
          {ticks.map((t, i) => (
            <span key={i}>{t}</span>
          ))}
        </div>
        <svg className="fc-funnel-svg" viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none" aria-hidden>
          <defs>
            <pattern id="fc-hatch" width="11" height="11" patternTransform="rotate(45)" patternUnits="userSpaceOnUse">
              <rect width="11" height="11" style={{ fill: 'var(--fc-hero-hatch)' }} />
              <rect width="4.5" height="11" style={{ fill: 'var(--fc-hero-stripe)' }} opacity=".55" />
            </pattern>
            <linearGradient id="fc-fadeB" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0" stopColor="white" stopOpacity="1" />
              <stop offset="1" stopColor="white" stopOpacity=".12" />
            </linearGradient>
            <linearGradient id="fc-solid" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0" style={{ stopColor: 'var(--fc-hero-a)' }} />
              <stop offset="1" style={{ stopColor: 'var(--fc-hero-b)' }} />
            </linearGradient>
            <mask id="fc-fade">
              <rect x="0" y="0" width={W} height={H} fill="url(#fc-fadeB)" />
            </mask>
          </defs>
          <g mask="url(#fc-fade)">
            {bars.map((b, i) => {
              if (b.stage === active) return null;
              const next = bars[i + 1];
              return (
                <g
                  key={b.stage}
                  className="fc-bar"
                  style={{ animationDelay: `${0.25 + i * 0.15}s` }}
                  onClick={() => onSelect(b.stage)}
                >
                  <rect x={b.x} y={b.y} width="140" height={b.h} fill="url(#fc-hatch)" />
                  {next && next.h > 0 && (
                    <polygon
                      points={`${b.x + 140},${b.y} ${next.x},${next.y} ${next.x},${H} ${b.x + 140},${H}`}
                      fill="url(#fc-hatch)"
                      opacity=".55"
                    />
                  )}
                </g>
              );
            })}
          </g>
          {/* the active stage rides outside the fade mask: full-strength mint */}
          {a && (
            <g className="fc-bar" style={{ animationDelay: `${0.25 + ai * 0.15}s` }} key={`active-${a.stage}`}>
              <rect x={a.x} y={a.y} width="140" height={a.h} fill="url(#fc-solid)" />
              {bars[ai + 1] && bars[ai + 1].h > 0 && (
                <polygon
                  points={`${a.x + 140},${a.y} ${bars[ai + 1].x},${bars[ai + 1].y} ${bars[ai + 1].x},${H} ${a.x + 140},${H}`}
                  fill="url(#fc-solid)"
                  opacity=".7"
                />
              )}
            </g>
          )}
          {bars.map((b, i) =>
            b.h > 0 ? (
              <rect
                key={`cap-${b.stage}`}
                className="fc-bar"
                style={{ animationDelay: `${0.5 + i * 0.1}s`, fill: 'var(--fc-cap)' }}
                x={b.x + 52}
                y={b.y - 8}
                width="36"
                height="7"
                rx="3.5"
              />
            ) : null,
          )}
        </svg>
        {a && (
          <div
            className="fc-tip"
            key={a.stage}
            style={{
              left: `${((a.x + 70) / W) * 100}%`,
              top: `${Math.max(4, (a.y / H) * 100 - 20)}%`,
            }}
          >
            <b>
              {num(a.total)} {stageLabel(a.stage).toLowerCase()}
            </b>
            {' · '}organic <b>{a.organic}</b> / ads <b>{a.ads}</b>
            {drop !== null && (
              <>
                {' | '}drop-off <b>−{drop}%</b>
              </>
            )}
            {openValue > 0 && (
              <>
                {' | '}value <b>{usd(openValue)}</b>
              </>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

/* ---------- explore bar ---------- */

type Command = { token: string; hint: string; run: () => void };

function PromptBar({ commands, inputRef }: { commands: Command[]; inputRef: React.RefObject<HTMLInputElement> }) {
  const [text, setText] = useState('');
  const [focused, setFocused] = useState(false);
  const [typed, setTyped] = useState('');
  const [tokenShown, setTokenShown] = useState(false);
  const [note, setNote] = useState<string | null>(null);
  const reduced = useReducedMotion();

  // The bar types its own example on load, then hands over to the real input.
  useEffect(() => {
    const phrase = 'Show me who is stalling after ';
    if (reduced) {
      setTyped(phrase);
      setTokenShown(true);
      return;
    }
    let i = 0;
    let t: ReturnType<typeof setTimeout>;
    const start = setTimeout(function type() {
      if (i <= phrase.length) {
        setTyped(phrase.slice(0, i++));
        t = setTimeout(type, 34);
      } else {
        setTokenShown(true);
      }
    }, 1700);
    return () => {
      clearTimeout(start);
      clearTimeout(t);
    };
  }, [reduced]);

  const matching = useMemo(() => {
    const m = text.match(/\/([\w-]*)$/);
    if (!m) return [];
    const q = m[1].toLowerCase();
    return commands.filter((c) => c.token.slice(1).startsWith(q));
  }, [text, commands]);

  const submit = () => {
    const hit = commands.find((c) => text.toLowerCase().includes(c.token));
    if (hit) {
      hit.run();
      setText('');
      setNote(null);
    } else {
      setNote('No token in that line. Type / to see what this bar can jump to.');
    }
  };

  const showOverlay = !focused && text === '';
  return (
    <div className="fc-prompt">
      <div className="fc-prompt-head">
        <span className="spark">{I.spark}</span>
        What would you like to explore next?
        <span className="hint">Slash tokens jump the page. No model behind this bar.</span>
      </div>
      <div className="fc-input" onClick={() => inputRef.current?.focus()}>
        <input
          ref={inputRef}
          value={text}
          onChange={(e) => {
            setText(e.target.value);
            setNote(null);
          }}
          onFocus={() => setFocused(true)}
          onBlur={() => setFocused(false)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') submit();
            if (e.key === 'Escape') {
              setText('');
              inputRef.current?.blur();
            }
          }}
          aria-label="Explore the funnel with a slash token"
        />
        {showOverlay && (
          <span className="fc-typed">
            {typed}
            <span className={`fc-token${tokenShown ? ' show' : ''}`}>/engaged</span>
            <span className="fc-caret" />
          </span>
        )}
      </div>
      {(focused || matching.length > 0) && (
        <div className="fc-slash">
          {(matching.length > 0 ? matching : commands).map((c) => (
            <button
              key={c.token}
              type="button"
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => {
                c.run();
                setText('');
                setNote(null);
              }}
            >
              {c.token}
              <span>{c.hint}</span>
            </button>
          ))}
        </div>
      )}
      {note && <div className="fc-prompt-note">{note}</div>}
    </div>
  );
}

/* ---------- step-line chart ---------- */

function StepLine({ series, labels, unit }: { series: number[]; labels: string[]; unit: string }) {
  const W = 320;
  const H = 170;
  const n = Math.max(1, series.length);
  const sw = W / n;
  const max = Math.max(1, ...series);
  const y = (v: number) => 150 - (v / max) * 126;
  let line = `M0,${y(series[0] ?? 0)}`;
  for (let i = 1; i < n; i++) line += ` H${i * sw} V${y(series[i])}`;
  line += ` H${W}`;
  const area = `${line} V${H} H0 Z`;
  const peak = series.indexOf(Math.max(...series));
  const cx = (peak + 0.5) * sw;
  const every = n > 8 ? 3 : 1;
  const any = series.some((v) => v > 0);
  return (
    <div className="fc-step">
      {any && (
        <div className="fc-peak" style={{ left: `${(cx / W) * 100}%` }}>
          {num(series[peak])} {unit}
        </div>
      )}
      <div className="fc-step-chart">
        <svg viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none" className="fc-step-svg" aria-hidden>
          <defs>
            <pattern id="fc-pin" width="7" height="10" patternUnits="userSpaceOnUse">
              <rect width="2.4" height="10" style={{ fill: 'var(--fc-line)' }} opacity=".22" />
            </pattern>
            <linearGradient id="fc-pinFade" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0" stopColor="white" />
              <stop offset="1" stopColor="white" stopOpacity=".1" />
            </linearGradient>
            <mask id="fc-pinMask">
              <rect width={W} height={H} fill="url(#fc-pinFade)" />
            </mask>
          </defs>
          <g mask="url(#fc-pinMask)">
            <path d={area} fill="url(#fc-pin)" />
          </g>
          <path
            className="fc-step-line"
            d={line}
            fill="none"
            style={{ stroke: 'var(--fc-line)' }}
            strokeWidth="3"
            strokeLinejoin="round"
            vectorEffect="non-scaling-stroke"
          />
        </svg>
        {any && (
          <span
            className="fc-peak-dot"
            style={{ left: `${(cx / W) * 100}%`, top: `${(y(series[peak]) / H) * 100}%` }}
          />
        )}
      </div>
      <div className="fc-months">
        {labels.map((l, i) => (
          <span key={i} style={i % every ? { visibility: 'hidden' } : undefined}>
            {l}
          </span>
        ))}
      </div>
    </div>
  );
}

/* ---------- dot matrix ---------- */

function DotMatrix({ cols, color, delay = 0.9 }: { cols: number[]; color: string; delay?: number }) {
  const max = Math.max(...cols);
  if (max === 0) return <div className="fc-empty">Nothing in this period</div>;
  const scale = (n: number) => (n === 0 ? 0 : Math.max(1, Math.round((n / max) * 7)));
  return (
    <div className="fc-dots">
      {cols.map((n, i) => {
        const cls = n === max ? '' : n >= max * 0.6 ? 'mid' : 'low';
        return (
          <div key={i} className={`fc-dcol ${cls}`} title={`${WEEKDAYS[i]}: ${n}`}>
            {Array.from({ length: scale(n) }).map((_, j) => (
              <div
                key={j}
                className="fc-dot"
                style={{ ['--c' as string]: color, animationDelay: `${delay + i * 0.06 + j * 0.05}s` }}
              />
            ))}
          </div>
        );
      })}
    </div>
  );
}

/* ---------- lead cells ---------- */

function Contact({ j, full = false }: { j: FunnelJourney; full?: boolean }) {
  const digits = j.phone?.replace(/[^\d]/g, '');
  const isGhl = j.url?.includes('gohighlevel');
  const crm = isGhl ? 'GoHighLevel' : 'Attio';
  return (
    <span className={full ? 'fc-contact full' : 'fc-contact'} onClick={(e) => e.stopPropagation()}>
      {j.email && (
        <a href={`mailto:${j.email}`} title={j.email}>
          {full ? j.email : 'email'}
        </a>
      )}
      {digits && (
        <a href={`https://wa.me/${digits}`} target="_blank" rel="noopener noreferrer" title={j.phone ?? undefined}>
          {full ? `WhatsApp ${j.phone}` : 'wa'}
        </a>
      )}
      {j.phone && full && <a href={`sms:${j.phone}`}>SMS</a>}
      {j.url && (
        <a href={j.url} target="_blank" rel="noopener noreferrer">
          {full ? `Open in ${crm}` : isGhl ? 'ghl↗' : 'attio↗'}
        </a>
      )}
      {j.linkedin && full && (
        <a href={j.linkedin} target="_blank" rel="noopener noreferrer">
          LinkedIn
        </a>
      )}
      {!j.email && !digits && !j.url && <span className="dim">{full ? 'No contact channel on record' : '—'}</span>}
    </span>
  );
}

function QuietCell({ j, now }: { j: FunnelJourney; now: Date }) {
  const meta = journeyMeta(j, now);
  const d = decayFactor(meta.daysSinceLastTouch, j.status);
  return (
    <span
      className="mono"
      title={
        d > 0
          ? `Fading: ${Math.round(d * 100)}% of the way to the archive`
          : meta.state === 'stalled'
            ? 'Stalled: quiet past a week before converting'
            : undefined
      }
      style={
        d > 0
          ? { color: `color-mix(in oklab, var(--err) ${Math.round(Math.sqrt(d) * 85)}%, var(--text-2))` }
          : meta.state === 'stalled'
            ? { color: 'var(--err)' }
            : { color: 'var(--text-3)' }
      }
    >
      {meta.daysSinceLastTouch}d
    </span>
  );
}

function Touches({ j, now }: { j: FunnelJourney; now: Date }) {
  const converted = j.status === 'converted';
  return (
    <div className="fc-touches">
      {j.touches.map((t, i) => (
        <span key={t.id} style={{ display: 'inline-flex', gap: 6 }}>
          {i > 0 && <span className="fc-arrow">→</span>}
          <span className="fc-touch" title={`${stageLabel(t.stage)} · via ${t.source} · ${t.at}`}>
            <span className="g">{CHANNEL_GLYPHS[t.channel] ?? '·'}</span>
            <span className="l">{t.label}</span>
            <span className="d">{t.at.slice(5)}</span>
          </span>
        </span>
      ))}
      {!converted && <span className="fc-arrow">→ …</span>}
      <span className="fc-arrow" style={{ marginLeft: 'auto' }}>
        {journeyMeta(j, now).daysSinceLastTouch}d quiet
      </span>
    </div>
  );
}

/* ---------- lead drawer ---------- */

function LeadDrawer({
  j,
  now,
  archived,
  onClose,
  onStage,
}: {
  j: FunnelJourney;
  now: Date;
  archived: boolean;
  onClose: () => void;
  onStage: (s: FunnelStage) => void;
}) {
  const ref = useRef<HTMLDivElement>(null);
  useDismiss(true, ref, onClose);
  const meta = journeyMeta(j, now);
  const decay = decayFactor(meta.daysSinceLastTouch, j.status);
  const reached = stageIndex(j.status);
  const lane = laneOf(j);
  return (
    <div className="fc-layer">
      <div className="fc-scrim" aria-hidden />
      <aside className="fc-drawer" ref={ref} role="dialog" aria-label={`${j.person ?? j.name} dossier`}>
        <div className="fc-drawer-head">
          <div>
            <div className="fc-drawer-name">{j.person ?? j.name}</div>
            <div className="fc-drawer-sub">{[j.company, j.role].filter(Boolean).join(' · ') || j.name}</div>
          </div>
          <button type="button" className="fc-icon-btn" aria-label="Close" onClick={onClose}>
            {I.close}
          </button>
        </div>

        <div className="fc-stepper" aria-label="Stage reached">
          {FUNNEL_STAGES.map((s, i) => (
            <button
              key={s.id}
              type="button"
              className={i <= reached ? 'on' : ''}
              title={`${s.label}: focus this stage on the board`}
              onClick={() => onStage(s.id)}
            >
              <span className="bar" />
              <span className="lab">{s.label}</span>
            </button>
          ))}
        </div>

        <dl className="fc-facts">
          <dt>Status</dt>
          <dd>
            {archived
              ? `Archived · quiet ${meta.daysSinceLastTouch}d, past the ${DECAY_DAYS}-day line`
              : meta.state === 'converted'
                ? 'Converted'
                : meta.state === 'stalled'
                  ? `Stalled · quiet ${meta.daysSinceLastTouch}d`
                  : decay > 0
                    ? `Fading · quiet ${meta.daysSinceLastTouch}d, ${Math.round(decay * 100)}% of the way to the archive`
                    : `Active · quiet ${meta.daysSinceLastTouch}d`}
          </dd>
          <dt>Relationship</dt>
          <dd style={{ textTransform: 'capitalize', color: `var(--funnel-${j.relationship})` }}>{j.relationship}</dd>
          <dt>Likelihood</dt>
          <dd className="mono">{j.likelihood}%</dd>
          <dt>Entry</dt>
          <dd>
            <span className="sw" style={{ ['--c' as string]: LANE_VAR[lane] }} /> {LANE_LABEL[lane]}
          </dd>
          <dt>Value</dt>
          <dd className="mono">
            {(j.amountUsd ?? 0) > 0 ? usd(j.amountUsd ?? 0) : '—'}
            {j.product ? ` · ${j.product}` : ''}
            {j.status !== 'converted' && (j.amountUsd ?? 0) > 0 ? ' (open)' : ''}
          </dd>
          <dt>First seen</dt>
          <dd>{fmtDate(firstAt(j))}</dd>
          <dt>Deal</dt>
          <dd>{j.name}</dd>
        </dl>

        <div className="fc-drawer-label">Contact</div>
        <Contact j={j} full />

        <div className="fc-drawer-label">Journey · {j.touches.length} touches</div>
        <ol className="fc-timeline">
          {j.touches.map((t) => (
            <li key={t.id}>
              <span className="when">{fmtDate(t.at)}</span>
              <span className="what">
                <span className="g">{CHANNEL_GLYPHS[t.channel] ?? '·'}</span>
                {t.label}
              </span>
              <span className="who">
                {stageLabel(t.stage)} · {t.source}
              </span>
            </li>
          ))}
          {j.status !== 'converted' && <li className="next">Next touch is yours</li>}
        </ol>
      </aside>
    </div>
  );
}

/* ================================================================ */

export function FunnelDashboard({
  journeys,
  archived,
  nowIso,
  source,
  connectors,
  pushNow,
  saveNow,
  initial,
  sampleOffer = false,
  channels,
  trakyo,
}: DashboardProps) {
  const router = useRouter();
  const [refreshing, startRefresh] = useTransition();
  const now = useMemo(() => new Date(nowIso), [nowIso]);
  const summary = useMemo(() => funnelSummary(journeys), [journeys]);

  // Where the funnel leaks most: the hero's default focus.
  const defaultStage = useMemo<FunnelStage>(() => {
    let worst: FunnelStage = 'opted_in';
    let worstConv = Infinity;
    for (const s of summary.stages) {
      if (s.conversionFromPrev !== null && s.conversionFromPrev < worstConv) {
        worstConv = s.conversionFromPrev;
        worst = s.stage;
      }
    }
    return worst;
  }, [summary]);

  const [view, setView] = useState<View>(initial.view ?? 'overview');
  const [period, setPeriod] = useState<Period>(initial.period ?? 90);
  const [grain, setGrain] = useState<Grain>(initial.grain ?? 'monthly');
  const [compare, setCompare] = useState<Compare>(initial.compare ?? 'previous');
  const [stage, setStage] = useState<FunnelStage>(initial.stage ?? defaultStage);
  const [segment, setSegment] = useState<FunnelStage | undefined>(initial.stage);
  const [lane, setLane] = useState<Lane | undefined>(initial.lane);
  const [selected, setSelected] = useState<string | undefined>(initial.lead);
  const [copied, setCopied] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  /* ---- the URL mirrors the view, so a copied link reproduces it ---- */
  useEffect(() => {
    const p = new URLSearchParams(window.location.search);
    const put = (k: string, v: string | number | undefined, def?: string | number) => {
      if (v === undefined || v === def) p.delete(k);
      else p.set(k, String(v));
    };
    put('view', view, 'overview');
    put('stage', stage, defaultStage);
    put('period', period, 90);
    put('grain', grain, 'monthly');
    put('compare', compare, 'previous');
    put('lane', lane);
    put('lead', selected);
    const qs = p.toString();
    const next = `${window.location.pathname}${qs ? `?${qs}` : ''}`;
    if (next !== `${window.location.pathname}${window.location.search}`) window.history.replaceState(null, '', next);
  }, [view, stage, defaultStage, period, grain, compare, lane, selected]);

  /* ---- period windows ---- */
  const stats = useMemo(() => {
    const end = now.getTime() + DAY;
    const days = period || 100_000;
    const from = end - days * DAY;
    const cmp: [number, number] | null =
      period === 0 || compare === 'none'
        ? null
        : compare === 'year'
          ? [from - 365 * DAY, end - 365 * DAY]
          : [from - days * DAY, from];
    const inWin = (ts: number, lo: number, hi: number) => ts >= lo && ts < hi;
    const inCmp = (ts: number) => (cmp ? inWin(ts, cmp[0], cmp[1]) : false);

    const leadsIn = journeys.filter((j) => inWin(dateOf(firstAt(j)), from, end));
    const leadsPrev = journeys.filter((j) => inCmp(dateOf(firstAt(j))));
    const convIn = journeys.filter((j) => j.status === 'converted' && inWin(dateOf(convertedAt(j)), from, end));
    const convPrev = journeys.filter((j) => j.status === 'converted' && inCmp(dateOf(convertedAt(j))));
    const revenue = convIn.reduce((s, j) => s + (j.amountUsd ?? 0), 0);
    const revenuePrev = convPrev.reduce((s, j) => s + (j.amountUsd ?? 0), 0);
    const byLane: Record<Lane, number> = { organic: 0, ads: 0, crm: 0 };
    for (const j of convIn) byLane[laneOf(j)] += j.amountUsd ?? 0;
    const wk = (list: FunnelJourney[], at: (j: FunnelJourney) => string) => {
      const cols = [0, 0, 0, 0, 0, 0, 0];
      for (const j of list) cols[weekday(at(j))]++;
      return cols;
    };
    const leadCols = wk(leadsIn, firstAt);
    const convCols = wk(convIn, convertedAt);
    const peakOf = (cols: number[]) => (Math.max(...cols) > 0 ? WEEKDAYS[cols.indexOf(Math.max(...cols))] : null);
    return {
      from,
      end,
      cmp,
      leadsIn,
      leadsPrev,
      convIn,
      convPrev,
      revenue,
      revenuePrev,
      byLane,
      leadCols,
      convCols,
      leadPeak: peakOf(leadCols),
      convPeak: peakOf(convCols),
    };
  }, [journeys, now, period, compare]);

  /* ---- touches over time (all journeys incl. archived: it is history) ---- */
  const touchSeries = useMemo(() => {
    const all = [...journeys, ...archived].flatMap((j) => j.touches);
    if (grain === 'monthly') {
      const buckets: { key: string; label: string; n: number }[] = [];
      for (let k = 5; k >= 0; k--) {
        const d = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth() - k, 1));
        buckets.push({ key: d.toISOString().slice(0, 7), label: MONTHS[d.getUTCMonth()], n: 0 });
      }
      for (const t of all) {
        const b = buckets.find((x) => x.key === t.at.slice(0, 7));
        if (b) b.n++;
      }
      return { series: buckets.map((b) => b.n), labels: buckets.map((b) => b.label) };
    }
    const weeks = 12;
    const end = Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate()) + DAY;
    const series = Array.from({ length: weeks }, () => 0);
    const labels = Array.from({ length: weeks }, (_, i) => fmtDay(end - (weeks - i) * 7 * DAY));
    for (const t of all) {
      const idx = weeks - 1 - Math.floor((end - 1 - dateOf(t.at)) / (7 * DAY));
      if (idx >= 0 && idx < weeks) series[idx]++;
    }
    return { series, labels };
  }, [journeys, archived, grain, now]);

  /* ---- insight copy: every number computed above ---- */
  const conversion = summary.clients > 0 ? Math.round((summary.converted / summary.clients) * 100) : 0;
  const worstStep = useMemo((): { from: FunnelStage; to: FunnelStage; drop: number } | null => {
    let w: { from: FunnelStage; to: FunnelStage; drop: number } | null = null;
    for (let i = 1; i < summary.stages.length; i++) {
      const s = summary.stages[i];
      if (s.conversionFromPrev === null) continue;
      const drop = Math.round(100 - s.conversionFromPrev);
      if (w === null || drop > w.drop) w = { from: summary.stages[i - 1].stage, to: s.stage, drop };
    }
    return w;
  }, [summary]);
  const openValue = journeys.filter((j) => j.status === stage).reduce((s, j) => s + (j.amountUsd ?? 0), 0);

  const byId = useMemo(() => new Map([...journeys, ...archived].map((j) => [j.id, j])), [journeys, archived]);
  const archivedIds = useMemo(() => new Set(archived.map((j) => j.id)), [archived]);
  const pushList = pushNow.map((id) => byId.get(id)).filter(Boolean) as FunnelJourney[];
  const saveList = saveNow.map((id) => byId.get(id)).filter(Boolean) as FunnelJourney[];
  const selectedLead = selected ? byId.get(selected) : undefined;
  const tableJourneys = journeys.filter((j) => (!segment || j.status === segment) && (!lane || laneOf(j) === lane));
  const stageCounts = new Map<FunnelStage, number>();
  for (const j of journeys) stageCounts.set(j.status, (stageCounts.get(j.status) ?? 0) + 1);
  const laneCounts: Record<Lane, number> = { organic: 0, ads: 0, crm: 0 };
  for (const j of journeys) laneCounts[laneOf(j)]++;

  const goJourneys = (s?: FunnelStage, l?: Lane) => {
    setSegment(s);
    setLane(l);
    if (s) setStage(s);
    setView('journeys');
  };

  const commands: Command[] = [
    ...FUNNEL_STAGES.map((s) => ({
      token: `/${s.id.replace('_', '-')}`,
      hint: `${s.label} segment`,
      run: () => goJourneys(s.id, lane),
    })),
    ...LANES.map((l) => ({ token: `/${l}`, hint: `${LANE_LABEL[l]} entries`, run: () => goJourneys(segment, l) })),
    { token: '/push', hint: 'Hot and moving', run: () => setView('push') },
    { token: '/save', hint: 'Fading toward the archive', run: () => setView('save') },
    { token: '/archive', hint: `Decayed past ${DECAY_DAYS}d`, run: () => setView('archive') },
    { token: '/overview', hint: 'Back to the board', run: () => setView('overview') },
  ];

  const revenueShown = useCountUp(stats.revenue, 500);
  const leadsShown = useCountUp(stats.leadsIn.length, 600);
  const convShown = useCountUp(stats.convIn.length, 700);
  const convRateShown = useCountUp(conversion, 800);
  const laneMax = Math.max(1, ...Object.values(stats.byLane));

  const copyLink = async () => {
    try {
      await navigator.clipboard.writeText(window.location.href);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      window.prompt('Copy this link', window.location.href);
    }
  };

  const crm = connectors.find((c) => c.ok && (c.id === 'attio' || c.id === 'ghl'));
  const crmHref = crm ? (crm.id === 'ghl' ? 'https://app.gohighlevel.com' : 'https://app.attio.com') : undefined;
  const cmpLabel = stats.cmp
    ? `${fmtDay(stats.cmp[0], compare === 'year')} – ${fmtDay(stats.cmp[1] - DAY, compare === 'year')}`
    : null;
  const vsLabel = compare === 'year' ? 'vs last year' : 'vs previous period';
  // H13: every count states its window. Period cards count inside the
  // selected period; the Pipeline counts every active journey.
  const periodWindow = PERIODS.find((p) => p.id === period)?.label ?? 'Selected period';

  const NAV: { id: View; label: string; n?: number }[] = [
    { id: 'overview', label: 'Overview' },
    { id: 'push', label: 'Push now', n: pushList.length },
    { id: 'save', label: 'Save now', n: saveList.length },
    { id: 'journeys', label: 'Journeys', n: journeys.length },
    { id: 'archive', label: 'Archive', n: archived.length },
  ];

  const leadRow = (j: FunnelJourney) => ({
    className: `row${selected === j.id ? ' on' : ''}`,
    onClick: () => setSelected(j.id),
    tabIndex: 0,
    onKeyDown: (e: React.KeyboardEvent) => e.key === 'Enter' && setSelected(j.id),
    title: 'Open dossier',
  });

  return (
    <div className="fc">
      <div className="fc-slab">
        {/* ---- topbar ---- */}
        <div className="fc-topbar">
          <nav className="fc-nav" aria-label="Funnel views">
            {NAV.map((n) => (
              <button
                key={n.id}
                type="button"
                className={view === n.id ? 'active' : ''}
                aria-current={view === n.id ? 'page' : undefined}
                onClick={() => setView(n.id)}
              >
                {n.label}
                {n.n !== undefined && <span className="n">{n.n}</span>}
              </button>
            ))}
          </nav>
          <div className="fc-actions">
            <button
              type="button"
              className={`fc-icon-btn${refreshing ? ' spin' : ''}`}
              aria-label="Refresh from the sources"
              title="Re-read the CRM and local journeys"
              disabled={refreshing}
              onClick={() => startRefresh(() => router.refresh())}
            >
              {I.refresh}
            </button>
            {crmHref ? (
              <a
                href={crmHref}
                target="_blank"
                rel="noopener noreferrer"
                className="fc-pill"
                title={`Open ${crm?.name} in a new tab`}
              >
                {I.external} Open {crm?.name}
              </a>
            ) : (
              <SetupLink
                href={connectors.find((c) => c.id === 'attio')?.href ?? '/integrations'}
                label="Connect a CRM"
                hint="Key Attio or GoHighLevel and the board reads the live pipeline"
              />
            )}
          </div>
        </div>

        {/* ---- title row ---- */}
        <div className="fc-title-row">
          <h1 className="fc-h1">Funnel</h1>
          <button
            type="button"
            className="fc-link-chip"
            onClick={copyLink}
            title="Copy a link to exactly this view"
            aria-label="Copy a link to this view"
          >
            {copied ? I.check : I.link}
          </button>
          {copied && <span className="fc-copied">Link copied</span>}
          <div className="fc-filters">
            <Dropdown value={period} options={PERIODS} icon={I.calendar} onChange={setPeriod} label="Period" />
            <span className="fc-pill ghost">compared to</span>
            {period === 0 ? (
              <span className="fc-pill ghost" title="All time has no earlier window to compare against">
                nothing
              </span>
            ) : (
              <Dropdown
                value={compare}
                label="Comparison window"
                icon={I.calendar}
                options={[
                  { id: 'previous', label: compare === 'previous' && cmpLabel ? cmpLabel : 'Previous period' },
                  { id: 'year', label: compare === 'year' && cmpLabel ? cmpLabel : 'Same period last year' },
                  { id: 'none', label: 'No comparison' },
                ]}
                onChange={setCompare}
              />
            )}
            <Dropdown
              value={grain}
              label="Touches chart grain"
              options={[
                { id: 'monthly', label: 'Monthly' },
                { id: 'weekly', label: 'Weekly' },
              ]}
              onChange={setGrain}
            />
          </div>
        </div>

        {/* ================= OVERVIEW ================= */}
        {view === 'overview' && (
          <div className="fc-grid">
            {/* hero: pipeline funnel */}
            <div className="fc-card fc-c1">
              <div className="fc-card-head">
                <div className="fc-card-title">
                  Pipeline<span className="fc-card-window">All active journeys</span>
                </div>
                <Kebab label="Pipeline" period={period} onPeriod={setPeriod} onJourneys={() => goJourneys(stage)} />
              </div>
              {summary.clients === 0 ? (
                <div className="fc-panel-empty" style={{ border: 0, padding: '80px 0' }}>
                  No journeys yet. Connect a CRM and the funnel fills itself.
                </div>
              ) : (
                <StepFunnel stages={summary.stages} active={stage} onSelect={setStage} openValue={openValue} />
              )}
              <PromptBar commands={commands} inputRef={inputRef} />
            </div>

            {/* revenue + meters */}
            <div className="fc-card fc-c2">
              <div className="fc-card-head">
                <div className="fc-card-title">
                  Revenue<span className="fc-card-window">{periodWindow}</span>
                </div>
                <Kebab label="Revenue" period={period} onPeriod={setPeriod} onJourneys={() => goJourneys('converted')} />
              </div>
              <div className="fc-num-row">
                <div className="fc-num">{usd(revenueShown)}</div>
                {stats.cmp && <Delta cur={stats.revenue} prev={stats.revenuePrev} money title={vsLabel} />}
              </div>
              <div className="fc-divider" />
              {LANES.map((l) => (
                <button
                  type="button"
                  className="fc-meter"
                  key={l}
                  title={`See the ${LANE_LABEL[l].toLowerCase()} deals that closed`}
                  onClick={() => goJourneys('converted', l)}
                >
                  <div className="fc-meter-row">
                    <span className="lab">{LANE_LABEL[l]}</span>
                    <span className="val">{usd(stats.byLane[l])}</span>
                  </div>
                  <div className="fc-track">
                    <div
                      className="fc-fill"
                      style={{
                        ['--c' as string]: LANE_VAR[l],
                        ['--w' as string]: `${Math.round((stats.byLane[l] / laneMax) * 100)}%`,
                      }}
                    />
                  </div>
                </button>
              ))}
              <div className="fc-sources" style={{ marginTop: 4 }}>
                {stats.convIn.length} closed in the period, split by how each deal entered
              </div>
            </div>

            {/* touches step-line */}
            <div className="fc-card fc-c3">
              <div className="fc-card-head">
                <div className="fc-card-title">Touches</div>
                <Kebab label="Touches" period={period} onPeriod={setPeriod} onJourneys={() => goJourneys()} />
              </div>
              <StepLine series={touchSeries.series} labels={touchSeries.labels} unit="touches" />
            </div>

            {/* dot matrices */}
            <div className="fc-card fc-c4">
              <div className="fc-sub">
                <div className="fc-card-head">
                  <div className="fc-card-title">
                    New leads<span className="fc-card-window">{periodWindow}</span>
                  </div>
                  <Kebab label="New leads" period={period} onPeriod={setPeriod} onJourneys={() => goJourneys('first_touch')} />
                </div>
                <div className="fc-stat">
                  <div className="fc-big">{num(leadsShown)}</div>
                  <div className="fc-dots-wrap">
                    {stats.leadPeak && (
                      <div className="fc-mini-pill">
                        Peak: <b>{stats.leadPeak}</b>
                      </div>
                    )}
                    <DotMatrix cols={stats.leadCols} color="var(--fc-dots)" />
                  </div>
                  {stats.cmp && (
                    <div className="fc-vs">
                      {vsLabel}
                      <b>
                        {stats.leadsIn.length - stats.leadsPrev.length >= 0 ? '+' : ''}
                        {num(stats.leadsIn.length - stats.leadsPrev.length)}
                      </b>
                    </div>
                  )}
                </div>
              </div>
              <div className="fc-sub">
                <div className="fc-card-head">
                  <div className="fc-card-title">
                    Converted<span className="fc-card-window">{periodWindow}</span>
                  </div>
                  <Kebab label="Converted" period={period} onPeriod={setPeriod} onJourneys={() => goJourneys('converted')} />
                </div>
                <div className="fc-stat">
                  <div className="fc-big">{num(convShown)}</div>
                  <div className="fc-dots-wrap">
                    {stats.convPeak && (
                      <div className="fc-mini-pill">
                        Peak: <b>{stats.convPeak}</b>
                      </div>
                    )}
                    <DotMatrix cols={stats.convCols} color="var(--fc-dots-2)" delay={1.1} />
                  </div>
                  {stats.cmp && (
                    <div className="fc-vs">
                      {vsLabel}
                      <b>
                        {stats.convIn.length - stats.convPrev.length >= 0 ? '+' : ''}
                        {num(stats.convIn.length - stats.convPrev.length)}
                      </b>
                    </div>
                  )}
                </div>
              </div>
            </div>

            {/* the one gradient card */}
            <div className="fc-card fc-c5">
              <div className="fc-ins-pill">{I.bulb} Insights</div>
              <div className="fc-ins-num">{convRateShown}%</div>
              <div className="fc-ins-head">
                {summary.clients === 0
                  ? 'Nothing to read yet.'
                  : worstStep
                    ? `${summary.converted} of ${summary.clients} leads converted. The leak is ${stageLabel(worstStep.from)} → ${stageLabel(worstStep.to)}.`
                    : `${summary.converted} of ${summary.clients} leads converted.`}
              </div>
              <div className="fc-ins-sub">
                {summary.clients === 0
                  ? 'Connect Attio or GHL and this card starts reading the pipeline.'
                  : `${worstStep ? `${worstStep.drop}% drop at that step. ` : ''}${pushList.length} hot lead${pushList.length === 1 ? '' : 's'} in motion, ${saveList.length} fading toward the archive, ${archived.length} already there.`}
              </div>
              <div className="fc-ins-actions">
                {worstStep && (
                  <button type="button" onClick={() => goJourneys(worstStep.from)}>
                    Who is stuck at {stageLabel(worstStep.from).toLowerCase()}
                  </button>
                )}
                {saveList.length > 0 && (
                  <button type="button" onClick={() => setView('save')}>
                    Save the {saveList.length} fading
                  </button>
                )}
              </div>
              <div className="fc-ins-dashes" aria-label="Focused stage">
                {FUNNEL_STAGES.map((s, i) => (
                  <button
                    key={s.id}
                    type="button"
                    className={i <= stageIndex(stage) ? 'on' : ''}
                    title={`Focus ${s.label.toLowerCase()}`}
                    onClick={() => setStage(s.id)}
                  />
                ))}
              </div>
            </div>

            {/* F6 link studio and F7 channel analytics (FounderOS merge), below the pipeline */}
            {trakyo && <FunnelLinks configured={trakyo.configured} setupHref={trakyo.setupHref} />}
            {channels && <FunnelChannels card={channels} />}
          </div>
        )}

        {/* ================= PUSH / SAVE ================= */}
        {(view === 'push' || view === 'save') && (
          <div className="fc-two">
            {(
              [
                { id: 'push', title: 'Push now', sub: 'Hot and moving. Close them.', dot: 'ok', list: pushList, empty: 'No hot leads in motion right now' },
                { id: 'save', title: 'Save now', sub: 'Fading toward the archive, highest likelihood first', dot: 'err', list: saveList, empty: 'Nothing fading. Every lead is fresh.' },
              ] as const
            ).map((q) => (
              <div className={`fc-panel${view === q.id ? '' : ' quiet'}`} key={q.id}>
                <div className="fc-panel-head">
                  <span className="t">
                    <span className={`dot ${q.dot}`} />
                    {q.title}
                  </span>
                  <span className="s">{q.sub}</span>
                </div>
                {q.list.length === 0 ? (
                  <div className="fc-panel-empty">{q.empty}</div>
                ) : (
                  <table className="fc-table">
                    <thead>
                      <tr>
                        <th>Lead</th>
                        <th>Stage</th>
                        <th>Quiet</th>
                        <th>Likelihood</th>
                        <th>Value</th>
                        <th>Contact</th>
                      </tr>
                    </thead>
                    <tbody>
                      {q.list.map((j) => (
                        <tr key={j.id} {...leadRow(j)}>
                          <td>
                            <span className="name">{j.person ?? j.name}</span>
                            {j.company && <span className="sub">{j.company}</span>}
                          </td>
                          <td className="muted">{stageLabel(j.status)}</td>
                          <td>
                            <QuietCell j={j} now={now} />
                          </td>
                          <td className="mono muted">{j.likelihood}%</td>
                          <td className="mono muted">{(j.amountUsd ?? 0) > 0 ? usd(j.amountUsd ?? 0) : '—'}</td>
                          <td>
                            <Contact j={j} />
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </div>
            ))}
          </div>
        )}

        {/* ================= JOURNEYS ================= */}
        {view === 'journeys' && (
          <div className="fc-panel">
            <div className="fc-panel-head">
              <span className="t">Journey data</span>
              <span className="s">
                {tableJourneys.length} of {journeys.length} leads · click a row for the dossier
              </span>
            </div>
            <div className="fc-chips">
              <button type="button" className={`fc-chip${!segment ? ' on' : ''}`} onClick={() => setSegment(undefined)}>
                All stages
              </button>
              {FUNNEL_STAGES.map((s, i) => (
                <button
                  key={s.id}
                  type="button"
                  className={`fc-chip${segment === s.id ? ' on' : ''}`}
                  onClick={() => {
                    setSegment(segment === s.id ? undefined : s.id);
                    setStage(s.id);
                  }}
                >
                  <span className="sw" style={{ ['--c' as string]: STAGE_VAR(i) }} />
                  {s.label} ({stageCounts.get(s.id) ?? 0})
                </button>
              ))}
              <span className="fc-chip-sep" />
              <button type="button" className={`fc-chip${!lane ? ' on' : ''}`} onClick={() => setLane(undefined)}>
                Any entry
              </button>
              {LANES.map((l) => (
                <button
                  key={l}
                  type="button"
                  className={`fc-chip${lane === l ? ' on' : ''}`}
                  onClick={() => setLane(lane === l ? undefined : l)}
                >
                  <span className="sw" style={{ ['--c' as string]: LANE_VAR[l] }} />
                  {LANE_LABEL[l]} ({laneCounts[l]})
                </button>
              ))}
            </div>
            {tableJourneys.length === 0 ? (
              <div className="fc-panel-empty">No leads match those filters.</div>
            ) : (
              <table className="fc-table">
                <thead>
                  <tr>
                    <th>Client</th>
                    <th>Stage</th>
                    <th>Quiet</th>
                    <th>Relationship</th>
                    <th>Likelihood</th>
                    <th>Entry</th>
                    <th>Value</th>
                    <th>Contact</th>
                  </tr>
                </thead>
                <tbody>
                  {tableJourneys.map((j) => (
                    <JourneyRows key={j.id} j={j} now={now} rowProps={leadRow(j)} />
                  ))}
                </tbody>
              </table>
            )}
          </div>
        )}

        {/* ================= ARCHIVE ================= */}
        {view === 'archive' && (
          <div className="fc-panel">
            <div className="fc-panel-head">
              <span className="t">Archive</span>
              <span className="s">Quiet past {DECAY_DAYS} days · click a row for the dossier</span>
            </div>
            {archived.length === 0 ? (
              <div className="fc-panel-empty">Nothing decayed. No lead has sat quiet past {DECAY_DAYS} days.</div>
            ) : (
              <table className="fc-table">
                <thead>
                  <tr>
                    <th>Lead</th>
                    <th>Reached</th>
                    <th>Quiet</th>
                    <th>Likelihood</th>
                    <th>Last touch</th>
                    <th>Contact</th>
                  </tr>
                </thead>
                <tbody>
                  {archived.map((j) => {
                    const last = j.touches[j.touches.length - 1];
                    return (
                      <tr key={j.id} {...leadRow(j)}>
                        <td>
                          <span className="name" style={{ color: 'var(--text-2)' }}>
                            {j.person ?? j.name}
                          </span>
                          {j.company && <span className="sub">{j.company}</span>}
                        </td>
                        <td className="muted">{stageLabel(j.status)}</td>
                        <td className="mono dim">{journeyMeta(j, now).daysSinceLastTouch}d</td>
                        <td className="mono dim">{j.likelihood}%</td>
                        <td
                          className="dim"
                          style={{ maxWidth: 300, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
                          title={last?.label}
                        >
                          {last?.label ?? '—'}
                        </td>
                        <td>
                          <Contact j={j} />
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            )}
          </div>
        )}

        {/* ---- sources: the honest state of every feed ---- */}
        <div className="fc-sources">
          {source.kind === 'live' ? (
            <span className="src">
              <span className="dot ok" /> Live · {source.label}
            </span>
          ) : source.kind === 'db' ? (
            <span className="src">
              <span className="dot warn" /> Local snapshot · CRM not keyed
            </span>
          ) : (
            <span className="src">
              <span className="dot warn" /> Sample data · CRM not keyed and no journeys stored
            </span>
          )}
          {source.kind !== 'live' && (
            <SetupLink
              href={connectors.find((c) => c.id === 'attio')?.href ?? '/integrations'}
              hint="Key Attio or GHL to replace this with the live pipeline"
            />
          )}
          {sampleOffer && (
            <Link href="/funnel?sample=1" className="src" title="See the page on a badged sample set; nothing is stored">
              Preview with sample data
            </Link>
          )}
          {source.kind === 'sample' && (
            <Link href="/funnel" className="src" title="Back to the empty, honest state">
              Leave sample data
            </Link>
          )}
          <span style={{ width: 1, height: 12, background: 'var(--fc-hair)' }} />
          {connectors.map((c) =>
            c.ok ? (
              <span key={c.id} className="src" title={c.detail}>
                <span className="dot ok" /> {c.name}
                {c.count != null ? ` ${c.count}` : ''}
              </span>
            ) : (
              <Link key={c.id} href={c.href} className="src off" title={`${c.detail} · set up`}>
                <span className="dot off" /> {c.name}
              </Link>
            ),
          )}
        </div>
      </div>

      {selectedLead && (
        <LeadDrawer
          j={selectedLead}
          now={now}
          archived={archivedIds.has(selectedLead.id)}
          onClose={() => setSelected(undefined)}
          onStage={(s) => {
            setSelected(undefined);
            setStage(s);
            setView('overview');
          }}
        />
      )}
    </div>
  );
}

function JourneyRows({
  j,
  now,
  rowProps,
}: {
  j: FunnelJourney;
  now: Date;
  rowProps: React.HTMLAttributes<HTMLTableRowElement>;
}) {
  const converted = j.status === 'converted';
  const lane = laneOf(j);
  return (
    <>
      <tr {...rowProps}>
        <td>
          <span className="name">{j.person ?? j.name}</span>
          {j.company && <span className="sub">{j.company}</span>}
        </td>
        <td className="muted">
          <span style={{ display: 'inline-flex', alignItems: 'center', gap: 8 }}>
            {converted ? <span className="dot ok" /> : journeyMeta(j, now).state === 'stalled' ? <span className="dot err" /> : null}
            {stageLabel(j.status)}
          </span>
        </td>
        <td>
          <QuietCell j={j} now={now} />
        </td>
        <td style={{ textTransform: 'capitalize', color: `var(--funnel-${j.relationship})` }}>{j.relationship}</td>
        <td className="mono muted">{j.likelihood}%</td>
        <td className="dim">{LANE_LABEL[lane]}</td>
        <td className="mono">
          {converted ? (
            <span title={j.product ?? undefined}>
              {usd(j.amountUsd ?? 0)}
              {j.product ? ` · ${j.product}` : ''}
            </span>
          ) : (j.amountUsd ?? 0) > 0 ? (
            <span className="dim">{usd(j.amountUsd ?? 0)} open</span>
          ) : (
            <span className="dim">—</span>
          )}
        </td>
        <td>
          <Contact j={j} />
        </td>
      </tr>
      <tr className="touch-row">
        <td colSpan={8} style={{ padding: 0, borderTop: 0 }}>
          <Touches j={j} now={now} />
        </td>
      </tr>
    </>
  );
}
