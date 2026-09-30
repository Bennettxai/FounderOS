import { describe, expect, test } from 'vitest';
import { filterLeadMagnets, leadMagnetVolume, LEAD_MAGNET_FILTERS, type LeadMagnetVolumeRow } from '@/lib/lead-magnet-volume';

/**
 * /content/lead-magnets in the Brand Deals look (Alex, 2026-09-24). The
 * Magnet Volume card, the pages-shipped step line, the capture and
 * destination dot matrices and the one "Since last launch" card, all from
 * the lead_magnets rows the table below renders. Nothing invented: no pages
 * means empty meters and honest copy.
 */
const TODAY = '2026-09-24';
const ROWS: LeadMagnetVolumeRow[] = [
  { name: 'Agent Stack', status: 'live', captures: 'email', destination: 'Beehiiv · newsletter + Cohort 2', source: 'IG reel', launchedAt: '2026-08-12' },
  { name: 'Call Kit', status: 'live', captures: 'booking', destination: 'Calendar · discovery', source: 'TikTok', launchedAt: '2026-09-20' },
  { name: 'Offer Doc', status: 'draft', captures: 'none', destination: 'Beehiiv', source: 'IG reel', launchedAt: '2026-09-01' },
  { name: 'Old Page', status: 'archived', captures: 'email', destination: 'Beehiiv · list', source: 'X thread', launchedAt: '2026-03-01' },
];

describe('leadMagnetVolume: the Magnet Volume card', () => {
  const v = leadMagnetVolume({ rows: ROWS, today: TODAY });

  test('headline is pages live, chips name what is not', () => {
    expect(v.headline).toBe(2);
    expect(v.total).toBe(4);
    expect(v.counts).toEqual({ live: 2, draft: 1, paused: 0, archived: 1 });
    expect(v.chips).toEqual([{ tone: 'warn', text: '1 draft' }, { text: '1 archived' }]);
    expect(v.caption).toBe('of 4 landing pages shipped');
  });

  test('meters: the live share and what each page captures, all of the same whole', () => {
    expect(v.meters).toEqual([
      { label: 'Live (2/4)', frac: 0.5, display: '2 of 4', hue: 'var(--ok)' },
      { label: 'Capturing email (2/4)', frac: 0.5, display: '2 of 4', hue: 'var(--ramp-1)' },
      { label: 'Capturing bookings (1/4)', frac: 0.25, display: '1 of 4', hue: 'var(--ramp-3)' },
      { label: 'No capture yet (1/4)', frac: 0.25, display: '1 of 4', hue: 'var(--warn)' },
    ]);
    expect(v.foot).toBe('2 destinations · 3 campaigns');
  });

  test('pages shipped: a cumulative weekly line over 12 weeks, ending today', () => {
    expect(v.series).toHaveLength(12);
    expect(v.series.at(-1)).toEqual({ label: 'Sep 24', count: 4 });
    expect(v.series.at(-2)).toEqual({ label: 'Sep 17', count: 3 });
    expect(v.series[0]).toEqual({ label: 'Jul 9', count: 1 });
    expect(v.shippedInWindow).toBe(3);
  });

  test('captures and destinations as dot-matrix columns, destinations by short name', () => {
    expect(v.captures).toEqual([
      { label: 'Email', count: 2 },
      { label: 'Booking', count: 1 },
      { label: 'None', count: 1 },
    ]);
    expect(v.destinations).toEqual([
      { label: 'Beehiiv', count: 3 },
      { label: 'Calendar', count: 1 },
    ]);
  });

  test('the one insight card is how long since the newest page shipped', () => {
    expect(v.insight).toEqual({
      value: 4,
      headline: '4 days since Call Kit shipped.',
      body: '2 of 4 live · ticks light with the live share.',
      frac: 0.5,
    });
  });

  test('a page launched today reads as today, a future date is not the newest', () => {
    const t = leadMagnetVolume({
      rows: [
        { ...ROWS[0], launchedAt: TODAY },
        { ...ROWS[1], name: 'Soon', launchedAt: '2026-10-01' },
      ],
      today: TODAY,
    });
    expect(t.insight.value).toBe(0);
    expect(t.insight.headline).toBe('Agent Stack went live today.');
  });
});

describe('leadMagnetVolume with nothing shipped', () => {
  const e = leadMagnetVolume({ rows: [], today: TODAY });

  test('empty meters, flat line, honest copy', () => {
    expect(e.headline).toBe(0);
    expect(e.chips).toEqual([]);
    expect(e.meters).toEqual([]);
    expect(e.caption).toBe('no landing pages recorded yet');
    expect(e.destinations).toEqual([]);
    expect(e.series.every((s) => s.count === 0)).toBe(true);
    expect(e.insight).toMatchObject({ display: 'none', headline: 'Nothing shipped yet.', frac: 0 });
  });
});

describe('filterLeadMagnets: the status pills over the table', () => {
  test('filters by a known status, anything else is all', () => {
    expect(LEAD_MAGNET_FILTERS).toEqual(['all', 'live', 'draft', 'paused', 'archived']);
    expect(filterLeadMagnets(ROWS, 'live').map((r) => r.name)).toEqual(['Agent Stack', 'Call Kit']);
    expect(filterLeadMagnets(ROWS, 'draft')).toHaveLength(1);
    expect(filterLeadMagnets(ROWS, undefined)).toHaveLength(4);
    expect(filterLeadMagnets(ROWS, 'bogus')).toHaveLength(4);
    expect(filterLeadMagnets(ROWS, ['live', 'draft'])).toHaveLength(2);
  });
});
