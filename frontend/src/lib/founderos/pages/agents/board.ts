/**
 * The presentational half of FounderOS v1 lib/board-live.ts: lane grouping,
 * seat order, run feed order and the small readouts BoardLive renders. The
 * numbers (stats, volume) are computed server-side in
 * internal/founderos/pages/agents and arrive with the payload.
 */
import type { Decision, PaperclipAgent, PaperclipIssue, PaperclipRun } from './types';

export const RUN_OK = new Set(['succeeded', 'completed', 'success', 'done']);

/** 'glm-5.2 x6 · claude-local x1': seats without a model show their adapter. */
export function modelSummary(agents: PaperclipAgent[]): string {
	const counts = new Map<string, number>();
	for (const a of agents) {
		const key = a.model ?? a.adapterType ?? 'unconfigured';
		counts.set(key, (counts.get(key) ?? 0) + 1);
	}
	return [...counts.entries()]
		.sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))
		.map(([model, n]) => `${model} x${n}`)
		.join(' · ');
}

/** The Conductor is the centralized agent, so it sits in the middle slot. */
export const SEAT_CENTRE_SLOT = 2;

export function orderSeats(agents: PaperclipAgent[]): PaperclipAgent[] {
	const seats = [...agents];
	if (seats.length <= SEAT_CENTRE_SLOT) return seats;
	const at = seats.findIndex((a) => /^conductor$/i.test(a.name));
	if (at < 0 || at === SEAT_CENTRE_SLOT) return seats;
	[seats[at], seats[SEAT_CENTRE_SLOT]] = [seats[SEAT_CENTRE_SLOT], seats[at]];
	return seats;
}

export const ISSUE_LANES = ['in_progress', 'in_review', 'review', 'todo', 'blocked', 'backlog', 'done'] as const;
/** Stages that render even when empty: a missing lane reads as a broken board. */
export const CORE_ISSUE_LANES = ['in_progress', 'todo', 'done'] as const;

export function groupIssues(issues: PaperclipIssue[]): { status: string; issues: PaperclipIssue[] }[] {
	const by = new Map<string, PaperclipIssue[]>();
	for (const i of issues) {
		if (i.status === 'cancelled') continue;
		if (!by.has(i.status)) by.set(i.status, []);
		by.get(i.status)!.push(i);
	}
	const core = new Set<string>(CORE_ISSUE_LANES);
	const known = ISSUE_LANES.filter((s) => by.has(s) || core.has(s));
	const unknown = [...by.keys()].filter((s) => !(ISSUE_LANES as readonly string[]).includes(s));
	return [...known, ...unknown].map((status) => ({ status, issues: by.get(status) ?? [] }));
}

/** Newest runs first; unstamped runs sort last, order stable. */
export function orderRuns(runs: PaperclipRun[]): PaperclipRun[] {
	return [...runs].sort((a, b) => {
		if (!a.startedAt && !b.startedAt) return 0;
		if (!a.startedAt) return 1;
		if (!b.startedAt) return -1;
		return b.startedAt.localeCompare(a.startedAt);
	});
}

/** When this seat's in-flight run started, or null when nothing is open. */
export function runningSince(runs: PaperclipRun[], agentId: string): string | null {
	const open = runs.filter((r) => r.agentId === agentId && r.status === 'running' && r.startedAt);
	return open.length === 0 ? null : orderRuns(open)[0].startedAt;
}

/** '14s' / '3m 12s' for a finished run, null while it is still going. */
export function runDuration(run: PaperclipRun): string | null {
	if (!run.startedAt || !run.finishedAt) return null;
	const ms = new Date(run.finishedAt).getTime() - new Date(run.startedAt).getTime();
	if (!Number.isFinite(ms) || ms < 0) return null;
	const s = Math.round(ms / 1000);
	if (s < 60) return `${s}s`;
	return `${Math.floor(s / 60)}m ${String(s % 60).padStart(2, '0')}s`;
}

export const runGlyph = (status: string) => (status === 'running' ? '▸' : RUN_OK.has(status) ? '✓' : '✕');
export const runTone = (status: string) => (status === 'running' ? 'ok' : RUN_OK.has(status) ? 'muted' : 'err');

/** Status overrides everything: green while on, red when broken. */
export function seatTone(agent: PaperclipAgent, live: boolean): 'err' | 'ok' | 'warn' | 'idle' {
	if (agent.status === 'error') return 'err';
	if (live || agent.status === 'running') return 'ok';
	if (agent.status === 'paused') return 'warn';
	return 'idle';
}

export function ago(iso: string | null, now = Date.now()): string {
	if (!iso) return '';
	const ms = now - new Date(iso).getTime();
	if (!Number.isFinite(ms) || ms < 0) return 'now';
	const s = Math.floor(ms / 1000);
	if (s < 60) return `${s}s ago`;
	const m = Math.floor(s / 60);
	if (m < 60) return `${m}m ago`;
	const h = Math.floor(m / 60);
	if (h < 24) return `${h}h ago`;
	return `${Math.floor(h / 24)}d ago`;
}

export const boardTaskDecisionId = (issueId: string) => `board:${issueId}`;

/** The decision standing against one board task; an agent touching the task
 *  (a new updatedAt) reopens it. */
export function boardDecisionFor(issue: Pick<PaperclipIssue, 'id' | 'updatedAt'>, decisions: Decision[]): Decision | undefined {
	const hit = decisions.find((d) => d.id === boardTaskDecisionId(issue.id));
	if (!hit) return undefined;
	if (hit.decidedRevision && hit.decidedRevision !== (issue.updatedAt ?? '')) return undefined;
	return hit;
}

/** Lane dot colour per stage, from the status tokens. */
export const LANE_DOT: Record<string, string> = {
	in_progress: 'var(--bn-ok)',
	in_review: 'var(--bn-warn)',
	review: 'var(--bn-warn)',
	blocked: 'var(--bn-err)',
	todo: 'var(--bn-text-2)',
	backlog: 'var(--bn-text-3)',
	done: 'var(--bn-text-3)'
};
