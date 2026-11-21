'use client';

import { forwardRef, useEffect, useId, useRef, useState, type ButtonHTMLAttributes, type InputHTMLAttributes, type ReactNode, type SelectHTMLAttributes, type TextareaHTMLAttributes } from 'react';
import Link from 'next/link';
import { ArrowLeft, ChevronLeft, ChevronRight, X } from 'lucide-react';
import { cx } from '@/components/g/cx';
import { pushEscape } from '@/lib/escape-stack';
import { useDialog } from '@/components/ui/use-dialog';
import { Layer } from '@/components/g/Overlays';
import { useMeltClose } from '@/lib/use-melt';

/**
 * The Creative Engine's Classic controls (glados-ui SPEC GP1, X5). Under the
 * Classic style the engine renders exactly the markup it always has; these are
 * that markup, moved out of the shared surfaces so the surfaces themselves hold
 * no Classic control class (the GP4 source guard scans them). components/
 * creative-engine/skin.tsx picks between these and the G kit. Nothing here is
 * rendered under the GladOS style.
 */

export type ClassicBtnProps = Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'className'> & {
  tone?: 'default' | 'ink' | 'danger';
  size?: 'md' | 'sm' | 'compact';
  icon?: boolean;
  href?: string;
  /** Next.js client navigation (the page's ‹ AdPilot link); otherwise a plain anchor. */
  link?: boolean;
  external?: boolean;
  download?: string | boolean;
  className?: string;
};

export const ClassicBtn = forwardRef<HTMLButtonElement & HTMLAnchorElement, ClassicBtnProps>(function ClassicBtn(
  { tone = 'default', size = 'md', icon = false, href, link = false, external = false, download, className, type, children, ...rest },
  ref,
) {
  const cls = cx(icon ? 'ae-icon-btn' : 'ae-btn', !icon && tone === 'ink' && 'ae-btn-ink', !icon && tone === 'danger' && 'ae-btn-danger', !icon && size !== 'md' && 'ae-btn-sm', className);
  if (href) {
    const a = { title: rest.title, 'aria-label': rest['aria-label'], onClick: rest.onClick as unknown as React.MouseEventHandler<HTMLAnchorElement> | undefined, className: cls };
    if (link) return <Link ref={ref} href={href} {...a}>{children}</Link>;
    return <a ref={ref} href={href} download={download === true ? '' : download || undefined} target={external ? '_blank' : undefined} rel={external ? 'noreferrer noopener' : undefined} {...a}>{children}</a>;
  }
  return <button ref={ref} type={type ?? 'button'} className={cls} {...rest}>{children}</button>;
});

/** An unstyled button whose look comes from the caller (underlined text, an octane row, a tile). */
export const ClassicTextButton = forwardRef<HTMLButtonElement, ButtonHTMLAttributes<HTMLButtonElement>>(function ClassicTextButton({ children, type, ...rest }, ref) {
  return <button ref={ref} type={type ?? 'button'} {...rest}>{children}</button>;
});

/** A pill that is one choice of several (Connect Arcads: product, UGC path). */
export function ClassicPickPill({ on, onClick, disabled, title, children }: { on: boolean; onClick: () => void; disabled?: boolean; title?: string; children: ReactNode }) {
  return <button type="button" onClick={onClick} disabled={disabled} title={title} aria-pressed={on} className="ae-pill" data-active={on}>{children}</button>;
}

export function ClassicOctaneLink({ cls, href, external, children }: { cls?: string; href: string; external: boolean; children: ReactNode }) {
  return <a className={cls} href={href} target={external ? '_blank' : undefined} rel={external ? 'noreferrer' : undefined}>{children}</a>;
}

export function ClassicDownload({ href, className, children }: { href: string; className?: string; children: ReactNode }) {
  return <a href={href} download className={className}>{children}</a>;
}

export const ClassicPill = forwardRef<HTMLButtonElement, { icon?: ReactNode; label: ReactNode; badge?: ReactNode; active?: boolean; onClick?: () => void; title?: string; 'aria-haspopup'?: 'true' | 'dialog' | 'menu'; 'aria-expanded'?: boolean }>(function ClassicPill(
  { icon, label, badge, active, onClick, title, ...aria },
  ref,
) {
  return (
    <button ref={ref} type="button" className="ae-pill" data-active={active ? 'true' : undefined} title={title} onClick={onClick} {...aria}>
      {icon} {label}
      {badge != null && <span className="ae-pill-badge">{badge}</span>}
    </button>
  );
});

export type ClassicInputProps = Omit<InputHTMLAttributes<HTMLInputElement>, 'size'> & { size?: 'md' | 'lg' | 'compact' };

export const ClassicInput = forwardRef<HTMLInputElement, ClassicInputProps>(function ClassicInput({ size = 'md', className, ...rest }, ref) {
  return <input ref={ref} className={cx('ae-input', size === 'lg' && 'ae-input-lg', className)} {...rest} />;
});

/** An input or textarea drawn only by the caller's classes (inline editors on a card). */
export const ClassicBareInput = forwardRef<HTMLInputElement, Omit<InputHTMLAttributes<HTMLInputElement>, 'size'> & { size?: string }>(function ClassicBareInput({ size: _size, ...rest }, ref) {
  return <input ref={ref} {...rest} />;
});
export const ClassicBareTextarea = forwardRef<HTMLTextAreaElement, TextareaHTMLAttributes<HTMLTextAreaElement>>(function ClassicBareTextarea(props, ref) {
  return <textarea ref={ref} {...props} />;
});

export const ClassicBareSelect = forwardRef<HTMLSelectElement, SelectHTMLAttributes<HTMLSelectElement>>(function ClassicBareSelect(props, ref) {
  return <select ref={ref} {...props} />;
});

export const ClassicTextarea = forwardRef<HTMLTextAreaElement, TextareaHTMLAttributes<HTMLTextAreaElement>>(function ClassicTextarea({ className, ...rest }, ref) {
  return <textarea ref={ref} className={cx('ae-input', className)} {...rest} />;
});

export const ClassicSelect = forwardRef<HTMLSelectElement, SelectHTMLAttributes<HTMLSelectElement>>(function ClassicSelect({ className, ...rest }, ref) {
  return <select ref={ref} className={cx('ae-input ae-select', className)} {...rest} />;
});

/** The angle card's round check: display state inside its card button (the card is the control). */
export function ClassicCheckMark({ on, children, className }: { on: boolean; children?: ReactNode; className?: string }) {
  return <span className={cx('ae-check', className)} data-on={on}>{children}</span>;
}

/** Engine / Runs title tabs (S04 B42): arrows, Home and End; the choice lives in the URL. */
export function ClassicTitleTabs<T extends string>({ tabs, value, onChange, label, panelId }: { tabs: readonly { value: T; label: string }[]; value: T; onChange: (v: T) => void; label: string; panelId: string }) {
  return (
    <div role="tablist" aria-label={label} className="flex items-baseline gap-5" onKeyDown={(e) => {
      if (e.key !== 'ArrowLeft' && e.key !== 'ArrowRight' && e.key !== 'Home' && e.key !== 'End') return;
      e.preventDefault();
      const i = tabs.findIndex((t) => t.value === value);
      const next = e.key === 'Home' ? tabs[0].value : e.key === 'End' ? tabs[tabs.length - 1].value : tabs[(i + (e.key === 'ArrowRight' ? 1 : -1) + tabs.length) % tabs.length].value;
      onChange(next);
      document.getElementById(`ae-tab-${next}`)?.focus();
    }}>
      {tabs.map((t) => (
        <button key={t.value} id={`ae-tab-${t.value}`} type="button" role="tab" aria-selected={value === t.value} aria-controls={panelId} tabIndex={value === t.value ? 0 : -1} onClick={() => onChange(t.value)} className={`ae-title ae-tab text-[28px] ${value === t.value ? '' : '!text-[var(--p-dim)]'}`}>{t.label}</button>
      ))}
    </div>
  );
}

export type ClassicMenuItem = { id: string; label: ReactNode; icon?: ReactNode; danger?: boolean; onSelect: () => void };

/**
 * The campaign H1 actions menu: Escape closes it (shared stack), a click
 * outside too; Up and Down move between items.
 */
export function ClassicActionsMenu({ items, label, triggerIcon }: { items: readonly ClassicMenuItem[]; label: string; triggerIcon: ReactNode }) {
  const [open, setOpen] = useState(false);
  const btn = useRef<HTMLButtonElement>(null);
  const menu = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const off = pushEscape(() => { setOpen(false); btn.current?.focus(); }, { id: 'ae-h1-menu' });
    const onDown = (e: MouseEvent) => {
      const el = e.target as Node;
      if (menu.current?.contains(el) || btn.current?.contains(el)) return;
      setOpen(false);
    };
    document.addEventListener('mousedown', onDown);
    return () => { off(); document.removeEventListener('mousedown', onDown); };
  }, [open]);
  return (
    <>
      <button ref={btn} type="button" onClick={() => setOpen((o) => !o)} aria-label={label} title={label} aria-haspopup="menu" aria-expanded={open} className="ae-icon-btn">{triggerIcon}</button>
      {open && (
        <div ref={menu} role="menu" className="ae-pop ae-pop-menu" onKeyDown={(e) => {
          const els = Array.from(menu.current?.querySelectorAll<HTMLButtonElement>('[role="menuitem"]') ?? []);
          const i = els.indexOf(document.activeElement as HTMLButtonElement);
          if (e.key === 'ArrowDown') { e.preventDefault(); els[(i + 1) % els.length]?.focus(); }
          if (e.key === 'ArrowUp') { e.preventDefault(); els[(i - 1 + els.length) % els.length]?.focus(); }
        }}>
          {items.map((it, i) => (
            <button key={it.id} type="button" role="menuitem" autoFocus={i === 0} onClick={() => { setOpen(false); it.onSelect(); }} className={it.danger ? 'ae-pop-item ae-pop-danger' : 'ae-pop-item'}>{it.icon} {it.label}</button>
          ))}
        </div>
      )}
    </>
  );
}

/** The anchor a Classic popover hangs from. */
export function ClassicPopAnchor({ children, className }: { children: ReactNode; className?: string }) {
  return <span className={cx('ae-pop-anchor', className)}>{children}</span>;
}

/* ── campaign switcher (Classic tree) ───────────────────────────────────── */

/** The switcher panel: the paper popover under Classic; under the kit the same panel on --g-* tokens. */
export function switcherPanelClass(g: boolean): string {
  return g ? 'ce-g-switcher' : 'ae-pop';
}

export function ClassicSwitcherRow({ g, current, children }: { g: boolean; current: boolean; children: ReactNode }) {
  return <li className={g ? 'ce-g-switcher-row' : 'ae-pop-row'} data-current={current}>{children}</li>;
}

export const ClassicSwitcherOpen = forwardRef<HTMLButtonElement, { onClick: () => void; onKeyDown: (e: React.KeyboardEvent) => void; current: boolean; title: string; children: ReactNode }>(function ClassicSwitcherOpen({ onClick, onKeyDown, current, title, children }, ref) {
  return <button type="button" ref={ref} onClick={onClick} onKeyDown={onKeyDown} className="ae-pop-open" aria-current={current ? 'true' : undefined} title={title}>{children}</button>;
});

/* ── dashboard pieces (Classic tree) ────────────────────────────────────── */

/** A whole-card button on paper (angle, rail, pending cards). */
export function ClassicCardButton({ onClick, className, title, children }: { onClick: () => void; className?: string; title?: string; children: ReactNode }) {
  return <button type="button" onClick={onClick} className={className} title={title}>{children}</button>;
}

/** The invisible button over a card that also holds its own control (Ready-for-you cards, F26). */
export function ClassicOverlayButton({ label, onClick }: { label: string; onClick: () => void }) {
  return <button type="button" aria-label={label} className="ae-ready-open" onClick={onClick} />;
}

/** One stat (S07 F1 to F5) as Classic drew it. The numerals come from the caller. */
export function ClassicStat({ lg, sm, badge, lime, title, ariaLabel, empty, onClick }: { lg: ReactNode; sm: ReactNode; badge: string; lime: boolean; title: string; ariaLabel: string; empty: boolean; onClick: () => void }) {
  return (
    <button type="button" className="ae-stat" onClick={onClick} title={title} aria-label={ariaLabel} data-empty={empty || undefined}>
      <span className="ae-stat-num ae-stat-lg">{lg}</span>
      <span className="ae-stat-num ae-stat-sm">{sm}</span>
      <span className={lime ? 'ae-badge-lime ae-stat-badge' : 'ae-tag ae-stat-badge'}>{badge}</span>
    </button>
  );
}

/** The gradient hero button, Classic markup. */
export function ClassicHeroButton({ tone, onClick, className, children }: { tone: 'green' | 'warm' | 'pink'; onClick: () => void; className: string; children: ReactNode }) {
  return <button type="button" onClick={onClick} className={`ae-hero ae-hero-${tone} ${className}`}>{children}</button>;
}

/** A display-only gauge (S07 F27): its label states the value and the scale; the mark sits where the label says. */
export function ClassicRuler({ pct, dark = false, label, tone }: { pct: number; dark?: boolean; label: string; tone?: 'warn' | 'over' }) {
  const n = 40;
  const mark = Math.round(Math.max(0, Math.min(1, pct)) * (n - 1));
  return (
    <div className={`ae-ruler ${dark ? 'dark' : ''}`} role="img" aria-label={label} title={label} data-tone={tone}>
      {Array.from({ length: n }, (_, i) => (<i key={i} className={i === mark ? 'mark' : i % 5 === 0 ? 'tall' : ''} style={{ height: i === mark ? 22 : i % 5 === 0 ? 14 : 8 }} />))}
    </div>
  );
}

/** S04 B11: one dot per pooled ad (cap 192); lit = proven, dim = pooled but not proven. */
export function ClassicDotCloud({ n, lit, label }: { n: number; lit: number; label: string }) {
  return (
    <div className="ae-dotcloud" role="img" aria-label={label} title={label}>
      {Array.from({ length: n }, (_, i) => <span key={i} data-on={i < lit || undefined} />)}
    </div>
  );
}

/** Classic's rings on the console card (decorative there; the kit's GRings is the readout, GP1 FR-GP1.4). */
export function ClassicRings() {
  return (
    <div className="ae-rings" aria-hidden="true">
      {[100, 74, 49].map((s) => (<span key={s} style={{ width: `${s}%`, height: `${s}%` }} />))}
      <i />
    </div>
  );
}

export function ClassicBadge({ children, className, ...data }: { children: ReactNode; className?: string; 'data-status'?: string; 'data-state'?: string; 'data-sample-badge'?: boolean }) {
  return <span className={cx('ae-badge-lime', className)} {...data}>{children}</span>;
}

export function ClassicTag({ children, className, ...data }: { children: ReactNode; className?: string; 'data-dry-tag'?: boolean; 'data-state'?: string }) {
  return <span className={cx('ae-tag', className)} {...data}>{children}</span>;
}

/* ── overlays (Classic tree) ────────────────────────────────────────────── */

/**
 * A paper sheet: the existing surfaces open here, re-skinned by the scope.
 * S04 B37 on SPEC X1: a modal dialog named by its heading, focus on Close,
 * Tab trapped, body scroll locked, focus back on the opener, and Escape
 * closes only the top layer (a drawer opened inside it closes first).
 */
export function ClassicSheet({ title, onClose, backLabel, children }: { title: string; onClose: () => void; backLabel?: string; children: ReactNode }) {
  const titleId = useId();
  // Escape, the scrim and Close melt the face (lib/use-melt); the parent unmounts it once it has melted.
  const dlg = useDialog({ open: true, onClose: () => melt.close(), labelledBy: titleId });
  const melt = useMeltClose(onClose, dlg.ref);
  const close = melt.close;
  // Portalled to <body> (a frosted ancestor traps a fixed layer); the layer carries the paper scope.
  return (
    <Layer>
      <div className="ae-paper ae-sheet-layer fixed inset-0 z-[120] flex items-start justify-center overflow-y-auto p-4" {...melt.layer}>
        <div className="ae-sheet-scrim gl-scrim" aria-hidden onClick={close} />
        <div {...dlg.props} onAnimationEnd={melt.onAnimationEnd} className={`ae-paper ae-sheet ae-card glass-face glass-face--strong relative my-6 w-full max-w-[1200px] p-7 outline-none${melt.melting ? ' is-melting' : ''}`}>
          <div className="flex items-center justify-between gap-3">
            <h2 id={titleId} className="ae-section !text-[30px]">{title}</h2>
            <button type="button" onClick={close} className="ae-btn" data-dialog-close title={backLabel ? `${backLabel} (Esc)` : 'Close (Esc)'}>{backLabel ? <><ChevronLeft className="h-4 w-4" strokeWidth={1.8} /> {backLabel}</> : <><X className="h-4 w-4" strokeWidth={1.8} /> Close</>}</button>
          </div>
          <div className="mt-5 min-h-[60vh]">{children}</div>
          <div className="ae-faint mt-5 flex justify-end"><span className="flex items-center gap-1">Esc closes <ChevronRight className="h-3 w-3" /></span></div>
        </div>
      </div>
    </Layer>
  );
}

/**
 * Right drawer on paper: scrim, panel, Esc closes (S04 B37 on SPEC X1). It is a
 * modal dialog named by its title: focus moves to the close button, Tab stays
 * inside, the body does not scroll behind it, and focus returns to the opener.
 */
export function ClassicDrawer({ title, onClose, children, width = 'max-w-[520px]', back }: { title: string; onClose: () => void; children: ReactNode; width?: string; back?: { label: string; onClick: () => void } }) {
  const titleId = useId();
  // Escape, the scrim and Close melt the face (lib/use-melt); the parent unmounts it once it has melted.
  const dlg = useDialog({ open: true, onClose: () => melt.close(), labelledBy: titleId });
  const melt = useMeltClose(onClose, dlg.ref);
  const close = melt.close;
  // Portalled to <body>: opened from a sheet it would otherwise sit inside that sheet's frosted face.
  return (
    <Layer>
      <div className="ae-paper ae-sheet-layer fixed inset-0 z-[125] flex justify-end" {...melt.layer}>
        <div className="ae-sheet-scrim gl-scrim" aria-hidden onClick={close} />
        <div {...dlg.props} onAnimationEnd={melt.onAnimationEnd} className={`ae-sheet ae-drawer glass-face glass-face--strong relative h-full w-full ${width} overflow-y-auto p-6 outline-none${melt.melting ? ' is-melting' : ''}`}>
          <div className="mb-5 flex items-center justify-between gap-3">
            <span className="flex min-w-0 items-center gap-2">
              {back && <button type="button" onClick={back.onClick} className="ae-btn ae-btn-sm" title={back.label}><ArrowLeft className="h-3.5 w-3.5" strokeWidth={1.8} /> {back.label}</button>}
              <h2 id={titleId} className="ae-title min-w-0 truncate text-[24px]">{title}</h2>
            </span>
            <button type="button" onClick={close} aria-label="Close" title="Close (Esc)" data-dialog-close className="ae-btn ae-btn-sm"><X className="h-4 w-4" strokeWidth={1.8} /></button>
          </div>
          {children}
        </div>
      </div>
    </Layer>
  );
}

/** Hard-delete confirm (S03 FR-6, Q-ENG-4) on the X1 dialog: Cancel has focus, Escape closes. */
export function ClassicConfirmDelete({ title, body, error, busy, onCancel, onConfirm }: { title: ReactNode; body: ReactNode; error: string | null; busy: boolean; onCancel: () => void; onConfirm: () => void }) {
  const titleId = useId();
  // Escape, the scrim and Cancel melt the face (lib/use-melt), never while the delete is in flight.
  const dlg = useDialog({ open: true, onClose: () => cancel(), labelledBy: titleId });
  const melt = useMeltClose(onCancel, dlg.ref);
  const cancel = () => {
    if (!busy) melt.close();
  };
  // Portalled to <body>: a frosted ancestor traps a fixed layer.
  return (
    <Layer>
      <div className="ae-paper ae-sheet-layer fixed inset-0 z-[140] flex items-center justify-center p-4" {...melt.layer}>
        <div className="ae-sheet-scrim gl-scrim" aria-hidden onClick={cancel} />
        <div {...dlg.props} onAnimationEnd={melt.onAnimationEnd} className={`ae-sheet ae-card glass-face glass-face--strong relative w-full max-w-[460px] p-6${melt.melting ? ' is-melting' : ''}`}>
          <h2 id={titleId} className="ae-title text-[22px]">{title}</h2>
          <p className="mt-3 text-[13px] leading-snug text-[var(--p-muted)]">{body}</p>
          {error && <p role="alert" className="mt-3 text-[12.5px] text-[var(--p-red)]">{error}</p>}
          <div className="mt-6 flex justify-end gap-2">
            <button type="button" data-dialog-close onClick={cancel} disabled={busy} className="ae-btn">Cancel</button>
            <button type="button" onClick={onConfirm} disabled={busy} className="ae-btn ae-btn-danger">{busy ? 'Deleting' : 'Delete'}</button>
          </div>
        </div>
      </div>
    </Layer>
  );
}
