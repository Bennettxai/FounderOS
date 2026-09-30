import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import path from 'node:path';

/**
 * /agents and /integrations wear the Brand Deals slab (Alex, 2026-09-24:
 * "rebuild the entire OS in that light"): the floating slab, the 46px title
 * row, a hero row whose right card is a "<Thing> Volume" count-up over the
 * sweeping meters, a second row of dot matrices beside exactly ONE gradient
 * insight card, and the existing panels in slab cards below. Every number
 * comes from a tested view-model and nothing that worked before is dropped.
 */
const read = (rel: string) => readFileSync(path.join(process.cwd(), rel), 'utf8');
const indices = (src: string) => [...src.matchAll(/<(?:Rise|SlabCard|InsightCard)[^>]*\bi=\{(\d+)\}/g)].map((m) => Number(m[1]));

describe('/agents in the Brand Deals look', () => {
  const page = read('app/agents/page.tsx');
  const panel = read('components/AgentsVolumePanel.tsx');

  test('the page floats on the kit slab with the kit title row, not the console header', () => {
    expect(page).toMatch(/from '@\/components\/slab'/);
    for (const piece of ['<Slab>', '<SlabTitle', '<SlabCard']) expect(page, piece).toContain(piece);
    expect(page).not.toContain('<PageHeader');
  });

  test('the hero + second row live in a client panel fed by agentsVolume', () => {
    expect(page).toContain('<AgentsVolumePanel');
    expect(panel).toContain("'use client'");
    expect(panel).toMatch(/from '@\/components\/slab'/);
    expect(panel).toContain('grid-cols-[2fr_1fr]');
    expect(panel).toContain('title="Agent Volume"');
    for (const piece of ['<BigStat', '<MeterStack', '<StepLine', '<DotMatrix']) expect(panel, piece).toContain(piece);
    expect((panel.match(/<InsightCard/g) ?? []).length).toBe(1);
    expect(panel).toMatch(/from '@\/lib\/agents-volume'/);
    expect(panel).toContain('agentsVolume(');
  });

  test('the hero moves with the board: BoardLive publishes each poll, the panel subscribes', () => {
    const board = read('components/BoardLive.tsx');
    expect(board).toContain('publishBoard(data)');
    expect(panel).toContain('useBoardSnapshot(');
  });

  test('keeps every existing surface: tabs, the live board, the Conductor rail, the cockpit height', () => {
    for (const piece of ['<AgentsTabs', '<BoardLive initial={boardInitial}', '<ConductorChat', 'HERMES_DASH_URL', 'xl:h-[calc(100dvh']) {
      expect(page, piece).toContain(piece);
    }
  });

  test('distinct stagger indices across page and panel, no raw hex, no transition-all', () => {
    const idx = [...indices(page), ...indices(panel)];
    expect(idx.length).toBeGreaterThan(4);
    expect(new Set(idx).size).toBe(idx.length);
    for (const src of [page, panel]) {
      expect(src).not.toMatch(/#[0-9a-f]{6}\b/i);
      expect(src).not.toMatch(/transition-(colors|all)\b/);
    }
  });
});

describe('/integrations in the Brand Deals look', () => {
  const page = read('app/integrations/page.tsx');
  const browser = read('components/IntegrationBrowser.tsx');

  test('composes the slab kit instead of the console header', () => {
    expect(page).toMatch(/from '@\/components\/slab'/);
    for (const piece of ['<Slab>', '<SlabTitle', '<SlabCard', '<BigStat', '<MeterStack', '<InsightCard']) expect(page, piece).toContain(piece);
    expect(page).not.toContain('<PageHeader');
  });

  test('the hero row is the 2fr/1fr split, its right card the Connection Volume card', () => {
    expect(page).toContain('grid-cols-[2fr_1fr]');
    expect(page).toContain('title="Connection Volume"');
  });

  test('the second row carries dot matrices and exactly one gradient insight card', () => {
    expect(page).toMatch(/from '@\/components\/slab-charts'/);
    expect(page).toContain('<DotMatrix');
    expect((page.match(/<InsightCard/g) ?? []).length).toBe(1);
  });

  test('every number flows through the tested view-model', () => {
    expect(page).toContain("from '@/lib/integrations-volume'");
    expect(page).toContain('integrationsVolume(');
  });

  test('categories filter through the kit pills, and every tile, flow and key row stays', () => {
    expect(page).toContain('<IntegrationBrowser');
    expect(browser).toContain("'use client'");
    expect(browser).toContain('chipClass(');
    expect(browser).toContain('<IntegrationCategory');
    for (const piece of ['<ConnectionCard', 'oauthReadiness', '<ApiKeys']) expect(page, piece).toContain(piece);
  });

  test('distinct stagger indices, no raw hex, no transition-all', () => {
    const idx = indices(page);
    expect(idx.length).toBeGreaterThan(4);
    expect(new Set(idx).size).toBe(idx.length);
    for (const src of [page, browser]) {
      expect(src).not.toMatch(/#[0-9a-f]{6}\b/i);
      expect(src).not.toMatch(/transition-(colors|all)\b/);
    }
  });
});
