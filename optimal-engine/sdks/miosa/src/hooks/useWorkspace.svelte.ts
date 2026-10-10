import { get } from "svelte/store";
import type { EngineContext } from "../engine.js";
import type { Workspace } from "@optimal-engine/client";

export function useWorkspace(engine: EngineContext) {
  let workspaces = $state<Workspace[]>([]);
  let current = $state<string>(engine.getWorkspace());
  let loading = $state(false);
  let error = $state<string | null>(null);

  // Keep current in sync with the engine store
  $effect(() => {
    const unsub = engine.workspace.subscribe((ws) => {
      current = ws;
    });
    return unsub;
  });

  async function loadWorkspaces() {
    loading = true;
    error = null;
    try {
      workspaces = await engine.client.workspaces.list();
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  async function createWorkspace(
    slug: string,
    name: string,
    description?: string,
  ) {
    loading = true;
    error = null;
    try {
      const ws = await engine.client.workspaces.create({
        slug,
        name,
        description,
      });
      workspaces = [...workspaces, ws];
      return ws;
    } catch (e) {
      error = (e as Error).message;
      return null;
    } finally {
      loading = false;
    }
  }

  function switchTo(id: string) {
    engine.setWorkspace(id);
  }

  $effect(() => {
    void loadWorkspaces();
  });

  return {
    get workspaces() {
      return workspaces;
    },
    get current() {
      return current;
    },
    get loading() {
      return loading;
    },
    get error() {
      return error;
    },
    switchTo,
    createWorkspace,
    refresh: loadWorkspaces,
  };
}
