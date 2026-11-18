import { afterEach, describe, expect, test } from 'vitest';
import { mkdtempSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { GovernedAgentExecutorRunner, type AgentExecutor } from '@/lib/control-plane/agent-executor';
import { SqliteInternalRuntimeStore } from '@/lib/control-plane/sqlite-runtime-store';

let tmp: string | null = null;
afterEach(() => {
  if (tmp) rmSync(tmp, { recursive: true, force: true });
  tmp = null;
});
function dbPath(): string {
  tmp = mkdtempSync(path.join(tmpdir(), 'startup-plan-retry-'));
  return path.join(tmp, 'cp.db');
}
async function seed(store: SqliteInternalRuntimeStore, workflowId: string) {
  const at = '2026-01-01T00:00:00.000Z';
  await store.createWorkflow({
    id: workflowId, state: 'RESEARCHING', revision: 0,
    createdBy: { kind: 'agent', id: 'lauti' },
    createdAt: at, updatedAt: at, responsibleResearchAgentId: 'lauti',
    artifacts: [], invalidations: [], metadata: {},
  });
  store.upsertAgentRuntimeTask({
    taskId: 'retry-task', handoffId: 'retry-handoff', workflowId,
    agentId: 'lauti', action: 'continue_research', summary: 'Governed research.',
    stateAtAssignment: 'RESEARCHING', createdAt: at, availableAt: at, updatedAt: at,
    retryPolicy: { maxAttempts: 3, initialDelayMs: 0, backoffMultiplier: 1, maxDelayMs: 0, timeoutMs: 30_000 },
    policyEvidence: [], payload: {},
  });
}
function makeRunner(store: SqliteInternalRuntimeStore, executor: AgentExecutor) {
  return new GovernedAgentExecutorRunner(
    store, executor,
    { workerId: 'plan-worker', batchSize: 1, leaseMs: 10_000 },
    () => '2026-01-01T00:00:10.000Z',
  );
}
class FailCompletionOnceStore extends SqliteInternalRuntimeStore {
  private failOnce = true;
  override completeAgentRuntimeTask(
    input: Parameters<SqliteInternalRuntimeStore['completeAgentRuntimeTask']>[0],
  ) {
    if (this.failOnce) {
      this.failOnce = false;
      throw new Error('temporary_task_completion_failure');
    }
    return super.completeAgentRuntimeTask(input);
  }
}

describe('durable plan and Control Plane retry safety', () => {
  test('missing research_proposal is denied once with its original audit reason', async () => {
    const store = new SqliteInternalRuntimeStore(dbPath());
    await seed(store, 'wf-denied');
    let calls = 0;
    const executor: AgentExecutor = {
      id: 'fake-denied', agentId: 'lauti',
      async execute() {
        calls++;
        return { summary: 'Advance without artifact.', operations: [{
          type: 'request_transition', fromState: 'RESEARCHING',
          toState: 'READY_FOR_RED_DESK', reason: 'Invalid stage progression.',
        }] };
      },
    };
    expect(await makeRunner(store, executor).runOnce())
      .toEqual({ claimed: 1, completed: 0, failed: 1 });
    const task = store.listAgentRuntimeTasks()[0]!;
    expect(task.status).toBe('dead_letter');
    expect(task.attempts).toBe(1);
    expect(task.lastError).toContain('research_proposal_required');
    expect(calls).toBe(1);
    expect(store.getAgentRuntimePlan('retry-task')).not.toBeNull();
    const workflow = await store.getWorkflow('wf-denied');
    expect(workflow?.state).toBe('RESEARCHING');
    expect(workflow?.artifacts).toHaveLength(0);
    store.close();
  });

  test('retries an already applied plan without re-calling the model or duplicating CP mutations', async () => {
    const store = new FailCompletionOnceStore(dbPath());
    await seed(store, 'wf-resume');
    let calls = 0;
    const executor: AgentExecutor = {
      id: 'fake-resume', agentId: 'lauti',
      async execute() {
        calls++;
        return { summary: 'Prepared governed research.', operations: [
          { type: 'register_artifact', artifactKind: 'research_proposal',
            summary: 'Evidence-backed test proposal.', metadata: { test: true } },
          { type: 'request_transition', fromState: 'RESEARCHING',
            toState: 'READY_FOR_RED_DESK', reason: 'Research proposal registered.' },
        ] };
      },
    };
    const runner = makeRunner(store, executor);
    expect(await runner.runOnce()).toEqual({ claimed: 1, completed: 0, failed: 1 });
    expect(store.listAgentRuntimeTasks()[0]?.status).toBe('queued');
    const firstWorkflow = await store.getWorkflow('wf-resume');
    expect(firstWorkflow?.state).toBe('READY_FOR_RED_DESK');
    expect(firstWorkflow?.artifacts).toHaveLength(1);
    expect(store.getAgentRuntimePlan('retry-task')).not.toBeNull();
    expect(await runner.runOnce()).toEqual({ claimed: 1, completed: 1, failed: 0 });
    const secondWorkflow = await store.getWorkflow('wf-resume');
    expect(secondWorkflow?.state).toBe('READY_FOR_RED_DESK');
    expect(secondWorkflow?.artifacts).toHaveLength(1);
    expect(secondWorkflow?.revision).toBe(2);
    expect(store.listAgentRuntimeTasks()[0]?.status).toBe('completed');
    expect(calls).toBe(1);
    store.close();
  });
});
