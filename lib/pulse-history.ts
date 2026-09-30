/**
 * Honest inputs for the operator-console pulse row. `runsPerDay` / `inboundPerDay`
 * turn real timestamped history (agent_runs, comms feed) into per-day series for
 * the sparklines; `stateOfWorld` distills live facts into one attention-first
 * status sentence. Pure + tested so the page stays a thin renderer  -  and so the
 * pulse row never shows a fabricated trend again.
 */

const DAY_MS = 86_400_000;

/** UTC date keys (YYYY-MM-DD) for the last `days` days, oldest first. */
function dayKeys(days: number, nowMs: number): string[] {
  const keys: string[] = [];
  for (let i = days - 1; i >= 0; i--) {
    keys.push(new Date(nowMs - i * DAY_MS).toISOString().slice(0, 10));
  }
  return keys;
}

/** Count ISO timestamps into the last `days` UTC-day buckets (oldest first). */
function countPerDay(timestamps: (string | undefined)[], days: number, nowMs: number): number[] {
  const keys = dayKeys(days, nowMs);
  const index = new Map(keys.map((k, i) => [k, i]));
  const counts = new Array<number>(days).fill(0);
  for (const ts of timestamps) {
    if (!ts) continue;
    const t = Date.parse(ts);
    if (!Number.isFinite(t)) continue;
    const i = index.get(new Date(t).toISOString().slice(0, 10));
    if (i !== undefined) counts[i] += 1;
  }
  return counts;
}

export function runsPerDay(runs: { startedAt?: string }[], days = 7, nowMs = Date.now()): number[] {
  return countPerDay(
    runs.map((r) => r.startedAt),
    days,
    nowMs,
  );
}

export function inboundPerDay(items: { ts?: string }[], days = 7, nowMs = Date.now()): number[] {
  return countPerDay(
    items.map((i) => i.ts),
    days,
    nowMs,
  );
}

export type Tone = 'ok' | 'warn' | 'err' | 'accent' | 'dim';
export type StateSegment = { text: string; tone: Tone };

export type PulseFacts = {
  activeAgents: number;
  totalAgents: number;
  connectorsDown: number; // connectors in an ERROR state; not_configured is NOT "down"
  inbound: number;
  health: number | null;
  brainConnected: boolean;
  failedRuns: number;
};

const plural = (n: number, word: string) => `${n} ${word}${n === 1 ? '' : 's'}`;

/**
 * One honest sentence about what needs the operator, worst-first: failed runs,
 * an unhealthy brain, downed connectors, then inbound to reply to  -  always
 * anchored by the live agent count. Leads with "All nominal" only when nothing
 * is actually wrong.
 */
export function stateOfWorld(f: PulseFacts): StateSegment[] {
  const segs: StateSegment[] = [];

  if (f.failedRuns > 0) segs.push({ text: `${plural(f.failedRuns, 'run')} failed`, tone: 'err' });
  if (!f.brainConnected) segs.push({ text: 'G-Brain offline', tone: 'err' });
  else if (f.health != null && f.health < 70) segs.push({ text: `G-Brain degraded ${f.health}/100`, tone: 'warn' });
  if (f.connectorsDown > 0) segs.push({ text: `${plural(f.connectorsDown, 'connector')} down`, tone: 'warn' });
  if (f.inbound > 0) segs.push({ text: `${f.inbound} inbound need reply`, tone: 'accent' });

  const hadAttention = segs.length > 0;

  // Mock 3a runs the roster as "3 agents live · 6 idle". Idle is not a
  // problem, so it never counts toward hadAttention, but leaving it out made
  // "19/32" a fraction the reader has to do arithmetic on to learn the only
  // thing it is actually saying: thirteen seats are sitting still.
  const idle = Math.max(0, f.totalAgents - f.activeAgents);
  segs.push({ text: `${f.activeAgents} agents live`, tone: f.activeAgents > 0 ? 'ok' : 'dim' });
  segs.push({ text: `${idle} idle`, tone: 'dim' });

  // The artboard states the brain score at every health ("brain 78/100"),
  // not only when it has gone bad. A number that appears only on failure
  // teaches the reader to read its absence as "fine", which is exactly the
  // habit that let /api/metrics report a null page count for two weeks.
  if (f.brainConnected && f.health != null && f.health >= 70) {
    segs.push({ text: `brain ${f.health}/100`, tone: 'ok' });
  }

  if (!hadAttention) segs.unshift({ text: 'All nominal', tone: 'ok' });

  return segs;
}

// ── Operating volume (Home's Deal Volume card, 2026-09-24) ──────────────────

export type VolumeMeterRow = { label: string; frac: number; display: string; hue: string };
export type OperatingVolume = {
  runsToday: number;
  failedToday: number;
  agentsToday: number;
  meters: VolumeMeterRow[];
};

const pct = (n: number, d: number) => (d > 0 ? n / d : 0);

/** Home's own numbers shaped like Brand Deals' Deal Volume: a headline for
    today plus four meters, each a real fraction of its own whole. Runs count
    toward "today" by local midnight, the day Alex lived. */
export function operatingVolume(x: {
  connected: number;
  totalConnections: number;
  activeAgents: number;
  totalAgents: number;
  health: number | null;
  runs: { agentId: string; ok: boolean; finishedAt: string }[];
  now?: Date;
}): OperatingVolume {
  const now = x.now ?? new Date();
  const start = new Date(now);
  start.setHours(0, 0, 0, 0);
  const today = x.runs.filter((r) => {
    const t = Date.parse(r.finishedAt);
    return Number.isFinite(t) && t >= start.getTime() && t <= now.getTime();
  });
  const ok = today.filter((r) => r.ok).length;
  return {
    runsToday: today.length,
    failedToday: today.length - ok,
    agentsToday: new Set(today.map((r) => r.agentId)).size,
    meters: [
      { label: `Systems connected (${x.connected}/${x.totalConnections})`, frac: pct(x.connected, x.totalConnections), display: `${Math.round(pct(x.connected, x.totalConnections) * 100)}%`, hue: 'var(--accent)' },
      { label: `Agents live (${x.activeAgents}/${x.totalAgents})`, frac: pct(x.activeAgents, x.totalAgents), display: `${Math.round(pct(x.activeAgents, x.totalAgents) * 100)}%`, hue: 'var(--ramp-1)' },
      { label: `Runs OK today (${ok}/${today.length})`, frac: pct(ok, today.length), display: today.length ? `${ok} ok · ${today.length - ok} failed` : 'no runs yet', hue: today.length - ok > 0 ? 'var(--warn)' : 'var(--ok)' },
      { label: 'G-Brain health', frac: x.health === null ? 0 : Math.max(0, Math.min(1, x.health / 100)), display: x.health === null ? 'offline' : `${x.health} / 100`, hue: 'var(--ramp-4)' },
    ],
  };
}

// ── Home's second row (the Brand Deals activity line, dots and insight) ─────

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

/** The last `days` LOCAL days, oldest first, each labelled "Sep 3" and counted
    from real timestamps. Quiet days stay in as zeros so the line is honest. */
export function dailySeries(timestamps: (string | undefined)[], days = 14, now: Date = new Date()): { label: string; count: number }[] {
  const start = new Date(now);
  start.setHours(0, 0, 0, 0);
  const out = Array.from({ length: days }, (_, i) => {
    const d = new Date(start);
    d.setDate(d.getDate() - (days - 1 - i));
    return { key: d.getTime(), label: `${MONTHS[d.getMonth()]} ${d.getDate()}`, count: 0 };
  });
  for (const ts of timestamps) {
    if (!ts) continue;
    const t = new Date(ts);
    if (!Number.isFinite(t.getTime()) || t.getTime() > now.getTime()) continue;
    t.setHours(0, 0, 0, 0);
    const slot = out.find((o) => o.key === t.getTime());
    if (slot) slot.count += 1;
  }
  return out.map(({ label, count }) => ({ label, count }));
}

const SOURCES = ['email', 'whatsapp', 'slack'] as const;

/** Inbound per source, in a fixed order, zeros kept (the DotMatrix columns). */
export function sourceMix(items: { source: string }[]): { label: string; count: number }[] {
  return SOURCES.map((s) => ({ label: s, count: items.filter((i) => i.source === s).length }));
}

/** Home's one gradient card: everything waiting on Alex right now, and
    how much of today's work that is (waiting / (waiting + done today)). */
export function homeAttention(x: { inbound: number; failedToday: number; connectorsDown: number; doneToday: number }): { count: number; headline: string; frac: number } {
  const count = x.inbound + x.failedToday + x.connectorsDown;
  const parts = [
    x.inbound > 0 ? `${x.inbound} inbound` : null,
    x.failedToday > 0 ? plural(x.failedToday, 'failed run') : null,
    x.connectorsDown > 0 ? `${plural(x.connectorsDown, 'connector')} down` : null,
  ].filter(Boolean);
  return {
    count,
    headline: count === 0 ? 'Nothing is waiting on you.' : parts.join(' · '),
    frac: count === 0 ? 0 : count / (count + Math.max(0, x.doneToday)),
  };
}
