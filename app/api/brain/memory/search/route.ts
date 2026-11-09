import { NextResponse } from 'next/server';

import { getDb } from '@/lib/data';
import {
  getCompanyAgentRuntimeProfile,
} from '@/lib/control-plane/agent-runtime-profile';
import {
  buildCanonicalStartupMemory,
} from '@/lib/brain/memory/canonical-store';
import {
  StartupMemorySearchRequestSchema,
} from '@/lib/brain/memory/canonical-schema';
import {
  searchCanonicalStartupMemory,
} from '@/lib/brain/memory/selective-retrieval';

export const dynamic = 'force-dynamic';

function validateRequester(requester: { kind: 'agent' | 'human'; id: string }) {
  if (requester.kind === 'human') {
    if (requester.id !== 'ceo') {
      throw new Error('startup_memory_human_requester_forbidden');
    }
    return;
  }

  getCompanyAgentRuntimeProfile(requester.id);
}

export async function POST(request: Request) {
  try {
    const body = StartupMemorySearchRequestSchema.parse(await request.json());
    validateRequester(body.requester);

    const db = getDb();
    const snapshot = buildCanonicalStartupMemory({ db });
    const results = searchCanonicalStartupMemory(snapshot, body);

    return NextResponse.json(
      {
        schemaVersion: snapshot.schemaVersion,
        generatedAt: snapshot.generatedAt,
        memorySystem: 'startup-memory',
        retrieval: 'selective',
        fullDumpAllowed: false,
        totalCanonicalRecords: snapshot.records.length,
        returnedResults: results.length,
        request: {
          requester: body.requester,
          query: body.query,
          limit: body.limit,
          kinds: body.kinds,
        },
        results,
      },
      {
        headers: {
          'Cache-Control': 'no-store',
        },
      },
    );
  } catch (error) {
    return NextResponse.json(
      {
        error:
          error instanceof Error
            ? error.message
            : 'startup_memory_search_failed',
      },
      {
        status: 400,
        headers: {
          'Cache-Control': 'no-store',
        },
      },
    );
  }
}
