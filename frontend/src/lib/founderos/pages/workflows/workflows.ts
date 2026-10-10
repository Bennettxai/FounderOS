// Presentational logic for /os/workflows: the tree walk (FounderOS v1
// app/workflows/tree.ts), tool names (lib/workflow-tool-brands.ts), the
// scheduled-row state and the builder's form ⇄ payload mapping
// (components/WorkflowBuilder.tsx). Pure, so it is unit-tested directly.
import type { AutomationState, JobRow, OwnerKind, SimpleAgent, Workflow, WorkflowInput, WorkflowStep } from './types';

export type WorkflowTreeNode = { step: WorkflowStep; index: number; depth: number; children: WorkflowTreeNode[] };
export type WorkflowTree = { root: WorkflowTreeNode | null; nodeCount: number; maxDepth: number };

const linearParent = (i: number) => (i === 0 ? null : i - 1);

/** A step with a resolvable `branch.from` threads there; otherwise the previous slot. */
export function workflowStepParent(index: number, steps: WorkflowStep[]): number | null {
	if (index === 0) return null;
	const branch = steps[index].branch;
	if (branch) {
		const p = steps.findIndex((s) => s.id === branch.from);
		if (p !== -1) return p;
	}
	return index - 1;
}

export function buildWorkflowTree(
	steps: WorkflowStep[],
	parentIndex: (index: number, steps: WorkflowStep[]) => number | null = linearParent
): WorkflowTree {
	if (steps.length === 0) return { root: null, nodeCount: 0, maxDepth: 0 };
	const nodes: WorkflowTreeNode[] = steps.map((step, index) => ({ step, index, depth: 0, children: [] }));
	const kids = new Map<number, number[]>();
	let rootIndex: number | null = null;
	steps.forEach((_, i) => {
		const p = parentIndex(i, steps);
		if (p === null) {
			if (rootIndex === null) rootIndex = i;
			return;
		}
		const list = kids.get(p);
		if (list) list.push(i);
		else kids.set(p, [i]);
	});
	if (rootIndex === null) return { root: null, nodeCount: 0, maxDepth: 0 };
	let nodeCount = 0;
	let maxDepth = 0;
	const attach = (i: number, depth: number): WorkflowTreeNode => {
		nodeCount += 1;
		maxDepth = Math.max(maxDepth, depth);
		const node = nodes[i];
		node.depth = depth;
		node.children = (kids.get(i) ?? []).map((k) => attach(k, depth + 1));
		return node;
	};
	const root = attach(rootIndex, 0);
	return { root, nodeCount, maxDepth };
}

/** Tools a workflow touches, first-seen order, deduped. */
export function workflowToolIds(steps: WorkflowStep[]): string[] {
	const seen = new Set<string>();
	for (const s of steps) for (const t of s.tools) seen.add(t);
	return [...seen];
}

export const TOOL_NAMES: Record<string, string> = {
	arcads: 'Arcads',
	calendar: 'Google Calendar',
	fathom: 'Fathom',
	gmail: 'Gmail',
	manychat: 'ManyChat',
	notion: 'Notion',
	'proposal-gen': 'Proposal Generator',
	skool: 'Skool',
	slack: 'Slack',
	stripe: 'Stripe',
	trakyo: 'Trakyo',
	typeform: 'Typeform',
	zernio: 'Zernio',
	camera: 'Camera',
	gsend: 'gsend',
	instagram: 'Instagram',
	linkedin: 'LinkedIn',
	premiere: 'Premiere Pro',
	telegram: 'Telegram',
	youtube: 'YouTube'
};
export const TOOL_IDS = Object.keys(TOOL_NAMES).sort();
export const toolName = (id: string) => TOOL_NAMES[id] ?? id;

/** FounderOS v1 lib/workflow-tool-brands.ts: tool id → brand-logo slug + name.
    An unknown id degrades to its own identity, drawn as a lettermark. */
const TOOL_SLUGS: Record<string, string> = { calendar: 'googlecalendar' };
export function toolBrand(id: string): { slug: string; name: string } {
	return TOOL_NAMES[id] ? { slug: TOOL_SLUGS[id] ?? id, name: TOOL_NAMES[id] } : { slug: id, name: id };
}

/** Step colour + word: status tokens only (Monolith: colour means status). */
export function stepTone(step: WorkflowStep): { color: string; word: string } {
	if (step.ownerKind === 'human') return { color: 'var(--bn-text-2)', word: 'human step' };
	if (step.automation?.state === 'live') return { color: 'var(--bn-accent)', word: 'automated · live' };
	if (step.automation?.state === 'suggested') return { color: 'var(--bn-warn)', word: 'automation planned' };
	return { color: 'var(--bn-text-2)', word: 'agent step' };
}

export function stepGlyph(step: WorkflowStep): string {
	if (step.ownerKind === 'human') return '◯';
	if (step.automation?.state === 'suggested') return '◌';
	if (step.automation?.state === 'live') return 'ϟ';
	return '▣';
}

export function workflowStats(wf: Workflow) {
	let manualHours = 0;
	let agentHours = 0;
	let agentSteps = 0;
	for (const s of wf.steps) {
		if (s.ownerKind === 'human') manualHours += s.hoursPerWeek;
		else {
			agentSteps += 1;
			agentHours += s.hoursPerWeek;
		}
	}
	return {
		manualHours,
		agentHours,
		agentSteps,
		live: wf.steps.filter((s) => s.automation?.state === 'live').length,
		planned: wf.steps.filter((s) => s.automation?.state === 'suggested').length
	};
}

/** ScheduledTasks' state(): a missing agent can only fail, a late slot is overdue. */
export function jobState(job: JobRow): 'err' | 'warn' | 'ok' | 'off' {
	if (!job.enabled) return 'off';
	if (job.unknownAgent) return 'err';
	if (job.overdue) return 'warn';
	return job.lastOk === false ? 'err' : 'ok';
}

export function ago(iso: string | null, now = Date.now()): string {
	if (!iso) return 'never';
	const ms = now - new Date(iso).getTime();
	if (!Number.isFinite(ms) || ms < 0) return 'now';
	const m = Math.floor(ms / 60_000);
	if (m < 1) return 'just now';
	if (m < 60) return `${m}m ago`;
	const h = Math.floor(m / 60);
	if (h < 24) return `${h}h ago`;
	return `${Math.floor(h / 24)}d ago`;
}

/** The step drawer's run clock (prod WorkflowTree relativeTime): rounded, not floored. */
export function relativeTime(iso: string, now = Date.now()): string {
	const mins = Math.round((now - new Date(iso).getTime()) / 60_000);
	if (!Number.isFinite(mins)) return '';
	if (mins < 1) return 'just now';
	if (mins < 60) return `${mins}m ago`;
	const hours = Math.round(mins / 60);
	if (hours < 24) return `${hours}h ago`;
	return `${Math.round(hours / 24)}d ago`;
}

/** "6m" / "2h 10m" until a future minute; null once it has passed. */
export function untilLabel(iso: string | null, now = Date.now()): string | null {
	if (!iso) return null;
	const ms = new Date(iso).getTime() - now;
	if (!Number.isFinite(ms) || ms <= 0) return null;
	const m = Math.ceil(ms / 60_000);
	if (m < 60) return `${m}m`;
	const h = Math.floor(m / 60);
	if (h < 24) return `${h}h ${m % 60}m`;
	return `${Math.floor(h / 24)}d`;
}

export const CRON_PRESETS = [
	{ label: 'Every morning 9am', expr: '0 9 * * *' },
	{ label: 'Weekdays 9am', expr: '0 9 * * 1-5' },
	{ label: 'Every morning 7am', expr: '0 7 * * *' },
	{ label: 'Every evening 6pm', expr: '0 18 * * *' },
	{ label: 'Every hour', expr: '0 * * * *' },
	{ label: 'Every 15 min', expr: '*/15 * * * *' }
];

// ── the builder's form ⇄ payload mapping ──────────────────────────────────
export type BuilderStep = {
	key: string;
	title: string;
	detail: string;
	ownerAgentId: string;
	ownerHumanName: string;
	ownerKind: OwnerKind;
	hoursPerWeek: string;
	tools: string[];
	automationOn: boolean;
	automationTitle: string;
	automationState: AutomationState;
	branchFromKey: string;
	branchCondition: string;
};

let seq = 0;
export const newKey = () => `k${Date.now().toString(36)}-${(seq += 1)}`;

export function emptyStep(): BuilderStep {
	return {
		key: newKey(),
		title: '',
		detail: '',
		ownerAgentId: '',
		ownerHumanName: '',
		ownerKind: 'agent',
		hoursPerWeek: '0',
		tools: [],
		automationOn: false,
		automationTitle: '',
		automationState: 'live',
		branchFromKey: '',
		branchCondition: ''
	};
}

export function stepsFromWorkflow(wf: Workflow, agents: SimpleAgent[]): BuilderStep[] {
	const byName = new Map(agents.map((a) => [a.name, a.id]));
	return wf.steps.map((s) => ({
		key: s.id,
		title: s.title,
		detail: s.detail,
		ownerAgentId: byName.get(s.owner) ?? '',
		ownerHumanName: s.ownerKind === 'human' ? s.owner : '',
		ownerKind: s.ownerKind,
		hoursPerWeek: String(s.hoursPerWeek),
		tools: [...s.tools],
		automationOn: Boolean(s.automation),
		automationTitle: s.automation?.title ?? '',
		automationState: s.automation?.state ?? 'live',
		branchFromKey: s.branch?.from ?? '',
		branchCondition: s.branch?.condition ?? ''
	}));
}

/** A draft fills the form; an owner matching a roster agent becomes that agent. */
export function stepsFromDraft(draft: WorkflowInput, agents: SimpleAgent[]): BuilderStep[] {
	const drafted = draft.steps.map(() => emptyStep());
	const keys = drafted.map((s) => s.key);
	return draft.steps.map((s, i) => {
		const matched = agents.find((a) => a.name.toLowerCase() === s.owner.toLowerCase());
		return {
			...drafted[i],
			title: s.title,
			detail: s.detail,
			ownerKind: matched ? 'agent' : s.ownerKind,
			ownerAgentId: matched?.id ?? '',
			ownerHumanName: matched ? '' : s.owner,
			hoursPerWeek: String(s.hoursPerWeek),
			tools: s.tools,
			automationOn: Boolean(s.automation),
			automationTitle: s.automation?.title ?? '',
			automationState: s.automation?.state ?? 'live',
			branchFromKey: s.branchFromIndex != null ? (keys[s.branchFromIndex] ?? '') : '',
			branchCondition: s.branchCondition ?? ''
		};
	});
}

export function builderPayload(name: string, subtitle: string, steps: BuilderStep[], agents: SimpleAgent[]): WorkflowInput {
	const keys = steps.map((s) => s.key);
	const agentName = (id: string) => agents.find((a) => a.id === id)?.name ?? '';
	return {
		name: name.trim(),
		subtitle: subtitle.trim(),
		steps: steps.map((s) => ({
			title: s.title.trim(),
			detail: s.detail.trim(),
			ownerKind: s.ownerKind,
			owner: s.ownerKind === 'agent' ? agentName(s.ownerAgentId) : s.ownerHumanName.trim(),
			hoursPerWeek: Number(s.hoursPerWeek) || 0,
			tools: s.tools,
			automation: s.automationOn ? { title: s.automationTitle.trim() || 'Automation', state: s.automationState, recoveredUsd: 0 } : null,
			branchFromIndex: s.branchFromKey ? keys.indexOf(s.branchFromKey) : null,
			branchCondition: s.branchFromKey ? s.branchCondition.trim() || null : null
		}))
	};
}

export function validateBuilder(name: string, steps: BuilderStep[]): string | null {
	if (!name.trim()) return 'Give the workflow a name.';
	if (steps.length === 0) return 'A workflow needs at least one step.';
	for (const [i, s] of steps.entries()) {
		if (!s.title.trim()) return `Step ${i + 1} needs a title.`;
		if (!s.detail.trim()) return `Step ${i + 1} needs a short description.`;
		if (s.ownerKind === 'agent' && !s.ownerAgentId) return `Step ${i + 1} needs an owner.`;
		if (s.ownerKind === 'human' && !s.ownerHumanName.trim()) return `Step ${i + 1} needs an owner.`;
		if (s.branchFromKey && s.branchFromKey === s.key) return `Step ${i + 1} cannot branch from itself.`;
	}
	return null;
}
