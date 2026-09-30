/**
 * /analytics in the Brand Deals look (2026-09-24). The page's own numbers,
 * shaped like Deal Volume: total reach as the headline with one meter per
 * channel (each its real share of reach), the run log split by agent, a
 * weekday rhythm for the dot matrix, and the one insight card asking for the
 * credentials still missing. Pure: the page feeds it repo rows, nothing here
 * invents a number, and an empty input stays empty.
 */

export type VolumeMeterRow = { label: string; frac: number; display: string; hue: string };
export type VolumeChip = { tone?: 'ok' | 'warn' | 'err' | 'accent'; text: string };

export type AnalyticsVolume = {
  reach: number;
  headline: string;
  chips: VolumeChip[];
  caption: string;
  meters: VolumeMeterRow[];
  foot: string;
  runs: { total: number; ok: number; failed: number; okPct: number | null; agents: number; meters: VolumeMeterRow[] };
  rhythm: { label: string; count: number }[];
  rhythmTotal: number;
  insight: { value: number; headline: string; body: string; frac: number };
};

const HUES = ['var(--ramp-1)', 'var(--ramp-2)', 'var(--ramp-3)', 'var(--ramp-4)'];
const WEEKDAYS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];
const n = (v: number) => v.toLocaleString('en-US');

/** Biggest first; past `max` rows the tail folds into one "Other" row. */
function topWithOther<T extends { label: string; value: number }>(rows: T[], max: number, other: (count: number) => string): Array<{ label: string; value: number }> {
  const sorted = [...rows].sort((a, b) => b.value - a.value);
  if (sorted.length <= max) return sorted;
  const head = sorted.slice(0, max - 1);
  const tail = sorted.slice(max - 1);
  return [...head, { label: other(tail.length), value: tail.reduce((s, r) => s + r.value, 0) }];
}

export function analyticsVolume(x: {
  channels: { key: string; label: string; followers: number | null }[];
  subs: number | null;
  growth7d: number | null;
  live: number;
  pending: { label: string; source: string }[];
  runs: { agentId: string; ok: boolean; startedAt: string }[];
  agentNames: Record<string, string>;
  /** YYYY-MM-DD, the last day of the rhythm window. */
  today: string;
  days?: number;
}): AnalyticsVolume {
  // ── Reach: every channel plus the email list ──
  // A channel with no snapshot yet (null) is unmeasured, not zero: left out.
  const channels = x.channels.flatMap((c) => (c.followers === null ? [] : [{ label: c.label, value: c.followers }]));
  if (x.subs) channels.push({ label: 'Email list', value: x.subs });
  const reach = channels.reduce((s, c) => s + c.value, 0);
  const meters = topWithOther(channels, 4, (k) => `Other channels (${k})`).map((c, i) => ({
    label: c.label,
    frac: reach > 0 ? c.value / reach : 0,
    display: `${n(c.value)} · ${reach > 0 ? Math.round((c.value / reach) * 100) : 0}%`,
    hue: HUES[i % HUES.length],
  }));

  const chips: VolumeChip[] = [];
  if (x.growth7d !== null && x.growth7d !== 0) {
    chips.push({ tone: x.growth7d > 0 ? 'ok' : 'err', text: `${x.growth7d > 0 ? '+' : ''}${x.growth7d.toFixed(1)}% 7d` });
  }
  if (x.subs) chips.push({ text: `${n(x.subs)} email subs` });

  const totalMetrics = x.live + x.pending.length;

  // ── Runs by agent (the whole log the page read) ──
  const byAgent = new Map<string, number>();
  for (const r of x.runs) byAgent.set(r.agentId, (byAgent.get(r.agentId) ?? 0) + 1);
  const ok = x.runs.filter((r) => r.ok).length;
  const total = x.runs.length;
  const agentRows = [...byAgent.entries()].map(([id, count]) => ({ label: x.agentNames[id] ?? id, value: count }));
  const runMeters = topWithOther(agentRows, 4, (k) => `Other agents (${k})`).map((a, i) => ({
    label: `${a.label} (${a.value})`,
    frac: total > 0 ? a.value / total : 0,
    display: `${total > 0 ? Math.round((a.value / total) * 100) : 0}%`,
    hue: HUES[i % HUES.length],
  }));

  // ── Weekday rhythm over the trailing window, Monday first ──
  const days = x.days ?? 30;
  const end = Date.parse(`${x.today}T00:00:00Z`);
  const start = new Date(end - (days - 1) * 86_400_000).toISOString().slice(0, 10);
  const byDay = [0, 0, 0, 0, 0, 0, 0];
  for (const r of x.runs) {
    const d = r.startedAt.slice(0, 10);
    if (d < start || d > x.today) continue;
    const dow = new Date(`${d}T00:00:00Z`).getUTCDay(); // 0 = Sunday
    byDay[(dow + 6) % 7] += 1;
  }
  const rhythm = WEEKDAYS.map((label, i) => ({ label, count: byDay[i] }));

  // ── The one insight: what still needs credentials ──
  const pendingCount = x.pending.length;

  return {
    reach,
    headline: n(reach),
    chips,
    caption: channels.length ? `total reach across ${channels.length} channel${channels.length === 1 ? '' : 's'}` : 'no audience snapshots yet',
    meters,
    foot: `reach = followers + email list · ${x.live} of ${totalMetrics} metrics live`,
    runs: {
      total,
      ok,
      failed: total - ok,
      okPct: total > 0 ? Math.round((ok / total) * 100) : null,
      agents: byAgent.size,
      meters: runMeters,
    },
    rhythm,
    rhythmTotal: byDay.reduce((s, c) => s + c, 0),
    insight: {
      value: pendingCount,
      headline: pendingCount === 0 ? 'Every metric is live' : `${pendingCount} metric${pendingCount === 1 ? '' : 's'} waiting on credentials`,
      body:
        x.pending
          .slice(0, 3)
          .map((p) => p.label)
          .join(' · ') || 'Nothing to wire. Every tile reads a real connector.',
      frac: totalMetrics > 0 ? x.live / totalMetrics : 0,
    },
  };
}
