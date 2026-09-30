import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { analyticsVolume } from '@/lib/analytics-volume';

/**
 * /analytics in the Brand Deals look (Alex, 2026-09-24). The page's own
 * numbers, shaped like Deal Volume: reach up top with a meter per channel,
 * the run log split by agent, a weekday rhythm for the dot matrix, and the
 * one insight card asking for the credentials still missing.
 */
const run = (agentId: string, ok: boolean, day: string) => ({ agentId, ok, startedAt: `${day}T10:00:00Z` });

const BASE = {
  channels: [
    { key: 'instagram', label: 'Instagram', followers: 6000 },
    { key: 'youtube', label: 'YouTube', followers: 2000 },
    { key: 'tiktok', label: 'TikTok', followers: null },
  ],
  subs: 2000,
  growth7d: 2.5,
  live: 6,
  pending: [
    { label: 'Stripe MRR', source: 'pending creds' },
    { label: 'Typeform leads', source: 'pending creds' },
  ],
  runs: [
    run('scout', true, '2026-09-21'), // Monday
    run('scout', true, '2026-09-21'),
    run('scout', false, '2026-09-22'),
    run('writer', true, '2026-09-23'),
    run('ledger', true, '2026-09-24'),
    run('ghost', true, '2026-09-24'),
    run('scout', true, '2026-07-01'), // outside the 30-day rhythm, still in the log
  ],
  agentNames: { scout: 'Scout', writer: 'Writer', ledger: 'Ledger', ghost: 'Ghost' },
  today: '2026-09-24',
};

describe('analyticsVolume', () => {
  const v = analyticsVolume(BASE);

  test('the headline is total reach: every channel plus the email list', () => {
    expect(v.reach).toBe(10000);
    expect(v.headline).toBe('10,000');
    expect(v.chips).toEqual([
      { tone: 'ok', text: '+2.5% 7d' },
      { text: '2,000 email subs' },
    ]);
    expect(v.caption).toBe('total reach across 3 channels');
    expect(v.foot).toBe('reach = followers + email list · 6 of 8 metrics live');
  });

  test('one meter per channel, biggest first, each its real share of reach', () => {
    expect(v.meters.map((m) => m.label)).toEqual(['Instagram', 'YouTube', 'Email list']);
    expect(v.meters[0]).toMatchObject({ frac: 0.6, display: '6,000 · 60%' });
    expect(v.meters[2].frac).toBeCloseTo(0.2);
  });

  test('more than four channels fold the tail into one honest Other meter', () => {
    const many = analyticsVolume({
      ...BASE,
      subs: null,
      channels: [1, 2, 3, 4, 5].map((n) => ({ key: `c${n}`, label: `C${n}`, followers: n * 100 })),
    });
    expect(many.meters.map((m) => m.label)).toEqual(['C5', 'C4', 'C3', 'Other channels (2)']);
    expect(many.meters[3].frac).toBeCloseTo(300 / 1500);
  });

  test('runs split by agent: success rate plus a share meter per agent', () => {
    expect(v.runs).toMatchObject({ total: 7, ok: 6, failed: 1, okPct: 86, agents: 4 });
    expect(v.runs.meters.map((m) => m.label)).toEqual(['Scout (4)', 'Writer (1)', 'Ledger (1)', 'Ghost (1)']);
    expect(v.runs.meters[0].frac).toBeCloseTo(4 / 7);
  });

  test('the rhythm counts runs by weekday over the trailing 30 days, Monday first', () => {
    expect(v.rhythm.map((c) => c.label)).toEqual(['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']);
    expect(v.rhythm.map((c) => c.count)).toEqual([2, 1, 1, 2, 0, 0, 0]);
    expect(v.rhythmTotal).toBe(6);
  });

  test('the insight asks for the missing credentials and lights the live share', () => {
    expect(v.insight).toMatchObject({
      value: 2,
      headline: '2 metrics waiting on credentials',
      body: 'Stripe MRR · Typeform leads',
    });
    expect(v.insight.frac).toBeCloseTo(6 / 8);
  });

  test('no data reads as empty meters and honest copy, never invented numbers', () => {
    const e = analyticsVolume({ ...BASE, channels: [], subs: null, growth7d: null, live: 0, pending: [], runs: [] });
    expect(e.reach).toBe(0);
    expect(e.chips).toEqual([]);
    expect(e.meters).toEqual([]);
    expect(e.runs).toMatchObject({ total: 0, okPct: null, meters: [] });
    expect(e.rhythmTotal).toBe(0);
    expect(e.insight).toMatchObject({ value: 0, frac: 0, headline: 'Every metric is live' });
  });

  test('a shrinking audience wears the error dot', () => {
    expect(analyticsVolume({ ...BASE, growth7d: -1.25 }).chips[0]).toEqual({ tone: 'err', text: '-1.3% 7d' });
  });
});

describe('/analytics wears the slab kit', () => {
  const page = readFileSync(path.join(process.cwd(), 'app/analytics/page.tsx'), 'utf8');

  test('the page floats on the slab with the 46px title instead of the old header', () => {
    expect(page).toMatch(/from '@\/components\/slab'/);
    expect(page).toContain('<Slab');
    expect(page).toContain('<SlabTitle');
    expect(page).not.toContain('PageHeader');
  });

  test('a Brand Deals hero row: the run chart beside a Volume card of sweeping meters', () => {
    expect(page).toContain('grid-cols-[2fr_1fr]');
    expect(page).toContain('analyticsVolume(');
    expect(page).toContain('<BigStat');
    expect(page).toContain('<MeterStack');
    expect(page).toContain('<RunVolumeCard');
  });

  test('a second row with the dot matrix and exactly one gradient insight card', () => {
    expect(page).toContain('<DotMatrix');
    expect((page.match(/<InsightCard/g) ?? []).length).toBe(1);
  });

  test('retains the demo social analytics and run history', () => {
    expect(page).toContain('<RunVolumeCard');
    expect(page).toContain('PLATFORM_LABELS');
  });});
