import type { Meter, Tone } from '@/components/slab';
import type { SeriesPoint } from '@/components/slab-charts';
import { newsletterSummary, type Newsletter } from '@/lib/newsletters';

/**
 * /social/beehiiv in the Brand Deals look (Alex, 2026-09-24). One pure
 * pass over the page's own rows (the Beehiiv subscriber count and the past
 * issues) gives the Newsletter Volume card, the sends step line, the
 * open-rate dot matrix and the one "Best open" card. The funnel meters are
 * ratios of SUMMED send stats, so a big send weighs more than a small one;
 * averaging per-issue percentages would not be a fraction of a real whole.
 * Seeded issues (id `seed-*`) are labelled as a preview, never passed off.
 */
export type NewsletterVolume = {
  headline: number | null;
  chips: Array<{ tone?: Tone; text: string }>;
  caption: string;
  meters: Meter[];
  foot: string;
  sends: SeriesPoint[];
  openRates: SeriesPoint[];
  /** mean of per-issue open rates (Beehiiv's own figure), null before the first send */
  avgOpenRate: number | null;
  totalRecipients: number;
  totalClicks: number;
  unsubscribes: number;
  spamReports: number;
  insight: { display: string; headline: string; body: string; frac: number };
};

const MATRIX_COLS = 6;
const clamp = (f: number) => (Number.isFinite(f) ? Math.max(0, Math.min(1, f)) : 0);
const pct = (f: number, digits = 1) => `${(f * 100).toFixed(digits)}%`;
const shortDate = (iso: string) => new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric', timeZone: 'UTC' });
const sum = (list: Newsletter[], k: keyof Pick<Newsletter, 'recipients' | 'delivered' | 'opens' | 'clicks' | 'unsubscribes' | 'spamReports'>) =>
  list.reduce((n, x) => n + x[k], 0);

export function newsletterVolume(x: { newsletters: Newsletter[]; subscribers: number | null }): NewsletterVolume {
  const list = x.newsletters;
  const count = list.length;
  const summary = newsletterSummary(list);
  const oldestFirst = [...list].sort((a, b) => a.publishedAt.localeCompare(b.publishedAt));
  const seeded = count > 0 && list.every((n) => n.id.startsWith('seed-'));

  const recipients = sum(list, 'recipients');
  const delivered = sum(list, 'delivered');
  const opens = sum(list, 'opens');
  const clicks = sum(list, 'clicks');
  const unsubscribes = sum(list, 'unsubscribes');
  const spamReports = sum(list, 'spamReports');

  const chips: NewsletterVolume['chips'] = [];
  if (count) {
    chips.push({ tone: 'accent', text: `${count} issue${count === 1 ? '' : 's'}` });
    chips.push({ tone: 'ok', text: `${summary.avgOpenRate.toFixed(1)}% avg open` });
  }

  const meters: Meter[] = [];
  if (recipients > 0) meters.push({ label: 'Delivered', frac: clamp(delivered / recipients), display: `${pct(delivered / recipients)} of sends`, hue: 'var(--ok)' });
  if (delivered > 0) meters.push({ label: 'Opened', frac: clamp(opens / delivered), display: `${pct(opens / delivered)} of delivered`, hue: 'var(--ramp-1)' });
  if (opens > 0) meters.push({ label: 'Clicked', frac: clamp(clicks / opens), display: `${pct(clicks / opens)} of opens`, hue: 'var(--ramp-3)' });
  if (delivered > 0) meters.push({ label: 'Unsubscribed', frac: clamp(unsubscribes / delivered), display: `${pct(unsubscribes / delivered, 2)} of delivered`, hue: 'var(--warn)' });

  const foot = count
    ? `${count} issue${count === 1 ? '' : 's'} · ${recipients.toLocaleString('en-US')} sends${seeded ? ' · seeded preview' : ''}`
    : 'no issues yet';

  const best = list.reduce<Newsletter | null>((b, n) => (b && b.openRate >= n.openRate ? b : n), null);
  const insight: NewsletterVolume['insight'] = best
    ? {
        display: `${best.openRate.toFixed(1)}%`,
        headline: `Best open rate: ${best.title}.`,
        body: `${summary.avgOpenRate.toFixed(1)}% average across ${count} issue${count === 1 ? '' : 's'} · ${shortDate(best.publishedAt)}`,
        frac: clamp(best.openRate / 100),
      }
    : { display: 'none', headline: 'No issues sent yet.', body: 'Open rates show here after the first send.', frac: 0 };

  return {
    headline: x.subscribers,
    chips,
    caption: x.subscribers != null ? 'subscribers, live via Beehiiv' : 'no live subscriber count · add BEEHIIV_API_KEY',
    meters,
    foot,
    sends: oldestFirst.map((n) => ({ label: shortDate(n.publishedAt), count: n.recipients })),
    openRates: oldestFirst.slice(-MATRIX_COLS).map((n) => ({ label: shortDate(n.publishedAt), count: n.openRate })),
    avgOpenRate: count ? summary.avgOpenRate : null,
    totalRecipients: recipients,
    totalClicks: clicks,
    unsubscribes,
    spamReports,
    insight,
  };
}
