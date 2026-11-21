'use client';

/**
 * Small Classic-style parts shared by the FounderOS-merge pages (/clients,
 * /lead-magnets) under the Classic tree: card, modal, line and bar charts,
 * meter, status pill. `os-*` tokens only, so every one of the five Classic
 * themes paints them (scripts/theme-audit.mjs); no raw colour anywhere.
 */
import { useId, useMemo, useState, type ReactNode } from 'react';
import { X } from 'lucide-react';
import { useDialog } from '@/components/ui/use-dialog';
import { Layer } from '@/components/g/Overlays';

export function CCard({ title, right, children, className = '', id }: { title?: ReactNode; right?: ReactNode; children: ReactNode; className?: string; id?: string }) {
  return (
    <section className={`fo-c-card rounded-lg-t border border-os-border bg-os-surface p-5 ${className}`} aria-labelledby={title && id ? id : undefined}>
      {(title || right) && (
        <div className="mb-4 flex flex-wrap items-baseline justify-between gap-3">
          {title && <h2 id={id} className="text-[14px] font-medium text-os-text">{title}</h2>}
          {right && <div className="flex items-center gap-2 text-[12px] text-os-muted">{right}</div>}
        </div>
      )}
      {children}
    </section>
  );
}

export type PillTone = 'default' | 'accent' | 'ok' | 'warn' | 'err';

export function CPill({ tone = 'default', children, title }: { tone?: PillTone; children: ReactNode; title?: string }) {
  return (
    <span className="fo-c-pill" data-tone={tone} title={title}>
      <span className="fo-c-pill-dot" aria-hidden />
      {children}
    </span>
  );
}

export function CSampleBadge({ title }: { title: string }) {
  return (
    <span data-sample-badge="" className="fo-c-pill" data-tone="warn" title={title}>
      Sample
    </span>
  );
}

export function CModal({ open, onClose, title, children, footer, width = 520 }: { open: boolean; onClose: () => void; title: ReactNode; children: ReactNode; footer?: ReactNode; width?: number }) {
  const titleId = useId();
  const dlg = useDialog({ open, onClose, labelledBy: titleId });
  if (!open) return null;
  // Portalled to <body>: a frosted ancestor would trap the fixed scrim.
  return (
    <Layer>
      <div className="fo-c-scrim" onClick={onClose}>
        <div {...dlg.props} className="fo-c-modal c-pop" style={{ maxWidth: width }} onClick={(e) => e.stopPropagation()}>
          <div className="flex items-start justify-between gap-4">
            <h2 id={titleId} className="text-[16px] font-medium text-os-text">{title}</h2>
            <button type="button" data-dialog-close="" onClick={onClose} className="fo-c-icon-btn" aria-label="Close">
              <X size={15} strokeWidth={1.75} aria-hidden />
            </button>
          </div>
          <div className="mt-4">{children}</div>
          {footer && <div className="mt-5 flex flex-wrap justify-end gap-2">{footer}</div>}
        </div>
      </div>
    </Layer>
  );
}

/** Two-line daily chart (Classic). Series colour by token name. */
export function CLineChart({
  labels, series, height = 150, label, empty,
}: {
  labels: string[];
  axisLabels?: string[];
  series: { id: string; label: string; values: number[]; tone: 'accent' | 'muted' | 'ok' }[];
  height?: number;
  label: string;
  empty?: ReactNode;
}) {
  const [hover, setHover] = useState<number | null>(null);
  const max = Math.max(1, ...series.flatMap((s) => s.values));
  const W = 600;
  const n = labels.length;
  const x = (i: number) => (n <= 1 ? 0 : (i / (n - 1)) * W);
  const y = (v: number) => height - 8 - (v / max) * (height - 24);
  const total = series.reduce((a, s) => a + s.values.reduce((b, v) => b + v, 0), 0);
  if (total === 0 && empty) return <div className="flex items-center text-[13px] text-os-muted" style={{ minHeight: height }}>{empty}</div>;
  return (
    <figure className="m-0">
      <svg viewBox={`0 0 ${W} ${height}`} preserveAspectRatio="none" className="block w-full" style={{ height }} role="img" aria-label={label}
        onMouseLeave={() => setHover(null)}
        onMouseMove={(e) => {
          const r = (e.currentTarget as SVGSVGElement).getBoundingClientRect();
          setHover(Math.max(0, Math.min(n - 1, Math.round(((e.clientX - r.left) / r.width) * (n - 1)))));
        }}
      >
        <line x1={0} x2={W} y1={height - 8} y2={height - 8} className="fo-c-axis" vectorEffect="non-scaling-stroke" />
        {series.map((s) => (
          <polyline key={s.id} data-tone={s.tone} className="fo-c-line" vectorEffect="non-scaling-stroke"
            points={s.values.flatMap((v, i) => (i === 0 ? [`${x(0)},${y(v)}`] : [`${x(i)},${y(s.values[i - 1])}`, `${x(i)},${y(v)}`])).join(' ')} />
        ))}
        {hover != null && <line x1={x(hover)} x2={x(hover)} y1={8} y2={height - 8} className="fo-c-axis" vectorEffect="non-scaling-stroke" />}
      </svg>
      <figcaption className="mt-2 flex flex-wrap items-center justify-between gap-3 text-[12px] text-os-muted">
        <span className="flex flex-wrap gap-3">
          {series.map((s) => (
            <span key={s.id} className="inline-flex items-center gap-1.5">
              <span className="fo-c-key" data-tone={s.tone} aria-hidden /> {s.label}
              {hover != null && <b className="font-mono font-normal text-os-text">{s.values[hover]}</b>}
            </span>
          ))}
        </span>
        <span className="font-mono">{hover != null ? labels[hover] : `${labels[0]} to ${labels[n - 1]}`}</span>
      </figcaption>
    </figure>
  );
}

/** Vertical bars with a label under each (weekday rhythm, weeks shipped). */
export function CBars({ bars, height = 96, label, highlight, empty }: { bars: { id: string; label: string; value: number; title?: string }[]; height?: number; label: string; highlight?: string | null; empty?: ReactNode }) {
  const max = Math.max(1, ...bars.map((b) => b.value));
  const total = useMemo(() => bars.reduce((a, b) => a + b.value, 0), [bars]);
  if (total === 0 && empty) return <div className="flex items-center text-[13px] text-os-muted" style={{ minHeight: height }}>{empty}</div>;
  return (
    <div role="img" aria-label={label}>
      <div className="flex items-end gap-1.5" style={{ height }}>
        {bars.map((b) => (
          <div key={b.id} className="flex h-full min-w-0 flex-1 flex-col justify-end" title={b.title ?? `${b.label}: ${b.value}`}>
            <div className="fo-c-bar mx-auto max-w-[44px]" data-on={highlight === b.id || undefined} style={{ height: `${Math.max(b.value ? 6 : 2, (b.value / max) * 100)}%` }} />
          </div>
        ))}
      </div>
      <div className="mt-2 flex gap-1.5">
        {bars.map((b) => (
          <span key={b.id} className="min-w-0 flex-1 truncate text-center font-mono text-[10.5px] text-os-dim">{b.label}</span>
        ))}
      </div>
    </div>
  );
}

/** Labelled meter row: "Live 2 of 3" with a filled track. */
export function CMeter({ label, value, of, tone = 'accent' }: { label: ReactNode; value: number; of: number; tone?: 'accent' | 'ok' | 'warn' | 'err' | 'muted' }) {
  const pct = of > 0 ? Math.round((value / of) * 100) : 0;
  return (
    <div>
      <div className="mb-1.5 flex items-baseline justify-between gap-3 text-[12.5px]">
        <span className="text-os-muted">{label}</span>
        <span className="font-mono text-os-text">{value}<span className="text-os-dim"> of {of}</span></span>
      </div>
      <div className="fo-c-track" role="meter" aria-valuemin={0} aria-valuemax={of} aria-valuenow={value} aria-label={typeof label === 'string' ? label : undefined}>
        <div className="fo-c-fill" data-tone={tone} style={{ width: `${pct}%` }} />
      </div>
    </div>
  );
}
