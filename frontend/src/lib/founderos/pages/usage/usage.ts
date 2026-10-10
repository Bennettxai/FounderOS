/**
 * /usage: the token-burn board's client half. Types mirror GET
 * /api/founderos/pages/usage (Go internal/founderos/pages/usage), and the helpers
 * are FounderOS v1 lib/usage.ts (display parts) + lib/usage-volume.ts, derived
 * on every 10s poll.
 *
 * Bridge difference, on purpose: the bridge parses no device files, so a plan
 * nobody pushed is null and reads unknown. Where FounderOS v1 showed "0" for an
 * empty box (it always had a local reading), this reads "—".
 */
import type { Meter, SeriesPoint, StatChip, Tone } from '$lib/founderos/kit';

// ── shapes ──────────────────────────────────────────────────────────────
export type Tot = { in: number; out: number; cacheWrite: number; cacheRead: number };
export type DayBucket = Tot & { day: string };
export type OfficialWindow = { usedPercent: number; windowMinutes: number; resetsAt: string | null };
export type Official = { session?: OfficialWindow | null; weekly?: OfficialWindow | null } | null;

export const BURN_SOURCES = ['board', 'sessions', 'terminal', 'automation'] as const;
export type BurnSource = (typeof BURN_SOURCES)[number];
export const SOURCE_LABEL: Record<BurnSource, string> = {
	board: 'Agent board',
	sessions: 'My sessions · Superset',
	terminal: 'Terminal',
	automation: 'Crons & headless'
};

export const WINDOWS = ['hour', 'session', 'day', 'week'] as const;
export type UsageWindow = (typeof WINDOWS)[number];
export const WINDOW_LABEL: Record<UsageWindow, string> = { hour: 'last hour', session: '5h session', day: 'today', week: '7 days' };

export type LaneTots = Record<BurnSource, Tot>;
export type Breakdown = {
	windows: Record<UsageWindow, LaneTots>;
	top: { source: BurnSource; label: string; burn: number }[];
};
export type PlanMachine = {
	id: string;
	label: string;
	source: 'local' | 'push';
	capturedAt: string;
	lastActivity: string | null;
	stale: boolean;
};
export type PlanUsage = {
	plan: string | null;
	official: Official;
	days: DayBucket[];
	byModel: Record<string, Tot>;
	breakdown?: Breakdown | null;
	lastActivity: string | null;
	machines: PlanMachine[];
	planConflict?: string[];
	note?: string;
};
export type RequestCounts = { chat: number; embed: number };
export type OllamaBoard = {
	state: 'up' | 'down';
	plan: string | null;
	models: { name: string; cloud: boolean; host: string }[];
	requests: Record<UsageWindow, RequestCounts> | null;
	note: string;
	machines: PlanMachine[];
};
export type UsageBody = {
	generatedAt: string;
	claude: PlanUsage | null;
	codex: PlanUsage | null;
	/** null: no machine reported an Ollama lane (unknown, not "down"). */
	ollama: OllamaBoard | null;
	errors?: Record<string, string>;
};

// ── burn ────────────────────────────────────────────────────────────────
/** Burn = input + output + cache writes; cache reads are re-read context. */
export const burnOf = (b: { in: number; out: number; cacheWrite: number }): number => b.in + b.out + b.cacheWrite;
export const totalBurn = (days: DayBucket[]): number => days.reduce((s, d) => s + burnOf(d), 0);
export const laneBurn = (l: LaneTots): number => BURN_SOURCES.reduce((a, s) => a + burnOf(l[s]), 0);

/** 1234567 → "1.2M": the board is a gauge, not an invoice. */
export function fmtTokens(n: number): string {
	if (n >= 1_000_000_000) return `${(n / 1_000_000_000).toFixed(1)}B`;
	if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
	if (n >= 1_000) return `${(n / 1_000).toFixed(1)}k`;
	return String(n);
}

export type WindowRow = { window: UsageWindow; burn: number; shareOfWeek: number; pctOfWeekly: number | null };

/** Each window's burn, its share of the week, and (only with an official
 *  weekly gauge) an ESTIMATED share of the weekly limit. */
export function windowTable(b: Breakdown, weeklyPct: number | null): WindowRow[] {
	const week = laneBurn(b.windows.week);
	return WINDOWS.map((w) => {
		const burn = laneBurn(b.windows[w]);
		const shareOfWeek = week > 0 ? burn / week : 0;
		return { window: w, burn, shareOfWeek, pctOfWeekly: weeklyPct === null ? null : weeklyPct * shareOfWeek };
	});
}

export function age(iso: string | null, now: number): string {
	if (!iso) return '—';
	const s = Math.max(0, Math.floor((now - new Date(iso).getTime()) / 1000));
	if (s < 90) return `${s}s ago`;
	if (s < 5400) return `${Math.round(s / 60)}m ago`;
	if (s < 90000) return `${Math.round(s / 3600)}h ago`;
	return `${Math.round(s / 86400)}d ago`;
}

export const pct = (n: number) => (n >= 10 || n === 0 ? `${n.toFixed(0)}%` : `${n.toFixed(1)}%`);

// ── volume (lib/usage-volume.ts) ────────────────────────────────────────
/** One hue per lane (prod lib/usage-volume.ts): accent, then globals.css
    --ramp-1 (brain-2), --ramp-3 (brain-1), --ramp-4 (brain-1 into accent). */
export const LANE_HUE: Record<BurnSource, string> = {
	board: 'var(--bn-accent)',
	sessions: 'var(--bn-brain-2)',
	terminal: 'var(--bn-brain-1)',
	automation: 'color-mix(in oklab, var(--bn-brain-1) 70%, var(--bn-accent))'
};
/** The step line / day bars hue (FounderOS v1 --send-activity). */
export const ACTIVITY_HUE = 'color-mix(in oklab, var(--bn-brain-2) 70%, var(--bn-text))';
/** FounderOS v1 --ramp-1 / --ramp-3 (model waffle, Ollama chat vs embeddings). */
export const RAMP_1 = 'var(--bn-brain-2)';
export const RAMP_3 = 'var(--bn-brain-1)';

export function pctTone(p: number): 'ok' | 'warn' | 'err' {
	return p >= 90 ? 'err' : p >= 70 ? 'warn' : 'ok';
}

const round = (n: number) => (n >= 10 || n === 0 ? n.toFixed(0) : n.toFixed(1));

function resetsIn(resetsAt: string | null, now: number): string {
	if (!resetsAt) return '';
	const ms = new Date(resetsAt).getTime() - now;
	if (!Number.isFinite(ms)) return '';
	if (ms < 0) return 'reset since';
	return `resets in ${ms > 86400_000 ? `${Math.round(ms / 86400_000)}d` : `${Math.max(1, Math.round(ms / 3600_000))}h`}`;
}

export function limitMeter(name: string, window: string, w: OfficialWindow, now: number): Meter {
	const reset = resetsIn(w.resetsAt, now);
	return {
		label: `${name} · ${window}`,
		frac: Math.max(0, Math.min(1, w.usedPercent / 100)),
		display: `${round(w.usedPercent)}%${reset ? ` · ${reset}` : ''}`,
		hue: `var(--bn-${pctTone(w.usedPercent)})`
	};
}

/** A model name short enough for a dot-matrix column. */
export function shortModel(m: string): string {
	return m.replace(/^claude-/, '').replace(/-\d{8}$/, '');
}

const dayLabel = (day: string) => new Date(`${day}T12:00:00`).toLocaleDateString('en-US', { month: 'short', day: 'numeric' });

export type UsageVolume = {
	/** null: no Claude or Codex plan reported, so today's burn is unknown. */
	headlineTokens: number | null;
	chips: StatChip[];
	caption: string;
	meta: string;
	meters: Meter[];
	foot: string;
	series: SeriesPoint[];
	weekTokens: number | null;
	models: SeriesPoint[];
	topModel: { label: string; share: number } | null;
	limits: Meter[];
	insight: { value: number | null; headline: string; body: string; frac: number | null };
};

export function usageVolume(x: { claude: PlanUsage | null; codex: PlanUsage | null; ollama: OllamaBoard | null; now?: number }): UsageVolume {
	const now = x.now ?? Date.now();
	const named = [
		{ name: 'Claude', plan: x.claude },
		{ name: 'Codex', plan: x.codex }
	].filter((p): p is { name: string; plan: PlanUsage } => p.plan !== null);
	const plans = named.map((p) => p.plan);
	const known = plans.length > 0;

	const byDay = new Map<string, number>();
	for (const p of plans) for (const d of p.days) byDay.set(d.day, (byDay.get(d.day) ?? 0) + burnOf(d));
	const dayKeys = [...byDay.keys()].sort();
	const series = dayKeys.map((k) => ({ label: dayLabel(k), count: Math.round((byDay.get(k) ?? 0) / 1000) }));
	const weekTokens = known ? dayKeys.reduce((n, k) => n + (byDay.get(k) ?? 0), 0) : null;
	const headlineTokens = known ? plans.reduce((n, p) => n + (p.days.length ? burnOf(p.days[p.days.length - 1]) : 0), 0) : null;

	// Lanes: today's window of every plan that carries a breakdown. No
	// breakdown anywhere means the lanes are unknown, not zero.
	const withLanes = plans.filter((p) => p.breakdown);
	const laneToday = Object.fromEntries(BURN_SOURCES.map((s) => [s, 0])) as Record<BurnSource, number>;
	for (const p of withLanes) for (const s of BURN_SOURCES) laneToday[s] += burnOf(p.breakdown!.windows.day[s]);
	const laneTotal = BURN_SOURCES.reduce((n, s) => n + laneToday[s], 0);
	const meters: Meter[] = BURN_SOURCES.map((s) => ({
		label: SOURCE_LABEL[s],
		frac: withLanes.length === 0 ? null : laneTotal > 0 ? laneToday[s] / laneTotal : 0,
		display: withLanes.length === 0 ? 'unknown' : fmtTokens(laneToday[s]),
		hue: LANE_HUE[s]
	}));

	const chips: StatChip[] = [];
	for (const { name, plan } of named) {
		const s = plan.official?.session;
		if (s) chips.push({ tone: pctTone(s.usedPercent) as Tone, text: `${name} 5h ${round(s.usedPercent)}%` });
	}
	if (x.ollama?.state === 'down') chips.push({ tone: 'err', text: 'Ollama down' });
	else if (x.ollama?.requests) {
		const r = x.ollama.requests.day.chat + x.ollama.requests.day.embed;
		chips.push({ tone: 'accent', text: `${r} Ollama request${r === 1 ? '' : 's'} today` });
	}

	const limits: Meter[] = [];
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

	const machines = new Set([...plans.flatMap((p) => p.machines), ...(x.ollama?.machines ?? [])].map((m) => m.id)).size;
	const planNames = [...plans.map((p) => p.plan), x.ollama?.plan ?? null].filter((p): p is string => Boolean(p));

	const insight: UsageVolume['insight'] = !known
		? { value: null, headline: 'No machine has reported usage yet.', body: 'Seats arrive from each Mac’s collector push.', frac: null }
		: top && top.burn > 0
			? {
					value: Math.round(topShare * 100),
					headline: top.label,
					body: `${SOURCE_LABEL[top.source]} · ${fmtTokens(top.burn)} of ${fmtTokens(weekLanes)} burned this week`,
					frac: topShare
				}
			: withLanes.length === 0
				? { value: null, headline: 'No lane breakdown reported.', body: 'The reporting machines sent burn without where it went.', frac: null }
				: { value: 0, headline: 'Nothing burned this week.', body: 'No transcripts on any reporting machine yet.', frac: 0 };

	return {
		headlineTokens,
		chips,
		caption: known
			? `tokens burned today across Claude + Codex · ${fmtTokens(weekTokens ?? 0)} over 7 days`
			: 'no Claude or Codex seat has been pushed yet · burn unknown',
		meta: [
			known ? `${fmtTokens(headlineTokens ?? 0)} today` : 'today unknown',
			known ? `${fmtTokens(weekTokens ?? 0)} this week` : 'week unknown',
			`${machines} machine${machines === 1 ? '' : 's'} reporting`,
			...planNames
		].join(' · '),
		meters,
		foot: 'burn = input + output + cache writes · cache reads not counted',
		series,
		weekTokens,
		models,
		topModel: models.length && modelTotal > 0 ? { label: models[0].label, share: models[0].count / modelTotal } : null,
		limits,
		insight
	};
}
