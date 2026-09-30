import { describe, expect, test } from 'vitest';
import { usageVolume, LANE_HUE } from '@/lib/usage-volume';
import type { OllamaBoard, PlanUsage, Tot } from '@/lib/usage';

/**
 * /usage wears Brand Deals' Deal Volume card (Alex, 2026-09-24). The board's
 * existing reading (one Claude plan, one ChatGPT/Codex plan, one Ollama plan)
 * becomes a headline, chips, lane meters, a 7-day step line, a model waffle,
 * the official limit gauges and one insight. Burn = in + out + cache writes,
 * the same measure the board has always used.
 */
const NOW = new Date('2026-09-24T15:00:00').getTime();
const t = (n: number): Tot => ({ in: n, out: 0, cacheWrite: 0, cacheRead: 0 });
const lanes = (board: number, sessions: number, terminal: number, automation: number) => ({ board: t(board), sessions: t(sessions), terminal: t(terminal), automation: t(automation) });
const days = (burns: number[]) =>
  burns.map((b, i) => ({ day: `2026-09-${String(18 + i).padStart(2, '0')}`, in: b, out: 0, cacheWrite: 0, cacheRead: 5 }));

const claude: PlanUsage = {
  plan: 'Claude Max 20x',
  official: {
    session: { usedPercent: 42, windowMinutes: 300, resetsAt: new Date(NOW + 3 * 3600_000).toISOString() },
    weekly: { usedPercent: 91, windowMinutes: 10080, resetsAt: new Date(NOW + 2 * 86400_000).toISOString() },
  },
  days: days([100_000, 0, 300_000, 0, 0, 200_000, 1_000_000]),
  byModel: { 'claude-opus-4-5': t(1_200_000), 'claude-haiku-4-5': t(400_000) },
  breakdown: {
    windows: { hour: lanes(0, 0, 0, 0), session: lanes(0, 0, 0, 0), day: lanes(500_000, 300_000, 150_000, 50_000), week: lanes(900_000, 400_000, 200_000, 100_000) },
    top: [
      { source: 'board', label: 'Conductor', burn: 600_000 },
      { source: 'sessions', label: 'awake-quality', burn: 300_000 },
    ],
  },
  lastActivity: new Date(NOW - 60_000).toISOString(),
  machines: [{ id: 'mbp', label: 'Claude · mbp', source: 'local', capturedAt: new Date(NOW).toISOString(), lastActivity: null, stale: false }],
};
const codex: PlanUsage = {
  plan: 'ChatGPT Pro',
  official: { session: { usedPercent: 75, windowMinutes: 300, resetsAt: null } },
  days: days([0, 0, 0, 0, 0, 0, 200_000]),
  byModel: { 'gpt-5-codex': t(200_000) },
  breakdown: {
    windows: { hour: lanes(0, 0, 0, 0), session: lanes(0, 0, 0, 0), day: lanes(0, 0, 200_000, 0), week: lanes(0, 0, 200_000, 0) },
    top: [{ source: 'terminal', label: 'Alex-OS', burn: 200_000 }],
  },
  lastActivity: null,
  machines: [{ id: 'mini', label: 'Codex · mini', source: 'push', capturedAt: new Date(NOW).toISOString(), lastActivity: null, stale: false }],
};
const ollama: OllamaBoard = {
  state: 'up',
  plan: 'Ollama Pro',
  models: [{ name: 'bge-m3', cloud: false, host: 'Ollama · mbp' }],
  requests: { hour: { chat: 1, embed: 2 }, session: { chat: 3, embed: 4 }, day: { chat: 5, embed: 7 }, week: { chat: 20, embed: 30 } },
  note: '',
  machines: [],
};

const v = usageVolume({ claude, codex, ollama, now: NOW });

describe('usageVolume: the headline', () => {
  test('is today’s burn across Claude and Codex, formatted like the board', () => {
    expect(v.headlineTokens).toBe(1_200_000);
    expect(v.headline).toBe('1.2M');
  });

  test('chips are the official 5h gauges in status tones, plus Ollama requests today', () => {
    expect(v.chips).toEqual([
      { tone: 'ok', text: 'Claude 5h 42%' },
      { tone: 'warn', text: 'Codex 5h 75%' },
      { tone: 'accent', text: '12 Ollama requests today' },
    ]);
  });

  test('the caption and the meta line say where the number came from', () => {
    expect(v.caption).toBe('tokens burned today across Claude + Codex · 1.8M over 7 days');
    expect(v.meta).toBe('1.2M today · 1.8M this week · 2 machines reporting · Claude Max 20x · ChatGPT Pro · Ollama Pro');
  });
});

describe('usageVolume: the lane meters', () => {
  test('one meter per lane, each its share of today’s lane burn', () => {
    expect(v.meters.map((m) => m.label)).toEqual(['Agent board', 'My sessions · Superset', 'Terminal', 'Crons & headless']);
    expect(v.meters[0]).toMatchObject({ frac: 500_000 / 1_200_000, display: '500.0k' });
    expect(v.meters[2]).toMatchObject({ frac: 350_000 / 1_200_000, display: '350.0k' });
    expect(v.meters.map((m) => m.hue)).toEqual([LANE_HUE.board, LANE_HUE.sessions, LANE_HUE.terminal, LANE_HUE.automation]);
    for (const h of Object.values(LANE_HUE)) expect(h).toMatch(/^var\(--(accent|ramp-[1-4]|send-activity)\)$/);
  });
});

describe('usageVolume: series, models, limits, insight', () => {
  test('7 days of burn in thousands, oldest first, labelled by date', () => {
    expect(v.series.map((s) => s.label)).toEqual(['Sep 18', 'Sep 19', 'Sep 20', 'Sep 21', 'Sep 22', 'Sep 23', 'Sep 24']);
    expect(v.series.map((s) => s.count)).toEqual([100, 0, 300, 0, 0, 200, 1200]);
    expect(v.weekTokens).toBe(1_800_000);
  });

  test('models: biggest burners first, names shortened, with the top share', () => {
    expect(v.models.map((m) => m.label)).toEqual(['opus-4-5', 'haiku-4-5', 'gpt-5-codex']);
    expect(v.models[0].count).toBe(1_200_000);
    expect(v.topModel).toEqual({ label: 'opus-4-5', share: 1_200_000 / 1_800_000 });
  });

  test('limits: every official gauge as a meter, hued by how close it runs', () => {
    expect(v.limits.map((m) => m.label)).toEqual(['Claude · 5h session', 'Claude · weekly', 'Codex · 5h session']);
    expect(v.limits[0]).toMatchObject({ frac: 0.42, display: '42% · resets in 3h', hue: 'var(--ok)' });
    expect(v.limits[1]).toMatchObject({ frac: 0.91, display: '91% · resets in 2d', hue: 'var(--err)' });
    expect(v.limits[2]).toMatchObject({ frac: 0.75, display: '75%', hue: 'var(--warn)' });
  });

  test('the insight is the week’s single biggest burner and its share of the lane burn', () => {
    expect(v.insight.value).toBe(33);
    expect(v.insight.headline).toBe('Conductor');
    expect(v.insight.body).toBe('Agent board · 600.0k of 1.8M burned this week');
    expect(v.insight.frac).toBeCloseTo(600_000 / 1_800_000);
  });
});

describe('usageVolume: nothing reported', () => {
  const e = usageVolume({ claude: null, codex: null, ollama: { ...ollama, state: 'down', plan: null, requests: null }, now: NOW });

  test('empty data gives empty meters and honest copy, never fabricated numbers', () => {
    expect(e.headlineTokens).toBe(0);
    expect(e.headline).toBe('0');
    expect(e.chips).toEqual([{ tone: 'err', text: 'Ollama down' }]);
    expect(e.meters.every((m) => m.frac === 0)).toBe(true);
    expect(e.series).toEqual([]);
    expect(e.models).toEqual([]);
    expect(e.topModel).toBeNull();
    expect(e.limits).toEqual([]);
    expect(e.insight).toMatchObject({ value: 0, frac: 0, headline: 'Nothing burned this week.' });
  });

  test('a plan with no lane breakdown still counts toward the headline, not the lanes', () => {
    const old = usageVolume({ claude: { ...claude, breakdown: undefined, official: null }, codex: null, ollama, now: NOW });
    expect(old.headlineTokens).toBe(1_000_000);
    expect(old.meters.every((m) => m.frac === 0)).toBe(true);
  });
});

describe('model labels fit the dot matrix', () => {
  test('the dated suffix and the claude- prefix are dropped', async () => {
    const { shortModel } = await import('@/lib/usage-volume');
    expect(shortModel('claude-haiku-4-5-20251001')).toBe('haiku-4-5');
    expect(shortModel('opus-5')).toBe('opus-5');
  });
});
