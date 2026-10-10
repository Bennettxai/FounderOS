import { HttpClient } from "./http.js";
import type {
  RenderContextOptions,
  RenderContextResult,
  ListRenderSpecsOptions,
  RenderSpecsResult,
} from "./types.js";

export class RenderClient {
  constructor(
    private readonly http: HttpClient,
    private readonly defaultWorkspace: string | undefined,
  ) {}

  private ws(override: string | undefined): string | undefined {
    return override ?? this.defaultWorkspace;
  }

  context(
    source: string,
    spec: string,
    opts: RenderContextOptions = {},
  ): Promise<RenderContextResult> {
    const workspace = this.ws(opts.workspace);
    return this.http.post<RenderContextResult>("/api/render/context", {
      source,
      spec,
      ...(workspace !== undefined ? { workspace } : {}),
    });
  }

  specs(opts: ListRenderSpecsOptions = {}): Promise<RenderSpecsResult> {
    const workspace = this.ws(opts.workspace);
    const qs = HttpClient.buildQuery({
      ...(workspace !== undefined ? { workspace } : {}),
    });
    return this.http.get<RenderSpecsResult>(`/api/render/specs${qs}`);
  }
}
