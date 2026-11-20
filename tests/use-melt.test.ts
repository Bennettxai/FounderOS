/**
 * Melt, the glass exit (FounderOS Glass brief §4, §5 B6): on close a face plays `.is-melting`
 * (rim, pool and frost to 0, scale to .98, 8 frames = 333ms) and unmounts on its own
 * animationend. Under reduced motion, or whenever the melt keyframe is not what the face
 * plays, animationend never fires, so the face goes at once.
 *
 * The state logic is pure (lib/use-melt.ts); the overlays wire it (GSheet, GDrawer, GDialog,
 * the shared toast, LDialog, both command palettes).
 */
import fs from 'node:fs';
import path from 'node:path';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, test } from 'vitest';
import { MELT_FALLBACK_MS, MELT_MS, isOwnMeltEnd, meltNext, mergeLeaving, skipMelt, type MeltPhase } from '@/lib/use-melt';
import { ToastStack, toToastItem } from '@/components/ui/Toast';
import { LDialog } from '@/components/legacy/Dialog';

const read = (rel: string) => fs.readFileSync(path.join(__dirname, '..', rel), 'utf8');

describe('meltNext: open, melting, closed', () => {
  const run = (from: MeltPhase, ...evs: Parameters<typeof meltNext>[1][]) => evs.reduce(meltNext, from);

  test('closing an open face melts it; the melt end unmounts it', () => {
    expect(meltNext('open', 'close')).toBe('melting');
    expect(meltNext('melting', 'end')).toBe('closed');
    expect(run('open', 'close', 'end')).toBe('closed');
  });

  test('reopening mid-melt keeps the face (no unmount, no remount)', () => {
    expect(meltNext('melting', 'open')).toBe('open');
    expect(run('open', 'close', 'open', 'end')).toBe('open');
  });

  test('a stray end or a second close changes nothing', () => {
    expect(meltNext('open', 'end')).toBe('open');
    expect(meltNext('closed', 'end')).toBe('closed');
    expect(meltNext('closed', 'close')).toBe('closed');
    expect(meltNext('melting', 'close')).toBe('melting');
    expect(meltNext('closed', 'open')).toBe('open');
  });

  test('the melt is 8 frames at 24fps, with a fallback just past it', () => {
    expect(MELT_MS).toBe(333);
    expect(MELT_FALLBACK_MS).toBeGreaterThan(MELT_MS);
    expect(MELT_FALLBACK_MS).toBeLessThanOrEqual(MELT_MS + 300);
  });
});

describe('skipMelt: when animationend will never come', () => {
  test('reduced motion skips the melt', () => {
    expect(skipMelt({ reducedMotion: true, animationName: 'gl-melt' })).toBe(true);
  });

  test('animation none (or anything that is not the melt) skips it', () => {
    expect(skipMelt({ reducedMotion: false, animationName: 'none' })).toBe(true);
    expect(skipMelt({ reducedMotion: false, animationName: '' })).toBe(true);
    expect(skipMelt({ reducedMotion: false, animationName: null })).toBe(true);
    // the entrance is still the computed animation: the melt rule did not apply
    expect(skipMelt({ reducedMotion: false, animationName: 'gl-materialise' })).toBe(true);
    expect(skipMelt({ reducedMotion: false, animationName: 'gl-meltdown' })).toBe(true);
  });

  test('the melt playing (alone or in a list) waits for its end', () => {
    expect(skipMelt({ reducedMotion: false, animationName: 'gl-melt' })).toBe(false);
    expect(skipMelt({ reducedMotion: false, animationName: 'gl-follow, gl-melt' })).toBe(false);
  });
});

describe("isOwnMeltEnd: only the face's own melt unmounts it", () => {
  const face = {};
  test('the face finishing gl-melt', () => {
    expect(isOwnMeltEnd({ target: face, currentTarget: face, animationName: 'gl-melt' })).toBe(true);
  });
  test('a child animation bubbling up, or another keyframe, is ignored', () => {
    expect(isOwnMeltEnd({ target: {}, currentTarget: face, animationName: 'gl-melt' })).toBe(false);
    expect(isOwnMeltEnd({ target: face, currentTarget: face, animationName: 'gl-materialise' })).toBe(false);
  });
});

describe('mergeLeaving: a toast stack keeps a dismissed toast until it melts', () => {
  const t = (id: number) => ({ id, text: `t${id}` });
  const ids = (xs: { item: { id: number }; leaving: boolean }[]) => xs.map((x) => `${x.item.id}${x.leaving ? '~' : ''}`);

  test('new items are appended, present ones stay put', () => {
    expect(ids(mergeLeaving([], [t(1), t(2)]))).toEqual(['1', '2']);
    expect(ids(mergeLeaving(mergeLeaving([], [t(1)]), [t(1), t(2)]))).toEqual(['1', '2']);
  });

  test('a removed item stays in its slot, marked leaving', () => {
    const before = mergeLeaving([], [t(1), t(2), t(3)]);
    expect(ids(mergeLeaving(before, [t(1), t(3)]))).toEqual(['1', '2~', '3']);
  });

  test('present items take the newest version', () => {
    const before = mergeLeaving([], [t(1)]);
    const after = mergeLeaving(before, [{ id: 1, text: 'edited' }]);
    expect(after[0].item.text).toBe('edited');
  });

  test('nothing changes when nothing changed (same array back)', () => {
    const before = mergeLeaving([], [t(1), t(2)]);
    expect(mergeLeaving(before, before.map((x) => x.item))).toBe(before);
  });
});

describe('the overlays wire the melt', () => {
  test('GSheet, GDrawer and GDialog stay mounted while they melt and release the dialog at once', () => {
    const src = read('components/g/Overlays.tsx');
    for (const name of ['GSheet', 'GDrawer', 'GDialog']) {
      const at = src.indexOf(`export function ${name}(`);
      expect(src.slice(at, at + 400), name).toMatch(/useMeltPhase\(props\.open\)/);
    }
    // each face: the class, its own animationend, and data-melting on the layer for the scrim
    expect(src.match(/melting && 'is-melting'/g)?.length).toBe(3);
    expect(src.match(/onAnimationEnd=\{onMeltEnd\}/g)?.length).toBe(3);
    expect(src.match(/data-melting=\{melting \? '' : undefined\}/g)?.length).toBe(3);
  });

  test('the shared toast melts on dismiss', () => {
    const src = read('components/ui/Toast.tsx');
    expect(src).toMatch(/mergeLeaving\(/);
    expect(src).toMatch(/leaving && 'is-melting'|leaving \? 'ui-toast is-melting'/);
    expect(src).toMatch(/onAnimationEnd=\{onMeltEnd\}/);
  });

  test('LDialog and both palettes melt', () => {
    expect(read('components/legacy/Dialog.tsx')).toMatch(/useMeltPhase\(open\)/);
    const pal = read('components/CommandPalette.tsx');
    expect(pal).toMatch(/useMeltPhase\(open\)/);
    expect(pal.match(/onAnimationEnd=\{onMeltEnd\}/g)?.length).toBe(2);
  });

  test('a toast stack renders as before, and nothing when empty', () => {
    const html = renderToStaticMarkup(createElement(ToastStack, { items: [toToastItem({ text: 'Saved' }, 1)], onDismiss: () => {} }));
    expect(html).toContain('class="ui-toast"');
    expect(html).not.toContain('is-melting');
    expect(renderToStaticMarkup(createElement(ToastStack, { items: [], onDismiss: () => {} }))).toBe('');
  });

  test('LDialog: closed renders nothing; open renders the face as a sibling of its scrim', () => {
    expect(renderToStaticMarkup(createElement(LDialog, { open: false, onClose: () => {}, title: 'x' }))).toBe('');
    const html = renderToStaticMarkup(createElement(LDialog, { open: true, onClose: () => {}, title: 'Rename', children: 'body' }));
    expect(html).not.toContain('is-melting');
    expect(html).not.toContain('data-melting');
  });
});
