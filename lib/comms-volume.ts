/**
 * /comms in the Brand Deals look (2026-09-24): the page's own numbers shaped
 * like Deal Volume. Pure and client-safe (no connector imports), so the page
 * stays a thin renderer and the shape is tested. Nothing here invents a
 * number: every meter is a real fraction of its own whole, and empty inputs
 * read as empty meters with copy that says so.
 */

export type CommsVolumeTone = 'ok' | 'warn' | 'err' | 'accent';
export type CommsMeter = { label: string; frac: number; display: string; hue: string };
export type CommsPoint = { label: string; count: number };

export type CommsVolumeInput = {
  lanes: Array<{
    name: string;
    source: 'email' | 'whatsapp';
    unread: number;
    items: Array<{ sender: string; ts: string; unread: number; priority?: 1 | 2 | 3 }>;
  }>;
  slackCards: Array<{ name: string; unread: number; waiting: 'you' | 'them' | 'none' }>;
  sources: Array<{ state: string }>;
  events: Array<{ start: string }>;
  recordings: Array<{ at: string }>;
  now?: Date;
  /** Days of message activity in the step line. */
  days?: number;
};

export type CommsVolume = {
  headline: number;
  chips: Array<{ tone: CommsVolumeTone; text: string }>;
  caption: string;
  meta: string;
  meters: CommsMeter[];
  foot: string;
  series: CommsPoint[];
  seriesTotal: number;
  meetings: CommsPoint[];
  meetingsTotal: number;
  insight: { value: number; headline: string; body: string; frac: number };
};

const plural = (n: number, word: string, many = `${word}s`) => `${n} ${n === 1 ? word : many}`;
const frac = (n: number, d: number) => (d > 0 ? n / d : 0);

/** Local calendar day, the day Alex lived. */
function localKey(t: number): string {
  const d = new Date(t);
  return `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`;
}
function localMidnight(now: Date, offsetDays: number): Date {
  return new Date(now.getFullYear(), now.getMonth(), now.getDate() + offsetDays);
}

function perDay(stamps: string[], start: Date, days: number, label: (d: Date) => string): CommsPoint[] {
  const counts = new Map<string, number>();
  for (const s of stamps) {
    const t = Date.parse(s);
    if (!Number.isFinite(t)) continue;
    const k = localKey(t);
    counts.set(k, (counts.get(k) ?? 0) + 1);
  }
  return Array.from({ length: days }, (_, i) => {
    const d = new Date(start.getFullYear(), start.getMonth(), start.getDate() + i);
    return { label: label(d), count: counts.get(localKey(d.getTime())) ?? 0 };
  });
}

export function commsVolume(x: CommsVolumeInput): CommsVolume {
  const now = x.now ?? new Date();
  const days = x.days ?? 14;
  const emailLanes = x.lanes.filter((l) => l.source === 'email');
  const emailUnread = emailLanes.reduce((n, l) => n + l.unread, 0);
  const waUnread = x.lanes.filter((l) => l.source === 'whatsapp').reduce((n, l) => n + l.unread, 0);
  const headline = emailUnread + waUnread;
  const items = x.lanes.flatMap((l) => l.items);
  const urgentItems = items.filter((i) => i.priority === 1 && i.unread > 0);
  const priorityItems = items.filter((i) => i.priority === 1).length;
  const slackUnread = x.slackCards.reduce((n, c) => n + c.unread, 0);
  const waitingCards = x.slackCards.filter((c) => c.waiting === 'you');
  const connected = x.sources.filter((s) => s.state === 'connected').length;
  const errored = x.sources.filter((s) => s.state === 'error').length;

  const chips: CommsVolume['chips'] = [];
  if (urgentItems.length > 0) chips.push({ tone: 'warn', text: `${urgentItems.length} priority unread` });
  if (slackUnread > 0) chips.push({ tone: 'accent', text: `${slackUnread} Slack unread` });
  if (errored > 0) chips.push({ tone: 'err', text: plural(errored, 'source error') });

  const caption =
    x.lanes.length === 0
      ? 'no inboxes configured yet'
      : `unread across ${plural(emailLanes.length, 'inbox', 'inboxes')}${x.lanes.length > emailLanes.length ? ' + WhatsApp' : ''} · ${plural(items.length, 'thread')} in view`;

  const today = localMidnight(now, 0);
  const series = perDay(
    items.map((i) => i.ts).filter((ts) => Date.parse(ts) <= now.getTime()),
    localMidnight(now, -(days - 1)),
    days,
    (d) => d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
  );
  const meetings = perDay(
    x.events.map((e) => e.start),
    today,
    7,
    (d) => d.toLocaleDateString('en-US', { weekday: 'short' }),
  );
  const meetingsTotal = meetings.reduce((n, m) => n + m.count, 0);
  const recs = x.recordings.length;

  const meters: CommsMeter[] = [
    { label: `Email (${emailUnread})`, frac: frac(emailUnread, headline), display: `${emailUnread} unread`, hue: 'var(--ramp-1)' },
    { label: `WhatsApp (${waUnread})`, frac: frac(waUnread, headline), display: `${waUnread} unread`, hue: 'var(--ramp-3)' },
    {
      label: `Slack clients waiting on you (${waitingCards.length}/${x.slackCards.length})`,
      frac: frac(waitingCards.length, x.slackCards.length),
      display: x.slackCards.length ? `${waitingCards.length} of ${x.slackCards.length}` : 'no clients linked',
      hue: 'var(--warn)',
    },
    {
      label: `Sources connected (${connected}/${x.sources.length})`,
      frac: frac(connected, x.sources.length),
      display: `${Math.round(frac(connected, x.sources.length) * 100)}%`,
      hue: 'var(--accent)',
    },
  ];

  const waitingNames = [...new Set([...waitingCards.map((c) => c.name), ...urgentItems.map((i) => i.sender)])];
  const value = waitingCards.length + urgentItems.length;

  return {
    headline,
    chips,
    caption,
    meta: `${headline} unread · ${connected}/${x.sources.length} sources connected · ${plural(meetingsTotal, 'meeting')} next 7 days · ${plural(recs, 'recording')}`,
    meters,
    foot: `${plural(meetingsTotal, 'meeting')} next 7 days · ${plural(recs, 'recent recording')}`,
    series,
    seriesTotal: series.reduce((n, s) => n + s.count, 0),
    meetings,
    meetingsTotal,
    insight: {
      value,
      headline: `${plural(waitingCards.length, 'client thread')} waiting on you · ${urgentItems.length} priority unread.`,
      body: waitingNames.slice(0, 3).join(' · ') || 'Nobody is waiting on you. The threads that matter are answered.',
      frac: frac(value, x.slackCards.length + priorityItems),
    },
  };
}
