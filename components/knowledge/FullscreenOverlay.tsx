'use client';

/**
 * The one full-screen affordance the stacked knowledge page uses for a view
 * that would otherwise force an inner scrollbar into a section (the cluster
 * matrix today; anything else that turns out to be "inherently huge"
 * tomorrow). Fixed inset-0, Esc closes, body scroll locked — the overlay is
 * the only place on this page allowed to scroll on its own.
 */
import { useEffect } from 'react';
import { Minimize2 } from 'lucide-react';
import { Layer } from '@/components/g/Overlays';

export function FullscreenOverlay({
  title,
  onClose,
  children,
}: {
  title: string;
  onClose: () => void;
  children: React.ReactNode;
}) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && onClose();
    window.addEventListener('keydown', onKey);
    const prev = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      window.removeEventListener('keydown', onKey);
      document.body.style.overflow = prev;
    };
  }, [onClose]);

  // Portalled to <body> (a frosted ancestor would trap it). It carries the page's token scope (.kv-page:
  // Legacy re-points the --seq/--cat ramps the matrix paints with) now that it renders outside the page.
  return (
    <Layer>
      <div className="kv-page fixed inset-0 z-[130] flex flex-col" style={{ background: 'var(--bg)' }}>
        <div className="flex shrink-0 items-center justify-between border-b border-os-border px-5 py-3">
          <span className="text-[13px] text-os-text">{title}</span>
          <button
            onClick={onClose}
            className="flex items-center gap-2 rounded-full border px-3 py-1.5 text-[11px] text-os-accent transition-colors hover:bg-[var(--accent-soft)]"
            style={{ borderColor: 'var(--accent-line)' }}
          >
            <Minimize2 className="h-3.5 w-3.5" /> Exit (esc)
          </button>
        </div>
        <div className="min-h-0 flex-1">{children}</div>
      </div>
    </Layer>
  );
}
