import {
  BrainGateway,
} from '@/lib/brain/gateway/gateway';

import {
  InMemoryBrainAuditStore,
  InMemoryBrainGraphStore,
} from '@/lib/brain/gateway/store';

import {
  createEmptyBrainGraph,
} from '@/lib/brain/graph-ops';

import {
  CogneeRecallClient,
} from '@/lib/brain/cognee/cognee-recall-client';

import {
  CogneeBrainRetrievalBackend,
} from '@/lib/brain/cognee/cognee-brain-retrieval-backend';

import {
  BrainGatewayAgentContextProvider,
  type AgentBrainContextProvider,
} from '@/lib/control-plane/brain-context-provider';

import {
  EnvironmentRuntimeSecretSource,
} from '@/lib/control-plane/runtime-secret-source';

import {
  getDb,
} from '@/lib/data';

import {
  CanonicalStartupMemoryRetrievalBackend,
} from '@/lib/brain/memory/canonical-memory-retrieval-backend';

import {
  DualStartupKnowledgeRetrievalBackend,
} from '@/lib/brain/gateway/dual-startup-knowledge-retrieval-backend';

import type {
  BrainRetrievalBackend,
} from '@/lib/brain/gateway/retrieval-backend';


type StartupBrainRuntimeMode =
  | 'disabled'
  | 'cognee';


let cachedProvider:
  AgentBrainContextProvider |
  null =
  null;


function runtimeMode(
  env:
    Readonly<
      Record<
        string,
        string | undefined
      >
    >,
): StartupBrainRuntimeMode {
  const raw =
    env
      .STARTUP_BRAIN_RUNTIME_MODE
      ?.trim()
      .toLowerCase();


  if (
    !raw ||
    raw ===
      'disabled'
  ) {
    return 'disabled';
  }


  if (
    raw ===
      'cognee'
  ) {
    return 'cognee';
  }


  throw new Error(
    `invalid_STARTUP_BRAIN_RUNTIME_MODE:${raw}`,
  );
}


function createGatewayProvider(
  retrievalBackend:
    BrainRetrievalBackend,
): AgentBrainContextProvider {
  const graphStore =
    new InMemoryBrainGraphStore(
      createEmptyBrainGraph(
        'startup-brain-runtime',
        new Date()
          .toISOString(),
      ),
    );


  const auditStore =
    new InMemoryBrainAuditStore();


  const gateway =
    new BrainGateway(
      graphStore,
      auditStore,
      undefined,
      retrievalBackend,
    );


  return new BrainGatewayAgentContextProvider(
    gateway,
  );
}


function createCanonicalMemoryBackend() {
  return new CanonicalStartupMemoryRetrievalBackend({
    db:
      getDb(),
  });
}


export function getStartupBrainContextProvider(
  env:
    Readonly<
      Record<
        string,
        string | undefined
      >
    > =
      process.env,
): AgentBrainContextProvider {
  if (
    cachedProvider
  ) {
    return cachedProvider;
  }


  const mode =
    runtimeMode(
      env,
    );


  const memoryRetrievalBackend =
    createCanonicalMemoryBackend();


  /*
   * "disabled" disables the experiential
   * Startup Brain backend only.
   *
   * Canonical Startup Memory remains
   * available because company identity,
   * governance, roles and procedures are
   * foundational runtime knowledge rather
   * than optional experiential memory.
   */
  if (
    mode ===
    'disabled'
  ) {
    const retrievalBackend =
      new DualStartupKnowledgeRetrievalBackend({
        memoryBackend:
          memoryRetrievalBackend,

        memoryShare:
          1,
      });


    cachedProvider =
      createGatewayProvider(
        retrievalBackend,
      );


    return cachedProvider;
  }


  const baseUrl =
    env
      .COGNEE_BASE_URL
      ?.trim() ??
    'http://127.0.0.1:8000';


  const email =
    env
      .COGNEE_API_EMAIL
      ?.trim();


  const dataset =
    env
      .COGNEE_DATASET
      ?.trim();


  if (
    !email
  ) {
    throw new Error(
      'COGNEE_API_EMAIL_required',
    );
  }


  if (
    !dataset
  ) {
    throw new Error(
      'COGNEE_DATASET_required',
    );
  }


  const secretSource =
    new EnvironmentRuntimeSecretSource(
      env,
    );


  const cogneeClient =
    new CogneeRecallClient({
      baseUrl,

      email,

      passwordSecretName:
        'COGNEE_API_PASSWORD',

      secretSource,
    });


  const brainRetrievalBackend =
    new CogneeBrainRetrievalBackend({
      client:
        cogneeClient,

      dataset,

      /*
       * Temporary experiential Brain
       * smoke scope.
       *
       * Production Cognee memories will
       * carry their own governed scope.
       */
      readableByAgentIds: [
        'lauti',
      ],
    });


  const retrievalBackend =
    new DualStartupKnowledgeRetrievalBackend({
      memoryBackend:
        memoryRetrievalBackend,

      brainBackend:
        brainRetrievalBackend,

      memoryShare:
        0.25,
    });


  cachedProvider =
    createGatewayProvider(
      retrievalBackend,
    );


  return cachedProvider;
}


export function resetStartupBrainRuntimeForTests():
  void {
  cachedProvider =
    null;
}

