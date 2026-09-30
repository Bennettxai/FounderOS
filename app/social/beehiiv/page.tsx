import Link from 'next/link';
import { ArrowLeft, ExternalLink, Mail } from 'lucide-react';
import { getNewsletters } from '@/lib/newsletters';
import { newsletterVolume } from '@/lib/newsletter-volume';
import { beehiivSubscribers } from '@/lib/connectors/beehiiv';
import { NewsletterList } from '@/components/NewsletterList';
import { Slab, SlabTitle, SlabCard, BigStat, Chip, MeterStack, InsightCard, PILL, PILL_ACCENT } from '@/components/slab';
import { StepLine, DotMatrix } from '@/components/slab-charts';

export const dynamic = 'force-dynamic';

/**
 * The Beehiiv newsletter, in the Brand Deals slab (2026-09-24): what you land
 * on from the /social email tile. Every number in the hero and the second
 * row comes from lib/newsletter-volume, fed with the page's own rows (the
 * live subscriber count and the past issues, seeded until a key resolves).
 */
export default async function BeehiivDashboardPage() {
  const [newsletters, subscribers] = await Promise.all([getNewsletters(), beehiivSubscribers()]);
  const live = subscribers != null; // a real key resolved a subscriber count
  const v = newsletterVolume({ newsletters, subscribers });
  const issues = newsletters.length;

  return (
    <Slab>
      <SlabTitle
        eyebrow="beehiiv · email list"
        title="Newsletter"
        meta={live ? 'live via Beehiiv API' : 'seeded preview · add BEEHIIV_API_KEY for live'}
        right={
          <>
            <Chip tone={live ? 'ok' : 'warn'}>{live ? 'beehiiv live' : 'seeded'}</Chip>
            <Link href="/social" className={PILL}>
              <ArrowLeft className="h-3.5 w-3.5" /> All platforms
            </Link>
            <a href="https://app.beehiiv.com" target="_blank" rel="noreferrer" data-lens="c" className={PILL_ACCENT}>
              Open Beehiiv <ExternalLink className="h-3.5 w-3.5" />
            </a>
          </>
        }
      />

      {/* Hero row, Brand Deals' shape: the sends line + the volume card */}
      <div className="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
        <SlabCard i={1} title="Sends" sub={`${issues} issue${issues === 1 ? '' : 's'}`} className="pb-2">
          <div className="px-6 pb-2 pt-3">
            <BigStat size={30} value={v.totalRecipients} kind="followers" caption="recipients across every issue, oldest to newest" />
          </div>
          <StepLine series={v.sends} hue="var(--send-activity)" unit=" sent" empty="No issues sent yet." />
        </SlabCard>

        <SlabCard i={2} title="Newsletter Volume" className="flex flex-col">
          <div className="flex flex-1 flex-col px-6 pb-6 pt-3">
            <BigStat
              value={v.headline ?? undefined}
              display={v.headline == null ? ' - ' : undefined}
              kind="followers"
              chips={v.chips}
              caption={v.caption}
            />
            <MeterStack meters={v.meters} foot={v.foot} empty="no sends to measure yet" />
          </div>
        </SlabCard>
      </div>

      {/* Second row: open rate per issue, engagement, THE gradient card */}
      <div className="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
        <SlabCard i={3} title="Open Rate" sub="last 6 issues">
          {/* stacked: dated columns are too wide to share a row with the caption */}
          <div className="flex flex-col gap-5 px-6 pb-6 pt-3">
            <BigStat
              size={30}
              display={v.avgOpenRate == null ? ' - ' : `${v.avgOpenRate.toFixed(1)}%`}
              caption={issues > 0 ? `average open rate across ${issues} issue${issues === 1 ? '' : 's'}` : 'no issues sent yet'}
            />
            {v.openRates.length > 0 && <DotMatrix cols={v.openRates} hue="var(--ramp-1)" />}
          </div>
        </SlabCard>

        <SlabCard i={4} title="Engagement" sub="summed across issues">
          <div className="px-6 pb-6 pt-3">
            <BigStat size={30} value={v.totalClicks} unit="clicks" caption={issues > 0 ? 'link clicks across every issue' : 'no clicks recorded yet'} />
            <div className="mt-4 flex flex-wrap gap-2">
              <Chip tone={v.unsubscribes > 0 ? 'warn' : 'ok'}>{v.unsubscribes.toLocaleString('en-US')} unsubscribed</Chip>
              <Chip tone={v.spamReports > 0 ? 'err' : 'ok'}>{v.spamReports.toLocaleString('en-US')} spam reports</Chip>
            </div>
          </div>
        </SlabCard>

        <InsightCard
          i={5}
          badge="Best open"
          icon={<Mail size={13} strokeWidth={1.7} />}
          display={v.insight.display}
          headline={v.insight.headline}
          body={v.insight.body}
          frac={v.insight.frac}
        />
      </div>

      <SlabCard i={6} title="Past newsletters" sub="click any issue to expand its analytics" className="mt-6">
        <div className="px-6 pb-6 pt-4">
          <NewsletterList newsletters={newsletters} />
        </div>
      </SlabCard>
    </Slab>
  );
}
