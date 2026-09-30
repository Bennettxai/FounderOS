import { describe, expect, test } from 'vitest';
import { createElement as h } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { PageSkeleton } from '@/components/PageSkeleton';

/**
 * "The way it opens" (Alex, 2026-09-24): the instant-paint shell every
 * route shows on click is the Brand Deals slab in outline, so the page that
 * lands next grows out of the same shape instead of swapping layouts.
 */
describe('PageSkeleton is the slab in outline', () => {
  const out = renderToStaticMarkup(h(PageSkeleton));

  test('it sits on the same floating slab the pages use', () => {
    expect(out).toContain('os-slab');
  });

  test('hero row is the Brand Deals 2fr/1fr split, then a row of three', () => {
    expect(out).toContain('grid-cols-[2fr_1fr]');
    expect(out).toContain('grid-cols-3');
  });

  test('the volume card outline carries meter tracks, rounded like the real bars', () => {
    expect((out.match(/data-skel-meter/g) ?? []).length).toBe(3);
    expect(out).toContain('rounded-full');
  });

  test('the blocks are the slab card radius and still pulse', () => {
    expect(out).toContain('rounded-[12px]');
    expect(out).toContain('animate-pulse');
  });
});
