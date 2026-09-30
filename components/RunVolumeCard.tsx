'use client';

import { useState } from 'react';
import { BigStat, SlabCard, chipClass } from '@/components/slab';

type DayPoint = { date: string; count: number };
type Range = 7 | 14 | 30;
const RANGES: Range[] = [7, 14, 30];
const EASE = 'cubic-bezier(.2,.7,.2,1)';
const HUE = 'var(--send-activity)';

const fmtShort = (iso: string) =>
  new Date(`${iso}T00:00:00Z`)
    .toLocaleDateString('en-US', { month: 'short', day: 'numeric', timeZone: 'UTC' })
    .toLowerCase();

/** Agent-run volume from the real run log, as the Brand Deals hero card
    (2026-09-24): a count-up headline, solid-accent range pills, and per-day
    bars in the Deal Volume meters' hatched glow that grow up on the stagger.
    Hover still answers with the day's number, and zero days keep an honest
    2px stub instead of vanishing. */
export function RunVolumeCard({ data, i = 1 }: { data: DayPoint[]; i?: number }) {
  const [range, setRange] = useState<Range>(14);
  const [hovered, setHovered] = useState<string | null>(null);
  const shown = data.slice(-range);
  const windowRuns = shown.reduce((s, p) => s + p.count, 0);
  const activeDays = shown.filter((p) => p.count > 0).length;
  const max = Math.max(1, ...shown.map((p) => p.count));
  const hot = hovered ? shown.find((p) => p.date === hovered) : null;

  return (
    <SlabCard
      i={i}
      title="Agent Run Volume"
      sub={`${range}d`}
      className="flex flex-col"
      action={RANGES.map((r) => (
        <button key={r} className={chipClass(range === r)} onClick={() => setRange(r)} aria-pressed={range === r}>
          {r}d
        </button>
      ))}
    >
      <div className="flex flex-1 flex-col px-6 pb-5 pt-3">
        <BigStat
          key={range}
          value={windowRuns}
          chips={[{ tone: 'accent', text: `${activeDays} of ${shown.length} days active` }]}
          caption={`agent runs in the last ${range} days, from the real run log`}
        />
        <div className="mt-5 flex flex-1 items-end gap-[3px] border-t border-os-border pt-5" style={{ minHeight: 190 }}>
          {shown.map((p, k) => {
            const on = hovered === p.date;
            const h = Math.max(2, (p.count / max) * 170);
            return (
              <div
                key={`${range}-${p.date}`}
                className="flex h-full flex-1 items-end self-stretch"
                onMouseEnter={() => setHovered(p.date)}
                onMouseLeave={() => setHovered(null)}
              >
                <div
                  className="w-full rounded-t-[4px] transition-[opacity] duration-150"
                  style={{
                    height: `${h}px`,
                    opacity: on ? 1 : hovered ? 0.35 : p.count === 0 ? 0.3 : 0.85,
                    transformOrigin: 'bottom',
                    background: `linear-gradient(0deg, transparent 60%, color-mix(in oklab, ${HUE} 60%, white) 100%), repeating-linear-gradient(45deg, ${HUE}, ${HUE} 5px, color-mix(in oklab, ${HUE} 45%, transparent) 5px, color-mix(in oklab, ${HUE} 45%, transparent) 10px)`,
                    boxShadow: p.count > 0 ? `0 0 12px color-mix(in oklab, ${HUE} 40%, transparent)` : undefined,
                    animation: `os-grow .9s ${EASE} ${400 + k * 25}ms both`,
                  }}
                />
              </div>
            );
          })}
        </div>

        <div className="mt-2 flex justify-between font-mono text-[10.5px] text-os-dim">
          <span>{fmtShort(shown[0].date)}</span>
          <span className={hot ? 'text-os-text' : ''}>{hot ? `${hot.count} runs · ${fmtShort(hot.date)}` : 'hover a day'}</span>
          <span>today</span>
        </div>
      </div>
    </SlabCard>
  );
}
