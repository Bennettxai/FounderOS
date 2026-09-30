import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import path from 'node:path';

/**
 * /tasks and /skills wear the Brand Deals slab (Alex, 2026-09-24: "rebuild
 * the entire OS in that light"): the floating slab, the 46px title, a 2fr/1fr
 * hero whose right card is a "<Thing> Volume" count-up over the sweeping
 * meters, a second row of dot matrices and exactly ONE gradient insight card,
 * then the page's existing boards in slab cards. Every number comes from a
 * tested view-model and every old feature stays.
 */
const read = (rel: string) => readFileSync(path.join(process.cwd(), rel), 'utf8');
const stagger = (src: string) => [...src.matchAll(/<(?:Rise|SlabCard|InsightCard)[^>]*\bi=\{(\d+)\}/g)].map((m) => Number(m[1]));

const PAGES = [
  { file: 'app/tasks/page.tsx', volume: 'Task Volume', model: 'tasksVolume', lib: '@/lib/tasks-volume' },
  { file: 'app/skills/page.tsx', volume: 'Skill Volume', model: 'skillsVolume', lib: '@/lib/skills-volume' },
];

describe.each(PAGES)('$file in the Brand Deals look', ({ file, volume, model, lib }) => {
  const src = read(file);

  test('composes the slab kit instead of the console header', () => {
    expect(src).toMatch(/from '@\/components\/slab'/);
    for (const piece of ['<Slab>', '<SlabTitle', '<SlabCard', '<BigStat', '<MeterStack', '<InsightCard']) expect(src, piece).toContain(piece);
    expect(src).not.toContain('<PageHeader');
    expect(src).not.toContain('<h1');
  });

  test('the hero row is the Brand Deals 2fr/1fr split, its right card the volume card', () => {
    expect(src).toContain('grid-cols-[2fr_1fr]');
    expect(src).toContain(`title="${volume}"`);
  });

  test('the second row carries dot matrices and exactly one gradient insight card', () => {
    expect(src).toMatch(/from '@\/components\/slab-charts'/);
    expect(src).toContain('<DotMatrix');
    expect((src.match(/<InsightCard/g) ?? []).length).toBe(1);
  });

  test('every number flows through the tested view-model', () => {
    expect(src).toContain(`from '${lib}'`);
    expect(src).toContain(`${model}(`);
  });

  test('stagger indices are distinct, no raw hex, no transition-all', () => {
    const idx = stagger(src);
    expect(idx.length).toBeGreaterThan(3);
    expect(new Set(idx).size).toBe(idx.length);
    expect(src).not.toMatch(/#[0-9a-f]{6}\b/i);
    expect(src).not.toMatch(/transition-(colors|all)\b/);
  });

  test('stays a server page', () => {
    expect(src).not.toContain("'use client'");
  });
});

describe('/tasks keeps the cron strip, the board queue and the kanban, each in a slab card', () => {
  const page = read('app/tasks/page.tsx');
  const parts = ['components/TaskCronStrip.tsx', 'components/BoardTasks.tsx', 'components/TaskBoard.tsx'];

  test('the work-activity step line leads the hero', () => {
    expect(page).toContain('<StepLine');
  });

  test('all three sections still render, fed their real rows', () => {
    for (const k of ['<TaskCronStrip', 'crons={crons}', '<BoardTasks', 'initialIssues={issues}', '<TaskBoard', 'initialTasks={tasks}', 'paperclipIssues(']) expect(page, k).toContain(k);
  });

  test('each section is a SlabCard on a stagger after the page’s own', () => {
    const pageMax = Math.max(...stagger(page));
    const passed = [...page.matchAll(/<(?:TaskCronStrip|BoardTasks|TaskBoard)[^>]*\bi=\{(\d+)\}/g)].map((m) => Number(m[1]));
    expect(passed).toHaveLength(3);
    expect(Math.min(...passed)).toBeGreaterThan(pageMax);
    expect(new Set(passed).size).toBe(3);
    for (const f of parts) {
      const src = read(f);
      expect(src, f).toMatch(/from '@\/components\/slab'/);
      expect(src, f).toContain('<SlabCard');
      expect(src, f).not.toMatch(/transition-(colors|all)\b/);
      expect(src, f).not.toMatch(/#[0-9a-f]{6}\b/i);
    }
  });

  test('the board queue routes with the Brand Deals pills and rows carry rounded status pills', () => {
    const board = read('components/BoardTasks.tsx');
    expect(board).toContain('chipClass(');
    expect(board).toContain('rounded-full');
    expect(board.startsWith("'use client'")).toBe(true);
    expect(read('components/TaskBoard.tsx').startsWith("'use client'")).toBe(true);
  });
});

describe('/skills keeps the card wall, now in a slab card', () => {
  const page = read('app/skills/page.tsx');
  const grid = read('components/SkillsGrid.tsx');

  test('the wall renders every card with its source note', () => {
    expect(page).toContain('<SkillsGrid');
    expect(page).toMatch(/cards=\{cards\}/);
    expect(page).toContain('sourceNote={sourceNote}');
    expect(page).toMatch(/<SkillsGrid[^>]*\bi=\{(\d+)\}/);
  });

  test('the wall is a SlabCard whose filters are the Brand Deals chips', () => {
    expect(grid).toMatch(/from '@\/components\/slab'/);
    expect(grid).toContain('<SlabCard');
    expect(grid).toContain('chipClass(');
    expect(grid.startsWith("'use client'")).toBe(true);
  });

  test('the SKILL.md reader and download survive', () => {
    for (const k of ['/api/skills/', 'download=1', 'skill.md', "e.key === 'Escape'", 'placeholder="filter skills']) expect(grid, k).toContain(k);
  });
});
