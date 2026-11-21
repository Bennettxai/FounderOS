'use client';

import { useState } from 'react';
import { ChevronLeft, ChevronRight, Clock } from 'lucide-react';
import { SetupLink } from '@/components/SetupLink';
import type { AgentTask } from '@/lib/schemas';

/**
 * "My Schedule" from the reference (docs/TASKS-REDESIGN.md §5.3): month
 * stepper, a five-day strip with the active day as a filled pill, and cards
 * for what is coming.
 *
 * Scheduled agent work would come from `agent_crons`, which is empty on every
 * install and has no runner (lib/cron.ts describes schedules, it does not run
 * them). Until that exists this rail shows tasks that carry a due date, and
 * says plainly when there is nothing to show.
 */
const MONTHS = ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December'];
const DAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

export function ScheduleRail({
  tasks,
  agentNames,
  now,
}: {
  tasks: AgentTask[];
  agentNames: Record<string, string>;
  now: number;
}) {
  const [offset, setOffset] = useState(0); // days from today
  const base = new Date(now);
  const anchor = new Date(base.getFullYear(), base.getMonth(), base.getDate() + offset);

  const strip = Array.from({ length: 5 }, (_, i) => {
    const d = new Date(anchor.getFullYear(), anchor.getMonth(), anchor.getDate() + i - 2);
    return d;
  });

  const sameDay = (iso: string, d: Date) => {
    const t = new Date(iso);
    return t.getFullYear() === d.getFullYear() && t.getMonth() === d.getMonth() && t.getDate() === d.getDate();
  };

  const dated = tasks.filter((t) => t.dueAt != null && t.status !== 'done');
  const due = dated
    .filter((t) => sameDay(t.dueAt!, anchor))
    .sort((a, b) => new Date(a.dueAt!).getTime() - new Date(b.dueAt!).getTime());

  return (
    <div className="glass-panel flex flex-col overflow-hidden">
      <div className="flex items-center justify-between px-5 pt-4">
        <div className="text-[13.5px] font-medium">My Schedule</div>
      </div>

      <div className="glass-well mx-5 mt-3 flex items-center justify-between px-2 py-1.5">
        <button onClick={() => setOffset((o) => o - 30)} className="rounded-sm-t p-1 text-os-dim transition-colors hover:text-os-text" aria-label="Previous month">
          <ChevronLeft className="h-3.5 w-3.5" strokeWidth={1.8} />
        </button>
        <span className="text-[12.5px] text-os-muted">
          {MONTHS[anchor.getMonth()]} {anchor.getFullYear()}
        </span>
        <button onClick={() => setOffset((o) => o + 30)} className="rounded-sm-t p-1 text-os-dim transition-colors hover:text-os-text" aria-label="Next month">
          <ChevronRight className="h-3.5 w-3.5" strokeWidth={1.8} />
        </button>
      </div>

      <div className="mt-3 flex items-center gap-1 px-4">
        <button onClick={() => setOffset((o) => o - 1)} className="shrink-0 rounded-sm-t p-1 text-os-dim transition-colors hover:text-os-text" aria-label="Previous day">
          <ChevronLeft className="h-3.5 w-3.5" strokeWidth={1.8} />
        </button>
        {strip.map((d) => {
          const active = d.getDate() === anchor.getDate() && d.getMonth() === anchor.getMonth();
          const has = dated.some((t) => sameDay(t.dueAt!, d));
          return (
            <button
              key={d.toISOString()}
              onClick={() => setOffset((o) => o + Math.round((d.getTime() - anchor.getTime()) / 86_400_000))}
              className={`flex flex-1 flex-col items-center gap-0.5 rounded-md-t px-1 py-1.5 transition-colors ${
                active ? 'bg-[var(--accent-soft)] text-[var(--accent)]' : 'text-os-dim hover:bg-os-surface2'
              }`}
            >
              <span className="text-[13px] tabular-nums">{String(d.getDate()).padStart(2, '0')}</span>
              <span className="text-[10px]">{DAYS[d.getDay()]}</span>
              <span className="h-1 w-1 rounded-full" style={{ background: has ? 'var(--accent)' : 'transparent' }} />
            </button>
          );
        })}
        <button onClick={() => setOffset((o) => o + 1)} className="shrink-0 rounded-sm-t p-1 text-os-dim transition-colors hover:text-os-text" aria-label="Next day">
          <ChevronRight className="h-3.5 w-3.5" strokeWidth={1.8} />
        </button>
      </div>

      <div className="flex flex-col gap-2 px-5 pb-4 pt-3">
        {due.map((t) => (
          <div key={t.id} className="tasks-card glass-well px-3 py-2.5">
            <div className="truncate text-[12.5px] text-os-text">{t.title}</div>
            <div className="mt-1 flex items-center gap-2 text-[11.5px] text-os-dim">
              <Clock className="h-3 w-3" strokeWidth={1.6} />
              {new Date(t.dueAt!).toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' })}
              <span className="ml-auto truncate rounded-sm-t border border-os-border px-1.5 py-0.5 text-[10.5px]">
                {t.agentId ? agentNames[t.agentId] ?? t.agentId : 'You'}
              </span>
            </div>
          </div>
        ))}

        {due.length === 0 && (
          <div className="rounded-md-t border border-dashed border-os-border px-3 py-6 text-center">
            <div className="text-[12.5px] text-os-muted">Nothing due this day</div>
            <div className="mt-1 text-[11.5px] leading-relaxed text-os-dim">
              Scheduled agent work will appear here once cron jobs exist and a
              runner is live.
            </div>
            <SetupLink href="/agents" hint="Schedules are defined per agent; no runner is installed yet." className="mt-2 inline-flex" />
          </div>
        )}
      </div>
    </div>
  );
}
