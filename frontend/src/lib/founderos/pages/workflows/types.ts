// Wire types for /os/workflows (GET /pages/workflows/view and friends).
import type { Meter, SeriesPoint, StatChip } from '$lib/founderos/kit';

export type OwnerKind = 'human' | 'agent';
export type AutomationState = 'live' | 'suggested';

export type WorkflowStep = {
	id: string;
	title: string;
	detail: string;
	ownerKind: OwnerKind;
	owner: string;
	hoursPerWeek: number;
	tools: string[];
	edgeLabel: string | null;
	leakUsd: number | null;
	automation: { title: string; state: AutomationState; recoveredUsd: number } | null;
	branch: { from: string; condition: string } | null;
};

export type Workflow = { id: string; name: string; subtitle: string; revenueUsd: number; order: number; steps: WorkflowStep[] };

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

export type AgentRun = { id: string; agentId: string; startedAt: string; finishedAt: string; ok: boolean; summary: string };

export type WorkflowsVolume = {
	headline: number;
	counts: { healthy: number; overdue: number; failing: number; paused: number; enabled: number };
	chips: StatChip[];
	caption: string;
	meters: Meter[];
	foot: string;
	series: SeriesPoint[];
	runsInWindow: number;
	failedInWindow: number;
	rhythm: SeriesPoint[];
	load: { manualHours: number; agentHours: number; perWorkflow: SeriesPoint[] };
	insight: { value: number; headline: string; body: string; frac: number };
};

export type SimpleAgent = { id: string; name: string };
export type AgentPresence = 'active' | 'inactive';

export type WorkflowsView = {
	workflows: Workflow[];
	jobs: JobRow[];
	volume: WorkflowsVolume;
	agents: SimpleAgent[];
	agentPresence: Record<string, AgentPresence>;
	runsByOwner: Record<string, AgentRun[]>;
	toolIds: string[];
};

/** The builder / draft shape: branch parents by index (app/api/workflows/shared.ts). */
export type StepInput = {
	title: string;
	detail: string;
	ownerKind: OwnerKind;
	owner: string;
	hoursPerWeek: number;
	tools: string[];
	automation: { title: string; state: AutomationState; recoveredUsd: number } | null;
	branchFromIndex: number | null;
	branchCondition: string | null;
};
export type WorkflowInput = { name: string; subtitle: string; steps: StepInput[] };

export type DraftResult = { ok: boolean; draft?: WorkflowInput; unavailable?: boolean; error?: string };
