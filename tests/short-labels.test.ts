import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { shortLabels } from '@/lib/short-labels';

/**
 * Dot-matrix column labels across the slab pages: the first word of a name,
 * numbered only when two would collide, so two columns can never share a
 * label (or a React key). Review 2026-09-24: /agents kept first words with no
 * de-dupe, so "Sales Agent" and "Sales Calls Data" both read "Sales".
 */
describe('shortLabels', () => {
  test('first word, numbered on a collision', () => {
    expect(shortLabels(['Sales Agent', 'Sales Calls Data', 'Comms Digest'])).toEqual(['Sales', 'Sales 2', 'Comms']);
  });
  test('a custom split and a length cap', () => {
    expect(shortLabels(['chrome-devtools-mcp:a11y', 'vercel:nextjs'], /[:]/, 12)).toEqual(['chrome-devto', 'vercel']);
  });
  test('blank names fall back instead of an empty column', () => {
    expect(shortLabels(['  '])).toEqual(['Agent']);
  });
  test('one helper, used by every slab view-model that labels columns', () => {
    for (const f of ['lib/agents-volume.ts', 'lib/content-volume.ts', 'lib/tasks-volume.ts', 'lib/skills-volume.ts', 'lib/workflows-volume.ts']) {
      const src = readFileSync(path.join(process.cwd(), f), 'utf8');
      expect(src, f).toContain("from '@/lib/short-labels'");
      expect(src, f).not.toMatch(/^function shortLabels/m);
    }
  });
});
