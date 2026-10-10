import { OptimalEngine } from "@optimal-engine/client";
import { writable, get, type Readable } from "svelte/store";

export interface EngineConfig {
  baseUrl?: string;
  apiKey?: string;
  workspace?: string;
}

export interface EngineContext {
  client: OptimalEngine;
  workspace: Readable<string>;
  setWorkspace: (id: string) => void;
  getWorkspace: () => string;
}

export function createEngine(config: EngineConfig = {}): EngineContext {
  const initialWorkspace = config.workspace ?? "default";
  const workspace = writable<string>(initialWorkspace);

  const client = new OptimalEngine({
    baseUrl: config.baseUrl ?? "http://localhost:4200",
    apiKey: config.apiKey,
    workspace: initialWorkspace,
  });

  return {
    client,
    workspace,
    setWorkspace: (id: string) => workspace.set(id),
    getWorkspace: () => get(workspace),
  };
}
