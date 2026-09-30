import Link from 'next/link';
import { ArrowLeft, Megaphone } from 'lucide-react';
import { getDb } from '@/lib/data';
import { filterLeadMagnets, leadMagnetFilter, leadMagnetVolume, LEAD_MAGNET_FILTERS } from '@/lib/lead-magnet-volume';
import { LeadMagnets } from '@/components/LeadMagnets';
import { NewLeadMagnet } from '@/components/NewLeadMagnet';
import { Slab, SlabTitle, SlabCard, BigStat, Chip, MeterStack, InsightCard, PILL, chipClass } from '@/components/slab';
import { StepLine, DotMatrix } from '@/components/slab-charts';

export const dynamic = 'force-dynamic';

const WINDOW_WEEKS = 12;

/**
 * Lead magnets, full page, in the Brand Deals slab (2026-09-24). Every landing
 * page we ship, as a Notion-style database with the real link on every row
 * so Alex can open or copy one straight to whoever asked for it. Every
 * number above the table comes from lib/lead-magnet-volume over ALL rows; the
 * status pills (?status=live) only narrow the table.
 */
// Props stay required in the signature (Next's PageProps check rejects an
// optional parameter); the smoke test may still call it bare, hence `props?.`.
export default function LeadMagnetsPage(props: { searchParams?: Record<string, string | string[] | undefined> }) {
  const searchParams = props?.searchParams;
  const all = getDb().leadMagnets.all();
  const active = leadMagnetFilter(searchParams?.status);
  const rows = filterLeadMagnets(all, active);
  const v = leadMagnetVolume({ rows: all, today: new Date().toISOString().slice(0, 10), weeks: WINDOW_WEEKS });
  const topDest = v.destinations[0];
  const topCapture = v.captures.reduce((best, c) => (c.count > best.count ? c : best), v.captures[0]);
  const countFor = (f: (typeof LEAD_MAGNET_FILTERS)[number]) => (f === 'all' ? v.total : v.counts[f]);

  return (
    <Slab>
      <SlabTitle
        eyebrow="content engine"
        title="Lead Magnets"
        meta={`${v.total} pages · ${v.counts.live} live · ${v.foot}`}
        right={
          <>
            <Chip tone="ok">{v.counts.live} live</Chip>
            <Link href="/content" className={PILL}>
              <ArrowLeft className="h-3.5 w-3.5" /> Content
            </Link>
          </>
        }
      />

      {/* Hero row, Brand Deals' shape: pages shipped + the volume card */}
      <div className="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
        <SlabCard i={1} title="Pages Shipped" sub={`last ${WINDOW_WEEKS} weeks`} className="pb-2">
          <div className="px-6 pb-2 pt-3">
            <BigStat
              size={30}
              value={v.total}
              chips={v.shippedInWindow > 0 ? [{ tone: 'accent', text: `${v.shippedInWindow} new in ${WINDOW_WEEKS} weeks` }] : []}
              caption="landing pages on record, week by week"
            />
          </div>
          <StepLine series={v.series} hue="var(--send-activity)" unit=" pages" empty="No landing pages on record yet." />
        </SlabCard>

        <SlabCard i={2} title="Magnet Volume" className="flex flex-col">
          <div className="flex flex-1 flex-col px-6 pb-6 pt-3">
            <BigStat value={v.headline} unit="live" chips={v.chips} caption={v.caption} />
            <MeterStack meters={v.meters} foot={v.foot} empty="no landing pages to measure yet" />
          </div>
        </SlabCard>
      </div>

      {/* Second row: what they capture, where the leads land, THE gradient card */}
      <div className="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
        <SlabCard i={3} title="Captures" sub="what each page asks for">
          <div className="flex items-end justify-between gap-4 px-6 pb-6 pt-3">
            <BigStat
              size={30}
              display={topCapture.count > 0 ? topCapture.label : 'none'}
              caption={topCapture.count > 0 ? `most pages · ${topCapture.count} of ${v.total}` : 'no pages yet'}
            />
            <DotMatrix cols={v.captures} hue="var(--ramp-1)" />
          </div>
        </SlabCard>

        <SlabCard i={4} title="Where Leads Land" sub={`${v.destinations.length} destinations`}>
          {/* stacked: destination names are too wide to share a row with the caption */}
          <div className="flex flex-col gap-5 px-6 pb-6 pt-3">
            <BigStat
              size={30}
              display={topDest ? topDest.label : 'none'}
              caption={topDest ? `top destination · ${topDest.count} page${topDest.count === 1 ? '' : 's'}` : 'no destinations yet'}
            />
            {v.destinations.length > 0 && <DotMatrix cols={v.destinations} hue="var(--ramp-3)" />}
          </div>
        </SlabCard>

        <InsightCard
          i={5}
          badge="Since last launch"
          icon={<Megaphone size={13} strokeWidth={1.7} />}
          value={v.insight.value}
          display={v.insight.display}
          headline={v.insight.headline}
          body={v.insight.body}
          frac={v.insight.frac}
        />
      </div>

      {/* The register: filter pills, the new-page form, the database table */}
      <SlabCard
        i={6}
        title="All pages"
        sub={`${rows.length} of ${v.total}`}
        className="mt-6"
        action={LEAD_MAGNET_FILTERS.map((f) => (
          <Link
            key={f}
            href={f === 'all' ? '/content/lead-magnets' : `/content/lead-magnets?status=${f}`}
            className={chipClass(active === f)}
            aria-current={active === f ? 'page' : undefined}
          >
            {f.charAt(0).toUpperCase() + f.slice(1)} {countFor(f)}
          </Link>
        ))}
      >
        <p className="px-6 pt-2 text-[12.5px] leading-relaxed text-os-dim">
          Every landing page we ship, with the live link on each row. Open it, or copy it straight to whoever asked.
        </p>
        <div className="px-6 pt-4">
          <NewLeadMagnet />
        </div>
        <div className="mt-2">
          {rows.length === 0 && v.total > 0 ? (
            <div className="border-t border-os-border px-6 py-8 text-center text-[12.5px] text-os-dim">Nothing matches that filter.</div>
          ) : (
            <LeadMagnets rows={rows} showCopy manage />
          )}
        </div>
      </SlabCard>
    </Slab>
  );
}
