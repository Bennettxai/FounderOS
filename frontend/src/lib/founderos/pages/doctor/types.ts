/** GET /api/founderos/pages/doctor: Optimal Engine health (internal/founderos/pages/doctor). */
import type { Meter, SeriesPoint, StatChip } from '$lib/founderos/kit';

export type DoctorCheck = { engine: string; name: string; status: 'ok' | 'warn' | 'error' | (string & {}); message: string };
export type EngineStore = { id: string; status: string; technology: string; rowCount: number; tables: Record<string, number> };
export type EngineReading = {
	name: string;
	url: string;
	reachable: boolean;
	error?: string;
	health?: string;
	/** null while the engine is unreachable: unknown, not empty. */
	workspaces: string[] | null;
	checks: DoctorCheck[] | null;
	stores: EngineStore[] | null;
	searches: number | null;
	uptimeMs: number | null;
};
export type StorageLayer = { name: string; sub: string; val: string; state: 'connected' | 'available' | 'error' };
export type PillarAxis = { id: string; label: string; color: string; score: number; roster: number; freshness: number; sop: number };
export type DoctorVolume = {
	headline: number | null;
	counts: { ok: number; warn: number; fail: number; total: number };
	chips: StatChip[];
	caption: string;
	meters: Meter[];
	foot: string;
	series: SeriesPoint[];
	runsInWindow: number;
	failedInWindow: number;
	store: {
		cols: SeriesPoint[];
		/** every knowledge table an engine reported (cols is capped); absent on older backends */
		tables?: number;
		total: number | null;
		top: { name: string; count: number } | null;
	};
	insight: { value: number | null; headline: string; body: string; frac: number };
	enginesUp: number;
	enginesTotal: number;
	workspaces: number;
	claims: number | null;
	facts: number | null;
	contexts: number | null;
};
export type DoctorBody = {
	generatedAt: string;
	windowDays: number;
	engines: EngineReading[];
	checks: DoctorCheck[];
	score: number | null;
	volume: DoctorVolume;
	layers: StorageLayer[];
	/** null when Postgres could not be read. */
	axes: PillarAxis[] | null;
	relational: { ok: boolean; error?: string };
	brain: { agents: string[]; last: { agentId: string; finishedAt: string; ok: boolean } | null };
};
