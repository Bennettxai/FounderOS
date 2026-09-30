import type { Meter, Tone } from '@/components/slab';
import type { SeriesPoint } from '@/components/slab-charts';
import type { LeadMagnet } from '@/lib/schemas';

/**
 * /content/lead-magnets in the Brand Deals look (Alex, 2026-09-24). One
 * pure pass over the lead_magnets rows the table renders gives the Magnet
 * Volume card (live pages, the live share and what each page captures, all
 * of the same whole), a cumulative pages-shipped line, the capture and
 * destination dot matrices and the one "Since last launch" card. Launch
 * dates are compared as UTC YYYY-MM-DD strings; the page passes `today`.
 */
export type LeadMagnetVolumeRow = Pick<LeadMagnet, 'name' | 'status' | 'captures' | 'destination' | 'source' | 'launchedAt'>;

export type LeadMagnetVolume = {
  headline: number;
  total: number;
  counts: Record<LeadMagnet['status'], number>;
  chips: Array<{ tone?: Tone; text: string }>;
  caption: string;
  meters: Meter[];
  foot: string;
  series: SeriesPoint[];
  shippedInWindow: number;
  captures: SeriesPoint[];
  destinations: SeriesPoint[];
  insight: { value: number; display?: string; headline: string; body: string; frac: number };
};

export const LEAD_MAGNET_FILTERS = ['all', 'live', 'draft', 'paused', 'archived'] as const;
export type LeadMagnetFilter = (typeof LEAD_MAGNET_FILTERS)[number];

/** The status the pills select; anything unknown (or absent) means all. */
export function leadMagnetFilter(param: string | string[] | undefined): LeadMagnetFilter {
  const v = Array.isArray(param) ? param[0] : param;
  return (LEAD_MAGNET_FILTERS as readonly string[]).includes(v ?? '') ? (v as LeadMagnetFilter) : 'all';
}

export function filterLeadMagnets<T extends Pick<LeadMagnet, 'status'>>(rows: T[], param: string | string[] | undefined): T[] {
  const f = leadMagnetFilter(param);
  return f === 'all' ? rows : rows.filter((r) => r.status === f);
}

const DAY_MS = 86_400_000;
const DEST_COLS = 6;
const plural = (n: number, one: string, many = `${one}s`) => `${n} ${n === 1 ? one : many}`;
const utcLabel = (d: Date) => d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', timeZone: 'UTC' });
/** "Beehiiv · newsletter + Cohort 2" reads as "Beehiiv" under a dot column. */
const shortDestination = (d: string) => {
  const head = d.split(/\s*[·|(]\s*/)[0].trim() || 'Other';
  return head.length > 12 ? head.split(/\s+/)[0] : head;
};
/** launchedAt is at least a year; pad it to a comparable UTC day. */
const dayKey = (s: string) => {
  const k = s.slice(0, 10);
  return k.length === 4 ? `${k}-01-01` : k.length === 7 ? `${k}-01` : k;
};

export function leadMagnetVolume(x: { rows: LeadMagnetVolumeRow[]; today: string; weeks?: number }): LeadMagnetVolume {
  const weeks = x.weeks ?? 12;
  const rows = x.rows;
  const total = rows.length;
  const count = (s: LeadMagnet['status']) => rows.filter((r) => r.status === s).length;
  const counts = { live: count('live'), draft: count('draft'), paused: count('paused'), archived: count('archived') };
  const cap = (c: LeadMagnet['captures']) => rows.filter((r) => r.captures === c).length;
  const email = cap('email');
  const booking = cap('booking');
  const none = cap('none');

  const chips: LeadMagnetVolume['chips'] = [];
  if (counts.draft) chips.push({ tone: 'warn', text: `${counts.draft} draft` });
  if (counts.paused) chips.push({ tone: 'err', text: `${counts.paused} paused` });
  if (counts.archived) chips.push({ text: `${counts.archived} archived` });

  const meters: Meter[] = [];
  if (total > 0) {
    const m = (label: string, n: number, hue: string): Meter => ({ label: `${label} (${n}/${total})`, frac: n / total, display: `${n} of ${total}`, hue });
    meters.push(m('Live', counts.live, 'var(--ok)'));
    meters.push(m('Capturing email', email, 'var(--ramp-1)'));
    meters.push(m('Capturing bookings', booking, 'var(--ramp-3)'));
    meters.push(m('No capture yet', none, 'var(--warn)'));
  }

  const destMap = new Map<string, number>();
  for (const r of rows) {
    const k = shortDestination(r.destination);
    destMap.set(k, (destMap.get(k) ?? 0) + 1);
  }
  const destinations = [...destMap.entries()]
    .sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))
    .slice(0, DEST_COLS)
    .map(([label, n]) => ({ label, count: n }));
  const campaigns = new Set(rows.map((r) => r.source.trim()).filter(Boolean)).size;

  // Pages on record at the end of each week, oldest first, the last week ending today.
  const end = new Date(`${x.today}T00:00:00Z`);
  const keys = rows.map((r) => dayKey(r.launchedAt));
  const series: SeriesPoint[] = [];
  for (let i = weeks - 1; i >= 0; i--) {
    const d = new Date(end.getTime() - i * 7 * DAY_MS);
    const k = d.toISOString().slice(0, 10);
    series.push({ label: utcLabel(d), count: keys.filter((l) => l <= k).length });
  }
  const windowStart = new Date(end.getTime() - weeks * 7 * DAY_MS).toISOString().slice(0, 10);
  const shippedInWindow = keys.filter((k) => k > windowStart && k <= x.today).length;

  // The one gradient card: how long since the newest page went out.
  const newest = rows
    .map((r) => ({ name: r.name, key: dayKey(r.launchedAt) }))
    .filter((r) => r.key <= x.today && Number.isFinite(new Date(`${r.key}T00:00:00Z`).getTime()))
    .sort((a, b) => b.key.localeCompare(a.key))[0];
  let insight: LeadMagnetVolume['insight'];
  if (!newest) {
    insight = { value: 0, display: 'none', headline: 'Nothing shipped yet.', body: 'The next landing page lands here.', frac: 0 };
  } else {
    const quiet = Math.max(0, Math.round((end.getTime() - new Date(`${newest.key}T00:00:00Z`).getTime()) / DAY_MS));
    insight = {
      value: quiet,
      headline: quiet === 0 ? `${newest.name} went live today.` : `${plural(quiet, 'day')} since ${newest.name} shipped.`,
      body: `${counts.live} of ${total} live · ticks light with the live share.`,
      frac: total > 0 ? counts.live / total : 0,
    };
  }

  return {
    headline: counts.live,
    total,
    counts,
    chips,
    caption: total > 0 ? `of ${plural(total, 'landing page')} shipped` : 'no landing pages recorded yet',
    meters,
    foot: `${plural(destMap.size, 'destination')} · ${plural(campaigns, 'campaign')}`,
    series,
    shippedInWindow,
    captures: [
      { label: 'Email', count: email },
      { label: 'Booking', count: booking },
      { label: 'None', count: none },
    ],
    destinations,
    insight,
  };
}
