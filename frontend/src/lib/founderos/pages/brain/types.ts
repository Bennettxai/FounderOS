/** Wire types for /api/founderos/pages/brain/* (desktop/backend-go/internal/founderos/api/page_brain.go). */

export type KGNodeKind = 'self' | 'board' | 'team' | 'task' | 'employee' | 'person' | 'tool';
export type KGNode = { id: string; kind: KGNodeKind; label: string; ring: number; color?: string };
export type KGEdgeKind = 'pillar' | 'sop' | 'does' | 'member' | 'uses' | 'reports' | 'board';
export type KGEdge = { source: string; target: string; kind: KGEdgeKind };
export type KnowledgeGraph = { nodes: KGNode[]; edges: KGEdge[] };

export type Department = { id: string; name: string; slug: string; tagline: string; color: string; order: number };
export type Agent = {
	id: string;
	departmentId: string;
	name: string;
	role: string;
	status: string;
	tier: string;
	description: string;
	model: string;
	tools: string[];
	parentId: string | null;
	instance: string;
};
export type Person = { id: string; departmentId: string; name: string; role: string; tools: string[] };
export type SopTask = {
	id: string;
	departmentId: string;
	title: string;
	summary: string;
	steps: string[];
	assigneeKind: 'agent' | 'person';
	assigneeId: string;
};
export type DirectoryRow = { id: string; label: string; sub: string; deptIds: string[] };
export type DirectoryGroup = { kind: 'employee' | 'person' | 'task' | 'tool'; title: string; rows: DirectoryRow[] };
export type AgentRun = { id: string; agentId: string; startedAt: string; finishedAt: string; ok: boolean; summary: string; model: string | null };
export type Seat = { id: string; name: string; status: string; model: string | null };

export type BrainPage = {
	graph: KnowledgeGraph;
	directory: DirectoryGroup[];
	departments: Department[];
	agents: Agent[];
	people: Person[];
	tasks: SopTask[];
	execTitles: Record<string, string>;
	runsByAgent: Record<string, AgentRun>;
	boardLeads: Record<string, Seat>;
	boardAgents: Seat[];
	board: { state: 'connected' | 'error'; detail: string };
};

export type MemoryNode = {
	id: string;
	type: 'folder' | 'page';
	label: string;
	folder: string;
	genre?: string;
	excerpt: string;
	vx: number;
	vy: number;
	cluster: number;
	links: number;
};
export type MemoryEdge = { source: string; target: string; type: 'member' | 'wikilink' };
export type MemoryGraph = { nodes: MemoryNode[]; edges: MemoryEdge[] };

export type BrainGraphBody = {
	constellation: MemoryGraph;
	workspaces: { workspace: string; engine: string; pages: number | null; edges: number | null; error?: string }[];
	unstaged: string[];
};

export type DoctorCheck = { name: string; status: 'ok' | 'warn' | 'error'; message: string };
export type BrainOverview = {
	store: { path: string; totalFiles: number; folders: { name: string; files: number }[] };
	doctor: { connected: boolean; status: string; healthScore: number | null; checks: DoctorCheck[]; detail: string };
};

export type BrainSatellites = {
	mounted: boolean;
	statusLine: string;
	pages: number;
	folders: number;
	clusters: { label: string; pages: number }[];
	freshPct: number | null;
	stalePct: number | null;
};

export type BrainHit = { title: string; snippet: string; source: string; workspace: string; engine: string; uri: string; score: number | null };
export type BrainQueryBody = {
	query: string;
	provider: string;
	ranked?: 'rerank' | 'provider';
	results: BrainHit[];
	error?: string;
	/** Set on a partial read: these workspaces' engines did not answer. */
	failedWorkspaces?: string[];
	degraded?: string;
};

export type DumpResult = { ok: true; title: string; relPath: string; workspace: string; embedded: boolean; slug: string };
