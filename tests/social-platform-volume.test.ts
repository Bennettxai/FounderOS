import { describe, expect, test } from 'vitest';
import { platformVolume } from '@/lib/social-volume';

/**
 * /social/[platform] in the Brand Deals look (Alex, 2026-09-24). The page
 * you land on from a /social account tile keeps the slab: a Follower Volume
 * card, a daily-gains step line and the one "30-day change" card, all shaped
 * from that platform's own snapshots and growth windows. No invented numbers:
 * one snapshot is not a trend, and no snapshots is an empty card.
 */
const SNAPS = [
  { capturedAt: '2026-06-01', followers: 1500, source: 'zernio-config' }, // the all-time peak
  { capturedAt: '2026-08-20', followers: 900, source: 'zernio-config' }, // before the window
  { capturedAt: '2026-08-26', followers: 1000, source: 'zernio-config' }, // +100, first day in window
  { capturedAt: '2026-09-10', followers: 1100, source: 'zernio-config' }, // +100
  { capturedAt: '2026-09-20', followers: 1050, source: 'zernio-config' }, // -50
  { capturedAt: '2026-09-23', followers: 1050, source: 'zernio-config' }, // flat
  { capturedAt: '2026-09-24', followers: 1250, source: 'zernio-live' }, // +200
];

const GROWTH = { d7: 1.234, d30: -2.5, d60: 4, allTime: -16.7 };

describe('platformVolume: the Follower Volume card', () => {
  const v = platformVolume({ label: 'Instagram', followers: 1250, growth: GROWTH, snapshots: SNAPS, today: '2026-09-24' });

  test('headline is the current follower count, chips carry the 7d and 30d growth', () => {
    expect(v.headline).toBe(1250);
    expect(v.chips).toEqual([
      { tone: 'ok', text: '+1.23% 7d' },
      { tone: 'err', text: '-2.50% 30d' },
    ]);
    expect(v.caption).toBe('Instagram followers · 7 snapshots since Jun 1');
  });

  test('meters are honest fractions: days up, days down, days tracked, share of the peak', () => {
    expect(v.meters).toEqual([
      { label: 'Days gained (3/5)', frac: 0.6, display: '3 of 5', hue: 'var(--ok)' },
      { label: 'Days dipped (1/5)', frac: 0.2, display: '1 of 5', hue: 'var(--err)' },
      { label: 'Tracked days (5/30)', frac: 5 / 30, display: '17%', hue: 'var(--ramp-1)' },
      { label: 'Of all-time peak', frac: 1250 / 1500, display: '1,250 of 1,500', hue: 'var(--accent)' },
    ]);
    expect(v.foot).toBe('7 snapshots · latest Sep 24 · zernio-live');
  });

  test('the step line is followers gained per day over 30 days, dips sit at zero', () => {
    expect(v.series).toHaveLength(30);
    expect(v.series[0]).toEqual({ label: 'Aug 26', count: 100 });
    expect(v.series.at(-1)).toEqual({ label: 'Sep 24', count: 200 });
    expect(v.series.find((s) => s.label === 'Sep 20')?.count).toBe(0);
    expect(v.gainedDays).toBe(3);
    expect(v.intervals).toBe(5);
    expect(v.net).toBe(350);
  });

  test('the one insight card is the net change over the window', () => {
    expect(v.insight).toEqual({
      value: 350,
      display: '+350',
      headline: 'Gained 350 followers in the last 30 days.',
      body: 'up on 3 of 5 tracked days · best day Sep 24 (+200)',
      frac: 0.6,
    });
  });

  test('a net loss says so', () => {
    const d = platformVolume({
      label: 'X',
      followers: 950,
      growth: { d7: null, d30: null, d60: null, allTime: null },
      snapshots: [
        { capturedAt: '2026-09-22', followers: 1000, source: 's' },
        { capturedAt: '2026-09-24', followers: 950, source: 's' },
      ],
      today: '2026-09-24',
    });
    expect(d.chips).toEqual([]);
    expect(d.insight).toMatchObject({ value: -50, display: '-50', headline: 'Lost 50 followers in the last 30 days.', frac: 0 });
    expect(d.insight.body).toBe('up on 0 of 1 tracked days');
  });
});

describe('platformVolume with little or nothing to show', () => {
  test('no snapshots: empty meters, empty line, honest copy', () => {
    const e = platformVolume({ label: 'LinkedIn', followers: null, growth: { d7: null, d30: null, d60: null, allTime: null }, snapshots: [], today: '2026-09-24' });
    expect(e.headline).toBeNull();
    expect(e.chips).toEqual([]);
    expect(e.caption).toBe('no follower reading yet');
    expect(e.meters).toEqual([]);
    expect(e.foot).toBe('no snapshots yet');
    expect(e.series).toHaveLength(30);
    expect(e.series.every((s) => s.count === 0)).toBe(true);
    expect(e.net).toBeNull();
    expect(e.insight).toMatchObject({ value: 0, display: 'none', frac: 0, headline: 'Not enough history for a trend yet.' });
  });

  test('one snapshot is a reading, not a trend: no gained/dipped meters', () => {
    const one = platformVolume({
      label: 'TikTok',
      followers: 400,
      growth: { d7: null, d30: null, d60: null, allTime: null },
      snapshots: [{ capturedAt: '2026-09-24', followers: 400, source: 's' }],
      today: '2026-09-24',
    });
    expect(one.meters.map((m) => m.label)).toEqual(['Tracked days (1/30)', 'Of all-time peak']);
    expect(one.intervals).toBe(0);
    expect(one.insight.display).toBe('none');
  });
});
