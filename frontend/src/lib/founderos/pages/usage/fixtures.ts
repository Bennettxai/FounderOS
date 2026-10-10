import type { OllamaBoard, PlanUsage, Tot } from './usage';

// Ported from FounderOS v1 tests/usage-volume.test.ts and usage-breakdown.test.ts.
export const NOW = new Date('2026-09-24T15:00:00').getTime();
const t = (n: number): Tot => ({ in: n, out: 0, cacheWrite: 0, cacheRead: 0 });
const lanes = (board: number, sessions: number, terminal: number, automation: number) => ({
	board: t(board),
	sessions: t(sessions),
	terminal: t(terminal),
	automation: t(automation)
});
const days = (burns: number[]) =>
	burns.map((b, i) => ({ day: `2026-09-${String(18 + i).padStart(2, '0')}`, in: b, out: 0, cacheWrite: 0, cacheRead: 5 }));

export const claude: PlanUsage = {
	plan: 'Claude Max 20x',
	official: {
		session: { usedPercent: 42, windowMinutes: 300, resetsAt: new Date(NOW + 3 * 3600_000).toISOString() },
		weekly: { usedPercent: 91, windowMinutes: 10080, resetsAt: new Date(NOW + 2 * 86400_000).toISOString() }
	},
	days: days([100_000, 0, 300_000, 0, 0, 200_000, 1_000_000]),
	byModel: { 'claude-opus-4-5': t(1_200_000), 'claude-haiku-4-5': t(400_000) },
	breakdown: {
		windows: {
			hour: lanes(0, 0, 0, 0),
			session: lanes(0, 0, 0, 0),
			day: lanes(500_000, 300_000, 150_000, 50_000),
			week: lanes(900_000, 400_000, 200_000, 100_000)
		},
		top: [
			{ source: 'board', label: 'Conductor', burn: 600_000 },
			{ source: 'sessions', label: 'awake-quality', burn: 300_000 }
		]
	},
	lastActivity: new Date(NOW - 60_000).toISOString(),
	machines: [{ id: 'mbp', label: 'Claude · mbp', source: 'push', capturedAt: new Date(NOW).toISOString(), lastActivity: null, stale: false }]
};
export const codex: PlanUsage = {
	plan: 'ChatGPT Pro',
	official: { session: { usedPercent: 75, windowMinutes: 300, resetsAt: null } },
	days: days([0, 0, 0, 0, 0, 0, 200_000]),
	byModel: { 'gpt-5-codex': t(200_000) },
	breakdown: {
		windows: { hour: lanes(0, 0, 0, 0), session: lanes(0, 0, 0, 0), day: lanes(0, 0, 200_000, 0), week: lanes(0, 0, 200_000, 0) },
		top: [{ source: 'terminal', label: 'FounderOS-v1', burn: 200_000 }]
	},
	lastActivity: null,
	machines: [{ id: 'mini', label: 'Codex · mini', source: 'push', capturedAt: new Date(NOW).toISOString(), lastActivity: null, stale: false }]
};
export const ollama: OllamaBoard = {
	state: 'up',
	plan: 'Ollama Pro',
	models: [{ name: 'bge-m3', cloud: false, host: 'Ollama · mbp' }],
	requests: { hour: { chat: 1, embed: 2 }, session: { chat: 3, embed: 4 }, day: { chat: 5, embed: 7 }, week: { chat: 20, embed: 30 } },
	note: '',
	machines: []
};

