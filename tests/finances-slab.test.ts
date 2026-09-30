import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, test } from 'vitest';

const read = (p: string) => readFileSync(join(process.cwd(), p), 'utf8');

/**
 * /finances wears the Brand Deals slab (Alex, 2026-09-24: "workflows,
 * social, trading, and finances could all be very similar to how this is").
 * Built from the shared kit, not restyled by hand, with every piece of the old
 * page still on it: the uploader, the expense panel and its report, the
 * business charts, the processors, recent income and Wise.
 */
describe('/finances on the slab kit', () => {
  const page = read('app/finances/page.tsx');

  test('floats as one slab with the kit title row instead of the old header', () => {
    expect(page).not.toMatch(/^'use client';/);
    expect(page).toMatch(/from '@\/components\/slab'/);
    expect(page).toContain('<Slab>');
    expect(page).toContain('<SlabTitle');
    expect(page).toContain('title="Finances"');
    expect(page).not.toContain('<PageHeader');
  });

  test('the hero row is the expense chart beside a Money Volume card fed by the view-model', () => {
    expect(page).toMatch(/grid-cols-\[2fr_1fr\]/);
    expect(page).toContain('title="Money Volume"');
    expect(page).toMatch(/moneyVolume\(/);
    expect(page).toContain('<BigStat');
    expect(page).toContain('<MeterStack');
    expect(page.indexOf('<MonthlyExpenses')).toBeLessThan(page.indexOf('title="Money Volume"'));
  });

  test('a second row: spend step line, charge-size dots, and exactly one insight card', () => {
    expect(page).toMatch(/from '@\/components\/slab-charts'/);
    expect(page).toContain('<StepLine');
    expect(page).toContain('<DotMatrix');
    expect((page.match(/<InsightCard/g) ?? []).length).toBe(1);
    expect(page).toMatch(/spendSeries\(/);
    expect(page).toMatch(/chargeSizes\(/);
  });

  test('keeps every existing piece of the page', () => {
    for (const s of ['<StatementUploader', '<MonthlyExpenses', '<BusinessIncomeChart', 'Income · by processor', 'Recent income', 'Outgoing · Wise']) {
      expect(page, s).toContain(s);
    }
  });

  test('card stagger indices are distinct', () => {
    const idx = [...page.matchAll(/<(?:Rise|SlabCard|InsightCard)[^>]*\bi=\{(\d+)\}/g)].map((m) => Number(m[1]));
    expect(idx.length).toBeGreaterThanOrEqual(6);
    expect(new Set(idx).size).toBe(idx.length);
  });

  test('house rules: no raw hex, no transition-all/colors', () => {
    expect(page).not.toMatch(/#[0-9a-fA-F]{3,6}\b/);
    expect(page).not.toMatch(/transition-(colors|all)\b/);
  });
});

describe('the expense panel inside the slab', () => {
  const panel = read('components/MonthlyExpenses.tsx');

  test('uses the kit: slab card head, filter-pill month chips, Deal Volume meters for categories', () => {
    expect(panel).toContain('text-[19px]');
    expect(panel).toMatch(/chipClass\(/);
    expect(panel).toContain('<VolumeMeter');
    expect(panel).not.toContain('SectionHead');
  });
});
