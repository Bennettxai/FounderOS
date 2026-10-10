/** Test fixtures shaped like the Go handlers' output (internal/founderos/api/page_brain.go). */
import type { BrainPage, KnowledgeGraph } from './types';

/** The Go builder's output for a tiny org (internal/founderos/pages/brain). */
export const GRAPH: KnowledgeGraph = {
	nodes: [
		{ id: 'self', kind: 'self', label: 'the operator', ring: 0 },
		{ id: 'board:p2', kind: 'board', label: 'Forge', ring: 1 },
		{ id: 'team:dept-tech', kind: 'team', label: 'TECH', ring: 1, color: '#a855f7' },
		{ id: 'team:dept-sales', kind: 'team', label: 'Sales', ring: 1, color: '#ef4444' },
		{ id: 'task:sop-c', kind: 'task', label: 'Run the board', ring: 2 },
		{ id: 'task:sop-l', kind: 'task', label: 'Close deals', ring: 2 },
		{ id: 'emp:conductor', kind: 'employee', label: 'Conductor', ring: 3 },
		{ id: 'person:len', kind: 'person', label: 'Len', ring: 3 },
		{ id: 'emp:loner', kind: 'employee', label: 'Loner', ring: 3 },
		{ id: 'tool:openclaw', kind: 'tool', label: 'Openclaw', ring: 4 },
		{ id: 'tool:attio@dept-sales', kind: 'tool', label: 'Attio', ring: 4 },
		{ id: 'tool:attio@dept-tech', kind: 'tool', label: 'Attio', ring: 4 }
	],
	edges: [
		{ source: 'self', target: 'board:p2', kind: 'board' },
		{ source: 'self', target: 'team:dept-tech', kind: 'pillar' },
		{ source: 'self', target: 'team:dept-sales', kind: 'pillar' },
		{ source: 'team:dept-tech', target: 'task:sop-c', kind: 'sop' },
		{ source: 'task:sop-c', target: 'emp:conductor', kind: 'does' },
		{ source: 'team:dept-sales', target: 'task:sop-l', kind: 'sop' },
		{ source: 'task:sop-l', target: 'person:len', kind: 'does' },
		{ source: 'emp:loner', target: 'team:dept-tech', kind: 'member' },
		{ source: 'emp:conductor', target: 'tool:openclaw', kind: 'uses' },
		{ source: 'emp:conductor', target: 'tool:attio@dept-tech', kind: 'uses' },
		{ source: 'person:len', target: 'tool:attio@dept-sales', kind: 'uses' }
	]
};

export const PAGE: BrainPage = {
	graph: GRAPH,
	directory: [
		{ kind: 'employee', title: 'AI agents', rows: [{ id: 'emp:conductor', label: 'Conductor', sub: 'TECH', deptIds: ['dept-tech'] }, { id: 'emp:loner', label: 'Loner', sub: 'TECH', deptIds: ['dept-tech'] }] },
		{ kind: 'person', title: 'Humans', rows: [{ id: 'person:len', label: 'Len', sub: 'Sales', deptIds: ['dept-sales'] }] },
		{ kind: 'task', title: 'SOPs', rows: [{ id: 'task:sop-l', label: 'Close deals', sub: 'Sales', deptIds: ['dept-sales'] }, { id: 'task:sop-c', label: 'Run the board', sub: 'TECH', deptIds: ['dept-tech'] }] },
		{ kind: 'tool', title: 'Tools', rows: [{ id: 'attio', label: 'Attio', sub: '2 users', deptIds: ['dept-sales', 'dept-tech'] }, { id: 'openclaw', label: 'Openclaw', sub: '1 user', deptIds: ['dept-tech'] }] }
	],
	departments: [
		{ id: 'dept-sales', name: 'Sales', slug: 'sales', tagline: 'Close the room', color: '#fff', order: 1 },
		{ id: 'dept-tech', name: 'TECH', slug: 'tech', tagline: '', color: '#fff', order: 5 }
	],
	agents: [
		{ id: 'conductor', departmentId: 'dept-tech', name: 'Conductor', role: 'Super agent', status: 'active', tier: 'lead', description: 'Runs the board', model: 'opus', tools: ['openclaw', 'attio'], parentId: null, instance: 'builtin' },
		{ id: 'loner', departmentId: 'dept-tech', name: 'Loner', role: '', status: 'idle', tier: 'worker', description: '', model: '', tools: [], parentId: null, instance: 'builtin' }
	],
	people: [{ id: 'len', departmentId: 'dept-sales', name: 'Len', role: 'Closer', tools: ['attio'] }],
	tasks: [
		{ id: 'sop-c', departmentId: 'dept-tech', title: 'Run the board', summary: '', steps: ['a', 'b', 'c'], assigneeKind: 'agent', assigneeId: 'conductor' },
		{ id: 'sop-l', departmentId: 'dept-sales', title: 'Close deals', summary: 'Book and close', steps: ['book', 'call', 'close'], assigneeKind: 'person', assigneeId: 'len' }
	],
	execTitles: { 'dept-sales': 'CRO', 'dept-tech': 'CTO' },
	runsByAgent: { conductor: { id: 'r2', agentId: 'conductor', startedAt: '2026-09-30T10:00:00Z', finishedAt: '2026-09-30T10:01:00Z', ok: true, summary: 'swept the board', model: null } },
	boardLeads: { 'dept-sales': { id: 'p1', name: 'Sales', status: 'idle', model: 'claude-opus' } },
	boardAgents: [{ id: 'p2', name: 'Forge', status: 'running', model: null }],
	board: { state: 'connected', detail: '2 live seats' }
};
