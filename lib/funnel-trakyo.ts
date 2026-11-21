/**
 * Trakyo seam: organic content attribution for the funnel (which post or
 * reel actually produced a lead). `trakyoTouches()`
 * reads Trakyo API v1 `GET /v1/leads` (lib/connectors/trakyo.ts) and turns each
 * lead's first touch into one attributed event (FounderOS merge F7). No key:
 * no request and no events. A failed read also yields no events, so the funnel
 * keeps its CRM touches rather than a guessed attribution.
 */
import { trakyoKey, trakyoLeads, type TrakyoLead, type TrakyoTouchType } from '@/lib/connectors/trakyo';
import { LIVE_READ, keyTag, swr } from '@/lib/swr';
import type { FunnelJourney } from '@/lib/schemas';

/** One attributed content event as Trakyo will report it. */
export type TrakyoEvent = {
  lead: string; // lead name as Trakyo captured it
  label: string; // the content piece, e.g. 'IG reel: "3 AI offers…"'
  channel: 'organic' | 'ads';
  at: string; // YYYY-MM-DD
};

const norm = (s: string) => s.trim().toLowerCase().replace(/\s+/g, ' ');

/**
 * Swap a journey's synthetic first touch for the real Trakyo-attributed
 * content touch (matched by normalized lead name). Everything else — stage
 * history, stall math, the node's transit — is untouched.
 */
export function mergeTrakyoTouches(journeys: FunnelJourney[], events: TrakyoEvent[]): FunnelJourney[] {
  if (events.length === 0) return journeys;
  const byLead = new Map(events.map((e) => [norm(e.lead), e]));
  return journeys.map((j) => {
    const event = byLead.get(norm(j.name));
    if (!event || j.touches.length === 0) return j;
    const [first, ...rest] = j.touches;
    return {
      ...j,
      touches: [
        { ...first, channel: event.channel, label: event.label, source: 'trakyo', at: event.at },
        ...rest,
      ],
    };
  });
}

const TYPE_LABEL: Partial<Record<TrakyoTouchType, string>> = { youtube: 'YouTube', instagram: 'Instagram', meta_ad: 'Meta ad', referrer: 'Referrer' };

/**
 * One lead's first touch as a funnel event. The label leads with the source
 * (`YouTube: How I…`) so lib/funnel-channels.ts can name the channel from it.
 * A lead with no name, no email or no dated first touch yields nothing.
 */
export function leadToEvent(lead: TrakyoLead): TrakyoEvent | null {
  const t = lead.firstTouch;
  const who = lead.names[0] ?? lead.emails[0];
  if (!t || !who) return null;
  const at = t.occurredAt.slice(0, 10);
  if (!/^\d{4}-\d{2}-\d{2}$/.test(at)) return null;
  const source = t.sourceName ?? TYPE_LABEL[t.type] ?? null;
  const name = t.name || source || t.type;
  return {
    lead: who,
    label: source && source !== name ? `${source}: ${name}` : name,
    channel: t.type === 'meta_ad' ? 'ads' : 'organic',
    at,
  };
}

/**
 * The live pull: the newest 200 leads (two pages), cached like the CRM pulls
 * (60 s fresh, 10 min stale-while-revalidate) so a funnel render never waits
 * on Trakyo twice. Every event is a real first touch; nothing is inferred.
 */
export async function trakyoTouches(): Promise<TrakyoEvent[]> {
  const key = trakyoKey();
  if (!key) return [];
  try {
    const { leads } = await swr(`trakyo:leads:${keyTag(key)}`, LIVE_READ, () => trakyoLeads({ max: 200 }));
    return leads.map(leadToEvent).filter((e): e is TrakyoEvent => e !== null);
  } catch {
    return [];
  }
}
