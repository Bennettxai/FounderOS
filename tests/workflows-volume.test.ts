import { describe, expect, test } from 'vitest';
import { workflowsVolume, type VolumeJob } from '@/lib/workflows-volume';
import type { Workflow } from '@/lib/schemas';

/**
 * /workflows in the Brand Deals look (Alex, 2026-09-24: "workflows, social,
 * trading, and finances could all be very similar"). The Cron Volume card, the
 * run-activity step line, the weekday rhythm and the one "Needs you" card all
 * come from this pure view-model, fed with the page's own rows: the scheduled
 * jobs, the workflows table and the real cron_runs history. Nothing invented.
 */
const NOW = new Date('2026-09-24T18:00:00');

const job = (over: Partial<VolumeJob> = {}): VolumeJob => ({
  description: 'Stack monitor',
  enabled: true,
  unknownAgent: false,
  overdue: false,
  lastOk: true,
  runs: 0,
  ok: 0,
  ...over,
});

const step = (over: Partial<Workflow['steps'][number]> = {}): Workflow['steps'][number] => ({
  id: 's',
  title: 't',
  detail: '',
  ownerKind: 'human',
  owner: 'Alex',
  hoursPerWeek: 0,
  tools: [],
  edgeLabel: null,
  leakUsd: null,
  automation: null,
  branch: null,
  ...over,
});

const WORKFLOWS: Workflow[] = [
  {
    id: 'w1',
    name: 'Vantage sales machine',
    subtitle: '',
    revenueUsd: 0,
    order: 0,
    steps: [
      step({ id: 'a', ownerKind: 'human', hoursPerWeek: 6, tools: ['gmail'], leakUsd: 1000 }),
      step({ id: 'b', ownerKind: 'agent', hoursPerWeek: 4, tools: ['gmail', 'slack'], automation: { title: 'x', state: 'live', recoveredUsd: 250 } }),
    ],
  },
  {
    id: 'w2',
    name: 'Vantage delivery',
    subtitle: '',
    revenueUsd: 0,
    order: 1,
    steps: [step({ id: 'c', ownerKind: 'human', hoursPerWeek: 10, tools: ['notion'], automation: { title: 'y', state: 'suggested', recoveredUsd: 900 } })],
  },
];

const at = (iso: string, ok = true) => ({ startedAt: iso, ok });

describe('workflowsVolume', () => {
  const jobs = [
    job({ description: 'Healthy one', runs: 5, ok: 5 }),
    job({ description: 'Healthy two', runs: 3, ok: 2 }),
    job({ description: 'Late heartbeat', overdue: true, runs: 2, ok: 2 }),
    job({ description: 'Broken ingest', lastOk: false, runs: 4, ok: 1 }),
    job({ description: 'Ghost agent', unknownAgent: true }),
    job({ description: 'Paused digest', enabled: false, runs: 1, ok: 1 }),
  ];
  const v = workflowsVolume({
    jobs,
    workflows: WORKFLOWS,
    runs: [
      at('2026-09-24T09:00:00'),
      at('2026-09-24T10:00:00', false),
      at('2026-09-23T09:00:00'),
      at('2026-09-10T09:00:00'), // outside a 14-day window
    ],
    now: NOW,
    days: 14,
  });

  test('crons fall into exactly one state, matching the Scheduled tasks panel', () => {
    expect(v.headline).toBe(6);
    expect(v.counts).toEqual({ healthy: 2, overdue: 1, failing: 2, paused: 1, enabled: 5 });
  });

  test('dot chips carry the states in status colors, zero states left out', () => {
    expect(v.chips).toEqual([
      { tone: 'ok', text: '2 healthy' },
      { tone: 'warn', text: '1 overdue' },
      { tone: 'err', text: '2 failing' },
      { text: '1 paused' },
    ]);
    expect(v.caption).toBe('5 enabled · 15 runs recorded');
  });

  test('four meters, each a real fraction of its own whole', () => {
    expect(v.meters.map((m) => m.label)).toEqual([
      'Crons healthy (2/5)',
      'Runs OK (11/15)',
      'Hours carried by agents (4/20h per wk)',
      'Leak recovered ($250/$1,000 mo)',
    ]);
    expect(v.meters[0]).toMatchObject({ frac: 0.4, display: '40%', hue: 'var(--warn)' });
    expect(v.meters[1]).toMatchObject({ display: '11 ok · 4 failed' });
    expect(v.meters[1].frac).toBeCloseTo(11 / 15);
    expect(v.meters[2]).toMatchObject({ frac: 0.2, display: '20%' });
    expect(v.meters[3]).toMatchObject({ frac: 0.25, display: '25%' });
    expect(v.foot).toBe('2 workflows · 3 steps · 3 tools');
  });

  test('the step line is one point per day, oldest first, ending today', () => {
    expect(v.series).toHaveLength(14);
    expect(v.series.at(-1)).toEqual({ label: 'Sep 24', count: 2 });
    expect(v.series.at(-2)).toEqual({ label: 'Sep 23', count: 1 });
    expect(v.series[0].label).toBe('Sep 11');
    expect(v.runsInWindow).toBe(3);
    expect(v.failedInWindow).toBe(1);
  });

  test('the weekday rhythm counts runs in the window, Monday first', () => {
    expect(v.rhythm.map((r) => r.label)).toEqual(['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']);
    // Sep 23 2026 is a Wednesday, Sep 24 a Thursday
    expect(v.rhythm.find((r) => r.label === 'Wed')?.count).toBe(1);
    expect(v.rhythm.find((r) => r.label === 'Thu')?.count).toBe(2);
  });

  test('process load: human hours per workflow, with unique short labels', () => {
    expect(v.load.manualHours).toBe(16);
    expect(v.load.agentHours).toBe(4);
    expect(v.load.perWorkflow).toEqual([
      { label: 'Vantage', count: 6 },
      { label: 'Vantage 2', count: 10 },
    ]);
  });

  test('the one insight card is what needs Alex: overdue plus failing', () => {
    expect(v.insight.value).toBe(3);
    expect(v.insight.headline).toBe('1 overdue · 2 failing.');
    expect(v.insight.body).toBe('Late heartbeat · Broken ingest · Ghost agent');
    expect(v.insight.frac).toBeCloseTo(3 / 5);
  });
});

describe('workflowsVolume with nothing to show', () => {
  const e = workflowsVolume({ jobs: [], workflows: [], runs: [], now: NOW });

  test('empty meters and honest copy, never a fake fill', () => {
    expect(e.headline).toBe(0);
    expect(e.chips).toEqual([]);
    expect(e.meters.every((m) => m.frac === 0)).toBe(true);
    expect(e.meters.map((m) => m.display)).toEqual(['none enabled', 'no runs yet', 'no steps mapped', 'no leaks mapped']);
    expect(e.series.every((s) => s.count === 0)).toBe(true);
    expect(e.insight.value).toBe(0);
    expect(e.insight.frac).toBe(0);
    expect(e.insight.body).toBe('No scheduled tasks yet.');
  });

  test('all green reads as all green', () => {
    const g = workflowsVolume({ jobs: [job({ runs: 2, ok: 2 })], workflows: [], runs: [], now: NOW });
    expect(g.meters[0]).toMatchObject({ frac: 1, hue: 'var(--ok)' });
    expect(g.insight.headline).toBe('Every enabled task ran on its slot.');
  });
});
