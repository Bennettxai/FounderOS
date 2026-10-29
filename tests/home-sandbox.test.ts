import { describe, expect, test } from 'vitest';
import { previewSandboxTasks, sandboxSpeed } from '@/lib/home-sandbox';
import type { Agent, Department, SopTask } from '@/lib/schemas';
import { readFileSync } from 'node:fs';

const departments: Department[] = [{ id: 'ops', name: 'Operations', slug: 'ops', tagline: '', color: 'mint', order: 1 }];
const agents: Agent[] = [
  { id: 'atlas', departmentId: 'ops', name: 'Atlas', role: 'Planner', status: 'active', tier: 'specialist', description: '', model: '', tools: [], parentId: null, instance: 'builtin' },
  { id: 'quill', departmentId: 'ops', name: 'Quill', role: 'Writer', status: 'active', tier: 'specialist', description: '', model: '', tools: [], parentId: null, instance: 'builtin' },
  { id: 'flint', departmentId: 'ops', name: 'Flint', role: 'Analyst', status: 'active', tier: 'specialist', description: '', model: '', tools: [], parentId: null, instance: 'builtin' },
  { id: 'unused', departmentId: 'ops', name: 'Unused', role: 'Worker', status: 'active', tier: 'specialist', description: '', model: '', tools: [], parentId: null, instance: 'builtin' },
];
const task = (id: string, assigneeId: string, title: string): SopTask => ({
  id, departmentId: 'ops', title, summary: '', steps: ['one', 'two', 'three'], assigneeKind: 'agent', assigneeId,
});
const tasks = [
  task('one', 'atlas', 'Map the next launch'),
  task('two', 'quill', 'Draft the weekly brief'),
  task('three', 'flint', 'Review campaign signals'),
  task('four', 'unused', 'Prepare an extra task'),
];

describe('landing sandbox preview', () => {
  test('uses three one-task-per-agent records from the team map', () => {
    expect(previewSandboxTasks(tasks, agents, departments)).toEqual([
      expect.objectContaining({ agent: 'Atlas', department: 'Operations', task: 'Map the next launch' }),
      expect.objectContaining({ agent: 'Quill', department: 'Operations', task: 'Draft the weekly brief' }),
      expect.objectContaining({ agent: 'Flint', department: 'Operations', task: 'Review campaign signals' }),
    ]);
  });

  test('relative bars are static and normalize 57 ms as the fastest sample', () => {
    const speed = sandboxSpeed();
    expect(speed[0]).toMatchObject({ latencyMs: 57, widthPct: 100 });
    expect(speed[1].widthPct).toBeLessThan(100);
    expect(speed[2].widthPct).toBeLessThan(speed[1].widthPct);
  });

  test('landing preview is labeled simulated and credits Miosa, while the existing composer still posts to Interject', () => {
    const home = readFileSync('components/Home6Console.tsx', 'utf8');
    const composer = readFileSync('components/home/Interject.tsx', 'utf8');
    expect(home).toContain('Simulated sandbox activity');
    expect(home).toContain('Powered by Miosa.');
    expect(composer).toContain("fetch('/api/interject'");
  });
});
