import type { EngineContext } from "../engine.js";
import type { Memory } from "@optimal-engine/client";

export interface UseMemoriesOptions {
  audience?: string;
  limit?: number;
}

export function useMemories(
  engine: EngineContext,
  opts: UseMemoriesOptions = {},
) {
  let memories = $state<Memory[]>([]);
  let loading = $state(false);
  let error = $state<string | null>(null);
  let total = $state(0);
  let query = $state("");

  async function load() {
    loading = true;
    error = null;
    try {
      const ws = engine.getWorkspace();
      const result = await engine.client.memory.list({
        workspace: ws,
        audience: opts.audience,
        limit: opts.limit ?? 50,
      });
      // Client-side filter when query is set (API may not support q param on list)
      const all = result.memories ?? [];
      memories = query
        ? all.filter((m) =>
            m.content.toLowerCase().includes(query.toLowerCase()),
          )
        : all;
      total = result.memories?.length ?? 0;
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  async function create(
    content: string,
    extraOpts?: { isStatic?: boolean; audience?: string; citationUri?: string },
  ) {
    const ws = engine.getWorkspace();
    const m = await engine.client.memory.create({
      content,
      workspace: ws,
      audience: extraOpts?.audience ?? opts.audience,
      isStatic: extraOpts?.isStatic,
      citationUri: extraOpts?.citationUri,
    });
    memories = [m, ...memories];
    total += 1;
    return m;
  }

  async function forget(id: string, reason?: string) {
    const { asMemoryId } = await import("@optimal-engine/client");
    await engine.client.memory.forget(asMemoryId(id), { reason });
    memories = memories.filter((m) => m.id !== id);
    total = Math.max(0, total - 1);
  }

  // Reload whenever workspace changes
  $effect(() => {
    const _ws = engine.getWorkspace();
    void load();
  });

  $effect(() => {
    engine.workspace.subscribe(() => void load());
  });

  return {
    get memories() {
      return memories;
    },
    get loading() {
      return loading;
    },
    get error() {
      return error;
    },
    get total() {
      return total;
    },
    get query() {
      return query;
    },
    set query(q: string) {
      query = q;
      void load();
    },
    create,
    forget,
    refresh: load,
  };
}
