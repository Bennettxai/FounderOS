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

  test('every keyframe an inline animation names in the slab kit exists', () => {
    // (The founderos charts still name os-draw, os-fade, os-sweep, vol-meter-in and om-blink, which
    // do not exist either; they are chart motion, not faces, and are left for their own pass.)
    const kf = keyframeBodies();
    const files = ['founderos/components/slab.tsx', 'founderos/components/motion.tsx'].map((f) => path.join(ROOT, f));
    const missing: string[] = [];
    for (const f of files) {
      for (const m of fs.readFileSync(f, 'utf8').matchAll(/animation:\s*['"`]([^'"`]+)['"`]/g)) {
        for (const part of m[1].split(/,(?![^(]*\))/)) {
          const name = part.trim().split(/\s+/)[0];
          if (name && name !== 'none' && !kf.has(name)) missing.push(`${path.relative(ROOT, f)}: ${name}`);
        }
      }
    }
    expect(missing).toEqual([]);
  });

  test('the InsightCard rises and drifts on the Legacy keyframes', () => {
    const m = html(createElement(InsightCard, { badge: 'Insight', headline: 'h', display: '42' }));
    expect(m).toMatch(/animation:l-rise [^;"]*calc\(var\(--l-rise-i\) \* 90ms\)[^;"]*, ?l-drift /);
    expect(classesOf(m)).toContain('l-rise-card');
  });
});
