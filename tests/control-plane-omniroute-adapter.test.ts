import { WorkflowArtifactKindSchema, InvestmentWorkflowStateSchema, InvalidationKindSchema } from '@/lib/control-plane/schema';
import {
  describe,
  expect,
  test,
} from 'vitest';

import {
  OmniRouteModelAdapter,
} from '@/lib/control-plane/omniroute-model-adapter';

import type {
  AgentToolInvoker,
} from '@/lib/control-plane/agent-tool-runtime';

import {
  EnvironmentRuntimeSecretSource,
} from '@/lib/control-plane/runtime-secret-source';

import {
  getCompanyAgentRuntimeProfile,
} from '@/lib/control-plane/agent-runtime-profile';


function jsonResponse(
  value:
    unknown,
): Response {
  return new Response(
    JSON.stringify(
      value,
    ),
    {
      status:
        200,

      headers: {
        'content-type':
          'application/json',
      },
    },
  );
}


describe(
  'OmniRoute model adapter',
  () => {
    test(
      'uses one explicitly assigned model and completes a governed tool loop',
      async () => {
        const requests:
          Array<{
            url:
              string;

            init?:
              RequestInit;
          }> = [];


        let turn =
          0;


        const fetchImpl:
          typeof fetch =
          async (
            url,
            init,
          ) => {
            requests.push({
              url:
                String(
                  url,
                ),

              init,
            });


            turn +=
              1;


            if (
              turn ===
              1
            ) {
              return jsonResponse({
                id:
                  'chatcmpl-1',

                model:
                  'groq/openai/gpt-oss-120b',

                choices: [
                  {
                    finish_reason:
                      'tool_calls',

                    message: {
                      role:
                        'assistant',

                      content:
                        null,

                      tool_calls: [
                        {
                          id:
                            'call-1',

                          type:
                            'function',

                          function: {
                            name:
                              'web-search',

                            arguments:
                              JSON.stringify({
                                query:
                                  'ABC latest filing',
                              }),
                          },
                        },
                      ],
                    },
                  },
                ],

                usage: {
                  prompt_tokens:
                    100,

                  completion_tokens:
                    20,
                },
              });
            }


            return jsonResponse({
              id:
                'chatcmpl-2',

              model:
                'groq/openai/gpt-oss-120b',

              choices: [
                {
                  finish_reason:
                    'tool_calls',

                  message: {
                    role:
                      'assistant',

                    content:
                      null,

                    tool_calls: [
                      {
                        id:
                          'final-1',

                        type:
                          'function',

                        function: {
                          name:
                            'submit_company_output',

                          arguments:
                            JSON.stringify({
                              summary:
                                'Research completed.',

                              operations: [
                                {
                                  type:
                                    'register_artifact',

                                  artifactKind:
                                    'research_proposal',

                                  summary:
                                    'ABC proposal.',

                                  metadata: {
                                    confidence:
                                      0.7,
                                  },
                                },
                              ],
                            }),
                        },
                      },
                    ],
                  },
                },
              ],

              usage: {
                prompt_tokens:
                  120,

                completion_tokens:
                  30,

                prompt_tokens_details: {
                  cached_tokens:
                    10,
                },

                completion_tokens_details: {
                  reasoning_tokens:
                    5,
                },
              },
            });
          };


        const toolCalls:
          string[] = [];


        const tools:
          AgentToolInvoker = {
          listAvailableTools() {
            return [
              {
                id:
                  'web-search',

                capability:
                  'web-research',

                description:
                  'Search the web.',

                effect:
                  'read',

                inputSchema: {
                  type:
                    'object',

                  properties: {
                    query: {
                      type:
                        'string',
                    },
                  },

                  required: [
                    'query',
                  ],
                },
              },
            ];
          },


          async invoke(
            toolId,
          ) {
            toolCalls.push(
              toolId,
            );


            return {
              summary:
                'Found filing.',

              data: {
                result:
                  true,
              },

              provenance:
                [],

              metadata:
                {},
            };
          },
        };


        const adapter =
          new OmniRouteModelAdapter({
            secretSource:
              new EnvironmentRuntimeSecretSource({
                OMNIROUTE_API_KEY:
                  'omniroute-test-key',
              }),

            model:
              'groq/openai/gpt-oss-120b',

            baseUrl:
              'http://localhost:20128/v1',

            fetchImpl,
          });


        const result =
          await adapter.execute({
            execution: {
              executionId:
                'exec-1',

              profile:
                getCompanyAgentRuntimeProfile(
                  'lauti',
                ),

              task: {
                taskId:
                  'task-1',

                handoffId:
                  'handoff-1',

                agentId:
                  'lauti',

                action:
                  'continue_research',

                summary:
                  'Research ABC.',

                priority:
                  'normal',

                status:
                  'running',

                attempts:
                  1,

                createdAt:
                  '2026-01-01T00:00:00.000Z',

                availableAt:
                  '2026-01-01T00:00:00.000Z',

                updatedAt:
                  '2026-01-01T00:00:00.000Z',

                retryPolicy: {
                  maxAttempts:
                    3,

                  initialDelayMs:
                    1_000,

                  backoffMultiplier:
                    2,

                  maxDelayMs:
                    10_000,

                  timeoutMs:
                    30_000,
                },

                policyEvidence:
                  [],

                payload:
                  {},

                result:
                  {},
              },

              workflow:
                null,

              brain: {
                route:
                  null,

                results:
                  [],

                truncated:
                  false,

                totalContentChars:
                  0,
              },

              availableTools:
                tools
                  .listAvailableTools(),

              idempotencyKey:
                'exec-1',

              constraints: {
                maxToolCalls:
                  5,

                maxInputTokens:
                  10_000,

                maxOutputTokens:
                  2_000,

                maxOutputChars:
                  20_000,

                timeoutMs:
                  30_000,

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
              new AbortController()
                .signal,
          });


        expect(
          toolCalls,
        ).toEqual([
          'web-search',
        ]);


        expect(
          result.output
            .summary,
        ).toBe(
          'Research completed.',
        );


        expect(
          result.usage,
        ).toMatchObject({
          inputTokens:
            220,

          outputTokens:
            50,

          cachedInputTokens:
            10,

          reasoningTokens:
            5,
        });


        expect(
          requests,
        ).toHaveLength(
          2,
        );


        expect(
          requests[0].url,
        ).toBe(
          'http://localhost:20128/v1/chat/completions',
        );


        const firstHeaders =
          new Headers(
            requests[0]
              .init
              ?.headers,
          );


        expect(
          firstHeaders.get(
            'authorization',
          ),
        ).toBe(
          'Bearer omniroute-test-key',
        );


        const firstBody =
          JSON.parse(
            String(
              requests[0]
                .init
                ?.body,
            ),
          );


        expect(
          firstBody.model,
        ).toBe(
          'groq/openai/gpt-oss-120b',
        );


        expect(
          firstBody.tool_choice,
        ).toBe(
          'auto',
        );

        const finalOutputDefinition =
          firstBody.tools.find(
            (tool: { function?: { name?: string } }) =>
              tool.function?.name === 'submit_company_output',
          );

        expect(finalOutputDefinition).toBeDefined();

        const finalOutputSchemaVariants =
          finalOutputDefinition.function.parameters
            .properties.operations.items.oneOf;

        const variant = (type: string) =>
          finalOutputSchemaVariants.find(
            (item: { properties: { type: { const: string } } }) =>
              item.properties.type.const === type,
          );

        expect(
          variant('register_artifact').properties.artifactKind.enum,
        ).toEqual(WorkflowArtifactKindSchema.options);

        expect(
          variant('request_transition').properties.fromState.enum,
        ).toEqual(InvestmentWorkflowStateSchema.options);

        expect(
          variant('request_transition').properties.toState.enum,
        ).toEqual(InvestmentWorkflowStateSchema.options);

        expect(
          variant('record_invalidation').properties.kind.enum,
        ).toEqual(InvalidationKindSchema.options);

        expect(
          variant('register_artifact').properties.artifactKind.enum,
        ).not.toContain('research_data_request');


        expect(
          firstBody.tools
            .some(
              (
                tool:
                  {
                    function?: {
                      name?:
                        string;
                    };
                  },
              ) =>
                tool.function
                  ?.name ===
                'submit_company_output',
            ),
        ).toBe(
          true,
        );


        const secondBody =
          JSON.parse(
            String(
              requests[1]
                .init
                ?.body,
            ),
          );


        expect(
          secondBody.messages
            .some(
              (
                message:
                  {
                    role?:
                      string;

                    tool_call_id?:
                      string;
                  },
              ) =>
                message.role ===
                  'tool' &&
                message
                  .tool_call_id ===
                  'call-1',
            ),
        ).toBe(
          true,
        );
      },
    );


    test(
      'refuses to start without an explicit model assignment',
      () => {
        expect(
          () =>
            new OmniRouteModelAdapter({
              secretSource:
                new EnvironmentRuntimeSecretSource({
                  OMNIROUTE_API_KEY:
                    'test-key',
                }),

              model:
                '   ',
            }),
        ).toThrow(
          'omniroute_model_assignment_required',
        );
      },
    );
  },
);