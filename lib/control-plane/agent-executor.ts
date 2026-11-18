import { z, ZodError } from 'zod';

import {
  InvalidationKindSchema,
  InvestmentWorkflowStateSchema,
  WorkflowArtifactKindSchema,
  type InvestmentWorkflow,
} from '@/lib/control-plane/schema';

import {
  AtomicCompanyControlPlane,
} from '@/lib/control-plane/atomic-engine';

import type {
  AtomicControlPlaneStore,
} from '@/lib/control-plane/atomic-store';

import type {
  InternalRuntimeStore,
} from '@/lib/control-plane/runtime-store';

import type {
  AgentRuntimeTask,
} from '@/lib/control-plane/runtime-schema';

export const AgentExecutorArtifactOperationSchema = z.object({
  type: z.literal('register_artifact'),
  artifactKind: WorkflowArtifactKindSchema,
  summary: z.string().min(1),
  metadata: z.record(z.unknown()).default({}),
});

export const AgentExecutorTransitionOperationSchema = z.object({
  type: z.literal('request_transition'),
  fromState: InvestmentWorkflowStateSchema,
  toState: InvestmentWorkflowStateSchema,
  reason: z.string().min(1),
});

export const AgentExecutorInvalidationOperationSchema = z.object({
  type: z.literal('record_invalidation'),
  kind: InvalidationKindSchema,
  reason: z.string().min(1),
  metadata: z.record(z.unknown()).default({}),
});

export const AgentExecutorResolveInvalidationOperationSchema = z.object({
  type: z.literal('resolve_invalidation'),
  invalidationId: z.string().min(1),
  reason: z.string().min(1),
});

export const AgentExecutorOperationSchema = z.discriminatedUnion(
  'type',
  [
    AgentExecutorArtifactOperationSchema,
    AgentExecutorTransitionOperationSchema,
    AgentExecutorInvalidationOperationSchema,
    AgentExecutorResolveInvalidationOperationSchema,
  ],
);

export const AgentExecutorOutputSchema = z.object({
  summary: z.string().min(1),
  operations: z
    .array(AgentExecutorOperationSchema)
    .max(20)
    .default([]),
  metadata: z.record(z.unknown()).default({}),
});

export type AgentExecutorOutput = z.infer<
  typeof AgentExecutorOutputSchema
>;

export type AgentExecutorOutputInput = z.input<
  typeof AgentExecutorOutputSchema
>;

export type AgentExecutorContext = {
  task: AgentRuntimeTask;
  workflow: InvestmentWorkflow | null;

  /**
   * Stable execution key. Future provider adapters should propagate this
   * where the provider supports idempotency.
   */
  idempotencyKey: string;
};

export interface AgentExecutor {
  id: string;
  agentId: string;

  execute(
    context: AgentExecutorContext,
  ): Promise<AgentExecutorOutputInput>;
}

export type AgentExecutorRunnerOptions = {
  workerId: string;
  batchSize: number;
  leaseMs: number;

  principalSessionId?: string;
};

export type AgentExecutorRunnerResult = {
  claimed: number;
  completed: number;
  failed: number;
};

type GovernedAgentRuntimeStore =
  InternalRuntimeStore &
  AtomicControlPlaneStore;

function errorMessage(
  error: unknown,
): string {
  return error instanceof Error
    ? error.message
    : String(error);
}

function isPermanentControlPlaneError(error: unknown): boolean {
  if (error instanceof ZodError) return true;
  const message = errorMessage(error);
  return [
    'workflow_gates_not_satisfied',
    'transition_shape_denied',
    'request_id_reuse_conflict:',
    'ceo_required_for_transition',
    'artifact_author_denied:',
    'principal_not_authorized',
    'principal_department_not_authorized',
    'non_agent_principal_not_authorized',
    'artifact_workflow_id_mismatch',
    'artifact_creator_must_match_request_principal',
    'agent_task_workflow_required:',
    'transition_not_allowed:',
    'workflow_state_is_',
  ].some((prefix) => message.startsWith(prefix));
}

function operationRequestId(
  taskId: string,
  operationIndex: number,
  suffix: string,
): string {
  return `agent-task:${taskId}:op:${operationIndex}:${suffix}`;
}

/**
 * Model-independent agent runtime.
 *
 * Executors never mutate workflow state or write artifacts directly.
 * They return a structured list of proposed operations. Every operation is
 * replayed through AtomicCompanyControlPlane under the identity of the
 * assigned agent, so existing authorship, permission and gate rules remain
 * authoritative.
 *
 * Deliberately absent from the output schema: financial execution actions.
 */
export class GovernedAgentExecutorRunner {
  private readonly controlPlane:
    AtomicCompanyControlPlane;

  constructor(
    private readonly store:
      GovernedAgentRuntimeStore,
    private readonly executor:
      AgentExecutor,
    private readonly options:
      AgentExecutorRunnerOptions,
    private readonly clock: () => string = () =>
      new Date().toISOString(),
  ) {
    if (!executor.id.trim()) {
      throw new Error(
        'agent_executor_id_required',
      );
    }

    if (!executor.agentId.trim()) {
      throw new Error(
        'agent_executor_agent_id_required',
      );
    }

    if (!options.workerId.trim()) {
      throw new Error(
        'agent_executor_worker_id_required',
      );
    }

    if (
      options.batchSize <= 0 ||
      options.leaseMs <= 0
    ) {
      throw new Error(
        'agent_executor_runner_limits_must_be_positive',
      );
    }

    this.controlPlane =
      new AtomicCompanyControlPlane(
        store,
        this.clock,
      );
  }

  async runOnce():
    Promise<AgentExecutorRunnerResult> {
    const now =
      this.clock();

    const tasks =
      this.store.claimAgentRuntimeTasks({
        agentId:
          this.executor.agentId,
        workerId:
          this.options.workerId,
        now,
        limit:
          this.options.batchSize,
        leaseMs:
          this.options.leaseMs,
      });

    const result:
      AgentExecutorRunnerResult = {
      claimed: tasks.length,
      completed: 0,
      failed: 0,
    };

    for (const task of tasks) {
      try {
        const workflow =
          task.workflowId
            ? await this.store.getWorkflow(
                task.workflowId,
              )
            : null;

        const savedPlan = this.store.getAgentRuntimePlan(task.taskId);
        const output = AgentExecutorOutputSchema.parse(
          savedPlan === null
            ? await this.executor.execute({
                task,
                workflow,
                idempotencyKey: `agent-execution:${task.taskId}`,
              })
            : savedPlan,
        );
        // Persist the exact normalized model output before CP mutation.
        // On retry, replay this plan rather than querying the model again.
        if (savedPlan === null) {
          this.store.saveAgentRuntimePlan({
            taskId: task.taskId,
            workerId: this.options.workerId,
            savedAt: this.clock(),
            plan: output,
          });
        }

        const appliedOperations:
          Array<{
            index: number;
            type: string;
            requestId: string;
          }> = [];

        for (
          let index = 0;
          index <
          output.operations.length;
          index += 1
        ) {
          const operation =
            output.operations[index];

          const requestId =
            await this.applyOperation(
              task,
              operation,
              index,
            );

          appliedOperations.push({
            index,
            type: operation.type,
            requestId,
          });
        }

        this.store.completeAgentRuntimeTask({
          taskId:
            task.taskId,
          workerId:
            this.options.workerId,
          completedAt:
            this.clock(),
          result: {
            executorId:
              this.executor.id,
            summary:
              output.summary,
            metadata:
              output.metadata,
            appliedOperations,
          },
        });

        result.completed += 1;
      } catch (error) {
        this.store.failAgentRuntimeTask({
          taskId:
            task.taskId,
          workerId:
            this.options.workerId,
          failedAt:
            this.clock(),
          error:
            errorMessage(error),
          retryable: !isPermanentControlPlaneError(error),
        });

        result.failed += 1;
      }
    }

    return result;
  }

  private async requireFreshWorkflow(
    task: AgentRuntimeTask,
  ): Promise<InvestmentWorkflow> {
    if (!task.workflowId) {
      throw new Error(
        `agent_task_workflow_required:${task.taskId}`,
      );
    }

    const workflow =
      await this.store.getWorkflow(
        task.workflowId,
      );

    if (!workflow) {
      throw new Error(
        `workflow_not_found:${task.workflowId}`,
      );
    }

    return workflow;
  }

  private async applyOperation(
    task: AgentRuntimeTask,
    operation: AgentExecutorOutput['operations'][number],
    index: number,
  ): Promise<string> {
    const principal = {
      actor: {
        kind: 'agent' as const,
        id: task.agentId,
      },

      sessionId:
        this.options
        .principalSessionId
    };

    switch (operation.type) {
      case 'register_artifact': {
        const workflow =
          await this.requireFreshWorkflow(
            task,
          );

        const requestId =
          operationRequestId(
            task.taskId,
            index,
            'artifact',
          );

        await this.controlPlane.registerArtifact({
          requestId,
          workflowId:
            workflow.id,
          principal,
          expectedRevision:
            workflow.revision,
          artifact: {
            id:
              `artifact:${task.taskId}:${index}`,
            workflowId:
              workflow.id,
            kind:
              operation.artifactKind,
            createdBy:
              principal.actor,
            createdAt:
              this.clock(),
            summary:
              operation.summary,
            status: 'active',
            metadata: {
              ...operation.metadata,
              sourceRuntimeTaskId:
                task.taskId,
              executorId:
                this.executor.id,
            },
          },
          reason:
            `Agent runtime task ${task.taskId} submitted a governed artifact.`,
        });

        return requestId;
      }

      case 'request_transition': {
        const workflow =
          await this.requireFreshWorkflow(
            task,
          );

        const requestId =
          operationRequestId(
            task.taskId,
            index,
            'transition',
          );

        await this.controlPlane.transition({
          requestId,
          workflowId:
            workflow.id,
          principal,
          fromState:
            operation.fromState,
          toState:
            operation.toState,
          expectedRevision:
            workflow.revision,
          reason:
            operation.reason,
        });

        return requestId;
      }

      case 'record_invalidation': {
        const workflow =
          await this.requireFreshWorkflow(
            task,
          );

        const requestId =
          operationRequestId(
            task.taskId,
            index,
            'invalidation',
          );

        await this.controlPlane.recordInvalidation({
          requestId,
          workflowId:
            workflow.id,
          principal,
          expectedRevision:
            workflow.revision,
          invalidation: {
            id:
              `invalidation:${task.taskId}:${index}`,
            workflowId:
              workflow.id,
            kind:
              operation.kind,
            status: 'active',
            reason:
              operation.reason,
            recordedBy:
              principal.actor,
            recordedAt:
              this.clock(),
            metadata: {
              ...operation.metadata,
              sourceRuntimeTaskId:
                task.taskId,
              executorId:
                this.executor.id,
            },
          },
          reason:
            operation.reason,
        });

        return requestId;
      }

      case 'resolve_invalidation': {
        const workflow =
          await this.requireFreshWorkflow(
            task,
          );

        const requestId =
          operationRequestId(
            task.taskId,
            index,
            'resolve-invalidation',
          );

        await this.controlPlane.resolveInvalidation({
          requestId,
          workflowId:
            workflow.id,
          principal,
          expectedRevision:
            workflow.revision,
          invalidationId:
            operation.invalidationId,
          reason:
            operation.reason,
        });

        return requestId;
      }
    }
  }
}
