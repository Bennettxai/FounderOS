import type { EngineContext } from "../engine.js";
import type { RecallResult } from "@optimal-engine/client";

export type RecallVerb = "actions" | "who" | "when" | "where" | "owns";

export function useRecall(engine: EngineContext) {
  let results = $state<RecallResult | null>(null);
  let loading = $state(false);
  let error = $state<string | null>(null);
  let activeVerb = $state<RecallVerb>("actions");
  let inputValue = $state("");

  async function recall(verb: RecallVerb, value: string) {
    if (!value.trim()) return;
    loading = true;
    error = null;
    activeVerb = verb;
    inputValue = value;
    const ws = engine.getWorkspace();
    try {
      switch (verb) {
        case "actions":
          results = await engine.client.recall.actions({
            topic: value,
            workspace: ws,
          });
          break;
        case "who":
          results = await engine.client.recall.who({
            topic: value,
            workspace: ws,
          });
          break;
        case "when":
          results = await engine.client.recall.when({
            event: value,
            workspace: ws,
          });
          break;
        case "where":
          results = await engine.client.recall.where({
            thing: value,
            workspace: ws,
          });
          break;
        case "owns":
          results = await engine.client.recall.owns({
            actor: value,
            workspace: ws,
          });
          break;
      }
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  function clear() {
    results = null;
    inputValue = "";
    error = null;
  }

  return {
    get results() {
      return results;
    },
    get loading() {
      return loading;
    },
    get error() {
      return error;
    },
    get activeVerb() {
      return activeVerb;
    },
    get inputValue() {
      return inputValue;
    },
    recall,
    clear,
  };
}
