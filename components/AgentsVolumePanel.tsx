'use client';

import { SlabCard, BigStat, MeterStack, InsightCard } from '@/components/slab';
import { StepLine, DotMatrix } from '@/components/slab-charts';
import { useBoardSnapshot } from '@/components/board-live-store';
import { agentsVolume } from '@/lib/agents-volume';
import type { BoardLivePayload } from '@/lib/board-live';

const WINDOW_DAYS = 14;

/**
 * The /agents slab hero (Alex, 2026-09-24: Brand Deals look OS-wide). Two
 * rows over the cockpit: run activity + the Agent Volume card, then runs by
 * seat, the open task lanes and the one "Needs you" card. It reads the same
 * snapshot BoardLive polls (via the board-live store), so the numbers here and
 * the board below never disagree. Every figure comes from lib/agents-volume;
 * "now" is the snapshot's own clock, so server and client render the same.
 */
export function AgentsVolumePanel({ initial }: { initial: BoardLivePayload }) {
  const data = useBoardSnapshot(initial);
  const v = agentsVolume(data, new Date(data.checkedAt), WINDOW_DAYS);
  const busiest = v.bySeat[0];

  return (
    <>
      {/* Hero row, Brand Deals' shape: the run line + the volume card */}
      <div className="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
        <SlabCard i={1} title="Run Activity" sub={v.window === `${WINDOW_DAYS}d` ? `last ${WINDOW_DAYS} days` : v.window} className="pb-2">
          <div className="px-6 pb-2 pt-3">
            <BigStat
              size={30}
              value={v.runsInWindow}
              chips={v.failedInWindow > 0 ? [{ tone: 'err', text: `${v.failedInWindow} failed` }] : []}
              caption="heartbeat runs off the live board (its latest 120)"
            />
          </div>
          <StepLine
            series={v.series}
            hue="var(--send-activity)"
            unit=" runs"
            empty={data.connected ? `No heartbeat runs in the last ${WINDOW_DAYS} days.` : 'Board unreachable, no runs to draw.'}
          />
        </SlabCard>

        <SlabCard i={2} title="Agent Volume" className="flex flex-col">
          <div className="flex flex-1 flex-col px-6 pb-6 pt-3">
            <BigStat value={v.headline} chips={v.chips} caption={v.caption} />
            <MeterStack meters={v.meters} foot={v.foot} empty="board unreachable, nothing to measure" />
          </div>
        </SlabCard>
      </div>

      {/* Second row: runs by seat, the open lanes, THE gradient card */}
      <div className="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
        <SlabCard i={3} title="Runs by Seat" sub={v.window === `${WINDOW_DAYS}d` ? `${WINDOW_DAYS} days` : v.window}>
          <div className="flex flex-col gap-5 px-6 pb-6 pt-3">
            <BigStat
              size={30}
              display={busiest ? busiest.label : 'none'}
              caption={busiest ? `busiest seat · ${busiest.count} runs` : `no runs in ${WINDOW_DAYS} days`}
            />
            <DotMatrix cols={v.bySeat} hue="var(--ramp-1)" />
          </div>
        </SlabCard>

        <SlabCard i={4} title="Task Lanes" sub="open stages">
          <div className="flex flex-col gap-5 px-6 pb-6 pt-3">
            <BigStat size={30} value={v.openTasks} caption={v.openTasks === 1 ? 'open task on the board' : 'open tasks on the board'} />
            <DotMatrix cols={v.lanes} hue="var(--accent)" />
          </div>
        </SlabCard>

        <InsightCard
          i={5}
          badge="Needs you"
          value={v.insight.value}
          headline={v.insight.headline}
          body={v.insight.body}
          frac={v.insight.frac}
        />
      </div>
    </>
  );
}
