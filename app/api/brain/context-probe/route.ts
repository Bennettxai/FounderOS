import {
  randomUUID,
} from 'node:crypto';

import {
  NextResponse,
} from 'next/server';

import {
  getCompanyAgentRuntimeProfile,
} from '@/lib/control-plane/agent-runtime-profile';

import {
  getStartupBrainContextProvider,
} from '@/lib/control-plane/startup-brain-runtime';

import {
  AgentRuntimeTaskSchema,
} from '@/lib/control-plane/runtime-schema';

import {
  summarizeAgentKnowledgeContext,
} from '@/lib/control-plane/knowledge-context-observability';

export const dynamic =
  'force-dynamic';

export async function GET(
  request: Request,
) {
  if (
    process.env.NODE_ENV ===
      'production'
  ) {
    return NextResponse.json(
      {
        error:
          'context_probe_disabled_in_production',
      },
      {
        status:
          404,
      },
    );
  }

  const url =
    new URL(
      request.url,
    );

  const agentId =
    (
      url.searchParams
        .get(
          'agentId',
        ) ??
      'lauti'
    )
      .trim();

  const query =
    (
      url.searchParams
        .get(
          'q',
        ) ??
      'Can Lauti execute trades while researching listed equities?'
    )
      .trim();

  if (
    !agentId ||
    !query
  ) {
    return NextResponse.json(
      {
        error:
          'agentId_and_query_required',
      },
      {
        status:
          400,
      },
    );
  }

  try {
    const profile =
      getCompanyAgentRuntimeProfile(
        agentId,
      );

    const now =
      new Date()
        .toISOString();

    const task =
      AgentRuntimeTaskSchema.parse({
        taskId:
          `context-probe:${randomUUID()}`,

        handoffId:
          `context-probe-handoff:${randomUUID()}`,

        agentId,

        action:
          'context_probe',

        summary:
          query,

        priority:
          'normal',

        status:
          'queued',

        attempts:
          0,

        createdAt:
          now,

        availableAt:
          now,

        updatedAt:
          now,

        retryPolicy: {
          maxAttempts:
            1,

          initialDelayMs:
            0,

          backoffMultiplier:
            1,

          maxDelayMs:
            0,

          timeoutMs:
            profile.budget
              .timeoutMs,
        },

        policyEvidence:
          [],

        payload: {
          diagnostic:
            true,
        },

        result:
          {},
      });

    const provider =
      getStartupBrainContextProvider(
        process.env,
      );

    const context =
      await provider.retrieve({
        profile,
        task,
        workflow:
          null,

        requestId:
          `dev-context-probe:${randomUUID()}`,
      });

    const summary =
      summarizeAgentKnowledgeContext(
        context,
      );

    return NextResponse.json(
      {
        mode:
          process.env
            .STARTUP_BRAIN_RUNTIME_MODE ??
          'disabled',

        diagnosticOnly:
          true,

        operationalMutation:
          false,

        omniRouteCalled:
          false,

        agentId,

        query,

        ...summary,
      },
      {
        headers: {
          'Cache-Control':
            'no-store',
        },
      },
    );
  } catch (
    error
  ) {
    return NextResponse.json(
      {
        error:
          error instanceof
            Error
            ? error.message
            : String(
                error,
              ),
      },
      {
        status:
          400,
      },
    );
  }
}

