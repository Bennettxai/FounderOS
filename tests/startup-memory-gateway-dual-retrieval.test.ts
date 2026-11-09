import {
  afterEach,
  describe,
  expect,
  test,
} from 'vitest';

import {
  openDb,
  type FounderDb,
} from '@/lib/db';

import {
  seedDatabase,
} from '@/lib/seed';

import {
  createEmptyBrainGraph,
} from '@/lib/brain/graph-ops';

import {
  BrainGraphNodeSchema,
} from '@/lib/brain/graph-schema';

import {
  BrainGateway,
} from '@/lib/brain/gateway/gateway';

import {
  InMemoryBrainAuditStore,
  InMemoryBrainGraphStore,
} from '@/lib/brain/gateway/store';

import type {
  BrainRetrievalBackend,
} from '@/lib/brain/gateway/retrieval-backend';

import {
  CanonicalStartupMemoryRetrievalBackend,
} from '@/lib/brain/memory/canonical-memory-retrieval-backend';

import {
  DualStartupKnowledgeRetrievalBackend,
} from '@/lib/brain/gateway/dual-startup-knowledge-retrieval-backend';


let db:
  FounderDb |
  null =
  null;


afterEach(
  () => {
    db?.close();

    db =
      null;
  },
);


function createGateway(
  brainBackend?:
    BrainRetrievalBackend,
) {
  db =
    openDb(
      ':memory:',
    );

  seedDatabase(
    db,
  );


  const memoryBackend =
    new CanonicalStartupMemoryRetrievalBackend({
      db,
      rootDir:
        process.cwd(),
    });


  const dualBackend =
    new DualStartupKnowledgeRetrievalBackend({
      memoryBackend,
      brainBackend,
      memoryShare:
        0.25,
    });


  const graphStore =
    new InMemoryBrainGraphStore(
      createEmptyBrainGraph(
        'test-dual-retrieval',
        '2026-01-01T00:00:00.000Z',
      ),
    );


  const auditStore =
    new InMemoryBrainAuditStore();


  const gateway =
    new BrainGateway(
      graphStore,
      auditStore,
      () =>
        '2026-01-01T01:00:00.000Z',
      dualBackend,
    );


  return {
    gateway,
    auditStore,
  };
}


function request() {
  return {
    requestId:
      'dual-read-1',

    principal: {
      actor: {
        kind:
          'agent' as const,

        id:
          'lauti',
      },

      departmentIds: [
        'dept-research',
      ],

      capabilities: [
        'brain.read' as const,
        'brain.retrieve' as const,
      ],

      issuedBy:
        'company-control-plane',
    },

    intent:
      'institutional_memory' as const,

    purpose:
      'Retrieve relevant Startup Memory and Startup Brain context for Lauti.',

    query: {
      text:
        'Can I execute trades while researching listed equities?',

      limit:
        6,
    },

    context: {
      departmentIds: [
        'dept-research',
      ],

      agentIds: [
        'lauti',
      ],
    },

    budget: {
      maxResults:
        6,

      maxHops:
        2,

      maxContentCharsPerNode:
        100_000,

      maxTotalContentChars:
        250_000,

      includeContent:
        true,

      includeMetadata:
        true,
    },

    requestedNodeTypes:
      [],
  };
}


function experientialNode() {
  return BrainGraphNodeSchema.parse({
    id:
      'cognee:test:experience-1',

    type:
      'source',

    label:
      'Prior equity research experience',

    summary:
      'A previous Startup Brain experience relevant to equity research.',

    content:
      'Prior experiential Brain context.',

    rationaleSummary:
      '',

    status:
      'active',

    tags: [
      'cognee',
      'startup-brain',
    ],

    keywords: [
      'equity',
      'research',
    ],

    context: {
      departmentIds: [
        'dept-research',
      ],

      agentIds: [
        'lauti',
      ],
    },

    governance: {
      visibility:
        'internal',

      readableByAgentIds: [
        'lauti',
      ],

      readableByDepartmentIds:
        [],

      writableByAgentIds:
        [],

      writableByDepartmentIds:
        [],

      humanApprovalRequired:
        false,

      immutable:
        false,

      legalHold:
        false,
    },

    audit:
      {},

    version:
      1,

    metadata: {
      backend:
        'fake-cognee',
    },
  });
}


describe(
  'Startup Memory + Startup Brain dual retrieval',
  () => {
    test(
      'returns canonical Startup Memory and experiential Brain through one BrainGateway read',
      async () => {
        const brainBackend:
          BrainRetrievalBackend = {
            id:
              'fake-brain',

            async retrieve() {
              return [
                {
                  node:
                    experientialNode(),

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
                      0.8,

                    retention:
                      0.8,

                    recency:
                      0.8,

                    graphProximity:
                      0,
                  },

                  reasons: [
                    'fake_experiential_memory',
                  ],
                },
              ];
            },
          };


        const {
          gateway,
          auditStore,
        } =
          createGateway(
            brainBackend,
          );


        const result =
          await gateway
            .read(
              request(),
            );


        expect(
          result.results
            .some(
              (
                item,
              ) =>
                item.node.id
                  .startsWith(
                    'startup-memory:',
                  ),
            ),
        )
          .toBe(
            true,
          );


        expect(
          result.results
            .some(
              (
                item,
              ) =>
                item.node.id ===
                'cognee:test:experience-1',
            ),
        )
          .toBe(
            true,
          );


        expect(
          result.results
            .some(
              (
                item,
              ) =>
                item.node.id ===
                'startup-memory:agent:lauti',
            ),
        )
          .toBe(
            true,
          );


        expect(
          result.results.length,
        )
          .toBeLessThanOrEqual(
            6,
          );


        const audit =
          await auditStore
            .list();


        expect(
          audit[0]
            ?.metadata
            ?.retrievalBackend,
        )
          .toBe(
            'startup-memory-plus-brain',
          );
      },
    );


    test(
      'keeps canonical Startup Memory available when experiential Brain retrieval fails',
      async () => {
        const brainBackend:
          BrainRetrievalBackend = {
            id:
              'failing-brain',

            async retrieve() {
              throw new Error(
                'brain_temporarily_unavailable',
              );
            },
          };


        const {
          gateway,
        } =
          createGateway(
            brainBackend,
          );


        const result =
          await gateway
            .read(
              request(),
            );


        expect(
          result.results.length,
        )
          .toBeGreaterThan(
            0,
          );


        expect(
          result.results
            .every(
              (
                item,
              ) =>
                item.node.id
                  .startsWith(
                    'startup-memory:',
                  ),
            ),
        )
          .toBe(
            true,
          );
      },
    );


    test(
      'respects requested node types and does not map canonical Memory to experiential node types',
      async () => {
        const {
          gateway,
        } =
          createGateway();


        const input =
          request();


        const result =
          await gateway
            .read({
              ...input,

              requestedNodeTypes: [
                'experience',
              ],
            });


        expect(
          result.results,
        )
          .toEqual(
            [],
          );
      },
    );
  },
);

