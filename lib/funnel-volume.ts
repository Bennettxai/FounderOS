import { attentionQueue, funnelSummary, journeyMeta, FUNNEL_STAGES } from '@/lib/funnel';
import type { FunnelJourney } from '@/lib/schemas';

/**
 * /funnel below the graph in the Brand Deals look (2026-09-24). The same
 * journeys the graph draws, shaped like Deal Volume: closed revenue as the
 * headline, one meter per stage hand-off (each a real fraction of the stage
 * before), a per-day touch series for the step line, and the "needs you"
 * count for the one insight card. Pure; the page hands it the active journeys.
 */

export type FunnelMeter = { label: string; frac: number; display: string; hue: string };
export type FunnelChip = { tone?: 'ok' | 'warn' | 'err' | 'accent'; text: string };

export type FunnelVolume = {
  revenueUsd: number;
  chips: FunnelChip[];
  caption: string;
  meters: FunnelMeter[];
  foot: string;
  series: { label: string; count: number }[];
  touchesInWindow: number;
  insight: { value: number; headline: string; body: string; frac: number };
};

// Short labels for the hand-off meters; the stage ids stay FUNNEL_STAGES'.
const HANDOFF = FUNNEL_STAGES.slice(1).map((stage, i) => `${FUNNEL_STAGES[i].label} → ${stage.label}`);
const HUES = ['var(--ramp-1)', 'var(--ramp-2)', 'var(--ramp-3)', 'var(--accent)'];

const MONTH = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
const dayLabel = (iso: string) => `${MONTH[Number(iso.slice(5, 7)) - 1]} ${Number(iso.slice(8, 10))}`;
const plural = (n: number, one: string, many = `${one}s`) => `${n} ${n === 1 ? one : many}`;

export function funnelVolume({ journeys, archived, now, days = 30 }: { journeys: FunnelJourney[]; archived: number; now: Date; days?: number }): FunnelVolume {
  const summary = funnelSummary(journeys);
  const totals = summary.stages.map((s) => s.total);
  const first = summary.stages[0];

  const meters: FunnelMeter[] = FUNNEL_STAGES.slice(1).map((_, k) => {
    const prev = totals[k] ?? 0;
    const next = totals[k + 1] ?? 0;
    const frac = prev > 0 ? next / prev : 0;
    return {
      label: `${HANDOFF[k]} (${next}/${prev})`,
      frac,
      display: prev > 0 ? `${Math.round(frac * 100)}%` : 'no leads',
      hue: HUES[k],
    };
  });

  const stalled = journeys.filter((j) => journeyMeta(j, now).state === 'stalled').length;
  const chips: FunnelChip[] = [{ tone: 'ok', text: `${summary.converted} closed` }];
  if (stalled > 0) chips.push({ tone: 'err', text: `${stalled} stalled` });
  if (archived > 0) chips.push({ text: `${archived} archived` });

  // Touches per UTC day over the trailing window ending today.
  const end = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate()));
  const keys = Array.from({ length: days }, (_, i) => new Date(end.getTime() - (days - 1 - i) * 86_400_000).toISOString().slice(0, 10));
  const counts = new Map(keys.map((k) => [k, 0]));
  for (const j of journeys) for (const t of j.touches) if (counts.has(t.at)) counts.set(t.at, (counts.get(t.at) ?? 0) + 1);
  const series = keys.map((k) => ({ label: dayLabel(k), count: counts.get(k) ?? 0 }));

  const { pushNow, saveNow } = attentionQueue(journeys, now);
  const value = pushNow.length + saveNow.length;
  const endToEnd = first && first.total > 0 ? Math.round((summary.converted / first.total) * 100) : null;

  return {
    revenueUsd: summary.revenueUsd,
    chips,
    caption: `closed revenue across ${plural(journeys.length, 'active client')} · ${first?.organic ?? 0} organic / ${first?.ads ?? 0} ads entry`,
    meters,
    foot: `each bar is a stage over the one before · ${endToEnd === null ? 'no leads yet' : `${endToEnd}% end to end`}`,
    series,
    touchesInWindow: series.reduce((n, s) => n + s.count, 0),
    insight: {
      value,
      headline: value === 0 ? 'Nothing waiting on you' : `${pushNow.length} to push · ${saveNow.length} to save`,
      body:
        [...pushNow, ...saveNow]
          .slice(0, 3)
          .map((j) => j.person ?? j.name)
          .join(' · ') || 'Every lead is fresh or already closed.',
      frac: journeys.length > 0 ? value / journeys.length : 0,
    },
  };
}
