import {
  describe,
  expect,
  test,
} from 'vitest';

import {
  BrainGraphNodeSchema,
} from '@/lib/brain/graph-schema';

import type {
  AgentBrainContext,
} from '@/lib/control-plane/brain-context-provider';

import {
  classifyStartupKnowledgeSource,
  projectAgentKnowledgeContextForModel,
  summarizeAgentKnowledgeContext,
} from '@/lib/control-plane/knowledge-context-observability';

function node(
  input: {
    id: string;
    metadata?: Record<string, unknown>;
  },
) {
  return BrainGraphNodeSchema.parse({
    id:
      input.id,

    type:
      'source',

    label:
      input.id,

    summary:
      '',

    content:
      'context',

    rationaleSummary:
      '',

    status:
      'active',

    tags:
      [],

    keywords:
      [],

    context:
      {},

    governance: {
      visibility:
        'internal',
    },

    audit:
      {},

    version:
      1,

    metadata:
      input.metadata ??
      {},
  });
}

function context():
  AgentBrainContext {
  return {
    route:
      null,

    truncated:
      false,

    totalContentChars:
      14,

    results: [
      {
        node:
          node({
            id:
              'startup-memory:agent:lauti',

            metadata: {
              memorySystem:
                'startup-memory',
            },
          }),

        score:
          0.9,

        breakdown: {
          text:
            0.9,
          salience:
            0,
          activation:
            0,
          confidence:
            1,
          retention:
            1,
          recency:
            1,
          graphProximity:
            0,
        },

        reasons: [
          'dual_retrieval:startup-memory',
        ],
      },

      {
        node:
          node({
            id:
              'cognee:dataset:experience-1',

            metadata: {
              backend:
                'cognee-http-retrieval',
            },
          }),

        score:
          0.8,

        breakdown: {
          text:
            0.8,
          salience:
            0,
          activation:
            0,
          confidence:
            0,
          retention:
            0,
          recency:
            0,
          graphProximity:
            0,
        },

        reasons: [
          'dual_retrieval:startup-brain',
        ],
      },
    ],
  };
}

describe(
  'Startup knowledge context observability',
  () => {
    test(
      'classifies canonical Memory and experiential Brain separately',
      () => {
        const input =
          context();

        expect(
          classifyStartupKnowledgeSource(
            input.results[0]!,
          ),
        )
          .toBe(
            'startup-memory',
          );

        expect(
          classifyStartupKnowledgeSource(
            input.results[1]!,
          ),
        )
          .toBe(
            'startup-brain',
          );
      },
    );

    test(
      'summarizes source counts without persisting node content',
      () => {
        const summary =
          summarizeAgentKnowledgeContext(
            context(),
          );

        expect(
          summary.startupMemoryResults,
        )
          .toBe(
            1,
          );

        expect(
          summary.startupBrainResults,
        )
          .toBe(
            1,
          );

        expect(
          summary.nodes[0],
        )
          .not
          .toHaveProperty(
            'content',
          );
      },
    );

    test(
      'projects source-aware content only for model input',
      () => {
        const projected =
          projectAgentKnowledgeContextForModel(
            context(),
          );

        expect(
          projected[0]
            ?.source,
        )
          .toBe(
            'startup-memory',
          );

        expect(
          projected[1]
            ?.source,
        )
          .toBe(
            'startup-brain',
          );

        expect(
          projected[0]
            ?.content,
        )
          .toBe(
            'context',
          );
      },
    );
  },
);

