'use client';

import { useMemo, useState } from 'react';
import { BarChart3, ChevronDown } from 'lucide-react';
import { taskOverview, type Grain } from '@/lib/tasks-metrics';
import type { AgentTask } from '@/lib/schemas';

/**
 * The reference's Task overview (docs/TASKS-REDESIGN.md §5.2): a hatched
 * Completed bar behind a solid Pending bar, per period, with the segmented
 * Daily/Monthly/Yearly control whose active pill slides.
 *
 * Both series are real: Pending counts tasks created in the bucket that are
 * not done, Completed counts tasks that reached done inside it.
 */
const GRAINS: { id: Grain; label: string }[] = [
  { id: 'daily', label: 'Daily' },
  { id: 'monthly', label: 'Monthly' },
  { id: 'yearly', label: 'Yearly' },
];

const H = 190; // plot height in px

export function TaskOverview({ tasks, now }: { tasks: AgentTask[]; now: number }) {
  const [grain, setGrain] = useState<Grain>('monthly');
  const [mode, setMode] = useState<'bar' | 'line'>('bar');
  const [menu, setMenu] = useState(false);
  const buckets = useMemo(() => taskOverview(tasks, grain, now), [tasks, grain, now]);

  const peak = Math.max(1, ...buckets.map((b) => Math.max(b.pending, b.completed)));
  // round the axis up to a friendly step, like the reference's 0/45/90/135/180
  const step = Math.max(1, Math.ceil(peak / 4));
  const top = step * 4;
  const ticks = [top, step * 3, step * 2, step, 0];
  const grainIndex = GRAINS.findIndex((g) => g.id === grain);

  const linePath = (key: 'pending' | 'completed') =>
    buckets
      .map((b, i) => {
        const x = ((i + 0.5) / buckets.length) * 100;
        const y = H - (b[key] / top) * H;
        return `${i === 0 ? 'M' : 'L'} ${x} ${y}`;
      })
      .join(' ');

  return (
    <div className="glass-panel flex flex-col overflow-hidden">
      <div className="flex flex-wrap items-start justify-between gap-3 px-5 pt-4">
        <div>
          <div className="text-[13.5px] font-medium">Task overview</div>
          <div className="mt-0.5 text-[12px] text-os-dim">
            What the fleet took on, and what it finished.
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {/* segmented control — the active pill slides between positions */}
          <div className="relative flex rounded-md-t bg-os-surface2 p-0.5">
            <span
              className="absolute inset-y-0.5 rounded-sm-t bg-os-surface shadow-sm transition-transform duration-200 ease-out"
              style={{
                width: `calc((100% - 4px) / ${GRAINS.length})`,
                transform: `translateX(${grainIndex * 100}%)`,
              }}
            />
            {GRAINS.map((g) => (
              <button
                key={g.id}
                onClick={() => setGrain(g.id)}
                className={`relative z-10 px-2.5 py-1 text-[12px] transition-colors ${
                  grain === g.id ? 'text-os-text' : 'text-os-dim hover:text-os-muted'
                }`}
              >
                {g.label}
              </button>
            ))}
          </div>

          <div className="relative">
            <button
              onClick={() => setMenu((m) => !m)}
              className="flex items-center gap-1.5 whitespace-nowrap rounded-md-t border border-os-border bg-os-surface2 px-2.5 py-1 text-[12px] text-os-muted transition-colors hover:bg-os-surface3"
            >
              <BarChart3 className="h-3.5 w-3.5" strokeWidth={1.6} />
              {mode === 'bar' ? 'Bar chart' : 'Line chart'}
              <ChevronDown className="h-3 w-3" strokeWidth={1.6} />
            </button>
            {menu && (
              <div className="glass-solid absolute right-0 z-20 mt-1 w-[132px] overflow-hidden rounded-md-t border border-os-border bg-os-surface shadow-lg">
                {(['bar', 'line'] as const).map((m) => (
                  <button
                    key={m}
                    onClick={() => { setMode(m); setMenu(false); }}
                    className={`block w-full px-3 py-2 text-left text-[12px] transition-colors hover:bg-os-surface3 ${
                      mode === m ? 'text-[var(--accent)]' : 'text-os-muted'
                    }`}
                  >
                    {m === 'bar' ? 'Bar chart' : 'Line chart'}
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      <div className="mt-3 flex items-center gap-4 px-5 text-[11.5px] text-os-dim">
        <span className="flex items-center gap-1.5">
          <span className="h-2.5 w-2.5 rounded-[3px]" style={{ background: 'var(--accent)' }} /> Pending
        </span>
        <span className="flex items-center gap-1.5">
          <span className="h-2.5 w-2.5 rounded-[3px]" style={{ background: 'var(--accent-soft)', backgroundImage: 'repeating-linear-gradient(45deg, var(--accent-line) 0 1px, transparent 1px 3px)' }} /> Completed
        </span>
      </div>

      <div className="flex gap-3 px-5 pb-4 pt-3">
        <div className="flex flex-col justify-between py-0.5 font-mono text-[10px] text-os-dim" style={{ height: H }}>
          {ticks.map((t) => <span key={t}>{t}</span>)}
        </div>

        <div className="relative min-w-0 flex-1">
          <div className="absolute inset-0 flex flex-col justify-between">
            {ticks.map((t) => <span key={t} className="h-px w-full" style={{ background: 'var(--hairline)' }} />)}
          </div>

          {mode === 'bar' ? (
            <div className="relative flex items-end gap-1" style={{ height: H }}>
              {buckets.map((b, i) => (
                <div key={`${b.label}-${i}`} className="group relative flex h-full flex-1 items-end justify-center">
                  {/* hatched Completed bar sits behind the solid Pending bar */}
                  <div
                    className="tasks-bar absolute bottom-0 w-full rounded-t-[6px]"
                    style={{
                      height: `${(b.completed / top) * 100}%`,
                      background: 'var(--accent-soft)',
                      backgroundImage: 'repeating-linear-gradient(45deg, var(--accent-line) 0 1px, transparent 1px 4px)',
                      animationDelay: `${i * 30}ms`,
                    }}
                  />
                  <div
                    className="tasks-bar relative w-[58%] rounded-t-[6px]"
                    style={{
                      height: `${(b.pending / top) * 100}%`,
                      background: 'var(--accent)',
                      opacity: 0.85,
                      animationDelay: `${i * 30 + 60}ms`,
                    }}
                  />
                  <div className="glass-solid pointer-events-none absolute -top-8 z-10 whitespace-nowrap rounded-sm-t border border-os-border bg-os-surface px-2 py-1 font-mono text-[10px] text-os-muted invisible translate-y-1.5 transition-transform group-hover:visible group-hover:translate-y-0">
                    {b.pending} pending · {b.completed} done
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <svg viewBox={`0 0 100 ${H}`} preserveAspectRatio="none" style={{ height: H }} className="relative w-full">
              <path d={linePath('completed')} fill="none" stroke="var(--accent-line)" strokeWidth="1.5" vectorEffect="non-scaling-stroke" />
              <path d={linePath('pending')} fill="none" stroke="var(--accent)" strokeWidth="2" vectorEffect="non-scaling-stroke" />
            </svg>
          )}

          <div className="mt-2 flex gap-1">
            {buckets.map((b, i) => (
              <span key={`${b.label}-${i}`} className="flex-1 text-center font-mono text-[10px] text-os-dim">
                {b.label}
              </span>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
