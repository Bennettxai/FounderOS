import Link from 'next/link';
import { notFound } from 'next/navigation';
import { ArrowLeft, ExternalLink } from 'lucide-react';
import { getDb } from '@/lib/data';
import { PLATFORM_LABELS, platformDetail, syncFromZernioConfig } from '@/lib/social';
import { platformVolume } from '@/lib/social-volume';
import type { SocialPlatform } from '@/lib/schemas';
import { formatFollowers, formatPct, GrowthBadge } from '@/components/SocialStats';
import { FollowerBarChart } from '@/components/FollowerBarChart';
import { Slab, SlabTitle, SlabCard, BigStat, MeterStack, InsightCard, PILL, PILL_ACCENT } from '@/components/slab';
import { StepLine } from '@/components/slab-charts';

export const dynamic = 'force-dynamic';

const WINDOW_DAYS = 30;

/**
 * One platform, in the Brand Deals slab (2026-09-24): what you land on from
 * a /social account tile. Every number in the hero and the second row comes
 * from lib/social-volume's platformVolume, fed with this platform's own
 * snapshots and growth windows.
 */
export default function SocialPlatformPage({ params }: { params: { platform: string } }) {
  const db = getDb();
  syncFromZernioConfig(db);
  const detail = platformDetail(db, params.platform as SocialPlatform);
  if (!detail) notFound();

  const { account, followers, growth, snapshots } = detail;
  const label = PLATFORM_LABELS[account.platform];
  const v = platformVolume({ label, followers, growth, snapshots, today: new Date().toISOString().slice(0, 10), days: WINDOW_DAYS });
  const newestFirst = [...snapshots].reverse();

  return (
    <Slab>
      <SlabTitle
        eyebrow={`audience · ${account.handle}`}
        title={label}
        meta={`${formatFollowers(followers)} followers · ${formatPct(growth.d7)} 7d · ${snapshots.length} snapshots`}
        right={
          <>
            <Link href="/social" className={PILL}>
              <ArrowLeft className="h-3.5 w-3.5" /> All platforms
            </Link>
            {account.url && (
              <a href={account.url} target="_blank" rel="noreferrer" data-lens="c" className={PILL_ACCENT}>
                Open profile <ExternalLink className="h-3.5 w-3.5" />
              </a>
            )}
          </>
        }
      />

      {/* Hero row, Brand Deals' shape: the follower history + the volume card */}
      <div className="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
        <SlabCard
          i={1}
          title="Follower History"
          sub={`${snapshots.length} snapshots`}
          action={
            <>
              <GrowthBadge label="7d" value={growth.d7} />
              <GrowthBadge label="30d" value={growth.d30} />
              <GrowthBadge label="60d" value={growth.d60} />
              <GrowthBadge label="all" value={growth.allTime} />
            </>
          }
        >
          <div className="px-6 pb-5 pt-2">
            {/* the diagram: one bar per snapshot  -  hover for exact count + change */}
            <FollowerBarChart series={snapshots.map((s) => ({ date: s.capturedAt, followers: s.followers }))} />
            {newestFirst.length > 0 && (
              <div className="mt-2 flex flex-wrap items-center gap-3 font-mono text-[10.5px] text-os-dim">
                <span className="flex items-center gap-1.5">
                  <span className="inline-block h-[3px] w-3 rounded-full bg-os-ok" /> gained vs prev
                </span>
                <span className="flex items-center gap-1.5">
                  <span className="inline-block h-[3px] w-3 rounded-full bg-os-err" /> dipped
                </span>
                <span className="ml-auto">
                  latest {newestFirst[0].capturedAt} · {newestFirst[0].source}
                </span>
              </div>
            )}
          </div>
        </SlabCard>

        <SlabCard i={2} title="Follower Volume" className="flex flex-col">
          <div className="flex flex-1 flex-col px-6 pb-6 pt-3">
            <BigStat
              value={v.headline ?? undefined}
              display={v.headline == null ? formatFollowers(null) : undefined}
              kind="followers"
              chips={v.chips}
              caption={v.caption}
            />
            <MeterStack meters={v.meters} foot={v.foot} empty="no snapshots recorded for this platform yet" />
          </div>
        </SlabCard>
      </div>

      {/* Second row: the growth windows, daily gains, THE gradient card */}
      <div className="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
        <SlabCard i={3} title="Growth Windows" sub="vs the snapshot that far back">
          <div className="grid grid-cols-2 gap-x-6 gap-y-5 px-6 pb-6 pt-4">
            {(
              [
                ['7 days', growth.d7],
                ['30 days', growth.d30],
                ['60 days', growth.d60],
                ['all time', growth.allTime],
              ] as const
            ).map(([windowLabel, value]) => (
              <BigStat
                key={windowLabel}
                size={30}
                display={formatPct(value)}
                chips={value == null ? [] : value === 0 ? [{ text: 'flat' }] : [{ tone: value > 0 ? 'ok' : 'err', text: value > 0 ? 'up' : 'down' }]}
                caption={value == null ? `${windowLabel} · not enough history` : windowLabel}
              />
            ))}
          </div>
        </SlabCard>

        <SlabCard i={4} title="Daily Gains" sub={`last ${WINDOW_DAYS} days`}>
          <div className="px-6 pt-3">
            <BigStat
              size={30}
              value={v.gainedDays}
              caption={v.intervals > 0 ? `days with a gain, of ${v.intervals} tracked` : 'no day-over-day history yet'}
            />
          </div>
          <StepLine series={v.series} hue="var(--ok)" unit=" gained" empty={`No follower gains in the last ${WINDOW_DAYS} days.`} />
        </SlabCard>

        <InsightCard
          i={5}
          badge={`${WINDOW_DAYS}-day change`}
          value={v.insight.value}
          display={v.insight.display}
          headline={v.insight.headline}
          body={v.insight.body}
          frac={v.insight.frac}
        />
      </div>
    </Slab>
  );
}
