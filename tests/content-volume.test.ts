import { describe, expect, test } from 'vitest';
import { contentVolume, type ContentVolumeInput } from '@/lib/content-volume';

/**
 * /content in the Brand Deals look (Alex, 2026-09-24: "rebuild the entire
 * OS in that light"). The Content Volume card, the posting step line, the
 * crew-runs dot matrix and the one "Since last post" card all come from this
 * pure view-model, fed with the page's own rows: the content crew, their real
 * agent runs, the lead magnets table and Zernio's posting history. Nothing
 * invented: no posts means an empty line and honest copy, not a zero dressed
 * up as a reading.
 */
const TODAY = '2026-09-24';

const base = (over: Partial<ContentVolumeInput> = {}): ContentVolumeInput => ({
  crew: [
    { id: 'social-agent', name: 'Social Agent', status: 'active' },
    { id: 'zernio-publisher', name: 'Zernio Publisher', status: 'active' },
    { id: 'arcads-creative', name: 'Arcads Creative', status: 'planned' },
  ],
  runs: [
    { agentId: 'social-agent', startedAt: '2026-09-23T10:00:00Z', ok: true },
    { agentId: 'social-agent', startedAt: '2026-09-20T10:00:00Z', ok: false },
    { agentId: 'zernio-publisher', startedAt: '2026-09-22T10:00:00Z', ok: true },
    // outside the 30-day window: ignored
    { agentId: 'zernio-publisher', startedAt: '2026-07-01T10:00:00Z', ok: true },
  ],
  leadMagnets: [
    { name: 'Hook vault', status: 'live' },
    { name: 'VSL teardown', status: 'live' },
    { name: 'Offer doc', status: 'draft' },
  ],
  recent: [{ status: 'published' }, { status: 'published' }, { status: 'scheduled' }],
  postDays: [
    { date: '2026-09-22', platforms: ['instagram', 'tiktok'] },
    { date: '2026-09-22', platforms: ['youtube'] },
    { date: '2026-09-15T12:00:00Z', platforms: ['instagram'] },
    // outside the window
    { date: '2026-06-01', platforms: ['instagram'] },
  ],
  today: TODAY,
  ...over,
});

describe('contentVolume: the Content Volume card', () => {
  test('headline is posts out in the window, chips name active days and live magnets', () => {
    const v = contentVolume(base());
    expect(v.headline).toBe(3);
    expect(v.postsInWindow).toBe(3);
    expect(v.activeDays).toBe(2);
    expect(v.chips.map((c) => c.text)).toEqual(['2 active days', '2 magnets live']);
    expect(v.caption).toContain('30 days');
  });

  test('meters are honest fractions of a real whole', () => {
    const v = contentVolume(base());
    const byLabel = Object.fromEntries(v.meters.map((m) => [m.label, m]));
    expect(byLabel['Active posting days (2/30)'].frac).toBeCloseTo(2 / 30);
    expect(byLabel['Lead magnets live (2/3)'].frac).toBeCloseTo(2 / 3);
    expect(byLabel['Crew active (2/3)'].frac).toBeCloseTo(2 / 3);
    expect(byLabel['Crew runs OK (2/3)'].frac).toBeCloseTo(2 / 3);
    expect(byLabel['Crew runs OK (2/3)'].display).toBe('2 ok · 1 failed');
    for (const m of v.meters) {
      expect(m.frac).toBeGreaterThanOrEqual(0);
      expect(m.frac).toBeLessThanOrEqual(1);
      expect(m.hue).toMatch(/^var\(--/);
    }
    expect(v.foot).toBe('3 agents · 3 lead magnets · 3 recent posts pulled');
  });

  test('the step line covers the window oldest first, ending today', () => {
    const v = contentVolume(base());
    expect(v.series).toHaveLength(30);
    expect(v.series[29].label).toBe('Sep 24');
    expect(v.series.reduce((n, s) => n + s.count, 0)).toBe(3);
    const sep22 = v.series.find((s) => s.label === 'Sep 22');
    expect(sep22?.count).toBe(2);
  });

  test('crew runs: one column per crew member, counted inside the window', () => {
    const v = contentVolume(base());
    expect(v.crewRuns).toEqual([
      { label: 'Social', count: 2 },
      { label: 'Zernio', count: 1 },
      { label: 'Arcads', count: 0 },
    ]);
    expect(v.runsInWindow).toBe(3);
  });
});

describe('contentVolume: the lead magnets card', () => {
  test('counts every magnet by status', () => {
    const v = contentVolume(base({ leadMagnets: [{ name: 'a', status: 'live' }, { name: 'b', status: 'draft' }, { name: 'c', status: 'paused' }, { name: 'd', status: 'draft' }] }));
    expect(v.magnets).toEqual({ total: 4, live: 1, draft: 2, paused: 1, archived: 0 });
  });
});

describe('contentVolume: the one insight card', () => {
  test('counts the days since the last post and lights ticks by posting consistency', () => {
    const v = contentVolume(base());
    expect(v.insight.value).toBe(2);
    expect(v.insight.display).toBeUndefined();
    expect(v.insight.headline).toBe('2 days since the last post went out.');
    expect(v.insight.frac).toBeCloseTo(2 / 30);
  });

  test('posted today reads as today', () => {
    const v = contentVolume(base({ postDays: [{ date: TODAY, platforms: ['instagram'] }] }));
    expect(v.insight.value).toBe(0);
    expect(v.insight.headline).toBe('Posted today.');
  });
});

describe('contentVolume: empty data stays honest', () => {
  test('no posts, no runs, no magnets: empty line, empty meters, no fake zero', () => {
    const v = contentVolume(base({ runs: [], leadMagnets: [], recent: [], postDays: [], crew: [] }));
    expect(v.headline).toBe(0);
    expect(v.chips).toEqual([]);
    expect(v.meters).toEqual([]);
    expect(v.series.every((s) => s.count === 0)).toBe(true);
    expect(v.crewRuns).toEqual([]);
    expect(v.caption).toBe('no posts on record in the last 30 days');
    expect(v.insight.display).toBe('none');
    expect(v.insight.headline).toBe('Nothing posted on record yet.');
    expect(v.insight.frac).toBe(0);
  });

  test('crew columns cap at six so the dot matrix stays readable', () => {
    const crew = Array.from({ length: 9 }, (_, i) => ({ id: `a${i}`, name: `Agent ${i}`, status: 'active' as const }));
    const v = contentVolume(base({ crew }));
    expect(v.crewRuns).toHaveLength(6);
    // colliding first words are numbered
    expect(v.crewRuns.map((c) => c.label).slice(0, 2)).toEqual(['Agent', 'Agent 2']);
  });
});

describe('an unreachable Zernio is unknown, not quiet (review 2026-09-24)', () => {
  test('postsKnown false: no "nothing posted" claims, the card says the history is unavailable', () => {
    const v = contentVolume(base({ postDays: [], postsKnown: false }));
    expect(v.caption).toMatch(/posting history unavailable/);
    expect(v.insight.display).toBe(' - ');
    expect(v.insight.headline).toBe('Posting history unavailable.');
    expect(v.insight.headline).not.toMatch(/Nothing posted/);
  });
});

describe('pages ask Zernio for a known-or-unknown history', () => {
  const fs = require('node:fs');
  test('the connector returns null for unknown; /content and /social use it', () => {
    expect(fs.readFileSync('lib/connectors/zernio.ts', 'utf8')).toMatch(/export async function zernioPostDaysKnown\(\): Promise<ZernioPostDay\[\] \| null>/);
    for (const f of ['app/content/page.tsx', 'app/social/page.tsx']) {
      const src = fs.readFileSync(f, 'utf8');
      expect(src, f).toContain('zernioPostDaysKnown(');
      expect(src, f).toContain('postsKnown');
    }
  });
});
