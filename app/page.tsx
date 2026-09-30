import Link from 'next/link';
import { getDb } from '@/lib/data';
import { allConnectorStatuses } from '@/lib/connectors';
import { createGBrainProvider } from '@/lib/connectors/gbrain';
import { stripeSnapshot } from '@/lib/connectors/payments';
import { gatherCommsFeed } from '@/lib/comms-feed';
import { inboundLast24h } from '@/lib/comms';
import { collapseRuns } from '@/lib/agents/run-digest';
import { CountUp } from '@/components/CountUp';
import { Rise } from '@/components/motion';
import { HomeNeedsYou } from '@/components/HomeNeedsYou';
import { InterjectComposer } from '@/components/InterjectComposer';
import { Kbd, Label, SparkBars } from '@/components/terminal';
import { runsPerDay, inboundPerDay, stateOfWorld, operatingVolume, dailySeries, sourceMix, homeAttention, type Tone } from '@/lib/pulse-history';
import { SlabTitle, SlabCard, BigStat, MeterStack, InsightCard } from '@/components/slab';
import { StepLine, DotMatrix } from '@/components/slab-charts';
import type { ConnectorStatus } from '@/lib/connectors/types';

export const dynamic = 'force-dynamic';

const TONE_CLASS: Record<Tone, string> = {
  ok: 'text-os-ok',
  warn: 'text-os-warn',
  err: 'text-os-err',
  accent: 'text-os-accent',
  dim: 'text-os-dim',
};

/** Live connector map  -  a bar per connector, colored by real state. Honest
    stand-in for a time series we don't store (connector uptime has no history). */
function ConnectorBars({ connections }: { connections: ConnectorStatus[] }) {
  const w = 72;
  const h = 22;
  const gap = 2;
  const items = connections.slice(0, 16);
  const bw = Math.max(2, (w - gap * (items.length - 1)) / Math.max(1, items.length));
  const color = (s: string) => (s === 'connected' ? 'var(--ok)' : s === 'error' ? 'var(--err)' : 'var(--dim)');
  const barH = (s: string) => (s === 'connected' ? h - 4 : s === 'error' ? h - 9 : 6);
  return (
    <svg width={w} height={h} viewBox={`0 0 ${w} ${h}`} aria-hidden="true">
      {items.map((c, i) => {
        const bh = barH(c.state);
        return (
          <rect key={c.id} x={(i * (bw + gap)).toFixed(1)} y={(h - bh).toFixed(1)} width={bw.toFixed(1)} height={bh} fill={color(c.state)} opacity={c.state === 'connected' ? 1 : 0.75} />
        );
      })}
    </svg>
  );
}

/** Real health meter  -  fills to the current score. No fabricated trend line. */
/** Ten fixed cells, lit while `i < score/10`  -  the artboard's G-Brain foot.
    A continuous rail reads as a percentage bar; the score is a graded check,
    and ten cells say so. Unlit cells stay on --border so the track is legible
    without competing with the lit run. */
function HealthMeter({ value }: { value: number | null }) {
  const CELLS = 10;
  const lit = value == null ? 0 : Math.round(Math.max(0, Math.min(100, value)) / CELLS);
  const color = value == null ? 'var(--dim)' : value >= 70 ? 'var(--accent)' : value >= 40 ? 'var(--warn)' : 'var(--err)';
  return (
    <div className="flex items-end gap-[2px]" style={{ height: 18 }} aria-hidden="true">
      {Array.from({ length: CELLS }, (_, i) => (
        <span
          key={i}
          className="flex-1"
          style={{
            height: 5,
            background: i < lit ? color : 'var(--border)',
            transition: 'background-color var(--dur-lens) var(--ease-lens)',
          }}
        />
      ))}
    </div>
  );
}

function greeting(): string {
  const hour = new Date().getHours();
  if (hour < 5) return 'Late night';
  if (hour < 12) return 'Good morning';
  if (hour < 18) return 'Good afternoon';
  return 'Good evening';
}

function relativeTime(iso: string): string {
  const ms = Date.now() - new Date(iso).getTime();
  if (!Number.isFinite(ms) || ms < 0) return 'just now';
  const m = Math.floor(ms / 60_000);
  if (m < 1) return 'just now';
  if (m < 60) return `${m}m`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h`;
  return `${Math.floor(h / 24)}d`;
}

/** Clickable pulse tile that routes to its detail page. */
function StatTile({
  href,
  label,
  value,
  unit,
  foot,
  valueClass = '',
}: {
  href: string;
  label: string;
  value: React.ReactNode;
  unit: string;
  foot: React.ReactNode;
  valueClass?: string;
}) {
  return (
    <Link
      href={href}
      data-lens="r" data-part="tile" className="pressable is-row group flex flex-col gap-2 rounded-tile border border-os-border bg-os-surface px-[18px] py-4"
    >
      <div className="flex items-center justify-between">
        <Label>{label}</Label>
        {/* the artboard keeps this visible at rest in #5c5c5c  -  it is the
            tile's "this opens something" tell, and a tell you only see once
            you are already hovering has told you nothing */}
        <span aria-hidden="true" className="lens-child font-mono text-[11px] leading-none text-os-dim group-hover:text-os-text">
          ↗
        </span>
      </div>
      <div className={`flex items-baseline gap-[7px] font-mono text-[26px] font-semibold tracking-[-0.02em] ${valueClass}`}>
        {value}
        <small className="whitespace-nowrap text-xs font-normal text-os-dim">{unit}</small>
      </div>
      {foot}
    </Link>
  );
}

type DoneItem = { key: string; time: number; head: string; headClass: string; body: string; when: string };

export default async function HomePage() {
  const db = getDb();
  const [connections, overview, feed, stripe] = await Promise.all([
    allConnectorStatuses(),
    createGBrainProvider().overview(),
    gatherCommsFeed(),
    // Fail-soft: no key (or a Stripe outage) means no charges row, never a 500.
    stripeSnapshot().catch(() => null),
  ]);

  const agents = db.agents.all();
  const recentRuns = db.agentRuns.recent(40);
  // The runs/day sparkline and the 14-day activity line, bounded in SQL (one
  // spare day so a local-day window never loses its first morning to UTC).
  const runsForSpark = db.agentRuns.since(new Date(Date.now() - 15 * 86_400_000).toISOString());

  const connected = connections.filter((c) => c.state === 'connected').length;
  // "Down" = genuinely erroring only; not_configured means no key set, not broken.
  const connectorsDown = connections.filter((c) => c.state === 'error').length;
  const activeAgents = agents.filter((a) => a.status === 'active').length;
  const health = overview.doctor.healthScore;
  const inbound = inboundLast24h(feed);
  const failedRuns = recentRuns.filter((r) => !r.ok).length;
  // Real sparkline series from actual history  -  no synthetic arrays.
  const agentsSpark = runsPerDay(runsForSpark, 7);
  const commsSpark = inboundPerDay(feed, 7);
  const hero = stateOfWorld({
    activeAgents,
    totalAgents: agents.length,
    connectorsDown,
    inbound,
    health: health ?? null,
    brainConnected: overview.doctor.connected,
    failedRuns,
  });

  // Done today  -  what the OS actually finished since local midnight: agent
  // runs that ended OK (collapsed so the 30-minute crons cost one line) plus
  // Stripe charges that landed today. Newest first, honest and small.
  const dayStart = new Date();
  dayStart.setHours(0, 0, 0, 0);
  const doneRuns: DoneItem[] = collapseRuns(recentRuns)
    .filter((r) => r.ok && new Date(r.finishedAt).getTime() >= dayStart.getTime())
    .map((r) => ({
      key: `run-${r.id}`,
      time: new Date(r.finishedAt).getTime(),
      head: '✓',
      headClass: 'text-os-ok',
      body: `${r.agentId} · ${r.summary}${r.repeat > 1 ? ` ×${r.repeat}` : ''}`,
      when: relativeTime(r.finishedAt),
    }));
  const doneCharges: DoneItem[] = (stripe?.recentCharges ?? [])
    .filter((c) => c.created * 1000 >= dayStart.getTime())
    .map((c) => ({
      key: `charge-${c.created}-${c.amount}`,
      time: c.created * 1000,
      // the artboard's ledger reads "$  Stripe  Vantage LLC paid $6,000 ·
      // August retainer": the glyph is the column, the money is in the line
      head: '$',
      headClass: 'text-os-accent',
      body: `${(c.amount / 100).toLocaleString('en-US', { style: 'currency', currency: c.currency.toUpperCase(), maximumFractionDigits: 0 })} · ${c.description || 'charge'}`,
      when: relativeTime(new Date(c.created * 1000).toISOString()),
    }));
  const doneToday = [...doneCharges, ...doneRuns].sort((a, b) => b.time - a.time).slice(0, 8);
  // Brand Deals' Deal Volume card, worn by Home's own numbers (2026-09-24).
  const vol = operatingVolume({
    connected,
    totalConnections: connections.length,
    activeAgents,
    totalAgents: agents.length,
    health: health ?? null,
    runs: runsForSpark,
  });
  const activity = dailySeries(runsForSpark.map((r) => r.startedAt), 14);
  const activityTotal = activity.reduce((n, d) => n + d.count, 0);
  const mix = sourceMix(feed);
  // The ledger shows 8 collapsed lines; the COUNT is every run that finished
  // OK today (from the wide pull) plus every charge that landed today.
  const doneCount = vol.runsToday - vol.failedToday + doneCharges.length;
  const attention = homeAttention({ inbound, failedToday: vol.failedToday, connectorsDown, doneToday: doneCount });
  const chargedToday = (stripe?.recentCharges ?? [])
    .filter((c) => c.created * 1000 >= dayStart.getTime())
    .reduce((sum, c) => sum + c.amount / 100, 0);

  return (
    <div className="os-slab">
      <SlabTitle
        eyebrow="command center"
        title={`${greeting()}, Alex`}
        meta={
          /* Honest state-of-the-world line  -  what needs you, straight from live data */
          <span className="flex flex-wrap items-center gap-x-2 gap-y-1 text-[12px]">
            {hero.map((s, i) => (
              <span key={i} className="flex items-center gap-2">
                {i > 0 && <span className="text-os-border-strong">·</span>}
                <span className={TONE_CLASS[s.tone]}>{s.text}</span>
              </span>
            ))}
          </span>
        }
        right={<Kbd>⌘K</Kbd>}
      />

      {/* Pulse row */}
      <Rise as="section" i={1} className="mb-6 grid grid-cols-4 gap-3 max-[1100px]:grid-cols-2">
        <StatTile
          href="/integrations"
          label="Systems"
          value={<CountUp value={connected} />}
          unit={`/ ${connections.length} connected`}
          foot={<ConnectorBars connections={connections} />}
        />
        <StatTile
          href="/agents"
          label="Agents live"
          value={<CountUp value={activeAgents} />}
          unit={`/ ${agents.length} roster`}
          foot={<SparkBars data={agentsSpark} />}
        />
        <StatTile
          href="/comms"
          label="Communications"
          value={<CountUp value={inbound} />}
          unit="inbound · 24h"
          foot={<SparkBars data={commsSpark} />}
        />
        <StatTile
          href="/brain"
          label="G-Brain health"
          value={health == null ? ' - ' : <CountUp value={health} />}
          unit={`/ 100${overview.doctor.connected ? ` · ${overview.doctor.status}` : ' · offline'}`}
          valueClass="text-os-accent"
          foot={<HealthMeter value={health ?? null} />}
        />
      </Rise>

      {/* Hero row, Brand Deals' shape: the queue that needs Alex + the volume card */}
      <div className="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
        {/* the queue stretches to the volume card's height, like Pipeline does */}
        <Rise as="section" i={2} className="min-w-0 [&>[data-part=card]]:h-full">
          <HomeNeedsYou boardUrl={process.env.PAPERCLIP_API_URL ?? null} />
        </Rise>

        <SlabCard
          i={3}
          title="Operating volume"
          className="flex flex-col self-start"
          action={
            <Link href="/agents" className="rounded-full border border-os-border px-3 py-1 font-mono text-[11px] text-os-dim hover:border-os-border-strong hover:text-os-text">
              runs →
            </Link>
          }
        >
          <div className="flex flex-1 flex-col px-6 pb-6 pt-3">
            <BigStat
              value={vol.runsToday}
              chips={[
                ...(vol.failedToday > 0 ? [{ tone: 'err' as const, text: `${vol.failedToday} failed` }] : []),
                ...(chargedToday > 0
                  ? [{ tone: 'ok' as const, text: `${chargedToday.toLocaleString('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 })} charged` }]
                  : []),
              ]}
              caption={`agent runs today across ${vol.agentsToday} agent${vol.agentsToday === 1 ? '' : 's'}`}
            />
            <MeterStack meters={vol.meters} foot={`${connected} systems · ${activeAgents} agents · brain ${health ?? ' - '}/100`} />
          </div>
        </SlabCard>
      </div>

      {/* Second row: activity line, inbound mix dots, THE gradient card */}
      <div className="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
        <SlabCard i={4} title="Agent activity">
          <div className="px-6 pt-3">
            <BigStat size={30} value={activityTotal} caption="agent runs, last 14 days" />
          </div>
          <StepLine series={activity} hue="var(--send-activity)" empty="No agent runs in the last 14 days." />
        </SlabCard>

        <SlabCard i={5} title="Inbound mix">
          <div className="flex items-end justify-between gap-4 px-6 pb-6 pt-3">
            <div>
              <BigStat size={30} value={feed.length} caption="latest messages in the feed" />
              <Link href="/comms" className="mt-4 inline-block rounded-full border border-os-border px-3 py-1 text-[12px] text-os-muted hover:text-os-text">
                <span className="font-semibold tabular-nums">{inbound}</span> in the last 24h →
              </Link>
            </div>
            <DotMatrix cols={mix} hue="var(--ramp-1)" />
          </div>
        </SlabCard>

        <InsightCard
          i={6}
          badge="Needs you now"
          value={attention.count}
          headline={attention.headline}
          body={`${doneCount} thing${doneCount === 1 ? '' : 's'} already done today.`}
          frac={attention.frac}
        />
      </div>

      {/* Talk to the OS, and what it finished */}
      <div className="mt-6 grid grid-cols-2 items-start gap-6 max-[1100px]:grid-cols-1">
        <Rise as="section" i={7} className="min-w-0">
          <InterjectComposer />
        </Rise>

        <Rise as="section" i={8} className="min-w-0">
          <div data-part="card" className="relative overflow-hidden rounded-panel border border-os-border bg-os-surface">
            <div data-part="card-head" className="flex items-center gap-3 border-b border-os-border px-4 py-2.5">
              <Label>Done today</Label>
              <span className="font-mono text-[10px] text-os-dim">{doneCount} since midnight</span>
              <Link href="/agents" className="ml-auto font-mono text-[10px] text-os-dim linky">
                runs →
              </Link>
            </div>
            {doneToday.length === 0 ? (
              <div className="px-4 py-5 font-mono text-[11px] text-os-dim">nothing finished yet today</div>
            ) : (
              <ul className="flex flex-col">
                {doneToday.map((d) => (
                  <li
                    key={d.key}
                    data-part="row"
                    className="flex items-baseline gap-2.5 border-b border-os-hairline px-4 py-2 font-mono text-[11px] last:border-b-0"
                  >
                    <span className={`shrink-0 font-bold ${d.headClass}`}>{d.head}</span>
                    <span className="min-w-0 flex-1 truncate text-os-dim">{d.body}</span>
                    <span className="shrink-0 text-os-dim">{d.when}</span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </Rise>
      </div>
    </div>
  );
}
