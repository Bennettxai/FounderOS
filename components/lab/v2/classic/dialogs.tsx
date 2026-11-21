'use client';
/**
 * Classic dialogs: the generic modal, CapDialog (01 4.4: pass the cap once, everywhere), Confirm, and
 * OutsideScoreDialog (01 12.4). The cap plumbing is shared: `ClassicCapHost` registers this tree's
 * CapDialog with the shared `LabCapProvider`, so a 402 from any `useLabPost` call under a Classic page
 * opens it, exactly as GladOS's `LabCapHost` does. The dialogs themselves stay per tree because each tree
 * draws its own control set (02); the GladOS ones are in g/Dialogs.tsx.
 */
import { useCallback, useEffect, useId, useRef, useState, useSyncExternalStore, type ReactNode } from 'react';
import { createPortal } from 'react-dom';
import { Glass, RoundButton } from '@/components/lab/v2/shared';
import { CLASSIC_COPY as C } from '@/lib/lab/view/classic-copy';
import { LabCapProvider, useLabPost, type CapRequest } from '@/components/lab/v2/shared/useLabPost';

const FOCUSABLE = 'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

const noSubscribe = () => () => {};

/**
 * Where a Lab layer renders: the Lab root (`.lab[data-lab-root]`, which scopes the --lab-* tokens and
 * never frosts), beside the page, never inside a frosted Glass card (a frosted ancestor is the containing
 * block of a fixed layer). Nothing on the server or during hydration, like the kit's Layer.
 */
function useLabLayerRoot(): Element | null {
  return useSyncExternalStore(noSubscribe, () => document.querySelector('.lab[data-lab-root]') ?? document.body, () => null);
}

export function Dialog({ open, onClose, title, children, footer, wide }: { open: boolean; onClose: () => void; title: ReactNode; children?: ReactNode; footer?: ReactNode; wide?: boolean }) {
  const ref = useRef<HTMLDivElement>(null);
  const back = useRef<HTMLElement | null>(null);
  const tid = useId();
  const root = useLabLayerRoot();
  useEffect(() => {
    if (!open || !root) return;
    back.current = document.activeElement as HTMLElement | null;
    const first = ref.current?.querySelector<HTMLElement>('[data-autofocus]') ?? ref.current?.querySelector<HTMLElement>(FOCUSABLE);
    first?.focus();
    return () => back.current?.focus?.({ preventScroll: true });
  }, [open, root]);
  if (!open || !root) return null;
  return createPortal(
    <div className="lab2c-dialog-layer" data-lab="Dialog">
      <div className="lab2-scrim" onClick={onClose} aria-hidden="true" />
      <Glass
        as="div"
        tone="solid"
        radius="sheet"
        priority={1}
        className="lab2c-dialog"
        data-wide={wide ? '' : undefined}
        role="dialog"
        aria-modal="true"
        aria-labelledby={tid}
        ref={ref}
        onKeyDown={(e) => {
          if (e.key === 'Escape') { e.stopPropagation(); onClose(); }
          if (e.key !== 'Tab' || !ref.current) return;
          const els = [...ref.current.querySelectorAll<HTMLElement>(FOCUSABLE)];
          if (!els.length) return;
          if (e.shiftKey && document.activeElement === els[0]) { e.preventDefault(); els[els.length - 1].focus(); }
          else if (!e.shiftKey && document.activeElement === els[els.length - 1]) { e.preventDefault(); els[0].focus(); }
        }}
      >
        <header className="lab2c-dialog-head">
          <h2 id={tid} className="lab2c-dialog-title" data-lab-ink="">{title}</h2>
          <RoundButton label={C.close} icon="close" onClick={onClose} />
        </header>
        <div className="lab2c-dialog-body">{children}</div>
        {footer && <footer className="lab2c-dialog-foot">{footer}</footer>}
      </Glass>
    </div>,
    root,
  );
}

export function Btn({ children, onClick, primary, disabled, title, type = 'button', busy, ...rest }: { children: ReactNode; onClick?: () => void; primary?: boolean; disabled?: boolean; title?: string; type?: 'button' | 'submit'; busy?: boolean } & { [k: `data-${string}`]: string | undefined }) {
  return (
    <button
      type={type}
      className={primary ? 'lab2c-btn lab2c-btn-primary' : 'lab2c-btn'}
      onClick={disabled ? undefined : onClick}
      aria-disabled={disabled ? 'true' : undefined}
      aria-busy={busy ? 'true' : undefined}
      title={title}
      {...rest}
    >
      {children}
    </button>
  );
}

export function ConfirmDialog({ open, title, body, confirmLabel, onConfirm, onClose, cancelLabel }: { open: boolean; title: ReactNode; body?: ReactNode; confirmLabel: string; cancelLabel?: string; onConfirm: () => void; onClose: () => void }) {
  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={title}
      footer={
        <>
          <Btn onClick={onClose} data-autofocus="">{cancelLabel ?? C.dialog.cancel}</Btn>
          <Btn primary onClick={onConfirm}>{confirmLabel}</Btn>
        </>
      }
    >
      {body && <p className="lab2c-dialog-text" data-lab-ink="">{body}</p>}
    </Dialog>
  );
}

/** 402 from any write: shows the server's capAsk, re-posts with override:true. A cap of 0 (an off line) has no pass. */
export function CapDialog({ ask, onClose, onPass }: { ask: Record<string, unknown> | null; onClose: () => void; onPass: () => void }) {
  const msg = typeof ask?.capAsk === 'string' ? ask.capAsk : null;
  const off = ask?.reason === 'off' || ask?.cap === 0;
  return (
    <Dialog
      open={!!ask}
      onClose={onClose}
      title={C.dialog.capTitle}
      footer={
        <>
          <Btn onClick={onClose} data-autofocus="">{C.dialog.cancel}</Btn>
          {!off && <Btn primary onClick={onPass}>{C.dialog.capPass}</Btn>}
        </>
      }
    >
      {msg && <p className="lab2c-dialog-text" data-lab-ink="">{msg}</p>}
    </Dialog>
  );
}

/** Wraps a Classic page so every 402 under it opens the Classic CapDialog (01 4.4). Adds no DOM. */
export function ClassicCapHost({ children }: { children: ReactNode }) {
  const [req, setReq] = useState<CapRequest | null>(null);
  const openCap = useCallback((r: CapRequest) => setReq(r), []);
  return (
    <LabCapProvider openCap={openCap}>
      {children}
      <CapDialog
        ask={req ? { capAsk: req.capAsk } : null}
        onClose={() => setReq(null)}
        onPass={() => { const r = req; setReq(null); void r?.retry(); }}
      />
    </LabCapProvider>
  );
}

const TOOLS = [
  { id: 'higgsfield', label: 'Higgsfield' },
  { id: 'opusclip', label: 'OpusClip' },
];

/** 01 12.4: log an outside tool's score against a draft script; the server stores no band. */
export function OutsideScoreDialog({ open, onClose, scripts, readOnly }: { open: boolean; onClose: () => void; scripts: { id: string; label: string }[]; readOnly: string | null }) {
  const { post, busy } = useLabPost();
  const [scriptId, setScript] = useState(scripts[0]?.id ?? '');
  const [tool, setTool] = useState('higgsfield');
  const [other, setOther] = useState('');
  const [score, setScore] = useState('');
  const [outOf, setOutOf] = useState('100');
  const slug = tool === 'other' ? other.toLowerCase().replace(/[^a-z0-9_]/g, '') : tool;
  const ok = !readOnly && scriptId && slug.length >= 2 && slug.length <= 24 && score !== '' && Number(outOf) > 0;
  const submit = async () => {
    if (!ok) return;
    const r = await post('/api/lab/predictions', {
      action: 'log_external', scriptId, tool: slug, toolLabel: tool === 'other' ? other : TOOLS.find((t) => t.id === tool)?.label, score: Number(score), scaleMax: Number(outOf),
    }, { key: 'outside', quiet: false });
    if (r.ok) onClose();
  };
  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={C.dialog.outsideTitle}
      footer={
        <>
          <Btn onClick={onClose}>{C.dialog.cancel}</Btn>
          <Btn primary disabled={!ok} title={readOnly ?? undefined} busy={busy === 'outside'} onClick={submit}>{C.dialog.outsideSubmit}</Btn>
        </>
      }
    >
      <form className="lab2c-form" onSubmit={(e) => { e.preventDefault(); void submit(); }}>
        <label className="lab2c-field">
          <span className="lab2c-field-label" data-lab-ink="">{C.dialog.script}</span>
          <select className="lab2c-input" value={scriptId} onChange={(e) => setScript(e.target.value)} disabled={!!readOnly} data-autofocus="">
            {scripts.map((s) => <option key={s.id} value={s.id}>{s.label}</option>)}
          </select>
        </label>
        <label className="lab2c-field">
          <span className="lab2c-field-label" data-lab-ink="">{C.dialog.tool}</span>
          <select className="lab2c-input" value={tool} onChange={(e) => setTool(e.target.value)} disabled={!!readOnly}>
            {TOOLS.map((t) => <option key={t.id} value={t.id}>{t.label}</option>)}
            <option value="other">{C.dialog.other}</option>
          </select>
        </label>
        {tool === 'other' && (
          <label className="lab2c-field">
            <span className="lab2c-field-label" data-lab-ink="">{C.dialog.other}</span>
            <input className="lab2c-input" value={other} maxLength={24} onChange={(e) => setOther(e.target.value)} />
          </label>
        )}
        <span className="lab2c-field-row">
          <label className="lab2c-field">
            <span className="lab2c-field-label" data-lab-ink="">{C.dialog.score}</span>
            <input className="lab2c-input" inputMode="decimal" value={score} onChange={(e) => setScore(e.target.value.replace(/[^0-9.]/g, ''))} disabled={!!readOnly} />
          </label>
          <label className="lab2c-field">
            <span className="lab2c-field-label" data-lab-ink="">{C.dialog.outOf}</span>
            <input className="lab2c-input" inputMode="decimal" value={outOf} onChange={(e) => setOutOf(e.target.value.replace(/[^0-9.]/g, ''))} disabled={!!readOnly} />
          </label>
        </span>
        {readOnly && <p className="lab2c-dialog-text" data-lab-ink="">{readOnly}</p>}
      </form>
    </Dialog>
  );
}
