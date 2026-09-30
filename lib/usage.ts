/**
 * Token-burn accounting across the model subscriptions Alex actually pays
 * for: ONE Claude plan (burned from the MacBook and the mini), ONE ChatGPT
 * plan (Codex), and ONE Ollama plan. Every token is attributed to the lane
 * that burned it (the agent board, Superset sessions, plain terminal
 * sessions, crons/headless runs) across the last hour, the 5-hour session
 * window, today and the week.
 *
 * Why this exists: the 2026-08-25 weekly-limit burn. Two 300k+ sessions ate
 * 73% of a week's Claude allowance before anyone noticed, because the only
 * gauge was `/usage` inside a running session. This module turns the local
 * evidence every CLI already writes to disk into one board, so "which seat do
 * I burn next" is a glance, not an incident.
 *
 * Sources are deliberately free and local -- no LLM, no paid API in the
 * refresh path:
 *   - Claude Code appends per-message `usage` blocks to
 *     ~/.claude/projects/**\/*.jsonl. Raw tokens, by model, by day. That is a
 *     burn ESTIMATE: Anthropic does not publish the weekly-limit formula, and
 *     this box's Keychain carries no OAuth token to ask the official number.
 *   - Codex logs `token_count` events that include the plan's OFFICIAL
 *     `used_percent` for the 5-hour and 7-day windows. Those are real gauge
 *     values, just as stale as the last time the CLI ran.
 *   - Ollama publishes no usage API at all; its lane is status-only and says
 *     so rather than faking a bar.
 *
 * Everything here is pure and fed by the connectors; the one impure surface
 * is the repo (usage_snapshots) where OTHER machines push their seat reading.
 */
import { z } from 'zod';

// ── shapes ──────────────────────────────────────────────────────────────────

export const OfficialWindowSchema = z.object({
  usedPercent: z.number().min(0),
  windowMinutes: z.number().int().positive(),
  resetsAt: z.string().nullable(),
});
export type OfficialWindow = z.infer<typeof OfficialWindowSchema>;

export const DayBucketSchema = z.object({
  day: z.string().regex(/^\d{4}-\d{2}-\d{2}$/),
  in: z.number().min(0),
  out: z.number().min(0),
  cacheWrite: z.number().min(0),
  cacheRead: z.number().min(0),
});
export type DayBucket = z.infer<typeof DayBucketSchema>;

const TotSchema = z.object({ in: z.number(), out: z.number(), cacheWrite: z.number(), cacheRead: z.number() });
export type Tot = z.infer<typeof TotSchema>;
const ModelTotalsSchema = z.record(TotSchema);

/** The lanes tokens burn in. */
export const BURN_SOURCES = ['board', 'sessions', 'terminal', 'automation'] as const;
export type BurnSource = (typeof BURN_SOURCES)[number];
export const SOURCE_LABEL: Record<BurnSource, string> = {
  board: 'Agent board',
  sessions: 'My sessions · Superset',
  terminal: 'Terminal',
  automation: 'Crons & headless',
};

export const WINDOWS = ['hour', 'session', 'day', 'week'] as const;
export type UsageWindow = (typeof WINDOWS)[number];
export const WINDOW_LABEL: Record<UsageWindow, string> = { hour: 'last hour', session: '5h session', day: 'today', week: '7 days' };

const LaneTotsSchema = z.object({ board: TotSchema, sessions: TotSchema, terminal: TotSchema, automation: TotSchema });
export type LaneTots = z.infer<typeof LaneTotsSchema>;

export const BreakdownSchema = z.object({
  windows: z.object({ hour: LaneTotsSchema, session: LaneTotsSchema, day: LaneTotsSchema, week: LaneTotsSchema }),
  /** The biggest burners this week: board seats, worktrees, projects. */
  top: z.array(z.object({ source: z.enum(BURN_SOURCES), label: z.string(), burn: z.number().min(0) })),
});
export type Breakdown = z.infer<typeof BreakdownSchema>;

export const UsageSnapshotSchema = z.object({
  id: z.string().min(1),
  kind: z.enum(['claude', 'codex']),
  label: z.string().min(1),
  /** 'local' = computed on this box just now; 'push' = another machine sent it. */
  source: z.enum(['local', 'push']),
  capturedAt: z.string().min(1),
  /** Oldest → today. Fixed length so cards line up across seats. */
  days: z.array(DayBucketSchema),
  byModel: ModelTotalsSchema,
  lastActivity: z.string().nullable(),
  /** Real plan gauges when a CLI recorded them; null means estimate-only. */
  official: z
    .object({ session: OfficialWindowSchema.optional(), weekly: OfficialWindowSchema.optional() })
    .nullable(),
  note: z.string().optional(),
  /** Where the burn went; optional so older pushed snapshots still parse. */
  breakdown: BreakdownSchema.optional(),
  /** Plan name as the provider reports it ("Claude Max 5x"), when known. */
  plan: z.string().nullable().optional(),
});
export type SeatUsage = z.infer<typeof UsageSnapshotSchema>;

export const USAGE_WINDOW_DAYS = 7;
/** A pushed snapshot older than this is shown as stale. */
export const STALE_PUSH_MS = 24 * 3600_000;

// ── day math ────────────────────────────────────────────────────────────────

/** Local-timezone calendar day for an ISO timestamp: the day Alex lived. */
export function dayKey(ts: string): string {
  const d = new Date(ts);
  const p = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

export function emptyDays(now: Date, days: number): DayBucket[] {
  const out: DayBucket[] = [];
  for (let i = days - 1; i >= 0; i--) {
    const d = new Date(now.getTime() - i * 24 * 3600_000);
    out.push({ day: dayKey(d.toISOString()), in: 0, out: 0, cacheWrite: 0, cacheRead: 0 });
  }
  return out;
}

// ── Claude transcript lines ─────────────────────────────────────────────────

export type ParsedUsageLine = {
  ts: string;
  model: string;
  cwd?: string;
  entrypoint?: string;
  sidechain?: boolean;
  in: number;
  out: number;
  cacheWrite: number;
  cacheRead: number;
};

const num = (v: unknown): number => (typeof v === 'number' && Number.isFinite(v) && v > 0 ? v : 0);

/** One transcript JSONL line → token counters, or null for anything that is
    not an assistant message with a usage block. Zeros are not evidence. */
export function parseClaudeLine(line: string): ParsedUsageLine | null {
  if (!line.trim()) return null;
  let d: Record<string, unknown>;
  try {
    d = JSON.parse(line) as Record<string, unknown>;
  } catch {
    return null;
  }
  if (d.type !== 'assistant' || typeof d.timestamp !== 'string') return null;
  const msg = d.message as { model?: unknown; usage?: Record<string, unknown> } | undefined;
  const usage = msg?.usage;
  if (!usage) return null;
  return {
    ts: d.timestamp,
    model: typeof msg?.model === 'string' ? msg.model : 'unknown',
    cwd: typeof d.cwd === 'string' ? d.cwd : undefined,
    entrypoint: typeof d.entrypoint === 'string' ? d.entrypoint : undefined,
    sidechain: d.isSidechain === true,
    in: num(usage.input_tokens),
    out: num(usage.output_tokens),
    cacheWrite: num(usage.cache_creation_input_tokens),
    cacheRead: num(usage.cache_read_input_tokens),
  };
}

/** Mutating fold: cheap, and the connector folds millions of lines. Entries
    outside the day window are dropped, never misfiled onto an edge day. */
export function foldInto(days: DayBucket[], byModel: SeatUsage['byModel'], p: ParsedUsageLine): void {
  const key = dayKey(p.ts);
  const bucket = days.find((d) => d.day === key);
  if (!bucket) return;
  bucket.in += p.in;
  bucket.out += p.out;
  bucket.cacheWrite += p.cacheWrite;
  bucket.cacheRead += p.cacheRead;
  const m = (byModel[p.model] ??= { in: 0, out: 0, cacheWrite: 0, cacheRead: 0 });
  m.in += p.in;
  m.out += p.out;
  m.cacheWrite += p.cacheWrite;
  m.cacheRead += p.cacheRead;
}

/** "Burn" = tokens that are new work for the provider: input + output + cache
    writes. Cache READS are shown separately -- they are re-read context, the
    88%-of-spend problem, but they are not the same axis. */
export function burnOf(b: { in: number; out: number; cacheWrite: number }): number {
  return b.in + b.out + b.cacheWrite;
}

export function totalBurn(days: DayBucket[]): number {
  return days.reduce((s, d) => s + burnOf(d), 0);
}

// ── Codex session events ────────────────────────────────────────────────────

export type CodexRateLimits = { session?: OfficialWindow; weekly?: OfficialWindow; planType?: string };
export type ParsedCodexLine = {
  ts: string;
  /** Cumulative for the session, as Codex reports it -- NOT a delta. */
  cumulative: { in: number; out: number; cacheRead: number };
  rateLimits: CodexRateLimits | null;
};

function codexWindow(w: unknown): OfficialWindow | undefined {
  const o = w as { used_percent?: unknown; window_minutes?: unknown; resets_at?: unknown } | undefined;
  if (!o || typeof o.used_percent !== 'number' || typeof o.window_minutes !== 'number') return undefined;
  return {
    usedPercent: o.used_percent,
    windowMinutes: o.window_minutes,
    resetsAt: typeof o.resets_at === 'number' ? new Date(o.resets_at * 1000).toISOString() : null,
  };
}

export function parseCodexLine(line: string): ParsedCodexLine | null {
  if (!line.trim()) return null;
  let d: Record<string, unknown>;
  try {
    d = JSON.parse(line) as Record<string, unknown>;
  } catch {
    return null;
  }
  const payload = d.payload as
    | { type?: unknown; info?: { total_token_usage?: Record<string, unknown> }; rate_limits?: Record<string, unknown> }
    | undefined;
  if (d.type !== 'event_msg' || payload?.type !== 'token_count') return null;
  const t = payload.info?.total_token_usage;
  if (!t) return null;
  const rl = payload.rate_limits;
  // Assign by window length, not by slot: Codex has reported the weekly
  // window as `primary` with no `secondary` since 2026-09.
  let session: OfficialWindow | undefined;
  let weekly: OfficialWindow | undefined;
  for (const w of rl ? [codexWindow(rl.primary), codexWindow(rl.secondary)] : []) {
    if (!w) continue;
    if (w.windowMinutes >= 7 * 24 * 60 * 0.9) weekly = w;
    else session = w;
  }
  const planType = typeof rl?.plan_type === 'string' ? rl.plan_type : undefined;
  return {
    ts: typeof d.timestamp === 'string' ? d.timestamp : '',
    cumulative: { in: num(t.input_tokens), out: num(t.output_tokens), cacheRead: num(t.cached_input_tokens) },
    rateLimits: session || weekly ? { session, weekly, ...(planType ? { planType } : {}) } : null,
  };
}

/** A session's worth of token_count events → its final totals plus the last
    rate-limit gauge the CLI printed. Cumulative counters: last one wins. */
export function codexSessionFold(events: ParsedCodexLine[]): {
  ts: string;
  cumulative: ParsedCodexLine['cumulative'];
  rateLimits: CodexRateLimits | null;
} {
  let ts = '';
  let cumulative = { in: 0, out: 0, cacheRead: 0 };
  let rateLimits: CodexRateLimits | null = null;
  for (const e of events) {
    ts = e.ts || ts;
    cumulative = e.cumulative;
    if (e.rateLimits) rateLimits = e.rateLimits;
  }
  return { ts, cumulative, rateLimits };
}

/** 1234567 → "1.2M" -- the board is a gauge, not an invoice. */
export function fmtTokens(n: number): string {
  if (n >= 1_000_000_000) return `${(n / 1_000_000_000).toFixed(1)}B`;
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}k`;
  return String(n);
}

// ── where the tokens go ─────────────────────────────────────────────────────

export type Lane = { source: BurnSource; label: string };

const basename = (p: string | undefined): string => {
  if (!p) return 'unknown';
  if (/^\/(private\/)?(tmp|var\/folders)(\/|$)/.test(p)) return 'tmp'; // macOS temp dirs end in a bare "T"
  return p.replace(/\/+$/, '').split('/').pop() || 'unknown';
};

/** Board seats, Superset sessions and everything else, from the working dir. */
function laneFromCwd(cwd: string | undefined): Lane | null {
  if (!cwd) return null;
  if (cwd.includes('/.paperclip/')) {
    const seat = cwd.match(/\/workspaces\/([^/]+)/)?.[1];
    return { source: 'board', label: seat ?? 'board project' };
  }
  const wt = cwd.match(/\/\.superset\/worktrees\/(.+)$/)?.[1]?.split('/').filter(Boolean);
  if (wt?.length) return { source: 'sessions', label: wt.length > 1 ? `${wt[0]} · ${wt[wt.length - 1]}` : wt[0] };
  const pj = cwd.match(/\/\.superset\/projects\/([^/]+)/)?.[1];
  if (pj) return { source: 'sessions', label: pj };
  return null;
}

/** One Claude transcript line → its lane. Headless (sdk-*) runs outside the
    board are crons and scripts; anything else interactive is a terminal. */
export function classifyClaude(x: { cwd?: string; entrypoint?: string }): Lane {
  const lane = laneFromCwd(x.cwd);
  if (lane) return lane;
  if (x.entrypoint?.startsWith('sdk')) return { source: 'automation', label: basename(x.cwd) };
  return { source: 'terminal', label: basename(x.cwd) };
}

/** One Codex session → its lane, from session_meta's cwd + originator. */
export function classifyCodex(x: { cwd?: string; originator?: string; source?: string; board?: boolean }): Lane {
  const lane = laneFromCwd(x.cwd);
  // a seat's own CODEX_HOME is the board, whichever tree it was pointed at
  if (x.board) return lane?.source === 'board' ? lane : { source: 'board', label: basename(x.cwd) };
  if (lane) return lane;
  if (x.originator === 'codex_exec' || x.source === 'exec') return { source: 'automation', label: basename(x.cwd) };
  return { source: 'terminal', label: basename(x.cwd) };
}

/** 10-minute slots: fine enough for an honest "last hour", sparse enough to keep per file. */
export const SLOT_MS = 10 * 60_000;
/** slot index (floor(ms / SLOT_MS)) → "source|label" → tokens */
export type SlotMap = Record<string, Record<string, Tot>>;

const zeroTot = (): Tot => ({ in: 0, out: 0, cacheWrite: 0, cacheRead: 0 });
const addTot = (a: Tot, b: Tot) => {
  a.in += b.in;
  a.out += b.out;
  a.cacheWrite += b.cacheWrite;
  a.cacheRead += b.cacheRead;
};
const zeroLanes = (): LaneTots => ({ board: zeroTot(), sessions: zeroTot(), terminal: zeroTot(), automation: zeroTot() });

export function slotKey(ts: string): string {
  return String(Math.floor(new Date(ts).getTime() / SLOT_MS));
}

export function addToSlots(slots: SlotMap, ts: string, lane: Lane, t: Tot): void {
  const k = slotKey(ts);
  if (k === 'NaN') return;
  const bucket = (slots[k] ??= {});
  addTot((bucket[`${lane.source}|${lane.label}`] ??= zeroTot()), t);
}

/** Drop slots older than the week so long-lived caches stay bounded. */
export function pruneSlots(slots: SlotMap, now: Date, days = USAGE_WINDOW_DAYS + 1): void {
  const floor = Math.floor((now.getTime() - days * 86400_000) / SLOT_MS);
  for (const k of Object.keys(slots)) if (Number(k) < floor) delete slots[k];
}

/** Fold slot maps (one per file/session) into the four windows by lane. */
export function breakdownFromSlots(maps: SlotMap[], now: Date, days: DayBucket[]): Breakdown {
  const windows = { hour: zeroLanes(), session: zeroLanes(), day: zeroLanes(), week: zeroLanes() };
  const hourFloor = Math.floor((now.getTime() - 3600_000) / SLOT_MS);
  const sessionFloor = Math.floor((now.getTime() - 5 * 3600_000) / SLOT_MS);
  const today = dayKey(now.toISOString());
  const weekDays = new Set(days.map((d) => d.day));
  const byKey = new Map<string, Tot>();
  for (const m of maps) {
    for (const [k, lanes] of Object.entries(m)) {
      const idx = Number(k);
      const day = dayKey(new Date(idx * SLOT_MS).toISOString());
      if (!weekDays.has(day)) continue;
      for (const [key, t] of Object.entries(lanes)) {
        const source = key.slice(0, key.indexOf('|')) as BurnSource;
        if (!BURN_SOURCES.includes(source)) continue;
        addTot(windows.week[source], t);
        addTot((byKey.get(key) ?? byKey.set(key, zeroTot()).get(key)!), t);
        if (day === today) addTot(windows.day[source], t);
        if (idx >= sessionFloor) addTot(windows.session[source], t);
        if (idx >= hourFloor) addTot(windows.hour[source], t);
      }
    }
  }
  const top = [...byKey.entries()]
    .map(([key, t]) => ({ source: key.slice(0, key.indexOf('|')) as BurnSource, label: key.slice(key.indexOf('|') + 1), burn: burnOf(t) }))
    .filter((x) => x.burn > 0)
    .sort((a, b) => b.burn - a.burn)
    .slice(0, 16);
  return { windows, top };
}

export function mergeBreakdowns(list: Breakdown[]): Breakdown | undefined {
  if (!list.length) return undefined;
  const out: Breakdown = { windows: { hour: zeroLanes(), session: zeroLanes(), day: zeroLanes(), week: zeroLanes() }, top: [] };
  const top = new Map<string, Breakdown['top'][number]>();
  for (const b of list) {
    for (const w of WINDOWS) for (const s of BURN_SOURCES) addTot(out.windows[w][s], b.windows[w][s]);
    for (const t of b.top) {
      const key = `${t.source}|${t.label}`;
      const hit = top.get(key);
      if (hit) hit.burn += t.burn;
      else top.set(key, { ...t });
    }
  }
  out.top = [...top.values()].sort((a, b) => b.burn - a.burn).slice(0, 16);
  return out;
}

export const laneBurn = (l: LaneTots): number => BURN_SOURCES.reduce((a, s) => a + burnOf(l[s]), 0);

export type WindowRow = {
  window: UsageWindow;
  burn: number;
  /** this window's burn as a share of the 7-day burn */
  shareOfWeek: number;
  /** ESTIMATED share of the weekly plan limit: official weekly % × shareOfWeek; null without an official gauge */
  pctOfWeekly: number | null;
};

export function windowTable(b: Breakdown, weeklyPct: number | null): WindowRow[] {
  const week = laneBurn(b.windows.week);
  return WINDOWS.map((w) => {
    const burn = laneBurn(b.windows[w]);
    const shareOfWeek = week > 0 ? burn / week : 0;
    return { window: w, burn, shareOfWeek, pctOfWeekly: weeklyPct === null ? null : weeklyPct * shareOfWeek };
  });
}

/** Board seat ids → agent names, when the board answered. */
export function nameBoardLabels(b: Breakdown, names: Record<string, string>): Breakdown {
  const nameOf = (label: string) => names[label] ?? Object.entries(names).find(([id]) => id.startsWith(label) || label.startsWith(id))?.[1];
  return { ...b, top: b.top.map((t) => (t.source === 'board' ? { ...t, label: nameOf(t.label) ?? t.label } : t)) };
}

// ── one plan per provider ───────────────────────────────────────────────────

export type PlanMachine = { id: string; label: string; source: SeatUsage['source']; capturedAt: string; lastActivity: string | null; stale: boolean };
export type PlanUsage = {
  plan: string | null;
  official: SeatUsage['official'];
  days: DayBucket[];
  byModel: SeatUsage['byModel'];
  breakdown: Breakdown | undefined;
  lastActivity: string | null;
  machines: PlanMachine[];
  /** set when reporting machines are logged into DIFFERENT plans: "Claude Max 20x (Claude · mini)" per plan */
  planConflict?: string[];
  note?: string;
};

/** Every machine's reading of the SAME plan, summed. The gauge comes from the
    freshest reading that has one; machines stay listed so nothing is hidden. */
export function combineSeats(seats: SeatUsage[], now: Date = new Date()): PlanUsage | null {
  if (!seats.length) return null;
  const days = seats[0].days.map((d) => ({ ...d, in: 0, out: 0, cacheWrite: 0, cacheRead: 0 }));
  const byModel: SeatUsage['byModel'] = {};
  for (const s of seats) {
    for (const d of s.days) {
      const b = days.find((x) => x.day === d.day);
      if (b) addTot(b, d);
    }
    for (const [m, t] of Object.entries(s.byModel)) addTot((byModel[m] ??= zeroTot()), t);
  }
  const withGauge = seats.filter((s) => s.official).sort((a, b) => b.capturedAt.localeCompare(a.capturedAt));
  const plans = new Map<string, string>();
  for (const s of seats) if (s.plan && !plans.has(s.plan)) plans.set(s.plan, s.label);
  const last = seats.map((s) => s.lastActivity).filter((x): x is string => Boolean(x)).sort().at(-1) ?? null;
  return {
    plan: seats.find((s) => s.plan)?.plan ?? null,
    official: withGauge[0]?.official ?? null,
    days,
    byModel,
    breakdown: mergeBreakdowns(seats.map((s) => s.breakdown).filter((b): b is Breakdown => Boolean(b))),
    lastActivity: last,
    machines: seats.map((s) => ({
      id: s.id,
      label: s.label,
      source: s.source,
      capturedAt: s.capturedAt,
      lastActivity: s.lastActivity,
      stale: s.source === 'push' && now.getTime() - new Date(s.capturedAt).getTime() > STALE_PUSH_MS,
    })),
    ...(plans.size > 1 ? { planConflict: [...plans.entries()].map(([p, who]) => `${p} (${who})`) } : {}),
    note: seats.find((s) => s.note)?.note,
  };
}

// ── plan names ──────────────────────────────────────────────────────────────

const titleCase = (s: string) => s.replace(/[_-]+/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());

/** From ~/.claude.json oauthAccount: organizationType + organizationRateLimitTier. */
export function claudePlanName(acc: { organizationType?: unknown; organizationRateLimitTier?: unknown } | undefined | null): string | null {
  const type = typeof acc?.organizationType === 'string' ? acc.organizationType : '';
  if (!type) return null;
  const tier = typeof acc?.organizationRateLimitTier === 'string' ? acc.organizationRateLimitTier : '';
  if (type === 'claude_max') {
    const x = tier.match(/(\d+)x/)?.[1];
    return x ? `Claude Max ${x}x` : 'Claude Max';
  }
  return `Claude ${titleCase(type.replace(/^claude_/, ''))}`;
}

const CHATGPT_PLANS: Record<string, string> = { plus: 'Plus', pro: 'Pro', prolite: 'Pro Lite', team: 'Team', business: 'Business', enterprise: 'Enterprise', edu: 'Edu', free: 'Free' };

/** From Codex's rate_limits.plan_type. */
export function codexPlanName(planType: string | undefined | null): string | null {
  if (!planType) return null;
  return `ChatGPT ${CHATGPT_PLANS[planType] ?? titleCase(planType)}`;
}

/** From the local Ollama server's POST /api/me `plan`. */
export function ollamaPlanName(plan: string | undefined | null): string | null {
  if (!plan) return null;
  return `Ollama ${titleCase(plan)}`;
}

// ── ollama request counts ───────────────────────────────────────────────────

export type RequestCounts = { chat: number; embed: number };
const CHAT_PATHS = ['/api/chat', '/api/generate', '/v1/chat/completions', '/v1/completions', '/v1/responses'];
const EMBED_PATHS = ['/api/embed', '/api/embeddings', '/v1/embeddings'];

/** Ollama logs one [GIN] line per request, stamped in local time, with no
    token counts. So the honest measure of how the plan burns locally is
    requests: chat/generate vs embeddings, per window. */
export function countOllamaRequests(lines: string[], now: Date): Record<UsageWindow, RequestCounts> {
  const out = { hour: { chat: 0, embed: 0 }, session: { chat: 0, embed: 0 }, day: { chat: 0, embed: 0 }, week: { chat: 0, embed: 0 } };
  const t = now.getTime();
  const today = dayKey(now.toISOString());
  for (const line of lines) {
    const m = line.match(/^\[GIN\] (\d{4})\/(\d{2})\/(\d{2}) - (\d{2}):(\d{2}):(\d{2}) \|.*?"([^"?]+)/);
    if (!m) continue;
    const p = m[7];
    const kind = CHAT_PATHS.includes(p) ? 'chat' : EMBED_PATHS.includes(p) ? 'embed' : null;
    if (!kind) continue;
    const when = new Date(+m[1], +m[2] - 1, +m[3], +m[4], +m[5], +m[6]).getTime();
    const age = t - when;
    if (age < 0 || age > USAGE_WINDOW_DAYS * 86400_000) continue;
    out.week[kind]++;
    if (dayKey(new Date(when).toISOString()) === today) out.day[kind]++;
    if (age <= 5 * 3600_000) out.session[kind]++;
    if (age <= 3600_000) out.hour[kind]++;
  }
  return out;
}

// ── the Ollama plan across machines ─────────────────────────────────────────

const CountsSchema = z.object({ chat: z.number().min(0), embed: z.number().min(0) });
export const OllamaLaneSchema = z.object({
  state: z.enum(['up', 'down']),
  plan: z.string().nullable(),
  models: z.array(z.object({ name: z.string(), cloud: z.boolean() })),
  requests: z.object({ hour: CountsSchema, session: CountsSchema, day: CountsSchema, week: CountsSchema }).nullable(),
  note: z.string(),
});
export type OllamaLaneData = z.infer<typeof OllamaLaneSchema>;

/** Another machine's Ollama reading: the plan is per ollama.com account, and
    only the machine signed in can see it. */
export const OllamaSnapshotSchema = z.object({
  kind: z.literal('ollama'),
  id: z.string().min(1),
  label: z.string().min(1),
  capturedAt: z.string().min(1),
  lane: OllamaLaneSchema,
});
export type OllamaSnapshot = z.infer<typeof OllamaSnapshotSchema>;

export type OllamaBoard = Omit<OllamaLaneData, 'models'> & {
  models: { name: string; cloud: boolean; host: string }[];
  machines: PlanMachine[];
};

/** This box's lane plus every pushed one: plan from whoever is signed in,
    requests summed, models listed per machine. */
export function combineOllama(local: { id: string; label: string; lane: OllamaLaneData } | null, pushed: OllamaSnapshot[], now: Date = new Date()): OllamaBoard {
  const all = [
    ...(local ? [{ id: local.id, label: local.label, capturedAt: now.toISOString(), lane: local.lane, source: 'local' as const }] : []),
    ...pushed.map((p) => ({ id: p.id, label: p.label, capturedAt: p.capturedAt, lane: p.lane, source: 'push' as const })),
  ];
  const withLog = all.filter((a) => a.lane.requests);
  const requests = withLog.length
    ? (Object.fromEntries(WINDOWS.map((w) => [w, withLog.reduce((acc, a) => ({ chat: acc.chat + a.lane.requests![w].chat, embed: acc.embed + a.lane.requests![w].embed }), { chat: 0, embed: 0 })])) as Record<UsageWindow, RequestCounts>)
    : null;
  return {
    state: all.some((a) => a.lane.state === 'up') ? 'up' : 'down',
    plan: all.find((a) => a.lane.plan)?.lane.plan ?? null,
    models: all.flatMap((a) => a.lane.models.map((m) => ({ ...m, host: a.label }))),
    requests,
    note: all[0]?.lane.note ?? '',
    machines: all.map((a) => ({
      id: a.id,
      label: a.label,
      source: a.source,
      capturedAt: a.capturedAt,
      lastActivity: null,
      stale: a.source === 'push' && now.getTime() - new Date(a.capturedAt).getTime() > STALE_PUSH_MS,
    })),
  };
}

export type Verdict = { recommend: string | null; reason: string };

/**
 * Which Claude seat should take the next heavy session. Official weekly
 * percentages win when both seats have them; otherwise local burn totals are
 * compared, clearly named as an estimate. One-sided data never produces a
 * recommendation -- a missing seat is a fact to surface, not a win by forfeit.
 */
export function seatVerdict(seats: SeatUsage[], now: Date = new Date()): Verdict {
  const claude = seats.filter((s) => s.kind === 'claude');
  if (claude.length < 2) return { recommend: null, reason: 'only one Claude seat configured' };

  const stale = claude.filter(
    (s) => s.source === 'push' && now.getTime() - new Date(s.capturedAt).getTime() > STALE_PUSH_MS,
  );
  const live = claude.filter((s) => !stale.includes(s));
  if (live.length < 2) {
    const names = stale.map((s) => s.label).join(', ');
    return { recommend: null, reason: `stale snapshot from ${names} — push a fresh reading before rotating` };
  }

  const official = live.filter((s) => s.official?.weekly);
  if (official.length === live.length) {
    const best = [...official].sort((a, b) => a.official!.weekly!.usedPercent - b.official!.weekly!.usedPercent)[0];
    return {
      recommend: best.id,
      reason: `lowest official weekly usage (${best.official!.weekly!.usedPercent.toFixed(0)}%)`,
    };
  }

  const withData = live.filter((s) => totalBurn(s.days) > 0 || s.lastActivity);
  if (withData.length < live.length) {
    const missing = live.filter((s) => !withData.includes(s)).map((s) => s.label);
    return { recommend: null, reason: `no data from ${missing.join(', ')}` };
  }
  const best = [...withData].sort((a, b) => totalBurn(a.days) - totalBurn(b.days))[0];
  return { recommend: best.id, reason: `lowest 7-day local burn estimate (${fmtTokens(totalBurn(best.days))})` };
}

/** 1234567 → "1.2M" -- the board is a gauge, not an invoice. */
