import type {
  ProvenanceRef,
} from '@/lib/brain/core-schema';

import {
  AgentExecutorOutputSchema,
  type AgentExecutor,
  type AgentExecutorContext,
  type AgentExecutorOutputInput,
} from '@/lib/control-plane/agent-executor';

import {
  GovernedAgentToolRuntime,
  type AgentToolCallTrace,
  type AgentReadToolRegistry,
} from '@/lib/control-plane/agent-tool-runtime';

import type {
  AgentCredentialVault,
} from '@/lib/control-plane/credential-vault';

import type {
  AgentBrainContextProvider,
} from '@/lib/control-plane/brain-context-provider';

import {
  summarizeAgentKnowledgeContext,
} from '@/lib/control-plane/knowledge-context-observability';

import {
  normalizeAgentModelExecutionResult,
  type AgentModelAdapter,
} from '@/lib/control-plane/agent-model-adapter';

import type {
  CompanyAgentRuntimeProfile,
} from '@/lib/control-plane/agent-runtime-profile';

type RuntimeEvidence = {
  brainNodeIds: string[];

  knowledgeContext:
    ReturnType<
      typeof summarizeAgentKnowledgeContext
    >;

  externalProvenance:
    ProvenanceRef[];

  toolCalls:
    AgentToolCallTrace[];
};

function attachEvidence(
  output:
    AgentExecutorOutputInput,

  evidence:
    RuntimeEvidence,
): AgentExecutorOutputInput {
  const parsed =
    AgentExecutorOutputSchema.parse(
      output,
    );

  return {
    ...parsed,

    operations:
      parsed.operations.map(
        (operation) => {
          if (
            operation.type ===
              'register_artifact' ||
            operation.type ===
              'record_invalidation'
          ) {
            return {
              ...operation,

              metadata: {
                ...operation.metadata,

                runtimeEvidence:
                  evidence,
              },
            };
          }

          return operation;
        },
      ),
  };
}

function outputChars(
  output:
    AgentExecutorOutputInput,
): number {
  return JSON.stringify(
    output,
  ).length;
}

/**
 * Provider-independent AgentExecutor.
 *
 * It assembles governed context, Brain retrieval and read-only tools around a
 * model/framework adapter. The resulting structured operations are then passed
 * to the V2G runner, which replays them through the Control Plane.
 */
export class BoundedModelAgentExecutor
  implements AgentExecutor
{
  readonly id: string;
  readonly agentId: string;

  constructor(
    private readonly profile:
      CompanyAgentRuntimeProfile,

    private readonly modelAdapter:
      AgentModelAdapter,

    private readonly brainContext:
      AgentBrainContextProvider,

    private readonly toolRegistry:
      AgentReadToolRegistry,

    private readonly clock:
      () => string = () =>
        new Date().toISOString(),

    private readonly credentialVault?:
      AgentCredentialVault,
  ) {
    this.agentId =
      profile.agentId;

    this.id =
      `bounded-model:${profile.agentId}:${modelAdapter.id}`;
  }

  async execute(
    context:
      AgentExecutorContext,
  ): Promise<
    AgentExecutorOutputInput
  > {
    if (
      context.task.agentId !==
      this.profile.agentId
    ) {
      throw new Error(
        `agent_executor_task_identity_mismatch:${this.profile.agentId}:${context.task.agentId}`,
      );
    }

    const controller =
      new AbortController();

    const timer =
      setTimeout(
        () =>
          controller.abort(
            'agent_execution_timeout',
          ),
        this.profile.budget
          .timeoutMs,
      );

    try {
      const executionId =
        context.idempotencyKey;

      const brain =
        await this
          .brainContext
          .retrieve({
            profile:
              this.profile,

            task:
              context.task,

            workflow:
              context.workflow,

            requestId:
              `${executionId}:brain-read`,
          });

      if (
        controller.signal
          .aborted
      ) {
        throw new Error(
          'agent_execution_timeout',
        );
      }

      const tools =
        new GovernedAgentToolRuntime(
          this.profile,
          context.task,
          this.toolRegistry,
          controller.signal,
          this.clock,
          this.credentialVault,
        );

      const modelPromise =
        this.modelAdapter
          .execute({
            execution: {
              executionId,

              profile:
                this.profile,

              task:
                context.task,

              workflow:
                context.workflow,

              brain,

              availableTools:
                tools
                  .listAvailableTools(),

              idempotencyKey:
                context
                  .idempotencyKey,

              constraints: {
                maxToolCalls:
                  this.profile
                    .budget
                    .maxToolCalls,

                maxInputTokens:
                  this.profile
                    .budget
                    .maxInputTokens,

                maxOutputTokens:
                  this.profile
                    .budget
                    .maxOutputTokens,

                maxOutputChars:
                  this.profile
                    .budget
                    .maxOutputChars,

                timeoutMs:
                  this.profile
                    .budget
                    .timeoutMs,

                hiddenChainOfThoughtMustNotBePersisted:
                  true,

                financialExecutionToolsAvailable:
                  false,

                workflowMutationOnlyThroughControlPlane:
                  true,
              },
            },

            tools,

            signal:
              controller.signal,
          });

      const timeoutPromise =
        new Promise<never>(
          (_, reject) => {
            controller.signal
              .addEventListener(
                'abort',
                () =>
                  reject(
                    new Error(
                      'agent_execution_timeout',
                    ),
                  ),
                {
                  once: true,
                },
              );
          },
        );

      const modelResult =
        normalizeAgentModelExecutionResult(
          await Promise.race([
            modelPromise,
            timeoutPromise,
          ]),
        );

      if (
        modelResult.usage
          .inputTokens >
        this.profile.budget
          .maxInputTokens
      ) {
        throw new Error(
          `agent_model_input_token_budget_exceeded:${modelResult.usage.inputTokens}`,
        );
      }

      if (
        modelResult.usage
          .outputTokens >
        this.profile.budget
          .maxOutputTokens
      ) {
        throw new Error(
          `agent_model_output_token_budget_exceeded:${modelResult.usage.outputTokens}`,
        );
      }

      if (
        outputChars(
          modelResult.output,
        ) >
        this.profile.budget
          .maxOutputChars
      ) {
        throw new Error(
          'agent_model_output_character_budget_exceeded',
        );
      }

      const evidence:
        RuntimeEvidence = {
        brainNodeIds:
          brain.results.map(
            (result) =>
              result.node.id,
          ),

        knowledgeContext:
          summarizeAgentKnowledgeContext(
            brain,
          ),

        externalProvenance:
          tools
            .getProvenance(),

        toolCalls:
          tools
            .getTraces(),
      };

      const enriched =
        attachEvidence(
          modelResult.output,
          evidence,
        );

      return AgentExecutorOutputSchema.parse({
        ...enriched,

        metadata: {
          ...enriched.metadata,

          runtime: {
            executorId:
              this.id,

            modelAdapterId:
              this.modelAdapter.id,

            model:
              modelResult.model,

            finishReason:
              modelResult
                .finishReason,

            usage:
              modelResult.usage,

            toolCallCount:
              tools
                .getCallCount(),

            brainNodeIds:
              evidence
                .brainNodeIds,

            knowledgeContext:
              evidence
                .knowledgeContext,

            externalProvenance:
              evidence
                .externalProvenance,

            toolCalls:
              evidence
                .toolCalls,

            modelMetadata:
              modelResult
                .metadata,

            hiddenChainOfThoughtStored:
              false,
          },
        },
      });
    } finally {
      clearTimeout(
        timer,
      );
    }
  }
}


