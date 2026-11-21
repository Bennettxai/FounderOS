/**
 * A minimal React hooks runtime, so a test can drive real hooks (and a component's hook logic)
 * without react-dom or a DOM. The test swaps it in for React's hooks:
 *
 *   vi.mock('react', async (orig) => ({ ...(await orig<typeof import('react')>()), ...(await import('./helpers/hooks-runtime')).hooks }));
 *
 * What it models, after React 18: useState (functional updates; a set during render re-runs the
 * render before anything commits; a set with the same value bails out), useRef, useMemo,
 * useCallback, useEffect and useLayoutEffect with deps (layout before passive, every cleanup before
 * every create, cleanups on a dep change and on unmount), and useSyncExternalStore (the client
 * snapshot). An update outside render (an event handler, a timer) renders and commits at once; an
 * update from an effect re-renders after that commit. `attach` runs after each render, before its
 * effects, where React would attach refs. JSX is not rendered: the result is whatever the hook or
 * component returned (for a component, the element tree its children would receive).
 */

type Deps = readonly unknown[] | undefined;
type Cleanup = (() => void) | void;
type EffectHook = { kind: 'effect'; layout: boolean; deps: Deps; cleanup: Cleanup; next?: { create: () => Cleanup; deps: Deps } };
type StateHook = { kind: 'state'; value: unknown; set: (u: unknown) => void };
type RefHook = { kind: 'ref'; ref: { current: unknown } };
type MemoHook = { kind: 'memo'; deps: Deps; value: unknown };
type Hook = EffectHook | StateHook | RefHook | MemoHook;

type Instance = {
  hooks: Hook[];
  index: number;
  fresh: boolean;
  rendering: boolean;
  renderAgain: boolean;
  dirty: boolean;
  flushing: boolean;
  alive: boolean;
  flush: () => void;
};

let current: Instance | null = null;

function slot<H extends Hook>(kind: H['kind'], make: () => H): { h: H; isNew: boolean; inst: Instance } {
  const inst = current;
  if (!inst) throw new Error(`hooks-runtime: ${kind} hook called outside renderHook`);
  const i = inst.index++;
  const had = inst.hooks[i];
  if (had && had.kind !== kind) throw new Error(`hooks-runtime: hook ${i} was ${had.kind}, now ${kind} (hook order changed)`);
  if (!had && !inst.fresh) throw new Error('hooks-runtime: more hooks than the previous render');
  const h = (had ?? (inst.hooks[i] = make())) as H;
  return { h, isNew: !had, inst };
}

const depsChanged = (prev: Deps, next: Deps) =>
  !prev || !next || prev.length !== next.length || prev.some((x, i) => !Object.is(x, next[i]));

function useState<S>(init: S | (() => S)): [S, (u: S | ((s: S) => S)) => void] {
  const { h } = slot<StateHook>('state', () => {
    const inst = current!;
    const s: StateHook = {
      kind: 'state',
      value: typeof init === 'function' ? (init as () => S)() : init,
      set: (u) => {
        if (!inst.alive) return;
        const next = typeof u === 'function' ? (u as (v: unknown) => unknown)(s.value) : u;
        if (Object.is(next, s.value)) return;
        s.value = next;
        if (inst.rendering) inst.renderAgain = true;
        else {
          inst.dirty = true;
          inst.flush();
        }
      },
    };
    return s;
  });
  return [h.value as S, h.set as (u: S | ((s: S) => S)) => void];
}

function useRef<T>(init: T): { current: T } {
  return slot<RefHook>('ref', () => ({ kind: 'ref', ref: { current: init } })).h.ref as { current: T };
}

function useMemo<T>(make: () => T, deps: Deps): T {
  const { h, isNew } = slot<MemoHook>('memo', () => ({ kind: 'memo', deps: undefined, value: undefined }));
  if (isNew || depsChanged(h.deps, deps)) {
    h.value = make();
    h.deps = deps;
  }
  return h.value as T;
}

function useCallback<T extends (...args: never[]) => unknown>(fn: T, deps: Deps): T {
  return useMemo(() => fn, deps);
}

function effect(layout: boolean, create: () => Cleanup, deps: Deps) {
  const { h, isNew } = slot<EffectHook>('effect', () => ({ kind: 'effect', layout, deps: undefined, cleanup: undefined }));
  h.next = isNew || depsChanged(h.deps, deps) ? { create, deps } : undefined;
}

function useEffect(create: () => Cleanup, deps?: Deps) {
  effect(false, create, deps);
}

function useLayoutEffect(create: () => Cleanup, deps?: Deps) {
  effect(true, create, deps);
}

function useSyncExternalStore<T>(_subscribe: unknown, getSnapshot: () => T): T {
  return getSnapshot();
}

/** Spread over the real module in vi.mock('react'). */
export const hooks = { useState, useRef, useMemo, useCallback, useEffect, useLayoutEffect, useSyncExternalStore };

const effectsOf = (inst: Instance, layout: boolean) =>
  inst.hooks.filter((h): h is EffectHook => h.kind === 'effect' && h.layout === layout);

function runCleanups(list: EffectHook[]) {
  for (const e of list) {
    const c = e.cleanup;
    e.cleanup = undefined;
    if (typeof c === 'function') c();
  }
}

export type RenderHook<P, R> = {
  result: { current: R; renders: number };
  rerender: (props?: P) => void;
  unmount: () => void;
};

/** Mount `fn` as one component. `attach` sees each render's result before that render's effects run. */
export function renderHook<P, R>(fn: (props: P) => R, initialProps?: P, opts: { attach?: (result: R) => void } = {}): RenderHook<P, R> {
  let props = initialProps as P;
  const result = { current: undefined as R, renders: 0 };

  const commit = () => {
    for (const layout of [true, false]) {
      const due = effectsOf(inst, layout).filter((e) => e.next);
      runCleanups(due);
      for (const e of due) {
        const { create, deps } = e.next!;
        e.next = undefined;
        e.deps = deps;
        e.cleanup = create();
      }
    }
  };

  const renderOnce = () => {
    let passes = 0;
    do {
      if (++passes > 25) throw new Error('hooks-runtime: too many re-renders during render');
      inst.renderAgain = false;
      inst.index = 0;
      const prev = current;
      current = inst;
      inst.rendering = true;
      try {
        result.current = fn(props);
      } finally {
        inst.rendering = false;
        current = prev;
      }
      if (inst.index !== inst.hooks.length) throw new Error('hooks-runtime: fewer hooks than the previous render');
      inst.fresh = false;
    } while (inst.renderAgain);
    result.renders++;
    opts.attach?.(result.current);
    commit();
  };

  const inst: Instance = {
    hooks: [],
    index: 0,
    fresh: true,
    rendering: false,
    renderAgain: false,
    dirty: true,
    flushing: false,
    alive: true,
    flush: () => {
      if (inst.flushing || !inst.alive) return;
      inst.flushing = true;
      try {
        let loops = 0;
        while (inst.dirty && inst.alive) {
          if (++loops > 50) throw new Error('hooks-runtime: update loop');
          inst.dirty = false;
          renderOnce();
        }
      } finally {
        inst.flushing = false;
      }
    },
  };

  inst.flush();
  return {
    result,
    rerender(next?: P) {
      if (arguments.length) props = next as P;
      inst.dirty = true;
      inst.flush();
    },
    unmount() {
      if (!inst.alive) return;
      inst.alive = false;
      runCleanups(effectsOf(inst, true));
      runCleanups(effectsOf(inst, false));
    },
  };
}

/** A stand-in DOM element: attributes, focus, containment, and the animation its CSS is playing. */
export class FakeElement {
  readonly attrs = new Map<string, string>();
  className = '';
  parent: FakeElement | null = null;
  blurs = 0;
  constructor(
    readonly name = 'div',
    /** Its CSS: what getComputedStyle(el).animationName says right now (it may read the className). */
    public css: (el: FakeElement) => string = () => 'none',
  ) {}
  setAttribute(n: string, v: string) {
    this.attrs.set(n, String(v));
  }
  removeAttribute(n: string) {
    this.attrs.delete(n);
  }
  hasAttribute(n: string) {
    return this.attrs.has(n);
  }
  getAttribute(n: string) {
    return this.attrs.get(n) ?? null;
  }
  contains(other: unknown): boolean {
    for (let n = other as FakeElement | null; n; n = n.parent) if (n === this) return true;
    return false;
  }
  focus() {
    const doc = (globalThis as { document?: { activeElement: unknown } }).document;
    if (doc) doc.activeElement = this;
  }
  blur() {
    this.blurs++;
    const doc = (globalThis as { document?: { activeElement: unknown } }).document;
    if (doc && doc.activeElement === this) doc.activeElement = null;
  }
  get tagName() {
    return this.name.toUpperCase();
  }
  get isContentEditable() {
    return false;
  }
}

/** getComputedStyle for FakeElements (only animationName is read). */
export const fakeComputedStyle = (el: FakeElement) => ({ animationName: el.css(el) });

/** window.matchMedia answering `(prefers-reduced-motion: reduce)` with `reduce`. */
export const fakeMatchMedia = (reduce: boolean) => (q: string) => ({ matches: reduce && q.includes('prefers-reduced-motion: reduce'), media: q });

/** A window with listeners (keydown, custom events) and matchMedia. */
export function fakeWindow(opts: { reduce?: boolean } = {}) {
  const listeners = new Map<string, Set<(e: unknown) => void>>();
  return {
    matchMedia: fakeMatchMedia(opts.reduce ?? false),
    addEventListener(type: string, fn: (e: unknown) => void) {
      (listeners.get(type) ?? listeners.set(type, new Set()).get(type)!).add(fn);
    },
    removeEventListener(type: string, fn: (e: unknown) => void) {
      listeners.get(type)?.delete(fn);
    },
    dispatchEvent(e: { type: string }) {
      for (const fn of [...(listeners.get(e.type) ?? [])]) fn(e);
      return true;
    },
    open() {
      return null;
    },
    listeners,
  };
}
