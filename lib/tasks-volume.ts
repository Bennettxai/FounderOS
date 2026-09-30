import type { Meter, Tone } from '@/components/slab';
import type { SeriesPoint } from '@/components/slab-charts';
import type { ScheduledJobRow } from '@/lib/scheduled-jobs';
import type { AgentTask } from '@/lib/schemas';
import type { PaperclipIssue } from '@/lib/connectors/paperclip';
import { shortLabels } from '@/lib/short-labels';

/**
 * /tasks in the Brand Deals look (Alex, 2026-09-24). One pure pass over the
 * page's own rows (the local kanban, the live Paperclip board issues, the
 * scheduled jobs and the real cron_runs history) gives the Task Volume
 * headline and meters, the work-activity step line, the cron weekday rhythm,
 * the owner dot matrix and the single "Needs you" card. The page renders.
 */
export type TasksVolumeInput = {
  tasks: Array<Pick<AgentTask, 'agentId' | 'status' | 'updatedAt'>>;
  issues: Array<Pick<PaperclipIssue, 'status' | 'updatedAt'>>;
  jobs: Array<Pick<ScheduledJobRow, 'description' | 'enabled' | 'unknownAgent' | 'overdue' | 'lastOk' | 'runs' | 'ok'>>;
  runs: Array<{ startedAt: string; ok: boolean }>;
  agentNames: Record<string, string>;
  now?: Date;
  days?: number;
};

export type TasksVolume = {
  headline: number;
  counts: Record<AgentTask['status'], number>;
  board: { total: number; inProgress: number; blocked: number; done: number };
  chips: Array<{ tone?: Tone; text: string }>;
  caption: string;
  meters: Meter[];
  foot: string;
  series: SeriesPoint[];
  touchesInWindow: number;
  cron: { runsInWindow: number; failedInWindow: number; rhythm: SeriesPoint[] };
  owners: SeriesPoint[];
  busiestOwner: { name: string; count: number } | null;
  insight: { value: number; headline: string; body: string; frac: number };
};

const OWNER_COLS = 6;
const WEEKDAYS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];
const frac = (n: number, d: number) => (d > 0 ? Math.max(0, Math.min(1, n / d)) : 0);
const pct = (f: number) => `${Math.round(f * 100)}%`;
const plural = (n: number, one: string, many = `${one}s`) => `${n} ${n === 1 ? one : many}`;
const localDay = (d: Date) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
const dayLabel = (d: Date) => d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });

/** Mirrors the scheduled-jobs state: a missing agent can only fail, a late
    slot is overdue, then the last outcome decides. */
const jobHealthy = (j: TasksVolumeInput['jobs'][number]) => j.enabled && !j.unknownAgent && !j.overdue && j.lastOk !== false;
const jobNeedsYou = (j: TasksVolumeInput['jobs'][number]) => j.enabled && !jobHealthy(j);


export function tasksVolume(x: TasksVolumeInput): TasksVolume {
  const now = x.now ?? new Date();
  const days = x.days ?? 14;

  // The local kanban: one lane each, so the chips add up to the kanban total.
  const counts = { open: 0, doing: 0, review: 0, done: 0 };
  for (const t of x.tasks) counts[t.status] += 1;
  const inFlight = counts.open + counts.doing + counts.review;

  // The live board: Paperclip statuses are free strings; only the three the
  // queue colors are counted by name.
  const board = {
    total: x.issues.length,
    inProgress: x.issues.filter((i) => i.status === 'in_progress').length,
    blocked: x.issues.filter((i) => i.status === 'blocked').length,
    done: x.issues.filter((i) => i.status === 'done').length,
  };

  const chips: TasksVolume['chips'] = [];
  if (counts.open) chips.push({ text: `${counts.open} to do` });
  if (counts.doing) chips.push({ tone: 'warn', text: `${counts.doing} in progress` });
  if (counts.review) chips.push({ tone: 'accent', text: `${counts.review} in review` });
  if (counts.done) chips.push({ tone: 'ok', text: `${counts.done} done` });

  // Crons: the same health the strip colors.
  const enabled = x.jobs.filter((j) => j.enabled).length;
  const healthy = x.jobs.filter(jobHealthy).length;
  const runsAll = x.jobs.reduce((n, j) => n + j.runs, 0);

  const shippedF = frac(counts.done, x.tasks.length);
  const reviewF = frac(counts.review, inFlight);
  const boardF = frac(board.inProgress, board.total);
  const healthyF = frac(healthy, enabled);
  const meters: Meter[] = [
    { label: `Kanban shipped (${counts.done}/${x.tasks.length})`, frac: shippedF, display: x.tasks.length ? pct(shippedF) : 'no tasks yet', hue: 'var(--ok)' },
    { label: `Waiting on review (${counts.review}/${inFlight} in flight)`, frac: reviewF, display: inFlight ? pct(reviewF) : 'nothing in flight', hue: 'var(--accent)' },
    { label: `Board issues in progress (${board.inProgress}/${board.total})`, frac: boardF, display: board.total ? pct(boardF) : 'board empty or offline', hue: 'var(--warn)' },
    {
      label: `Crons healthy (${healthy}/${enabled})`,
      frac: healthyF,
      display: enabled ? pct(healthyF) : 'none enabled',
      hue: enabled > 0 && healthy === enabled ? 'var(--ok)' : 'var(--warn)',
    },
  ];

  // The window: local days, oldest first, ending today.
  const start = new Date(now);
  start.setHours(0, 0, 0, 0);
  start.setDate(start.getDate() - (days - 1));
  const buckets = new Map<string, number>();
  const labels: Array<[string, string]> = [];
  for (let i = 0; i < days; i++) {
    const d = new Date(start);
    d.setDate(start.getDate() + i);
    buckets.set(localDay(d), 0);
    labels.push([localDay(d), dayLabel(d)]);
  }
  const inWindow = (iso: string | null): Date | null => {
    if (!iso) return null;
    const t = new Date(iso);
    if (!Number.isFinite(t.getTime()) || t < start || t > now || !buckets.has(localDay(t))) return null;
    return t;
  };

  // Work activity: every kanban task and board issue, on the day it last moved.
  let touchesInWindow = 0;
  for (const iso of [...x.tasks.map((t) => t.updatedAt), ...x.issues.map((i) => i.updatedAt)]) {
    const t = inWindow(iso);
    if (!t) continue;
    buckets.set(localDay(t), (buckets.get(localDay(t)) ?? 0) + 1);
    touchesInWindow += 1;
  }
  const series = labels.map(([key, label]) => ({ label, count: buckets.get(key) ?? 0 }));

  // Cron runs in the same window, by weekday (Monday first).
  const rhythm = WEEKDAYS.map((label) => ({ label, count: 0 }));
  let runsInWindow = 0;
  let failedInWindow = 0;
  for (const r of x.runs) {
    const t = inWindow(r.startedAt);
    if (!t) continue;
    rhythm[(t.getDay() + 6) % 7].count += 1;
    runsInWindow += 1;
    if (!r.ok) failedInWindow += 1;
  }

  // Owners: unfinished kanban work per agent, busiest first.
  const perAgent = new Map<string, number>();
  for (const t of x.tasks) if (t.status !== 'done') perAgent.set(t.agentId, (perAgent.get(t.agentId) ?? 0) + 1);
  const ranked = [...perAgent.entries()].sort((a, b) => b[1] - a[1]).slice(0, OWNER_COLS);
  const names = ranked.map(([id]) => x.agentNames[id] ?? id);
  const short = shortLabels(names);
  const owners = ranked.map(([, count], i) => ({ label: short[i], count }));
  const busiestOwner = ranked.length ? { name: names[0], count: ranked[0][1] } : null;
  const agentsOnKanban = new Set(x.tasks.map((t) => t.agentId)).size;

  // The one gradient card: what is waiting on Alex right now.
  const late = x.jobs.filter(jobNeedsYou);
  const value = counts.review + board.blocked + late.length;
  const parts: string[] = [];
  if (counts.review) parts.push(`${counts.review} in review`);
  if (board.blocked) parts.push(`${board.blocked} blocked`);
  if (late.length) parts.push(`${plural(late.length, 'cron')} late or failing`);
  const openIssues = board.total - board.done;
  const whole = inFlight + openIssues + enabled;
  const insight = {
    value,
    headline: parts.length ? `${parts.join(' · ')}.` : 'Nothing is waiting on you.',
    body: late.length
      ? late
          .slice(0, 3)
          .map((j) => j.description)
          .join(' · ')
      : value
        ? `${plural(inFlight, 'task')} in flight on the kanban.`
        : whole
          ? `${plural(inFlight, 'task')} open, nothing waiting on you.`
          : 'No tasks, issues or crons yet.',
    frac: frac(value, whole),
  };

  return {
    headline: x.tasks.length + x.issues.length,
    counts,
    board,
    chips,
    caption: `${x.tasks.length} on the local kanban · ${x.issues.length} on the board`,
    meters,
    foot: `${plural(x.jobs.length, 'cron')} · ${runsAll} runs recorded · ${plural(agentsOnKanban, 'agent')} on the kanban`,
    series,
    touchesInWindow,
    cron: { runsInWindow, failedInWindow, rhythm },
    owners,
    busiestOwner,
    insight,
  };
}
