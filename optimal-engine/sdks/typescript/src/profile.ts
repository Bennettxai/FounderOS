import { HttpClient } from "./http.js";
import type {
  GetWikiOptions,
  ListWikiOptions,
  WikiArticle,
  WikiContradiction,
  WikiSlug,
} from "./types.js";

export class WikiClient {
  constructor(
    private readonly http: HttpClient,
    private readonly defaultWorkspace: string | undefined,
  ) {}

  private ws(override: string | undefined): string | undefined {
    return override ?? this.defaultWorkspace;
  }

  list(opts: ListWikiOptions = {}): Promise<WikiArticle[]> {
    const workspace = this.ws(opts.workspace);
    const qs = HttpClient.buildQuery({
      ...(workspace !== undefined ? { workspace } : {}),
    });
    return this.http.get<WikiArticle[]>(`/api/wiki${qs}`);
  }

  get(slug: WikiSlug, opts: GetWikiOptions = {}): Promise<WikiArticle> {
    const workspace = this.ws(opts.workspace);
    const qs = HttpClient.buildQuery({
      ...(workspace !== undefined ? { workspace } : {}),
      ...(opts.audience !== undefined ? { audience: opts.audience } : {}),
      ...(opts.format !== undefined ? { format: opts.format } : {}),
    });
    return this.http.get<WikiArticle>(`/api/wiki/${slug}${qs}`);
  }

  contradictions(opts: ListWikiOptions = {}): Promise<WikiContradiction[]> {
    const workspace = this.ws(opts.workspace);
    const qs = HttpClient.buildQuery({
      ...(workspace !== undefined ? { workspace } : {}),
    });
    return this.http.get<WikiContradiction[]>(`/api/wiki/contradictions${qs}`);
  }
}
