import { describe, expect, test } from 'vitest';
import { socialVolume } from '@/lib/social-volume';

/**
 * /social in the Brand Deals look (Alex, 2026-09-24). The Audience Volume
 * card, the posting step line, the platform mix and the one "Needs reply"
 * card, all shaped from the page's own real rows: the channel follower
 * counts, the audience growth, the Instagram DM threads, the publish queue
 * and Zernio's posting history. No numbers are invented here.
 */
const CHANNELS = [
  { key: 'instagram', label: 'Instagram', value: 6000 },
  { key: 'tiktok', label: 'TikTok', value: 2000 },
  { key: 'twitter', label: 'X', value: 500 },
  { key: 'youtube', label: 'YouTube', value: 1000 },
  { key: 'linkedin', label: 'LinkedIn', value: null },
  { key: 'email', label: 'Email list', value: 500 },
];

describe('socialVolume', () => {
  const v = socialVolume({
    channels: CHANNELS,
    total: 10_000,
    growth7: 1.234,
    leader: 'TikTok',
    dmThreads: [
      { name: 'Ava', unreplied: true },
      { name: 'Ben', unreplied: false },
      { name: 'Cy', unreplied: true },
      { name: 'Dee', unreplied: false },
    ],
    queued: 2,
    postDays: [
      { date: '2026-09-24', platforms: ['instagram', 'tiktok'] },
      { date: '2026-09-24', platforms: ['instagram'] },
      { date: '2026-09-20', platforms: ['twitter', 'mastodon'] },
      { date: '2026-07-01', platforms: ['youtube'] }, // outside 30 days
    ],
    today: '2026-09-24',
  });

  test('headline is the whole audience, chips carry growth, replies owed and the queue', () => {
    expect(v.headline).toBe(10_000);
    expect(v.chips).toEqual([
      { tone: 'ok', text: '+1.23% 7d' },
      { tone: 'warn', text: '2 need reply' },
      { tone: 'accent', text: '2 queued' },
    ]);
    expect(v.caption).toBe('across 5 live channels · TikTok leads 7-day growth');
  });

  test('meters are the four biggest channels as shares of total reach', () => {
    expect(v.meters.map((m) => m.label)).toEqual(['Instagram', 'TikTok', 'YouTube', 'X']);
    expect(v.meters[0]).toMatchObject({ frac: 0.6, display: '6,000 · 60%', hue: 'var(--ramp-1)' });
    expect(v.meters[3]).toMatchObject({ frac: 0.05, display: '500 · 5%', hue: 'var(--ramp-4)' });
    expect(v.foot).toBe('share of total reach · 2 smaller channels');
  });

  test('posting series: one point per day for 30 days, ending today', () => {
    expect(v.series).toHaveLength(30);
    expect(v.series.at(-1)).toEqual({ label: 'Sep 24', count: 2 });
    expect(v.series.find((s) => s.label === 'Sep 20')?.count).toBe(1);
    expect(v.series[0].label).toBe('Aug 26');
    expect(v.postsInWindow).toBe(3);
  });

  test('platform mix counts cross-posts per platform, unknown platforms ignored', () => {
    expect(v.mix).toEqual([
      { label: 'IG', count: 2 },
      { label: 'TT', count: 1 },
      { label: 'X', count: 1 },
      { label: 'YT', count: 0 },
      { label: 'LI', count: 0 },
    ]);
  });

  test('the one insight card is the DMs waiting on Alex', () => {
    expect(v.insight.value).toBe(2);
    expect(v.insight.headline).toBe('2 of 4 Instagram threads need a reply.');
    expect(v.insight.body).toBe('Ava · Cy');
    expect(v.insight.frac).toBeCloseTo(0.5);
  });

  test('negative growth is an err chip', () => {
    const d = socialVolume({ channels: CHANNELS, total: 10_000, growth7: -12.5, leader: null, dmThreads: [], queued: 0, postDays: [], today: '2026-09-24' });
    expect(d.chips).toEqual([{ tone: 'err', text: '-12.5% 7d' }]);
    expect(d.caption).toBe('across 5 live channels');
  });
});

describe('socialVolume with nothing to show', () => {
  const e = socialVolume({ channels: [], total: 0, growth7: null, leader: null, dmThreads: [], queued: 0, postDays: [], today: '2026-09-24' });

  test('empty meters and honest copy', () => {
    expect(e.headline).toBe(0);
    expect(e.chips).toEqual([]);
    expect(e.meters).toEqual([]);
    expect(e.caption).toBe('no channels reporting yet');
    expect(e.series.every((s) => s.count === 0)).toBe(true);
    expect(e.insight).toMatchObject({ value: 0, frac: 0, headline: 'Inbox clear.', body: 'No Instagram threads yet.' });
  });

  test('a channel with no reading is an empty meter, not a zero pretending to be data', () => {
    const o = socialVolume({
      channels: [{ key: 'linkedin', label: 'LinkedIn', value: null }],
      total: 0,
      growth7: null,
      leader: null,
      dmThreads: [],
      queued: 0,
      postDays: [],
      today: '2026-09-24',
    });
    expect(o.meters).toEqual([{ label: 'LinkedIn', frac: 0, display: 'offline', hue: 'var(--ramp-1)' }]);
  });
});
