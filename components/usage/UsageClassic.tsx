'use client';

import { useState, type ReactNode } from 'react';
import { PageHeader } from '@/components/PageHeader';
import { SetupLink } from '@/components/SetupLink';
import { Badge, Dot } from '@/components/terminal';
import type { UsagePlan } from '@/lib/schemas';
import type { UsagePayload } from '@/lib/usage-payload';
import { USAGE_SUBTITLE, USAGE_TITLE, type PlanCardView, type UsageReadyView } from '@/lib/usage-view';
import { setupHrefFor } from '@/lib/integrations-catalog';
import { pushUrl, useUsage } from './use-usage';
import { Layer } from '@/components/g/Overlays';

/**
 * /usage under the Classic style (pages/usage.md §7): PageHeader with range
 * chips, Token burn (8) + Burn volume meters (4), Burn by model, Plan limits
 * and Top burner (4 each), then the three plan cards. os-* tokens only; plan
 * colours are the Classic categorical tokens --cat-1..3.
 */

const CARD = 'rounded-lg-t border border-os-border bg-os-surface';

function Card({ title, caption, children, className = '', id }: { title: string; caption?: string; children: ReactNode; className?: string; id?: string }) {
  return (
    <section className={`${CARD} min-w-0 p-5 ${className}`} aria-labelledby={id}>
      <header className="mb-4 flex flex-wrap items-baseline gap-x-2 gap-y-1">
        <h2 id={id} className="text-[15px] font-medium text-os-text">
          {title}
        </h2>
        {caption && <span className="text-[12.5px] text-os-muted">{caption}</span>}
      </header>
      {children}
    </section>
  );
}

function Meter({ label, value, display, max, slot }: { label: string; value: number; display: string; max: number; slot?: 1 | 2 | 3 }) {
  const pct = max > 0 ? Math.max(value > 0 ? 1.5 : 0, (value / max) * 100) : 0;
  return (
    <div className="us-c-meter" title={`${label}: ${display}`}>
      <div className="flex items-baseline justify-between gap-3 text-[12.5px]">
        <span className="min-w-0 truncate text-os-muted">{label}</span>
        <span className="font-mono tabular-nums text-os-text">{display}</span>
      </div>
      <div className="us-c-track" role="img" aria-label={`${label}: ${display}`}>
        <span className="us-c-fill" data-slot={slot} style={{ width: `${pct}%` }} />
      </div>
    </div>
  );
}

function CopyPushUrl({ path }: { path: string }) {
  const [state, setState] = useState<'idle' | 'copied' | 'failed'>('idle');
  return (
    <button
      type="button"
      className="c-btn"
      title={`POST readings here with Authorization: Bearer <USAGE_PUSH_TOKEN> (${path})`}
      onClick={async () => {
        try {
          if (!navigator.clipboard) throw new Error('no clipboard');
          await navigator.clipboard.writeText(pushUrl(path));
          setState('copied');
        } catch {
          setState('failed');
        }
        window.setTimeout(() => setState('idle'), 1800);
      }}
    >
      {state === 'copied' ? 'Copied' : state === 'failed' ? 'Copy failed' : 'Copy push URL'}
    </button>
  );
}

function Panel({ title, caption, onClose, children, footer }: { title: string; caption?: string; onClose: () => void; children: ReactNode; footer?: ReactNode }) {
  // Portalled to <body>: a frosted ancestor would trap the fixed layer.
  return (
    <Layer>
      <div className="us-c-layer" role="presentation" onKeyDown={(e) => e.key === 'Escape' && onClose()}>
        <div className="us-c-scrim" aria-hidden onClick={onClose} />
        <div role="dialog" aria-modal="true" aria-label={title} className={`${CARD} us-c-panel`}>
          <header className="mb-4 flex items-start justify-between gap-4">
            <div className="min-w-0">
              <h2 className="text-[18px] font-medium text-os-text">{title}</h2>
              {caption && <p className="mt-1 text-[12.5px] text-os-muted">{caption}</p>}
            </div>
            <button type="button" className="c-btn" onClick={onClose} autoFocus>
              Close
            </button>
          </header>
          <div className="us-c-panel-body">{children}</div>
          {footer && <footer className="mt-4 flex justify-end gap-2">{footer}</footer>}
        </div>
      </div>
    </Layer>
  );
}

function Bars({ v, hidden }: { v: UsageReadyView; hidden: UsagePlan[] }) {
  const cols = v.burnChart.columns.map((c) => ({
    ...c,
    shown: c.parts.map((p, j) => (hidden.includes(v.burnChart.keys[j].id) ? 0 : p)),
  }));
  const max = Math.max(0, ...cols.map((c) => c.shown.reduce((a, b) => a + b, 0)));
  return (
    <div className="us-c-bars" data-range={v.range} role="group" aria-label={`Token burn per day by plan, ${v.rangeLabel}`}>
      {cols.map((c) => {
        const total = c.shown.reduce((a, b) => a + b, 0);
        const label = `${c.long}: ${v.burnChart.keys.map((k, j) => `${k.label} ${c.parts[j].toLocaleString('en-US')}`).join(', ')}`;
        return (
          <div key={c.id} className="us-c-col" title={label} role="img" aria-label={label}>
            <div
              className="us-c-stack"
              style={{
                height: `${max > 0 ? Math.max(total > 0 ? 2 : 0, (total / max) * 100) : 0}%`,
              }}
            >
              {v.burnChart.keys.map((k, j) => (c.shown[j] > 0 ? <span key={k.id} data-slot={k.slot} style={{ flexGrow: c.shown[j] }} /> : null))}
            </div>
            <span className="us-c-xl">{c.label}</span>
          </div>
        );
      })}
    </div>
  );
}

function PlanCard({ p, onOpen }: { p: PlanCardView; onOpen: () => void }) {
  return (
    <button type="button" onClick={onOpen} className={`${CARD} us-c-plan p-5 text-left`} title={`${p.label}: open the plan detail`}>
      <span className="flex items-center justify-between gap-3">
        <span className="flex items-center gap-2 text-[14px] font-medium text-os-text">
          <i className="us-c-key" data-slot={p.slot} aria-hidden />
          {p.label}
        </span>
        {p.status && (
          <Badge tone={p.status.tone === 'neutral' ? 'default' : p.status.tone}>
            <Dot state={p.status.tone === 'neutral' ? 'off' : p.status.tone} /> {p.status.text}
          </Badge>
        )}
      </span>
      <span className={`mt-3 block text-[28px] font-medium leading-tight tabular-nums ${p.reported ? 'text-os-text' : 'text-os-muted'}`}>{p.total}</span>
      <span className="mt-3 block">{p.spark ? <Spark values={p.spark} slot={p.slot} label={`${p.label} burn per day`} /> : <span className="us-c-spark-none" aria-hidden />}</span>
      <span className="mt-3 block text-[12.5px] text-os-muted">{p.meta}</span>
    </button>
  );
}

function Spark({ values, slot, label }: { values: number[]; slot: 1 | 2 | 3; label: string }) {
  const max = Math.max(1, ...values);
  const w = 100;
  const h = 28;
  const pts = values.map((v, i) => `${((i / Math.max(1, values.length - 1)) * w).toFixed(2)},${(h - 2 - (v / max) * (h - 4)).toFixed(2)}`).join(' ');
  return (
    <svg className="us-c-spark" data-slot={slot} viewBox={`0 0 ${w} ${h}`} preserveAspectRatio="none" role="img" aria-label={label}>
      <polyline points={pts} fill="none" strokeWidth="1.5" vectorEffect="non-scaling-stroke" />
    </svg>
  );
}

function PlanPanel({ p, v, onClose }: { p: PlanCardView; v: UsageReadyView; onClose: () => void }) {
  const d = p.detail;
  const dayMax = Math.max(0, ...(d.days ?? []).map((x) => x.burn));
  const laneMax = Math.max(0, ...(d.lanes ?? []).map((x) => x.burn));
  return (
    <Panel
      title={p.label}
      caption={`${p.total === 'Not reported' ? 'Not reported' : `${p.total} tokens`} · ${v.rangeLabel}`}
      onClose={onClose}
      footer={<CopyPushUrl path={v.push.path} />}
    >
      <div className="grid gap-6 md:grid-cols-2">
        <div className="min-w-0">
          <h3 className="mb-3 text-[13px] font-medium text-os-muted">Per day</h3>
          {d.days ? (
            <div className="flex flex-col gap-2">
              {d.days.slice(-10).map((x) => (
                <Meter
                  key={x.id}
                  label={x.long}
                  value={x.burn}
                  display={new Intl.NumberFormat('en-US', {
                    notation: 'compact',
                    maximumFractionDigits: 1,
                  }).format(x.burn)}
                  max={dayMax}
                  slot={p.slot}
                />
              ))}
            </div>
          ) : (
            <p className="text-[13px] text-os-muted">Not reported. No machine reported this plan in the window.</p>
          )}
        </div>
        <div className="min-w-0">
          <h3 className="mb-3 text-[13px] font-medium text-os-muted">Lanes</h3>
          {d.lanes && (
            <div className="mb-2 flex flex-col gap-2">
              {d.lanes.map((l) => (
                <Meter key={l.id} label={l.label} value={l.burn} display={l.display} max={laneMax} />
              ))}
            </div>
          )}
          <p className="text-[12.5px] text-os-muted">{d.lanesNote}</p>
          <h3 className="mb-3 mt-6 text-[13px] font-medium text-os-muted">Machines</h3>
          {d.machines.length ? (
            <ul className="divide-y divide-[var(--hairline)]">
              {d.machines.map((m) => (
                <li key={m.machine} className="flex items-center justify-between gap-3 py-2 text-[13px]" title={m.title}>
                  <span className="min-w-0">
                    <span className="block truncate font-mono text-os-text">{m.machine}</span>
                    <span className="text-[12px] text-os-muted">
                      {m.sourceLabel} · {m.age}
                    </span>
                  </span>
                  <span className="flex items-center gap-2">
                    {m.stale && <Badge tone="warn">Stale</Badge>}
                    <span className="font-mono tabular-nums">{m.burn}</span>
                  </span>
                </li>
              ))}
            </ul>
          ) : (
            <p className="text-[12.5px] text-os-muted">No machine has reported this plan. Another machine can push to {v.push.path}.</p>
          )}
          <h3 className="mb-3 mt-6 text-[13px] font-medium text-os-muted">Windows</h3>
          {d.windows.length ? (
            d.windows.map((w) => <Meter key={w.id} label={`${w.label}${w.resets ? ` · ${w.resets}` : ''}`} value={w.used} display={`${Math.round(w.used)}%`} max={100} />)
          ) : (
            <p className="text-[12.5px] text-os-muted">No provider reported an official window for this plan.</p>
          )}
        </div>
      </div>
    </Panel>
  );
}

function Ready({ v }: { v: UsageReadyView }) {
  const [hidden, setHidden] = useState<UsagePlan[]>([]);
  const [open, setOpen] = useState<UsagePlan | null>(null);
  const plan = open ? (v.plans.find((p) => p.plan === open) ?? null) : null;
  const laneMax = Math.max(0, ...(v.lanes.rows ?? []).map((l) => l.burn));
  const modelMax = Math.max(0, ...v.models.rows.map((m) => m.burn));
  return (
    <div className="flex flex-col gap-4">
      <div className="us-c-row us-c-row-a">
        <Card title="Token burn" caption={v.burnChart.caption} id="usc-burn">
          {v.empty ? (
            <p className="py-10 text-center text-[13px] text-os-muted">No burn in this window.</p>
          ) : (
            <>
              <Bars v={v} hidden={hidden} />
              <div className="mt-3 flex flex-wrap gap-2" role="group" aria-label="Plans in the chart">
                {v.burnChart.keys.map((k) => {
                  const on = !hidden.includes(k.id);
                  return (
                    <button
                      key={k.id}
                      type="button"
                      aria-pressed={on}
                      className="c-chip us-c-legend"
                      data-on={on || undefined}
                      onClick={() => setHidden((h) => (h.includes(k.id) ? h.filter((x) => x !== k.id) : [...h, k.id]))}
                    >
                      <i className="us-c-key" data-slot={k.slot} aria-hidden /> {k.label}
                    </button>
                  );
                })}
              </div>
            </>
          )}
        </Card>
        <Card title="Burn volume" caption={v.rangeLabel} id="usc-volume">
          <div className="flex flex-wrap items-baseline gap-x-3 gap-y-2">
            <span className="text-[34px] font-medium leading-none tabular-nums text-os-text" title={v.hero.label}>
              {v.hero.value}
            </span>
            <span className="text-[13px] text-os-muted">tokens</span>
            <Badge tone={v.ollama.up ? 'ok' : 'err'}>
              <Dot state={v.ollama.up ? 'ok' : 'err'} /> {v.ollama.up ? 'Ollama up' : 'Ollama down'}
            </Badge>
          </div>
          <p className="mt-2 text-[12.5px] text-os-muted">{v.hero.status}</p>
          <div className="mt-4 flex flex-col gap-3 border-t border-[var(--hairline)] pt-4">
            {v.lanes.rows ? v.lanes.rows.map((l) => <Meter key={l.id} label={l.label} value={l.burn} display={l.display} max={laneMax} />) : null}
            <p className="text-[12px] text-os-muted">{v.lanes.note}</p>
          </div>
        </Card>
      </div>

      <div className="us-c-row us-c-row-b">
        <Card title="Burn by model" caption={v.models.caption} id="usc-models">
          {v.models.rows.length ? (
            <div className="flex flex-col gap-3">
              {v.models.rows.slice(0, 6).map((m) => (
                <Meter key={m.id} label={m.label} value={m.burn} display={m.display} max={modelMax} slot={m.plan === 'claude' ? 1 : m.plan === 'codex' ? 2 : 3} />
              ))}
            </div>
          ) : (
            <p className="text-[13px] text-os-muted">No model burn in this window.</p>
          )}
        </Card>
        <Card title="Plan limits" caption="official gauges only" id="usc-limits">
          {v.limits.gauges.length ? (
            <div className="flex flex-col gap-3">
              {v.limits.gauges.map((g) => (
                <Meter key={g.id} label={`${g.label}${g.resets ? ` · ${g.resets}` : ''}`} value={g.used} display={`${Math.round(g.used)}%`} max={100} />
              ))}
            </div>
          ) : (
            <p className="text-[13px] leading-relaxed text-os-muted">{v.limits.note} Burn is measured from transcripts, not against a guessed limit.</p>
          )}
        </Card>
        <Card title="Top burner" caption={v.rangeLabel} id="usc-top">
          {v.topBurner ? (
            <div>
              <span className="block text-[34px] font-medium leading-none tabular-nums text-os-text">{v.topBurner.share}</span>
              <p className="mt-2 text-[12.5px] text-os-muted">{v.topBurner.caption}</p>
              <p className="mt-4 text-[13px] text-os-text">
                <span className="font-mono">{v.topBurner.machine}</span> on {v.topBurner.planLabel}
              </p>
              <p className="mt-1 text-[12.5px] text-os-muted">{v.topBurner.value} tokens</p>
            </div>
          ) : (
            <p className="text-[13px] text-os-muted">No burn in this window.</p>
          )}
        </Card>
      </div>

      <div className="us-c-plans">
        {v.plans.map((p) => (
          <PlanCard key={p.plan} p={p} onOpen={() => setOpen(p.plan)} />
        ))}
      </div>
      <p className="text-[12px] text-os-muted">
        {v.footnote} Read {v.readAgo}.
      </p>
      {plan && <PlanPanel p={plan} v={v} onClose={() => setOpen(null)} />}
    </div>
  );
}

export function UsageClassic({ initial, initialNow, sample }: { initial: UsagePayload; initialNow: number; sample: boolean }) {
  const { data, view, range, setRange, error, lastGoodAgo, retry } = useUsage(initial, initialNow, sample);
  const [raw, setRaw] = useState(false);
  const showSample = view.kind === 'ready' ? view.sample : view.kind === 'off' && view.filming;
  const right = (
    <>
      {showSample && (
        <Badge tone="warn">
          <span data-sample-badge="" title={view.kind === 'off' ? 'Filming mode: Usage is off' : 'Sample data: lib/fo-sample.ts sampleUsageReadings'}>
            Sample
          </span>
        </Badge>
      )}
      {view.kind === 'ready' && (
        <div className="flex gap-1.5" role="radiogroup" aria-label="Range">
          {([7, 30] as const).map((r) => (
            <button key={r} type="button" role="radio" aria-checked={range === r} className="c-chip us-c-range" data-on={range === r || undefined} onClick={() => setRange(r)}>
              {r} d
            </button>
          ))}
        </div>
      )}
      {view.kind !== 'off' && (
        <button type="button" className="c-btn" onClick={() => setRaw(true)}>
          Raw reading
        </button>
      )}
    </>
  );
  return (
    <div className="us-c">
      <PageHeader title={USAGE_TITLE} eyebrow={USAGE_SUBTITLE} right={right} />
      {error && view.kind !== 'off' && (
        <div className="mb-4 flex flex-wrap items-center gap-3 text-[13px] text-os-err" role="alert">
          Refresh failed ({error}). {lastGoodAgo ? `Showing the reading from ${lastGoodAgo}.` : ''}
          <button type="button" className="c-btn" onClick={retry}>
            Retry
          </button>
        </div>
      )}
      {view.kind === 'off' ? (
        <div className={`${CARD} p-6`}>
          <p className="text-[14px] text-os-text">{view.title}</p>
          <p className="mt-1 text-[13px] text-os-muted">{view.body}</p>
        </div>
      ) : view.kind === 'unconfigured' ? (
        <div className="us-c-row us-c-row-a items-start">
          <div className={`${CARD} p-5`}>
            <div className="flex flex-wrap items-center gap-3">
              <Badge>Not configured</Badge>
              <span className="text-[13.5px] text-os-text">{view.title}</span>
              <SetupLink href={setupHrefFor('usage')} hint={view.setupHint} />
            </div>
            <p className="mt-3 text-[13px] leading-relaxed text-os-muted">{view.body}</p>
          </div>
          <Card title="Sources checked" caption="read locally" id="usc-sources">
            <ul className="flex flex-col gap-2 text-[13px]">
              {view.checks.map((c) => (
                <li key={c.label} className="flex items-center justify-between gap-3">
                  <span className="text-os-muted">{c.label}</span>
                  <span className="flex items-center gap-2 text-right text-os-text">
                    <Dot state={c.ok ? 'ok' : 'off'} /> {c.value}
                  </span>
                </li>
              ))}
              <li className="flex items-center justify-between gap-3">
                <span className="text-os-muted">Ollama</span>
                <span className="flex items-center gap-2 text-os-text">
                  <Dot state={view.ollama.up ? 'ok' : 'err'} /> {view.ollama.text}
                </span>
              </li>
              <li className="flex items-center justify-between gap-3">
                <span className="text-os-muted">Push URL</span>
                <span className="font-mono text-os-text">{view.push.path}</span>
              </li>
            </ul>
            <div className="mt-4">
              <CopyPushUrl path={view.push.path} />
            </div>
          </Card>
        </div>
      ) : (
        <Ready v={view} />
      )}
      {raw && (
        <Panel title="Raw reading" caption="The /api/usage payload this page renders" onClose={() => setRaw(false)}>
          <pre className="us-c-raw font-mono">{JSON.stringify(data, null, 2)}</pre>
        </Panel>
      )}
    </div>
  );
}
