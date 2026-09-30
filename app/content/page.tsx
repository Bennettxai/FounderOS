import Link from 'next/link';
import { ArrowUpRight, BarChart3, Brain, Clapperboard, ExternalLink, Megaphone, Play } from 'lucide-react';
import { getDb } from '@/lib/data';
import { contentAgents } from '@/lib/content';
import { contentVolume } from '@/lib/content-volume';
import { zernioRecentPosts, zernioPostDaysKnown } from '@/lib/connectors/zernio';
import { Dot } from '@/components/terminal';
import { ContentAgentCard } from '@/components/ContentAgentCard';
import { Slab, SlabTitle, SlabCard, BigStat, Chip, MeterStack, InsightCard, PILL } from '@/components/slab';
import { StepLine, DotMatrix } from '@/components/slab-charts';

export const dynamic = 'force-dynamic';

const WINDOW_DAYS = 30;

/** Rounded status pill for a Zernio post, Brand Deals style: color means status only. */
function postPillStyle(status: string) {
  const hue = status === 'published' ? 'var(--ok)' : status === 'failed' ? 'var(--err)' : status === 'scheduled' ? 'var(--warn)' : 'var(--muted)';
  return { background: `color-mix(in oklab, ${hue} 16%, transparent)`, color: hue };
}

// The Vantage content-intelligence system this view backlinks out to.
const INTEL_URL = 'https://intel.example.com';
const INTEL_ANALYTICS_URL = 'https://intel.example.com/my-analytics';

function agoFrom(iso: string | null): string {
  if (!iso) return '';
  const ms = Date.now() - new Date(iso).getTime();
  if (!Number.isFinite(ms) || ms < 0) return '';
  const mins = Math.round(ms / 60_000);
  if (mins < 60) return `${Math.max(1, mins)}m`;
  const hrs = Math.round(mins / 60);
  if (hrs < 48) return `${hrs}h`;
  return `${Math.round(hrs / 24)}d`;
}

function platformLabel(p: string): string {
  return p.charAt(0).toUpperCase() + p.slice(1);
}

function BacklinkCard({
  href, icon: Icon, mark, title, sub, internal = false, meta,
}: {
  href: string;
  icon?: typeof Brain;
  /** real brand mark (public/*.png) instead of a lucide glyph */
  mark?: string;
  title: string;
  sub: string;
  internal?: boolean;
  meta?: string;
}) {
  const inner = (
    <>
      <span className="grid h-9 w-9 shrink-0 place-items-center rounded-ctl border border-os-border bg-os-surface2 text-os-accent">
        {mark ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={mark} alt="" className="h-5 w-5 object-contain" />
        ) : Icon ? (
          <Icon className="h-4 w-4" strokeWidth={1.8} />
        ) : null}
      </span>
      <span className="min-w-0 flex-1">
        <span className="flex items-center gap-1.5 text-[13.5px] font-semibold">
          {title}
          {internal ? (
            <ArrowUpRight className="h-3.5 w-3.5 lens-child text-os-dim group-hover:text-os-accent" />
          ) : (
            <ExternalLink className="h-3.5 w-3.5 lens-child text-os-dim group-hover:text-os-accent" />
          )}
        </span>
        <span className="mt-0.5 block text-[12px] leading-relaxed text-os-dim [text-wrap:pretty]">{sub}</span>
        <span className="mt-1.5 block truncate font-mono text-[10px] text-os-muted">
          {meta ?? href.replace('https://', '')}
        </span>
      </span>
    </>
  );
  const cls =
    'pressable is-row group flex h-full items-start gap-3 rounded-panel border border-os-border bg-os-bg p-4';
  return internal ? (
    <Link href={href} data-lens="r" className={cls}>{inner}</Link>
  ) : (
    <a href={href} data-lens="r" target="_blank" rel="noopener noreferrer" className={cls}>{inner}</a>
  );
}

export default async function ContentPage() {
  const db = getDb();
  const crew = contentAgents(db.agents.all());
  const leadMagnets = db.leadMagnets.all();
  const lead = crew[0] ?? null;
  const workers = lead ? crew.slice(1) : crew;

  const posts = await zernioRecentPosts(8).catch(() => []);
  // null = Zernio gave no answer: the history is unknown, never "nothing posted".
  const known = await zernioPostDaysKnown().catch(() => null);
  const postsKnown = known !== null;
  const days = known ?? [];
  // zernioPostDays() is one entry per POST; a day with three posts is one active day.
  const activeDays = new Set(days.filter((d) => d.platforms.length > 0).map((d) => d.date)).size;

  // 2026-09-24: the Brand Deals slab. Every number in the hero and the second
  // row comes from lib/content-volume, fed with the same rows the panels
  // below render (the crew, their real runs, the magnets, Zernio's history).
  const v = contentVolume({
    crew,
    runs: db.agentRuns.since(new Date(Date.now() - (WINDOW_DAYS + 1) * 86_400_000).toISOString(), crew.map((a) => a.id)),
    leadMagnets,
    recent: posts,
    postDays: days,
    postsKnown,
    today: new Date().toISOString().slice(0, 10),
    days: WINDOW_DAYS,
  });
  const busiest = v.crewRuns.reduce((best, c) => (c.count > best.count ? c : best), { label: 'none', count: 0 });

  return (
    <Slab>
      <SlabTitle
        eyebrow="content engine"
        title="Content Creation"
        meta={`${crew.length} agents · ${leadMagnets.length} lead magnets · ${v.activeDays} active days in ${WINDOW_DAYS}`}
        right={
          <>
            <Chip tone="accent">{crew.length} agents</Chip>
            <Link href="/content/lead-magnets" className={PILL}>
              Lead magnets
            </Link>
            <Link href="/social" className={PILL}>
              Social
            </Link>
          </>
        }
      />

      {/* Hero row, Brand Deals' shape: the posting line + the volume card */}
      <div className="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
        <SlabCard i={1} title="Posting Activity" sub={`last ${WINDOW_DAYS} days`} className="pb-2">
          <div className="px-6 pb-2 pt-3">
            <BigStat
              size={30}
              value={v.postsInWindow}
              display={postsKnown ? undefined : ' - '}
              chips={v.activeDays > 0 ? [{ tone: 'accent', text: `${v.activeDays} active days` }] : []}
              caption={postsKnown ? 'posts out through Zernio, one per cross-post' : 'Zernio not answering · posting history unknown'}
            />
          </div>
          <StepLine
            series={v.series}
            hue="var(--send-activity)"
            unit=" posts"
            empty={postsKnown ? `No posts on record in the last ${WINDOW_DAYS} days.` : 'Posting history unavailable right now.'}
          />
        </SlabCard>

        <SlabCard i={2} title="Content Volume" className="flex flex-col">
          <div className="flex flex-1 flex-col px-6 pb-6 pt-3">
            <BigStat value={v.headline} chips={v.chips} caption={v.caption} />
            <MeterStack meters={v.meters} foot={v.foot} empty="no posts, magnets or crew runs to measure yet" />
          </div>
        </SlabCard>
      </div>

      {/* Second row: crew runs, the lead magnets, THE gradient card */}
      <div className="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
        <SlabCard i={3} title="Crew Runs" sub={`last ${WINDOW_DAYS} days`}>
          {/* stacked: seven agent names are too wide to share a row with the caption */}
          <div className="flex flex-col gap-5 px-6 pb-6 pt-3">
            <BigStat
              size={30}
              value={v.runsInWindow}
              caption={busiest.count > 0 ? `busiest · ${busiest.label} with ${busiest.count}` : 'no crew runs in this window'}
            />
            <DotMatrix cols={v.crewRuns} hue="var(--ramp-1)" />
          </div>
        </SlabCard>

        <SlabCard
          i={4}
          title="Lead Magnets"
          sub={`${v.magnets.total} pages`}
          action={
            <Link href="/content/lead-magnets" className={PILL}>
              Open <ArrowUpRight className="h-3.5 w-3.5" />
            </Link>
          }
        >
          <div className="px-6 pb-6 pt-3">
            <BigStat
              size={30}
              value={v.magnets.live}
              unit="live"
              caption={v.magnets.total > 0 ? `of ${v.magnets.total} landing pages shipped` : 'no landing pages recorded yet'}
            />
            <div className="mt-4 flex flex-wrap gap-2">
              {v.magnets.draft > 0 && <Chip tone="warn">{v.magnets.draft} draft</Chip>}
              {v.magnets.paused > 0 && <Chip tone="err">{v.magnets.paused} paused</Chip>}
              {v.magnets.archived > 0 && <Chip>{v.magnets.archived} archived</Chip>}
            </div>
          </div>
        </SlabCard>

        <InsightCard
          i={5}
          badge="Since last post"
          value={v.insight.value}
          display={v.insight.display}
          headline={v.insight.headline}
          body={v.insight.body}
          frac={v.insight.frac}
        />
      </div>

      {/* Three equal sections (Alex, 2026-08-13): the lead magnet index and
          both Vantage intelligence surfaces, side by side. */}
      <SlabCard i={6} title="Content intelligence" sub="3 surfaces" className="mt-6">
        <div className="grid gap-3 px-6 pb-6 pt-4 md:grid-cols-3">
          <BacklinkCard
            href="/content/lead-magnets"
            internal
            icon={Megaphone}
            title="Lead Magnets"
            sub="Every landing page we ship, with the live link on each row."
            meta={`${leadMagnets.length} page${leadMagnets.length === 1 ? '' : 's'} · ${leadMagnets.filter((m) => m.status === 'live').length} live`}
          />
          <BacklinkCard
            href={INTEL_URL}
            mark="/vantage-mark.png"
            title="Vantage Intel"
            sub="Your content intelligence system  -  research, hooks, and what's working, feeding the content agent."
          />
          <BacklinkCard
            href={INTEL_ANALYTICS_URL}
            icon={BarChart3}
            title="My Analytics"
            sub="Per-piece performance and audience analytics from the intelligence system."
          />
        </div>
      </SlabCard>

      {/* The content agent + crew (real seed roster) */}
      <SlabCard
        i={7}
        title="Content agents"
        sub={`${crew.length}`}
        className="mt-6"
        action={
          <Link href="/agents" className={PILL}>
            Agents <ArrowUpRight className="h-3.5 w-3.5" />
          </Link>
        }
      >
        <div className="px-6 pb-6 pt-3">
          <p className="mb-4 flex items-center gap-1.5 text-[12.5px] text-os-dim">
            <Clapperboard className="h-3.5 w-3.5" /> Tied to your social media. Run one here or from Agents.
          </p>
          {lead && <ContentAgentCard agent={lead} lead />}
          {workers.length > 0 && (
            <div className="mt-3 grid gap-3 lg:grid-cols-2">
              {workers.map((a) => (
                <ContentAgentCard key={a.id} agent={a} />
              ))}
            </div>
          )}
        </div>
      </SlabCard>

      {/* Zernio content pipeline  -  recent published content + cadence */}
      <SlabCard
        i={8}
        title="Zernio content pipeline"
        sub={posts.length > 0 ? `${posts.length} recent` : 'no live pull'}
        className="mt-6"
        action={
          <Link href="/social" className={PILL}>
            Social <ArrowUpRight className="h-3.5 w-3.5" />
          </Link>
        }
      >
        <p className="px-6 pt-2 font-mono text-[11px] text-os-dim">
          Published across six platforms via Zernio · {activeDays} active days tracked
        </p>
        {posts.length > 0 ? (
          <ul className="mt-4 border-t border-os-border">
            {posts.map((p, i) => (
              <li
                key={`${p.url}-${i}`}
                data-lens="r"
                className="pressable is-row flex items-center gap-4 border-b border-os-border px-6 py-3.5 last:border-0"
              >
                <Play className="h-3.5 w-3.5 shrink-0 text-os-dim" />
                <Dot state={p.status === 'published' ? 'ok' : 'available'} />
                <span className="w-24 shrink-0 font-mono text-[11px] uppercase tracking-wide text-os-muted">{platformLabel(p.platform)}</span>
                <span className="min-w-0 flex-1 truncate text-[13.5px]">{p.caption || 'Untitled post'}</span>
                <span
                  className="hidden shrink-0 rounded-full px-2.5 py-0.5 font-mono text-[10.5px] uppercase tracking-[0.12em] sm:inline"
                  style={postPillStyle(p.status)}
                >
                  {p.status}
                </span>
                {p.url ? (
                  <a href={p.url} data-lens="c" target="_blank" rel="noopener noreferrer" className="pressable shrink-0 rounded-full px-1 text-os-dim">
                    <ExternalLink className="h-3.5 w-3.5" />
                  </a>
                ) : null}
                <span className="w-10 shrink-0 text-right font-mono text-[11px] text-os-dim">{agoFrom(p.publishedAt)}</span>
              </li>
            ))}
          </ul>
        ) : (
          <p className="mx-6 mb-6 mt-4 rounded-[12px] border border-dashed border-os-border px-4 py-5 text-center font-mono text-[11.5px] text-os-dim">
            No live Zernio pull right now  -  recent content shows here once the API responds (key from ~/.social-media/.env).
          </p>
        )}
      </SlabCard>
    </Slab>
  );
}
