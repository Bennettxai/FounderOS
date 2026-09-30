import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { operatingVolume } from '@/lib/pulse-history';

/**
 * Home borrows Brand Deals' Deal Volume card as pure aesthetics (Alex,
 * 2026-09-24: "I like the way the deal volume bars go out"): the same glowing
 * hatched meters, fed with Home's own numbers, never deal data.
 */
const NOW = new Date('2026-09-24T18:00:00');
const at = (h: number) => new Date(new Date('2026-09-24T00:00:00').getTime() + h * 3600_000).toISOString();

describe('operatingVolume', () => {
  const v = operatingVolume({
    connected: 19,
    totalConnections: 24,
    activeAgents: 19,
    totalAgents: 32,
    health: 95,
    runs: [
      { agentId: 'a', ok: true, finishedAt: at(9) },
      { agentId: 'a', ok: true, finishedAt: at(10) },
      { agentId: 'b', ok: false, finishedAt: at(11) },
      { agentId: 'c', ok: true, finishedAt: '2026-09-23T12:00:00' }, // yesterday
    ],
    now: NOW,
  });

  test('the headline is today: runs, failures, and how many agents ran', () => {
    expect(v.runsToday).toBe(3);
    expect(v.failedToday).toBe(1);
    expect(v.agentsToday).toBe(2);
  });

  test('four meters, each a real fraction of its own whole', () => {
    expect(v.meters.map((m) => m.label)).toEqual(['Systems connected (19/24)', 'Agents live (19/32)', 'Runs OK today (2/3)', 'G-Brain health']);
    expect(v.meters[0].frac).toBeCloseTo(19 / 24);
    expect(v.meters[2].frac).toBeCloseTo(2 / 3);
    expect(v.meters[3]).toMatchObject({ frac: 0.95, display: '95 / 100' });
  });

  test('no runs and no brain read as empty meters, not fake fills', () => {
    const e = operatingVolume({ connected: 0, totalConnections: 0, activeAgents: 0, totalAgents: 0, health: null, runs: [], now: NOW });
    expect(e.meters.every((m) => m.frac === 0)).toBe(true);
    expect(e.meters[3].display).toBe('offline');
  });
});

describe('one meter, two pages', () => {
  const read = (rel: string) => readFileSync(path.join(process.cwd(), rel), 'utf8');
  test('Brand Deals and Home draw the same VolumeMeter', () => {
    expect(read('components/brand-deals/DealBoard.tsx')).toContain("from '@/components/VolumeMeter'");
    // Home draws it through the kit's MeterStack, which renders VolumeMeter.
    expect(read('app/page.tsx')).toContain('<MeterStack');
    expect(read('components/slab.tsx')).toContain('<VolumeMeter');
    expect(read('app/page.tsx')).toContain('Operating volume');
  });
  test('the sweep-out keyframe is global so the bars grow on Home too', () => {
    expect(read('app/globals.css')).toMatch(/@keyframes vol-meter-in/);
  });
});

describe('Home second row: the Brand Deals activity line, mix dots and insight', () => {
  test('dailySeries labels each of the last N local days and counts into them, quiet days kept', async () => {
    const { dailySeries } = await import('@/lib/pulse-history');
    const s = dailySeries([at(9), at(10), '2026-09-22T12:00:00', 'garbage', undefined], 3, NOW);
    expect(s).toEqual([
      { label: 'Sep 22', count: 1 },
      { label: 'Sep 23', count: 0 },
      { label: 'Sep 24', count: 2 },
    ]);
  });

  test('sourceMix counts inbound per source in a fixed order, zeros kept', async () => {
    const { sourceMix } = await import('@/lib/pulse-history');
    expect(sourceMix([{ source: 'email' }, { source: 'slack' }, { source: 'email' }])).toEqual([
      { label: 'email', count: 2 },
      { label: 'whatsapp', count: 0 },
      { label: 'slack', count: 1 },
    ]);
  });

  test('homeAttention: what is waiting on Alex, as a share of today\'s work', async () => {
    const { homeAttention } = await import('@/lib/pulse-history');
    const a = homeAttention({ inbound: 3, failedToday: 1, connectorsDown: 2, doneToday: 6 });
    expect(a.count).toBe(6);
    expect(a.headline).toBe('3 inbound · 1 failed run · 2 connectors down');
    expect(a.frac).toBeCloseTo(6 / 12);
    const calm = homeAttention({ inbound: 0, failedToday: 0, connectorsDown: 0, doneToday: 4 });
    expect(calm.count).toBe(0);
    expect(calm.frac).toBe(0);
    expect(calm.headline).toBe('Nothing is waiting on you.');
  });
});

describe('Home counts what got done, not what fits on the ledger (review 2026-09-24)', () => {
  const page = readFileSync(path.join(process.cwd(), 'app/page.tsx'), 'utf8');
  test('the done count is runs OK today plus charges today, taken before the ledger is trimmed', () => {
    expect(page).toMatch(/const doneCount = vol\.runsToday - vol\.failedToday \+ doneCharges\.length/);
    expect(page).toContain('doneToday: doneCount');
    expect(page).not.toContain('doneToday: doneToday.length');
    expect(page).not.toMatch(/\$\{doneToday\.length\} thing/);
  });
});
