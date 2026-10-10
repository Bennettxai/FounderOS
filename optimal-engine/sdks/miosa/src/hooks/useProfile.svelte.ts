import type { EngineContext } from "../engine.js";
import type { ProfileResult, Bandwidth } from "@optimal-engine/client";

export interface UseProfileOptions {
  audience?: string;
  bandwidth?: Bandwidth;
  node?: string;
}

export function useProfile(
  engine: EngineContext,
  opts: UseProfileOptions = {},
) {
  let profile = $state<ProfileResult | null>(null);
  let loading = $state(false);
  let error = $state<string | null>(null);
  let bandwidth = $state<Bandwidth>(opts.bandwidth ?? "l1");

  async function load() {
    loading = true;
    error = null;
    try {
      const ws = engine.getWorkspace();
      profile = await engine.client.profile({
        workspace: ws,
        audience: opts.audience,
        bandwidth,
        node: opts.node,
      });
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    engine.workspace.subscribe(() => void load());
  });

  // Reload when bandwidth changes
  $effect(() => {
    const _bw = bandwidth;
    void load();
  });

  return {
    get profile() {
      return profile;
    },
    get loading() {
      return loading;
    },
    get error() {
      return error;
    },
    get bandwidth() {
      return bandwidth;
    },
    set bandwidth(bw: Bandwidth) {
      bandwidth = bw;
    },
    refresh: load,
  };
}
