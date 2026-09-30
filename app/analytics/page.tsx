import Link from 'next/link';
import { Instagram, Linkedin, Music2, Youtube } from 'lucide-react';
import { XLogo } from '@/components/XLogo';

/** Any icon that takes a className: the lucide set and the hand-rolled X mark
    both satisfy it, and the map does not care which it is holding. */
type PlatformIcon = React.ComponentType<{ className?: string }>;
import { getDb } from '@/lib/data';
import { buildSocialDashboard, syncFromZernioConfig, audienceGrowthPct, PLATFORM_LABELS } from '@/lib/social';
import { agentRunVolume, runsWithin } from '@/lib/analytics';
import { analyticsVolume } from '@/lib/analytics-volume';
import { splitMetrics, sparkSeries, type MetricTile } from '@/lib/operating-metrics';
import { gatherOperatingMetrics } from '@/lib/analytics-refresh';
import type { SocialPlatform } from '@/lib/schemas';
import { Spark } from '@/components/terminal';
import { formatFollowers, formatPct, MiniBars } from '@/components/SocialStats';
import { RunVolumeCard } from '@/components/RunVolumeCard';
import { Slab, SlabTitle, SlabCard, BigStat, Chip, MeterStack, InsightCard, PILL, PILL_ACCENT } from '@/components/slab';
import { DotMatrix } from '@/components/slab-charts';

export const dynamic = 'force-dynamic';

const PLATFORM_ICONS: Record<SocialPlatform, PlatformIcon> = {
  instagram: Instagram,
  tiktok: Music2,
  twitter: XLogo,
  youtube: Youtube,
  linkedin: Linkedin,
};

// Value + small-unit split per tile (compact for audience, $ for money).
function tileValue(value: number, unit: string): { main: string; small: string } {
  if (unit === 'usd') return { main: `$${value.toLocaleString('en-US')}`, small: '' };
  if (unit === 'followers') return { main: formatFollowers(value), small: '' };
  return { main: value.toLocaleString('en-US'), small: unit };
}

// Deterministic rising bars per channel  -  fallback until a platform has
// enough real snapshot history (two points) to draw the truth.
function barsFor(seed: string): number[] {
  const base = [...seed].reduce((s, c) => s + c.charCodeAt(0), 0);
  return Array.from({ length: 12 }, (_, i) => 4 + i * 1.3 + ((base + i * 7) % 5));
}

/** One live operating metric, as a Brand Deals inset tile: 30px numeral,
    a dot chip for its movement, the real sparkline and its source. */
function MetricTileCard({ tile, spark }: { tile: MetricTile; spark: number[] }) {
  const up = tile.delta > 0;
  const flat = tile.delta === 0;
  const { main, small } = tileValue(tile.value, tile.unit);
  return (
    <div data-lens="r" className="pressable is-row flex min-w-0 flex-col gap-3 rounded-[12px] border border-os-border px-5 py-4">
      <div className="flex items-center justify-between gap-2">
        <span className="truncate text-[13px] text-os-muted">{tile.label}</span>
        {!flat && (
          <Chip tone={up ? 'ok' : 'err'}>
            {up ? '+' : ''}
            {tile.delta}
            {tile.deltaPct ? '%' : ''}
          </Chip>
        )}
      </div>
      <div className="flex items-baseline gap-2 text-[30px] font-semibold leading-none tracking-[-0.03em] tabular-nums">
        {main}
        {small && <small className="text-[13px] font-normal tracking-normal text-os-dim">{small}</small>}
      </div>
      <div className="flex items-end justify-between gap-2">
        <Spark data={spark} w={96} h={26} />
        <span className="font-mono text-[10.5px] text-os-dim">{tile.source}</span>
      </div>
    </div>
  );
}

export default async function AnalyticsPage() {
  const db = getDb();
  syncFromZernioConfig(db);
  const today = new Date().toISOString().slice(0, 10);

  // Real agent-run activity: the run-volume hero, runs by agent, the rhythm.
  // The 30-day window every run card here charts, bounded in SQL (+1 spare day).
  const runs = db.agentRuns.since(new Date(Date.now() - 31 * 86_400_000).toISOString());
  const runVolume = agentRunVolume(runs, today, 30);

  // Real audience: Zernio snapshot totals + true 7d growth.
  const dash = buildSocialDashboard(db);
  const totalFollowers = dash.totalFollowers;
  const audience7d = audienceGrowthPct(db, 7);

  // Every tile is a real connector read, or honest pending (value === null),
  // the same sweep the /api/analytics/refresh cron snapshots every 15 min.
  const { inputs, subs } = await gatherOperatingMetrics(db);
  const { live, pending } = splitMetrics(inputs);

  // Real sparklines from snapshot history (per-day last value); a tile with
  // fewer than two captured days keeps the deterministic placeholder shape.
  const sparkOf = (id: string, value: number) =>
    sparkSeries(
      db.metricSnapshots.history(id, 7, today).map((h) => h.value),
      id,
      value,
    );

  // The page's own numbers in the Deal Volume shape (lib/analytics-volume).
  const vol = analyticsVolume({
    channels: dash.platforms.map((p) => ({ key: p.platform, label: PLATFORM_LABELS[p.platform], followers: p.followers })),
    subs,
    growth7d: audience7d,
    live: live.length,
    pending: pending.map((m) => ({ label: m.label, source: m.source })),
    runs,
    agentNames: Object.fromEntries(db.agents.all().map((a) => [a.id, a.name])),
    today,
  });
  const busiest = vol.rhythm.reduce((b, c) => (c.count > b.count ? c : b), vol.rhythm[0]);

  // By-platform bars: real follower snapshots (last 12 captures) once a
  // platform has two, else the deterministic placeholder.
  const realBarsByPlatform = new Map<SocialPlatform, number[]>();
  for (const p of dash.platforms) {
    const snaps = db.social.snapshots(p.platform).map((s) => s.followers).slice(-12);
    if (snaps.length >= 2) realBarsByPlatform.set(p.platform, snaps);
  }

  return (
    <Slab>
      <SlabTitle
        eyebrow="operating metrics · connectors · agent runs"
        title="Analytics"
        meta={`${vol.headline} reach · ${live.length} live · ${pending.length} pending · ${runsWithin(runs, today, 30).toLocaleString('en-US')} runs in 30 days`}
        right={
          <>
            <span className="rounded-full border border-os-border px-4 py-2 text-[13px] text-os-muted">
              {live.length} live · {pending.length} pending
            </span>
            <Link href="/social" className={PILL}>
              Open Social
            </Link>
            <Link href="/integrations" className={PILL_ACCENT}>
              Wire connectors
            </Link>
          </>
        }
      />

      {/* Hero row: the run log as the big chart, reach as the Volume card. */}
      <div className="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
        <RunVolumeCard data={runVolume} i={1} />
        <SlabCard title="Reach Volume" i={2} className="flex flex-col">
          <div className="flex flex-1 flex-col px-6 pb-6 pt-3">
            <BigStat value={vol.reach} chips={vol.chips} caption={vol.caption} />
            <MeterStack meters={vol.meters} foot={vol.foot} empty="no audience snapshots yet" />
          </div>
        </SlabCard>
      </div>

      {/* Second row: runs by agent, the weekday rhythm, THE insight card. */}
      <div className="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
        <SlabCard title="Runs by Agent" sub={`${vol.runs.total.toLocaleString('en-US')} runs`} i={3} className="flex flex-col">
          <div className="flex flex-1 flex-col px-6 pb-6 pt-3">
            <BigStat
              size={30}
              display={vol.runs.okPct === null ? 'none yet' : `${vol.runs.okPct}%`}
              chips={vol.runs.total ? [{ tone: 'ok', text: `${vol.runs.ok} ok` }, ...(vol.runs.failed ? [{ tone: 'err' as const, text: `${vol.runs.failed} failed` }] : [])] : []}
              caption="of runs succeeded in the last 30 days"
            />
            <MeterStack meters={vol.runs.meters} foot={`${vol.runs.agents} agents in the log`} empty="no agent runs recorded yet" />
          </div>
        </SlabCard>

        <SlabCard title="Run Rhythm" sub="last 30 days" i={4}>
          <div className="flex items-end justify-between gap-4 px-6 pb-6 pt-3">
            <div className="shrink-0">
              <BigStat size={30} value={vol.rhythmTotal} />
              <div className="mt-1 whitespace-nowrap text-[13px] text-os-dim">runs by weekday</div>
              {vol.rhythmTotal > 0 && (
                <div className="mt-4 w-fit whitespace-nowrap rounded-full border border-os-border px-3 py-1 text-[12px] text-os-muted">
                  Busiest: <span className="font-semibold tabular-nums">{busiest.label}</span>
                </div>
              )}
            </div>
            <DotMatrix cols={vol.rhythm} hue="var(--send-activity)" />
          </div>
        </SlabCard>

        <InsightCard
          i={5}
          badge="Awaiting credentials"
          value={vol.insight.value}
          headline={vol.insight.headline}
          body={vol.insight.body}
          frac={vol.insight.frac}
        />
      </div>

      {/* Live metric tiles + every metric still waiting on a key. */}
      <SlabCard
        title="Operating Metrics"
        sub={`${live.length} live · ${pending.length} pending`}
        i={6}
        className="mt-6"
        action={
          <Link href="/integrations" className={PILL}>
            wire connectors → flip to live
          </Link>
        }
      >
        <div className="px-6 pb-6 pt-4">
          {live.length > 0 ? (
            <div className="grid gap-3.5 sm:grid-cols-2 xl:grid-cols-4 ultra:grid-cols-6">
              {live.map((tile) => (
                <MetricTileCard key={tile.id} tile={tile} spark={sparkOf(tile.id, tile.value)} />
              ))}
            </div>
          ) : (
            <p className="py-4 text-[12.5px] text-os-dim">No connector has handed back a live number yet.</p>
          )}
          <div className="mt-5 flex flex-wrap items-center gap-2 border-t border-os-border pt-4">
            <span className="mr-1 text-[13px] text-os-dim">Awaiting credentials</span>
            {pending.length === 0 ? (
              <Chip tone="ok">all connectors live</Chip>
            ) : (
              pending.map((m) => (
                <span key={m.id} title={m.source}>
                  <Chip tone="warn">
                    {m.label} · {m.source}
                  </Chip>
                </span>
              ))
            )}
          </div>
        </div>
      </SlabCard>


      {/* Audience by platform: real Zernio snapshot data, one roomy row each. */}
      <SlabCard
        title="Audience"
        sub={`${formatFollowers(totalFollowers)} followers`}
        i={9}
        className="mt-6"
        action={
          <Link href="/social" className={PILL}>
            Open Social
          </Link>
        }
      >
        <div className="mt-4 border-t border-os-border">
          {dash.platforms.map((p) => {
            const Icon = PLATFORM_ICONS[p.platform];
            const share = totalFollowers > 0 && p.followers != null ? (p.followers / totalFollowers) * 100 : 0;
            const d7 = p.growth.d7;
            return (
              <Link
                key={p.platform}
                href={`/social/${p.platform}`}
                data-lens="r"
                className="pressable is-row group grid w-full grid-cols-[minmax(180px,1fr)_auto_auto_auto] items-center gap-5 border-b border-os-border px-6 py-3.5 last:border-0 hover:bg-[color-mix(in_oklab,var(--text)_4%,transparent)] max-[800px]:grid-cols-[1fr_auto]"
              >
                <span className="flex min-w-0 items-center gap-3">
                  <span className="lens-child grid h-9 w-9 shrink-0 place-items-center rounded-full border border-os-border group-hover:bg-os-accent group-hover:[&>svg]:text-os-ink">
                    <Icon className="h-4 w-4 text-os-text" />
                  </span>
                  <span className="min-w-0">
                    <span className="block truncate text-[13.5px] font-medium">{PLATFORM_LABELS[p.platform]}</span>
                    <span className="block truncate font-mono text-[11px] text-os-dim">{p.handle}</span>
                  </span>
                </span>
                <span className="max-[800px]:hidden">
                  <MiniBars bars={realBarsByPlatform.get(p.platform) ?? barsFor(p.platform)} />
                </span>
                <span className="text-right text-[20px] font-semibold tabular-nums tracking-[-0.02em]">{formatFollowers(p.followers)}</span>
                <span className="flex items-center gap-2 max-[800px]:hidden">
                  <span className="rounded-full px-2.5 py-0.5 font-mono text-[10.5px] tabular-nums tracking-[0.04em]" style={{ background: 'var(--accent-soft)', color: 'var(--accent)' }}>
                    {share.toFixed(0)}% of reach
                  </span>
                  <span
                    className="rounded-full px-2.5 py-0.5 font-mono text-[10.5px] uppercase tabular-nums tracking-[0.08em]"
                    style={
                      d7 === null
                        ? { background: 'color-mix(in oklab, var(--text) 8%, transparent)', color: 'var(--text-2)' }
                        : d7 >= 0
                          ? { background: 'color-mix(in oklab, var(--ok) 16%, transparent)', color: 'var(--ok)' }
                          : { background: 'color-mix(in oklab, var(--err) 16%, transparent)', color: 'var(--err)' }
                    }
                  >
                    7d {formatPct(d7)}
                  </span>
                </span>
              </Link>
            );
          })}
          {dash.platforms.length === 0 && <div className="px-6 py-8 text-center text-[12.5px] text-os-dim">No platforms synced yet.</div>}
        </div>
      </SlabCard>
    </Slab>
  );
}
