// Pure helpers behind the /os/tasks sections (FounderOS v1 TaskCronStrip,
// BoardTasks and TaskBoard), kept out of the .svelte files so they are tested.
import type { SeriesPoint } from '$lib/founderos/kit/format';
import type { AgentTask, JobRow, TaskStatus } from './types';

export const WINDOW_DAYS = 14;

/** '5m ago' / '3h ago' / '2d ago'; 'never' for a job that has not run. */
export function ago(iso: string | null, now = Date.now()): string {
	if (!iso) return 'never';
	const ms = now - Date.parse(iso);
	if (!Number.isFinite(ms) || ms < 0) return 'just now';
	const m = Math.floor(ms / 60_000);
	if (m < 60) return `${m}m ago`;
	const h = Math.floor(m / 60);
	if (h < 24) return `${h}h ago`;
	return `${Math.floor(h / 24)}d ago`;
}

/** The board queue's compact age: '5m', '3h', '2d'. */
export function age(iso: string | null, now = Date.now()): string {
	if (!iso) return '';
	const ms = now - Date.parse(iso);
	if (!Number.isFinite(ms) || ms < 0) return 'now';
	const m = Math.floor(ms / 60_000);
	if (m < 60) return `${m}m`;
	const h = Math.floor(m / 60);
	if (h < 24) return `${h}h`;
	return `${Math.floor(h / 24)}d`;
}

/** Time until the next fire ('12m', '3h', '2d'), null when past or unknown. */
export function inNext(iso: string | null, now = Date.now()): string | null {
	if (!iso) return null;
	const ms = Date.parse(iso) - now;
	if (!Number.isFinite(ms) || ms < 0) return null;
	const m = Math.round(ms / 60_000);
	if (m < 60) return `${m}m`;
	const h = Math.round(m / 60);
	if (h < 24) return `${h}h`;
	return `${Math.round(h / 24)}d`;
}

/** A job reads red when its last run failed, its slot passed, or its agent is missing. */
export const jobFailing = (r: Pick<JobRow, 'lastOk' | 'overdue' | 'unknownAgent'>) =>
	r.lastOk === false || r.overdue || r.unknownAgent;

/** Board status → the status token the queue tints it with (color means status). */
export function issueTone(status: string): string {
	if (status === 'done') return 'var(--bn-ok)';
	if (status === 'in_progress') return 'var(--bn-warn)';
	if (status === 'blocked') return 'var(--bn-err)';
	return 'var(--bn-text-2)';
}

export const ROUTES = ['Conductor routes', 'TECH', 'Sales'] as const;
export type Route = (typeof ROUTES)[number];

/** The composer's body: the routing chip rides as a hint in the description. */
export function issueBody(title: string, route: Route): { title: string; description?: string } {
	return { title: title.trim(), ...(route === 'Conductor routes' ? {} : { description: `Route to the ${route} pillar.` }) };
}

export const COLUMNS: { status: TaskStatus; label: string; tone: string }[] = [
	{ status: 'open', label: 'To do', tone: 'var(--bn-text-3)' },
	{ status: 'doing', label: 'In progress', tone: 'var(--bn-warn)' },
	{ status: 'review', label: 'In review', tone: 'var(--bn-accent)' },
	{ status: 'done', label: 'Done', tone: 'var(--bn-ok)' }
];

/** Click-advance target for each lane; done is terminal (drag it back). */
export const ADVANCE: Partial<Record<TaskStatus, TaskStatus>> = { open: 'doing', doing: 'review', review: 'done' };

export function moveTask(tasks: AgentTask[], id: string, status: TaskStatus): AgentTask[] {
	return tasks.map((t) => (t.id === id ? { ...t, status } : t));
}

/** The busiest weekday, or null when the window had no runs. */
export function busiestDay(rhythm: SeriesPoint[]): SeriesPoint | null {
	const best = rhythm.reduce<SeriesPoint | null>((b, d) => (!b || d.count > b.count ? d : b), null);
	return best && best.count > 0 ? best : null;
}
