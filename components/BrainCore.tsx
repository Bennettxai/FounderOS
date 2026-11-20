'use client';

import { useEffect, useState } from 'react';
import { SetupLink } from '@/components/SetupLink';
import { Search, X, Zap } from 'lucide-react';
import Link from 'next/link';
import type { BrainCluster } from '@/lib/brain-viz';
import { BrainViz } from '@/components/BrainViz';
import { BrainQuery } from '@/components/BrainQuery';

type DoctorCheck = { name: string; status: string; message: string };
type Doctor = {
  connected: boolean;
  healthScore: number | null;
  detail: string;
  checks: DoctorCheck[];
};

const CHECK_DOT: Record<string, string> = { ok: 'ok', warn: 'warn', error: 'err' };

/**
 * The knowledge-core radar, made interactive: its central health gauge is a
 * button. Clicking it opens a pop-out with doctor health/warnings and the
 * hybrid-search prompt. (Brain-dump capture is its own section on the page,
 * between the core and the life map.) The SVG itself is unchanged.
 */
export function BrainCore({
  clusters,
  health,
  doctor,
  fallbackActive,
  version = null,
  supabasePages = null,
  supabaseState = null,
}: {
  clusters: BrainCluster[];
  health: number | null;
  doctor: Doctor;
  fallbackActive: boolean;
  version?: string | null;
  supabasePages?: number | null;
  supabaseState?: string | null;
}) {
  const [open, setOpen] = useState(false);
  const warnings = doctor.checks.filter((c) => c.status !== 'ok');

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && setOpen(false);
    window.addEventListener('keydown', onKey);
    const prev = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      window.removeEventListener('keydown', onKey);
      document.body.style.overflow = prev;
    };
  }, [open]);

  return (
    <>
      <div className="relative">
        <BrainViz clusters={clusters} health={health} version={version} supabasePages={supabasePages} supabaseState={supabaseState} />
        {/* Hotspot over the central health gauge (~50%/50% of the SVG box). */}
        <button
          type="button"
          onClick={() => setOpen(true)}
          aria-label="Open Optimal Engine doctor and search"
          className="group absolute left-1/2 top-1/2 grid h-[27%] w-[27%] -translate-x-1/2 -translate-y-1/2 place-items-end justify-center rounded-full pb-[6%] outline-none ring-os-accent/50 focus-visible:ring-2"
        >
          <span className="pointer-events-none flex translate-y-3 items-center gap-1.5 rounded-[6px] border border-os-border-strong bg-os-surface2 px-2 py-0.5 text-[11.5px] text-os-muted opacity-0 transition-opacity duration-150 group-hover:opacity-100 group-focus-visible:opacity-100">
            <Search className="h-2.5 w-2.5" strokeWidth={1.7} /> Doctor · search
          </span>
        </button>
      </div>

      {open && (
        <>
        {/* The scrim is a sibling, not the panel's parent: a glass face never sits inside a blurred or fading element. */}
        <div aria-hidden className="overlay-in fixed inset-0 z-50 bg-black/70 backdrop-blur-sm" />
        <div
          className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto p-4 sm:p-8"
          onClick={() => setOpen(false)}
        >
          <div
            className="panel-in c-pop mb-[6vh] mt-[5vh] w-full max-w-2xl"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="sticky top-0 flex items-center justify-between rounded-t-lg-t border-b border-[var(--elevated-border)] bg-[var(--elevated)] px-5 py-3.5">
              <div className="flex items-center gap-3">
                <span className="text-[12.5px] text-os-dim">Optimal Engine</span>
                <span className="text-sm font-medium">Doctor · search</span>
              </div>
              <button
                type="button"
                onClick={() => setOpen(false)}
                aria-label="Close"
                className="grid h-7 w-7 place-items-center rounded-sm-t border border-os-border text-os-muted transition-colors hover:border-os-border-strong hover:text-os-text"
              >
                <X className="h-3.5 w-3.5" />
              </button>
            </div>

            <div className="space-y-6 px-5 py-5">
              {/* Doctor — engine doctor --json --fast */}
              <div>
                <div className="mb-2 flex items-baseline justify-between">
                  <span className="text-[12.5px] font-medium text-os-muted">
                    Doctor — warnings
                  </span>
                  <span className="text-[11.5px] text-os-dim">
                    {doctor.connected ? `${warnings.length} of ${doctor.checks.length} checks` : 'offline'}
                  </span>
                </div>
                <div className="rounded-md-t border border-os-border bg-os-surface px-4 py-3">
                  {doctor.connected && warnings.length === 0 && doctor.checks.length > 0 && (
                    <div className="flex items-center gap-2.5 text-[12.5px] text-os-muted">
                      <span className="dot ok" /> All {doctor.checks.length} checks green · health {doctor.healthScore ?? '—'}/100
                    </div>
                  )}
                  {!doctor.connected && (
                    <div className="flex items-center gap-2.5 text-[12.5px] text-os-muted">
                      <span className="dot err" /> Doctor offline
                      <SetupLink href="/setup" hint={doctor.detail} className="ml-auto" />
                    </div>
                  )}
                  {warnings.length > 0 && (
                    <ul className="space-y-3">
                      {warnings.map((check) => (
                        <li key={check.name} className="flex items-start gap-3.5 text-[12.5px]">
                          <span className={`dot mt-1 ${CHECK_DOT[check.status] ?? 'err'}`} />
                          <span className="text-os-muted">
                            <span className="font-mono text-os-text">{check.name}</span> — {check.message}
                          </span>
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              </div>

              {/* Hybrid search */}
              <div className="border-t border-os-border pt-5">
                <div className="mb-2 text-[12.5px] font-medium text-os-muted">
                  Hybrid search
                </div>
                <BrainQuery fallbackActive={fallbackActive} />
                {fallbackActive && (
                  <div className="mt-3.5 flex items-center gap-3.5 rounded-md-t border border-os-border bg-os-surface px-4 py-3.5">
                    <Zap className="h-3.5 w-3.5 shrink-0 text-os-warn" strokeWidth={1.7} />
                    <div className="flex-1 text-[13px] text-os-muted">
                      Supabase free tier is paused — hybrid queries fall back to local grep until revived.
                    </div>
                    <Link
                      href="/roadmap"
                      className="shrink-0 rounded-sm-t border border-os-border-strong bg-os-surface2 px-2.5 py-1 text-[12.5px] transition-colors hover:bg-os-surface3"
                    >
                      Roadmap →
                    </Link>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
        </>
      )}
    </>
  );
}
