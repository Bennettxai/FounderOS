import type { Meter, Tone } from '@/components/slab';
import type { SeriesPoint } from '@/components/slab-charts';
import type { BoardLivePayload } from '@/lib/board-live';
import { boardStats, RUN_OK } from '@/lib/board-live';
import { dailySeries } from '@/lib/pulse-history';
import { shortLabels } from '@/lib/short-labels';

/**
 * /agents in the Brand Deals look (Alex, 2026-09-24). One pure pass over
 * the live Paperclip payload BoardLive already polls produces every number the
 * slab shows: the Agent Volume headline, chips and meters, the run-activity
 * step line, runs by seat, the open task lanes and the single "Needs you"
 * insight. An unreachable board yields empty meters and says so.
 */
export type AgentsVolume = {
  headline: number;
  counts: { running: number; idle: number; paused: number; error: number };
  chips: Array<{ tone?: Tone; text: string }>;
  caption: string;
  meters: Meter[];
  foot: string;
  openTasks: number;
  series: SeriesPoint[];
  /** What the run numbers cover: '14d', or 'since Sep 22' when the board's
      capped page of runs starts inside the window. */
  window: string;
  runsInWindow: number;
  failedInWindow: number;
  bySeat: SeriesPoint[];
  lanes: SeriesPoint[];
  insight: { value: number; headline: string; body: string; frac: number };
};

const PENDING = new Set(['running', 'queued', 'pending']);
const frac = (n: number, d: number) => (d > 0 ? Math.max(0, Math.min(1, n / d)) : 0);
const pct = (f: number) => `${Math.round(f * 100)}%`;
const plural = (n: number, word: string) => `${n} ${word}${n === 1 ? '' : 's'}`;

/** Open stages in board order; the board spells review two ways. */
const LANES: Array<[string, string[]]> = [
  ['doing', ['in_progress']],
  ['review', ['in_review', 'review']],
  ['todo', ['todo']],
  ['blocked', ['blocked']],
  ['backlog', ['backlog']],
];

/** The board hands back its latest runs only (app/agents + /api/board/live
    fetch 120); a full page may not reach back across the whole window. */
export const RUNS_FETCHED = 120;

export function agentsVolume(
  p: Pick<BoardLivePayload, 'connected' | 'agents' | 'issues' | 'runs'>,
  now: Date = new Date(),
  days = 14,
): AgentsVolume {
  const stats = boardStats(p, now.getTime());
  const byStatus = (s: string) => p.agents.filter((a) => a.status === s).length;
  const counts = { running: byStatus('running'), idle: byStatus('idle'), paused: byStatus('paused'), error: byStatus('error') };

  // Runs inside the window, by local day. The board returns its latest runs
  // only, so the line is the fetched history, never an extrapolation.
  const series = dailySeries(p.runs.map((r) => r.startedAt ?? undefined), days, now);
  const start = new Date(now);
  start.setHours(0, 0, 0, 0);
  start.setDate(start.getDate() - (days - 1));
  const inWindow = p.runs.filter((r) => {
    const t = r.startedAt ? new Date(r.startedAt).getTime() : NaN;
    return Number.isFinite(t) && t >= start.getTime() && t <= now.getTime();
  });
  const oldest = Math.min(...p.runs.map((r) => (r.startedAt ? new Date(r.startedAt).getTime() : Infinity)));
  const truncated = p.runs.length >= RUNS_FETCHED && Number.isFinite(oldest) && oldest > start.getTime();
  const window = truncated
    ? `since ${new Date(oldest).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}`
    : `${days}d`;
  const failed = (s: string) => !RUN_OK.has(s) && !PENDING.has(s);
  const finished = inWindow.filter((r) => !PENDING.has(r.status));
  const okRuns = finished.filter((r) => RUN_OK.has(r.status)).length;
  const failedInWindow = finished.filter((r) => failed(r.status)).length;
  const dayAgo = now.getTime() - 86_400_000;
  const failed24h = inWindow.filter((r) => failed(r.status) && new Date(r.startedAt as string).getTime() > dayAgo).length;

  // Runs by seat: busiest first, the seat's first word as the column label.
  const nameById = new Map(p.agents.map((a) => [a.id, a.name]));
  const perSeat = new Map<string, number>();
  for (const r of inWindow) {
    const name = r.agentName ?? nameById.get(r.agentId) ?? r.agentId.slice(0, 8);
    perSeat.set(name, (perSeat.get(name) ?? 0) + 1);
  }
  const topSeats = [...perSeat.entries()].sort((a, b) => b[1] - a[1]).slice(0, 6);
  const seatLabels = shortLabels(topSeats.map(([name]) => name));
  const bySeat = topSeats.map(([, count], i) => ({ label: seatLabels[i], count }));

  const live = p.issues.filter((i) => i.status !== 'cancelled');
  const done = live.filter((i) => i.status === 'done').length;
  const lanes = LANES.map(([label, statuses]) => ({ label, count: live.filter((i) => statuses.includes(i.status)).length }));
  const models = new Set(p.agents.map((a) => a.model).filter(Boolean)).size;

  if (!p.connected) {
    return {
      headline: 0,
      counts,
      chips: [{ tone: 'err', text: 'board unreachable' }],
      caption: 'Paperclip is not answering, so there is nothing to count',
      meters: [],
      foot: 'no live board',
      openTasks: 0,
      series,
      window,
      runsInWindow: 0,
      failedInWindow: 0,
      bySeat: [],
      lanes,
      insight: { value: 0, headline: 'Board unreachable.', body: 'No numbers until Paperclip answers on the tailnet.', frac: 0 },
    };
  }

  const chips: AgentsVolume['chips'] = [];
  if (counts.running) chips.push({ tone: 'ok', text: `${counts.running} running` });
  if (counts.idle) chips.push({ text: `${counts.idle} idle` });
  if (counts.paused) chips.push({ tone: 'warn', text: `${counts.paused} paused` });
  if (counts.error) chips.push({ tone: 'err', text: `${counts.error} in error` });

  const seats = stats.seats;
  const runF = frac(counts.running, seats);
  const beatF = frac(stats.heartbeats24h, seats);
  const okF = frac(okRuns, finished.length);
  const doneF = frac(done, live.length);
  const meters: Meter[] = [
    { label: `Seats running (${counts.running}/${seats})`, frac: runF, display: counts.running ? pct(runF) : 'none running', hue: 'var(--ok)' },
    { label: `Heartbeat in 24h (${stats.heartbeats24h}/${seats})`, frac: beatF, display: seats ? pct(beatF) : 'no seats', hue: 'var(--accent)' },
    {
      label: `Runs OK · ${window} (${okRuns}/${finished.length})`,
      frac: okF,
      display: finished.length ? `${okRuns} ok · ${failedInWindow} failed` : 'no finished runs',
      hue: 'var(--ramp-1)',
    },
    { label: `Tasks done (${done}/${live.length})`, frac: doneF, display: live.length ? pct(doneF) : 'no tasks', hue: 'var(--ramp-4)' },
  ];

  const waiting = lanes[1].count + lanes[3].count;
  const insight = {
    value: waiting,
    headline: waiting ? `${plural(waiting, 'task')} waiting on review or unblocking.` : 'Nothing on the board is waiting on you.',
    body: `${plural(counts.error, 'seat')} in error · ${plural(failed24h, 'failed run')} in 24h`,
    frac: frac(waiting, stats.openTasks),
  };

  return {
    headline: seats,
    counts,
    chips,
    caption: `seats on the Paperclip board · ${plural(models, 'model')}`,
    meters,
    foot: `${plural(stats.openTasks, 'open task')} · ${plural(stats.runs24h, 'run')} in 24h`,
    openTasks: stats.openTasks,
    series,
    window,
    runsInWindow: inWindow.length,
    failedInWindow,
    bySeat,
    lanes,
    insight,
  };
}
