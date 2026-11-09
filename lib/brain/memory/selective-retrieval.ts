import {
  StartupMemorySearchRequestSchema,
  type StartupMemoryRecord,
  type StartupMemorySearchRequest,
  type StartupMemorySnapshot,
} from '@/lib/brain/memory/canonical-schema';

const STOP_WORDS = new Set([
  'a',
  'an',
  'and',
  'are',
  'can',
  'could',
  'do',
  'does',
  'for',
  'from',
  'how',
  'i',
  'in',
  'is',
  'it',
  'may',
  'of',
  'on',
  'or',
  'the',
  'to',
  'what',
  'when',
  'where',
  'which',
  'who',
  'why',
  'with',
]);

function normalize(value: string): string {
  return value
    .toLowerCase()
    .replace(/[_/.-]+/g, ' ')
    .replace(/[^a-z0-9 ]+/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

function queryTerms(query: string): string[] {
  const unique = new Set(
    normalize(query)
      .split(' ')
      .filter((token) => token.length >= 2 && !STOP_WORDS.has(token)),
  );

  return [...unique];
}

function policyIntent(query: string): boolean {
  return /\b(can|cannot|may|allowed|permission|approve|approval|execute|execution|authority|forbidden)\b/i.test(
    query,
  );
}

function executionIntent(query: string): boolean {
  return /\b(execute|execution|trade|trades|buy|sell|broker|order|orders|deposit|withdraw)\b/i.test(
    query,
  );
}

function departmentIntent(query: string): boolean {
  return /\b(department|team|research team|risk team|monitoring team)\b/i.test(
    query,
  );
}

function explicitlyNamedAgentIds(
  snapshot: StartupMemorySnapshot,
  request: StartupMemorySearchRequest,
): Set<string> {
  const terms = new Set(queryTerms(request.query));

  return new Set(
    snapshot.records
      .filter(
        (record) =>
          record.kind === 'agent' &&
          record.agentId &&
          terms.has(normalize(record.agentId)),
      )
      .map((record) => record.agentId as string),
  );
}

function scoreRecord(
  record: StartupMemoryRecord,
  request: StartupMemorySearchRequest,
) {
  const terms = queryTerms(request.query);
  const matchedTerms = new Set<string>();
  const reasons: string[] = [];

  const title = normalize(record.title);
  const summary = normalize(record.summary);
  const content = normalize(record.content);
  const id = normalize(record.id);
  const keywords = new Set(record.keywords.map(normalize));

  let score = 0;

  for (const term of terms) {
    let matched = false;

    if (title.includes(term)) {
      score += 5;
      matched = true;
    }

    if (id.includes(term)) {
      score += 4;
      matched = true;
    }

    if (keywords.has(term)) {
      score += 3;
      matched = true;
    }

    if (summary.includes(term)) {
      score += 2;
      matched = true;
    }

    if (content.includes(term)) {
      score += 1;
      matched = true;
    }

    if (matched) matchedTerms.add(term);
  }

  const isPolicyIntent = policyIntent(request.query);
  const isExecutionIntent = executionIntent(request.query);

  if (isPolicyIntent && record.kind === 'governance' && score > 0) {
    score += 3;
    reasons.push('governance_intent_boost');
  }

  if (isPolicyIntent && record.kind === 'agent' && score > 0) {
    score += 1;
    reasons.push('agent_policy_context');
  }

  if (record.kind === 'governance' && isExecutionIntent && score > 0) {
    const section =
      typeof record.metadata.section === 'string'
        ? record.metadata.section
        : null;

    if (section === 'financial_execution_policy') {
      score += 12;
      reasons.push('execution_policy_primary');
    } else if (section === 'separation_of_duties') {
      score += 7;
      reasons.push('execution_separation_secondary');
    } else if (section === 'approval_policy') {
      score += 4;
      reasons.push('execution_approval_context');
    }
  }

  if (
    request.requester.kind === 'agent' &&
    record.agentId === request.requester.id &&
    score > 0
  ) {
    score += 1.5;
    reasons.push('requesting_agent_context');
  }

  if (matchedTerms.size > 0) {
    reasons.push(`matched_terms:${[...matchedTerms].join(',')}`);
  }

  return {
    score,
    matchedTerms: [...matchedTerms],
    reasons,
  };
}

export function searchCanonicalStartupMemory(
  snapshot: StartupMemorySnapshot,
  input: StartupMemorySearchRequest,
) {
  const request = StartupMemorySearchRequestSchema.parse(input);
  const allowedKinds = new Set(request.kinds);
  const namedAgentIds = explicitlyNamedAgentIds(snapshot, request);
  const suppressDepartments =
    namedAgentIds.size > 0 &&
    policyIntent(request.query) &&
    !departmentIntent(request.query);

  return snapshot.records
    .filter(
      (record) =>
        (allowedKinds.size === 0 || allowedKinds.has(record.kind)) &&
        (
          record.kind !== 'agent' ||
          namedAgentIds.size === 0 ||
          (record.agentId ? namedAgentIds.has(record.agentId) : false)
        ) &&
        !(suppressDepartments && record.kind === 'department'),
    )
    .map((record) => ({
      record,
      ...scoreRecord(record, request),
    }))
    .filter((result) => result.score > 0)
    .sort((a, b) => {
      if (b.score !== a.score) return b.score - a.score;
      return a.record.id.localeCompare(b.record.id);
    })
    .slice(0, request.limit);
}

