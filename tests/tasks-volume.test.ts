import { describe, expect, test } from 'vitest';
import { tasksVolume, type TasksVolumeInput } from '@/lib/tasks-volume';

/**
 * /tasks in the Brand Deals look (Alex, 2026-09-24: "rebuild the entire OS
 * in that light"). The Task Volume card, the work-activity step line, the cron
 * rhythm, the owner dot matrix and the one "Needs you" card all come from this
 * pure view-model, fed with the page's own rows: the local kanban (agent_tasks),
 * the live Paperclip board issues, the scheduled jobs and the real cron_runs.
 */
const NOW = new Date('2026-09-24T18:00:00');

type Job = TasksVolumeInput['jobs'][number];
const job = (over: Partial<Job> = {}): Job => ({
  description: 'Stack monitor',
  enabled: true,
  unknownAgent: false,
  overdue: false,
  lastOk: true,
  runs: 0,
  ok: 0,
  ...over,
});

const task = (agentId: string, status: 'open' | 'doing' | 'review' | 'done', updatedAt = '2026-09-24T09:00:00') => ({ agentId, status, updatedAt });
const issue = (status: string, updatedAt: string | null = '2026-09-23T09:00:00') => ({ status, updatedAt });

const INPUT: TasksVolumeInput = {
  tasks: [
    task('conductor', 'open'),
    task('conductor', 'doing', '2026-09-23T10:00:00'),
    task('closer', 'review'),
    task('closer', 'review', '2026-09-01T10:00:00'), // outside the window
    task('scout', 'done'),
    task('scout', 'done'),
  ],
  issues: [issue('in_progress'), issue('blocked'), issue('todo', null), issue('done')],
  jobs: [
    job({ description: 'Healthy', runs: 4, ok: 4 }),
    job({ description: 'Late heartbeat', overdue: true, runs: 2, ok: 2 }),
    job({ description: 'Broken ingest', lastOk: false, runs: 4, ok: 1 }),
    job({ description: 'Paused digest', enabled: false, runs: 1, ok: 1 }),
  ],
  runs: [
    { startedAt: '2026-09-24T08:00:00', ok: true },
    { startedAt: '2026-09-24T09:00:00', ok: false },
    { startedAt: '2026-09-23T08:00:00', ok: true },
    { startedAt: '2026-09-01T08:00:00', ok: true }, // outside
  ],
  agentNames: { conductor: 'Conductor Prime', closer: 'Closer', scout: 'Scout' },
  now: NOW,
  days: 14,
};

describe('tasksVolume', () => {
  const v = tasksVolume(INPUT);

  test('headline counts every work item across both queues', () => {
    expect(v.headline).toBe(10);
    expect(v.counts).toEqual({ open: 1, doing: 1, review: 2, done: 2 });
    expect(v.board).toEqual({ total: 4, inProgress: 1, blocked: 1, done: 1 });
    expect(v.caption).toBe('6 on the local kanban · 4 on the board');
  });

  test('dot chips carry the kanban lanes in status colors, zero lanes left out', () => {
    expect(v.chips).toEqual([
      { text: '1 to do' },
      { tone: 'warn', text: '1 in progress' },
      { tone: 'accent', text: '2 in review' },
      { tone: 'ok', text: '2 done' },
    ]);
  });

  test('four meters, each a real fraction of its own whole', () => {
    expect(v.meters.map((m) => m.label)).toEqual([
      'Kanban shipped (2/6)',
      'Waiting on review (2/4 in flight)',
      'Board issues in progress (1/4)',
      'Crons healthy (1/3)',
    ]);
    expect(v.meters[0].frac).toBeCloseTo(2 / 6);
    expect(v.meters[0]).toMatchObject({ display: '33%', hue: 'var(--ok)' });
    expect(v.meters[1]).toMatchObject({ frac: 0.5, display: '50%', hue: 'var(--accent)' });
    expect(v.meters[2]).toMatchObject({ frac: 0.25, display: '25%', hue: 'var(--warn)' });
    expect(v.meters[3].frac).toBeCloseTo(1 / 3);
    expect(v.meters[3]).toMatchObject({ display: '33%', hue: 'var(--warn)' });
    expect(v.foot).toBe('4 crons · 11 runs recorded · 3 agents on the kanban');
  });

  test('the step line is kanban + board touches per day, oldest first, ending today', () => {
    expect(v.series).toHaveLength(14);
    expect(v.series[0].label).toBe('Sep 11');
    // today: 4 local tasks (open, review, 2 done); yesterday: 1 task + 3 issues
    expect(v.series.at(-1)).toEqual({ label: 'Sep 24', count: 4 });
    expect(v.series.at(-2)).toEqual({ label: 'Sep 23', count: 4 });
    expect(v.touchesInWindow).toBe(8);
  });

  test('cron runs in the window by weekday, Monday first', () => {
    expect(v.cron.runsInWindow).toBe(3);
    expect(v.cron.failedInWindow).toBe(1);
    expect(v.cron.rhythm.map((r) => r.label)).toEqual(['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']);
    expect(v.cron.rhythm.find((r) => r.label === 'Wed')?.count).toBe(1);
    expect(v.cron.rhythm.find((r) => r.label === 'Thu')?.count).toBe(2);
  });

  test('owners: unfinished kanban work per agent, busiest first, short labels', () => {
    expect(v.owners).toEqual([
      { label: 'Conductor', count: 2 },
      { label: 'Closer', count: 2 },
    ]);
    expect(v.busiestOwner).toEqual({ name: 'Conductor Prime', count: 2 });
  });

  test('the one insight card is what needs Alex: review, blocked, late or failing crons', () => {
    expect(v.insight.value).toBe(5);
    expect(v.insight.headline).toBe('2 in review · 1 blocked · 2 crons late or failing.');
    expect(v.insight.body).toBe('Late heartbeat · Broken ingest');
    // whole: 4 unfinished tasks + 3 unfinished issues + 3 enabled crons
    expect(v.insight.frac).toBeCloseTo(5 / 10);
  });
});

describe('tasksVolume with nothing to show', () => {
  const e = tasksVolume({ tasks: [], issues: [], jobs: [], runs: [], agentNames: {}, now: NOW });

  test('empty meters and honest copy, never a fake fill', () => {
    expect(e.headline).toBe(0);
    expect(e.chips).toEqual([]);
    expect(e.meters.every((m) => m.frac === 0)).toBe(true);
    expect(e.meters.map((m) => m.display)).toEqual(['no tasks yet', 'nothing in flight', 'board empty or offline', 'none enabled']);
    expect(e.series.every((s) => s.count === 0)).toBe(true);
    expect(e.owners).toEqual([]);
    expect(e.busiestOwner).toBeNull();
    expect(e.insight).toMatchObject({ value: 0, frac: 0, headline: 'Nothing is waiting on you.', body: 'No tasks, issues or crons yet.' });
  });

  test('unknown agents fall back to their id, and all-healthy crons read green', () => {
    const g = tasksVolume({ tasks: [task('ghost-agent', 'open')], issues: [], jobs: [job({ runs: 1, ok: 1 })], runs: [], agentNames: {}, now: NOW });
    expect(g.owners).toEqual([{ label: 'ghost-agent', count: 1 }]);
    expect(g.meters[3]).toMatchObject({ frac: 1, hue: 'var(--ok)' });
    expect(g.insight.body).toBe('1 task open, nothing waiting on you.');
  });
});
