'use client';
/**
 * Glass surface and the glass budget (02 section 4). At most GLASS_BUDGET blurred surfaces are in view
 * (24: page glass takes the glass pack's 3px light frost, see tokens.ts); the rest paint the solid face.
 * Small controls use tone 'tint' and never blur.
 */
import {
  createContext, forwardRef, useCallback, useContext, useEffect, useLayoutEffect, useMemo, useRef, useState,
  type HTMLAttributes, type ReactNode, type RefObject,
} from 'react';
import { GLASS_BUDGET } from './tokens';
import { useKitMark } from '@/components/lab/v2/shared/kitMark';

export type GlassTone = 'base' | 'strong' | 'tint' | 'solid';
export type GlassRadius = 'tile' | 'narrative' | 'sheet' | 'chip' | 'pill';
export type GlassPriority = 0 | 1 | 2 | 3; // 0 chrome, 1 sheet, 2 tiles, 3 everything else

type Slot = { priority: GlassPriority; set: (on: boolean) => void; visible: boolean };
type Budget = { register: (el: HTMLElement, priority: GlassPriority, set: (on: boolean) => void) => () => void; enabled: () => boolean };

const BudgetContext = createContext<Budget | null>(null);
const useIso = typeof window === 'undefined' ? useEffect : useLayoutEffect;

/** Can this browser and style blur at all? Read once per provider from the .lab root. */
function blurEnabled(): boolean {
  try {
    if (window.matchMedia?.('(prefers-reduced-transparency: reduce)').matches) return false;
    const root = document.querySelector('.lab');
    if (!root) return false;
    const v = parseFloat(getComputedStyle(root).getPropertyValue('--lab-glass-blur'));
    return Number.isFinite(v) && v > 0;
  } catch {
    return false;
  }
}

export function GlassBudgetProvider({ children }: { children: ReactNode }) {
  const slots = useRef(new Map<HTMLElement, Slot>());
  const io = useRef<IntersectionObserver | null>(null);
  const raf = useRef(0);
  const enabledRef = useRef<boolean | null>(null);

  const enabled = useCallback(() => {
    if (enabledRef.current === null) enabledRef.current = blurEnabled();
    return enabledRef.current;
  }, []);

  const recompute = useCallback(() => {
    raf.current = 0;
    const on = enabled();
    const visible = [...slots.current.entries()].filter(([, s]) => s.visible);
    visible.sort(([a, sa], [b, sb]) => {
      if (sa.priority !== sb.priority) return sa.priority - sb.priority;
      return a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING ? -1 : 1;
    });
    const granted = new Set(on ? visible.slice(0, GLASS_BUDGET).map(([el]) => el) : []);
    for (const [el, s] of slots.current) s.set(granted.has(el));
  }, [enabled]);

  const schedule = useCallback(() => {
    if (raf.current) return;
    raf.current = requestAnimationFrame(recompute);
  }, [recompute]);

  useEffect(() => {
    if (typeof IntersectionObserver === 'undefined') return;
    io.current = new IntersectionObserver((entries) => {
      for (const e of entries) {
        const s = slots.current.get(e.target as HTMLElement);
        if (s) s.visible = e.isIntersecting;
      }
      schedule();
    }, { rootMargin: '0px', threshold: 0 });
    for (const el of slots.current.keys()) io.current.observe(el);
    return () => {
      io.current?.disconnect();
      io.current = null;
      if (raf.current) cancelAnimationFrame(raf.current);
    };
  }, [schedule]);

  const register = useCallback<Budget['register']>((el, priority, set) => {
    slots.current.set(el, { priority, set, visible: false });
    io.current?.observe(el);
    return () => {
      slots.current.delete(el);
      io.current?.unobserve(el);
      schedule();
    };
  }, [schedule]);

  const value = useMemo<Budget>(() => ({ register, enabled }), [register, enabled]);
  return <BudgetContext.Provider value={value}>{children}</BudgetContext.Provider>;
}

/** true = this surface may blur. Priority 0 and 1 render blurred on first paint; others wait for a grant. */
export function useGlassSlot(ref: RefObject<HTMLElement>, opts: { priority: GlassPriority; eligible: boolean }): boolean {
  const budget = useContext(BudgetContext);
  const [on, setOn] = useState(opts.eligible && opts.priority <= 1);
  useIso(() => {
    const el = ref.current;
    if (!el || !opts.eligible || !budget) {
      if (!opts.eligible) setOn(false);
      return;
    }
    return budget.register(el, opts.priority, setOn);
  }, [budget, ref, opts.eligible, opts.priority]);
  return opts.eligible && on;
}

type GlassTag = 'div' | 'section' | 'article' | 'li' | 'button' | 'a' | 'form' | 'nav';
export type GlassProps = {
  as?: GlassTag;
  tone?: GlassTone;
  radius?: GlassRadius;
  priority?: GlassPriority;
  interactive?: boolean;
  className?: string;
  children?: ReactNode;
  href?: string;
  type?: 'button' | 'submit';
} & Omit<HTMLAttributes<HTMLElement>, 'className' | 'children'> & { [attr: `data-${string}`]: string | undefined };

/** A glass surface. Blur only when tone is base or strong and the budget grants a slot. */
export const Glass = forwardRef<HTMLElement, GlassProps>(function Glass(
  { as = 'div', tone = 'base', radius = 'tile', priority = 3, interactive, className, children, ...rest },
  fwd,
) {
  const km = useKitMark();
  const local = useRef<HTMLElement | null>(null);
  const eligible = tone === 'base' || tone === 'strong';
  const blur = useGlassSlot(local, { priority, eligible });
  const setRef = (el: HTMLElement | null) => {
    local.current = el;
    if (typeof fwd === 'function') fwd(el);
    else if (fwd) fwd.current = el;
  };
  const Tag = as as 'div';
  return (
    <Tag
      ref={setRef as (el: HTMLDivElement | null) => void}
      className={['lab2-glass', className].filter(Boolean).join(' ')}
      data-tone={tone}
      data-radius={radius}
      data-blur={blur ? 'on' : 'off'}
      data-interactive={interactive ? '' : undefined}
      data-lab="Glass"
      {...km('LabGlass')}
      {...(rest as Record<string, unknown>)}
    >
      {children}
    </Tag>
  );
});
