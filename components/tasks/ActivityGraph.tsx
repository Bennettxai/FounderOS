'use client';

import { RunsEmpty } from '@/components/tasks/RunsEmpty';
import type { Activity } from '@/lib/tasks-metrics';

/**
 * "My Activity" from the dark reference (docs/TASKS-REDESIGN.md §5.6), cloned:
 * a big success percentage with its delta above, seven hatched weekday bars
 * carrying their % at the BASE rather than the top, the peak bar lit, and a
 * marker line crossing it with a floating "N runs" pill.
 *
 * Source is `agent_runs.ok`. With no runs in the window the panel says so,
 * names the last run and offers a wider range (H27); it never draws a flat
 * zero week, and never claims nothing has run when something has.
 */
export function ActivityGraph({
  activity,
  rangeLabel,
  rangeDays,
  lastRunAt,
  now,
}: {
  activity: Activity | null;
  rangeLabel: string;
  rangeDays: number;
  lastRunAt: string | null;
  now: number;
}) {
  if (!activity) {
    return (
      <div className="glass-panel flex min-h-[300px] flex-col px-5 py-4">
        <div className="glass-label">My Activity</div>
        <div className="flex flex-1 flex-col items-center justify-center gap-2 text-center">
          <RunsEmpty
            windowLabel={`the ${rangeLabel.toLowerCase()}`}
            lastRunAt={lastRunAt}
            now={now}
            minDays={rangeDays}
          />
        </div>
      </div>
    );
  }

  const peak = Math.max(1, ...activity.days.map((d) => d.runs));
  const up = (activity.delta ?? 0) >= 0;

  return (
    <div className="glass-panel flex min-h-[300px] flex-col px-5 py-4">
      <div className="flex items-start justify-between">
        <div className="glass-label">My Activity</div>
        <span className="glass-chip rounded-full px-2 py-0.5 text-[11.5px] text-os-dim">
          {rangeLabel}
        </span>
      </div>

      <div className="mt-3">
        {activity.delta != null && (
          <div className={`text-[11.5px] tabular-nums ${up ? 'text-[var(--accent)]' : 'text-[var(--err)]'}`}>
            {up ? '+' : ''}{activity.delta}%
          </div>
        )}
        <div className="text-[40px] font-normal leading-none tabular-nums">{activity.pct}%</div>
        <div className="mt-1 text-[11.5px] text-os-dim">of agent runs succeeded</div>
      </div>

      <div className="relative mt-5 flex flex-1 items-end gap-2">
        {/* marker line across the busiest day, with its floating count pill */}
        <div
          className="pointer-events-none absolute left-0 right-0 flex items-center"
          style={{ bottom: `${(activity.days[activity.peakIndex].runs / peak) * 70 + 22}%` }}
        >
          <span className="tasks-marker h-px flex-1" style={{ background: 'var(--hairline)' }} />
          <span className="glass-chip ml-2 shrink-0 rounded-full px-2 py-0.5 font-mono text-[10px] text-os-muted">
            {activity.peakRuns} runs
          </span>
        </div>

        {activity.days.map((d, i) => {
          const isPeak = i === activity.peakIndex;
          const low = d.runs > 0 && d.pct < 50;
          return (
            <div key={d.label} className="group relative flex flex-1 flex-col items-center justify-end gap-1.5">
              <div
                className="tasks-bar w-full rounded-t-[6px]"
                style={{
                  height: `${Math.max(6, (d.runs / peak) * 140)}px`,
                  background: isPeak ? 'var(--accent-soft)' : 'var(--surface-2)',
                  backgroundImage: `repeating-linear-gradient(45deg, ${
                    isPeak ? 'var(--accent-line)' : 'var(--hairline)'
                  } 0 1px, transparent 1px 4px)`,
                  border: `1px solid ${isPeak ? 'var(--accent-line)' : 'var(--hairline)'}`,
                  animationDelay: `${i * 40}ms`,
                }}
              />
              {/* the % label sits at the base of the bar, as in the reference */}
              <span
                className={`font-mono text-[10px] tabular-nums ${
                  low ? 'text-[var(--err)]' : isPeak ? 'text-[var(--accent)]' : 'text-os-muted'
                }`}
              >
                {d.runs === 0 ? '—' : `${d.pct}%`}
              </span>
              <span className="font-mono text-[9.5px] text-os-dim">{d.label}</span>
              <div className="glass-solid pointer-events-none absolute -top-7 z-10 whitespace-nowrap rounded-sm-t border border-os-border bg-os-surface px-2 py-1 font-mono text-[10px] text-os-muted invisible translate-y-1.5 transition-transform group-hover:visible group-hover:translate-y-0">
                {d.runs} runs
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
