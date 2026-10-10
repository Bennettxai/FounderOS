import { describe, expect, test } from 'vitest';
import { ACTIVITY_HUE, LANE_HUE, fmtTokens, shortModel, usageVolume, windowTable, type Breakdown } from './usage';

import { NOW, claude, codex, ollama } from './fixtures';

const v = usageVolume({ claude, codex, ollama, now: NOW });

describe('usageVolume: the headline', () => {
	test('is today’s burn across Claude and Codex', () => {
		expect(v.headlineTokens).toBe(1_200_000);
		expect(fmtTokens(v.headlineTokens!)).toBe('1.2M');
	});
	test('chips are the official 5h gauges in status tones, plus Ollama requests today', () => {
		expect(v.chips).toEqual([
			{ tone: 'ok', text: 'Claude 5h 42%' },
			{ tone: 'warn', text: 'Codex 5h 75%' },
			{ tone: 'accent', text: '12 Ollama requests today' }
		]);
	});
	test('the caption and the meta line say where the number came from', () => {
		expect(v.caption).toBe('tokens burned today across Claude + Codex · 1.8M over 7 days');
		expect(v.meta).toBe('1.2M today · 1.8M this week · 2 machines reporting · Claude Max 20x · ChatGPT Pro · Ollama Pro');
	});
});

describe('usageVolume: the lane meters', () => {
	test('one meter per lane, each its share of today’s lane burn, in Monolith greys', () => {
		expect(v.meters.map((m) => m.label)).toEqual(['Agent board', 'My sessions · Superset', 'Terminal', 'Crons & headless']);
		expect(v.meters[0]).toMatchObject({ frac: 500_000 / 1_200_000, display: '500.0k' });
		expect(v.meters[2]).toMatchObject({ frac: 350_000 / 1_200_000, display: '350.0k' });
		expect(v.meters.map((m) => m.hue)).toEqual([LANE_HUE.board, LANE_HUE.sessions, LANE_HUE.terminal, LANE_HUE.automation]);
		// prod lib/usage-volume.ts: accent, ramp-1 (brain-2), ramp-3 (brain-1), ramp-4 (brain-1 70% into accent)
		expect(LANE_HUE).toEqual({
			board: 'var(--bn-accent)',
			sessions: 'var(--bn-brain-2)',
			terminal: 'var(--bn-brain-1)',
			automation: 'color-mix(in oklab, var(--bn-brain-1) 70%, var(--bn-accent))'
		});
		// --send-activity
		expect(ACTIVITY_HUE).toBe('color-mix(in oklab, var(--bn-brain-2) 70%, var(--bn-text))');
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
		expect(v.topModel).toEqual({ label: 'opus-4-5', share: 1_200_000 / 1_800_000 });
	});
	test('limits: every official gauge as a meter, hued by status token', () => {
		expect(v.limits.map((m) => m.label)).toEqual(['Claude · 5h session', 'Claude · weekly', 'Codex · 5h session']);
		expect(v.limits[0]).toMatchObject({ frac: 0.42, display: '42% · resets in 3h', hue: 'var(--bn-ok)' });
		expect(v.limits[1]).toMatchObject({ frac: 0.91, display: '91% · resets in 2d', hue: 'var(--bn-err)' });
		expect(v.limits[2]).toMatchObject({ frac: 0.75, display: '75%', hue: 'var(--bn-warn)' });
	});
	test('the insight is the week’s biggest burner and its share of the lane burn', () => {
		expect(v.insight.value).toBe(33);
		expect(v.insight.headline).toBe('Conductor');
		expect(v.insight.body).toBe('Agent board · 600.0k of 1.8M burned this week');
		expect(v.insight.frac).toBeCloseTo(600_000 / 1_800_000);
	});
});

describe('usageVolume: nothing reported reads unknown, never zero', () => {
	const e = usageVolume({ claude: null, codex: null, ollama: null, now: NOW });
	test('no pushed plan: headline, week, lanes and insight are unknown', () => {
		expect(e.headlineTokens).toBeNull();
		expect(e.weekTokens).toBeNull();
		expect(e.chips).toEqual([]);
		expect(e.meters.every((m) => m.frac === null && m.display === 'unknown')).toBe(true);
		expect(e.series).toEqual([]);
		expect(e.topModel).toBeNull();
		expect(e.limits).toEqual([]);
		expect(e.insight).toMatchObject({ value: null, frac: null, headline: 'No machine has reported usage yet.' });
		expect(e.meta).toBe('today unknown · week unknown · 0 machines reporting');
	});
	test('a reported Ollama that is down still says so', () => {
		const d = usageVolume({ claude: null, codex: null, ollama: { ...ollama, state: 'down', requests: null }, now: NOW });
		expect(d.chips).toEqual([{ tone: 'err', text: 'Ollama down' }]);
	});
	test('a plan with no lane breakdown counts toward the headline; its lanes are unknown', () => {
		const old = usageVolume({ claude: { ...claude, breakdown: undefined, official: null }, codex: null, ollama, now: NOW });
		expect(old.headlineTokens).toBe(1_000_000);
		expect(old.meters.every((m) => m.frac === null)).toBe(true);
		expect(old.insight.value).toBeNull();
	});
});

describe('windowTable', () => {
	const b = claude.breakdown as Breakdown;
	test('with an official weekly %, each window becomes an estimated share of the weekly limit', () => {
		const rows = windowTable(b, 91);
		expect(rows.map((r) => r.window)).toEqual(['hour', 'session', 'day', 'week']);
		expect(rows[2].shareOfWeek).toBeCloseTo(1_000_000 / 1_600_000);
		expect(rows[2].pctOfWeekly).toBeCloseTo(91 * (1_000_000 / 1_600_000));
		expect(rows[3].pctOfWeekly).toBe(91);
	});
	test('without an official gauge there is no % of limit, only share of the week', () => {
		expect(windowTable(b, null).every((r) => r.pctOfWeekly === null)).toBe(true);
	});
});

test('model labels drop the claude- prefix and the dated suffix', () => {
	expect(shortModel('claude-haiku-4-5-20251001')).toBe('haiku-4-5');
	expect(shortModel('opus-5')).toBe('opus-5');
});
