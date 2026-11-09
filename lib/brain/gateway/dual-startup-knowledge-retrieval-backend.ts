import type {
  BrainRetrievalBackend,
} from '@/lib/brain/gateway/retrieval-backend';

import type {
  BrainRetrievalResult,
} from '@/lib/brain/retrieval';


type TaggedResult = {
  source:
    'memory' |
    'brain';

  result:
    BrainRetrievalResult;
};


function descending(
  a:
    BrainRetrievalResult,

  b:
    BrainRetrievalResult,
): number {
  return (
    b.score -
      a.score ||
    a.node.id
      .localeCompare(
        b.node.id,
      )
  );
}


function dedupe(
  results:
    BrainRetrievalResult[],
): BrainRetrievalResult[] {
  const seen =
    new Set<
      string
    >();


  return results
    .sort(
      descending,
    )
    .filter(
      (
        result,
      ) => {
        if (
          seen.has(
            result.node.id,
          )
        ) {
          return false;
        }


        seen.add(
          result.node.id,
        );

        return true;
      },
    );
}


export class DualStartupKnowledgeRetrievalBackend
  implements BrainRetrievalBackend
{
  readonly id =
    'startup-memory-plus-brain';


  constructor(
    private readonly input: {
      memoryBackend:
        BrainRetrievalBackend;

      brainBackend?:
        BrainRetrievalBackend;

      memoryShare?:
        number;
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
    const limit =
      Math.min(
        request.query
          .limit,

        request.budget
          .maxResults,
      );


    const memoryPromise =
      this.input
        .memoryBackend
        .retrieve({
          request,
          now,
        });


    const brainPromise =
      this.input
        .brainBackend
        ? this.input
            .brainBackend
            .retrieve({
              request,
              now,
            })
        : Promise.resolve(
            [],
          );


    /*
     * Startup Memory is foundational and
     * local. If it fails, the request
     * should fail visibly.
     *
     * Experiential Brain retrieval may
     * degrade temporarily (for example
     * Cognee being offline) without
     * depriving the employee of canonical
     * company knowledge.
     */
    const memory =
      dedupe(
        await memoryPromise,
      );


    let brain:
      BrainRetrievalResult[] =
        [];


    try {
      brain =
        dedupe(
          await brainPromise,
        );
    } catch {
      brain =
        [];
    }


    if (
      limit <=
        0
    ) {
      return [];
    }


    if (
      brain.length ===
        0
    ) {
      return memory
        .slice(
          0,
          limit,
        )
        .map(
          (
            result,
          ) => ({
            ...result,

            reasons: [
              ...result.reasons,
              'dual_retrieval:startup-memory',
            ],
          }),
        );
    }


    if (
      memory.length ===
        0
    ) {
      return brain
        .slice(
          0,
          limit,
        )
        .map(
          (
            result,
          ) => ({
            ...result,

            reasons: [
              ...result.reasons,
              'dual_retrieval:startup-brain',
            ],
          }),
        );
    }


    const requestedShare =
      this.input
        .memoryShare ??
      0.25;


    const memoryShare =
      Math.max(
        0.1,
        Math.min(
          0.75,
          requestedShare,
        ),
      );


    const memoryTarget =
      Math.min(
        memory.length,

        Math.max(
          1,
          Math.ceil(
            limit *
              memoryShare,
          ),
        ),
      );


    const selected:
      TaggedResult[] =
        memory
          .slice(
            0,
            memoryTarget,
          )
          .map(
            (
              result,
            ) => ({
              source:
                'memory' as const,

              result,
            }),
          );


    const remainingAfterMemory =
      Math.max(
        0,
        limit -
          selected.length,
      );


    selected.push(
      ...brain
        .slice(
          0,
          remainingAfterMemory,
        )
        .map(
          (
            result,
          ) => ({
            source:
              'brain' as const,

            result,
          }),
        ),
    );


    const selectedIds =
      new Set(
        selected.map(
          (
            item,
          ) =>
            item.result
              .node.id,
        ),
      );


    if (
      selected.length <
        limit
    ) {
      const remaining =
        [
          ...memory.map(
            (
              result,
            ) => ({
              source:
                'memory' as const,

              result,
            }),
          ),

          ...brain.map(
            (
              result,
            ) => ({
              source:
                'brain' as const,

              result,
            }),
          ),
        ]
          .filter(
            (
              item,
            ) =>
              !selectedIds.has(
                item.result
                  .node.id,
              ),
          )
          .sort(
            (
              a,
              b,
            ) =>
              descending(
                a.result,
                b.result,
              ),
          );


      for (
        const item
        of remaining
      ) {
        if (
          selected.length >=
            limit
        ) {
          break;
        }


        selected.push(
          item,
        );

        selectedIds.add(
          item.result
            .node.id,
        );
      }
    }


    return selected.map(
      (
        item,
      ) => ({
        ...item.result,

        reasons: [
          ...item.result
            .reasons,

          item.source ===
            'memory'
            ? 'dual_retrieval:startup-memory'
            : 'dual_retrieval:startup-brain',
        ],
      }),
    );
  }
}

