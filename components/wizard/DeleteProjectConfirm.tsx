'use client';

import { useEffect, useId, useRef, useState } from 'react';
import { Trash2 } from 'lucide-react';
import { useDialog } from '@/components/ui/use-dialog';

/**
 * Delete project, behind a confirm (S04 F-Z3, H30). Deleting a project drops
 * its whole history and cannot be undone, so the trash button opens a small
 * anchored confirm instead of deleting on one click. Cancel has focus first;
 * Escape and a click outside close it without deleting. Disabled while a
 * send is pending (S04 edge 13).
 */
export function DeleteProjectConfirm({
  onDelete,
  disabled = false,
}: {
  onDelete: () => void;
  disabled?: boolean;
}) {
  const [open, setOpen] = useState(false);
  const cancelRef = useRef<HTMLButtonElement>(null);
  const headingId = useId();
  const close = () => setOpen(false);
  const dialog = useDialog<HTMLDivElement>({
    open,
    onClose: close,
    labelledBy: headingId,
    initialFocus: cancelRef,
    lockScroll: false,
  });
  // A press outside closes it (Escape is the dialog's). No fixed click-away: under a frosted ancestor it
  // would cover that ancestor instead of the page.
  const root = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const onDown = (e: PointerEvent) => {
      if (root.current && !root.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('pointerdown', onDown);
    return () => document.removeEventListener('pointerdown', onDown);
  }, [open]);

  return (
    <div ref={root} className="relative">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        disabled={disabled}
        aria-label="Delete project"
        aria-haspopup="dialog"
        aria-expanded={open}
        title={disabled ? 'Wait for the reply before deleting' : 'Delete project'}
        className="wz-icon-btn disabled:opacity-40"
      >
        <Trash2 className="h-3.5 w-3.5" />
      </button>
      {open && (
        <div
          ref={dialog.ref}
          role="alertdialog"
          aria-modal="true"
          aria-labelledby={headingId}
          tabIndex={-1}
          className="wz-confirm c-pop absolute right-0 top-full z-50 mt-2 w-[260px] p-3.5"
        >
          <div id={headingId} className="text-[13px] text-os-text">
            Delete this project and its history?
          </div>
          <div className="mt-1 text-[12px] leading-snug text-os-dim">This cannot be undone.</div>
          <div className="mt-3 flex justify-end gap-2">
            <button ref={cancelRef} type="button" onClick={close} className="c-btn h-8 px-3 text-[12.5px]">
              Cancel
            </button>
            <button
              type="button"
              onClick={() => {
                close();
                onDelete();
              }}
              className="c-btn wz-confirm-danger h-8 px-3 text-[12.5px]"
            >
              Delete
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
