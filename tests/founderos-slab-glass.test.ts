/**
 * founderos/ (the /agents and /social slab kit) on the glass pack. Its cards named os-slab, rise and
 * rise-card, and played os-rise and os-drift: none of those exist in app/globals.css, so the cards
 * were flat boxes with no entrance. They now use the kit's classes: a SlabCard is a glass face that
 * materialises on the slab's 90ms stagger; Rise is the Legacy l-rise (transform only, never a fade).
 * Tailwind also scans founderos/, so its utilities generate.
 */
import fs from 'node:fs';
import path from 'node:path';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, test } from 'vitest';
import config from '@/tailwind.config';
import { InsightCard, Slab, SlabCard, SlabTitle } from '@/founderos/components/slab';
import { Rise } from '@/founderos/components/motion';
import { CSS_NO_COMMENTS } from './helpers/theme-css';
import { keyframeBodies } from './helpers/glass-lint';

const ROOT = path.join(__dirname, '..');
const html = (el: Parameters<typeof renderToStaticMarkup>[0]) => renderToStaticMarkup(el);
const classesOf = (markup: string) => (/class="([^"]*)"/.exec(markup)?.[1] ?? '').split(/\s+/);
const hasRule = (cls: string) => new RegExp(`\\.${cls.replace(/[-]/g, '\\-')}(?![\\w-])[^{]*\\{`).test(CSS_NO_COMMENTS);

describe('Tailwind scans founderos/', () => {
  test('content includes ./founderos/**/*.{ts,tsx}', () => {
    expect(config.content).toEqual(expect.arrayContaining(['./founderos/**/*.{ts,tsx}']));
  });
});

describe('the founderos slab kit uses classes that exist', () => {
  test('SlabCard is a glass face that materialises on the stagger', () => {
    const m = html(createElement(SlabCard, { i: 3, title: 'Accounts', className: 'p-6' }, 'x'));
    const cls = classesOf(m);
    expect(cls).toEqual(expect.arrayContaining(['glass-face', 'gl-materialise', 'l-rise', 'l-rise-card', 'p-6']));
    expect(cls).not.toContain('rise-card');
    expect(cls).not.toContain('rise');
    expect(m).toMatch(/style="--l-rise-i:3"/);
    expect(m).toContain('Accounts');
  });

  test('Rise is the Legacy rise (transform only), staggered by --l-rise-i', () => {
    const m = html(createElement(Rise, { i: 6, className: 'mb-6' }, 'x'));
    expect(classesOf(m)).toEqual(['l-rise', 'mb-6']);
    expect(m).toMatch(/style="--l-rise-i:6"/);
    expect(keyframeBodies().get('l-rise')).not.toMatch(/opacity/);
  });

  test('Slab and SlabTitle carry no dead class', () => {
    expect(html(createElement(Slab, { className: 'x', children: 'y' }))).not.toMatch(/os-slab/);
    expect(classesOf(html(createElement(SlabTitle, { title: 'Real Agents' })))).toContain('l-rise');
  });

  test('every kit class the founderos slab names has a rule in app/globals.css', () => {
    for (const cls of ['glass-face', 'gl-materialise', 'l-rise', 'l-rise-card']) expect(hasRule(cls), cls).toBe(true);
    const src = ['founderos/components/slab.tsx', 'founderos/components/motion.tsx'].map((f) => fs.readFileSync(path.join(ROOT, f), 'utf8')).join('\n');
    expect(src).not.toMatch(/\bos-slab\b|['" `]rise-card\b|\bos-rise\b|\bos-drift\b|--rise-i\b|className=\{`rise /);
  });

  test('every keyframe an inline animation names anywhere in founderos/ exists (charts included)', () => {
    const kf = keyframeBodies();
    const walk = (d: string): string[] =>
      fs.readdirSync(path.join(ROOT, d), { withFileTypes: true }).flatMap((e) =>
        e.isDirectory() ? walk(path.join(d, e.name)) : /\.tsx?$/.test(e.name) ? [path.join(d, e.name)] : [],
      );
    const missing: string[] = [];
    for (const f of walk('founderos')) {
      for (const m of fs.readFileSync(path.join(ROOT, f), 'utf8').matchAll(/animation:\s*['"`]([^'"`]+)['"`]/g)) {
        for (const part of m[1].split(/,(?![^(]*\))/)) {
          const name = part.trim().split(/\s+/)[0];
          if (name && name !== 'none' && !/^\$\{/.test(name) && !kf.has(name)) missing.push(`${f}: ${name}`);
        }
      }
    }
    expect(missing).toEqual([]);
  });

  test('the charts map to real motion: a dashoffset draw, leaf fades, the meter grow, the caret blink; the peak label pops', () => {
    const read = (f: string) => fs.readFileSync(path.join(ROOT, f), 'utf8');
    const charts = read('founderos/components/slab-charts.tsx');
    expect(charts).toMatch(/strokeDasharray: 1, strokeDashoffset: 1, animation: `l-draw /);
    expect(charts).toMatch(/className="glass-solid pointer-events-none absolute/);
    expect(charts).toMatch(/animation: `gl-pop /);
    expect(charts).not.toMatch(/backdrop-blur/);
    const pie = read('founderos/components/SharePie.tsx');
    expect(pie).toMatch(/strokeDasharray: 1, strokeDashoffset: 1, animation: `l-draw \$\{dur\}ms linear both`/);
    expect(read('founderos/components/VolumeMeter.tsx')).toMatch(/animation: `l-meter-in /);
    expect(read('founderos/components/Synthesizing.tsx')).toMatch(/animation: 'blink 1s steps\(1\) infinite'/);
    expect(keyframeBodies().get('gl-pop')).toMatch(/from \{\s*visibility: hidden;\s*scale: 0\.9;\s*\}/);
    expect(keyframeBodies().get('gl-pop')).not.toMatch(/opacity/);
  });

  test('the InsightCard paints from tokens that exist in every style (no undefined var, no literal white)', () => {
    const m = html(createElement(InsightCard, { badge: 'Insight', headline: 'h', display: '42', frac: 0.5 }));
    const names = [...m.matchAll(/var\((--[\w-]+)/g)].map((x) => x[1]);
    expect(names.length).toBeGreaterThan(0);
    const missing = [...new Set(names)].filter((n) => !new RegExp(`${n}\\s*:`).test(CSS_NO_COMMENTS));
    expect(missing).toEqual([]);
    expect(m).not.toMatch(/text-white|bg-white|border-white|#fff\b|rgba\(255,\s*255,\s*255/);
    expect(m).toMatch(/color:var\(--fc-ins-ink\)/);
  });

  test('the InsightCard rises and drifts on the Legacy keyframes', () => {
    const m = html(createElement(InsightCard, { badge: 'Insight', headline: 'h', display: '42' }));
    expect(m).toMatch(/animation:l-rise [^;"]*calc\(var\(--l-rise-i\) \* 90ms\)[^;"]*, ?l-drift /);
    expect(classesOf(m)).toContain('l-rise-card');
  });
});
