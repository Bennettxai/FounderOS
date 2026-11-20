'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { GErrorLine, GToggle } from '@/components/g';
import { NAV_GROUPS } from '@/lib/nav';
import { isToggleable } from '@/lib/modules';

/**
 * Settings → Modules (lib/modules.ts): one switch per sidebar view. Both
 * trees share the hook; the change lands in the sidebar, palette and digit
 * keys on the refresh that follows.
 */
export function useModules(initialOff: string[]) {
  const router = useRouter();
  const [off, setOff] = useState(initialOff);
  const [busy, setBusy] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const toggle = async (href: string, on: boolean) => {
    const prev = off;
    setOff(on ? off.filter((h) => h !== href) : [...off, href]);
    setBusy(href);
    setError(null);
    try {
      const res = await fetch('/api/settings/modules', {
        method: 'POST',
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ href, on }),
      });
      const body = (await res.json()) as { ok: boolean; off?: string[]; error?: string };
      if (!body.ok || !body.off) throw new Error(body.error ?? 'Could not save the change');
      setOff(body.off);
      router.refresh();
    } catch (e) {
      setOff(prev);
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(null);
    }
  };
  return { off, busy, error, toggle };
}

const GROUPS = NAV_GROUPS.map((g) => ({ ...g, items: g.items.filter((i) => isToggleable(i.href)) })).filter((g) => g.items.length > 0);

export function ModulesG({ initialOff }: { initialOff: string[] }) {
  const { off, busy, error, toggle } = useModules(initialOff);
  return (
    <div className="kg-modules">
      {GROUPS.map((g) => (
        <div key={g.id} className="kg-modules-group" role="group" aria-label={g.label}>
          <p className="kg-modules-title">{g.title ?? 'Command'}</p>
          {g.items.map(({ href, label, icon: Icon }) => {
            const on = !off.includes(href);
            return (
              <div key={href} className="kg-modules-row" data-off={on ? undefined : ''}>
                <Icon size={15} strokeWidth={1.7} aria-hidden="true" />
                <span className="kg-modules-label">{label}</span>
                <GToggle checked={on} onCheckedChange={(next) => void toggle(href, next)} disabled={busy === href} aria-label={`${label} module`} />
              </div>
            );
          })}
        </div>
      ))}
      {error ? <GErrorLine align="left">{error}</GErrorLine> : null}
    </div>
  );
}

export function ModulesClassic({ initialOff }: { initialOff: string[] }) {
  const { off, busy, error, toggle } = useModules(initialOff);
  return (
    <div className="grid gap-x-8 gap-y-5 sm:grid-cols-2">
      {GROUPS.map((g) => (
        <div key={g.id} role="group" aria-label={g.label}>
          <p className="mb-1.5 text-[11.5px] text-os-dim">{g.title ?? 'Command'}</p>
          <div className="divide-y divide-os-border rounded-lg-t border border-os-border bg-os-surface">
            {g.items.map(({ href, label, icon: Icon }) => {
              const on = !off.includes(href);
              return (
                <div key={href} className="flex items-center gap-2.5 px-3 py-2">
                  <Icon size={14} strokeWidth={1.7} className={on ? 'text-os-muted' : 'text-os-dim'} aria-hidden="true" />
                  <span className={`flex-1 text-[12.5px] ${on ? 'text-os-text' : 'text-os-dim'}`}>{label}</span>
                  <button
                    type="button"
                    role="switch"
                    aria-checked={on}
                    aria-label={`${label} module`}
                    disabled={busy === href}
                    onClick={() => void toggle(href, !on)}
                    className="relative h-[18px] w-[32px] shrink-0 rounded-full border border-os-border-strong transition-colors disabled:opacity-50"
                    style={{ background: on ? 'var(--accent)' : 'var(--surface-3)' }}
                  >
                    <span
                      aria-hidden
                      className="absolute top-[2px] h-[12px] w-[12px] rounded-full transition-[left]"
                      style={{ left: on ? 16 : 2, background: on ? 'var(--bg)' : 'var(--text-3)' }}
                    />
                  </button>
                </div>
              );
            })}
          </div>
        </div>
      ))}
      {error ? <p className="text-[11.5px] text-os-warn sm:col-span-2">{error}</p> : null}
    </div>
  );
}
