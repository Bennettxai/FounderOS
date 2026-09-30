import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import path from 'node:path';

/**
 * Home wears the Brand Deals look all the way down (Alex, 2026-09-24: "I
 * love that look"): inset rounded cards that lift on hover, big mixed-case
 * card titles, header actions as pills, roomy rows, a rounded input. The
 * pieces carry data-part hooks and only `.os-slab` restyles them, so the
 * same components elsewhere (the Agents tabs) keep the console look.
 */
const read = (rel: string) => readFileSync(path.join(process.cwd(), rel), 'utf8');
const css = read('app/globals.css');
const rule = (selector: string) => {
  const i = css.indexOf(selector);
  expect(i, `missing ${selector}`).toBeGreaterThan(-1);
  return css.slice(i, css.indexOf('}', i));
};

describe('Home: the Brand Deals slab, all the way down', () => {
  test('the shared primitives expose stable hooks', () => {
    expect(read('components/terminal.tsx')).toContain('data-part="label"');
    const list = read('components/NeedsYouList.tsx');
    for (const p of ['card', 'card-head', 'row']) expect(list, p).toContain(`data-part="${p}"`);
    const composer = read('components/InterjectComposer.tsx');
    for (const p of ['card', 'card-head', 'input']) expect(composer, p).toContain(`data-part="${p}"`);
    const page = read('app/page.tsx');
    for (const p of ['card', 'card-head', 'row', 'tile']) expect(page, p).toContain(`data-part="${p}"`);
  });

  test('card titles are Brand Deals titles: 19px, semibold, mixed case', () => {
    const r = rule('.os-slab [data-part="card-head"] [data-part="label"]');
    expect(r).toMatch(/font-size:\s*19px/);
    expect(r).toMatch(/text-transform:\s*none/);
    expect(r).toMatch(/font-weight:\s*600/);
  });

  test('cards are inset, rounded 12px, and lift on hover; heads lose the console rule', () => {
    expect(rule('.os-slab [data-part="card"] {')).toMatch(/border-radius:\s*12px/);
    expect(rule('.os-slab [data-part="card"]:hover')).toMatch(/translateY\(-1px\)/);
    expect(rule('.os-slab [data-part="card-head"] {')).toMatch(/border-bottom:\s*0/);
  });

  test('tile labels read like "All deals", not a console eyebrow', () => {
    expect(rule('.os-slab [data-part="tile"] [data-part="label"]')).toMatch(/text-transform:\s*none/);
  });

  test('scoped: nothing restyles the hooks outside .os-slab', () => {
    const unscoped = css.split('\n').filter((l) => l.includes('[data-part=') && !l.includes('.os-slab'));
    expect(unscoped).toEqual([]);
  });
});
