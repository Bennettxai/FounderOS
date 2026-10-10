import { HttpClient } from "./http.js";
import type {
  CreateSubscriptionInput,
  ListSubscriptionsOptions,
  Subscription,
  SubscriptionId,
} from "./types.js";

// ---------------------------------------------------------------------------
// SSE surface stream
// ---------------------------------------------------------------------------

export interface SurfaceEvent {
  type: string;
  data: unknown;
}

export type SurfaceEventHandler = (event: SurfaceEvent) => void;
export type SurfaceErrorHandler = (err: Error) => void;

/**
 * A lightweight SSE stream handle. Call `.close()` to disconnect.
 *
 * Works in Node 20+ (fetch + ReadableStream), browsers, and Cloudflare
 * Workers. Does NOT depend on the browser `EventSource` API so it also works
 * in environments that lack it.
 */
export interface SurfaceStream {
  /** Register a handler for incoming events. */
  on(handler: SurfaceEventHandler): this;
  /** Register an error handler. */
  onError(handler: SurfaceErrorHandler): this;
  /** Abort the stream. */
  close(): void;
}

function parseSseLine(line: string): { field: string; value: string } | null {
  const colon = line.indexOf(":");
  if (colon === -1) return null;
  return {
    field: line.slice(0, colon).trim(),
    value: line.slice(colon + 1).trimStart(),
  };
}

function createSurfaceStream(
  url: string,
  headers: Record<string, string>,
): SurfaceStream {
  const controller = new AbortController();
  const eventHandlers: SurfaceEventHandler[] = [];
  const errorHandlers: SurfaceErrorHandler[] = [];

  const stream: SurfaceStream = {
    on(handler) {
      eventHandlers.push(handler);
      return this;
    },
    onError(handler) {
      errorHandlers.push(handler);
      return this;
    },
    close() {
      controller.abort();
    },
  };

  // Fire-and-forget async loop
  void (async () => {
    let res: Response;
    try {
      res = await fetch(url, {
        headers: { ...headers, Accept: "text/event-stream" },
        signal: controller.signal,
      });
      if (!res.ok) {
        const err = new Error(`SSE connect failed: HTTP ${res.status}`);
        for (const h of errorHandlers) h(err);
        return;
      }
    } catch (err) {
      if ((err as Error).name === "AbortError") return;
      for (const h of errorHandlers) h(err as Error);
      return;
    }

    const body = res.body;
    if (body === null) {
      for (const h of errorHandlers) h(new Error("SSE response body is null"));
      return;
    }

    const reader = body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";
    let eventType = "message";
    let dataLines: string[] = [];

    const flush = (): void => {
      if (dataLines.length === 0) return;
      const data = dataLines.join("\n");
      let parsed: unknown = data;
      try {
        parsed = JSON.parse(data);
      } catch {
        // keep raw string
      }
      const ev: SurfaceEvent = { type: eventType, data: parsed };
      for (const h of eventHandlers) h(ev);
      eventType = "message";
      dataLines = [];
    };

    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split("\n");
        buffer = lines.pop() ?? "";
        for (const line of lines) {
          if (line === "") {
            flush();
          } else {
            const parsed = parseSseLine(line);
            if (parsed === null) continue;
            if (parsed.field === "event") eventType = parsed.value;
            else if (parsed.field === "data") dataLines.push(parsed.value);
          }
        }
      }
    } catch (err) {
      if ((err as Error).name !== "AbortError") {
        for (const h of errorHandlers) h(err as Error);
      }
    }
  })();

  return stream;
}

// ---------------------------------------------------------------------------
// Subscription client
// ---------------------------------------------------------------------------

export class SubscriptionClient {
  constructor(
    private readonly http: HttpClient,
    private readonly defaultWorkspace: string | undefined,
  ) {}

  private ws(override: string | undefined): string | undefined {
    return override ?? this.defaultWorkspace;
  }

  list(opts: ListSubscriptionsOptions = {}): Promise<Subscription[]> {
    const workspace = this.ws(opts.workspace);
    const qs = HttpClient.buildQuery({
      ...(workspace !== undefined ? { workspace } : {}),
    });
    return this.http.get<Subscription[]>(`/api/subscriptions${qs}`);
  }

  create(input: CreateSubscriptionInput): Promise<Subscription> {
    const workspace = this.ws(input.workspace);
    return this.http.post<Subscription>("/api/subscriptions", {
      workspace,
      scope: input.scope,
      ...(input.scopeValue !== undefined
        ? { scope_value: input.scopeValue }
        : {}),
      ...(input.categories !== undefined
        ? { categories: input.categories }
        : {}),
    });
  }
}

// ---------------------------------------------------------------------------
// Surface client (SSE)
// ---------------------------------------------------------------------------

export class SurfaceClient {
  constructor(
    private readonly http: HttpClient,
    private readonly _defaultWorkspace: string | undefined,
  ) {}

  /**
   * Open a server-sent event stream for the given subscription.
   *
   * @example
   * const stream = client.surface.stream(subscriptionId)
   * stream.on(ev => console.log(ev)).onError(console.error)
   * // later:
   * stream.close()
   */
  stream(subscriptionId: SubscriptionId): SurfaceStream {
    const url = `${this.http.baseUrl}/api/surface/stream?subscription=${subscriptionId}`;
    const headers = this.http.authHeaders();
    return createSurfaceStream(url, headers);
  }
}
