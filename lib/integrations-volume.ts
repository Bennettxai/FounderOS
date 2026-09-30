import type { Meter, Tone } from '@/components/slab';
import type { SeriesPoint } from '@/components/slab-charts';
import type { ConnectorStatus } from '@/lib/connectors/types';
import { connectKeysFor, type CatalogEntry } from '@/lib/integrations-catalog';
import { INTEGRATION_CATEGORIES } from '@/lib/schemas';

/**
 * /integrations in the Brand Deals look (Alex, 2026-09-24). One pure pass
 * over the page's own rows (the live connector checks and the catalog merged
 * onto them) produces every number the slab shows: the Connection Volume
 * headline, chips and meters, connected tools by category, the connector
 * health columns and the single "Needs you" insight. A stored key is never
 * counted as connected; only a connector that answered is.
 */
export type IntegrationsVolume = {
  headline: number;
  counts: { connected: number; notConfigured: number; error: number; total: number };
  chips: Array<{ tone?: Tone; text: string }>;
  caption: string;
  meters: Meter[];
  foot: string;
  byCategory: SeriesPoint[];
  topCategory: { name: string; count: number } | null;
  health: SeriesPoint[];
  insight: { value: number; headline: string; body: string; frac: number };
};

const frac = (n: number, d: number) => (d > 0 ? Math.max(0, Math.min(1, n / d)) : 0);
const pct = (f: number) => `${Math.round(f * 100)}%`;
const plural = (n: number, word: string) => `${n} ${word}${n === 1 ? '' : 's'}`;

/** Dot-matrix column labels: short enough that five sit beside a 30px stat. */
const SHORT: Record<string, string> = {
  Productivity: 'Prod',
  Communication: 'Comm',
  'CRM & Sales': 'CRM',
  Developer: 'Dev',
  Scheduling: 'Sched',
  Finance: 'Fin',
  Marketing: 'Mkt',
  Storage: 'Store',
  Knowledge: 'Know',
  'AI & Automation': 'AI',
  Creative: 'Create',
};

export function integrationsVolume(x: {
  statuses: Pick<ConnectorStatus, 'id' | 'name' | 'state'>[];
  catalog: CatalogEntry[];
}): IntegrationsVolume {
  const total = x.statuses.length;
  const live = x.statuses.filter((s) => s.state === 'connected');
  const erroring = x.statuses.filter((s) => s.state === 'error');
  const counts = { connected: live.length, notConfigured: total - live.length - erroring.length, error: erroring.length, total };

  const chips: IntegrationsVolume['chips'] = [];
  if (counts.connected) chips.push({ tone: 'ok', text: `${counts.connected} live` });
  if (counts.notConfigured) chips.push({ text: `${counts.notConfigured} not configured` });
  if (counts.error) chips.push({ tone: 'err', text: `${counts.error} erroring` });

  const tools = x.catalog.length;
  const connectedTools = x.catalog.filter((c) => c.connected).length;
  const keyed = x.catalog.filter((c) => connectKeysFor(c).length > 0);
  const saved = keyed.filter((c) => c.keySaved).length;
  const popular = x.catalog.filter((c) => c.popular);
  const popularLive = popular.filter((c) => c.connected).length;
  const savedNotLive = x.catalog.filter((c) => c.keySaved && !c.connected).length;

  const meters: Meter[] = [];
  if (total > 0) {
    const f = frac(counts.connected, total);
    meters.push({ label: `Connectors live (${counts.connected}/${total})`, frac: f, display: pct(f), hue: 'var(--ok)' });
  }
  if (tools > 0) {
    const f = frac(connectedTools, tools);
    meters.push({ label: `Catalog tools connected (${connectedTools}/${tools})`, frac: f, display: pct(f), hue: 'var(--accent)' });
  }
  if (keyed.length > 0) {
    const f = frac(saved, keyed.length);
    meters.push({ label: `Keys saved (${saved}/${keyed.length})`, frac: f, display: pct(f), hue: 'var(--ramp-1)' });
  }
  if (popular.length > 0) {
    const f = frac(popularLive, popular.length);
    meters.push({ label: `Popular connected (${popularLive}/${popular.length})`, frac: f, display: pct(f), hue: 'var(--ramp-4)' });
  }

  // Connected tools per category, busiest first; ties keep catalog order.
  const cats = INTEGRATION_CATEGORIES.filter((cat) => x.catalog.some((c) => c.category === cat)).map((cat, order) => ({
    name: cat as string,
    order,
    count: x.catalog.filter((c) => c.category === cat && c.connected).length,
  }));
  const ranked = [...cats].sort((a, b) => b.count - a.count || a.order - b.order);
  const byCategory = ranked.slice(0, 5).map((c) => ({ label: SHORT[c.name] ?? c.name, count: c.count }));
  const topCategory = ranked[0] && ranked[0].count > 0 ? { name: ranked[0].name, count: ranked[0].count } : null;

  const insight = {
    value: counts.error,
    headline: counts.error ? `${plural(counts.error, 'connector')} erroring.` : 'No connector is erroring.',
    body: counts.error
      ? erroring
          .slice(0, 3)
          .map((s) => s.name)
          .join(' · ')
      : savedNotLive
        ? `${plural(savedNotLive, 'saved key')} not live yet.`
        : total
          ? 'Every configured connector answers.'
          : 'No connector checks ran.',
    frac: frac(counts.error, total),
  };

  return {
    headline: counts.connected,
    counts,
    chips,
    caption: `of ${plural(total, 'connector check')} · live status, never a stored key alone`,
    meters,
    foot: `${plural(tools, 'tool')} · ${cats.length} ${cats.length === 1 ? 'category' : 'categories'}${savedNotLive ? ` · ${plural(savedNotLive, 'saved key')} not live` : ''}`,
    byCategory,
    topCategory,
    health: [
      { label: 'live', count: counts.connected },
      { label: 'unset', count: counts.notConfigured },
      { label: 'error', count: counts.error },
    ],
    insight,
  };
}
