// Wire types for GET /api/founderos/pages/tasks (internal/founderos/api/page_tasks.go).
import type { Meter, SeriesPoint, StatChip } from '$lib/founderos/kit/format';

export type TaskStatus = 'open' | 'doing' | 'review' | 'done';

export type AgentTask = {
	id: string;
	agentId: string;
	title: string;
	status: TaskStatus;
	createdAt: string;
	updatedAt: string;
};

export type BoardIssue = {
	id: string;
	identifier: string;
	title: string;
	status: string;
	assigneeName: string | null;
	updatedAt: string | null;
};

export type AgentCron = {
	id: string;
	agentId: string;
	schedule: string;
	description: string;
	enabled: boolean;
	createdAt: string;
};

export type JobRow = {
	id: string;
	description: string;
	agentId: string;
	agentName: string;
	unknownAgent: boolean;
	schedule: string;
	scheduleLabel: string;
	enabled: boolean;
	runs: number;
	ok: number;
	lastRunAt: string | null;
	lastOk: boolean | null;
	nextRunAt: string | null;
	overdue: boolean;
	history: boolean[];
	lastSummary: string | null;
};

export type TasksVolume = {
	headline: number;
	counts: Record<TaskStatus, number>;
	board: { total: number; inProgress: number; blocked: number; done: number };
	boardOnline: boolean;
	chips: StatChip[];
	caption: string;
	meters: Meter[];
	foot: string;
	series: SeriesPoint[];
	touchesInWindow: number;
	cron: { runsInWindow: number; failedInWindow: number; rhythm: SeriesPoint[] };
	owners: SeriesPoint[];
	busiestOwner: { name: string; count: number } | null;
	insight: { value: number; headline: string; body: string; frac: number };
};

export type TasksView = {
	tasks: AgentTask[];
	agentNames: Record<string, string>;
	issues: BoardIssue[];
	board: { connected: boolean; error?: string; url: string | null };
	crons: AgentCron[];
	cronStats: Record<string, { runs: number; ok: number; lastRunAt: string | null; lastOk: boolean | null }>;
	jobs: JobRow[];
	volume: TasksVolume;
};
