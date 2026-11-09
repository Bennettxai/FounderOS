import type {
  AgentBrainContext,
} from '@/lib/control-plane/brain-context-provider';

export type StartupKnowledgeSource =
  | 'startup-memory'
  | 'startup-brain'
  | 'unknown';

export type StartupKnowledgeContextTrace = {
  nodeId: string;
  source: StartupKnowledgeSource;
  nodeType: string;
  score: number;
  reasons: string[];
};

function metadataRecord(
  value: unknown,
): Record<string, unknown> {
  return (
    value &&
    typeof value === 'object' &&
    !Array.isArray(value)
  )
    ? value as Record<string, unknown>
    : {};
}

export function classifyStartupKnowledgeSource(
  item: AgentBrainContext['results'][number],
): StartupKnowledgeSource {
  const metadata =
    metadataRecord(
      item.node.metadata,
    );

  if (
    metadata.memorySystem ===
      'startup-memory' ||
    item.node.id.startsWith(
      'startup-memory:',
    ) ||
    item.reasons.includes(
      'dual_retrieval:startup-memory',
    )
  ) {
    return 'startup-memory';
  }

  if (
    item.node.id.startsWith(
      'cognee:',
    ) ||
    item.reasons.includes(
      'dual_retrieval:startup-brain',
    ) ||
    metadata.backend ===
      'cognee-http-retrieval'
  ) {
    return 'startup-brain';
  }

  return 'unknown';
}

export function summarizeAgentKnowledgeContext(
  context: AgentBrainContext,
) {
  const nodes:
    StartupKnowledgeContextTrace[] =
      context.results.map(
        (item) => ({
          nodeId:
            item.node.id,

          source:
            classifyStartupKnowledgeSource(
              item,
            ),

          nodeType:
            item.node.type,

          score:
            item.score,

          reasons:
            [...item.reasons],
        }),
      );

  return {
    totalResults:
      nodes.length,

    startupMemoryResults:
      nodes.filter(
        (node) =>
          node.source ===
          'startup-memory',
      ).length,

    startupBrainResults:
      nodes.filter(
        (node) =>
          node.source ===
          'startup-brain',
      ).length,

    unknownResults:
      nodes.filter(
        (node) =>
          node.source ===
          'unknown',
      ).length,

    truncated:
      context.truncated,

    totalContentChars:
      context.totalContentChars,

    route:
      context.route,

    nodes,
  };
}

export function projectAgentKnowledgeContextForModel(
  context: AgentBrainContext,
) {
  return context.results.map(
    (item) => ({
      source:
        classifyStartupKnowledgeSource(
          item,
        ),

      nodeId:
        item.node.id,

      type:
        item.node.type,

      label:
        item.node.label,

      summary:
        item.node.summary,

      content:
        item.node.content,

      score:
        item.score,

      reasons:
        item.reasons,
    }),
  );
}

