import { HttpClient } from "./http.js";
import type {
  RecallActionsOptions,
  RecallOwnsOptions,
  RecallResult,
  RecallWhenOptions,
  RecallWhereOptions,
  RecallWhoOptions,
} from "./types.js";

export class RecallClient {
  constructor(
    private readonly http: HttpClient,
    private readonly defaultWorkspace: string | undefined,
  ) {}

  private ws(override: string | undefined): string | undefined {
    return override ?? this.defaultWorkspace;
  }

  actions(opts: RecallActionsOptions): Promise<RecallResult> {
    const workspace = this.ws(opts.workspace);
    const qs = HttpClient.buildQuery({
      topic: opts.topic,
      ...(opts.actor !== undefined ? { actor: opts.actor } : {}),
      ...(opts.since !== undefined ? { since: opts.since } : {}),
      ...(workspace !== undefined ? { workspace } : {}),
    });
    return this.http.get<RecallResult>(`/api/recall/actions${qs}`);
  }

  who(opts: RecallWhoOptions): Promise<RecallResult> {
    const workspace = this.ws(opts.workspace);
    const qs = HttpClient.buildQuery({
      topic: opts.topic,
      ...(opts.role !== undefined ? { role: opts.role } : {}),
      ...(workspace !== undefined ? { workspace } : {}),
    });
    return this.http.get<RecallResult>(`/api/recall/who${qs}`);
  }

  when(opts: RecallWhenOptions): Promise<RecallResult> {
    const workspace = this.ws(opts.workspace);
    const qs = HttpClient.buildQuery({
      event: opts.event,
      ...(workspace !== undefined ? { workspace } : {}),
    });
    return this.http.get<RecallResult>(`/api/recall/when${qs}`);
  }

  where(opts: RecallWhereOptions): Promise<RecallResult> {
    const workspace = this.ws(opts.workspace);
    const qs = HttpClient.buildQuery({
      thing: opts.thing,
      ...(workspace !== undefined ? { workspace } : {}),
    });
    return this.http.get<RecallResult>(`/api/recall/where${qs}`);
  }

  owns(opts: RecallOwnsOptions): Promise<RecallResult> {
    const workspace = this.ws(opts.workspace);
    const qs = HttpClient.buildQuery({
      actor: opts.actor,
      ...(workspace !== undefined ? { workspace } : {}),
    });
    return this.http.get<RecallResult>(`/api/recall/owns${qs}`);
  }
}
