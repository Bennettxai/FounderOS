'use client';

import { useId, useState, type ReactNode } from 'react';
import { Bot, X } from 'lucide-react';
import { useDialog } from '@/components/ui/use-dialog';
import { useMeltEnd, useMeltPhase } from '@/lib/use-melt';
import { cx } from './format';

/**
 * Bennett Legacy overlays and small parts the page trees share: the modal
 * sheet (FO v1 dialog: slab corners, hairline, mono), the agent face and the
 * on/off switch. Colours from --l-* only; every element carries data-l.
 */

/**
 * Modal over a scrim: focus trapped and returned, Esc closes (components/ui/use-dialog). The scrim
 * is the face's sibling, never its parent (glass brief §3.2: a frosting parent kills the face's
 * own frost): it frosts and dims the page, closes on a press, and the face sits above it. On
 * close the dialog releases at once and the face melts out (lib/use-melt).
 */
export function LDialog({
  open = true,
  onClose,
  title,
  sub,
  children,
  footer,
  width = 520,
}: {
  open?: boolean;
  onClose: () => void;
  title: ReactNode;
  sub?: ReactNode;
  children?: ReactNode;
  footer?: ReactNode;
  width?: number;
}) {
  const titleId = useId();
  const dlg = useDialog({ open, onClose, labelledBy: titleId });
  const melt = useMeltPhase(open);
  const onMeltEnd = useMeltEnd(dlg.ref, melt.melting, melt.end);
  if (!melt.mounted) return null;
  return (
    <div data-l="DialogLayer" className="l-layer" data-melting={melt.melting ? '' : undefined}>
      <div data-l="DialogScrim" className="l-scrim gl-scrim" aria-hidden onClick={onClose} />
      <div {...dlg.props} data-l="Dialog" className={cx('l-dialog glass-face--strong', melt.melting && 'is-melting')} style={{ maxWidth: width }} onAnimationEnd={onMeltEnd}>
        <div className="l-dialog-head">
          <div className="min-w-0">
            <h2 id={titleId} className="l-dialog-title">
              {title}
            </h2>
            {sub && <div className="l-dialog-sub">{sub}</div>}
          </div>
          <button type="button" data-dialog-close="" data-l="Pressable" data-lens="c" className="l-pressable is-dark l-btn-icon l-btn-icon-sm" onClick={onClose} aria-label="Close">
            <X size={14} strokeWidth={1.7} aria-hidden />
          </button>
        </div>
        <div className="l-dialog-body">{children}</div>
        {footer && <div className="l-dialog-foot">{footer}</div>}
      </div>
    </div>
  );
}

/** An agent's photo, or a stroke glyph in a hairline ring when it has none (never an emoji or a fake face). */
export function LAvatar({ src, name, size = 28 }: { src: string | null; name?: string; size?: number }) {
  const [failed, setFailed] = useState(false);
  if (!src || failed) {
    return (
      <span data-l="Avatar" className="l-avatar" style={{ width: size, height: size }} aria-hidden>
        <Bot size={Math.max(11, Math.round(size * 0.5))} strokeWidth={1.6} />
      </span>
    );
  }
  return (
    // eslint-disable-next-line @next/next/no-img-element
    <img data-l="Avatar" src={src} alt={name ? `${name} avatar` : ''} width={size} height={size} className="l-avatar is-photo" style={{ width: size, height: size }} onError={() => setFailed(true)} />
  );
}

/** The FO knob switch (autopilot): a 36×20 track, the knob slides 200 ms. */
export function LSwitch({
  on,
  onToggle,
  label,
  disabled,
  title,
  className,
}: {
  on: boolean;
  onToggle: () => void;
  label: string;
  disabled?: boolean;
  title?: string;
  className?: string;
}) {
  return (
    <button type="button" role="switch" aria-checked={on} aria-label={label} disabled={disabled} title={title} data-l="Switch" data-on={on ? 'true' : 'false'} className={cx('l-switch', className)} onClick={onToggle}>
      <i aria-hidden />
    </button>
  );
}
