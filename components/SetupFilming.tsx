'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Clapperboard, Lock } from 'lucide-react';

type State = { filming: boolean; pinned: boolean; dataDir: string };

/**
 * Filming mode switch.
 *
 * Flipping this changes WHICH directory every store reads — data/ or
 * demo-data/. It never seeds, copies or deletes anything, so real finance
 * history cannot be lost by filming. On a pinned host (GLADOS_DATA_DIR set,
 * which the always-on mac mini does) the switch is disabled outright.
 */
export function SetupFilming({ filming, pinned, dataDir }: State) {
  const { busy, error, toggle } = useFilmingToggle(filming);
  return <FilmingClassic filming={filming} pinned={pinned} dataDir={dataDir} busy={busy} error={error} toggle={toggle} />;
}

/** Switch state and action, shared by both trees of /setup (GladOS: components/setup/SetupG.tsx). */
export function useFilmingToggle(filming: boolean) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const toggle = async () => {
    setBusy(true);
    setError(null);
    try {
      const res = await fetch('/api/settings/filming', {
        method: 'POST',
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ filming: !filming }),
      });
      const body = (await res.json()) as { ok: boolean; error?: string };
      if (!body.ok) setError(body.error ?? 'Could not change mode');
      router.refresh();
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  };
  return { busy, error, toggle };
}

function FilmingClassic({ filming, pinned, dataDir, busy, error, toggle }: State & { busy: boolean; error: string | null; toggle: () => void }) {
  return (
    <div className="glass-well flex items-start justify-between gap-4 px-4 py-3">
      <div className="min-w-0">
        <div className="flex items-center gap-2 text-[12.5px] text-os-text">
          <Clapperboard size={14} className="shrink-0" />
          Filming mode
          {filming ? <span className="dot warn" /> : null}
        </div>
        <p className="mt-1 text-[11.5px] leading-relaxed text-os-muted">
          {filming
            ? 'Reading the sample dataset. Nothing on screen is real, and nothing written here reaches your live stores.'
            : 'Reading your real data. Turn this on before you record anything.'}
        </p>
        <p className="mt-1 font-mono text-[10.5px] text-os-muted [overflow-wrap:anywhere]">{dataDir}</p>
        {error ? <p className="mt-1 text-[11.5px] text-os-warn">{error}</p> : null}
      </div>

      {pinned ? (
        <span className="glass-chip inline-flex shrink-0 items-center gap-1.5 rounded-full px-3 py-1 text-[11.5px] text-os-muted">
          <Lock size={12} /> Pinned by host
        </span>
      ) : (
        <button
          onClick={toggle}
          disabled={busy}
          className="inline-flex shrink-0 items-center gap-1.5 rounded-[6px] border border-os-border-strong bg-os-surface2 px-3 py-1 text-[11.5px] text-os-muted transition-colors hover:bg-os-surface3 hover:text-os-text disabled:opacity-50"
        >
          {filming ? 'Switch to real data' : 'Switch to sample data'}
        </button>
      )}
    </div>
  );
}
