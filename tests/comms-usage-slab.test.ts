import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import path from 'node:path';

/**
 * /comms and /usage in the Brand Deals look (Alex, 2026-09-24: "the way it
 * opens, the way the animations happen, the way the bars are"). Both pages
 * float on the shared slab, open with the kit's title row, and show their
 * numbers in a Volume card of sweeping meters fed by a pure view-model.
 */
const read = (rel: string) => readFileSync(path.join(process.cwd(), rel), 'utf8');

describe('/comms: the Brand Deals slab', () => {
  const page = read('app/comms/page.tsx');

  test('floats on the kit slab with the kit title row, not the console PageHeader', () => {
    expect(page).toMatch(/from '@\/components\/slab'/);
    expect(page).toContain('<Slab');
    expect(page).toContain('<SlabTitle');
    expect(page).not.toContain('PageHeader');
  });

  test('a Message Volume card: the 50px headline over the sweeping meters, fed by commsVolume', () => {
    expect(page).toContain('Message Volume');
    expect(page).toContain('<BigStat');
    expect(page).toContain('<MeterStack');
    expect(page).toMatch(/from '@\/lib\/comms-volume'/);
  });

  test('the second row: a step line, a dot matrix and exactly one insight card', () => {
    expect(page).toContain('<StepLine');
    expect(page).toContain('<DotMatrix');
    expect((page.match(/<InsightCard/g) ?? []).length).toBe(1);
  });

  test('keeps every existing surface: sources, the morning report, the tabs', () => {
    for (const piece of ['<CommsDigestPanel', '<CommsTabs', 'sources.map']) expect(page).toContain(piece);
  });
});

describe('/usage: the Brand Deals slab', () => {
  const page = read('app/usage/page.tsx');
  const board = read('components/UsageBoard.tsx');

  test('the page floats on the kit slab and opens with the kit title row', () => {
    expect(page).toMatch(/from '@\/components\/slab'/);
    expect(page).toContain('<Slab');
    expect(page).toContain('<SlabTitle');
    expect(page).not.toContain('PageHeader');
  });

  test('the board carries a Burn Volume card of sweeping meters, fed by usageVolume', () => {
    expect(board).toMatch(/from '@\/components\/slab'/);
    expect(board).toContain('Burn Volume');
    expect(board).toContain('<BigStat');
    expect(board).toContain('<MeterStack');
    expect(board).toMatch(/from '@\/lib\/usage-volume'/);
    expect(board).toContain('<StepLine');
    expect(board).toContain('<DotMatrix');
    expect((board.match(/<InsightCard/g) ?? []).length).toBe(1);
  });

  test('the plan gauges are the Brand Deals bars now, not square console bars', () => {
    expect(board).toContain('<VolumeMeter');
    expect(board).not.toMatch(/function OfficialBar/);
  });

  test('the window filter wears the list-head filter pills', () => {
    expect(board).toContain('chipClass(');
  });

  test('stagger indices are distinct across the board', () => {
    const idx = [...board.matchAll(/<(?:Rise|SlabCard|InsightCard)[^>]*\bi=\{(\d+)\}/g)].map((m) => Number(m[1]));
    expect(idx.length).toBeGreaterThan(3);
    expect(new Set(idx).size).toBe(idx.length);
  });
});
