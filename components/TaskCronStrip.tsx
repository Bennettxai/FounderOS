import Link from 'next/link';
import { AlertTriangle } from 'lucide-react';
import { Dot } from '@/components/terminal';
import { SlabCard, PILL } from '@/components/slab';
import { scheduledJobRows, type CronStat } from '@/lib/scheduled-jobs';
import type { AgentCron } from '@/lib/schemas';

/**
 * Scheduled jobs on /tasks (Alex, 2026-08-18: "make the cron jobs show up
 * in tasks and in workflows ... keep data on it: how many times this runs"),
 * reshaped for mock 5f into a compact card grid: square status dot, job name,
 * cadence plus the next fire, and the ok/runs count. A failing job reads red.
 *
 * Deliberately built on lib/scheduled-jobs.ts, which also feeds the /workflows
 * panel: one shaping, two renderers. Every number comes from `cron_runs`,
 * written by the tick that really fires the job. A schedule that has never run
 * reads zero, not "fine".
 *
 * 2026-09-24: the strip is a Brand Deals SlabCard; the page passes its stagger.
 */
function ago(iso: string | null): string {
  if (!iso) return 'never';
  const ms = Date.now() - Date.parse(iso);
  if (!Number.isFinite(ms) || ms < 0) return 'just now';
  const m = Math.floor(ms / 60_000);
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  return `${Math.floor(h / 24)}d ago`;
}

function inNext(iso: string | null): string | null {
  if (!iso) return null;
  const ms = Date.parse(iso) - Date.now();
  if (!Number.isFinite(ms) || ms < 0) return null;
  const m = Math.round(ms / 60_000);
  if (m < 60) return `${m}m`;
  const h = Math.round(m / 60);
  if (h < 24) return `${h}h`;
  return `${Math.round(h / 24)}d`;
}

export function TaskCronStrip({
  crons,
  stats,
  agentNames,
  i = 6,
}: {
  crons: AgentCron[];
  stats: Record<string, CronStat>;
  agentNames: Record<string, string>;
  i?: number;
}) {
  const rows = scheduledJobRows({ crons, stats, agentNames });
  if (rows.length === 0) return null;
  const enabled = rows.filter((r) => r.enabled).length;

  return (
    <SlabCard
      i={i}
      className="mt-6"
      title="Scheduled jobs"
      sub={`${enabled} of ${rows.length} enabled`}
      action={
        <Link href="/workflows" className={PILL}>
          full panel →
        </Link>
      }
    >
      <div className="grid gap-3 px-6 pb-6 pt-4 sm:grid-cols-2 xl:grid-cols-3">
        {rows.map((r) => {
          const failing = r.lastOk === false || r.overdue || r.unknownAgent;
          const next = r.enabled ? inNext(r.nextRunAt) : null;
          return (
            <div
              key={r.id}
              className={`rounded-[10px] border bg-os-bg px-4 py-3.5 ${
                failing ? 'border-os-err/50' : 'border-os-border'
              } ${!r.enabled ? 'opacity-60' : ''}`}
            >
              <div className="flex items-center gap-2">
                <Dot state={!r.enabled ? 'off' : failing ? 'error' : 'connected'} />
                <span className="min-w-0 flex-1 truncate text-[13.5px] font-medium text-os-text">{r.description}</span>
                <span
                  className={`shrink-0 rounded-full px-2.5 py-0.5 font-mono text-[10.5px] tabular-nums ${failing ? 'text-os-err' : 'text-os-muted'}`}
                  style={{ background: failing ? 'color-mix(in oklab, var(--err) 14%, transparent)' : 'color-mix(in oklab, var(--text) 6%, transparent)' }}
                >
                  {r.ok}/{r.runs}
                </span>
              </div>
              <div className="mt-2 flex items-center gap-2 font-mono text-[11px] text-os-dim">
                <span className="min-w-0 truncate">{r.scheduleLabel}</span>
                {next && <span className="shrink-0">· next {next}</span>}
                {!r.enabled && <span className="shrink-0">· paused</span>}
              </div>
              <div className="mt-1 flex items-center gap-1.5 font-mono text-[11px]">
                <span className="min-w-0 truncate text-os-dim">{r.agentName}</span>
                <span className={`ml-auto shrink-0 ${r.lastOk === false ? 'text-os-err' : 'text-os-dim'}`}>
                  {r.lastOk === false ? `last failed · ${ago(r.lastRunAt)}` : ago(r.lastRunAt)}
                </span>
              </div>
              {(r.overdue || r.unknownAgent) && (
                <div className="mt-1.5 flex items-center gap-1 font-mono text-[9.5px] text-os-err">
                  <AlertTriangle className="h-3 w-3" />
                  {r.unknownAgent ? 'agent missing from the runtime' : 'overdue · slot passed with no run'}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </SlabCard>
  );
}
