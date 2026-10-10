import { HttpClient } from "./http.js";
import type {
  AskOptions,
  AskResult,
  GrepOptions,
  GrepResult,
  ProfileOptions,
  ProfileResult,
  SearchOptions,
  SearchResult,
} from "./types.js";

export class RetrievalClient {
  constructor(
    private readonly http: HttpClient,
    private readonly defaultWorkspace: string | undefined,
  ) {}

  private ws(override: string | undefined): string | undefined {
    return override ?? this.defaultWorkspace;
  }

  ask(query: string, opts: AskOptions = {}): Promise<AskResult> {
    const workspace = this.ws(opts.workspace);
    return this.http.post<AskResult>("/api/rag", {
      query,
      ...(workspace !== undefined ? { workspace } : {}),
      ...(opts.audience !== undefined ? { audience: opts.audience } : {}),
      ...(opts.format !== undefined ? { format: opts.format } : {}),
      ...(opts.bandwidth !== undefined ? { bandwidth: opts.bandwidth } : {}),
    });
  }

  search(query: string, opts: SearchOptions = {}): Promise<SearchResult> {
    const workspace = this.ws(opts.workspace);
    const qs = HttpClient.buildQuery({
      q: query,
      ...(workspace !== undefined ? { workspace } : {}),
      ...(opts.limit !== undefined ? { limit: opts.limit } : {}),
    });
    return this.http.get<SearchResult>(`/api/search${qs}`);
  }

  grep(query: string, opts: GrepOptions = {}): Promise<GrepResult> {
    const workspace = this.ws(opts.workspace);
    const qs = HttpClient.buildQuery({
      q: query,
      ...(workspace !== undefined ? { workspace } : {}),
      ...(opts.intent !== undefined ? { intent: opts.intent } : {}),
      ...(opts.scale !== undefined ? { scale: opts.scale } : {}),
      ...(opts.modality !== undefined ? { modality: opts.modality } : {}),
      ...(opts.limit !== undefined ? { limit: opts.limit } : {}),
      ...(opts.literal !== undefined ? { literal: opts.literal } : {}),
    });
    return this.http.get<GrepResult>(`/api/grep${qs}`);
  }

  profile(opts: ProfileOptions = {}): Promise<ProfileResult> {
    const workspace = this.ws(opts.workspace);
    const qs = HttpClient.buildQuery({
      ...(workspace !== undefined ? { workspace } : {}),
      ...(opts.audience !== undefined ? { audience: opts.audience } : {}),
      ...(opts.bandwidth !== undefined ? { bandwidth: opts.bandwidth } : {}),
      ...(opts.node !== undefined ? { node: opts.node } : {}),
    });
    return this.http.get<ProfileResult>(`/api/profile${qs}`);
  }
}
