'use client';

import { useEffect, useState } from 'react';
import { Terminal } from 'lucide-react';
import type { WizardProject } from '@/lib/wizard/store';
import { relativeTime } from '@/lib/wizard/helpers';
import { tokensLabel } from '@/lib/wizard/ui';

type Engine = { present: boolean; version: string | null; host: string };

/** Every project's Claude session, plus the engine that runs them. Only
 *  numbers the envelope actually reported — no context meter without usage. */
export function SessionsTab({ projects, now }: { projects: WizardProject[]; now: number }) {
  const [engine, setEngine] = useState<Engine | null>(null);
  useEffect(() => {
    let alive = true;
    fetch('/api/wizard/engine')
      .then((r) => r.json())
      .then((b) => {
        if (alive && b.ok) setEngine(b.engine as Engine);
      })
      .catch(() => {});
    return () => {
      alive = false;
    };
  }, []);

  const withSession = projects.filter((p) => p.sessionId);
  const totalCost = projects.reduce((s, p) => s + p.totalCostUsd, 0);

  return (
    <div className="grid min-h-0 grid-cols-[320px_minmax(0,1fr)] gap-4 max-[1100px]:grid-cols-1">
      <div className="flex flex-col gap-4">
        <div className="wz-panel p-5">
          <div className="flex items-center gap-2 text-[13.5px] text-os-text">
            <Terminal className="h-4 w-4 text-os-muted" />
            Engine
          </div>
          <div className="mt-4 flex items-center gap-2 text-[13px] text-os-text">
            <span className={`dot ${engine ? (engine.present ? 'ok' : 'err') : 'off'}`} />
            {engine ? (engine.present ? 'claude binary present' : 'claude binary not found') : 'checking'}
          </div>
          <dl className="mt-3 grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5 text-[12px]">
            <dt className="text-os-dim">Version</dt>
            <dd className="font-mono text-os-muted">{engine?.version ?? '—'}</dd>
            <dt className="text-os-dim">Host</dt>
            <dd className="font-mono text-os-muted">{engine?.host ?? '—'}</dd>
            <dt className="text-os-dim">Mode</dt>
            <dd className="text-os-muted">one exchange per message, session resumed by id</dd>
          </dl>
        </div>
        <div className="wz-panel p-5">
          <div className="text-[13.5px] text-os-text">Totals</div>
          <div className="mt-3 grid grid-cols-2 gap-3">
            <Stat label="Sessions" value={String(withSession.length)} />
            <Stat label="Projects" value={String(projects.length)} />
            <Stat label="Spend" value={`$${totalCost.toFixed(2)}`} />
            <Stat label="Turns" value={String(projects.reduce((s, p) => s + (p.lastUsage?.turns ?? 0), 0))} />
          </div>
        </div>
      </div>

      <div className="wz-panel flex min-h-0 flex-col">
        <div className="border-b border-os-hairline px-5 py-3 text-[13.5px] text-os-text">Sessions</div>
        <div className="min-h-0 flex-1 overflow-y-auto">
          {withSession.length === 0 && <p className="px-5 py-6 text-[12.5px] text-os-dim">No sessions yet. The first message in a project opens one.</p>}
          {withSession.map((p) => (
            <div key={p.id} className="grid grid-cols-[minmax(0,1.4fr)_120px_120px_90px_80px_90px] items-center gap-3 border-b border-os-hairline px-5 py-3 text-[12.5px]">
              <div className="min-w-0">
                <div className="truncate text-os-text">{p.name}</div>
                <div className="truncate font-mono text-[11px] text-os-dim">{p.sessionId}</div>
              </div>
              <span className="truncate font-mono text-[11px] text-os-muted">{p.model ?? '—'}</span>
              <span className="tabular-nums text-os-muted">{p.lastUsage ? `${tokensLabel(p.lastUsage.inputTokens)} in · ${tokensLabel(p.lastUsage.outputTokens)} out` : '—'}</span>
              <span className="tabular-nums text-os-muted">{p.messages.length} msgs</span>
              <span className="tabular-nums text-os-muted">${p.totalCostUsd.toFixed(2)}</span>
              <span className="text-os-dim">{relativeTime(p.updatedAt, now)}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="glass-well border border-os-border bg-os-surface2/60 px-3 py-2.5">
      <div className="text-[11px] text-os-dim">{label}</div>
      <div className="mt-0.5 text-[20px] font-light tabular-nums text-os-text">{value}</div>
    </div>
  );
}
