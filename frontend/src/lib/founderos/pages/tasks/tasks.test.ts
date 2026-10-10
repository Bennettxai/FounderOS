import { describe, expect, it } from 'vitest';
import { ADVANCE, ago, age, busiestDay, inNext, issueBody, issueTone, jobFailing, moveTask } from './tasks';
import type { AgentTask } from './types';

const NOW = Date.parse('2026-09-30T12:00:00Z');

describe('tasks helpers', () => {
	it('ages read the way the FounderOS v1 strips did', () => {
		expect(ago(null, NOW)).toBe('never');
		expect(ago('2026-09-30T11:55:00Z', NOW)).toBe('5m ago');
		expect(ago('2026-09-30T09:00:00Z', NOW)).toBe('3h ago');
		expect(ago('2026-09-28T12:00:00Z', NOW)).toBe('2d ago');
		expect(age('2026-09-30T09:00:00Z', NOW)).toBe('3h');
		expect(age(null, NOW)).toBe('');
		expect(inNext('2026-09-30T12:12:00Z', NOW)).toBe('12m');
		expect(inNext('2026-09-30T11:00:00Z', NOW)).toBeNull();
	});

	it('a job is failing on a failed run, a missed slot or a missing agent', () => {
		expect(jobFailing({ lastOk: false, overdue: false, unknownAgent: false })).toBe(true);
		expect(jobFailing({ lastOk: null, overdue: true, unknownAgent: false })).toBe(true);
		expect(jobFailing({ lastOk: null, overdue: false, unknownAgent: true })).toBe(true);
		expect(jobFailing({ lastOk: true, overdue: false, unknownAgent: false })).toBe(false);
	});

	it('the composer carries the routing chip as a description hint', () => {
		expect(issueBody('  Ship it ', 'Conductor routes')).toEqual({ title: 'Ship it' });
		expect(issueBody('Ship it', 'TECH')).toEqual({ title: 'Ship it', description: 'Route to the TECH pillar.' });
	});

	it('status tones are status tokens only', () => {
		expect(issueTone('blocked')).toBe('var(--bn-err)');
		expect(issueTone('todo')).toBe('var(--bn-text-2)');
	});

	it('kanban moves are optimistic and done is terminal', () => {
		const t: AgentTask[] = [{ id: 'a', agentId: 'x', title: 't', status: 'open', createdAt: '', updatedAt: '' }];
		expect(moveTask(t, 'a', 'doing')[0].status).toBe('doing');
		expect(ADVANCE.done).toBeUndefined();
	});

	it('busiest weekday is null on a quiet window', () => {
		expect(busiestDay([{ label: 'Mon', count: 0 }])).toBeNull();
		expect(busiestDay([{ label: 'Mon', count: 1 }, { label: 'Tue', count: 3 }])).toEqual({ label: 'Tue', count: 3 });
	});
});
