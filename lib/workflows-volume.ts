import type { Meter, Tone } from '@/components/slab';
import type { SeriesPoint } from '@/components/slab-charts';
import type { ScheduledJobRow } from '@/lib/scheduled-jobs';
import type { Workflow } from '@/lib/schemas';
import { workflowStats } from '@/lib/workflow-stats';
import { shortLabels } from '@/lib/short-labels';

/**
 * /workflows in the Brand Deals look (Alex, 2026-09-24). One pure pass over
 * the page's own rows (the scheduled jobs, the workflows table and the real
 * cron_runs history) produces every number the slab shows: the Cron Volume
 * headline and meters, the run-activity step line, the weekday rhythm, the
 * process load and the single "Needs you" insight. The page stays a renderer.
 */
export type VolumeJob = Pick<ScheduledJobRow, 'description' | 'enabled' | 'unknownAgent' | 'overdue' | 'lastOk' | 'runs' | 'ok'>;

export type WorkflowsVolume = {
  headline: number;
  counts: { healthy: number; overdue: number; failing: number; paused: number; enabled: number };
  chips: Array<{ tone?: Tone; text: string }>;
  caption: string;
  meters: Meter[];
  foot: string;
  series: SeriesPoint[];
  runsInWindow: number;
  failedInWindow: number;
  rhythm: SeriesPoint[];
  load: { manualHours: number; agentHours: number; perWorkflow: SeriesPoint[] };
  insight: { value: number; headline: string; body: string; frac: number };
};

type JobState = 'healthy' | 'overdue' | 'failing' | 'paused';

/** Mirrors ScheduledTasks' own state(): a missing agent can only fail, a late
    slot is overdue, then the last outcome decides. */
function jobState(j: VolumeJob): JobState {
  if (!j.enabled) return 'paused';
  if (j.unknownAgent) return 'failing';
  if (j.overdue) return 'overdue';
  return j.lastOk === false ? 'failing' : 'healthy';
}

const frac = (n: number, d: number) => (d > 0 ? Math.max(0, Math.min(1, n / d)) : 0);
const pct = (f: number) => `${Math.round(f * 100)}%`;
const usd = (n: number) => `$${Math.round(n).toLocaleString('en-US')}`;
const hours = (n: number) => Math.round(n * 10) / 10;
const localDay = (d: Date) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
const dayLabel = (d: Date) => d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
const WEEKDAYS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];


export function workflowsVolume(x: {
  jobs: VolumeJob[];
  workflows: Workflow[];
  runs: { startedAt: string; ok: boolean }[];
  now?: Date;
  days?: number;
}): WorkflowsVolume {
  const now = x.now ?? new Date();
  const days = x.days ?? 14;

  // Crons: one state each, so the chips always add up to the headline.
  const states = x.jobs.map(jobState);
  const count = (s: JobState) => states.filter((t) => t === s).length;
  const counts = { healthy: count('healthy'), overdue: count('overdue'), failing: count('failing'), paused: count('paused'), enabled: 0 };
  counts.enabled = x.jobs.length - counts.paused;
  const runsAll = x.jobs.reduce((n, j) => n + j.runs, 0);
  const okAll = x.jobs.reduce((n, j) => n + j.ok, 0);

  const chips: WorkflowsVolume['chips'] = [];
  if (counts.healthy) chips.push({ tone: 'ok', text: `${counts.healthy} healthy` });
  if (counts.overdue) chips.push({ tone: 'warn', text: `${counts.overdue} overdue` });
  if (counts.failing) chips.push({ tone: 'err', text: `${counts.failing} failing` });
  if (counts.paused) chips.push({ text: `${counts.paused} paused` });

  // Process map: hours and money straight off the workflow steps.
  const stats = x.workflows.map(workflowStats);
  const manualHours = hours(stats.reduce((n, s) => n + s.manualHours, 0));
  const agentHours = hours(stats.reduce((n, s) => n + s.agentHours, 0));
  const totalHours = hours(manualHours + agentHours);
  const leak = stats.reduce((n, s) => n + s.leakUsd, 0);
  const recovered = stats.reduce((n, s) => n + s.liveReturnsUsd, 0);
  const steps = x.workflows.reduce((n, w) => n + w.steps.length, 0);
  const tools = new Set(x.workflows.flatMap((w) => w.steps.flatMap((s) => s.tools))).size;

  const healthyF = frac(counts.healthy, counts.enabled);
  const runsF = frac(okAll, runsAll);
  const hoursF = frac(agentHours, totalHours);
  const leakF = frac(recovered, leak);
  const meters: Meter[] = [
    {
      label: `Crons healthy (${counts.healthy}/${counts.enabled})`,
      frac: healthyF,
      display: counts.enabled ? pct(healthyF) : 'none enabled',
      hue: counts.enabled > 0 && counts.healthy === counts.enabled ? 'var(--ok)' : 'var(--warn)',
    },
    {
      label: `Runs OK (${okAll}/${runsAll})`,
      frac: runsF,
      display: runsAll ? `${okAll} ok · ${runsAll - okAll} failed` : 'no runs yet',
      hue: 'var(--accent)',
    },
    {
      label: `Hours carried by agents (${agentHours}/${totalHours}h per wk)`,
      frac: hoursF,
      display: totalHours ? pct(hoursF) : 'no steps mapped',
      hue: 'var(--ramp-1)',
    },
    {
      label: `Leak recovered (${usd(recovered)}/${usd(leak)} mo)`,
      frac: leakF,
      display: leak ? pct(leakF) : 'no leaks mapped',
      hue: 'var(--ramp-4)',
    },
  ];

  // Real cron runs, bucketed by local day over the window, oldest first.
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
  const rhythm = WEEKDAYS.map((label) => ({ label, count: 0 }));
  let runsInWindow = 0;
  let failedInWindow = 0;
  for (const r of x.runs) {
    const t = new Date(r.startedAt);
    if (!Number.isFinite(t.getTime()) || t < start || t > now) continue;
    const key = localDay(t);
    if (!buckets.has(key)) continue;
    buckets.set(key, (buckets.get(key) ?? 0) + 1);
    rhythm[(t.getDay() + 6) % 7].count += 1;
    runsInWindow += 1;
    if (!r.ok) failedInWindow += 1;
  }
  const series = labels.map(([key, label]) => ({ label, count: buckets.get(key) ?? 0 }));

  const names = shortLabels(x.workflows.map((w) => w.name), /\s+/, Infinity, 'Workflow');
  const perWorkflow = stats.map((s, i) => ({ label: names[i], count: hours(s.manualHours) }));

  // The one gradient card: what is waiting on Alex right now.
  const needs = x.jobs.filter((_, i) => states[i] === 'overdue' || states[i] === 'failing');
  const insight = {
    value: needs.length,
    headline: needs.length ? `${counts.overdue} overdue · ${counts.failing} failing.` : 'Every enabled task ran on its slot.',
    body: needs.length
      ? needs
          .slice(0, 3)
          .map((j) => j.description)
          .join(' · ')
      : x.jobs.length
        ? `${counts.healthy} healthy, nothing waiting on you.`
        : 'No scheduled tasks yet.',
    frac: frac(needs.length, counts.enabled),
  };

  return {
    headline: x.jobs.length,
    counts,
    chips,
    caption: `${counts.enabled} enabled · ${runsAll} runs recorded`,
    meters,
    foot: `${x.workflows.length} workflows · ${steps} steps · ${tools} tools`,
    series,
    runsInWindow,
    failedInWindow,
    rhythm,
    load: { manualHours, agentHours, perWorkflow },
    insight,
  };
}
