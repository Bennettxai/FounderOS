import type {
  FounderDb,
} from '@/lib/db';

import {
  BrainGraphNodeSchema,
  type BrainGraphNodeType,
} from '@/lib/brain/graph-schema';

import type {
  BrainRetrievalBackend,
} from '@/lib/brain/gateway/retrieval-backend';

import type {
  BrainRetrievalResult,
} from '@/lib/brain/retrieval';

import {
  buildCanonicalStartupMemory,
} from '@/lib/brain/memory/canonical-store';

import {
  searchCanonicalStartupMemory,
} from '@/lib/brain/memory/selective-retrieval';

import type {
  StartupMemoryRecord,
  StartupMemoryRecordKind,
} from '@/lib/brain/memory/canonical-schema';


function nodeTypeFor(
  kind:
    StartupMemoryRecordKind,
): BrainGraphNodeType {
  switch (kind) {
    case 'agent':
      return 'agent';

    case 'department':
      return 'department';

    case 'governance':
    case 'memory_policy':
      return 'policy';

    case 'company':
    default:
      return 'source';
  }
}


function requestedKindsFor(
  nodeTypes:
    BrainGraphNodeType[],
): StartupMemoryRecordKind[] | null {
  if (
    nodeTypes.length ===
      0
  ) {
    return [];
  }


  const kinds =
    new Set<
      StartupMemoryRecordKind
    >();


  for (
    const nodeType
    of nodeTypes
  ) {
    switch (
      nodeType
    ) {
      case 'agent':
        kinds.add(
          'agent',
        );
        break;

      case 'department':
        kinds.add(
          'department',
        );
        break;

      case 'policy':
        kinds.add(
          'governance',
        );
        kinds.add(
          'memory_policy',
        );
        break;

      case 'source':
        kinds.add(
          'company',
        );
        break;

      default:
        break;
    }
  }


  return kinds.size >
      0
    ? [
        ...kinds,
      ]
    : null;
}


function normalizedScore(
  raw:
    number,
): number {
  if (
    !Number.isFinite(
      raw,
    ) ||
    raw <=
      0
  ) {
    return 0;
  }


  return Math.min(
    0.99,
    raw /
      (
        raw +
        5
      ),
  );
}


function toResult(
  record:
    StartupMemoryRecord,

  score:
    number,

  reasons:
    string[],

  matchedTerms:
    string[],
): BrainRetrievalResult {
  const nodeType =
    nodeTypeFor(
      record.kind,
    );


  const node =
    BrainGraphNodeSchema.parse({
      id:
        `startup-memory:${record.id}`,

      type:
        nodeType,

      label:
        record.title,

      summary:
        record.summary,

      content:
        record.content,

      rationaleSummary:
        '',

      status:
        'active',

      canonicalKey:
        record.id,

      tags: [
        'startup-memory',
        'canonical',
        record.kind,
      ],

      keywords:
        record.keywords,

      context: {
        departmentIds:
          record.departmentId
            ? [
                record.departmentId,
              ]
            : [],

        agentIds:
          record.agentId
            ? [
                record.agentId,
              ]
            : [],

        tags: [
          'startup-memory',
          'canonical',
        ],
      },

      governance: {
        visibility:
          'internal',

        readableByAgentIds:
          [],

        readableByDepartmentIds:
          [],

        writableByAgentIds:
          [],

        writableByDepartmentIds:
          [],

        humanApprovalRequired:
          false,

        immutable:
          true,

        legalHold:
          false,
      },

      audit:
        {},

      version:
        record.version,

      metadata: {
        backend:
          'canonical-startup-memory',

        memorySystem:
          'startup-memory',

        originalRecordId:
          record.id,

        recordKind:
          record.kind,

        provenance:
          record.provenance,

        currentModel:
          record.currentModel ??
          null,

        matchedTerms,

        canonical:
          true,

        recordMetadata:
          record.metadata,
      },
    });


  const normalized =
    normalizedScore(
      score,
    );


  return {
    node,

    score:
      normalized,

    breakdown: {
      text:
        normalized,

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
      'canonical_startup_memory',
      ...reasons,
    ],
  };
}


export class CanonicalStartupMemoryRetrievalBackend
  implements BrainRetrievalBackend
{
  readonly id =
    'canonical-startup-memory';


  constructor(
    private readonly input: {
      db:
        FounderDb;

      rootDir?:
        string;
    },
  ) {}


  async retrieve({
    request,
    now,
  }: Parameters<
    BrainRetrievalBackend[
      'retrieve'
    ]
  >[0]): Promise<
    BrainRetrievalResult[]
  > {
    if (
      request.principal
        .actor.kind !==
        'agent' &&
      request.principal
        .actor.kind !==
        'human'
    ) {
      return [];
    }


    /*
     * Canonical Startup Memory is not
     * pretending to be episodic Brain
     * memory. Typed epistemic / memory
     * filters belong to the experiential
     * Brain side until explicitly mapped.
     */
    if (
      request.query
        .memorySystems
        .length >
        0 ||
      request.query
        .epistemicKinds
        .length >
        0 ||
      request.query
        .minConfidence !==
        undefined
    ) {
      return [];
    }


    const requestedNodeTypes =
      request
        .requestedNodeTypes
        .length >
      0
        ? request
            .requestedNodeTypes
        : request.query
            .nodeTypes;


    const kinds =
      requestedKindsFor(
        requestedNodeTypes,
      );


    if (
      kinds ===
        null
    ) {
      return [];
    }


    const topK =
      Math.min(
        request.query
          .limit,

        request.budget
          .maxResults,
      );


    const snapshot =
      buildCanonicalStartupMemory({
        db:
          this.input
            .db,

        rootDir:
          this.input
            .rootDir,

        generatedAt:
          now,
      });


    /*
     * Always include the requesting
     * employee identity in the Memory
     * query. This gives the runtime its
     * own role, mission, constraints and
     * model assignment without dumping
     * other employee records.
     */
    const query =
      request.principal
        .actor.kind ===
        'agent'
        ? [
            request.query
              .text,

            request.principal
              .actor.id,
          ]
            .filter(
              Boolean,
            )
            .join(
              ' ',
            )
        : request.query
            .text;


    const results =
      searchCanonicalStartupMemory(
        snapshot,
        {
          query,

          requester: {
            kind:
              request.principal
                .actor.kind,

            id:
              request.principal
                .actor.id,
          },

          limit:
            topK,

          kinds,
        },
      );


    return results.map(
      (
        result,
      ) =>
        toResult(
          result.record,
          result.score,
          result.reasons,
          result.matchedTerms,
        ),
    );
  }
}

