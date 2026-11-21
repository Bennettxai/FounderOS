'use client';

import {
  cloneElement,
  isValidElement,
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
  useSyncExternalStore,
  type KeyboardEvent,
  type ReactElement,
  type ReactNode,
  type RefObject,
  type TouchEvent,
} from 'react';
import { createPortal } from 'react-dom';
import Link from 'next/link';
import { ChevronLeft, X } from 'lucide-react';
import { useDialog, FOCUSABLE_SELECTOR } from '@/components/ui/use-dialog';
import { pushEscape } from '@/lib/escape-stack';
import { useMeltEnd, useMeltPhase } from '@/lib/use-melt';
import { cx } from './cx';
import { rovingIndex, swipeCloses } from './logic';
import { GButton } from './Controls';

export { ToastPill as GToast, ToastProvider, useToast } from '@/components/ui/Toast';

/**
 * Overlays (SPEC S00 §5.3, C17, GP3). Every modal layer runs on the shared
 * `useDialog` (focus trap, focus return, scroll lock) and the one Escape
 * stack (X6): Escape closes only the top layer, across surfaces. Sheet
 * actions and dialog buttons are GButton; toasts are the existing
 * components/ui/Toast.tsx reskinned (C16), exported here as GToast.
 *
 * Glass (brief §3.2, §4): each scrim is its face's sibling, never its parent, and frosts the page
 * light to strong while it dims it (.gl-scrim). On close the dialog behaviour releases at once and
 * the face melts out before it unmounts (lib/use-melt); the layer's data-melting clears the scrim.
 */

/** What a panel needs to melt: set by the exported overlay, which owns mounting. */
type MeltProps = { melting: boolean; onMelted: () => void };

const noSubscribe = () => () => {};

/**
 * Render a layer into document.body (a transformed ancestor would trap
 * `position: fixed`). Nothing renders on the server or during hydration, so
 * the panel mounts once, in the portal, and useDialog's focus-in runs on the
 * real node.
 */
export function Layer({ children }: { children: ReactNode }) {
  const target = useSyncExternalStore(
    noSubscribe,
    () => document.body,
    () => null,
  );
  return target ? createPortal(children, target) : null;
}

function CloseButton({ onClose, label = 'Close' }: { onClose: () => void; label?: string }) {
  return (
    <GButton variant="icon" aria-label={label} data-dialog-close="" onClick={onClose}>
      <X size={16} strokeWidth={1.75} aria-hidden />
    </GButton>
  );
}

/* ── GSheet ───────────────────────────────────────────────────────────── */

export type GSheetProps = {
  open: boolean;
  onClose: () => void;
  title: ReactNode;
  /** One muted line under the title. */
  caption?: ReactNode;
  children?: ReactNode;
  /** Actions, right-aligned: GButtons. */
  footer?: ReactNode;
  /** Max width in px (default 1200). */
  width?: number;
  initialFocus?: RefObject<HTMLElement | null>;
  className?: string;
};

/** Centred modal sheet: --g-card, radius 28, Section 30 title, --g-scrim, 240ms in. */
export function GSheet(props: GSheetProps) {
  const melt = useMeltPhase(props.open);
  if (!melt.mounted) return null;
  return (
    <Layer>
      <SheetPanel {...props} melting={melt.melting} onMelted={melt.end} />
    </Layer>
  );
}

function SheetPanel({ open, onClose, title, caption, children, footer, width = 1200, initialFocus, className, melting, onMelted }: GSheetProps & MeltProps) {
  const titleId = useId();
  const dlg = useDialog({ open, onClose, labelledBy: titleId, initialFocus });
  const onMeltEnd = useMeltEnd(dlg.ref, melting, onMelted);
  return (
    <>
      <div className="g-layer" data-kind="sheet" data-melting={melting ? '' : undefined}>
        <div className="g-scrim gl-scrim" aria-hidden onClick={onClose} />
        <div {...dlg.props} data-g="GSheet" className={cx('g-sheet', className, melting && 'is-melting')} style={{ maxWidth: width }} onAnimationEnd={onMeltEnd}>
          <header className="g-sheet-head">
            <div>
              <h2 id={titleId} className="g-sheet-title">
                {title}
              </h2>
              {caption != null && <p className="g-sheet-cap">{caption}</p>}
            </div>
            <CloseButton onClose={onClose} />
          </header>
          <div className="g-sheet-body">{children}</div>
          {footer != null && <footer className="g-sheet-foot">{footer}</footer>}
        </div>
      </div>
    </>
  );
}

/* ── GDrawer ──────────────────────────────────────────────────────────── */

export type GDrawerProps = {
  open: boolean;
  onClose: () => void;
  title: ReactNode;
  children?: ReactNode;
  footer?: ReactNode;
  /** Back-stack: shows a back button that calls this. */
  onBack?: () => void;
  backLabel?: string;
  /** Width in px (AgentSheet 560, lead 460, deal 480, reader 640, inspector 360). */
  width?: number;
  /** `false`: docked, non-modal (no scrim, no focus trap, no scroll lock); Escape still closes. */
  modal?: boolean;
  initialFocus?: RefObject<HTMLElement | null>;
  className?: string;
};

/** Right sheet with a back stack; swipe right 80px to close on touch. */
export function GDrawer(props: GDrawerProps) {
  const melt = useMeltPhase(props.open);
  if (!melt.mounted) return null;
  return (
    <Layer>
      <DrawerPanel {...props} melting={melt.melting} onMelted={melt.end} />
    </Layer>
  );
}

function DrawerPanel({ open, onClose, title, children, footer, onBack, backLabel = 'Back', width = 480, modal = true, initialFocus, className, melting, onMelted }: GDrawerProps & MeltProps) {
  const titleId = useId();
  const dlg = useDialog({ open: open && modal, onClose, labelledBy: titleId, initialFocus });
  const onMeltEnd = useMeltEnd(dlg.ref, melting, onMelted);
  const onCloseRef = useRef(onClose);
  onCloseRef.current = onClose;
  useEffect(() => {
    if (modal || !open) return;
    return pushEscape(() => onCloseRef.current());
  }, [modal, open]);

  const touch = useRef<{ x: number; y: number } | null>(null);
  const onTouchStart = (e: TouchEvent) => {
    const t = e.touches[0];
    touch.current = t ? { x: t.clientX, y: t.clientY } : null;
  };
  const onTouchEnd = (e: TouchEvent) => {
    const start = touch.current;
    const t = e.changedTouches[0];
    touch.current = null;
    if (start && t && swipeCloses(t.clientX - start.x, t.clientY - start.y)) onClose();
  };

  const panelProps = modal
    ? dlg.props
    : { ref: dlg.ref, role: 'dialog' as const, 'aria-modal': false as const, 'aria-labelledby': titleId };
  return (
    <>
      <div className="g-layer" data-kind="drawer" data-modal={modal ? 'true' : 'false'} data-melting={melting ? '' : undefined}>
        {modal && <div className="g-scrim gl-scrim" aria-hidden onClick={onClose} />}
        <aside
          {...panelProps}
          data-g="GDrawer"
          className={cx('g-drawer', className, melting && 'is-melting')}
          style={{ width: `min(${width}px, 100vw)` }}
          onTouchStart={onTouchStart}
          onTouchEnd={onTouchEnd}
          onAnimationEnd={onMeltEnd}
        >
          <header className="g-drawer-head">
            {onBack && (
              <GButton variant="icon" aria-label={backLabel} onClick={onBack}>
                <ChevronLeft size={16} strokeWidth={1.75} aria-hidden />
              </GButton>
            )}
            <h2 id={titleId} className="g-drawer-title">
              {title}
            </h2>
            <CloseButton onClose={onClose} />
          </header>
          <div className="g-drawer-body">{children}</div>
          {footer != null && <footer className="g-drawer-foot">{footer}</footer>}
        </aside>
      </div>
    </>
  );
}

/* ── GDialog ──────────────────────────────────────────────────────────── */

export type GDialogProps = {
  open: boolean;
  onClose: () => void;
  title: ReactNode;
  children?: ReactNode;
  confirmLabel: string;
  onConfirm: () => void;
  cancelLabel?: string;
  /** `danger` for deletes: the confirm is a danger GButton. */
  tone?: 'default' | 'danger';
  busy?: boolean;
  /** Disable confirm with a reason (e.g. a typed name does not match yet). */
  confirmDisabledReason?: string;
};

/** 420px confirm. Focus starts on Cancel so Enter never confirms by accident. */
export function GDialog(props: GDialogProps) {
  const melt = useMeltPhase(props.open);
  if (!melt.mounted) return null;
  return (
    <Layer>
      <DialogPanel {...props} melting={melt.melting} onMelted={melt.end} />
    </Layer>
  );
}

function DialogPanel({ open, onClose, title, children, confirmLabel, onConfirm, cancelLabel = 'Cancel', tone = 'default', busy = false, confirmDisabledReason, melting, onMelted }: GDialogProps & MeltProps) {
  const titleId = useId();
  const dlg = useDialog({ open, onClose: busy ? () => {} : onClose, labelledBy: titleId });
  const onMeltEnd = useMeltEnd(dlg.ref, melting, onMelted);
  return (
    <>
      <div className="g-layer" data-kind="dialog" data-melting={melting ? '' : undefined}>
        <div className="g-scrim gl-scrim" aria-hidden onClick={busy ? undefined : onClose} />
        <div {...dlg.props} role={tone === 'danger' ? 'alertdialog' : 'dialog'} data-g="GDialog" data-tone={tone} className={cx('g-dialog', melting && 'is-melting')} onAnimationEnd={onMeltEnd}>
          <h2 id={titleId} className="g-dialog-title">
            {title}
          </h2>
          {children != null && <div className="g-dialog-body">{children}</div>}
          <div className="g-dialog-actions">
            <GButton data-dialog-close="" onClick={onClose} disabled={busy}>
              {cancelLabel}
            </GButton>
            <GButton
              variant={tone === 'danger' ? 'danger' : 'ink'}
              onClick={onConfirm}
              busy={busy}
              disabled={Boolean(confirmDisabledReason)}
              disabledReason={confirmDisabledReason}
            >
              {confirmLabel}
            </GButton>
          </div>
        </div>
      </div>
    </>
  );
}

/* ── GPopover ─────────────────────────────────────────────────────────── */

export type PopoverTriggerProps = {
  ref: RefObject<HTMLButtonElement & HTMLAnchorElement>;
  onClick: () => void;
  'aria-expanded': boolean;
  'aria-controls': string;
  'aria-haspopup': 'dialog' | 'menu';
};

export type GPopoverProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Renders the anchor; spread the props on a GButton. */
  trigger: (props: PopoverTriggerProps) => ReactNode;
  children: ReactNode;
  /** Edge the panel aligns to. */
  align?: 'start' | 'end';
  side?: 'bottom' | 'top';
  /** Accessible name of the panel. */
  label: string;
  /** Internal: the menu variant. */
  role?: 'dialog' | 'menu';
  /** Focus this inside the panel on open (default: first focusable). */
  initialFocus?: RefObject<HTMLElement | null>;
  className?: string;
  panelClassName?: string;
};

/**
 * Anchored card: radius 20, --g-card-2, --g-shadow. Opens below its
 * trigger, closes on Escape (top of the stack), outside press or Tab out,
 * and hands focus back to the trigger.
 */
export function GPopover({ open, onOpenChange, trigger, children, align = 'start', side = 'bottom', label, role = 'dialog', initialFocus, className, panelClassName }: GPopoverProps) {
  const id = useId();
  const wrap = useRef<HTMLSpanElement>(null);
  const panel = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement & HTMLAnchorElement>(null);
  const setOpen = useRef(onOpenChange);
  setOpen.current = onOpenChange;

  const close = useCallback((refocus: boolean) => {
    setOpen.current(false);
    if (refocus) triggerRef.current?.focus({ preventScroll: true });
  }, []);

  useEffect(() => {
    if (!open) return;
    const target = initialFocus?.current ?? panel.current?.querySelector<HTMLElement>(role === 'menu' ? '[role="menuitem"]:not([disabled])' : FOCUSABLE_SELECTOR) ?? panel.current;
    target?.focus({ preventScroll: true });
    const off = pushEscape(() => close(true));
    const onDown = (e: PointerEvent) => {
      if (wrap.current && !wrap.current.contains(e.target as Node)) close(false);
    };
    document.addEventListener('pointerdown', onDown);
    return () => {
      off();
      document.removeEventListener('pointerdown', onDown);
    };
  }, [open, close, role, initialFocus]);

  const onKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (e.key !== 'Tab') return;
    // Tab past the last (or before the first) item leaves the popover: close it.
    queueMicrotask(() => {
      if (panel.current && !panel.current.contains(document.activeElement)) setOpen.current(false);
    });
  };

  return (
    <span ref={wrap} data-g={role === 'menu' ? undefined : 'GPopover'} className={cx('g-pop-wrap', className)}>
      {trigger({ ref: triggerRef, onClick: () => onOpenChange(!open), 'aria-expanded': open, 'aria-controls': id, 'aria-haspopup': role })}
      {open && (
        <div
          ref={panel}
          id={id}
          role={role}
          aria-label={label}
          tabIndex={-1}
          data-align={align}
          data-side={side}
          className={cx('g-pop', panelClassName)}
          onKeyDown={onKeyDown}
        >
          {children}
        </div>
      )}
    </span>
  );
}

/* ── GMenu ────────────────────────────────────────────────────────────── */

export type GMenuItem = {
  id: string;
  label: ReactNode;
  onSelect?: () => void;
  /** A menu item that navigates. */
  href?: string;
  icon?: ReactNode;
  danger?: boolean;
  disabled?: boolean;
  disabledReason?: string;
};

export type GMenuProps = {
  items: readonly GMenuItem[];
  trigger: (props: PopoverTriggerProps) => ReactNode;
  label: string;
  align?: 'start' | 'end';
  className?: string;
};

/**
 * Menu on a GPopover: arrows, Home/End, Enter. Only for two or more real
 * actions (§5.4 rule 4: no "…" menu for one).
 */
export function GMenu({ items, trigger, label, align = 'end', className }: GMenuProps) {
  const [open, setOpen] = useState(false);
  const list = useRef<HTMLDivElement>(null);
  const onKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    const els = Array.from(list.current?.querySelectorAll<HTMLElement>('[role="menuitem"]') ?? []);
    const current = els.indexOf(document.activeElement as HTMLElement);
    const next = rovingIndex(e.key, current < 0 ? 0 : current, items.map((i) => Boolean(i.disabled)), 'vertical');
    if (next === null) return;
    e.preventDefault();
    els[next]?.focus();
  };
  const pick = (it: GMenuItem) => {
    setOpen(false);
    it.onSelect?.();
  };
  return (
    <span data-g="GMenu" className={cx('g-menu-wrap', className)}>
      <GPopover open={open} onOpenChange={setOpen} trigger={trigger} label={label} align={align} role="menu" panelClassName="g-menu">
        <div ref={list} onKeyDown={onKeyDown} className="g-menu-list">
          {items.map((it) =>
            it.href && !it.disabled ? (
              <Link key={it.id} href={it.href} role="menuitem" tabIndex={-1} className="g-menu-item" data-danger={it.danger || undefined} onClick={() => pick(it)}>
                {it.icon != null && <span className="g-menu-icon" aria-hidden>{it.icon}</span>}
                {it.label}
              </Link>
            ) : (
              <button
                key={it.id}
                type="button"
                role="menuitem"
                tabIndex={-1}
                className="g-menu-item"
                data-danger={it.danger || undefined}
                disabled={it.disabled}
                title={it.disabled ? it.disabledReason : undefined}
                onClick={() => pick(it)}
              >
                {it.icon != null && <span className="g-menu-icon" aria-hidden>{it.icon}</span>}
                {it.label}
              </button>
            ),
          )}
        </div>
      </GPopover>
    </span>
  );
}

/* ── GTooltip ─────────────────────────────────────────────────────────── */

export const TOOLTIP_DELAY_MS = 120;

export type GTooltipProps = {
  /** The tooltip text. Never the only place a control's name lives. */
  content: ReactNode;
  /** One focusable element; it gets aria-describedby. */
  children: ReactElement;
  side?: 'top' | 'bottom';
  className?: string;
};

/**
 * Non-chart tooltip: a small floating pill after 120ms on hover or focus.
 * Escape hides it without closing the layer underneath (WCAG 1.4.13).
 */
export function GTooltip({ content, children, side = 'top', className }: GTooltipProps) {
  const id = useId();
  const [show, setShow] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const clear = () => {
    if (timer.current) clearTimeout(timer.current);
    timer.current = null;
  };
  const open = () => {
    clear();
    timer.current = setTimeout(() => setShow(true), TOOLTIP_DELAY_MS);
  };
  const hide = () => {
    clear();
    setShow(false);
  };
  useEffect(() => clear, []);
  const child = isValidElement<{ 'aria-describedby'?: string }>(children)
    ? cloneElement(children, { 'aria-describedby': cx(children.props['aria-describedby'], id) })
    : children;
  return (
    <span
      data-g="GTooltip"
      className={cx('g-tip-wrap', className)}
      onPointerEnter={open}
      onPointerLeave={hide}
      onFocus={open}
      onBlur={hide}
      onKeyDown={(e) => {
        if (e.key === 'Escape' && show) {
          e.preventDefault();
          hide();
        }
      }}
    >
      {child}
      <span id={id} role="tooltip" className="g-tip" data-side={side} data-show={show || undefined}>
        {content}
      </span>
    </span>
  );
}
