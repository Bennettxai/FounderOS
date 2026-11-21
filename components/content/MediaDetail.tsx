'use client';

import { useEffect } from 'react';
import Image from 'next/image';
import { Download, X } from 'lucide-react';
import type { TaskSpec } from '@/lib/content-schema';
import { formatTimestamp, modelDisplayName } from '@/lib/content-feed';
import { useMeltClose } from '@/lib/use-melt';
import { Layer } from '@/components/g/Overlays';

/**
 * The detail overlay shared by Production's delivered cards and the
 * Library's Generated cards: full-size output, the complete spec pretty-
 * printed (receipts for the render), timestamps, campaign, credits, and a
 * download link. Download lives here and only here — the card action rows
 * stay to the four verbs.
 */
export function MediaDetailOverlay({
  title,
  outputRef,
  kind,
  spec,
  model,
  createdAt,
  deliveredAt,
  campaignName,
  credits,
  onClose,
  actions,
}: {
  title: string;
  outputRef: string;
  kind: 'image' | 'video';
  spec?: TaskSpec;
  model: string;
  createdAt: string;
  /** Set only when the caller has the manifest asset's own delivery timestamp. */
  deliveredAt?: string | null;
  campaignName: string;
  credits: number;
  onClose: () => void;
  actions?: React.ReactNode;
}) {
  // Close melts the face (lib/use-melt); the page unmounts it once it has melted.
  const melt = useMeltClose(onClose);
  const close = melt.close;
  useEffect(() => {
    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') close();
    }
    window.addEventListener('keydown', onKey);
    return () => {
      document.body.style.overflow = prevOverflow;
      window.removeEventListener('keydown', onKey);
    };
  }, [close]);

  // Portalled to <body>: under a frosted ancestor the fixed layer would be trapped.
  return (
    <Layer>
      <div className="fixed inset-0 z-[100] flex items-center justify-center p-6" role="presentation" {...melt.layer}>
        {/* The scrim is the face's sibling (glass brief §3.2): it frosts and dims the page; the face frosts on its own and melts on close. */}
        <div className="gl-scrim" aria-hidden onClick={close} />
        <div
          ref={melt.faceRef}
          onAnimationEnd={melt.onAnimationEnd}
          className={`glass-panel glass-face--strong gl-materialise relative flex max-h-[88vh] w-full max-w-4xl flex-col overflow-hidden${melt.melting ? ' is-melting' : ''}`}
          role="dialog"
          aria-modal="true"
          aria-label={title}
        >
          <button
            type="button"
            onClick={close}
            aria-label="Close"
            className="absolute right-4 top-4 z-10 grid h-8 w-8 place-items-center rounded-full border border-[rgb(var(--mist-rgb) / 0.14)] bg-black/30 text-os-dim transition-colors hover:text-os-text"
          >
            <X className="h-4 w-4" />
          </button>
          <div className="relative min-h-[280px] flex-1 shrink-0 bg-black/40">
            {kind === 'image' ? (
              <Image src={outputRef} alt={title} fill sizes="90vw" className="object-contain" />
            ) : (
              // eslint-disable-next-line jsx-a11y/media-has-caption
              <video src={outputRef} controls autoPlay className="h-full max-h-[56vh] w-full object-contain" />
            )}
          </div>
          <div className="flex-1 overflow-y-auto border-t border-[var(--hairline)]">
            <div className="flex flex-wrap items-center justify-between gap-3 px-5 py-3.5">
              <span className="min-w-0 truncate text-[13px] text-os-text">{title}</span>
              <div className="flex shrink-0 items-center gap-2">
                {actions}
                <a
                  href={outputRef}
                  download
                  className="c-btn"
                >
                  <Download className="h-3.5 w-3.5" strokeWidth={1.7} />
                  Download
                </a>
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4 border-t border-[var(--hairline)] px-5 py-4 sm:grid-cols-4">
              <div>
                <div className="glass-label">Model</div>
                <p className="mt-1.5 font-mono text-[12px] text-os-muted">{modelDisplayName(model)}</p>
              </div>
              <div>
                <div className="glass-label">Campaign</div>
                <p className="mt-1.5 text-[12.5px] text-os-muted">{campaignName}</p>
              </div>
              <div>
                <div className="glass-label">Credits</div>
                <p className="mt-1.5 font-mono text-[12px] tabular-nums text-os-muted">~{credits}</p>
              </div>
              <div>
                <div className="glass-label">Created</div>
                <p className="mt-1.5 font-mono text-[12px] text-os-muted">{formatTimestamp(createdAt)}</p>
              </div>
              {deliveredAt && (
                <div>
                  <div className="glass-label">Delivered</div>
                  <p className="mt-1.5 font-mono text-[12px] text-os-muted">{formatTimestamp(deliveredAt)}</p>
                </div>
              )}
            </div>
            {spec && (
              <div className="border-t border-[var(--hairline)] px-5 py-4">
                <div className="glass-label mb-2">Spec</div>
                <pre className="glass-chip max-h-64 overflow-auto whitespace-pre-wrap break-words rounded-[10px] px-3.5 py-3 font-mono text-[11px] leading-relaxed text-os-muted">
                  {JSON.stringify(spec, null, 2)}
                </pre>
              </div>
            )}
          </div>
        </div>
      </div>
    </Layer>
  );
}
