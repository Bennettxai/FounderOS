import type { Meter, Tone } from '@/components/slab';
import type { SeriesPoint } from '@/components/slab-charts';
import type { Agent, AgentRun, LeadMagnet } from '@/lib/schemas';
import type { ZernioPost, ZernioPostDay } from '@/lib/connectors/zernio';
import { shortLabels } from '@/lib/short-labels';

/**
 * /content in the Brand Deals look (Alex, 2026-09-24). One pure pass over
 * the page's own rows (the content crew, their real agent runs, the lead
 * magnets table, the Zernio recent pull and its posting history) gives the
 * Content Volume headline and meters, the posting step line, the crew-runs
 * dot matrix and the single "Since last post" card. Posting days are UTC
 * dates (Zernio stamps them that way), so the page passes `today` in UTC.
 */
export type ContentVolumeInput = {
  crew: Array<Pick<Agent, 'id' | 'name' | 'status'>>;
  runs: Array<Pick<AgentRun, 'agentId' | 'startedAt' | 'ok'>>;
  leadMagnets: Array<Pick<LeadMagnet, 'name' | 'status'>>;
  recent: Array<Pick<ZernioPost, 'status'>>;
  postDays: ZernioPostDay[];
  /** false when Zernio gave no answer: the history is unknown, not empty. */
  postsKnown?: boolean;
  /** YYYY-MM-DD (UTC), computed by the page so the axis cannot drift */
  today: string;
  days?: number;
};

export type ContentVolume = {
  headline: number;
  chips: Array<{ tone?: Tone; text: string }>;
  caption: string;
  meters: Meter[];
  foot: string;
  series: SeriesPoint[];
  postsInWindow: number;
  activeDays: number;
  crewRuns: SeriesPoint[];
  runsInWindow: number;
  magnets: { total: number; live: number; draft: number; paused: number; archived: number };
  insight: { value: number; display?: string; headline: string; body: string; frac: number };
};

const DAY_MS = 86_400_000;
const CREW_COLS = 6;
const frac = (n: number, d: number) => (d > 0 ? Math.max(0, Math.min(1, n / d)) : 0);
const pct = (f: number) => `${Math.round(f * 100)}%`;
const plural = (n: number, one: string, many = `${one}s`) => `${n} ${n === 1 ? one : many}`;
const utcLabel = (d: Date) => d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', timeZone: 'UTC' });


export function contentVolume(x: ContentVolumeInput): ContentVolume {
  const days = x.days ?? 30;
  const end = new Date(`${x.today}T00:00:00Z`);
  const start = new Date(end.getTime() - (days - 1) * DAY_MS);
  const endOfToday = end.getTime() + DAY_MS;

  // Posts per UTC day across the window, oldest first, ending today.
  const keys: string[] = [];
  const buckets = new Map<string, number>();
  const labels = new Map<string, string>();
  for (let i = 0; i < days; i++) {
    const d = new Date(start.getTime() + i * DAY_MS);
    const key = d.toISOString().slice(0, 10);
    keys.push(key);
    buckets.set(key, 0);
    labels.set(key, utcLabel(d));
  }
  let lastPost: string | null = null;
  for (const p of x.postDays) {
    const key = p.date.slice(0, 10);
    if (key <= x.today && (lastPost === null || key > lastPost)) lastPost = key;
    if (buckets.has(key)) buckets.set(key, (buckets.get(key) ?? 0) + 1);
  }
  const series = keys.map((k) => ({ label: labels.get(k) ?? k, count: buckets.get(k) ?? 0 }));
  const postsInWindow = series.reduce((n, s) => n + s.count, 0);
  const activeDays = series.filter((s) => s.count > 0).length;

  // Crew runs inside the same window.
  const inWindow = x.runs.filter((r) => {
    const t = new Date(r.startedAt).getTime();
    return Number.isFinite(t) && t >= start.getTime() && t < endOfToday;
  });
  const crewIds = new Set(x.crew.map((a) => a.id));
  const crewRunsAll = inWindow.filter((r) => crewIds.has(r.agentId));
  const runsOk = crewRunsAll.filter((r) => r.ok).length;
  const runsInWindow = crewRunsAll.length;
  const names = shortLabels(x.crew.map((a) => a.name));
  const crewRuns = x.crew
    .map((a, i) => ({ label: names[i], count: crewRunsAll.filter((r) => r.agentId === a.id).length }))
    .slice(0, CREW_COLS);

  const magnetCount = (s: LeadMagnet['status']) => x.leadMagnets.filter((m) => m.status === s).length;
  const magnets = { total: x.leadMagnets.length, live: magnetCount('live'), draft: magnetCount('draft'), paused: magnetCount('paused'), archived: magnetCount('archived') };
  const live = magnets.live;
  const activeCrew = x.crew.filter((a) => a.status === 'active').length;

  const chips: ContentVolume['chips'] = [];
  if (activeDays) chips.push({ tone: 'accent', text: plural(activeDays, 'active day') });
  if (live) chips.push({ tone: 'ok', text: `${live} magnet${live === 1 ? '' : 's'} live` });

  const meters: Meter[] = [];
  if (postsInWindow > 0) {
    const f = frac(activeDays, days);
    meters.push({ label: `Active posting days (${activeDays}/${days})`, frac: f, display: pct(f), hue: 'var(--accent)' });
  }
  if (x.leadMagnets.length > 0) {
    const f = frac(live, x.leadMagnets.length);
    meters.push({ label: `Lead magnets live (${live}/${x.leadMagnets.length})`, frac: f, display: `${live} of ${x.leadMagnets.length}`, hue: 'var(--ok)' });
  }
  if (x.crew.length > 0) {
    const f = frac(activeCrew, x.crew.length);
    meters.push({ label: `Crew active (${activeCrew}/${x.crew.length})`, frac: f, display: pct(f), hue: 'var(--ramp-1)' });
  }
  if (runsInWindow > 0) {
    meters.push({
      label: `Crew runs OK (${runsOk}/${runsInWindow})`,
      frac: frac(runsOk, runsInWindow),
      display: `${runsOk} ok · ${runsInWindow - runsOk} failed`,
      hue: 'var(--ramp-4)',
    });
  }

  // The one gradient card: how long the feed has been quiet.
  let insight: ContentVolume['insight'];
  if (x.postsKnown === false) {
    insight = { value: 0, display: ' - ', headline: 'Posting history unavailable.', body: 'Zernio did not answer, so days since the last post are unknown.', frac: 0 };
  } else if (lastPost === null) {
    insight = { value: 0, display: 'none', headline: 'Nothing posted on record yet.', body: 'Posts show here once Zernio reports its history.', frac: 0 };
  } else {
    const quiet = Math.max(0, Math.round((end.getTime() - new Date(`${lastPost}T00:00:00Z`).getTime()) / DAY_MS));
    insight = {
      value: quiet,
      headline: quiet === 0 ? 'Posted today.' : `${plural(quiet, 'day')} since the last post went out.`,
      body: `${plural(activeDays, 'active day')} in the last ${days} · ticks light with posting consistency.`,
      frac: frac(activeDays, days),
    };
  }

  return {
    headline: postsInWindow,
    chips,
    caption:
      x.postsKnown === false
        ? 'posting history unavailable · Zernio not answering'
        : postsInWindow > 0
          ? `posts out through Zernio, last ${days} days`
          : `no posts on record in the last ${days} days`,
    meters,
    foot: `${plural(x.crew.length, 'agent')} · ${plural(x.leadMagnets.length, 'lead magnet')} · ${plural(x.recent.length, 'recent post')} pulled`,
    series,
    postsInWindow,
    activeDays,
    crewRuns,
    runsInWindow,
    magnets,
    insight,
  };
}
