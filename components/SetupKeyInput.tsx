'use client';

import { useState } from 'react';

/** Key entry state and save, shared by both trees of /setup (GladOS: components/setup/SetupG.tsx). */
export function useKeySave(envVar: string, present: boolean) {
  const [value, setValue] = useState('');
  const [state, setState] = useState<'idle' | 'saving' | 'saved' | 'error'>(present ? 'saved' : 'idle');
  const [error, setError] = useState('');

  const save = async () => {
    if (!value.trim()) return;
    setState('saving');
    setError('');
    try {
      const r = await fetch('/api/keys', {
        method: 'POST',
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ envVar, value: value.trim() }),
      });
      if (r.ok) {
        setState('saved');
        setValue('');
      } else {
        setState('error');
        setError((await r.json()).error?.toString?.() ?? 'save failed');
      }
    } catch (e) {
      setState('error');
      setError(String(e));
    }
  };

  return { value, setValue, state, setState, error, save };
}

/** Inline env-key entry for agent setup — posts to /api/keys (value lands in
 *  .env.local and takes effect immediately; never echoed back). */
export function SetupKeyInput({ envVar, present, masked }: { envVar: string; present: boolean; masked: string }) {
  const { value, setValue, state, setState, error, save } = useKeySave(envVar, present);

  return (
    <div className="flex flex-wrap items-center gap-2.5">
      {state === 'saved' ? (
        <span className="glass-chip inline-flex items-center gap-2 rounded-full px-3 py-1 text-[11.5px] text-os-muted">
          <span className="dot ok" /> set {masked && <span className="font-mono">· {masked}</span>}
        </span>
      ) : (
        <>
          <input
            type="password"
            value={value}
            onChange={(e) => setValue(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && save()}
            placeholder={envVar}
            className="w-[260px] max-w-full rounded-md-t border border-os-border bg-os-bg px-3 py-1.5 font-mono text-[12px] outline-none focus:border-os-border-strong"
          />
          <button
            onClick={save}
            disabled={state === 'saving' || !value.trim()}
            className="c-btn c-btn-primary h-auto px-3.5 py-1.5 text-[12px]"
          >
            {state === 'saving' ? 'Saving...' : 'Save'}
          </button>
        </>
      )}
      {state === 'saved' && (
        <button onClick={() => setState('idle')} className="text-[11.5px] text-os-dim transition-colors hover:text-os-text">
          replace
        </button>
      )}
      {error && (
        <span className="inline-flex items-center gap-2 text-[11.5px] text-os-muted">
          <span className="dot err" /> {error}
        </span>
      )}
    </div>
  );
}
