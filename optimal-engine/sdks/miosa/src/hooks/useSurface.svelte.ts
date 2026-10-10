import { onDestroy } from "svelte";
import type { EngineContext } from "../engine.js";
import type {
  SurfaceEvent,
  SurfaceStream,
  Subscription,
} from "@optimal-engine/client";
import { asSubscriptionId } from "@optimal-engine/client";

export function useSurface(engine: EngineContext) {
  let events = $state<SurfaceEvent[]>([]);
  let subscriptions = $state<Subscription[]>([]);
  let loading = $state(false);
  let error = $state<string | null>(null);
  let connected = $state(false);

  const streams = new Map<string, SurfaceStream>();

  function openStream(subId: string) {
    if (streams.has(subId)) return;
    const stream = engine.client.surface.stream(asSubscriptionId(subId));
    stream
      .on((ev) => {
        events = [ev, ...events].slice(0, 200); // cap at 200
      })
      .onError((err) => {
        error = err.message;
        connected = false;
        streams.delete(subId);
      });
    streams.set(subId, stream);
    connected = true;
  }

  function closeAllStreams() {
    for (const s of streams.values()) s.close();
    streams.clear();
    connected = false;
  }

  async function bootstrap() {
    closeAllStreams();
    error = null;
    loading = true;
    try {
      const ws = engine.getWorkspace();
      subscriptions = await engine.client.subscriptions.list({ workspace: ws });
      for (const sub of subscriptions) {
        openStream(sub.id);
      }
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  async function createSubscription(
    scope: "workspace" | "topic" | "actor" | "global",
    scopeValue?: string,
    categories?: string[],
  ) {
    const ws = engine.getWorkspace();
    const sub = await engine.client.subscriptions.create({
      workspace: ws,
      scope,
      scopeValue,
      categories,
    });
    subscriptions = [...subscriptions, sub];
    openStream(sub.id);
    return sub;
  }

  function clearEvents() {
    events = [];
  }

  // Bootstrap on workspace change
  $effect(() => {
    engine.workspace.subscribe(() => void bootstrap());
  });

  onDestroy(() => {
    closeAllStreams();
  });

  return {
    get events() {
      return events;
    },
    get subscriptions() {
      return subscriptions;
    },
    get loading() {
      return loading;
    },
    get error() {
      return error;
    },
    get connected() {
      return connected;
    },
    createSubscription,
    clearEvents,
    refresh: bootstrap,
  };
}
