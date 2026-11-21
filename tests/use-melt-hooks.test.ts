/**
 * Melt, the glass exit, driven through its hooks (lib/use-melt.ts) rather than read from source.
 * There is no DOM here, so React's hooks are swapped for a minimal runtime (tests/helpers/hooks-runtime.ts)
 * and the face is a FakeElement whose CSS plays gl-melt while it carries `.is-melting` (the
 * `attach` step commits the class and the refs after each render, before its effects, like React).
 *
 * Canon: per melt the face's onGone runs exactly once (its own gl-melt animationend, the fallback
 * timer, or at once when no melt will play), a melting face is inert, and reopening mid-melt keeps
 * the face, interactive. The command palette keeps its query and list through the melt and clears
 * them once the face has gone; Enter mid-melt runs nothing.
 */
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';

vi.mock('react', async (orig) => ({ ...(await orig<typeof import('react')>()), ...(await import('./helpers/hooks-runtime')).hooks }));

const router = vi.hoisted(() => ({ push: vi.fn(), replace: vi.fn(), prefetch: vi.fn(), back: vi.fn(), forward: vi.fn(), refresh: vi.fn() }));
vi.mock('next/navigation', () => ({ useRouter: () => router, usePathname: () => '/' }));

import type { AnimationEvent } from 'react';
import { MELT_FALLBACK_MS, useMeltClose, useMeltEnd, useMeltPhase } from '@/lib/use-melt';
import { CommandPalette } from '@/components/CommandPalette';
import type { Command } from '@/lib/palette';
import { FakeElement, fakeComputedStyle, fakeWindow, renderHook } from './helpers/hooks-runtime';

type Ref<T> = { current: T | null };

/** The glass CSS: a face carrying `.is-melting` plays gl-melt; otherwise its entrance. */
const glassCss = (el: FakeElement): string => (el.className.includes('is-melting') ? 'gl-melt' : 'gl-materialise');
/** Reduced motion (or any rule that stops it): the melt rule is `animation: none`. */
const noMotionCss = () => 'none';

const animationEnd = (target: unknown, currentTarget: unknown, animationName = 'gl-melt') =>
  ({ target, currentTarget, animationName }) as unknown as AnimationEvent<HTMLElement>;

let win: ReturnType<typeof fakeWindow>;
function setup(opts: { reduce?: boolean } = {}) {
  win = fakeWindow(opts);
  vi.stubGlobal('window', win);
  vi.stubGlobal('getComputedStyle', fakeComputedStyle);
}

beforeEach(() => {
  vi.useFakeTimers();
  setup();
});
afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
  vi.clearAllMocks();
});

describe('useMeltClose: an ad-hoc modal the parent unmounts on close', () => {
  function mount(css = glassCss) {
    const onClose = vi.fn();
    const face = new FakeElement('div', css);
    const r = renderHook(() => useMeltClose(onClose), undefined, {
      attach: (m) => {
        face.className = m.melting ? 'c-pop is-melting' : 'c-pop';
        (m.faceRef as Ref<FakeElement>).current = face;
      },
    });
    return { onClose, face, r, m: () => r.result.current };
  }

  test('close melts the face (is-melting, inert); its own gl-melt end calls onClose once; the fallback after it does not', () => {
    const { onClose, face, m } = mount();
    expect(m().melting).toBe(false);
    expect(face.hasAttribute('inert')).toBe(false);

    m().close();
    expect(m().melting).toBe(true);
    expect(face.className).toContain('is-melting');
    expect(m().layer['data-melting']).toBe('');
    expect(face.hasAttribute('inert')).toBe(true);
    expect(onClose).not.toHaveBeenCalled();

    m().onAnimationEnd(animationEnd(face, face));
    expect(onClose).toHaveBeenCalledTimes(1);
    // the parent has not unmounted the face yet: the fallback must not close it a second time
    expect(vi.getTimerCount()).toBe(0);
    vi.advanceTimersByTime(MELT_FALLBACK_MS * 3);
    expect(onClose).toHaveBeenCalledTimes(1);
    // nor a repeated animationend (the face keeps melting until the parent drops it)
    m().onAnimationEnd(animationEnd(face, face));
    expect(onClose).toHaveBeenCalledTimes(1);
    expect(face.hasAttribute('inert')).toBe(true);
  });

  test("a child's animationend or another keyframe does not end the melt; the fallback alone ends it, once", () => {
    const { onClose, face, m } = mount();
    m().close();
    const child = new FakeElement('span');
    child.parent = face;
    m().onAnimationEnd(animationEnd(child, face, 'gl-melt'));
    m().onAnimationEnd(animationEnd(face, face, 'gl-materialise'));
    expect(onClose).not.toHaveBeenCalled();

    vi.advanceTimersByTime(MELT_FALLBACK_MS - 1);
    expect(onClose).not.toHaveBeenCalled();
    vi.advanceTimersByTime(1);
    expect(onClose).toHaveBeenCalledTimes(1);
    vi.advanceTimersByTime(MELT_FALLBACK_MS * 3);
    m().onAnimationEnd(animationEnd(face, face));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  test('reduced motion: onClose at once, once (no timer left behind)', () => {
    setup({ reduce: true });
    const { onClose, face, m } = mount();
    m().close();
    expect(onClose).toHaveBeenCalledTimes(1);
    expect(face.hasAttribute('inert')).toBe(true);
    expect(vi.getTimerCount()).toBe(0);
    vi.advanceTimersByTime(MELT_FALLBACK_MS * 3);
    m().onAnimationEnd(animationEnd(face, face));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  test('a face that is not playing gl-melt (animation: none): onClose at once, once', () => {
    const { onClose, m } = mount(noMotionCss);
    m().close();
    expect(onClose).toHaveBeenCalledTimes(1);
    vi.advanceTimersByTime(MELT_FALLBACK_MS * 3);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  test('a double close is still one melt and one onClose', () => {
    const { onClose, face, m } = mount();
    m().close();
    m().close();
    expect(vi.getTimerCount()).toBe(1);
    m().onAnimationEnd(animationEnd(face, face));
    vi.advanceTimersByTime(MELT_FALLBACK_MS * 3);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  test('once the parent unmounts the face nothing fires and the face is not left inert', () => {
    const { onClose, face, r, m } = mount();
    m().close();
    r.unmount();
    expect(face.hasAttribute('inert')).toBe(false);
    expect(vi.getTimerCount()).toBe(0);
    vi.advanceTimersByTime(MELT_FALLBACK_MS * 3);
    expect(onClose).not.toHaveBeenCalled();
  });
});

describe('useMeltPhase + useMeltEnd: an overlay with an open prop (GSheet, LDialog, palettes)', () => {
  function mount(open: boolean, css = glassCss) {
    const face = new FakeElement('div', css);
    const ref: Ref<FakeElement> = { current: null };
    const gone = vi.fn();
    const r = renderHook(
      ({ open }: { open: boolean }) => {
        const melt = useMeltPhase(open);
        const onMeltEnd = useMeltEnd(ref as unknown as Ref<HTMLElement>, melt.melting, () => {
          gone();
          melt.end();
        });
        return { ...melt, onMeltEnd };
      },
      { open },
      {
        attach: (m) => {
          face.className = m.melting ? 'g-sheet is-melting' : 'g-sheet';
          ref.current = m.mounted ? face : null;
        },
      },
    );
    return { face, gone, r, m: () => r.result.current };
  }

  test('open, close: mounted and inert while it melts; its own end unmounts it, once', () => {
    const { face, gone, r, m } = mount(false);
    expect(m().mounted).toBe(false);
    r.rerender({ open: true });
    expect(m()).toMatchObject({ mounted: true, melting: false });
    expect(face.hasAttribute('inert')).toBe(false);

    r.rerender({ open: false });
    expect(m()).toMatchObject({ mounted: true, melting: true });
    expect(face.className).toContain('is-melting');
    expect(face.hasAttribute('inert')).toBe(true);

    m().onMeltEnd(animationEnd(face, face));
    expect(m()).toMatchObject({ mounted: false, melting: false });
    expect(gone).toHaveBeenCalledTimes(1);
    expect(face.hasAttribute('inert')).toBe(false);
    vi.advanceTimersByTime(MELT_FALLBACK_MS * 3);
    expect(gone).toHaveBeenCalledTimes(1);
    expect(m().mounted).toBe(false);
  });

  test('closing a face that was never open mounts nothing', () => {
    const { gone, r, m } = mount(false);
    r.rerender({ open: false });
    expect(m().mounted).toBe(false);
    vi.advanceTimersByTime(MELT_FALLBACK_MS * 3);
    expect(gone).not.toHaveBeenCalled();
  });

  test('reopening mid-melt keeps the face mounted and interactive; the old melt ends nothing', () => {
    const { face, gone, r, m } = mount(true);
    r.rerender({ open: false });
    expect(face.hasAttribute('inert')).toBe(true);
    const staleEnd = m().onMeltEnd;

    r.rerender({ open: true });
    expect(m()).toMatchObject({ mounted: true, melting: false });
    expect(face.hasAttribute('inert')).toBe(false);
    expect(vi.getTimerCount()).toBe(0);
    staleEnd(animationEnd(face, face));
    m().onMeltEnd(animationEnd(face, face));
    vi.advanceTimersByTime(MELT_FALLBACK_MS * 3);
    expect(m().mounted).toBe(true);
    expect(gone).not.toHaveBeenCalled();
  });

  test('closing again after a reopen is a new melt: inert again, and the fallback ends it once', () => {
    const { face, gone, r, m } = mount(true);
    r.rerender({ open: false });
    vi.advanceTimersByTime(MELT_FALLBACK_MS / 2);
    r.rerender({ open: true });
    r.rerender({ open: false });
    expect(face.hasAttribute('inert')).toBe(true);
    vi.advanceTimersByTime(MELT_FALLBACK_MS - 1);
    expect(m().mounted).toBe(true);
    vi.advanceTimersByTime(1);
    expect(m().mounted).toBe(false);
    expect(gone).toHaveBeenCalledTimes(1);
  });

  test('reduced motion: closing unmounts at once', () => {
    setup({ reduce: true });
    const { gone, r, m } = mount(true);
    r.rerender({ open: false });
    expect(m().mounted).toBe(false);
    expect(gone).toHaveBeenCalledTimes(1);
    expect(vi.getTimerCount()).toBe(0);
  });
});

describe('useMeltEnd alone: a toast whose stack drops it later', () => {
  test('leaving: inert, then onGone once even if the stack re-renders it before dropping it', () => {
    const face = new FakeElement('div', glassCss);
    const ref: Ref<FakeElement> = { current: face };
    const onGone = vi.fn();
    const r = renderHook(({ leaving }: { leaving: boolean }) => useMeltEnd(ref as unknown as Ref<HTMLElement>, leaving, onGone), { leaving: false }, {
      attach: () => {},
    });
    face.className = 'ui-toast is-melting';
    r.rerender({ leaving: true });
    expect(face.hasAttribute('inert')).toBe(true);
    r.result.current(animationEnd(face, face));
    expect(onGone).toHaveBeenCalledTimes(1);
    r.rerender({ leaving: true });
    vi.advanceTimersByTime(MELT_FALLBACK_MS * 3);
    r.result.current(animationEnd(face, face));
    expect(onGone).toHaveBeenCalledTimes(1);
  });
});

describe('CommandPalette: the query and list stay through the melt; Enter mid-melt runs nothing', () => {
  const COMMANDS: Command[] = [
    { id: 'alpha', label: 'Alpha', keywords: 'first', href: '/alpha' },
    { id: 'beta', label: 'Beta', keywords: 'second', href: '/beta' },
  ];
  type El = { type: unknown; props: Record<string, unknown> & { children?: unknown }; ref?: unknown };
  function* walk(node: unknown): Generator<El> {
    if (node == null || typeof node !== 'object') return;
    if (Array.isArray(node)) {
      for (const n of node) yield* walk(n);
      return;
    }
    const el = node as El;
    if (!el.props) return;
    yield el;
    yield* walk(el.props.children);
  }
  const find = (tree: unknown, pred: (e: El) => boolean) => [...walk(tree)].find(pred);
  const shows = (tree: unknown, label: string) => [...walk(tree)].some((e) => e.props.title === label || e.props.children === label);

  function mount(ui: 'glados' | 'classic') {
    const html = new FakeElement('html');
    html.setAttribute('data-ui', ui);
    const doc = { activeElement: null as unknown, documentElement: html };
    vi.stubGlobal('document', doc);
    vi.stubGlobal('HTMLElement', FakeElement);
    const face = new FakeElement('div', glassCss);
    const input = new FakeElement('input');
    input.parent = face;
    const refs = new Set<Ref<FakeElement>>();
    const r = renderHook(() => CommandPalette({ commands: COMMANDS, ui }), undefined, {
      attach: (tree) => {
        for (const ref of refs) ref.current = null;
        const f = find(tree, (e) => e.props.role === 'dialog');
        const i = find(tree, (e) => !!e.ref && 'value' in e.props);
        if (f) {
          face.className = String(f.props.className);
          (f.ref as Ref<FakeElement>).current = face;
          refs.add(f.ref as Ref<FakeElement>);
        }
        if (i) {
          (i.ref as Ref<FakeElement>).current = input;
          refs.add(i.ref as Ref<FakeElement>);
        }
      },
    });
    const tree = () => r.result.current;
    const field = () => find(tree(), (e) => !!e.ref && 'value' in e.props)!;
    const type = (v: string) => {
      const p = field().props as { onValueChange?: (v: string) => void; onChange?: (e: unknown) => void };
      if (p.onValueChange) p.onValueChange(v);
      else p.onChange!({ target: { value: v } });
    };
    const key = (k: string) => (field().props.onKeyDown as (e: unknown) => void)({ key: k, preventDefault() {} });
    const scrim = () => find(tree(), (e) => e.props.className === 'gl-scrim')!.props.onClick as () => void;
    const open = () => win.dispatchEvent({ type: 'glados:palette' });
    return { doc, face, input, tree, field, type, key, scrim, open };
  }

  test.each(['glados', 'classic'] as const)('%s: close keeps query and list while melting, blurs the input; the melt end clears them', (ui) => {
    const p = mount(ui);
    expect(p.tree()).toBeNull();
    p.open();
    expect(p.doc.activeElement).toBe(p.input);
    p.type('alp');
    expect(shows(p.tree(), 'Alpha')).toBe(true);
    expect(shows(p.tree(), 'Beta')).toBe(false);

    p.scrim()();
    expect(p.tree()).not.toBeNull();
    expect(p.face.className).toContain('is-melting');
    expect(p.face.hasAttribute('inert')).toBe(true);
    // the list does not empty mid-melt
    expect(p.field().props.value).toBe('alp');
    expect(shows(p.tree(), 'Alpha')).toBe(true);
    expect(shows(p.tree(), 'Beta')).toBe(false);
    // and the input has let go, so no key reaches it
    expect(p.input.blurs).toBeGreaterThan(0);
    expect(p.doc.activeElement).not.toBe(p.input);

    // a key that still lands mid-melt runs nothing
    p.key('Enter');
    expect(router.push).not.toHaveBeenCalled();

    (find(p.tree(), (e) => e.props.role === 'dialog')!.props.onAnimationEnd as (e: unknown) => void)(animationEnd(p.face, p.face));
    expect(p.tree()).toBeNull();

    p.open();
    expect(p.field().props.value).toBe('');
    expect(shows(p.tree(), 'Alpha')).toBe(true);
    expect(shows(p.tree(), 'Beta')).toBe(true);
  });

  test.each(['glados', 'classic'] as const)('%s: Enter while open runs the selected command and starts the melt', (ui) => {
    const p = mount(ui);
    p.open();
    p.type('bet');
    p.key('Enter');
    expect(router.push).toHaveBeenCalledTimes(1);
    expect(router.push).toHaveBeenCalledWith('/beta');
    expect(p.face.className).toContain('is-melting');
    expect(p.field().props.value).toBe('bet');
    p.key('Enter');
    expect(router.push).toHaveBeenCalledTimes(1);
  });

  test.each(['glados', 'classic'] as const)('%s: Cmd+K closes into the melt the same way (input blurred, query kept)', (ui) => {
    const p = mount(ui);
    p.open();
    p.type('alp');
    win.dispatchEvent({ type: 'keydown', key: 'k', metaKey: true, preventDefault() {} } as { type: string });
    expect(p.face.className).toContain('is-melting');
    expect(p.doc.activeElement).not.toBe(p.input);
    expect(p.field().props.value).toBe('alp');
    vi.advanceTimersByTime(MELT_FALLBACK_MS);
    expect(p.tree()).toBeNull();
  });
});
