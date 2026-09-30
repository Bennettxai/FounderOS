'use client';

/**
 * The live token-burn board. Polls /api/usage every 10s -- the refresh path
 * is pure file parsing on the server, so polling costs nothing billable.
 *
 * One card per plan Alex pays for (one Claude plan, one ChatGPT plan, one
 * Ollama plan), each with the active plan name, the official gauges when the
 * provider exposes them, and the burn for the last hour, the 5-hour session,
 * today and the week. Clicking a card opens where those tokens went: the
 * agent board, Superset sessions, the terminal, crons/headless runs, and the
 * biggest single burners inside each.
 *
 * House rule: a bar labelled "% of limit" only ever renders an OFFICIAL
 * number or an estimate derived from one, and says "est." when derived.
 *
 * Brand Deals look (2026-09-24): the reading opens as the slab's hero row
 * (a 7-day burn step line + a Burn Volume card of sweeping meters), a second
 * row (models waffle, the official limit gauges, one insight card), then the
 * plan cards. The numbers come from lib/usage-volume.ts, derived per poll.
 */
import { useCallback, useEffect, useRef, useState } from 'react';
import { Badge, Dot, Label } from '@/components/terminal';
import { VolumeMeter } from '@/components/VolumeMeter';
import { SlabCard, BigStat, MeterStack, InsightCard, chipClass } from '@/components/slab';
import { StepLine, DotMatrix } from '@/components/slab-charts';
import { LANE_HUE, limitMeter, usageVolume } from '@/lib/usage-volume';
import {
  BURN_SOURCES,
  SOURCE_LABEL,
  WINDOWS,
  WINDOW_LABEL,
  burnOf,
  fmtTokens,
  laneBurn,
  totalBurn,
  windowTable,
  type OllamaBoard,
  type OfficialWindow,
  type PlanUsage,
  type UsageWindow,
} from '@/lib/usage';

const POLL_MS = 10_000;

type Board = {
  generatedAt: string;
  claude: PlanUsage | null;
  codex: PlanUsage | null;
  ollama: OllamaBoard;
  errors?: Record<string, string>;
};
type PlanKey = 'claude' | 'codex' | 'ollama';

function age(iso: string | null, now: number): string {
  if (!iso) return ' - ';
  const s = Math.max(0, Math.floor((now - new Date(iso).getTime()) / 1000));
  if (s < 90) return `${s}s ago`;
  if (s < 5400) return `${Math.round(s / 60)}m ago`;
  if (s < 90000) return `${Math.round(s / 3600)}h ago`;
  return `${Math.round(s / 86400)}d ago`;
}

const pct = (n: number) => (n >= 10 || n === 0 ? `${n.toFixed(0)}%` : `${n.toFixed(1)}%`);

/** One stacked bar across the four lanes, one hue per lane. */
function LaneBar({ plan, window, height = 'h-2' }: { plan: PlanUsage; window: UsageWindow; height?: string }) {
  const lanes = plan.breakdown?.windows[window];
  const total = lanes ? laneBurn(lanes) : 0;
  return (
    <div className={`flex ${height} w-full gap-[2px] overflow-hidden rounded-full`} style={{ background: 'color-mix(in oklab, var(--text) 8%, transparent)' }}>
      {total > 0 &&
        BURN_SOURCES.map((s) => {
          const w = (burnOf(lanes![s]) / total) * 100;
          return w > 0 ? <div key={s} className="h-full" style={{ width: `${w}%`, background: LANE_HUE[s] }} title={`${SOURCE_LABEL[s]} · ${fmtTokens(burnOf(lanes![s]))}`} /> : null;
        })}
    </div>
  );
}

function LaneLegend() {
  return (
    <div className="flex flex-wrap gap-x-3 gap-y-1">
      {BURN_SOURCES.map((s) => (
        <span key={s} className="flex items-center gap-1.5 text-[11.5px] text-os-muted">
          <span className="inline-block h-2 w-2 rounded-full" style={{ background: LANE_HUE[s] }} />
          {SOURCE_LABEL[s]}
        </span>
      ))}
    </div>
  );
}

function WindowStrip({ plan }: { plan: PlanUsage }) {
  const rows = plan.breakdown ? windowTable(plan.breakdown, plan.official?.weekly?.usedPercent ?? null) : null;
  return (
    <div className="grid grid-cols-4 gap-2">
      {WINDOWS.map((w) => {
        const r = rows?.find((x) => x.window === w);
        return (
          <div key={w} className="border-l border-os-border pl-2">
            <div className="text-[19px] font-semibold tabular-nums tracking-[-0.02em]">{fmtTokens(r?.burn ?? 0)}</div>
            <div className="font-mono text-[9.5px] uppercase tracking-[0.16em] text-os-dim">{WINDOW_LABEL[w]}</div>
            <div className="font-mono text-[9.5px] text-os-muted">
              {r?.pctOfWeekly !== null && r?.pctOfWeekly !== undefined ? `≈${pct(r.pctOfWeekly)} of wk${w === 'week' ? '' : ' est.'}` : r ? `${pct(r.shareOfWeek * 100)} of wk burn` : ' - '}
            </div>
          </div>
        );
      })}
    </div>
  );
}

function PlanCard({ title, plan, selected, onSelect, now }: { title: string; plan: PlanUsage | null; selected: boolean; onSelect: () => void; now: number }) {
  if (!plan) {
    return (
      <div className="rounded-[12px] border border-dashed border-os-border p-5 text-[12.5px] text-os-dim">
        <span className="text-[15px] font-semibold text-os-muted">{title}</span>
        <p className="mt-2">no sessions found on any reporting machine</p>
      </div>
    );
  }
  const hasData = totalBurn(plan.days) > 0 || plan.lastActivity !== null;
  return (
    <button type="button" onClick={onSelect} aria-expanded={selected} data-lens="r" className={`pressable is-row w-full rounded-[12px] border bg-os-bg p-5 text-left ${selected ? 'border-os-accent' : 'border-os-border'}`}>
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <Dot state={hasData ? 'ok' : 'off'} />
          <span className="text-[15px] font-semibold tracking-[-0.01em]">{title}</span>
        </div>
        {plan.planConflict ? <Badge tone="warn">plans differ</Badge> : <Badge tone={plan.plan ? 'accent' : 'default'}>{plan.plan ? `${plan.plan} · active` : 'plan unknown'}</Badge>}
      </div>
      {plan.planConflict && (
        <p className="mt-2 font-mono text-[10px] leading-relaxed text-os-warn">
          machines are logged into different plans: {plan.planConflict.join(' · ')}
        </p>
      )}

      {plan.official?.session || plan.official?.weekly ? (
        <div className="mt-4 space-y-4">
          {plan.official.session && <VolumeMeter {...limitMeter('Session limit', '5h', plan.official.session, now)} />}
          {plan.official.weekly && <VolumeMeter {...limitMeter('Weekly limit', '7d', plan.official.weekly, now)} delay={150} />}
        </div>
      ) : (
        <p className="mt-4 text-[12px] leading-relaxed text-os-dim">no official gauge reported · burn below is measured from transcripts</p>
      )}

      <div className="mt-4">
        <WindowStrip plan={plan} />
      </div>

      <div className="mt-4 space-y-1.5">
        <LaneBar plan={plan} window="week" />
        <div className="flex items-center justify-between font-mono text-[10px] text-os-dim">
          <span>
            {plan.machines.length} machine{plan.machines.length === 1 ? '' : 's'} · last activity {age(plan.lastActivity, now)}
          </span>
          <span>{selected ? 'breakdown ▾' : 'breakdown ▸'}</span>
        </div>
      </div>
    </button>
  );
}

function OllamaCard({ lane, selected, onSelect }: { lane: OllamaBoard; selected: boolean; onSelect: () => void }) {
  const cloud = lane.models.filter((m) => m.cloud).length;
  return (
    <button type="button" onClick={onSelect} aria-expanded={selected} data-lens="r" className={`pressable is-row w-full rounded-[12px] border bg-os-bg p-5 text-left ${selected ? 'border-os-accent' : 'border-os-border'}`}>
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <Dot state={lane.state === 'up' ? 'ok' : 'err'} />
          <span className="text-[15px] font-semibold tracking-[-0.01em]">Ollama</span>
        </div>
        <Badge tone={lane.plan ? 'accent' : 'default'}>{lane.plan ? `${lane.plan} · active` : lane.state === 'up' ? 'plan unknown' : 'server down'}</Badge>
      </div>
      <p className="mt-4 text-[12px] leading-relaxed text-os-dim">no official gauge · ollama.com exposes plan % only on its settings page</p>
      <div className="mt-4 grid grid-cols-4 gap-2">
        {WINDOWS.map((w) => {
          const r = lane.requests?.[w];
          return (
            <div key={w} className="border-l border-os-border pl-2">
              <div className="text-[19px] font-semibold tabular-nums tracking-[-0.02em]">{r ? r.chat + r.embed : ' - '}</div>
              <div className="font-mono text-[9.5px] uppercase tracking-[0.16em] text-os-dim">{WINDOW_LABEL[w]}</div>
              <div className="font-mono text-[9.5px] text-os-muted">{r ? `${r.chat} chat · ${r.embed} emb` : 'no log'}</div>
            </div>
          );
        })}
      </div>
      <div className="mt-4 flex items-center justify-between font-mono text-[10px] text-os-dim">
        <span>
          {lane.models.length} models · {cloud} cloud (bill the plan)
        </span>
        <span>{selected ? 'breakdown ▾' : 'breakdown ▸'}</span>
      </div>
    </button>
  );
}

const DETAIL = 'grid gap-6 rounded-[12px] border border-os-border bg-os-bg p-5 lg:grid-cols-[1.4fr_1fr]';
const TRACK = { background: 'color-mix(in oklab, var(--text) 8%, transparent)' };

function PlanDetail({ plan, now }: { plan: PlanUsage; now: number }) {
  const [win, setWin] = useState<UsageWindow>('day');
  const b = plan.breakdown;
  const lanes = b?.windows[win];
  const total = lanes ? laneBurn(lanes) : 0;
  const weekly = plan.official?.weekly?.usedPercent ?? null;
  const weekTotal = b ? laneBurn(b.windows.week) : 0;
  const topMax = Math.max(1, ...(b?.top.map((t) => t.burn) ?? [1]));
  const models = Object.entries(plan.byModel).sort((a, c) => burnOf(c[1]) - burnOf(a[1]));
  const modelTotal = models.reduce((s, [, t]) => s + burnOf(t), 0);
  const dayMax = Math.max(1, ...plan.days.map((d) => burnOf(d)));

  return (
    <div className={DETAIL}>
      <div className="space-y-5">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <Label>where it burns</Label>
          <div className="flex flex-wrap gap-2" role="tablist">
            {WINDOWS.map((w) => (
              <button key={w} type="button" role="tab" aria-selected={win === w} onClick={() => setWin(w)} className={chipClass(win === w)}>
                {WINDOW_LABEL[w]}
              </button>
            ))}
          </div>
        </div>

        {!b ? (
          <p className="text-[12.5px] text-os-dim">this reading carries no lane breakdown yet (an older push)</p>
        ) : (
          <>
            <div>
              <div className="flex flex-wrap items-baseline justify-between gap-2">
                <span className="text-[30px] font-semibold tabular-nums tracking-[-0.03em]">{fmtTokens(total)}</span>
                <span className="font-mono text-[11px] text-os-muted">
                  {weekly !== null && weekTotal > 0 ? `≈${pct((weekly * total) / weekTotal)} of the weekly limit${win === 'week' ? '' : ' (est.)'}` : weekTotal > 0 ? `${pct((total / weekTotal) * 100)} of this week's burn` : 'no burn'}
                </span>
              </div>
              <div className="mt-3">
                <LaneBar plan={plan} window={win} height="h-[10px]" />
              </div>
              <div className="mt-3">
                <LaneLegend />
              </div>
            </div>

            <div className="space-y-5 border-t border-os-border pt-5">
              {BURN_SOURCES.map((s, i) => {
                const v = burnOf(lanes![s]);
                const share = total > 0 ? v / total : 0;
                return (
                  <VolumeMeter key={`${win}-${s}`} label={SOURCE_LABEL[s]} frac={share} display={`${fmtTokens(v)} · ${pct(share * 100)}`} hue={LANE_HUE[s]} delay={i * 120} />
                );
              })}
            </div>

            <div>
              <Label>top burners · 7 days</Label>
              <div className="mt-3 space-y-2">
                {b.top.length === 0 && <p className="text-[12.5px] text-os-dim">nothing burned this week</p>}
                {b.top.slice(0, 12).map((t) => (
                  <div key={`${t.source}|${t.label}`} className="grid grid-cols-[1fr_120px_60px] items-center gap-3 text-[12.5px]">
                    <span className="truncate" title={t.label}>
                      <span className="text-os-text">{t.label}</span>
                      <span className="ml-2 font-mono text-[9.5px] uppercase tracking-[0.14em] text-os-dim">{SOURCE_LABEL[t.source]}</span>
                    </span>
                    <div className="h-1.5 overflow-hidden rounded-full" style={TRACK}>
                      <div className="h-full rounded-full" style={{ width: `${(t.burn / topMax) * 100}%`, background: LANE_HUE[t.source] }} />
                    </div>
                    <span className="text-right font-mono text-[11px] tabular-nums">{fmtTokens(t.burn)}</span>
                  </div>
                ))}
              </div>
            </div>
          </>
        )}
      </div>

      <div className="space-y-5">
        <div>
          <Label>7-day burn · by day</Label>
          <div className="mt-3 flex h-20 items-end gap-1.5">
            {plan.days.map((d, i) => {
              const v = burnOf(d);
              const h = v === 0 ? 2 : Math.max(3, Math.round((v / dayMax) * 72));
              const today = i === plan.days.length - 1;
              return (
                <div key={d.day} className="flex flex-1 flex-col items-center gap-1" title={`${d.day}: ${fmtTokens(v)} burn · ${fmtTokens(d.cacheRead)} context re-read`}>
                  <div className="w-full rounded-t-[4px]" style={{ height: h, background: today ? 'var(--send-activity)' : 'color-mix(in oklab, var(--send-activity) 35%, transparent)' }} />
                  <span className="font-mono text-[9px] text-os-dim">{'SMTWTFS'[new Date(d.day + 'T12:00:00').getDay()]}</span>
                </div>
              );
            })}
          </div>
          <p className="mt-1.5 font-mono text-[10px] text-os-dim">context re-read (cache reads, not burn): {fmtTokens(plan.days.reduce((s, d) => s + d.cacheRead, 0))}</p>
        </div>

        {models.length > 0 && (
          <div>
            <Label>models</Label>
            <div className="mt-2 space-y-1.5">
              {models.slice(0, 5).map(([m, t]) => (
                <div key={m} className="flex items-center justify-between text-[12.5px]">
                  <span className="text-os-muted">{m.replace(/^claude-/, '')}</span>
                  <span className="font-mono text-[11px] tabular-nums">
                    {fmtTokens(burnOf(t))}
                    <span className="text-os-dim"> · {modelTotal ? Math.round((burnOf(t) / modelTotal) * 100) : 0}%</span>
                  </span>
                </div>
              ))}
            </div>
          </div>
        )}

        <div>
          <Label>machines reporting</Label>
          <div className="mt-2 space-y-1.5">
            {plan.machines.map((m) => (
              <div key={m.id} className="flex items-center justify-between text-[12.5px]">
                <span className="flex items-center gap-2">
                  <Dot state={m.stale ? 'warn' : 'ok'} />
                  {m.label}
                </span>
                <span className="font-mono text-[10.5px] text-os-dim">{m.source === 'local' ? 'this box · live' : `pushed ${age(m.capturedAt, now)}${m.stale ? ' · stale' : ''}`}</span>
              </div>
            ))}
          </div>
          <p className="mt-2 font-mono text-[10px] leading-relaxed text-os-dim">
            Usage from connected machines appears here when they report.
          </p>
        </div>
        {plan.note && <p className="font-mono text-[10px] leading-relaxed text-os-dim">{plan.note}</p>}
      </div>
    </div>
  );
}

function OllamaDetail({ lane, now }: { lane: OllamaBoard; now: number }) {
  const max = Math.max(1, ...WINDOWS.map((w) => (lane.requests ? lane.requests[w].chat + lane.requests[w].embed : 0)));
  return (
    <div className={DETAIL}>
      <div>
        <Label>requests by window · chat vs embeddings</Label>
        <div className="mt-3 space-y-3">
          {!lane.requests && <p className="text-[12.5px] text-os-dim">no Ollama server log on this box</p>}
          {lane.requests &&
            WINDOWS.map((w) => {
              const r = lane.requests![w];
              return (
                <div key={w} className="grid grid-cols-[90px_1fr_120px] items-center gap-3 text-[12.5px]">
                  <span className="font-mono text-[10px] uppercase tracking-[0.14em] text-os-dim">{WINDOW_LABEL[w]}</span>
                  <div className="flex h-2 gap-[2px] overflow-hidden rounded-full" style={TRACK}>
                    <div className="h-full" style={{ width: `${(r.chat / max) * 100}%`, background: 'var(--ramp-1)' }} title={`${r.chat} chat`} />
                    <div className="h-full" style={{ width: `${(r.embed / max) * 100}%`, background: 'var(--ramp-3)' }} title={`${r.embed} embeddings`} />
                  </div>
                  <span className="text-right font-mono text-[11px] tabular-nums">
                    {r.chat} chat · {r.embed} emb
                  </span>
                </div>
              );
            })}
        </div>
        <p className="mt-4 font-mono text-[10px] leading-relaxed text-os-dim">{lane.note}</p>
      </div>
      <div>
        <Label>models</Label>
        <div className="mt-2 space-y-1.5">
          {lane.models.map((m) => (
            <div key={`${m.host}|${m.name}`} className="flex items-center justify-between text-[12.5px]">
              <span className="text-os-muted">
                {m.name}
                <span className="ml-2 font-mono text-[10px] text-os-dim">{m.host.replace(/^Ollama · /, '')}</span>
              </span>
              <Badge tone={m.cloud ? 'warn' : 'default'}>{m.cloud ? 'cloud · bills plan' : 'local · free'}</Badge>
            </div>
          ))}
          {lane.models.length === 0 && <p className="text-[12.5px] text-os-dim">no models listed</p>}
        </div>
        <div className="mt-5">
          <Label>machines reporting</Label>
          <div className="mt-2 space-y-1.5">
            {lane.machines.map((m) => (
              <div key={m.id} className="flex items-center justify-between text-[12.5px]">
                <span className="flex items-center gap-2">
                  <Dot state={m.stale ? 'warn' : 'ok'} />
                  {m.label}
                </span>
                <span className="font-mono text-[10.5px] text-os-dim">{m.source === 'local' ? 'this box · live' : `pushed ${age(m.capturedAt, now)}${m.stale ? ' · stale' : ''}`}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

export default function UsageBoard() {
  const [board, setBoard] = useState<Board | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [fetchedAt, setFetchedAt] = useState(0);
  const [open, setOpen] = useState<PlanKey | null>('claude');
  const timer = useRef<ReturnType<typeof setInterval> | null>(null);

  const load = useCallback(async () => {
    try {
      const res = await fetch('/api/usage', { cache: 'no-store' });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setBoard((await res.json()) as Board);
      setError(null);
      setFetchedAt(Date.now());
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }, []);

  useEffect(() => {
    void load();
    timer.current = setInterval(() => void load(), POLL_MS);
    return () => {
      if (timer.current) clearInterval(timer.current);
    };
  }, [load]);

  if (!board) {
    return (
      <div className="rounded-[12px] border border-os-border bg-os-surface px-6 py-8 text-[13px] text-os-dim">
        {error ? `usage read failed: ${error}` : 'reading local transcripts…'}
      </div>
    );
  }

  const now = fetchedAt || Date.now();
  const toggle = (k: PlanKey) => setOpen((o) => (o === k ? null : k));
  const vol = usageVolume({ claude: board.claude, codex: board.codex, ollama: board.ollama, now });

  return (
    <div>
      {/* Hero row: the week's burn line + the Deal Volume card worn by token burn */}
      <div className="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
        <SlabCard i={1} title="Token Burn" sub="Claude + Codex · last 7 days">
          <div className="px-6 pt-3">
            <span className="text-[30px] font-semibold tabular-nums tracking-[-0.03em]">{fmtTokens(vol.weekTokens)}</span>
            <span className="ml-2 text-[13px] text-os-dim">tokens burned, last 7 days</span>
          </div>
          <StepLine series={vol.series} hue="var(--send-activity)" unit="k tokens" empty="No burn in this window." />
          <div className="px-6 pb-5 font-mono text-[11px] text-os-dim">{vol.meta}</div>
        </SlabCard>

        <SlabCard i={2} title="Burn Volume" className="flex flex-col">
          <div className="flex flex-1 flex-col px-6 pb-6 pt-3">
            <BigStat value={vol.headlineTokens} kind="tokens" unit="tokens" chips={vol.chips} caption={vol.caption} />
            <MeterStack meters={vol.meters} foot={vol.foot} />
          </div>
        </SlabCard>
      </div>

      {/* Second row: models, the official limits, THE insight card */}
      <div className="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
        <SlabCard i={3} title="Burn by Model">
          {/* stacked, not side by side: model names are too long to share a row */}
          <div className="flex flex-col gap-5 px-6 pb-6 pt-3">
            <div className="min-w-0">
              {vol.topModel ? (
                <BigStat value={Math.round(vol.topModel.share * 100)} kind="pct" size={30} caption="of the week on the top model" />
              ) : (
                <BigStat display=" - " size={30} caption="no model burn recorded this week" />
              )}
              {vol.topModel && (
                <div className="mt-4 w-fit max-w-full truncate rounded-full border border-os-border px-3 py-1 text-[12px] text-os-muted">
                  Top: <span className="font-semibold">{vol.topModel.label}</span>
                </div>
              )}
            </div>
            <DotMatrix cols={vol.models} hue="var(--ramp-1)" />
          </div>
        </SlabCard>

        <SlabCard i={4} title="Plan Limits" sub="official gauges only">
          <div className="px-6 pb-6">
            <MeterStack meters={vol.limits} empty="No provider reported an official gauge. Burn is measured from transcripts." baseDelay={700} />
          </div>
        </SlabCard>

        <InsightCard i={5} badge="Top burner · 7 days" value={vol.insight.value} kind="pct" headline={vol.insight.headline} body={vol.insight.body} frac={vol.insight.frac} />
      </div>

      {/* The plans: click a card to open where its tokens went */}
      <SlabCard
        i={6}
        className="mt-6"
        title="Plans"
        sub={`${[board.claude, board.codex].filter(Boolean).length + 1} providers`}
        action={
          <>
            {(
              [
                ['claude', 'Claude'],
                ['codex', 'ChatGPT · Codex'],
                ['ollama', 'Ollama'],
              ] as Array<[PlanKey, string]>
            ).map(([k, label]) => (
              <button key={k} type="button" onClick={() => toggle(k)} className={chipClass(open === k)}>
                {label}
              </button>
            ))}
          </>
        }
      >
        <div className="space-y-4 px-6 pb-5 pt-4">
          <div className="grid gap-4 lg:grid-cols-3">
            <PlanCard title="Claude" plan={board.claude} selected={open === 'claude'} onSelect={() => toggle('claude')} now={now} />
            <PlanCard title="ChatGPT · Codex" plan={board.codex} selected={open === 'codex'} onSelect={() => toggle('codex')} now={now} />
            <OllamaCard lane={board.ollama} selected={open === 'ollama'} onSelect={() => toggle('ollama')} />
          </div>

          {open === 'claude' && board.claude && <PlanDetail plan={board.claude} now={now} />}
          {open === 'codex' && board.codex && <PlanDetail plan={board.codex} now={now} />}
          {open === 'ollama' && <OllamaDetail lane={board.ollama} now={now} />}

          <div className="flex flex-wrap items-center justify-between gap-2 border-t border-os-border pt-3 font-mono text-[10.5px] text-os-dim">
            <span>refreshes every {POLL_MS / 1000}s · local file parsing only · no paid calls</span>
            <span>{error ? `stale  -  last poll failed (${error})` : `updated ${age(board.generatedAt, Date.now())}`}</span>
          </div>
        </div>
      </SlabCard>
    </div>
  );
}
