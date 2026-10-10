// Wire types for /os/agents, mirroring internal/founderos/api/page_agents.go
// and internal/founderos/pages/agents (FounderOS v1 lib/board-live, agents-volume,
// board-deliverables, board-approvals).
import type { Meter, SeriesPoint, StatChip } from '$lib/founderos/kit/format';

export type PaperclipAgent = {
	id: string;
	name: string;
	status: string;
	adapterType: string | null;
	model: string | null;
	lastHeartbeatAt: string | null;
};

export type PaperclipIssue = {
	id: string;
	identifier: string;
	title: string;
	status: string;
	assigneeName: string | null;
	updatedAt: string | null;
};

export type PaperclipRun = {
	id: string;
	agentId: string;
	agentName: string | null;
	status: string;
	startedAt: string | null;
	finishedAt: string | null;
};

export type Decision = {
	id: string;
	decision: 'approved' | 'dismissed';
	decidedAt: string;
	decidedRevision: string;
	note: string;
};

export type BoardStats = { seats: number; running: number; openTasks: number; runs24h: number; heartbeats24h: number };

export type AgentsVolume = {
	headline: number;
	counts: { running: number; idle: number; paused: number; error: number };
	chips: StatChip[];
	caption: string;
	meters: Meter[];
	foot: string;
	openTasks: number;
	series: SeriesPoint[];
	window: string;
	runsInWindow: number;
	failedInWindow: number;
	bySeat: SeriesPoint[];
	lanes: SeriesPoint[];
	insight: { value: number; headline: string; body: string; frac: number };
};

/** GET /pages/board/live */
export type BoardLive = {
	connected: boolean;
	error?: string;
	decisionsError?: string;
	agents: PaperclipAgent[];
	issues: PaperclipIssue[];
	runs: PaperclipRun[];
	checkedAt: string;
	decisions: Decision[];
	volume: AgentsVolume;
	stats: BoardStats;
};

/** GET /pages/agents/view */
export type AgentsView = {
	boardUrl: string | null;
	hermesUrl: string;
	board: Omit<BoardLive, 'volume' | 'stats'>;
	volume: AgentsVolume;
	stats: BoardStats;
};

export type DeliverableItem = {
	id: string;
	name: string;
	kind: 'file' | 'link';
	url: string | null;
	meta: string;
	modifiedAt: string;
	sizeBytes: number | null;
	accessCode: string;
	title: string;
	summary: string;
	alsoAs?: string[];
	revision: string;
};
export type DeliverableGroup = { name: string; items: DeliverableItem[] };
export type Classified = DeliverableItem & {
	ask: string;
	needsYou: boolean;
	deadline: string | null;
	overdue: boolean;
	label: string;
	action: string;
	person: boolean;
	glyph: string;
	glyphTone: 'ok' | 'warn' | 'err' | 'dim';
	why: string;
};

/** GET /pages/board/deliverables */
export type DeliverablesBody = {
	groups: DeliverableGroup[];
	decisions: Decision[];
	dir: string;
	available: boolean;
	reason: string;
	needsYou: { open: Classified[]; decided: { item: DeliverableItem; decision: Decision }[] };
};

/** GET /pages/board/deliverables?file=&mode=view */
export type DeliverableView = { id: string; name: string; kind: 'text' | 'image' | 'pdf' | 'binary'; text: string | null; truncated: boolean };
