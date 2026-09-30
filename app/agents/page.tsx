import { paperclipAgents, paperclipIssues, paperclipRuns } from '@/lib/connectors/paperclip';
import { getDb } from '@/lib/data';
import type { BoardLivePayload } from '@/lib/board-live';
import Link from 'next/link';
import { ArrowUpRight } from 'lucide-react';
import { Slab, SlabTitle, SlabCard, PILL } from '@/components/slab';
import { AgentsTabs } from '@/components/AgentsTabs';
import { BoardLive } from '@/components/BoardLive';
import { ConductorChat } from '@/components/ConductorChat';
import { AgentsVolumePanel } from '@/components/AgentsVolumePanel';

export const dynamic = 'force-dynamic';

/**
 * /agents, completely de-larped (Alex, 2026-08-07: "I need to use this").
 * Everything on this page is live board data or a live embed  -  the seeded
 * roster cards, fake-tool-block chats, seeded stats/crons/cost analysis are
 * gone. What remains: the polling BoardLive strip (real stats row, seat chips
 * with models, run feed, task lanes), the real CEO chat, and the Hermes
 * worker-pool dashboard tab.
 *
 * 2026-09-24: the page wears the Brand Deals slab (components/slab). The hero
 * and second row (AgentsVolumePanel) read the same snapshot BoardLive polls,
 * through lib/agents-volume; the cockpit below keeps its viewport height.
 */
export default async function AgentsPage() {
  // first paint of the live board comes from the server; BoardLive then polls
  const [liveAgents, liveIssues, liveRuns] = await Promise.all([
    paperclipAgents(),
    // Deep enough to clear the done backlog. At 40 the board's 132 finished
    // issues filled the whole response, so the lanes showed nothing but DONE
    // and "Open tasks" reported 0 while two tasks were actually open.
    paperclipIssues(250),
    paperclipRuns(120),
  ]);
  // decisions come from the repo layer, not the board: an approve is ours
  const decisions = getDb().deliverableDecisions.all();
  const boardInitial: BoardLivePayload = {
    connected: liveAgents.length > 0,
    agents: liveAgents,
    issues: liveIssues,
    runs: liveRuns,
    checkedAt: new Date().toISOString(),
    decisions,
  };

  const boardUrl = process.env.PAPERCLIP_API_URL ?? null;

  return (
    <Slab>
      <SlabTitle
        eyebrow="runtime"
        title="Real Agents"
        meta="live Paperclip board · polls every 4s · Conductor on the rail"
        right={
          <>
            {boardUrl && (
              <a href={boardUrl} target="_blank" rel="noreferrer" className={PILL}>
                Open board <ArrowUpRight size={13} strokeWidth={1.7} />
              </a>
            )}
            <Link href="/workflows" className={PILL}>
              Workflows
            </Link>
          </>
        }
      />

      {/* Hero + second row: run activity, Agent Volume, seats, lanes, insight */}
      <AgentsVolumePanel initial={boardInitial} />

      {/* The cockpit (Alex, 2026-08-10: "I don't want to scroll at all"):
          it owns one viewport of height, both panels stretch to its bottom
          edge, and anything long scrolls INSIDE its panel. Narrow screens fall
          back to normal flow. */}
      <SlabCard i={6} className="mt-6 p-5">
        <div className="flex flex-col xl:h-[calc(100dvh-12rem)]">
          <AgentsTabs
            hermesUrl={process.env.HERMES_DASH_URL ?? 'https://os.example.internal:9000'}
            boardUrl={boardUrl}
          >
            {/* Board left, Conductor rail right, both full height. On narrow
                screens the Conductor rides on top. */}
            <div className="grid gap-6 xl:h-full xl:min-h-0 xl:grid-cols-[minmax(0,1fr)_400px]">
              <div className="order-2 min-h-0 min-w-0 xl:order-none xl:col-start-1 xl:row-start-1 xl:h-full">
                <BoardLive initial={boardInitial} boardUrl={boardUrl} />
              </div>
              <div className="order-1 min-h-0 xl:order-none xl:col-start-2 xl:row-start-1 xl:h-full">
                <ConductorChat model={liveAgents.find((a) => a.name === 'Conductor')?.model ?? null} />
              </div>
            </div>
          </AgentsTabs>
        </div>
      </SlabCard>
    </Slab>
  );
}
