import { describe, expect, it } from 'vitest';
import {
	ago,
	buildWorkflowTree,
	builderPayload,
	emptyStep,
	jobState,
	stepsFromDraft,
	stepsFromWorkflow,
	stepTone,
	untilLabel,
	validateBuilder,
	workflowStepParent,
	workflowToolIds,
	toolBrand,
	relativeTime
} from './workflows';
import { brandMarkKind } from '../integrations/brand-logos';
import { TOOL_ICONS } from './tool-marks';
import type { JobRow, WorkflowStep } from './types';

// Ported from FounderOS v1 tests/workflow-tree.test.ts, plus the builder mapping.

function step(id: string, o: Partial<WorkflowStep> = {}): WorkflowStep {
	return { id, title: id, detail: '', ownerKind: 'agent', owner: 'Bot', hoursPerWeek: 0, tools: [], edgeLabel: null, leakUsd: null, automation: null, branch: null, ...o };
}

describe('buildWorkflowTree', () => {
	it('is empty for an empty list', () => {
		expect(buildWorkflowTree([])).toEqual({ root: null, nodeCount: 0, maxDepth: 0 });
	});
	it('threads the default topology into one spine, in order', () => {
		const t = buildWorkflowTree([step('a'), step('b'), step('c'), step('d')]);
		expect(t.nodeCount).toBe(4);
		expect(t.maxDepth).toBe(3);
		const ids: string[] = [];
		let n = t.root;
		while (n) {
			ids.push(n.step.id);
			n = n.children[0] ?? null;
		}
		expect(ids).toEqual(['a', 'b', 'c', 'd']);
	});
	it('fans out real branches in source order', () => {
		const t = buildWorkflowTree([step('a'), step('b'), step('c'), step('d')], (i) => (i === 0 ? null : i < 3 ? 0 : 1));
		expect(t.root!.children.map((c) => c.step.id)).toEqual(['b', 'c']);
		expect(t.root!.children[0].children.map((c) => c.step.id)).toEqual(['d']);
		expect(t.maxDepth).toBe(2);
	});
	it('drops steps unreachable from the root', () => {
		const t = buildWorkflowTree([step('a'), step('b'), step('c')], (i) => (i === 0 ? null : i === 1 ? 0 : 5));
		expect(t.nodeCount).toBe(2);
	});
});

describe('workflowStepParent', () => {
	it('forks siblings on a shared branch.from and keeps the default after', () => {
		const steps = [
			step('triage'),
			step('noise', { branch: { from: 'triage', condition: 'noise' } }),
			step('deal', { branch: { from: 'triage', condition: 'deal' } }),
			step('send')
		];
		const t = buildWorkflowTree(steps, workflowStepParent);
		expect(t.root!.children.map((c) => c.step.id)).toEqual(['noise', 'deal']);
		expect(t.root!.children[1].children.map((c) => c.step.id)).toEqual(['send']);
	});
	it('overrides the previous slot when branch.from points further back', () => {
		const steps = [step('approve'), step('send', { branch: { from: 'approve', condition: 'approved' } }), step('track'), step('revise', { branch: { from: 'approve', condition: 'rejected' } })];
		const t = buildWorkflowTree(steps, workflowStepParent);
		expect(t.root!.children.map((c) => c.step.id)).toEqual(['send', 'revise']);
		expect(t.root!.children[0].children.map((c) => c.step.id)).toEqual(['track']);
	});
	it('falls back to the previous slot for an unknown branch.from', () => {
		const t = buildWorkflowTree([step('a'), step('b', { branch: { from: 'nope', condition: 'x' } })], workflowStepParent);
		expect(t.root!.children.map((c) => c.step.id)).toEqual(['b']);
	});
});

describe('workflowToolIds', () => {
	it('dedupes in first-seen order', () => {
		expect(workflowToolIds([step('a', { tools: ['gmail', 'telegram'] }), step('b', { tools: ['gmail'] }), step('c', { tools: ['attio', 'telegram'] })])).toEqual(['gmail', 'telegram', 'attio']);
	});
});

describe('row + step helpers', () => {
	const job = (o: Partial<JobRow>): JobRow => ({ id: 'j', description: 'd', agentId: 'a', agentName: 'A', unknownAgent: false, schedule: '* * * * *', scheduleLabel: '', enabled: true, runs: 0, ok: 0, lastRunAt: null, lastOk: null, nextRunAt: null, overdue: false, history: [], lastSummary: null, ...o });
	it('job state follows ScheduledTasks: off, missing agent, overdue, last outcome', () => {
		expect(jobState(job({ enabled: false, unknownAgent: true }))).toBe('off');
		expect(jobState(job({ unknownAgent: true }))).toBe('err');
		expect(jobState(job({ overdue: true }))).toBe('warn');
		expect(jobState(job({ lastOk: false }))).toBe('err');
		expect(jobState(job({ lastOk: null }))).toBe('ok');
	});
	it('relative clocks', () => {
		const now = Date.parse('2026-09-30T12:00:00Z');
		expect(ago(null, now)).toBe('never');
		expect(ago('2026-09-30T11:30:00Z', now)).toBe('30m ago');
		expect(untilLabel('2026-09-30T14:10:00Z', now)).toBe('2h 10m');
		expect(untilLabel('2026-09-30T11:00:00Z', now)).toBeNull();
	});
	it('step tones use status tokens only', () => {
		expect(stepTone(step('a', { automation: { title: 't', state: 'live', recoveredUsd: 0 } })).color).toBe('var(--bn-accent)');
		expect(stepTone(step('a', { ownerKind: 'human' })).word).toBe('human step');
	});
});

describe('builder mapping', () => {
	const agents = [{ id: 'crm-pulse', name: 'CRM Pulse' }];
	it('validates like the FounderOS v1 builder', () => {
		const s = emptyStep();
		expect(validateBuilder('', [s])).toBe('Give the workflow a name.');
		expect(validateBuilder('x', [])).toBe('A workflow needs at least one step.');
		expect(validateBuilder('x', [s])).toBe('Step 1 needs a title.');
		expect(validateBuilder('x', [{ ...s, title: 't', detail: 'd' }])).toBe('Step 1 needs an owner.');
		expect(validateBuilder('x', [{ ...s, title: 't', detail: 'd', ownerAgentId: 'crm-pulse' }])).toBeNull();
	});
	it('payload resolves owners and branch keys to indexes', () => {
		const a = { ...emptyStep(), title: ' A ', detail: 'd', ownerAgentId: 'crm-pulse' };
		const b = { ...emptyStep(), title: 'B', detail: 'd', ownerKind: 'human' as const, ownerHumanName: 'Alex', branchFromKey: a.key, branchCondition: 'approved', hoursPerWeek: '2.5' };
		const p = builderPayload(' Flow ', '', [a, b], agents);
		expect(p.name).toBe('Flow');
		expect(p.steps[0]).toMatchObject({ title: 'A', owner: 'CRM Pulse', branchFromIndex: null });
		expect(p.steps[1]).toMatchObject({ owner: 'Alex', hoursPerWeek: 2.5, branchFromIndex: 0, branchCondition: 'approved' });
	});
	it('a draft owner matching the roster becomes that agent', () => {
		const steps = stepsFromDraft(
			{ name: 'n', subtitle: '', steps: [
				{ title: 't', detail: 'd', ownerKind: 'human', owner: 'crm pulse', hoursPerWeek: 1, tools: [], automation: null, branchFromIndex: null, branchCondition: null },
				{ title: 'u', detail: 'd', ownerKind: 'human', owner: 'Sam', hoursPerWeek: 1, tools: [], automation: null, branchFromIndex: 0, branchCondition: 'yes' }
			] },
			agents
		);
		expect(steps[0]).toMatchObject({ ownerKind: 'agent', ownerAgentId: 'crm-pulse' });
		expect(steps[1]).toMatchObject({ ownerHumanName: 'Sam', branchFromKey: steps[0].key });
	});
	it('editing keeps step ids as keys so branches survive', () => {
		const wf = { id: 'w', name: 'W', subtitle: '', revenueUsd: 0, order: 0, steps: [step('s1', { owner: 'CRM Pulse' }), step('s2', { branch: { from: 's1', condition: 'c' } })] };
		const s = stepsFromWorkflow(wf, agents);
		expect(s[0].ownerAgentId).toBe('crm-pulse');
		expect(builderPayload('W', '', s, agents).steps[1].branchFromIndex).toBe(0);
	});
});

describe('toolBrand (FounderOS v1 lib/workflow-tool-brands.ts)', () => {
	it('maps every known tool id to a slug the brand-logo set can draw', () => {
		expect(toolBrand('calendar')).toEqual({ slug: 'googlecalendar', name: 'Google Calendar' });
		expect(toolBrand('zernio')).toEqual({ slug: 'zernio', name: 'Zernio' });
		// the Connections set, else the process map's own simple-icons glyphs (tool-marks.ts)
		for (const id of ['arcads', 'calendar', 'fathom', 'gmail', 'manychat', 'notion', 'skool', 'slack', 'stripe', 'trakyo', 'typeform', 'zernio', 'instagram', 'youtube', 'telegram']) {
			const slug = toolBrand(id).slug;
			expect(brandMarkKind(slug) !== 'none' || slug in TOOL_ICONS, id).toBe(true);
		}
	});
	it('an unknown id degrades to its own identity (a lettermark)', () => {
		expect(toolBrand('mystery')).toEqual({ slug: 'mystery', name: 'mystery' });
	});
});

describe('relativeTime (the step drawer clock, prod rounds)', () => {
	it('rounds minutes, hours and days the way prod does', () => {
		const now = Date.parse('2026-10-01T12:00:00Z');
		expect(relativeTime('2026-10-01T11:59:50Z', now)).toBe('just now');
		expect(relativeTime('2026-10-01T11:30:00Z', now)).toBe('30m ago');
		expect(relativeTime('2026-10-01T09:20:00Z', now)).toBe('3h ago');
		expect(relativeTime('2026-09-19T00:00:00Z', now)).toBe('13d ago');
	});
});
