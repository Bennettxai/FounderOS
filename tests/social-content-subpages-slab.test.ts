import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import path from 'node:path';

/**
 * The pages you land on when you click through from /social and /content
 * keep the Brand Deals slab (Alex, 2026-09-24: "rebuild the entire OS in
 * that light"): /social/[platform], /social/beehiiv and /content/lead-magnets.
 * Each wears the floating slab and the 46px title with a pill back to its
 * parent, a 2fr/1fr hero whose right card is a "<Thing> Volume" count-up
 * with the sweeping meters, a second row with a mini chart and exactly ONE
 * gradient insight card, and keeps everything it did before.
 */
const read = (rel: string) => readFileSync(path.join(process.cwd(), rel), 'utf8');

const PAGES = [
  {
    file: 'app/social/[platform]/page.tsx',
    volume: 'Follower Volume',
    model: 'platformVolume',
    lib: '@/lib/social-volume',
    back: 'href="/social"',
    chart: '<StepLine',
    keeps: ['<FollowerBarChart', '<GrowthBadge', 'account.url', 'notFound()', 'syncFromZernioConfig(db)'],
  },
  {
    file: 'app/social/beehiiv/page.tsx',
    volume: 'Newsletter Volume',
    model: 'newsletterVolume',
    lib: '@/lib/newsletter-volume',
    back: 'href="/social"',
    chart: '<DotMatrix',
    keeps: ['<NewsletterList newsletters={newsletters} />', 'https://app.beehiiv.com', 'beehiivSubscribers()'],
  },
  {
    file: 'app/content/lead-magnets/page.tsx',
    volume: 'Magnet Volume',
    model: 'leadMagnetVolume',
    lib: '@/lib/lead-magnet-volume',
    back: 'href="/content"',
    chart: '<DotMatrix',
    keeps: ['<NewLeadMagnet />', 'showCopy', 'manage', 'chipClass(', 'filterLeadMagnets('],
  },
];

describe.each(PAGES)('$file in the Brand Deals look', ({ file, volume, model, lib, back, chart, keeps }) => {
  const src = read(file);

  test('composes the slab kit instead of the console header', () => {
    expect(src).toMatch(/from '@\/components\/slab'/);
    for (const piece of ['<Slab>', '<SlabTitle', '<SlabCard', '<BigStat', '<MeterStack', '<InsightCard']) expect(src, piece).toContain(piece);
    expect(src).not.toContain('<PageHeader');
    expect(src).not.toContain('text-[25px]');
    expect(src).not.toContain('rounded-xl border');
  });

  test('a pill in the title leads back to the parent tab', () => {
    expect(src).toContain(back);
    expect(src).toMatch(/className=\{PILL\}/);
  });

  test('the hero row is the Brand Deals 2fr/1fr split, its right card the volume card', () => {
    expect(src).toContain('grid-cols-[2fr_1fr]');
    expect(src).toContain(`title="${volume}"`);
  });

  test('the second row carries a mini chart and exactly one gradient insight card', () => {
    expect(src).toMatch(/from '@\/components\/slab-charts'/);
    expect(src).toContain(chart);
    expect((src.match(/<InsightCard/g) ?? []).length).toBe(1);
  });

  test('every number flows through the tested view-model', () => {
    expect(src).toContain(`from '${lib}'`);
    expect(src).toContain(`${model}(`);
  });

  test('nothing the page did before is lost', () => {
    for (const k of keeps) expect(src, k).toContain(k);
  });

  test('server page, distinct stagger, no raw hex, no transition-all', () => {
    expect(src).not.toMatch(/^['"]use client['"]/m);
    const idx = [...src.matchAll(/<(?:Rise|SlabCard|InsightCard)[^>]*\bi=\{(\d+)\}/g)].map((m) => Number(m[1]));
    expect(idx.length).toBeGreaterThan(3);
    expect(new Set(idx).size).toBe(idx.length);
    expect(src).not.toMatch(/#[0-9a-f]{6}\b/i);
    expect(src).not.toMatch(/transition-(colors|all)\b/);
  });
});

describe('the lists inside the slab', () => {
  test('lead magnet rows are roomy with rounded status pills', () => {
    const list = read('components/LeadMagnets.tsx');
    expect(list).toContain('rounded-full');
    expect(list).toMatch(/py-4/);
    expect(list).not.toMatch(/transition-(colors|all)\b/);
  });

  test('the new lead magnet control is a slab pill', () => {
    const src = read('components/NewLeadMagnet.tsx');
    expect(src).toContain('PILL');
    for (const m of src.matchAll(/<button[^>]*className=\{?["'`]?([^"'`}]*)/g)) {
      expect(m[0]).toMatch(/pressable|PILL|chipClass/);
    }
  });

  test('newsletter rows are rounded, roomy and pressable', () => {
    const src = read('components/NewsletterList.tsx');
    expect(src).toContain('rounded-[10px]');
    expect(src).toContain('pressable');
    expect(src).toContain('rounded-full');
  });
});
