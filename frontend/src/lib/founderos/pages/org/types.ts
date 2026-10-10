// Wire types for GET /api/founderos/pages/org (internal/founderos/pages/org View).

export type Department = { id: string; name: string; slug: string; tagline: string; color: string; order: number };

export type Agent = {
	id: string;
	departmentId: string;
	name: string;
	role: string;
	status: 'active' | 'idle' | 'training' | 'planned' | string;
	tier: 'lead' | 'specialist' | 'worker' | string;
	description: string;
	model: string;
	tools: string[];
	parentId: string | null;
	instance: string;
};

export type AgentNode = { agent: Agent; children: AgentNode[] };

export type BoardAgent = {
	id: string;
	name: string;
	status: 'running' | 'idle' | 'paused' | 'error' | string;
	adapterType: string | null;
	model: string | null;
	lastHeartbeatAt: string | null;
};

export type LifeArea = { id: string; label: string; color: string; detail: string; agents: string[]; departmentIds: string[] };

export type Venture = {
	id: string;
	label: string;
	kind: string;
	color: string;
	detail: string;
	brainTag: string;
	focus: string[];
	areaAgents: Record<string, string[]>;
	agentIds: string[];
};

export type Crew = {
	department: Department;
	area: LifeArea | null;
	live: BoardAgent | null;
	leads: Agent[];
	pills: AgentNode[];
	tools: string[];
	empty: boolean;
};

export type BroadcastReply = { id?: string; agentId: string; ok: boolean; reply: string; finishedAt: string };
export type Broadcast = { id: string; message: string; createdAt: string; replies: BroadcastReply[] };

export type OrgView = {
	departments: Department[];
	agents: Agent[];
	agentNames: Record<string, string>;
	conductor: Agent | null;
	tree: { totalAgents: number; activeAgents: number };
	crews: Crew[];
	live: {
		connected: boolean;
		error?: string;
		conductor: BoardAgent | null;
		byDepartment: Record<string, BoardAgent>;
		extras: BoardAgent[];
	};
	ventures: Venture[];
	lifeAreas: LifeArea[];
	lastBroadcast: Broadcast | null;
};
