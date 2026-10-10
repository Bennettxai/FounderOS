import { HttpClient } from "./http.js";
import type {
  CreateMemoryInput,
  DeriveMemoryInput,
  ExtendMemoryInput,
  ForgetMemoryInput,
  ListMemoriesOptions,
  Memory,
  MemoryId,
  MemoryListResult,
  MemoryRelation,
  MemoryVersion,
  UpdateMemoryInput,
} from "./types.js";

export class MemoryClient {
  constructor(
    private readonly http: HttpClient,
    private readonly defaultWorkspace: string | undefined,
  ) {}

  private ws(override: string | undefined): string | undefined {
    return override ?? this.defaultWorkspace;
  }

  create(input: CreateMemoryInput): Promise<Memory> {
    const workspace = this.ws(input.workspace);
    return this.http.post<Memory>("/api/memory", {
      content: input.content,
      ...(workspace !== undefined ? { workspace } : {}),
      ...(input.isStatic !== undefined ? { is_static: input.isStatic } : {}),
      ...(input.audience !== undefined ? { audience: input.audience } : {}),
      ...(input.citationUri !== undefined
        ? { citation_uri: input.citationUri }
        : {}),
      ...(input.metadata !== undefined ? { metadata: input.metadata } : {}),
    });
  }

  get(id: MemoryId): Promise<Memory> {
    return this.http.get<Memory>(`/api/memory/${id}`);
  }

  list(opts: ListMemoriesOptions = {}): Promise<MemoryListResult> {
    const workspace = this.ws(opts.workspace);
    const qs = HttpClient.buildQuery({
      ...(workspace !== undefined ? { workspace } : {}),
      ...(opts.audience !== undefined ? { audience: opts.audience } : {}),
      ...(opts.includeForgotten !== undefined
        ? { include_forgotten: opts.includeForgotten }
        : {}),
      ...(opts.limit !== undefined ? { limit: opts.limit } : {}),
    });
    return this.http.get<MemoryListResult>(`/api/memory${qs}`);
  }

  forget(id: MemoryId, input: ForgetMemoryInput = {}): Promise<Memory> {
    return this.http.post<Memory>(`/api/memory/${id}/forget`, {
      ...(input.reason !== undefined ? { reason: input.reason } : {}),
      ...(input.forgetAfter !== undefined
        ? { forget_after: input.forgetAfter }
        : {}),
    });
  }

  update(id: MemoryId, input: UpdateMemoryInput): Promise<Memory> {
    return this.http.post<Memory>(`/api/memory/${id}/update`, input);
  }

  extend(id: MemoryId, input: ExtendMemoryInput): Promise<Memory> {
    return this.http.post<Memory>(`/api/memory/${id}/extend`, input);
  }

  derive(id: MemoryId, input: DeriveMemoryInput): Promise<Memory> {
    return this.http.post<Memory>(`/api/memory/${id}/derive`, input);
  }

  versions(id: MemoryId): Promise<MemoryVersion[]> {
    return this.http.get<MemoryVersion[]>(`/api/memory/${id}/versions`);
  }

  relations(id: MemoryId): Promise<MemoryRelation[]> {
    return this.http.get<MemoryRelation[]>(`/api/memory/${id}/relations`);
  }

  delete(id: MemoryId): Promise<void> {
    return this.http.delete<void>(`/api/memory/${id}`);
  }
}
