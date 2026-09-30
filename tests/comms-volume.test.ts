import { describe, expect, test } from 'vitest';
import { commsVolume } from '@/lib/comms-volume';

/**
 * /comms wears Brand Deals' Deal Volume card (Alex, 2026-09-24: "the
 * current metrics displayed on each page should be displayed very similarly
 * to how they are in the Brand Deals tab"). The view-model turns the page's
 * existing data (lanes, Slack client cards, source states, the week's
 * meetings, recordings) into a headline, chips, meters, a per-day series and
 * the one insight. Every meter is a real fraction of its own whole.
 */
const NOW = new Date('2026-09-24T15:00:00');
const day = (d: number, h = 10) => new Date(2026, 8, d, h).toISOString();

const lanes = [
  {
    name: 'Work',
    source: 'email' as const,
    unread: 3,
    items: [
      { sender: 'Acme', ts: day(24), unread: 2, priority: 1 as const },
      { sender: 'Beta', ts: day(23), unread: 1 },
      { sender: 'Gamma', ts: day(23), unread: 0, priority: 1 as const },
    ],
  },
  { name: 'Personal', source: 'email' as const, unread: 0, items: [{ sender: 'Mom', ts: day(20), unread: 0 }] },
  { name: 'WhatsApp', source: 'whatsapp' as const, unread: 1, items: [{ sender: 'Sam', ts: day(24, 9), unread: 1 }] },
];
const slackCards = [
  { name: 'Vantage', unread: 2, waiting: 'you' as const },
  { name: 'Zentra', unread: 0, waiting: 'them' as const },
  { name: 'Futurism', unread: 1, waiting: 'you' as const },
  { name: 'Quiet', unread: 0, waiting: 'none' as const },
];
const sources = [{ state: 'connected' }, { state: 'connected' }, { state: 'error' }, { state: 'not_configured' }, { state: 'connected' }];
const events = [{ start: day(24, 16) }, { start: day(25, 11) }, { start: day(25, 14) }, { start: day(30, 9) }];
const recordings = [{ at: day(22) }, { at: day(21) }];

const v = commsVolume({ lanes, slackCards, sources, events, recordings, now: NOW });

describe('commsVolume: the headline', () => {
  test('is unread across the email inboxes and WhatsApp, the same number the tabs count', () => {
    expect(v.headline).toBe(4);
  });

  test('chips carry priority unread, Slack unread and erroring sources in status colors', () => {
    expect(v.chips).toEqual([
      { tone: 'warn', text: '1 priority unread' },
      { tone: 'accent', text: '3 Slack unread' },
      { tone: 'err', text: '1 source error' },
    ]);
  });

  test('the caption names the inbox count and the threads in view', () => {
    expect(v.caption).toBe('unread across 2 inboxes + WhatsApp · 5 threads in view');
  });

  test('the meta line is real numbers only', () => {
    expect(v.meta).toBe('4 unread · 3/5 sources connected · 4 meetings next 7 days · 2 recordings');
  });
});

describe('commsVolume: the meters', () => {
  test('four meters, each a real fraction of its own whole', () => {
    expect(v.meters.map((m) => m.label)).toEqual(['Email (3)', 'WhatsApp (1)', 'Slack clients waiting on you (2/4)', 'Sources connected (3/5)']);
    expect(v.meters[0]).toMatchObject({ frac: 0.75, display: '3 unread' });
    expect(v.meters[1]).toMatchObject({ frac: 0.25, display: '1 unread' });
    expect(v.meters[2]).toMatchObject({ frac: 0.5, display: '2 of 4' });
    expect(v.meters[3]).toMatchObject({ frac: 0.6, display: '60%' });
  });

  test('one hue per domain, from the shared CSS vars only', () => {
    for (const m of v.meters) expect(m.hue).toMatch(/^var\(--(accent|ok|warn|err|ramp-[1-4]|send-activity)\)$/);
    expect(new Set(v.meters.map((m) => m.hue)).size).toBe(4);
  });

  test('the foot says what the week and the recorder hold', () => {
    expect(v.foot).toBe('4 meetings next 7 days · 2 recent recordings');
  });
});

describe('commsVolume: the series and the insight', () => {
  test('messages per local day, oldest first, ending today', () => {
    expect(v.series).toHaveLength(14);
    expect(v.series.at(-1)).toEqual({ label: 'Sep 24', count: 2 });
    expect(v.series.at(-2)).toEqual({ label: 'Sep 23', count: 2 });
    expect(v.series.find((s) => s.label === 'Sep 20')?.count).toBe(1);
    expect(v.seriesTotal).toBe(5);
  });

  test('meetings per day for the next 7 days, today first', () => {
    expect(v.meetings.map((m) => m.label)).toEqual(['Thu', 'Fri', 'Sat', 'Sun', 'Mon', 'Tue', 'Wed']);
    expect(v.meetings.map((m) => m.count)).toEqual([1, 2, 0, 0, 0, 0, 1]);
    expect(v.meetingsTotal).toBe(4);
  });

  test('the insight is who is waiting on Alex: Slack clients plus priority unread', () => {
    expect(v.insight.value).toBe(3);
    expect(v.insight.headline).toBe('2 client threads waiting on you · 1 priority unread.');
    expect(v.insight.body).toBe('Vantage · Futurism · Acme');
    // 3 of the 6 things that could need him (4 client cards + 2 priority threads)
    expect(v.insight.frac).toBeCloseTo(0.5);
  });
});

describe('commsVolume: nothing to show', () => {
  const e = commsVolume({ lanes: [], slackCards: [], sources: [], events: [], recordings: [], now: NOW });

  test('empty data gives empty meters and honest copy, never fabricated numbers', () => {
    expect(e.headline).toBe(0);
    expect(e.chips).toEqual([]);
    expect(e.meters.every((m) => m.frac === 0)).toBe(true);
    expect(e.meters[2].display).toBe('no clients linked');
    expect(e.caption).toBe('no inboxes configured yet');
    expect(e.seriesTotal).toBe(0);
    expect(e.meetingsTotal).toBe(0);
    expect(e.insight).toMatchObject({ value: 0, frac: 0 });
    expect(e.insight.body).toMatch(/Nobody is waiting on you/);
  });
});
