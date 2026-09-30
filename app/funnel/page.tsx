import Link from 'next/link';
import { FunnelLayoutToggle } from '@/components/FunnelLayoutToggle';
import {
  attentionQueue,
  funnelSummary,
  journeyMeta,
  splitFunnelJourneys,
  decayFactor,
  DECAY_DAYS,
  FUNNEL_STAGES,
  CHANNEL_GLYPHS,
} from '@/lib/funnel';
import { funnelSpaceModel } from '@/lib/funnel';
import { funnelRadialModel } from '@/lib/funnel-radial';
import { composeFunnelJourneys } from '@/lib/funnel-compose';
import { lastMessageFor } from '@/lib/funnel-contact';
import { gatherCommsFeed } from '@/lib/comms-feed';
import type { CommsItem } from '@/lib/comms';
import { attioStatus } from '@/lib/connectors/attio';
import { ghlStatus } from '@/lib/connectors/ghl';
import { trakyoStatus } from '@/lib/connectors/trakyo';
import { metaAdsStatus } from '@/lib/connectors/meta-ads';
import { getVenture } from '@/lib/ventures';
import { FunnelRadialLazy, FunnelSpaceLazy } from '@/components/FunnelGraphsLazy';
import { Badge } from '@/components/terminal';
import { Rise } from '@/components/motion';
import { Slab, SlabTitle, SlabCard, BigStat, MeterStack, InsightCard, chipClass } from '@/components/slab';
import { StepLine } from '@/components/slab-charts';
import { funnelVolume } from '@/lib/funnel-volume';
import {
  FunnelStageSchema,
  FunnelVentureSchema,
  type FunnelJourney,
  type FunnelStage,
  type FunnelTouch,
  type FunnelVenture,
} from '@/lib/schemas';
import type { ConnectorStatus } from '@/lib/connectors/types';

export const dynamic = 'force-dynamic';
const STAGE_LABEL = Object.fromEntries(FUNNEL_STAGES.map((s) => [s.id, s.label]));

const VENTURE_TABS: { id: FunnelVenture | 'all'; label: string }[] = [
  { id: 'all', label: 'All clients' },
  { id: 'vantage', label: 'Vantage' },
  { id: 'launchpad-cohort', label: 'Launchpad Cohort' },
];

const VIEWS: { id: 'flow' | 'radial'; label: string }[] = [
  { id: 'flow', label: 'Flow' },
  { id: 'radial', label: 'Radial' },
];

function usd(amount: number): string {
  return `$${Math.round(amount).toLocaleString('en-US')}`;
}

function ventureColor(id: FunnelVenture): string {
  return getVenture(id)?.color ?? 'var(--accent)';
}

/** Compact source check: ✓ when connected, ○ when pending  -  detail on hover. */
function SourceCheck({ status, live, count }: { status: ConnectorStatus; live?: boolean; count?: number }) {
  const ok = status.state === 'connected';
  return (
    <span
      title={status.detail}
      className={`inline-flex items-center gap-1 font-mono text-[9.5px] uppercase tracking-[0.12em] ${
        ok ? (live ? 'text-os-ok' : 'text-os-muted') : 'text-os-dim'
      }`}
    >
      {ok ? '✓' : '○'} {status.name}
      {live && count != null ? ` ${count}` : ''}
    </span>
  );
}

/** One touch on a journey, as a rounded pill: channel glyph, label, date. */
function TouchChip({ touch }: { touch: FunnelTouch }) {
  return (
    <span
      className="inline-flex max-w-[260px] items-center gap-1.5 rounded-full border border-os-border px-2.5 py-1"
      title={`${touch.stage} · via ${touch.source} · ${touch.at}`}
    >
      <span className="shrink-0 font-mono text-[10.5px] text-os-accent">{CHANNEL_GLYPHS[touch.channel] ?? '·'}</span>
      <span className="truncate text-[11.5px] text-os-muted">{touch.label}</span>
      <span className="shrink-0 font-mono text-[10px] text-os-dim">{touch.at.slice(5)}</span>
    </span>
  );
}

const RELATIONSHIP_VAR: Record<FunnelJourney['relationship'], string> = {
  hot: 'var(--funnel-hot)',
  warm: 'var(--funnel-warm)',
  cold: 'var(--funnel-cold)',
};

const PILL_BASE = 'shrink-0 rounded-full px-2.5 py-0.5 font-mono text-[10.5px] uppercase tracking-[0.1em]';

/** Brand Deals status pills: closed reads ok, a stalled lead err, the rest neutral. */
function stagePillStyle(journey: FunnelJourney, now: Date): React.CSSProperties {
  const state = journeyMeta(journey, now).state;
  if (state === 'converted') return { background: 'color-mix(in oklab, var(--ok) 16%, transparent)', color: 'var(--ok)' };
  if (state === 'stalled' || state === 'decayed') return { background: 'color-mix(in oklab, var(--err) 16%, transparent)', color: 'var(--err)' };
  return { background: 'color-mix(in oklab, var(--text) 8%, transparent)', color: 'var(--text-2)' };
}

/** Quiet days fade toward red as a lead decays (same rule as the graph). */
function quietStyle(days: number, journey: FunnelJourney): React.CSSProperties | undefined {
  const decay = decayFactor(days, journey.status);
  return decay > 0 ? { color: `color-mix(in oklab, var(--err) ${Math.round(Math.sqrt(decay) * 85)}%, var(--text-2))` } : undefined;
}

function ValuePill({ amount }: { amount: number }) {
  return (
    <span className="shrink-0 rounded-full px-2.5 py-0.5 font-mono text-[10.5px] tabular-nums tracking-[0.04em]" style={{ background: 'var(--accent-soft)', color: 'var(--accent)' }}>
      {usd(amount)}
    </span>
  );
}

/** Compact outreach links, only the channels this lead actually has. */
function ContactActions({ journey }: { journey: FunnelJourney }) {
  const digits = journey.phone?.replace(/[^\d]/g, '');
  const isCall = journey.url?.includes('fathom');
  return (
    <span className="flex items-center gap-2.5 font-mono text-[10.5px] uppercase tracking-wide">
      {journey.email && (
        <a href={`mailto:${journey.email}`} title={journey.email} className="text-os-muted hover:text-os-accent">
          email
        </a>
      )}
      {digits && (
        <a
          href={`https://wa.me/${digits}`}
          target="_blank"
          rel="noopener noreferrer"
          title={journey.phone ?? undefined}
          className="text-os-muted hover:text-os-accent"
        >
          wa
        </a>
      )}
      {journey.phone && (
        <a href={`sms:${journey.phone}`} title={journey.phone} className="text-os-muted hover:text-os-accent">
          sms
        </a>
      )}
      {journey.url && (
        <a
          href={journey.url}
          target="_blank"
          rel="noopener noreferrer"
          className="text-os-dim hover:text-os-accent"
        >
          {isCall ? 'call↗' : 'open↗'}
        </a>
      )}
      {!journey.email && !journey.phone && !journey.url && <span className="text-os-dim">no contact</span>}
    </span>
  );
}

/** One attention row, roomy: clicking it pins that lead's dossier in the canvas. */
function AttentionRow({
  journey,
  now,
  href,
}: {
  journey: FunnelJourney;
  now: Date;
  href: string;
}) {
  const meta = journeyMeta(journey, now);
  return (
    <Link
      href={href}
      data-lens="r"
      className="pressable is-row group grid grid-cols-[1fr_auto_auto] items-center gap-4 border-b border-os-border px-6 py-3.5 last:border-0 hover:bg-[color-mix(in_oklab,var(--text)_4%,transparent)]"
    >
      <span className="min-w-0">
        <span className="block truncate text-[13.5px] font-medium group-hover:text-os-accent">{journey.person ?? journey.name}</span>
        <span className="block truncate font-mono text-[11px] text-os-dim">
          {[journey.company, `${journey.likelihood}% likely`].filter(Boolean).join(' · ')}
        </span>
      </span>
      <span className="flex items-center gap-2">
        {(journey.amountUsd ?? 0) > 0 && <ValuePill amount={journey.amountUsd ?? 0} />}
        <span className={PILL_BASE} style={stagePillStyle(journey, now)}>
          {STAGE_LABEL[journey.status]}
        </span>
      </span>
      <span className="w-[44px] text-right font-mono text-[11px] tabular-nums text-os-dim" style={quietStyle(meta.daysSinceLastTouch, journey)}>
        {meta.daysSinceLastTouch}d
      </span>
    </Link>
  );
}

const agoDays = (ts: string, now: Date): string => {
  const d = Math.max(0, Math.floor((now.getTime() - Date.parse(ts)) / 86_400_000));
  return d === 0 ? 'today' : `${d}d ago`;
};

/** One roomy row per client: the data line, then their touches. */
function JourneyRow({
  journey,
  now,
  lastMsg,
}: {
  journey: FunnelJourney;
  now: Date;
  /** undefined = lookup not run (no segment selected) · null = no thread found */
  lastMsg?: CommsItem | null;
}) {
  const converted = journey.status === 'converted';
  const meta = journeyMeta(journey, now);
  const entrySource = journey.touches[0]?.source;
  const lane =
    entrySource === 'attio' || entrySource === 'ghl'
      ? 'crm'
      : journey.touches[0]?.channel === 'ads'
        ? 'ads'
        : 'organic';
  return (
    <div data-lens="r" className="is-row border-b border-os-border px-6 py-3.5 last:border-0 hover:bg-[color-mix(in_oklab,var(--text)_4%,transparent)]">
      <div className="grid grid-cols-[minmax(200px,1fr)_auto_auto_auto_auto] items-center gap-5 max-[1000px]:grid-cols-[1fr_auto]">
        <span className="flex min-w-0 items-center gap-2.5">
          <span
            className="h-2 w-2 shrink-0 rounded-full"
            style={{ background: ventureColor(journey.venture) }}
            title={getVenture(journey.venture)?.label}
          />
          <span className="min-w-0">
            <span className="block truncate text-[13.5px] font-medium" title={journey.name}>
              {journey.person ?? journey.name}
            </span>
            <span className="block truncate font-mono text-[11px] text-os-dim">
              {[journey.company, converted && journey.product ? journey.product : null].filter(Boolean).join(' · ') || lane}
            </span>
          </span>
        </span>
        <span className="flex items-center gap-2">
          {converted && <ValuePill amount={journey.amountUsd ?? 0} />}
          <span className={PILL_BASE} style={{ background: `color-mix(in oklab, ${RELATIONSHIP_VAR[journey.relationship]} 16%, transparent)`, color: RELATIONSHIP_VAR[journey.relationship] }}>
            {journey.relationship}
          </span>
          <span className={PILL_BASE} style={stagePillStyle(journey, now)}>
            {STAGE_LABEL[journey.status]}
          </span>
        </span>
        <span
          className={`w-[64px] font-mono text-[11px] tabular-nums max-[1000px]:hidden ${meta.state === 'stalled' && decayFactor(meta.daysSinceLastTouch, journey.status) === 0 ? 'text-os-err' : 'text-os-dim'}`}
          style={quietStyle(meta.daysSinceLastTouch, journey)}
          title="days since the last touch"
        >
          {meta.daysSinceLastTouch}d quiet
        </span>
        <span className="w-[72px] font-mono text-[11px] tabular-nums text-os-muted max-[1000px]:hidden" title="likelihood to buy">
          {journey.likelihood}% likely
        </span>
        <span className="max-[1000px]:hidden">
          <ContactActions journey={journey} />
        </span>
      </div>
      <div className="mt-2.5 flex flex-wrap items-center gap-1.5 pl-[18px]">
        <span className="mr-1 font-mono text-[10.5px] uppercase tracking-wide text-os-dim" title="entry lane">
          {lane}
        </span>
        {journey.touches.map((t, i) => (
          <span key={t.id} className="flex items-center gap-1.5">
            {i > 0 && <span className="font-mono text-[10px] text-os-dim">→</span>}
            <TouchChip touch={t} />
          </span>
        ))}
        {!converted && <span className="font-mono text-[10px] text-os-dim">→ …</span>}
        {lastMsg !== undefined && (
          <span className="ml-auto flex min-w-0 items-center gap-1.5 font-mono text-[10.5px]">
            <span className="shrink-0 uppercase tracking-wide text-os-dim">last msg</span>
            {lastMsg ? (
              <span className="min-w-0 truncate text-os-muted" title={lastMsg.preview}>
                via {lastMsg.source} · {agoDays(lastMsg.ts, now)} · “{lastMsg.preview.slice(0, 60)}”
              </span>
            ) : (
              <span className="text-os-dim">no thread on record</span>
            )}
          </span>
        )}
      </div>
    </div>
  );
}

export default async function FunnelPage({
  searchParams,
}: {
  searchParams?: { venture?: string; view?: string; stage?: string; layout?: string; lead?: string };
}) {
  const parsed = FunnelVentureSchema.safeParse(searchParams?.venture);
  const venture = parsed.success ? parsed.data : undefined;
  const view = searchParams?.view === 'archive' ? 'archive' : 'live';
  const stageParsed = FunnelStageSchema.safeParse(searchParams?.stage);
  const stage = stageParsed.success ? stageParsed.data : undefined;
  // Two ways to see the same journeys: hubs left → right, or the circle
  // running outside → in (acquisition wedges around the rim, purchase center).
  const layout = searchParams?.layout === 'radial' ? 'radial' : 'flow';
  const href = (
    v: FunnelVenture | undefined,
    w: 'live' | 'archive',
    s: FunnelStage | undefined = stage,
    l: 'flow' | 'radial' = layout,
    leadId?: string,
  ) => {
    const params = new URLSearchParams();
    if (v) params.set('venture', v);
    if (w === 'archive') params.set('view', 'archive');
    if (s) params.set('stage', s);
    if (l === 'radial') params.set('layout', 'radial');
    if (leadId) params.set('lead', leadId);
    const qs = params.toString();
    return qs ? `/funnel?${qs}` : '/funnel';
  };

  const now = new Date();
  // Same composer the /api/funnel route uses, so the page can never again miss
  // a source the route has (this is how every Stripe buyer went invisible):
  // Typeform leads, calendar bookings + Fathom calls, Trakyo attribution and
  // Stripe payments folded on, venture-filtered; seeded funnel otherwise.
  const composed = await composeFunnelJourneys(now, venture);
  const { attioLive, ghlLive, isLive } = composed;
  const excludedCount = (attioLive?.closedLost ?? 0) + (ghlLive?.excluded ?? 0);
  const liveLabel = [
    attioLive && attioLive.journeys.length > 0 ? `Attio ${attioLive.total}` : null,
    ghlLive && ghlLive.journeys.length > 0 ? `GHL ${ghlLive.total}` : null,
    composed.stripeWins && composed.stripeWins.length > 0 ? `Stripe ${composed.stripeWins.length}` : null,
  ]
    .filter(Boolean)
    .join(' + ');
  const allJourneys = composed.journeys;
  // Quiet past DECAY_DAYS → out of the space, into the archive tab.
  const { active: journeys, archived } = splitFunnelJourneys(allJourneys, now);
  const summary = funnelSummary(journeys);
  const radial = layout === 'radial' ? funnelRadialModel(journeys, now) : null;
  const spaceNodes = layout === 'flow' ? funnelSpaceModel(journeys, now) : null;
  // ?lead= (attention-rail clicks) pins that lead's dossier in the canvas
  const lead = journeys.some((j) => j.id === searchParams?.lead) ? searchParams?.lead : undefined;
  const attention = attentionQueue(journeys, now);

  // Segment select: the table narrows to one stage; with a bounded row set we
  // can afford the live comms lookup (last message per lead, honest on miss).
  const tableJourneys = stage ? journeys.filter((j) => j.status === stage) : journeys;
  const stageCounts = new Map<FunnelStage, number>();
  for (const j of journeys) stageCounts.set(j.status, (stageCounts.get(j.status) ?? 0) + 1);
  let commsFeed: CommsItem[] | null = null;
  if (stage && tableJourneys.length > 0) {
    commsFeed = await gatherCommsFeed(200).catch(() => null);
  }
  const [attio, ghl, trakyo, metaAds] = await Promise.all([
    attioStatus(),
    ghlStatus(),
    trakyoStatus(),
    metaAdsStatus(),
  ]);
  // Funnel Volume, the step line and the insight card (lib/funnel-volume).
  const vol = funnelVolume({ journeys, archived: archived.length, now });

  return (
    <Slab>
      <SlabTitle
        eyebrow="client journeys · leads → conversations → sales"
        title="Funnel"
        meta={`${summary.clients} active client${summary.clients === 1 ? '' : 's'} · ${summary.converted} closed · ${usd(summary.revenueUsd)} revenue · ${archived.length} archived`}
        right={
          <>
            {isLive ? (
              <Badge tone="ok">live · {liveLabel}</Badge>
            ) : (
              <Badge tone="warn" ghost>
                demo data
              </Badge>
            )}
            <span className="rounded-full border border-os-border px-4 py-2 text-[13px] tabular-nums text-os-muted">
              {summary.converted}/{summary.clients} converted · {usd(summary.revenueUsd)}
            </span>
          </>
        }
      />

      {/* one control line: venture filter · synced sources · view toggle */}
      <Rise i={1} className="mb-3 flex flex-wrap items-center gap-x-3 gap-y-1.5">
        <span className="flex items-center gap-1.5">
          {VENTURE_TABS.map((tab) => {
            const active = (venture ?? 'all') === tab.id;
            return (
              <Link
                key={tab.id}
                href={href(tab.id === 'all' ? undefined : (tab.id as FunnelVenture), view)}
                title={tab.id !== 'all' && isLive ? 'Live split = name heuristic: Vantage when the form or call names it' : undefined}
                data-lens="c" className={`pressable rounded-ctl border px-2 py-0.5 font-mono text-[10px] uppercase tracking-wide ${
 active
 ? 'border-[var(--accent-line)] bg-[var(--accent-soft)] text-os-accent'
 : 'border-os-border text-os-dim hover:border-os-border-strong hover:text-os-muted'
 }`}
              >
                {tab.id !== 'all' && (
                  <span
                    className="mr-1 inline-block h-1.5 w-1.5 rounded-full align-middle"
                    style={{ background: ventureColor(tab.id as FunnelVenture) }}
                  />
                )}
                {tab.label}
              </Link>
            );
          })}
        </span>
        <span className="h-3 w-px bg-os-border" />
        <span className="flex items-center gap-2.5" title="leads · calls booked · calls held · attribution · paid">
          <SourceCheck status={attio} live={Boolean(attioLive?.total)} count={attioLive?.total} />
          <SourceCheck status={ghl} live={Boolean(ghlLive?.total)} count={ghlLive?.total} />
          <SourceCheck status={trakyo} />
          <SourceCheck status={metaAds} />
        </span>
        <span className="ml-auto flex items-center gap-1.5">
          <FunnelLayoutToggle layout={layout} archived={view === 'archive'}
            options={VIEWS.map(v => ({ ...v, href: href(venture, 'live', stage, v.id, lead) }))} />
          <span className="mx-0.5 h-3 w-px bg-os-border" />
          <Link
            href={view === 'archive' ? href(venture, 'live') : href(venture, 'archive')}
            title="Leads quiet past the decay window rest here"
            data-lens="c"
            className={`pressable rounded-ctl border px-2 py-0.5 font-mono text-[10px] uppercase tracking-wide ${
 view === 'archive'
 ? 'border-[var(--accent-line)] bg-[var(--accent-soft)] text-os-accent'
 : 'border-os-border text-os-dim hover:border-os-border-strong hover:text-os-muted'
 }`}
          >
            Archive ({archived.length})
          </Link>
        </span>
      </Rise>

      {/* The space  -  every node is a client travelling toward conversion.
          Leads quiet past DECAY_DAYS decay into the archive tab. */}
      {view === 'archive' ? (
        <SlabCard title="Archive" sub={`${archived.length} quiet past ${DECAY_DAYS} days`} i={3}>
          <div className="mt-4 border-t border-os-border">
            {archived.length === 0 ? (
              <p className="px-6 py-8 text-center text-[12.5px] text-os-dim">
                Nothing decayed · no lead has sat quiet past {DECAY_DAYS} days.
              </p>
            ) : (
              archived.map((j) => {
                const meta = journeyMeta(j, now);
                const last = j.touches[j.touches.length - 1];
                return (
                  <div
                    key={j.id}
                    className="grid grid-cols-[minmax(200px,1fr)_auto_auto_auto] items-center gap-5 border-b border-os-border px-6 py-3.5 last:border-0 max-[1000px]:grid-cols-[1fr_auto]"
                  >
                    <span className="flex min-w-0 items-center gap-2.5">
                      <span className="h-2 w-2 shrink-0 rounded-full opacity-60" style={{ background: ventureColor(j.venture) }} />
                      <span className="min-w-0">
                        <span className="block truncate text-[13.5px] font-medium text-os-muted">{j.name}</span>
                        <span className="block truncate font-mono text-[11px] text-os-dim" title={last?.label}>
                          {last?.label ?? 'no touches'}
                        </span>
                      </span>
                    </span>
                    <span className={PILL_BASE} style={stagePillStyle(j, now)}>
                      {STAGE_LABEL[j.status]}
                    </span>
                    <span className="font-mono text-[11px] tabular-nums text-os-dim max-[1000px]:hidden">
                      {meta.daysSinceLastTouch}d quiet · {j.likelihood}% likely
                    </span>
                    <span className="w-[56px] text-right max-[1000px]:hidden">
                      {j.url && (
                        <a
                          href={j.url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="font-mono text-[10.5px] uppercase tracking-wide text-os-dim hover:text-os-accent"
                        >
                          {j.url.includes('fathom') ? 'call ↗' : 'open ↗'}
                        </a>
                      )}
                    </span>
                  </div>
                );
              })
            )}
          </div>
        </SlabCard>
      ) : (
        <Rise as="section" i={2}>
          <div className="rounded-lg-t border border-os-border bg-os-surface p-2">
            {radial ? (
              <FunnelRadialLazy model={radial} initialLeadId={lead} />
            ) : spaceNodes ? (
              <FunnelSpaceLazy nodes={spaceNodes} summary={summary} initialLeadId={lead} />
            ) : null}
          </div>
        </Rise>
      )}

      {/* Below the graph, the Brand Deals rows: volume, activity, THE insight. */}
      {view === 'live' && (
        <div className="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
          <SlabCard title="Funnel Volume" sub={`${summary.clients} active`} i={4} className="flex flex-col">
            <div className="flex flex-1 flex-col px-6 pb-6 pt-3">
              <BigStat value={vol.revenueUsd} kind="usd" chips={vol.chips} caption={vol.caption} />
              <MeterStack meters={vol.meters} foot={vol.foot} empty="no leads yet" />
            </div>
          </SlabCard>

          <SlabCard title="Lead Activity" sub="last 30 days" i={5}>
            <div className="px-6 pt-3">
              <BigStat size={30} value={vol.touchesInWindow} caption="touches across every journey: forms, bookings, calls, payments" />
            </div>
            <StepLine series={vol.series} hue="var(--send-activity)" unit=" touches" empty="No touches in this window." />
          </SlabCard>

          <InsightCard
            i={6}
            badge="Needs you today"
            value={vol.insight.value}
            headline={vol.insight.headline}
            body={vol.insight.body}
            frac={vol.insight.frac}
          />
        </div>
      )}

      {/* What to act on today: the funnel answering a question. Every row
          click pins that lead's dossier in the canvas above. */}
      {view === 'live' && (attention.pushNow.length > 0 || attention.saveNow.length > 0) && (
        <div className="mt-6 grid grid-cols-1 gap-6 lg:grid-cols-2">
          <SlabCard title="Push Now" sub="hot + moving · close them" i={7}>
            <div className="mt-4 border-t border-os-border">
              {attention.pushNow.length === 0 ? (
                <p className="px-6 py-6 text-center text-[12.5px] text-os-dim">no hot leads in motion right now</p>
              ) : (
                attention.pushNow.map((j) => (
                  <AttentionRow key={j.id} journey={j} now={now} href={href(venture, view, stage, layout, j.id)} />
                ))
              )}
            </div>
          </SlabCard>
          <SlabCard title="Save Now" sub="fading toward the archive · highest likelihood first" i={8}>
            <div className="mt-4 border-t border-os-border">
              {attention.saveNow.length === 0 ? (
                <p className="px-6 py-6 text-center text-[12.5px] text-os-dim">nothing fading · every lead is fresh</p>
              ) : (
                attention.saveNow.map((j) => (
                  <AttentionRow key={j.id} journey={j} now={now} href={href(venture, view, stage, layout, j.id)} />
                ))
              )}
            </div>
          </SlabCard>
        </div>
      )}


      {/* The same clients as a list: pick a segment, contact them. */}
      <SlabCard
        title="Journeys"
        sub={`${tableJourneys.length} of ${journeys.length}`}
        i={10}
        className="mt-6"
        action={
          <>
            <Link href={href(venture, view, undefined)} data-lens="c" className={chipClass(!stage)}>
              All {journeys.length}
            </Link>
            {FUNNEL_STAGES.map((s, i) => {
              const active = stage === s.id;
              return (
                <Link
                  key={s.id}
                  href={href(venture, view, active ? undefined : s.id)}
                  data-lens="c"
                  className={`${chipClass(active)} inline-flex items-center gap-1.5`}
                >
                  <span className="inline-block h-1.5 w-1.5 rounded-full" style={{ background: `var(--funnel-s${i})` }} />
                  {s.label} {stageCounts.get(s.id) ?? 0}
                </Link>
              );
            })}
          </>
        }
      >
        {stage && !commsFeed && tableJourneys.length > 0 && (
          <p className="px-6 pt-3 font-mono text-[11px] text-os-dim">comms feed unavailable · last messages hidden</p>
        )}
        <div className="mt-4 border-t border-os-border">
          {tableJourneys.length === 0 ? (
            <p className="px-6 py-8 text-center text-[12.5px] text-os-dim">No leads in this segment.</p>
          ) : (
            tableJourneys.map((j) => (
              <JourneyRow key={j.id} journey={j} now={now} lastMsg={commsFeed ? lastMessageFor(j, commsFeed) : undefined} />
            ))
          )}
        </div>
      </SlabCard>
    </Slab>
  );
}
