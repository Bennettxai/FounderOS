import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { funnelVolume } from '@/lib/funnel-volume';
import type { FunnelJourney, FunnelStage } from '@/lib/schemas';

/**
 * /funnel below the graph in the Brand Deals look (Alex, 2026-09-24: "the
 * bottom part of the funnel, not the actual graph, but all the other stuff").
 * The Funnel Volume card is fed by the journeys the graph already draws:
 * closed revenue up top, one sweeping meter per stage hand-off underneath.
 */
const NOW = new Date('2026-09-24T12:00:00Z');

function journey(
  id: string,
  status: FunnelStage,
  touches: Array<[FunnelStage, 'organic' | 'ads' | 'call' | 'checkout', string]>,
  extra: Partial<FunnelJourney> = {},
): FunnelJourney {
  return {
    id,
    name: id,
    venture: 'vantage',
    status,
    product: null,
    amountUsd: null,
    relationship: 'warm',
    likelihood: 50,
    url: null,
    email: null,
    phone: null,
    person: null,
    company: null,
    role: null,
    linkedin: null,
    createdAt: touches[0]?.[2] ?? '2026-09-01',
    touches: touches.map(([stage, channel, at], i) => ({
      id: `${id}-${i}`,
      contactId: id,
      seq: i + 1,
      stage,
      channel,
      label: `${stage} touch`,
      source: 'manual',
      at,
    })),
    ...extra,
  };
}

const JOURNEYS: FunnelJourney[] = [
  journey('won-a', 'converted', [['first_touch', 'organic', '2026-09-01'], ['engaged', 'call', '2026-09-05'], ['nurtured', 'call', '2026-09-08'], ['converted', 'checkout', '2026-09-10']], { amountUsd: 3000 }),
  journey('won-b', 'converted', [['first_touch', 'ads', '2026-09-02'], ['engaged', 'call', '2026-09-06'], ['nurtured', 'call', '2026-09-09'], ['converted', 'checkout', '2026-09-20']], { amountUsd: 1500 }),
  journey('held', 'nurtured', [['first_touch', 'organic', '2026-09-03'], ['engaged', 'call', '2026-09-20'], ['nurtured', 'call', '2026-09-23']], { likelihood: 80, relationship: 'hot', person: 'Hot Holly' }),
  journey('booked', 'engaged', [['first_touch', 'organic', '2026-09-04'], ['engaged', 'call', '2026-09-10']], { person: 'Stalled Sam' }),
  journey('lead', 'first_touch', [['first_touch', 'ads', '2026-09-22']]),
];

describe('funnelVolume', () => {
  const v = funnelVolume({ journeys: JOURNEYS, archived: 2, now: NOW });

  test('the headline is closed revenue, with closed / stalled / archived chips', () => {
    expect(v.revenueUsd).toBe(4500);
    expect(v.chips).toEqual([
      { tone: 'ok', text: '2 closed' },
      { tone: 'err', text: '1 stalled' },
      { text: '2 archived' },
    ]);
    expect(v.caption).toBe('closed revenue across 5 active clients · 3 organic / 2 ads entry');
  });

  test('four meters, one per stage hand-off, each a real fraction of the stage before', () => {
    expect(v.meters.map((m) => m.label)).toEqual([
      'First touch → Engaged (4/5)',
      'Engaged → Nurtured (3/4)',
      'Nurtured → Opted in (2/3)',
      'Opted in → Converted (2/2)',
    ]);
    expect(v.meters[0].frac).toBeCloseTo(4 / 5);
    expect(v.meters[1].frac).toBeCloseTo(3 / 4);
    expect(v.meters[2].frac).toBeCloseTo(2 / 3);
    expect(v.meters.map((m) => m.display)).toEqual(['80%', '75%', '67%', '100%']);
    expect(v.foot).toBe('each bar is a stage over the one before · 40% end to end');
  });

  test('the activity series counts real touches per day over the last 30 days', () => {
    expect(v.series).toHaveLength(30);
    expect(v.series.at(-1)).toEqual({ label: 'Sep 24', count: 0 });
    expect(v.series.find((s) => s.label === 'Sep 10')?.count).toBe(2);
    expect(v.touchesInWindow).toBe(14);
  });

  test('the insight counts who needs Alex: push now + save now, never the closed', () => {
    expect(v.insight.value).toBe(2);
    expect(v.insight.headline).toBe('1 to push · 1 to save');
    expect(v.insight.body).toBe('Hot Holly · Stalled Sam');
    expect(v.insight.frac).toBeCloseTo(2 / 5);
  });

  test('an empty funnel reads as empty meters and honest copy, never invented numbers', () => {
    const e = funnelVolume({ journeys: [], archived: 0, now: NOW });
    expect(e.revenueUsd).toBe(0);
    expect(e.chips).toEqual([{ tone: 'ok', text: '0 closed' }]);
    expect(e.meters.every((m) => m.frac === 0)).toBe(true);
    expect(e.meters.map((m) => m.display)).toEqual(['no leads', 'no leads', 'no leads', 'no leads']);
    expect(e.foot).toBe('each bar is a stage over the one before · no leads yet');
    expect(e.touchesInWindow).toBe(0);
    expect(e.insight).toMatchObject({ value: 0, frac: 0, headline: 'Nothing waiting on you' });
  });
});

describe('/funnel wears the slab kit around an untouched graph', () => {
  const page = readFileSync(path.join(process.cwd(), 'app/funnel/page.tsx'), 'utf8');

  test('the page floats on the slab with the 46px title', () => {
    expect(page).toMatch(/from '@\/components\/slab'/);
    expect(page).toContain('<Slab');
    expect(page).toContain('<SlabTitle');
    expect(page).not.toContain('PageHeader');
  });

  test('the graph block is still the graph: both engines and the layout toggle render', () => {
    expect(page).toContain('<FunnelSpaceLazy');
    expect(page).toContain('<FunnelRadialLazy');
    expect(page).toContain('<FunnelLayoutToggle');
  });

  test('the listed information is slab cards: Funnel Volume meters, the step line, one insight', () => {
    expect(page).toContain('funnelVolume(');
    expect(page).toContain('<MeterStack');
    expect(page).toContain('<BigStat');
    expect(page).toContain('<StepLine');
    expect((page.match(/<InsightCard/g) ?? []).length).toBe(1);
  });

  test('journey segments filter with the Brand Deals pills', () => {
    expect(page).toContain('chipClass(');
  });
});
