import Link from 'next/link';
import { headers } from 'next/headers';
import { ChevronRight, ArrowUpRight, Calendar } from 'lucide-react';
import { getDb } from '@/lib/data';
import { allConnectorStatuses } from '@/lib/connectors';
import { createGBrainProvider } from '@/lib/connectors/gbrain';
import { audienceSeries } from '@/lib/social';
import { zernioPostDays } from '@/lib/connectors/zernio';
import { postSeriesFromDays } from '@/lib/posting-activity';
import { channelColor } from '@/lib/series';
import { PLATFORM_LABELS } from '@/lib/social';
import { SocialPlatformSchema, type SocialPlatform } from '@/lib/schemas';
import { gatherCommsFeed } from '@/lib/comms-feed';
import { inboundLast24h } from '@/lib/comms';
import { groupRoadmapByQuarter } from '@/lib/roadmap';
import { runsPerDay, stateOfWorld, statusPill } from '@/lib/pulse-history';
import { hourlyBucketStarts, inboundSummary, lastRunCaption, relAge, revenueSummary } from '@/lib/home-summary';
import { formatDelta } from '@/lib/delta';
import { netChange, seriesWindow, windowPoints } from '@/lib/social-range';
import { getAppearance } from '@/lib/appearance-server';
import { treeFor } from '@/lib/appearance';
import { HomeG, type HomeGData } from '@/components/home/HomeG';
import { InterjectClassic } from '@/components/home/Interject';
import { DONE_TODAY_EMPTY, NEEDS_YOU_EMPTY, doneCaption, doneToday, needsYou, type DoneToday, type NeedsYou } from '@/lib/home-today';
import { loadObservatoryData } from '@/lib/gbrain-observatory';
import { formatCompact, formatFull } from '@/lib/format-number';
import { setupHrefFor } from '@/lib/integrations-catalog';
import { Home6Chart, type Home6Series } from '@/components/Home6Chart';
import { SetupLink } from '@/components/SetupLink';
import { HomeCopyValue } from '@/components/HomeCopyValue';
import { FinancePipelineStrip } from '@/components/FinancePipelineStrip';
import { configuredProcessors, monthToDateIncome, fanbasisMonthToDateIncome } from '@/lib/connectors/payments';
import { incomeAccounts, incomeCategories } from '@/lib/finances';
import { OPERATOR_NAME } from '@/lib/operator';
import { isFilming } from '@/lib/demo-mode';
import { runtimeEnv } from '@/lib/creds';
import { demoIncomeCategories } from '@/lib/finance-demo';
import { previewSandboxTasks, sandboxSpeed } from '@/lib/home-sandbox';

// The operator console in the Example-6 skin (Control AI dashboard) — the
// real home page since 2026-08-23. Teal glass gradient cards, gold chrome,
// dotted-leader details, stacked posting chart (Zernio, comms fallback),
// real freshness gauge. Geist stays. Rendered at / (the /design/home-6 copy is archived).
//
// Layout (SPEC S10): the root is the `home` size container and every card an
// `h6-card` container, so the tiers follow the real content width (dock,
// sidebar, cap) instead of the viewport. DOM order is main -> left -> right;
// grid areas put it back to left | main | right on wide screens. Rules live
// in the "S10 home responsive" block of app/globals.css.

const GOLD = 'var(--accent)'; // the app accent token — follows the OS theme
// Every series colour is a token: the h6-* family re-keys per theme, so no
// colourway shows a foreign hue.
function greeting(): string {
  const hour = new Date().getHours();
  if (hour < 5) return 'Late night';
  if (hour < 12) return 'Good morning';
  if (hour < 18) return 'Good afternoon';
  return 'Good evening';
}

/** The port this server answers on: its env, else the request's Host header, else unknown (row hidden). */
function servingPort(): string | null {
  if (process.env.PORT) return process.env.PORT;
  try {
    const host = headers().get('host') ?? '';
    const m = host.match(/:(\d+)$/);
    return m ? m[1] : null;
  } catch {
    return null;
  }
}

function DottedRow({ k, v, children }: { k: string; v: string; children?: React.ReactNode }) {
  return (
    <div className="h6-drow">
      <span className="shrink-0 text-[12px]" style={{ color: 'var(--text-2)' }}>
        {k}
      </span>
      <span className="h6-drow-lead mx-1 flex-1 border-b border-dotted" style={{ borderColor: 'rgb(var(--mist-rgb) / 0.18)' }} />
      <span className="h6-drow-v ml-auto min-w-0 truncate text-right font-mono text-[11.5px]" style={{ color: 'var(--text)' }} title={children ? undefined : v}>
        {children ?? v}
      </span>
    </div>
  );
}

function SandboxPreview({ tasks }: { tasks: ReturnType<typeof previewSandboxTasks> }) {
  const speed = sandboxSpeed();
  return (
    <section className="h6-card mt-4 p-4" aria-labelledby="h6-sandbox-title" data-sandbox-preview="">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div>
          <h2 id="h6-sandbox-title" className="text-[14px] font-medium" style={{ color: 'var(--text)' }}>Sandbox activity*</h2>
          <p className="mt-1 text-[11px]" style={{ color: 'var(--text-3)' }}>Simulated sandbox activity · sample timings</p>
        </div>
        <Link href="/org" className="text-[11px] hover:underline" style={{ color: GOLD }}>Team map <ArrowUpRight className="ml-0.5 inline h-3 w-3" aria-hidden="true" /></Link>
      </div>

      <div className="mt-4 grid gap-x-5 gap-y-2 sm:grid-cols-3" aria-label="Relative sandbox speed, simulated sample">
        {speed.map((sample) => (
          <div key={sample.name} className="grid grid-cols-[44px_minmax(0,1fr)_48px] items-center gap-2 text-[11px]">
            <span style={{ color: 'var(--text-2)' }}>{sample.name}</span>
            <div className="h-2 overflow-hidden rounded-full" style={{ background: 'rgb(var(--mist-rgb) / 0.1)' }}>
              <div className="h-full rounded-full" style={{ width: `${sample.widthPct}%`, background: GOLD }} />
            </div>
            <span className="text-right font-mono tabular-nums" style={{ color: 'var(--text)' }}>{sample.latencyMs} ms</span>
          </div>
        ))}
      </div>

      <div className="mt-4 grid grid-cols-1 gap-2 sm:grid-cols-3" aria-label="Simulated team tasks">
        {tasks.length ? tasks.map((item, index) => (
          <article key={`${item.agent}-${item.task}`} className="flex min-h-[112px] flex-col justify-between rounded-xl border p-3" style={{ borderColor: 'rgb(var(--mist-rgb) / 0.13)', background: 'rgb(var(--mist-rgb) / 0.025)' }}>
            <div className="flex items-center justify-between gap-2">
              <span className="text-[10px] uppercase tracking-wide" style={{ color: GOLD }}>Sandbox {index + 1}*</span>
              <span className="rounded-full border px-1.5 py-0.5 text-[9px]" style={{ borderColor: 'rgb(var(--mist-rgb) / 0.16)', color: 'var(--text-3)' }}>Simulated</span>
            </div>
            <p className="my-3 line-clamp-2 text-[12px] leading-snug" style={{ color: 'var(--text)' }}>{item.task}</p>
            <p className="truncate text-[10px]" style={{ color: 'var(--text-3)' }}>{item.department} · {item.agent}</p>
          </article>
        )) : (
          <p className="text-[11px]" style={{ color: 'var(--text-3)' }}>No team-map tasks are available for the preview.</p>
        )}
      </div>

      <p className="mt-3 border-t pt-2.5 text-[10px]" style={{ borderColor: 'rgb(var(--mist-rgb) / 0.1)', color: 'var(--text-3)' }}>* Simulated demo data · Powered by Miosa.</p>
    </section>
  );
}

const DAY_FMT: Intl.DateTimeFormatOptions = { weekday: 'short', day: 'numeric', month: 'short', timeZone: 'UTC' };

/** Reference-style dot-matrix mini histogram — one column per day, real counts; each column says its day. */
function DotMatrix({ values, days }: { values: number[]; days: string[] }) {
  const max = Math.max(1, ...values);
  const COLS = values.length;
  const ROWS = 7;
  return (
    <svg viewBox={`0 0 ${COLS * 11} ${ROWS * 9}`} className="h6-matrix shrink-0" role="img" aria-label="Runs per day, last 7 days">
      {values.map((v, x) => {
        const lit = Math.round((v / max) * ROWS);
        const day = days[x] ? new Date(`${days[x]}T00:00:00Z`).toLocaleDateString('en-GB', DAY_FMT) : '';
        return (
          <g key={x}>
            <title>{`${day} · ${v} run${v === 1 ? '' : 's'}`}</title>
            <rect x={x * 11} y={0} width={11} height={ROWS * 9} fill="transparent" />
            {Array.from({ length: ROWS }, (_, y) => (
              <circle key={y} cx={x * 11 + 4} cy={(ROWS - 1 - y) * 9 + 5} r={2.1} fill={y < lit ? GOLD : 'rgb(var(--mist-rgb) / 0.12)'} />
            ))}
          </g>
        );
      })}
    </svg>
  );
}

/** 220° gauge with proportional segments — the reference's sentiment dial, on real freshness. The label sits inside the SVG so it scales with the arc. */
function Gauge({ segs, center, sub }: { segs: { pct: number; color: string }[]; center: string; sub: string }) {
  const R = 62;
  const SW = 9;
  const START = 150; // degrees; opens at the bottom like the reference dial
  const SWEEP = 240;
  let acc = 0;
  const arc = (from: number, to: number) => {
    const a0 = ((START + (from / 100) * SWEEP) * Math.PI) / 180;
    const a1 = ((START + (to / 100) * SWEEP) * Math.PI) / 180;
    // large-arc-flag is about the swept ANGLE, not the percentage: with a
    // 240° sweep a 53% segment is only 127°, so comparing the raw percent
    // against 50 made it draw the long way round and detach from the dial.
    const large = ((to - from) / 100) * SWEEP > 180 ? 1 : 0;
    return `M ${80 + Math.cos(a0) * R} ${78 + Math.sin(a0) * R} A ${R} ${R} 0 ${large} 1 ${80 + Math.cos(a1) * R} ${78 + Math.sin(a1) * R}`;
  };
  return (
    <div className="flex justify-center">
      <svg viewBox="0 0 160 120" style={{ width: 'min(160px, 100%)', height: 'auto' }} role="img" aria-label={`${center} ${sub}`}>
        <path d={arc(0, 100)} fill="none" stroke="rgb(var(--mist-rgb) / 0.1)" strokeWidth={SW} strokeLinecap="round" />
        {segs
          .filter((s) => s.pct > 0)
          .map((s, i) => {
            const from = acc;
            acc += s.pct;
            return <path key={i} d={arc(from, Math.min(100, acc))} fill="none" stroke={s.color} strokeWidth={SW} strokeLinecap="round" />;
          })}
        {/* tick fringe */}
        <circle cx="80" cy="78" r={R - 12} fill="none" stroke="rgb(var(--mist-rgb) / 0.14)" strokeWidth="4" strokeDasharray="1 4" />
        <text x="80" y="74" textAnchor="middle" fontSize="26" fill="var(--text)" style={{ fontVariantNumeric: 'tabular-nums' }}>
          {center}
        </text>
        <text x="80" y="92" textAnchor="middle" fontSize="10.5" fill="var(--text-3)">
          {sub}
        </text>
      </svg>
    </div>
  );
}

/** F2 Needs you (Classic): tasks blocked on the operator, top five, each opening on the board. */
function NeedsYouCard({ needs }: { needs: NeedsYou }) {
  return (
    <section className="h6-card p-4" aria-labelledby="h6-needs-title" data-home-needs="">
      <div className="mb-2.5 flex items-center gap-2.5">
        <span
          className="flex h-5 min-w-5 items-center justify-center rounded-full px-1 text-[10.5px] tabular-nums"
          style={
            needs.total > 0
              ? { background: 'color-mix(in srgb, var(--warn) 12%, transparent)', color: 'var(--warn)', border: '1px solid color-mix(in srgb, var(--warn) 45%, transparent)' }
              : { color: 'var(--text-2)', border: '1px solid rgb(var(--mist-rgb) / 0.16)' }
          }
          title={`${needs.total} ${needs.total === 1 ? 'task is' : 'tasks are'} blocked until you act`}
        >
          {needs.total}
        </span>
        <h2 id="h6-needs-title" className="text-[13.5px] font-medium" style={{ color: 'var(--text)' }}>
          Needs you
        </h2>
        {needs.total > needs.rows.length && (
          <Link href="/tasks" className="ml-auto whitespace-nowrap text-[11.5px] hover:underline" style={{ color: GOLD }}>
            All {needs.total} →
          </Link>
        )}
      </div>
      {needs.rows.length === 0 ? (
        <div className="text-[11.5px]" style={{ color: 'var(--text-2)' }}>
          {NEEDS_YOU_EMPTY}
        </div>
      ) : (
        <ul className="h6-today-list">
          {needs.rows.map((t) => (
            <li key={t.id}>
              <Link
                href={t.href}
                className="h6-row mb-1.5 flex items-center gap-2 rounded-[10px] border px-3 py-2"
                style={{ borderColor: 'rgb(var(--mist-rgb) / 0.09)' }}
                title={`${t.title}${t.agentId ? ` · ${t.agentId}` : ''} · waiting ${t.age}`}
              >
                <span
                  className="h6-led h-1.5 w-1.5 shrink-0 rounded-full"
                  style={{ background: t.priority === 'high' ? 'var(--err)' : 'var(--warn)' }}
                  role="img"
                  aria-label={t.priority === 'high' ? 'high priority' : 'blocked'}
                />
                <span className="min-w-0 flex-1 truncate text-[12px]" style={{ color: 'var(--text)' }}>
                  {t.title}
                </span>
                <span className="shrink-0 whitespace-nowrap font-mono text-[10px]" style={{ color: 'var(--text-2)' }}>
                  {t.age}
                </span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

/** F2 Done today (Classic): runs and tasks finished since local midnight, the count and the newest three. */
function DoneTodayCard({ done }: { done: DoneToday }) {
  return (
    <section className="h6-card p-4" aria-labelledby="h6-done-title" data-home-done="">
      <div className="flex items-center justify-between gap-2">
        <h2 id="h6-done-title" className="text-[13.5px] font-medium" style={{ color: 'var(--text)' }}>
          Done today
        </h2>
        {done.failed > 0 && (
          <span
            className="whitespace-nowrap rounded-full border px-2 py-0.5 text-[10px]"
            style={{ borderColor: 'color-mix(in srgb, var(--err) 50%, transparent)', color: 'var(--err)' }}
          >
            {done.failed} failed
          </span>
        )}
      </div>
      <div className="mt-2 flex items-baseline gap-2">
        <span className="h6-done-num text-[34px] font-normal tabular-nums leading-none" style={{ color: 'var(--text)' }}>
          {done.count}
        </span>
        <span className="text-[11.5px]" style={{ color: 'var(--text-2)' }}>
          {doneCaption(done)}
        </span>
      </div>
      {done.rows.length === 0 ? (
        <div className="mt-2 text-[11.5px]" style={{ color: 'var(--text-2)' }}>
          {DONE_TODAY_EMPTY}
        </div>
      ) : (
        <ul className="h6-today-list mt-3">
          {done.rows.map((r) => (
            <li key={`${r.kind}-${r.id}`}>
              <Link
                href={r.href}
                className="h6-run border-b py-2 text-[11.5px] last:border-b-0 hover:bg-[var(--h6-row-hover)]"
                style={{ borderColor: 'rgb(var(--mist-rgb) / 0.06)' }}
                title={r.title}
              >
                <span
                  className="h6-run-dot h6-led h-1.5 w-1.5 rounded-full"
                  style={{ background: r.ok ? 'var(--h6-live-green)' : 'var(--err)' }}
                  role="img"
                  aria-label={r.ok ? 'finished' : 'failed'}
                />
                <span className="h6-run-id truncate font-mono text-[10.5px]" style={{ color: 'var(--text-2)' }}>
                  {r.kind}
                </span>
                <span className="h6-run-sum min-w-0 truncate" style={{ color: 'var(--text-2)' }}>
                  {r.title}
                </span>
                <span className="h6-run-time whitespace-nowrap font-mono text-[10px]" style={{ color: 'var(--text-2)' }}>
                  {r.age}
                </span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

export async function Home6Console() {
  const db = getDb();
  // F1 (founderos-ui, K3): no Optimal Engine health on Home, so no `engine doctor`
  // subprocess here. The store figures are a local file walk (localStats).
  const [connections, store, feed, postDays] = await Promise.all([
    allConnectorStatuses(),
    createGBrainProvider().localStats(),
    gatherCommsFeed(200),
    zernioPostDays(),
  ]);

  // Real income by processor, the same derivation /finances uses — never a
  // demo chart. Only Stripe's MTD total (not the full balance snapshot) is
  // needed for this compact strip.
  const stripeKeyed = configuredProcessors(runtimeEnv()).some((p) => p.id === 'stripe' && p.configured);
  // Filming never touches the live processors: the strip shows the demo books.
  const filming = isFilming();
  const [mtd, fbAa, fbMer] = filming
    ? [null, null, null]
    : await Promise.all([
        stripeKeyed ? monthToDateIncome().catch(() => null) : Promise.resolve(null),
        fanbasisMonthToDateIncome(runtimeEnv().FANBASIS_KEY_2).catch(() => null),
        fanbasisMonthToDateIncome(runtimeEnv().FANBASIS_KEY_1).catch(() => null),
      ]);
  const financeConfiguredMap = Object.fromEntries(configuredProcessors(runtimeEnv()).map((p) => [p.id, p.configured]));
  const financeLiveIncomeUsd: Record<string, number> = {};
  if (fbAa != null) financeLiveIncomeUsd['fanbasis-2'] = fbAa;
  if (fbMer != null) financeLiveIncomeUsd['fanbasis-1'] = fbMer;
  const financeAccounts = incomeAccounts(
    { connected: stripeKeyed && mtd != null, mtdUsd: mtd ? mtd.amountCents / 100 : null },
    financeConfiguredMap,
    financeLiveIncomeUsd,
  );
  // Filming shows the demo books, the same split /finances shows in Filming mode.
  const financeCategories = filming ? demoIncomeCategories() : incomeCategories(financeAccounts);
  const hasFinanceData = financeCategories.some((c) => c.usd > 0);
  // Same rule as /finances: the demo books are year to date, live pulls month to date.
  const financePeriod: 'MTD' | 'YTD' = filming ? 'YTD' : 'MTD';
  const brain = loadObservatoryData();

  const agents = db.agents.all();
  const sandboxTasks = previewSandboxTasks(db.sopTasks.all(), agents, db.departments.all());
  const recentRuns = db.agentRuns.recent(400);
  const connected = connections.filter((c) => c.state === 'connected').length;
  const activeAgents = agents.filter((a) => a.status === 'active').length;
  const inbound = inboundLast24h(feed);
  const runs7 = runsPerDay(recentRuns, 7);
  const runs7Days = Array.from({ length: 7 }, (_, i) => new Date(Date.now() - (6 - i) * 86_400_000).toISOString().slice(0, 10));
  const runs7Total = runs7.reduce((s, v) => s + v, 0);
  const failed = db.agentRuns.recent(40).filter((r) => !r.ok).length;
  const hero = stateOfWorld({
    activeAgents,
    totalAgents: agents.length,
    connected,
    totalConnectors: connections.length,
    inbound,
    failedRuns: failed,
  });
  // H2: the lead pill is derived from the segments, never a static "Live".
  const pill = statusPill(hero);
  const PILL_STYLE: Record<typeof pill.tone, { text: string; dot: string; rgb: string }> = {
    ok: { text: 'var(--h6-live-green)', dot: 'var(--h6-live-green)', rgb: 'rgb(var(--live-badge-rgb) / 0.45)' },
    warn: { text: 'var(--text)', dot: 'var(--warn)', rgb: 'color-mix(in srgb, var(--warn) 50%, transparent)' },
    err: { text: 'var(--err)', dot: 'var(--err)', rgb: 'color-mix(in srgb, var(--err) 50%, transparent)' },
  };
  const pillStyle = PILL_STYLE[pill.tone];
  // H4: a 0 in "runs · 7d" is explained by when the newest run happened (C20: read latest()).
  const lastRun = db.agentRuns.latest();
  const lastRunText = lastRunCaption(lastRun?.finishedAt ?? null);

  // posting chart — real Zernio history, stacked by platform
  const TRACKED: SocialPlatform[] = ['instagram', 'tiktok', 'twitter', 'youtube', 'linkedin'];
  const today = new Date().toISOString().slice(0, 10);
  const posting = postSeriesFromDays(postDays, today, 90, TRACKED);
  const days = posting[0]?.points.map((p) => p.date) ?? [];
  const chartSeries: Home6Series[] = posting.map((s) => ({
    key: s.key,
    label: PLATFORM_LABELS[s.key as SocialPlatform],
    color: channelColor(s.key), // one series colour per channel (lib/series.ts, SPEC C19)
    points: s.points.map((p) => p.count),
  }));
  const totalPosts = posting.reduce((s, p) => s + p.total, 0);

  // fallback series when Zernio has no posting history (creds pending): real
  // inbound comms per day over 30 days, stacked by source
  const COMMS_COLORS: Record<string, string> = { email: 'var(--h6-gold)', whatsapp: 'var(--h6-teal)', slack: 'var(--h6-indigo)' };
  const commsAxis = Array.from({ length: 30 }, (_, i) => {
    const d = new Date();
    d.setDate(d.getDate() - (29 - i));
    return d.toISOString().slice(0, 10);
  });
  const commsSources = [...new Set(feed.map((f) => f.source))];
  const commsSeries: Home6Series[] = commsSources.map((src) => ({
    key: src,
    label: src,
    color: COMMS_COLORS[src] ?? GOLD,
    points: commsAxis.map((day) => feed.filter((f) => f.source === src && f.ts?.slice(0, 10) === day).length),
  }));
  const totalComms = commsSeries.reduce((s, sr) => s + sr.points.reduce((a, b) => a + b, 0), 0);

  // Filming mode keeps the home chart on the comms mix even when posting history exists.
  const usePosting = !isFilming() && totalPosts > 0;
  const chartTitle = usePosting
    ? `Posting activity · 90 days · ${totalPosts} posts`
    : `Inbound comms · 30 days · ${totalComms} messages`;
  const chartHref = usePosting ? '/social' : '/inbox';
  const activeDays = usePosting ? days : commsAxis;
  const activeSeries = usePosting ? chartSeries : commsSeries;
  const hasChart = usePosting || totalComms > 0;

  // audience tiles — latest real value + real delta when history exists
  const { channels } = audienceSeries(db);
  const audienceTiles = channels.slice(0, 4).map((c) => {
    const pts = c.points;
    const last = pts[pts.length - 1]?.value ?? 0;
    const prev = pts.length > 1 ? pts[pts.length - 2].value : null;
    const delta = prev != null && prev > 0 ? ((last - prev) / prev) * 100 : null;
    // each tile opens its own audience page; the email list is the beehiiv newsletter (Q-HOME-4)
    const href = c.key === 'email' ? '/social/beehiiv' : SocialPlatformSchema.safeParse(c.key).success ? `/social/${c.key}` : '/social';
    return { key: c.key, label: c.label, last, delta, href };
  });

  // freshness gauge — real mtime buckets from the brain store
  const dated = brain.pages.filter((p) => p.days != null);
  const freshN = dated.filter((p) => (p.days ?? 0) <= 30).length;
  const staleN = dated.filter((p) => (p.days ?? 0) >= 90).length;
  const agingN = dated.length - freshN - staleN;
  const pct = (n: number) => (dated.length > 0 ? Math.round((n / dated.length) * 100) : 0);

  const lastRuns = db.agentRuns.recent(5);
  const nowQuarter = groupRoadmapByQuarter(db.roadmap.all())[0];
  const nowItems = nowQuarter?.items.slice(0, 4) ?? [];
  // H3: the header counts every active agent; the list shows the first 4 and says how many more.
  const liveAgents = agents.filter((a) => a.status === 'active').slice(0, 4);
  const moreLive = activeAgents - liveAgents.length;
  const storeShort = store.storePath.replace(process.env.HOME ?? '', '~');
  const port = servingPort();
  const journal = db.journalMode().toUpperCase();

  const kpis = [
    { label: 'Systems', value: connected, unit: `/${connections.length}`, tick: 'var(--h6-teal)', href: '/integrations' },
    { label: 'Agents', value: activeAgents, unit: `/${agents.length}`, tick: GOLD, href: '/agents' },
    { label: 'Inbound', value: inbound, unit: '24h', tick: 'var(--h6-indigo)', href: '/inbox' },
    { label: 'Runs 7d', value: runs7Total, unit: failed > 0 ? `${failed} fail` : '', tick: 'var(--h6-gold)', href: '/agents' },
    { label: 'Pages', value: store.markdownFiles, unit: '', tick: 'var(--h6-ice)', href: '/brain' },
  ];

  // F2: what waits on the operator and what finished since local midnight.
  const nowDate = new Date();
  const allTasks = db.agentTasks.all();
  const needs = needsYou(allTasks, nowDate);
  const done = doneToday(allTasks, recentRuns, nowDate);

  // GladOS tree (glados-ui SPEC S01, X5): the same data, fetched once above, in the kit's presentation.
  if (treeFor(getAppearance(), '/') === 'glados') {
    const now = new Date();
    const todayIso = now.toISOString().slice(0, 10);
    const data: HomeGData = {
      greeting: `${greeting()}, ${OPERATOR_NAME}`,
      status: {
        pill,
        segments: hero.slice(0, 3).map((s) => ({ text: s.text, href: s.href ?? null, tone: s.tone })),
        degraded: hero.some((s) => s.tone === 'warn' || s.tone === 'err'),
      },
      revenue: hasFinanceData ? revenueSummary(financeCategories) : null,
      financePeriod,
      financeCategories: hasFinanceData ? financeCategories : [],
      inbound: inboundSummary(feed, connections, now, 200),
      hourStarts: hourlyBucketStarts(now),
      stats: {
        connected,
        connectors: connections.length,
        activeAgents,
        agents: agents.length,
        runs7: runs7Total,
        failed,
        brainPages: store.markdownFiles,
      },
      chart: {
        usePosting,
        title: usePosting ? 'Posting activity' : 'Inbound comms',
        caption: usePosting ? `90 days · ${totalPosts} posts` : `30 days · ${totalComms} messages`,
        href: chartHref,
        hasChart,
        days: activeDays,
        series: (usePosting ? chartSeries : commsSeries).map((s) => ({ key: s.key, label: s.label, points: s.points })),
      },
      runs: {
        perDay: runs7,
        days: runs7Days,
        lastRunText,
        lastRunTitle: lastRun ? `Newest run: ${lastRun.agentId}, finished ${new Date(lastRun.finishedAt).toLocaleString('en-US')}` : null,
        recent: lastRuns.map((r) => ({ id: r.id, agentId: r.agentId, ok: r.ok, summary: r.summary, age: relAge(r.finishedAt, now), at: r.finishedAt })),
      },
      freshness: { dated: dated.length, fresh: freshN, aging: agingN, stale: staleN },
      audience: channels.map((c) => {
        const pts = c.points.map((p) => ({ date: p.date.slice(0, 10), value: p.value }));
        const win = seriesWindow(30, pts, todayIso);
        const change = netChange(pts, win);
        return {
          key: c.key,
          label: c.label,
          last: pts[pts.length - 1]?.value ?? 0,
          spark: windowPoints(pts, win).map((p) => p.value),
          delta: change ? formatDelta(change.current, change.baseline) : null,
          window: `${win.start} to ${win.end}`,
          href: c.key === 'email' ? '/social/beehiiv' : SocialPlatformSchema.safeParse(c.key).success ? `/social/${c.key}` : '/social',
        };
      }),
      needs,
      done: { ...done, caption: doneCaption(done) },
      live: { agents: liveAgents.map((a) => ({ id: a.id, name: a.name })), active: activeAgents, more: moreLive },
      now: nowItems.length > 0 ? { quarter: nowQuarter?.quarter ?? '', items: nowItems.map((it) => ({ id: it.id, title: it.title })) } : null,
    };
    return <HomeG data={data} />;
  }

  return (
    <div
      className="h6-home relative -mx-8 -mb-16 -mt-7 min-h-[calc(100vh-52px)] pt-7 wide:-mx-10 ultra:-mx-12"
      style={{ background: 'var(--bg)', color: 'var(--text)' }}
    >
      {/* atmosphere — teal under console, ember under red (--h6-glow-bg) */}
      <div className="pointer-events-none absolute inset-0" style={{ background: 'var(--h6-glow-bg)' }} />

      <div className="h6-inner relative">
        {/* header */}
        <div className="mb-5">
          <div className="h6-greeting font-medium leading-tight tracking-[-0.01em]" style={{ color: 'var(--text)' }}>
            {greeting()}, {OPERATOR_NAME}
          </div>
          <div className="mt-2.5 flex flex-wrap items-center gap-x-2.5 gap-y-1.5 text-[12.5px]">
            <span
              className="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-[11px]"
              style={{ borderColor: pillStyle.rgb, color: pillStyle.text }}
              title="Overall status from failed runs and connectors"
            >
              <span aria-hidden="true" className="h-1.5 w-1.5 rounded-full" style={{ background: pillStyle.dot }} />
              {pill.text}
            </span>
            <span className="flex flex-wrap items-center gap-x-1.5 gap-y-1" style={{ color: 'var(--text-2)' }}>
              {hero.slice(0, 3).map((s, i) => (
                <span key={s.text} className="h6-seg">
                  {i > 0 && <span aria-hidden="true">· </span>}
                  {s.href ? (
                    <Link href={s.href} className="h6-seg" style={{ color: 'inherit' }}>
                      {s.text}
                    </Link>
                  ) : (
                    s.text
                  )}
                </span>
              ))}
            </span>
            {hero.some((s) => s.tone === 'warn' || s.tone === 'err') && (
              <SetupLink href="/setup" hint="something needs wiring — open the guided setup" />
            )}
          </div>
        </div>

        <div className="h6-grid">
          {/* ——— main: KPIs, chart, finances ——— */}
          <div className="h6-col h6-col-main">
            <div className="h6-kpis">
              {kpis.map((k) => {
                const full = formatFull(k.value);
                const shown = formatCompact(k.value);
                return (
                  <Link
                    key={k.label}
                    href={k.href}
                    className="h6-card h6-kpi p-3"
                    aria-label={`${k.label} ${full}${k.unit ? ` ${k.unit}` : ''}`}
                  >
                    <div className="flex items-center gap-1.5 whitespace-nowrap text-[10.5px]" style={{ color: 'var(--text-2)' }}>
                      <span className="h-3 w-[3px] shrink-0 rounded-full" style={{ background: k.tick }} />
                      <span className="truncate" title={k.label}>
                        {k.label}
                      </span>
                    </div>
                    <div className="h6-kpi-value mt-2 font-normal tabular-nums leading-none" style={{ color: 'var(--text)' }} title={full}>
                      <span>{shown}</span>
                      {k.unit && (
                        <span className="whitespace-nowrap text-[10px]" style={{ color: 'var(--text-3)' }}>
                          {k.unit}
                        </span>
                      )}
                    </div>
                  </Link>
                );
              })}
            </div>

            <InterjectClassic />

            <div className="h6-card h6-chart-card">
              <div className="h6-chart-head mb-3">
                <span className="min-w-0 text-[12.5px]" style={{ color: 'var(--text-2)' }}>
                  {chartTitle}
                  <Link href={chartHref} className="ml-2.5 whitespace-nowrap text-[11.5px] hover:underline" style={{ color: GOLD }}>
                    View all →
                  </Link>
                </span>
                {hasChart && (
                  <div className="h6-chart-legend" aria-label="Legend">
                    {activeSeries.map((s) => (
                      <span key={s.key} className="flex items-center gap-1.5 whitespace-nowrap text-[10.5px]" style={{ color: 'var(--text-2)' }}>
                        <span className="h-2 w-2 rounded-[3px]" style={{ background: s.color }} />
                        {s.label}
                      </span>
                    ))}
                  </div>
                )}
              </div>
              {hasChart ? (
                <Home6Chart days={activeDays} series={activeSeries} title={chartTitle} />
              ) : (
                <div className="h6-chart-skeleton flex flex-col items-center justify-center gap-2 text-[12px]" style={{ color: 'var(--text-3)' }}>
                  No posting or comms history yet
                  <SetupLink href={setupHrefFor('zernio')} hint="connect Zernio for posting history" />
                </div>
              )}
            </div>

            {hasFinanceData && <FinancePipelineStrip categories={financeCategories} period={financePeriod} />}
          </div>

          {/* ——— left column ——— */}
          <div className="h6-col h6-col-left">
            {/* agent activity */}
            <Link
              href="/agents"
              className="h6-card p-4"
              aria-label={`Agent activity: ${runs7Total} runs in 7 days, ${failed > 0 ? `${failed} failed` : 'all ok'}${lastRunText ? `, ${lastRunText}` : ''}`}
            >
              <div className="flex items-center justify-between gap-2">
                <span className="text-[12.5px]" style={{ color: 'var(--text-2)' }}>
                  Agent activity
                </span>
                <span
                  className="whitespace-nowrap rounded-full border px-2 py-0.5 text-[10px]"
                  style={
                    failed > 0
                      ? { borderColor: 'color-mix(in srgb, var(--err) 50%, transparent)', color: 'var(--err)' }
                      : { borderColor: 'color-mix(in srgb, var(--accent) 40%, transparent)', color: GOLD }
                  }
                >
                  {failed > 0 ? `${failed} failed` : 'all ok'}
                </span>
              </div>
              <div className="mt-2 flex flex-wrap items-end justify-between gap-3">
                <div className="h6-activity-num whitespace-nowrap font-normal tabular-nums leading-none" style={{ color: 'var(--text)' }}>
                  {runs7Total}
                  <span className="ml-1.5 text-[11.5px]" style={{ color: 'var(--text-3)' }}>
                    runs · 7d
                  </span>
                </div>
                <DotMatrix values={runs7} days={runs7Days} />
              </div>
              <div
                className="mt-2 text-[11px]"
                style={{ color: 'var(--text-3)' }}
                title={lastRun ? `Newest run: ${lastRun.agentId}, finished ${new Date(lastRun.finishedAt).toLocaleString('en-US')}` : undefined}
              >
                {lastRunText ?? 'No agent runs recorded yet'}
              </div>
            </Link>

            {/* brain freshness gauge */}
            <div className="h6-card p-4">
              <div className="mb-1 text-[12.5px]" style={{ color: 'var(--text-2)' }}>
                Brain freshness
              </div>
              {dated.length === 0 ? (
                <div className="flex flex-col items-start gap-2 py-3 text-[11.5px]" style={{ color: 'var(--text-3)' }}>
                  Brain store not mounted
                  <SetupLink href="/setup" hint="expected at ~/knowledge/brain-store (or GBRAIN_STORE)" />
                </div>
              ) : (
                <>
                  <Gauge
                    segs={[
                      { pct: pct(freshN), color: GOLD },
                      { pct: pct(agingN), color: 'var(--text-2)' },
                      { pct: pct(staleN), color: 'var(--h6-stale)' },
                    ]}
                    center={`${brain.totalPages}`}
                    sub="Pages tracked"
                  />
                  <div className="h6-fresh mt-2">
                    {[
                      { label: 'Fresh', n: freshN, color: GOLD, href: '/brain', hint: 'changed in the last 30 days' },
                      { label: 'Aging', n: agingN, color: 'var(--text-2)', href: '/brain', hint: 'changed 31 to 89 days ago' },
                      { label: 'Stale', n: staleN, color: 'var(--h6-stale)', href: '/drift', hint: 'untouched for 90 days or more' },
                    ].map((s) => (
                      <Link
                        key={s.label}
                        href={s.href}
                        className="h6-fresh-tile h6-row rounded-[10px] border p-2"
                        style={{ borderColor: 'rgb(var(--mist-rgb) / 0.09)' }}
                        title={`${s.label}: ${s.n} pages ${s.hint}`}
                      >
                        <div className="flex items-center gap-1.5 text-[10px]" style={{ color: 'var(--text-2)' }}>
                          <span className="h-2.5 w-[3px] rounded-full" style={{ background: s.color }} />
                          {s.label}
                        </div>
                        <div className="h6-fresh-num mt-1 whitespace-nowrap text-[14px] font-medium tabular-nums" style={{ color: 'var(--text)' }}>
                          {pct(s.n)}%
                          <span className="ml-1 text-[10px] font-normal" style={{ color: 'var(--text-3)' }}>
                            /{s.n}
                          </span>
                        </div>
                        <div className="h6-fresh-track mt-1.5 h-[2px] rounded-full" style={{ background: `color-mix(in oklab, ${s.color} 33%, transparent)` }}>
                          <div className="h-full rounded-full" style={{ width: `${pct(s.n)}%`, background: s.color }} />
                        </div>
                      </Link>
                    ))}
                  </div>
                </>
              )}
            </div>

            {/* audience */}
            <div className="h6-card p-4">
              <div className="mb-3 text-[12.5px]" style={{ color: 'var(--text-2)' }}>
                Audience
              </div>
              {audienceTiles.length === 0 ? (
                <div className="flex flex-col items-start gap-2 text-[11.5px]" style={{ color: 'var(--text-3)' }}>
                  No audience history yet
                  <SetupLink href={setupHrefFor('zernio')} hint="connect Zernio to track followers" />
                </div>
              ) : (
                <div className="h6-audience">
                  {audienceTiles.map((t) => (
                    <Link
                      key={t.key}
                      href={t.href}
                      className="h6-row min-w-0 rounded-[11px] border p-2.5"
                      style={{ borderColor: 'rgb(var(--mist-rgb) / 0.09)' }}
                      aria-label={`${t.label} ${formatFull(t.last)}${t.delta != null ? `, ${t.delta >= 0 ? 'up' : 'down'} ${Math.abs(t.delta).toFixed(1)}%` : ''}`}
                    >
                      <div className="truncate text-[10.5px]" style={{ color: 'var(--text-2)' }} title={t.label}>
                        {t.label}
                      </div>
                      <div className="h6-audience-num mt-1.5 text-[16px] font-medium tabular-nums" style={{ color: 'var(--text)' }} title={formatFull(t.last)}>
                        <span>{formatCompact(t.last)}</span>
                        {t.delta != null && (
                          <span
                            className="whitespace-nowrap text-[10px] font-normal"
                            style={{ color: Math.abs(t.delta) < 0.05 ? 'var(--text-3)' : t.delta > 0 ? 'var(--h6-live-green)' : 'var(--err)' }}
                          >
                            {t.delta > 0 && Math.abs(t.delta) >= 0.05 ? '+' : ''}
                            {Math.abs(t.delta) < 0.05 ? '0.0' : t.delta.toFixed(1)}%
                          </span>
                        )}
                      </div>
                    </Link>
                  ))}
                </div>
              )}
            </div>

            {/* recent runs */}
            <div className="h6-card p-4">
              <div className="mb-2.5 text-[12.5px]" style={{ color: 'var(--text-2)' }}>
                Recent runs
              </div>
              {lastRuns.length === 0 && (
                <div className="text-[11.5px]" style={{ color: 'var(--text-3)' }}>
                  no agent runs recorded yet
                </div>
              )}
              {lastRuns.map((r) => (
                <Link
                  key={r.id}
                  href={`/agents?agent=${encodeURIComponent(r.agentId)}`}
                  className="h6-run border-b py-2 text-[11.5px] last:border-b-0 hover:bg-[var(--h6-row-hover)]"
                  style={{ borderColor: 'rgb(var(--mist-rgb) / 0.06)' }}
                >
                  <span className="h6-run-dot h-1.5 w-1.5 rounded-full" style={{ background: r.ok ? 'var(--h6-live-green)' : 'var(--err)' }} />
                  <span className="h6-run-id truncate font-mono text-[10.5px]" style={{ color: 'var(--text-2)' }} title={r.agentId}>
                    {r.agentId}
                  </span>
                  <span className="h6-run-sum min-w-0 truncate" style={{ color: 'var(--text-3)' }} title={r.summary}>
                    {r.summary}
                  </span>
                  <span className="h6-run-time whitespace-nowrap font-mono text-[10px]" style={{ color: 'var(--text-3)' }} title={new Date(r.finishedAt).toLocaleString('en-US')}>
                    {relAge(r.finishedAt)}
                  </span>
                </Link>
              ))}
            </div>
          </div>

          {/* ——— right rail ——— */}
          <div className="h6-col h6-col-right">
            <div className="h6-card p-4">
              <div className="mb-1 text-[13.5px] font-medium" style={{ color: 'var(--text)' }}>
                Details
              </div>
              <DottedRow k="Store" v={storeShort}>
                <HomeCopyValue text={storeShort} copy={store.storePath} />
              </DottedRow>
              <DottedRow k="Pages" v={formatFull(store.markdownFiles)} />
              {journal && <DottedRow k="Database" v={`sqlite · ${journal}`} />}
              {port && <DottedRow k="Port" v={port} />}
            </div>

            {/* F2 takes the slot the Optimal Engine health tile had (F1) */}
            <NeedsYouCard needs={needs} />
            <DoneTodayCard done={done} />

            <div className="h6-card p-4">
              <div className="mb-2.5 flex items-center gap-2.5">
                <span
                  className="flex h-5 min-w-5 items-center justify-center rounded-full px-1 text-[10.5px] tabular-nums"
                  style={{ background: 'color-mix(in srgb, var(--accent) 10%, transparent)', color: GOLD, border: '1px solid color-mix(in srgb, var(--accent) 35%, transparent)' }}
                  title={`${activeAgents} of ${agents.length} agents active`}
                >
                  {activeAgents}
                </span>
                <span className="text-[13.5px] font-medium" style={{ color: 'var(--text)' }}>
                  Agents live
                </span>
              </div>
              {liveAgents.map((a) => (
                <Link
                  key={a.id}
                  href={`/agents?agent=${encodeURIComponent(a.id)}`}
                  className="h6-row mb-1.5 flex items-center gap-2 rounded-[10px] border px-3 py-2"
                  style={{ borderColor: 'rgb(var(--mist-rgb) / 0.09)' }}
                >
                  <ChevronRight className="h-3 w-3 shrink-0" style={{ color: GOLD }} />
                  <span className="min-w-0 flex-1 truncate text-[12px]" style={{ color: 'var(--text)' }} title={a.name}>
                    {a.name}
                  </span>
                  <ArrowUpRight className="h-3 w-3 shrink-0" style={{ color: 'var(--text-3)' }} />
                </Link>
              ))}
              {moreLive > 0 && (
                <Link
                  href="/agents"
                  className="h6-row flex items-center rounded-[10px] border px-3 py-2 text-[12px]"
                  style={{ borderColor: 'rgb(var(--mist-rgb) / 0.09)', color: 'var(--text-2)' }}
                >
                  +{moreLive} more
                </Link>
              )}
              {liveAgents.length === 0 && (
                <div className="flex flex-col items-start gap-2 text-[11.5px]" style={{ color: 'var(--text-3)' }}>
                  No agents live
                  <SetupLink href="/agents" hint="no agent is active; open Agents to set one up" />
                </div>
              )}
            </div>

            {nowItems.length > 0 && (
              <div className="h6-card p-4">
                <div className="mb-2.5 flex items-center gap-2.5">
                  <span
                    className="flex h-5 w-5 items-center justify-center rounded-full text-[10.5px] tabular-nums"
                    style={{ background: 'color-mix(in srgb, var(--accent) 10%, transparent)', color: GOLD, border: '1px solid color-mix(in srgb, var(--accent) 35%, transparent)' }}
                  >
                    {nowItems.length}
                  </span>
                  <span className="text-[13.5px] font-medium" style={{ color: 'var(--text)' }}>
                    Now · {nowQuarter?.quarter}
                  </span>
                </div>
                {nowItems.map((it) => (
                  <Link
                    key={it.id}
                    href="/roadmap"
                    className="h6-row mb-1.5 flex items-start gap-2 rounded-[10px] border px-3 py-2"
                    style={{ borderColor: 'rgb(var(--mist-rgb) / 0.09)' }}
                  >
                    <ChevronRight className="mt-0.5 h-3 w-3 shrink-0" style={{ color: GOLD }} />
                    <span className="min-w-0 flex-1 text-[12px] leading-snug" style={{ color: 'var(--text)' }}>
                      <span className="line-clamp-3 break-words" title={it.title}>
                        {it.title}
                      </span>
                      <span className="mt-1 flex items-center gap-1 text-[10px]" style={{ color: 'var(--text-3)' }}>
                        <Calendar className="h-2.5 w-2.5" /> {nowQuarter?.quarter}
                      </span>
                    </span>
                  </Link>
                ))}
              </div>
            )}
          </div>
        </div>
        <SandboxPreview tasks={sandboxTasks} />
      </div>
    </div>
  );
}
