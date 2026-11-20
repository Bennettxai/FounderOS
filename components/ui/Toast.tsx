'use client';

import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import Link from 'next/link';
import { X } from 'lucide-react';
import { GButton } from '@/components/g/Controls';
import { useUiStyle } from '@/lib/use-ui-style';
import { mergeLeaving, useMeltEnd, type Leaving } from '@/lib/use-melt';

/**
 * Shared toasts (SPEC X2). One provider is mounted in app/layout.tsx; any
 * client component calls:
 *
 *   const { toast } = useToast();
 *   toast({ text: 'Removed from watchlist', tone: 'ok', action: { label: 'Undo', onClick: restore }, ms: 5000 });
 *
 * At most three are stacked (the oldest goes first). Info and ok toasts are
 * role="status"; err is role="alert". Colours are tokens only; inside
 * `.ae-octane` the surface reads `--o-toast-bg`. Reduced motion: no pop.
 *
 * A surface that wants its notices near the action (the workflow builder
 * stage) mounts its own `<ToastProvider anchor>` inside a positioned
 * container: calls from inside it stack bottom-centre of that container
 * instead of the viewport.
 *
 * Cmd/Ctrl+Z runs the newest visible toast's Undo (SPEC C40), unless focus is
 * in a text field, where the key keeps its text meaning.
 *
 * A dismissed toast melts out (glass brief §4, lib/use-melt): the stack keeps it in its slot,
 * inert, until its melt ends; under reduced motion it goes at once.
 */

export type ToastTone = 'info' | 'ok' | 'err';

export type ToastInput = {
  text: string;
  tone?: ToastTone;
  action?: { label: string; onClick: () => void };
  /** A second, navigating action: "Open" on the thing the toast is about. */
  link?: { label: string; href: string };
  /** Auto-dismiss after this many ms. 0 keeps it until dismissed. */
  ms?: number;
};

export type ToastItem = Required<Pick<ToastInput, 'text' | 'tone' | 'ms'>> &
  Pick<ToastInput, 'action' | 'link'> & { id: number };

export const TOAST_MAX = 3;
export const TOAST_DEFAULT_MS = 4000;

/** Pure: add one toast, keeping the newest `max`. */
export function pushToast(list: ToastItem[], item: ToastItem, max = TOAST_MAX): ToastItem[] {
  const next = [...list, item];
  return next.length > max ? next.slice(next.length - max) : next;
}

/** Pure: the toast Cmd/Ctrl+Z acts on: the newest one whose action is an Undo (C40). */
export function undoTarget(list: ToastItem[]): ToastItem | null {
  for (let i = list.length - 1; i >= 0; i--) {
    const a = list[i].action;
    if (a && /^undo\b/i.test(a.label.trim())) return list[i];
  }
  return null;
}

/** Pure: normalise caller input into a stored toast. */
export function toToastItem(input: ToastInput, id: number): ToastItem {
  return {
    id,
    text: input.text,
    tone: input.tone ?? 'info',
    ms: input.ms ?? TOAST_DEFAULT_MS,
    action: input.action,
    ...(input.link ? { link: input.link } : {}),
  };
}

export type ToastApi = {
  toast: (input: ToastInput) => number;
  dismiss: (id: number) => void;
};

const ToastContext = createContext<ToastApi | null>(null);

const NOOP_API: ToastApi = {
  toast: () => {
    if (process.env.NODE_ENV !== 'production') {
      console.warn('useToast: no <ToastProvider> above this component; toast dropped.');
    }
    return -1;
  },
  dismiss: () => {},
};

export function useToast(): ToastApi {
  return useContext(ToastContext) ?? NOOP_API;
}

let nextId = 1;

export function ToastProvider({ children, anchor = false }: { children?: ReactNode; anchor?: boolean }) {
  const [items, setItems] = useState<ToastItem[]>([]);
  const timers = useRef(new Map<number, ReturnType<typeof setTimeout>>());

  const dismiss = useCallback((id: number) => {
    const t = timers.current.get(id);
    if (t) clearTimeout(t);
    timers.current.delete(id);
    setItems((list) => list.filter((x) => x.id !== id));
  }, []);

  const toast = useCallback(
    (input: ToastInput) => {
      const item = toToastItem(input, nextId++);
      setItems((list) => {
        const next = pushToast(list, item);
        // Toasts pushed out of the stack lose their timers too.
        for (const old of list) {
          if (!next.includes(old)) {
            const t = timers.current.get(old.id);
            if (t) clearTimeout(t);
            timers.current.delete(old.id);
          }
        }
        return next;
      });
      if (item.ms > 0) timers.current.set(item.id, setTimeout(() => dismiss(item.id), item.ms));
      return item.id;
    },
    [dismiss],
  );

  useEffect(() => {
    const map = timers.current;
    return () => {
      map.forEach((t) => clearTimeout(t));
      map.clear();
    };
  }, []);

  // C40: Cmd/Ctrl+Z triggers the newest toast's Undo while it is visible.
  // An anchored stack (a surface's own provider) leaves the key to the page one.
  const itemsRef = useRef(items);
  itemsRef.current = items;
  useEffect(() => {
    if (anchor) return;
    const onKey = (e: KeyboardEvent) => {
      if (!(e.metaKey || e.ctrlKey) || e.shiftKey || e.altKey || e.key.toLowerCase() !== 'z' || e.defaultPrevented) return;
      const el = document.activeElement as HTMLElement | null;
      if (el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.isContentEditable)) return;
      const t = undoTarget(itemsRef.current);
      if (!t?.action) return;
      e.preventDefault();
      t.action.onClick();
      dismiss(t.id);
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [anchor, dismiss]);

  const api = useMemo<ToastApi>(() => ({ toast, dismiss }), [toast, dismiss]);

  return (
    <ToastContext.Provider value={api}>
      {children}
      <ToastStack items={items} anchor={anchor} onDismiss={dismiss} />
    </ToastContext.Provider>
  );
}

export function ToastStack({
  items,
  anchor = false,
  onDismiss,
}: {
  items: ToastItem[];
  anchor?: boolean;
  onDismiss: (id: number) => void;
}) {
  // Dismissed toasts stay, marked leaving, until they melt (derived during render, React's pattern).
  const [shown, setShown] = useState<Leaving<ToastItem>[]>(() => mergeLeaving([], items));
  const merged = mergeLeaving(shown, items);
  if (merged !== shown) setShown(merged);
  const gone = useCallback((id: number) => setShown((list) => list.filter((x) => x.item.id !== id)), []);
  if (merged.length === 0) return null;
  return (
    <div className="ui-toasts" data-anchor={anchor ? '' : undefined}>
      {merged.map(({ item, leaving }) => (
        <ToastPill key={item.id} item={item} onDismiss={onDismiss} leaving={leaving} onGone={gone} />
      ))}
    </div>
  );
}

/**
 * One toast. Under the GladOS style this is the kit's GToast (SPEC X1, C16):
 * `[data-ui='glados'] .ui-toast` reskins the pill and the action is a real
 * GButton sm (§3.2). Behaviour is the same in both styles; toasts only exist
 * after an interaction, so the client-side style read never meets hydration.
 */
export function ToastPill({
  item: t,
  onDismiss,
  leaving = false,
  onGone,
}: {
  item: ToastItem;
  onDismiss: (id: number) => void;
  /** Dismissed: melting out, no longer announced or interactive. */
  leaving?: boolean;
  /** Called when the melt is over (or skipped): drop it from the stack. */
  onGone?: (id: number) => void;
}) {
  const g = useUiStyle('classic') === 'glados';
  const ref = useRef<HTMLDivElement>(null);
  const onMeltEnd = useMeltEnd(ref, leaving, () => onGone?.(t.id));
  const act = () => {
    t.action?.onClick();
    onDismiss(t.id);
  };
  return (
    <div
      ref={ref}
      className={leaving ? 'ui-toast is-melting' : 'ui-toast'}
      data-g="GToast"
      data-tone={t.tone}
      role={leaving ? undefined : t.tone === 'err' ? 'alert' : 'status'}
      aria-hidden={leaving || undefined}
      onAnimationEnd={onMeltEnd}
    >
      <span className="ui-toast-dot" aria-hidden />
      <span className="ui-toast-text">{t.text}</span>
      {t.link &&
        (g ? (
          <GButton variant="sm" className="ui-toast-gaction" href={t.link.href} onClick={() => onDismiss(t.id)}>
            {t.link.label}
          </GButton>
        ) : (
          <Link href={t.link.href} className="ui-toast-action" onClick={() => onDismiss(t.id)}>
            {t.link.label}
          </Link>
        ))}
      {t.action &&
        (g ? (
          <GButton variant="sm" className="ui-toast-gaction" onClick={act}>
            {t.action.label}
          </GButton>
        ) : (
          <button type="button" className="ui-toast-action" onClick={act}>
            {t.action.label}
          </button>
        ))}
      <button type="button" className="ui-toast-close" aria-label="Dismiss notification" onClick={() => onDismiss(t.id)}>
        <X size={14} strokeWidth={1.75} aria-hidden />
      </button>
    </div>
  );
}
