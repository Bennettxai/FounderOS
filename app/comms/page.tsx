import { CalendarDays, Hash, Mail, MessageSquare, Mic, type LucideIcon } from 'lucide-react';
import { CommsTabs } from '@/components/CommsTabs';
import { CommsDigestPanel } from '@/components/CommsDigestPanel';
import { Rise } from '@/components/motion';
import { getDb } from '@/lib/data';
import type { DigestRunResult } from '@/lib/comms-digest-run';
import { gatherCommsLanes } from '@/lib/comms-lanes';
import { gatherRecordings } from '@/lib/recordings';
import { gatherSlackClientBoard } from '@/lib/slack-clients';
import { listChannels } from '@/lib/connectors/slack';
import { caldavAccounts, calendarStatus, upcomingEvents } from '@/lib/connectors/gcal';
import { Badge, Dot } from '@/components/terminal';
import { CountUp } from '@/components/CountUp';
import { Slab, SlabTitle, SlabCard, BigStat, MeterStack, InsightCard, PILL } from '@/components/slab';
import { StepLine, DotMatrix } from '@/components/slab-charts';
import { commsVolume } from '@/lib/comms-volume';

export const dynamic = 'force-dynamic';

const SOURCE_ICON: Record<string, LucideIcon> = {
  whatsapp: MessageSquare,
  email: Mail,
  slack: Hash,
  calendar: CalendarDays,
  plaud: Mic,
};

export default async function CommsPage() {
  const [{ lanes, emailState, whatsappState }, { cards: slackCards, status: slackState }, channels, calendar, weekEvents, recordings] =
    await Promise.all([
      gatherCommsLanes(),
      gatherSlackClientBoard(),
      listChannels(),
      calendarStatus(),
      upcomingEvents(undefined, { days: 7, limit: 200 }),
      gatherRecordings(30),
    ]);

  const calLegend = caldavAccounts().map((a) => ({ name: a.name, color: a.color }));
  const nowISO = new Date().toISOString();
  // Plaud is the fifth source: the recorder in the room, feeding the Recordings tab.
  const sources = [emailState, whatsappState, slackState, calendar, ...recordings.sources.filter((s) => s.id === 'plaud')];

  // The 09:00 cron writes this; reading the stored row keeps the page fast
  // (a live re-scrape of six connectors is ~20s cold).
  const stored = getDb().commsDigests.latest();
  let digestRun: DigestRunResult | null = null;
  if (stored) {
    try {
      digestRun = JSON.parse(stored.payload) as DigestRunResult;
    } catch {
      digestRun = null;
    }
  }
  const vol = commsVolume({ lanes, slackCards, sources, events: weekEvents, recordings: recordings.recordings });
  const connectedSources = sources.filter((s) => s.state === 'connected').length;

  return (
    <Slab>
      <SlabTitle
        eyebrow="by source · inboxes · whatsapp · slack · calendar · recorder"
        title="Comms"
        meta={vol.meta}
        right={
          <>
            <Badge tone="accent">{vol.headline} unread</Badge>
            <a href="#inbox" className={PILL}>
              Open inbox
            </a>
          </>
        }
      />

      {/* Hero row: the sources + the Deal Volume card worn by comms numbers */}
      <div className="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
        <SlabCard i={1} title="Sources" sub={`${connectedSources}/${sources.length} connected`} className="pb-6">
          <div className="mt-4 grid gap-3 px-6 sm:grid-cols-2 xl:grid-cols-3">
            {sources.map((source) => {
              const Icon = SOURCE_ICON[source.id] ?? Mail;
              const ok = source.state === 'connected';
              return (
                <div key={source.id} data-lens="r" className="pressable is-row rounded-[12px] border border-os-border bg-os-bg px-4 py-3.5">
                  <div className="flex items-center gap-[9px]">
                    <Icon className={`h-[15px] w-[15px] shrink-0 ${ok ? 'text-os-accent' : 'text-os-dim'}`} strokeWidth={1.7} />
                    <span className="text-[13.5px] font-medium">{source.name}</span>
                    <span className="ml-auto flex items-center gap-2">
                      {ok && <Dot state="connected" pulse />}
                      <Badge tone={ok ? 'ok' : source.state === 'error' ? 'err' : 'default'} ghost={source.state === 'not_configured'}>
                        {ok ? 'Connected' : source.state === 'error' ? 'Error' : 'Not configured'}
                      </Badge>
                    </span>
                  </div>
                  <p className="mt-[9px] font-mono text-[10.5px] leading-relaxed text-os-dim">{source.detail}</p>
                </div>
              );
            })}
          </div>
        </SlabCard>

        <SlabCard i={2} title="Message Volume" className="flex flex-col">
          <div className="flex flex-1 flex-col px-6 pb-6 pt-3">
            <BigStat value={vol.headline} chips={vol.chips} caption={vol.caption} />
            <MeterStack meters={vol.meters} foot={vol.foot} />
          </div>
        </SlabCard>
      </div>

      {/* Second row: message activity, the week's meetings, THE insight card */}
      <div className="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
        <SlabCard i={3} title="Message Activity">
          <div className="px-6 pt-3">
            <span className="text-[30px] font-semibold tabular-nums tracking-[-0.03em]">
              <CountUp value={vol.seriesTotal} />
            </span>
            <span className="ml-2 text-[13px] text-os-dim">threads in view, last 14 days</span>
          </div>
          <StepLine series={vol.series} hue="var(--send-activity)" empty="No messages in this window." />
        </SlabCard>

        <SlabCard i={4} title="Meetings This Week">
          <div className="flex items-end justify-between gap-4 px-6 pb-6 pt-3">
            <div>
              <BigStat value={vol.meetingsTotal} size={30} caption="meetings, next 7 days" />
              <div className="mt-4 rounded-full border border-os-border px-3 py-1 text-[12px] text-os-muted">
                Recordings: <span className="font-semibold tabular-nums">{recordings.recordings.length}</span>
              </div>
            </div>
            <DotMatrix cols={vol.meetings} hue="var(--ramp-1)" />
          </div>
        </SlabCard>

        <InsightCard
          i={5}
          badge="Waiting on you"
          value={vol.insight.value}
          headline={vol.insight.headline}
          body={vol.insight.body}
          frac={vol.insight.frac}
        />
      </div>

      <Rise i={6} className="mt-6">
        <CommsDigestPanel
          initial={digestRun?.digest ?? null}
          sources={digestRun?.sources ?? []}
          generatedAt={stored?.generatedAt ?? null}
          initialRead={getDb().digestReads.keys()}
        />
      </Rise>

      {/* Swappable front: the messaging board (source lanes + Slack) or the 7-day meetings calendar */}
      <SlabCard i={7} title="Inbox" sub={`${vol.headline} unread`}>
        <div id="inbox" className="scroll-mt-6 px-6 pb-6 pt-4">
          <CommsTabs
            lanes={lanes}
            slackCards={slackCards}
            channels={channels}
            events={weekEvents}
            accounts={calLegend}
            recordings={recordings}
            nowISO={nowISO}
          />
        </div>
      </SlabCard>

      <Rise as="p" i={8} className="mt-6 rounded-full border border-dashed border-os-border-strong px-4 py-3 text-center font-mono text-[10.5px] text-os-dim">
        Four inboxes (expand to read + reply) and WhatsApp as lanes · Slack per client + every current channel · meetings via CalDAV · recordings from Plaud + Fathom
      </Rise>
    </Slab>
  );
}
