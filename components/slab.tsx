import type { ReactNode } from 'react';
import { Lightbulb } from 'lucide-react';
import { Rise } from '@/components/motion';
import { CountUp, type CountKind } from '@/components/CountUp';
import { VolumeMeter } from '@/components/VolumeMeter';

/**
 * The Brand Deals slab as a kit (2026-09-24, Alex: "rebuild the entire OS
 * in that light"). DealBoard is the reference; every page composes these so
 * the opening beat, the sweeping meters and the numerals read the same OS-wide:
 *
 *   <Slab>                       the floating 28px surface (.os-slab)
 *     <SlabTitle … />            eyebrow · 46px title · mono meta · actions
 *     <SlabCard title i>          inset card, rises on the stagger, lifts on hover
 *       <BigStat …/>              50px count-up + dot chips + caption
 *       <MeterStack meters/>      the Deal Volume bars
 *     <InsightCard …/>            the ONE gradient card a page may carry
 *
 * Server-safe: CountUp and VolumeMeter are the only moving parts and both
 * are self-contained. Pages pass their own numbers; nothing here invents one.
 */

export type Meter = { label: string; frac: number; display: string; hue: string };
export type Tone = 'ok' | 'warn' | 'err' | 'accent';

const DOT: Record<Tone, string> = { ok: 'bg-os-ok', warn: 'bg-os-warn', err: 'bg-os-err', accent: 'bg-os-accent' };

/** A header action in the slab's pill shape (links and buttons alike). */
export const PILL =
  'pressable inline-flex items-center gap-1.5 rounded-full border border-os-border px-4 py-2 text-[13px] text-os-muted hover:border-os-border-strong hover:text-os-text';
/** The accent pill Brand Deals uses for "Open in Notion". */
export const PILL_ACCENT =
  'inline-flex items-center gap-2 rounded-full border border-[var(--accent-line)] bg-[var(--accent-soft)] px-4 py-2 text-[13px] text-os-accent transition-transform hover:-translate-y-[1px]';
/** Filter chips under a list head: the active one is solid accent. */
export const chipClass = (active: boolean) =>
  `pressable rounded-full px-4 py-1.5 text-[12.5px] ${active ? 'bg-os-accent font-semibold text-os-ink' : 'border border-os-border text-os-muted hover:text-os-text'}`;

export function Slab({ children, className = '' }: { children: ReactNode; className?: string }) {
  return <div className={`os-slab ${className}`.trim()}>{children}</div>;
}

export function SlabTitle({ eyebrow, title, meta, right }: { eyebrow?: string; title: string; meta?: ReactNode; right?: ReactNode }) {
  return (
    <Rise i={0} className="mb-7 flex flex-wrap items-end justify-between gap-5">
      <div className="min-w-0">
        {eyebrow && <div className="mb-2 font-mono text-[11px] uppercase tracking-[0.32em] text-os-dim">{`// ${eyebrow}`}</div>}
        <h1 className="text-[46px] font-semibold leading-none tracking-[-0.035em] max-[600px]:text-[34px]">{title}</h1>
        {meta && <div className="mt-3 font-mono text-[11px] text-os-dim">{meta}</div>}
      </div>
      {right && <div className="flex flex-wrap items-center gap-2.5">{right}</div>}
    </Rise>
  );
}

export function SlabCard({
  title,
  sub,
  action,
  i = 1,
  className = '',
  children,
}: {
  title?: string;
  sub?: ReactNode;
  action?: ReactNode;
  i?: number;
  className?: string;
  children?: ReactNode;
}) {
  return (
    <Rise i={i} className={`rise-card relative min-w-0 rounded-[12px] border border-os-border bg-os-surface ${className}`.trim()}>
      {title && (
        <div className="flex flex-wrap items-center justify-between gap-3 px-6 pt-5">
          <div className="flex min-w-0 items-baseline gap-2">
            <h2 className="text-[19px] font-semibold tracking-[-0.01em]">{title}</h2>
            {sub && <span className="font-mono text-[12px] tabular-nums text-os-dim">{sub}</span>}
          </div>
          {action && <div className="flex flex-wrap items-center gap-2">{action}</div>}
        </div>
      )}
      {children}
    </Rise>
  );
}

export function Chip({ tone, children }: { tone?: Tone; children: ReactNode }) {
  return (
    <span className="inline-flex items-center gap-1.5 rounded-full border border-os-border px-2.5 py-1 font-mono text-[11.5px] tabular-nums text-os-muted">
      {tone && <span className={`h-1.5 w-1.5 rounded-full ${DOT[tone]}`} />}
      {children}
    </span>
  );
}

export function BigStat({
  value,
  kind = 'int',
  display,
  unit,
  chips = [],
  caption,
  size = 50,
}: {
  value?: number;
  kind?: CountKind;
  /** Preformatted headline when CountUp's formats do not fit (4.2M, 68%). */
  display?: string;
  unit?: string;
  chips?: Array<{ tone?: Tone; text: string }>;
  caption?: ReactNode;
  size?: 50 | 30;
}) {
  return (
    <div>
      <div className="flex flex-wrap items-center gap-2.5">
        <span className={`${size === 50 ? 'text-[50px] tracking-[-0.035em]' : 'text-[30px] tracking-[-0.03em]'} font-semibold leading-none tabular-nums`}>
          {display ?? <CountUp value={value ?? 0} kind={kind} />}
          {unit && <span className="ml-1.5 text-[15px] font-normal tracking-normal text-os-dim">{unit}</span>}
        </span>
        {chips.map((c) => (
          <Chip key={c.text} tone={c.tone}>
            {c.text}
          </Chip>
        ))}
      </div>
      {caption && <div className="mt-2 text-[13px] text-os-dim">{caption}</div>}
    </div>
  );
}

export function MeterStack({ meters, foot, empty = 'nothing to measure yet', baseDelay = 500 }: { meters: Meter[]; foot?: ReactNode; empty?: string; baseDelay?: number }) {
  return (
    <div className="mt-5 flex flex-1 flex-col">
      <div className="flex flex-1 flex-col justify-around gap-6 border-t border-os-border pt-5">
        {meters.length === 0 ? (
          <div className="text-[12.5px] text-os-dim">{empty}</div>
        ) : (
          meters.map((m, i) => <VolumeMeter key={m.label} {...m} delay={baseDelay + i * 150} />)
        )}
      </div>
      {foot && <div className="mt-5 border-t border-os-border pt-3 text-center font-mono text-[10.5px] tracking-[0.1em] text-os-dim">{foot}</div>}
    </div>
  );
}

/** How many of the insight card's 8 ticks a fraction lights. */
export function insightTicks(frac: number): number {
  if (!Number.isFinite(frac) || frac <= 0) return 0;
  return Math.min(8, Math.round(frac * 8));
}

const GRAIN =
  "url(\"data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='160' height='160'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='2'/%3E%3C/filter%3E%3Crect width='160' height='160' filter='url(%23n)' opacity='0.55'/%3E%3C/svg%3E\")";

/** The ONE gradient card: drifting glow, grain, a glass shard, a big white number. */
export function InsightCard({
  badge,
  icon,
  value,
  kind = 'int',
  display,
  headline,
  body,
  frac = 0,
  i = 5,
  className = '',
}: {
  badge: string;
  icon?: ReactNode;
  value?: number;
  kind?: CountKind;
  display?: string;
  headline: ReactNode;
  body?: ReactNode;
  frac?: number;
  i?: number;
  className?: string;
}) {
  const lit = insightTicks(frac);
  return (
    <Rise
      i={i}
      className={`rise-card relative min-w-0 overflow-hidden rounded-[12px] border border-transparent ${className}`.trim()}
      style={{
        background: [
          'radial-gradient(120% 90% at 85% 8%, color-mix(in oklab, var(--tile-glow-a) 55%, transparent), transparent 60%)',
          'radial-gradient(130% 110% at 12% 92%, color-mix(in oklab, var(--tile-glow-b) 55%, transparent), transparent 62%)',
          'radial-gradient(110% 110% at 55% 55%, color-mix(in oklab, var(--tile-glow-c) 45%, transparent), transparent 70%)',
          'var(--insight-base)',
        ].join(', '),
        backgroundSize: '160% 160%',
        // the rise beat and the slow glow drift share one animation list
        animation: 'os-rise .6s var(--ease) calc(var(--rise-i) * 90ms) backwards, os-drift 14s ease-in-out 1s infinite alternate',
      }}
    >
      <div className="pointer-events-none absolute inset-0" style={{ backgroundImage: GRAIN, mixBlendMode: 'overlay', opacity: 0.85 }} />
      <div className="pointer-events-none absolute -right-14 -top-16 h-56 w-56 rotate-[24deg] rounded-[36px] border border-white/25 bg-white/5" style={{ backdropFilter: 'blur(3px)' }} />
      <div className="relative flex h-full flex-col px-6 py-5 text-white">
        <span className="inline-flex w-fit items-center gap-1.5 rounded-full bg-white/15 px-3 py-1 text-[12px] backdrop-blur">
          {icon ?? <Lightbulb size={13} strokeWidth={1.7} />} {badge}
        </span>
        <div className="mt-4 text-[64px] font-semibold leading-none tracking-[-0.03em] tabular-nums">{display ?? <CountUp value={value ?? 0} kind={kind} />}</div>
        <div className="mt-2 text-[16px] font-semibold leading-snug">{headline}</div>
        {body && <div className="mt-1.5 text-[12.5px] leading-relaxed text-white/75">{body}</div>}
        <div className="mt-auto flex gap-1.5 pt-4">
          {Array.from({ length: 8 }, (_, t) => (
            <span key={t} data-tick className="h-[3px] flex-1 rounded-full" style={{ background: t < lit ? '#fff' : 'rgba(255,255,255,.25)' }} />
          ))}
        </div>
      </div>
    </Rise>
  );
}
