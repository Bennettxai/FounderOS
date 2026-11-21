import Link from 'next/link';
import { Instagram, Linkedin, Music2, Twitter, Youtube, type LucideIcon } from 'lucide-react';
import { getDb } from '@/lib/data';
import { buildSocialDashboard, audienceGrowthPct, audienceSeries, audienceTotal, PLATFORM_LABELS } from '@/lib/social';
import { syncFromZernioLive } from '@/lib/social-live';
import { buildEmailList, syncBeehiivEmail } from '@/lib/email-list';
import { channelColor } from '@/lib/series';
import { agentRunVolume, runsWithin } from '@/lib/analytics';
import { splitMetrics, type MetricInput, type MetricTile } from '@/lib/operating-metrics';
import { SetupLink } from '@/components/SetupLink';
import { setupHrefFor } from '@/lib/integrations-catalog';

/** Pending metric → the connector whose connect flow fixes it. */
const PENDING_CONNECTOR: Record<string, string> = {
  audience: 'zernio',
  subscribers: 'beehiiv',
  pipeline: 'attio',
  stripe: 'payments',
  unread: 'email',
  dictations: 'wispr',
  brain: 'gbrain',
};
import { attioStatus } from '@/lib/connectors/attio';
import { cachedStatus } from '@/lib/connectors';
import { wisprStatus } from '@/lib/connectors/wispr';
import { readStoreNotes } from '@/lib/connectors/gbrain';
import { stripeSnapshot } from '@/lib/connectors/payments';
import { unreadCounts } from '@/lib/connectors/email';
import type { SocialPlatform } from '@/lib/schemas';
import type { PieItem } from '@/lib/social-chart';
import { PageHeader } from '@/components/PageHeader';
import { Badge, Label, SectionHead, Spark } from '@/components/terminal';
import { SharePie } from '@/components/SharePie';
import { formatFollowers, GrowthBadge } from '@/components/SocialStats';
import { AnalyticsG, type AnalyticsTileG } from '@/components/analytics/AnalyticsG';
import { getAppearance } from '@/lib/appearance-server';
import { treeFor } from '@/lib/appearance';
import { channelsNetChange, netChange, seriesWindow } from '@/lib/social-range';
import { formatFull } from '@/lib/format-number';
import { SERIES_INDEX } from '@/lib/series';

export const dynamic = 'force-dynamic';

const PLATFORM_ICONS: Record<SocialPlatform, LucideIcon> = {
  instagram: Instagram,
  tiktok: Music2,
  twitter: Twitter,
  youtube: Youtube,
  linkedin: Linkedin,
};

// Value + small-unit split per tile (compact for audience, $ for money).
function tileValue(value: number, unit: string): { main: string; small: string } {
  if (unit === 'usd') return { main: `$${value.toLocaleString('en-US')}`, small: '' };
  if (unit === 'followers') return { main: formatFollowers(value), small: '' };
  return { main: value.toLocaleString('en-US'), small: unit };
}

// Sparks only where real history exists (H21): no invented shapes. A tile
// without history renders no spark at all.
const SPARK_POINTS = 14;

function fmtShort(iso: string): string {
  return new Date(`${iso}T00:00:00Z`)
    .toLocaleDateString('en-US', { month: 'short', day: 'numeric', timeZone: 'UTC' })
    .toLowerCase();
}

const fmtCount = (n: number) => n.toLocaleString('en-US');

/** Responsive area chart for daily agent-run counts (real log, honest zeros). */
function RunVolumeChart({ data }: { data: { date: string; count: number }[] }) {
  const W = 640;
  const H = 170;
  const pad = 8;
  const max = Math.max(...data.map((d) => d.count), 1);
  const n = data.length;
  const xAt = (i: number) => (n <= 1 ? W / 2 : (i / (n - 1)) * W);
  const yAt = (v: number) => H - pad - (v / max) * (H - pad * 2);
  const pts = data.map((d, i) => `${xAt(i).toFixed(1)},${yAt(d.count).toFixed(1)}`);
  return (
    <svg viewBox={`0 0 ${W} ${H}`} className="h-[170px] w-full" preserveAspectRatio="none" aria-hidden="true">
      {[0.25, 0.5, 0.75].map((g) => (
        <line key={g} x1="0" x2={W} y1={H * g} y2={H * g} stroke="var(--border)" strokeWidth="1" />
      ))}
      <polygon points={`0,${H} ${pts.join(' ')} ${W},${H}`} fill="var(--accent)" opacity="0.1" />
      <polyline
        points={pts.join(' ')}
        fill="none"
        stroke="var(--accent)"
        strokeWidth="2"
        strokeLinejoin="round"
        vectorEffect="non-scaling-stroke"
      />
    </svg>
  );
}

function MetricCard({ tile, spark }: { tile: MetricTile; spark?: number[] }) {
  const up = tile.delta > 0;
  const flat = tile.delta === 0;
  const { main, small } = tileValue(tile.value, tile.unit);
  return (
    <div className="hoverable flex flex-col gap-3.5 rounded-lg-t border border-os-border bg-os-surface px-[18px] py-4">
      <div className="flex items-center justify-between gap-3">
        <Label>{tile.label}</Label>
        <span className={`text-[12px] tabular-nums ${flat ? 'text-os-dim' : up ? 'text-os-ok' : 'text-os-err'}`}>
          {flat ? '' : up ? '▲ +' : '▼ '}
          {flat ? '' : tile.delta}
          {!flat && tile.deltaPct ? '%' : ''}
        </span>
      </div>
      <div className="flex items-baseline gap-3 text-[30px] font-normal leading-none tracking-[-0.01em] tabular-nums">
        {main}
        {small && <small className="text-[12.5px] font-normal tracking-normal text-os-dim">{small}</small>}
      </div>
      <div className="flex items-end justify-between gap-3">
        {spark && spark.length > 1 ? <Spark data={spark} w={96} h={26} /> : <span />}
        <span className="text-[11.5px] text-os-dim">{tile.source}</span>
      </div>
    </div>
  );
}

/** A titled card that hosts one share donut — the Distribution row's unit. */
function PieCard({
  title, sub, items, total, centerLabel, format, ariaLabel,
}: {
  title: string;
  sub: string;
  items: PieItem[];
  total: number;
  centerLabel: string;
  format: (v: number) => string;
  ariaLabel: string;
}) {
  return (
    <div className="rounded-lg-t border border-os-border bg-os-surface p-6">
      <div className="mb-4 flex items-baseline justify-between gap-3">
        <Label>{title}</Label>
        <span className="text-[11.5px] text-os-dim">{sub}</span>
      </div>
      <SharePie
        framed={false}
        stacked
        donutPx={158}
        items={items}
        total={total}
        centerLabel={centerLabel}
        format={format}
        ariaLabel={ariaLabel}
      />
    </div>
  );
}

export default async function AnalyticsPage() {
  const db = getDb();
  // Every live read at once (perf, SPEC 4.1 a5); each is a cached real result (lib/swr.ts) or, on a
  // cold server, one real call. The same live syncs as /social, so both pages read one audience
  // (H22), plus the connector reads behind the tiles (each degrades to pending).
  const [, beehiivLive, [attio, wispr, stripe, emailUnread]] = await Promise.all([
    syncFromZernioLive(db),
    syncBeehiivEmail(db),
    Promise.all([
      cachedStatus('attio', attioStatus).catch(() => null),
      cachedStatus('wispr', wisprStatus).catch(() => null),
      stripeSnapshot().catch(() => null),
      unreadCounts()
        .then((cs) => cs.reduce((sum, c) => sum + c.unread, 0))
        .catch(() => null),
    ]),
  ]);
  const today = new Date().toISOString().slice(0, 10);

  // Real agent-run activity — powers the agent-runs tile, the volume chart, and
  // the run-distribution pies.
  const runs = db.agentRuns.recent(2000);
  const runVolume = agentRunVolume(runs, today, 14);
  const windowRuns = runVolume.reduce((s, p) => s + p.count, 0);
  const runs7d = runsWithin(runs, today, 7);

  // Real audience: one definition with /social (H22, C19). "Total audience"
  // is every platform plus the email list, from audienceTotal.
  const dash = buildSocialDashboard(db);
  const totalAudience = audienceTotal(db);
  const audience7d = audienceGrowthPct(db, 7);
  const audience = audienceSeries(db);
  const emailSubs = buildEmailList(db).subscribers;

  // Subscribers is live only when Beehiiv answered; then it is the same
  // number the /social email cell shows.
  const subs = beehiivLive ? emailSubs : null;
  const pipelineDeals = attio?.state === 'connected' ? Number(attio.meta?.deals ?? 0) : null;
  const dictations = wispr?.state === 'connected' ? Number(wispr.meta?.dictations ?? 0) : null;
  const stripeAvail = stripe ? Math.round((stripe.available[0]?.amount ?? 0) / 100) : null;
  let brainPages = 0;
  try {
    brainPages = readStoreNotes().length;
  } catch {
    brainPages = 0;
  }

  // Every tile is a real connector read, or honest pending (value === null).
  const inputs: MetricInput[] = [
    {
      id: 'audience',
      label: 'Total audience',
      unit: 'followers',
      source: '7d · Zernio + email',
      value: totalAudience || null,
      delta: audience7d != null ? Math.round(audience7d * 10) / 10 : 0,
      deltaPct: audience7d != null,
    },
    { id: 'subscribers', label: 'Subscribers', unit: 'subs', source: 'Beehiiv', value: subs },
    { id: 'pipeline', label: 'Open Pipeline', unit: 'deals', source: 'Attio', value: pipelineDeals },
    { id: 'stripe', label: 'Stripe Available', unit: 'usd', source: 'Stripe', value: stripeAvail },
    { id: 'agent-runs', label: 'Agent Runs', unit: 'runs', source: 'all time', value: runs.length || null, delta: runs7d },
    { id: 'unread', label: 'Unread · all inboxes', unit: 'emails', source: 'Email', value: emailUnread },
    { id: 'brain', label: 'Brain-store Pages', unit: 'pages', source: 'Optimal Engine', value: brainPages },
    { id: 'dictations', label: 'Dictations', unit: 'dictations', source: 'Wispr Flow', value: dictations },
  ];
  const { live, pending } = splitMetrics(inputs);
  const sparks: Record<string, number[]> = {
    audience: audience.all.points.slice(-SPARK_POINTS).map((p) => p.value),
    // a 14-day run spark only when the window had runs (else it is a flat zero
    // under an all-time total; the volume card explains the gap)
    'agent-runs': windowRuns > 0 ? runVolume.map((p) => p.count) : [],
  };
  const platformSpark = new Map(
    audience.channels.map((c) => [c.key, c.points.slice(-SPARK_POINTS).map((p) => p.value)]),
  );
  const lastRunAt = runs[0]?.startedAt ?? null;

  // ---- Distribution pies (all real: live snapshots + the real run log) ----

  // Audience share by channel — every social platform plus the email list.
  const audienceItems: PieItem[] = dash.platforms.map((p) => ({
    key: p.platform,
    label: PLATFORM_LABELS[p.platform],
    value: p.followers,
    color: channelColor(p.platform),
  }));
  if (emailSubs) audienceItems.push({ key: 'email', label: 'Email list', value: emailSubs, color: channelColor('email') });
  const audienceReach = totalAudience;

  // Agent runs by agent — top handful, the long tail folded into "Other".
  const agentName = new Map(db.agents.all().map((a) => [a.id, a.name]));
  const byAgent = new Map<string, number>();
  for (const r of runs) byAgent.set(r.agentId, (byAgent.get(r.agentId) ?? 0) + 1);
  const rankedAgents = [...byAgent.entries()].sort((a, b) => b[1] - a[1]);
  const runsByAgentItems: PieItem[] = rankedAgents
    .slice(0, 6)
    .map(([id, n]) => ({ key: id, label: agentName.get(id) ?? id, value: n }));
  const tailRuns = rankedAgents.slice(6).reduce((s, [, n]) => s + n, 0);
  if (tailRuns > 0) runsByAgentItems.push({ key: 'other', label: 'Other agents', value: tailRuns });

  // Run outcomes — reliability at a glance.
  const okRuns = runs.filter((r) => r.ok).length;
  const outcomeItems: PieItem[] = [
    { key: 'ok', label: 'Succeeded', value: okRuns },
    { key: 'fail', label: 'Failed', value: runs.length - okRuns },
  ];

  // Two trees, one fetch (glados-ui SPEC X5): the GladOS style renders AnalyticsG.
  if (treeFor(getAppearance(), '/analytics') === 'glados') {
    const aud7 = channelsNetChange(audience.channels.map((c) => c.points), 7, today);
    const runCount = db.agentRuns.count();
    const tiles: AnalyticsTileG[] = live.map((t) => {
      const base = { id: t.id, label: t.label, source: t.source, spark: sparks[t.id] };
      if (t.id === 'audience') {
        return { ...base, value: formatFull(t.value), unit: '', change: aud7 ? { cur: aud7.current, prev: aud7.baseline, window: '7 days' } : null, href: '/social' };
      }
      if (t.id === 'agent-runs') {
        return { ...base, value: formatFull(runCount), unit: 'runs', note: `${formatFull(runs7d)} in the last 7 days`, href: '/agents' };
      }
      const v = tileValue(t.value, t.unit);
      return { ...base, value: v.main, unit: v.small, note: t.source };
    });
    const week = (key: string) => {
      const pts = audience.channels.find((c) => c.key === key)?.points ?? [];
      const c = pts.length ? netChange(pts, seriesWindow(7, pts, today)) : null;
      return { cur: c?.current ?? null, prev: c?.baseline ?? null };
    };
    const slot = (k: string) => `series-${SERIES_INDEX[k as keyof typeof SERIES_INDEX]}` as const;
    return (
      <AnalyticsG
        liveCount={live.length}
        pendingCount={pending.length}
        tiles={tiles}
        audience={{
          total: totalAudience,
          slices: audienceItems.map((i) => ({ id: i.key, label: i.label, value: i.value ?? 0, color: slot(i.key) })),
        }}
        runs={{
          scope: runCount > runs.length ? `latest ${formatFull(runs.length)}` : 'all time',
          total: runs.length,
          byAgent: runsByAgentItems.map((i) => ({ id: i.key, label: i.label, value: i.value ?? 0 })),
          ok: okRuns,
          recentFailures: runs.filter((r) => !r.ok).slice(0, 3).map((r) => ({ id: r.id, agent: agentName.get(r.agentId) ?? r.agentId, at: r.startedAt })),
          volume: runVolume,
          windowRuns,
          lastRunAt,
        }}
        pending={pending.map((m) => ({ id: m.id, label: m.label, source: m.source, setupHref: setupHrefFor(PENDING_CONNECTOR[m.id] ?? '') }))}
        platforms={dash.platforms.map((pl) => ({
          platform: pl.platform,
          label: PLATFORM_LABELS[pl.platform],
          href: `/social/${pl.platform}`,
          followers: pl.followers,
          ...week(pl.platform),
          spark: platformSpark.get(pl.platform) ?? [],
        }))}
      />
    );
  }

  return (
    <div>
      <PageHeader
        eyebrow="Operating metrics"
        title="Analytics"
        right={<Badge tone="accent">{live.length} live · {pending.length} pending</Badge>}
      />

      {/* Live metric tiles */}
      {live.length > 0 && (
        <section className="mb-8 grid grid-cols-1 gap-5 sm:grid-cols-2 xl:grid-cols-4 ultra:grid-cols-6">
          {live.map((tile) => (
            <MetricCard key={tile.id} tile={tile} spark={sparks[tile.id]} />
          ))}
        </section>
      )}

      {/* Distribution — share donuts across audience, agents, run outcomes */}
      <section className="mb-8">
        <SectionHead label="Distribution" count="share of totals" />
        <div className="grid grid-cols-1 gap-5 lg:grid-cols-3">
          {audienceReach > 0 && (
            <PieCard
              title="Audience share"
              sub={`${formatFollowers(audienceReach)} total audience`}
              items={audienceItems}
              total={audienceReach}
              centerLabel="total audience"
              format={formatFollowers}
              ariaLabel="Audience share by channel"
            />
          )}
          {runs.length > 0 && (
            <PieCard
              title="Agent runs · by agent"
              sub={`${fmtCount(runs.length)} runs`}
              items={runsByAgentItems}
              total={runs.length}
              centerLabel="agent runs"
              format={fmtCount}
              ariaLabel="Agent runs by agent"
            />
          )}
          {runs.length > 0 && (
            <PieCard
              title="Run outcomes"
              sub={`${Math.round((okRuns / runs.length) * 100)}% ok`}
              items={outcomeItems}
              total={runs.length}
              centerLabel="run outcomes"
              format={fmtCount}
              ariaLabel="Agent run outcomes"
            />
          )}
        </div>
      </section>

      {/* Agent run volume (real log) + awaiting-credentials sidebar */}
      <section className="mb-8 grid grid-cols-1 gap-5 xl:grid-cols-3">
        <div className="rounded-lg-t border border-os-border bg-os-surface p-6 xl:col-span-2">
          <div className="flex items-center justify-between gap-3">
            <Label>Agent run volume · 14d</Label>
            <span className="font-mono text-[12.5px] text-os-muted">{windowRuns} runs</span>
          </div>
          {windowRuns === 0 ? (
            // A flat zero line explains nothing (H23): say why it is empty.
            <div className="mt-5 flex h-[170px] flex-col items-center justify-center gap-1.5 text-center" data-run-volume-empty="">
              <span className="text-[13px] text-os-muted">
                {runs.length === 0 ? 'No agent runs recorded yet' : 'No runs in the last 14 days'}
              </span>
              {lastRunAt && <span className="text-[12px] text-os-dim">Last run {fmtShort(lastRunAt.slice(0, 10))}</span>}
            </div>
          ) : (
            <>
              <div className="mt-5">
                <RunVolumeChart data={runVolume} />
              </div>
              <div className="mt-3 flex justify-between font-mono text-[11px] text-os-dim">
                <span>{fmtShort(runVolume[0].date)}</span>
                <span>{fmtShort(runVolume[Math.floor(runVolume.length / 2)].date)}</span>
                <span>{fmtShort(runVolume[runVolume.length - 1].date)}</span>
              </div>
            </>
          )}
        </div>

        <div className="flex flex-col rounded-lg-t border border-os-border bg-os-surface p-6">
          <Label>Awaiting credentials</Label>
          <div className="mt-4 flex flex-1 flex-col gap-3">
            {pending.length === 0 ? (
              <div className="flex flex-1 items-center justify-center gap-2.5 text-[12.5px] text-os-muted">
                <span className="dot ok" /> All connectors live
              </div>
            ) : (
              pending.map((m) => (
                <div
                  key={m.id}
                  className="glass-well flex items-center gap-4 px-3.5 py-3"
                >
                  <span className="dot off" />
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-[13.5px] font-medium text-os-muted">{m.label}</div>
                    <div className="mt-0.5 text-[11.5px] text-os-dim">{m.source}</div>
                  </div>
                  <SetupLink href={setupHrefFor(PENDING_CONNECTOR[m.id] ?? '')} className="shrink-0" />
                </div>
              ))
            )}
          </div>
          <Link
            href="/integrations"
            className="mt-4 flex items-center justify-center gap-2.5 rounded-sm-t border border-os-border-strong bg-os-surface2 py-3.5 text-[13px] text-os-text transition-colors hover:bg-os-surface3"
          >
            Wire connectors → flip to live
          </Link>
        </div>
      </section>

      {/* Audience by platform — real Zernio snapshot data */}
      <section>
        <SectionHead label="Audience · by platform" count={`${formatFollowers(totalAudience)} total audience`} link="Open Social" href="/social" />
        <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 xl:grid-cols-3 ultra:grid-cols-4">
          {dash.platforms.map((p) => {
            const Icon = PLATFORM_ICONS[p.platform];
            const share = totalAudience > 0 && p.followers != null ? (p.followers / totalAudience) * 100 : 0;
            const spark = platformSpark.get(p.platform) ?? [];
            return (
              <Link
                key={p.platform}
                href={`/social/${p.platform}`}
                className="hoverable group rounded-lg-t border border-os-border bg-os-surface p-6"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="flex items-center gap-3.5">
                    <div className="flex h-8 w-8 items-center justify-center rounded-sm-t bg-os-surface2 transition-colors group-hover:bg-os-accent group-hover:[&>svg]:text-os-ink">
                      <Icon className="h-4 w-4 text-os-text" />
                    </div>
                    <div>
                      <div className="text-sm font-medium">{PLATFORM_LABELS[p.platform]}</div>
                      <div className="font-mono text-[11.5px] text-os-dim">{p.handle}</div>
                    </div>
                  </div>
                  <GrowthBadge label="7d" value={p.growth.d7} />
                </div>
                <div className="mt-5 flex items-end justify-between gap-4">
                  <div className="text-[26px] font-normal tracking-[-0.01em] tabular-nums">
                    {formatFollowers(p.followers)}
                  </div>
                  {spark.length > 1 && <Spark data={spark} w={96} h={26} />}
                </div>
                <div className="mt-4 h-1 overflow-hidden rounded-sm-t bg-os-surface2">
                  <div className="h-full bg-os-accent opacity-60" style={{ width: `${share}%` }} />
                </div>
                <div className="mt-1.5 text-[11.5px] text-os-dim">{share.toFixed(0)}% of total audience</div>
              </Link>
            );
          })}
        </div>
      </section>
    </div>
  );
}
