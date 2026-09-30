/**
 * /usage in the Brand Deals look (2026-09-24): the token-burn board's own
 * reading shaped like Deal Volume. Pure and client-safe, so UsageBoard (a
 * client island that polls /api/usage) can derive it on every poll. Burn is
 * the board's measure (in + out + cache writes, cache reads excluded), and
 * the only "% of limit" numbers here are the providers' OFFICIAL gauges.
 */
import {
  BURN_SOURCES,
  SOURCE_LABEL,
  burnOf,
  fmtTokens,
  laneBurn,
  type BurnSource,
  type OllamaBoard,
  type OfficialWindow,
  type PlanUsage,
} from '@/lib/usage';

export type UsageTone = 'ok' | 'warn' | 'err' | 'accent';
export type UsageMeter = { label: string; frac: number; display: string; hue: string };
export type UsagePoint = { label: string; count: number };

/** One hue per lane, the same in the meters, the stacked bars and the legend. */
export const LANE_HUE: Record<BurnSource, string> = {
  board: 'var(--accent)',
  sessions: 'var(--ramp-1)',
  terminal: 'var(--ramp-3)',
  automation: 'var(--ramp-4)',
};

export function pctTone(p: number): 'ok' | 'warn' | 'err' {
  return p >= 90 ? 'err' : p >= 70 ? 'warn' : 'ok';
}

export type UsageVolume = {
  headline: string;
  headlineTokens: number;
  chips: Array<{ tone: UsageTone; text: string }>;
  caption: string;
  meta: string;
  meters: UsageMeter[];
  foot: string;
  series: UsagePoint[];
  weekTokens: number;
  models: UsagePoint[];
  topModel: { label: string; share: number } | null;
  limits: UsageMeter[];
  insight: { value: number; headline: string; body: string; frac: number };
};

const round = (n: number) => (n >= 10 || n === 0 ? n.toFixed(0) : n.toFixed(1));

function resetsIn(resetsAt: string | null, now: number): string {
  if (!resetsAt) return '';
  const ms = new Date(resetsAt).getTime() - now;
  if (!Number.isFinite(ms)) return '';
  if (ms < 0) return 'reset since';
  return `resets in ${ms > 86400_000 ? `${Math.round(ms / 86400_000)}d` : `${Math.max(1, Math.round(ms / 3600_000))}h`}`;
}

export function limitMeter(name: string, window: string, w: OfficialWindow, now: number): UsageMeter {
  const reset = resetsIn(w.resetsAt, now);
  return {
    label: `${name} · ${window}`,
    frac: Math.max(0, Math.min(1, w.usedPercent / 100)),
    display: `${round(w.usedPercent)}%${reset ? ` · ${reset}` : ''}`,
    hue: `var(--${pctTone(w.usedPercent)})`,
  };
}

const dayLabel = (day: string) => new Date(`${day}T12:00:00`).toLocaleDateString('en-US', { month: 'short', day: 'numeric' });

export function usageVolume(x: { claude: PlanUsage | null; codex: PlanUsage | null; ollama: OllamaBoard; now?: number }): UsageVolume {
  const now = x.now ?? Date.now();
  const named = [
    { name: 'Claude', plan: x.claude },
    { name: 'Codex', plan: x.codex },
  ].filter((p): p is { name: string; plan: PlanUsage } => p.plan !== null);
  const plans = named.map((p) => p.plan);

  // 7-day series: every plan's day buckets summed by calendar day.
  const byDay = new Map<string, number>();
  for (const p of plans) for (const d of p.days) byDay.set(d.day, (byDay.get(d.day) ?? 0) + burnOf(d));
  const dayKeys = [...byDay.keys()].sort();
  const series = dayKeys.map((k) => ({ label: dayLabel(k), count: Math.round((byDay.get(k) ?? 0) / 1000) }));
  const weekTokens = dayKeys.reduce((n, k) => n + (byDay.get(k) ?? 0), 0);
  const headlineTokens = plans.reduce((n, p) => n + (p.days.length ? burnOf(p.days[p.days.length - 1]) : 0), 0);

  // Lanes: today's window of every plan that carries a breakdown.
  const laneToday = Object.fromEntries(BURN_SOURCES.map((s) => [s, 0])) as Record<BurnSource, number>;
  for (const p of plans) if (p.breakdown) for (const s of BURN_SOURCES) laneToday[s] += burnOf(p.breakdown.windows.day[s]);
  const laneTotal = BURN_SOURCES.reduce((n, s) => n + laneToday[s], 0);
  const meters = BURN_SOURCES.map((s) => ({
    label: SOURCE_LABEL[s],
    frac: laneTotal > 0 ? laneToday[s] / laneTotal : 0,
    display: fmtTokens(laneToday[s]),
    hue: LANE_HUE[s],
  }));

  const chips: UsageVolume['chips'] = [];
  for (const { name, plan } of named) {
    const s = plan.official?.session;
    if (s) chips.push({ tone: pctTone(s.usedPercent), text: `${name} 5h ${round(s.usedPercent)}%` });
  }
  if (x.ollama.state === 'down') chips.push({ tone: 'err', text: 'Ollama down' });
  else if (x.ollama.requests) {
    const r = x.ollama.requests.day.chat + x.ollama.requests.day.embed;
    chips.push({ tone: 'accent', text: `${r} Ollama request${r === 1 ? '' : 's'} today` });
  }

  const limits: UsageMeter[] = [];
  for (const { name, plan } of named) {
    if (plan.official?.session) limits.push(limitMeter(name, '5h session', plan.official.session, now));
    if (plan.official?.weekly) limits.push(limitMeter(name, 'weekly', plan.official.weekly, now));
  }

  const modelBurn = new Map<string, number>();
  for (const p of plans) for (const [m, t] of Object.entries(p.byModel)) modelBurn.set(m, (modelBurn.get(m) ?? 0) + burnOf(t));
  const modelTotal = [...modelBurn.values()].reduce((a, b) => a + b, 0);
  const models = [...modelBurn.entries()]
    .filter(([, b]) => b > 0)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 5)
    .map(([m, b]) => ({ label: shortModel(m), count: b }));

  const top = plans.flatMap((p) => p.breakdown?.top ?? []).sort((a, b) => b.burn - a.burn)[0];
  const weekLanes = plans.reduce((n, p) => n + (p.breakdown ? laneBurn(p.breakdown.windows.week) : 0), 0);
  const topShare = top && weekLanes > 0 ? top.burn / weekLanes : 0;

  const machines = new Set([...plans.flatMap((p) => p.machines), ...x.ollama.machines].map((m) => m.id)).size;
  const planNames = [...plans.map((p) => p.plan), x.ollama.plan].filter((p): p is string => Boolean(p));

  return {
    headline: fmtTokens(headlineTokens),
    headlineTokens,
    chips,
    caption: `tokens burned today across Claude + Codex · ${fmtTokens(weekTokens)} over 7 days`,
    meta: [`${fmtTokens(headlineTokens)} today`, `${fmtTokens(weekTokens)} this week`, `${machines} machine${machines === 1 ? '' : 's'} reporting`, ...planNames].join(' · '),
    meters,
    foot: 'burn = input + output + cache writes · cache reads not counted',
    series,
    weekTokens,
    models,
    topModel: models.length && modelTotal > 0 ? { label: models[0].label, share: models[0].count / modelTotal } : null,
    limits,
    insight: top && top.burn > 0
      ? {
          value: Math.round(topShare * 100),
          headline: top.label,
          body: `${SOURCE_LABEL[top.source]} · ${fmtTokens(top.burn)} of ${fmtTokens(weekLanes)} burned this week`,
          frac: topShare,
        }
      : { value: 0, headline: 'Nothing burned this week.', body: 'No transcripts on any reporting machine yet.', frac: 0 },
  };
}

/** A model name short enough for a dot-matrix column: no claude- prefix, no dated suffix. */
export function shortModel(m: string): string {
  return m.replace(/^claude-/, '').replace(/-\d{8}$/, '');
}
