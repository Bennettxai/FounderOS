import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import path from 'node:path';

/**
 * /workflows and /social wear the Brand Deals slab (Alex, 2026-09-24:
 * "workflows, social, trading, and finances could all be very similar"):
 * the floating slab, the 46px title, a hero row whose right card is a
 * "<Thing> Volume" count-up with the sweeping meters, a second row with the
 * step line, the dot matrix and exactly ONE gradient insight card. Every
 * number comes from a tested view-model, and every old feature stays.
 */
const read = (rel: string) => readFileSync(path.join(process.cwd(), rel), 'utf8');

const PAGES = [
  { file: 'app/workflows/page.tsx', volume: 'Cron Volume', model: 'workflowsVolume', lib: '@/lib/workflows-volume' },
  { file: 'app/social/page.tsx', volume: 'Audience Volume', model: 'socialVolume', lib: '@/lib/social-volume' },
];

describe.each(PAGES)('$file in the Brand Deals look', ({ file, volume, model, lib }) => {
  const src = read(file);

  test('composes the slab kit instead of the console header', () => {
    expect(src).toMatch(/from '@\/components\/slab'/);
    for (const piece of ['<Slab>', '<SlabTitle', '<SlabCard', '<BigStat', '<MeterStack', '<InsightCard']) expect(src, piece).toContain(piece);
    expect(src).not.toContain('<PageHeader');
  });

  test('the hero row is the Brand Deals 2fr/1fr split, its right card the volume card', () => {
    expect(src).toContain('grid-cols-[2fr_1fr]');
    expect(src).toContain(`title="${volume}"`);
  });

  test('the second row carries the step line and the dot matrix', () => {
    expect(src).toMatch(/from '@\/components\/slab-charts'/);
    expect(src).toContain('<StepLine');
    expect(src).toContain('<DotMatrix');
  });

  test('exactly one gradient insight card', () => {
    expect((src.match(/<InsightCard/g) ?? []).length).toBe(1);
  });

  test('every number flows through the tested view-model', () => {
    expect(src).toContain(`from '${lib}'`);
    expect(src).toContain(`${model}(`);
  });

  test('stagger indices are distinct, no raw hex, no transition-all', () => {
    const idx = [...src.matchAll(/<(?:Rise|SlabCard|InsightCard)[^>]*\bi=\{(\d+)\}/g)].map((m) => Number(m[1]));
    expect(idx.length).toBeGreaterThan(3);
    expect(new Set(idx).size).toBe(idx.length);
    expect(src).not.toMatch(/#[0-9a-f]{6}\b/i);
    expect(src).not.toMatch(/transition-(colors|all)\b/);
  });
});

describe('nothing was removed in the rework', () => {
  test('/workflows keeps the scheduled tasks panel and the process map tree', () => {
    const src = read('app/workflows/page.tsx');
    for (const k of ['<ScheduledTasks', '<WorkflowTree', 'Process map', 'runsByOwner', 'avatarByOwner', 'toolLogos']) expect(src, k).toContain(k);
  });

  test('/social keeps accounts, the stat strip, the charts, recent posts and the composer', () => {
    const src = read('app/social/page.tsx');
    for (const k of ['href={`/social/${p.platform}`}', 'href="/social/beehiiv"', '<SocialStatStrip', '<AudienceConsistencyLazy', '<AudiencePie', 'Recent posts', '<PostComposer']) {
      expect(src, k).toContain(k);
    }
  });
});
