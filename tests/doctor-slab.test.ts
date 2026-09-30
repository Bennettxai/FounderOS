import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import path from 'node:path';

/**
 * /doctor wears the Brand Deals slab (Alex, 2026-09-24: "rebuild the entire
 * OS in that light"): the floating slab, the 46px title, a hero row whose big
 * card keeps the pillar radar and whose right card is a "Health Volume"
 * count-up with the sweeping meters, a second row with the step line, the dot
 * matrix and exactly ONE gradient insight card, then the readouts as cards.
 * Every number comes from lib/doctor-volume, and every old feature stays.
 */
const read = (rel: string) => readFileSync(path.join(process.cwd(), rel), 'utf8');
const src = read('app/doctor/page.tsx');

describe('/doctor in the Brand Deals look', () => {
  test('composes the slab kit instead of the console header', () => {
    expect(src).toMatch(/from '@\/components\/slab'/);
    for (const piece of ['<Slab>', '<SlabTitle', '<SlabCard', '<BigStat', '<MeterStack', '<InsightCard']) expect(src, piece).toContain(piece);
    expect(src).not.toContain('<PageHeader');
  });

  test('the hero row is the 2fr/1fr split: the radar beside the Health Volume card', () => {
    expect(src).toContain('grid-cols-[2fr_1fr]');
    expect(src).toContain('title="Health Volume"');
    const hero = src.slice(src.indexOf('grid-cols-[2fr_1fr]'), src.indexOf('title="Health Volume"'));
    expect(hero).toContain('<PillarRadar');
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
    expect(src).toContain("from '@/lib/doctor-volume'");
    expect(src).toContain('doctorVolume(');
    expect(src).toContain('doctorLayers(');
  });

  test('stagger indices are distinct, no raw hex, no transition-all', () => {
    const idx = [...src.matchAll(/<(?:Rise|SlabCard|InsightCard)[^>]*\bi=\{(\d+)\}/g)].map((m) => Number(m[1]));
    expect(idx.length).toBeGreaterThan(5);
    expect(new Set(idx).size).toBe(idx.length);
    expect(src).not.toMatch(/#[0-9a-f]{6}\b/i);
    expect(src).not.toMatch(/transition-(colors|all)\b/);
  });

  test('storage layer rows carry a rounded status pill', () => {
    const box = src.slice(src.indexOf('{/* Core status'), src.indexOf('{/* The pipeline'));
    expect(box).toContain('rounded-full');
    expect(box).toContain('pillStyle(');
  });
});

describe('nothing was removed in the rework', () => {
  test('rerun, run provider, checks, both visuals, pipeline and query path stay', () => {
    for (const k of ['<DoctorRunProvider', '<DoctorRerun', '<DoctorChecks', '<BrainCore', '<PillarRadar', 'Storage layers', 'Pipeline', 'Query path', 'Fallback: local grep']) {
      expect(src, k).toContain(k);
    }
  });

  test('the page stays a server component', () => {
    expect(src.startsWith("'use client'")).toBe(false);
  });
});
