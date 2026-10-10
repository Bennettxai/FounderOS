import { HttpClient } from "./http.js";
import type {
  CreateWorkspaceInput,
  ListWorkspacesOptions,
  Workspace,
  WorkspaceConfig,
  WorkspaceId,
} from "./types.js";

export class WorkspaceClient {
  constructor(private readonly http: HttpClient) {}

  list(opts: ListWorkspacesOptions = {}): Promise<Workspace[]> {
    const qs = HttpClient.buildQuery({
      ...(opts.tenant !== undefined ? { tenant: opts.tenant } : {}),
    });
    return this.http.get<Workspace[]>(`/api/workspaces${qs}`);
  }

  create(input: CreateWorkspaceInput): Promise<Workspace> {
    return this.http.post<Workspace>("/api/workspaces", {
      slug: input.slug,
      name: input.name,
      ...(input.description !== undefined
        ? { description: input.description }
        : {}),
      ...(input.tenant !== undefined ? { tenant: input.tenant } : {}),
    });
  }

  get(id: WorkspaceId): Promise<Workspace> {
    return this.http.get<Workspace>(`/api/workspaces/${id}`);
  }

  config(id: WorkspaceId): Promise<WorkspaceConfig> {
    return this.http.get<WorkspaceConfig>(`/api/workspaces/${id}/config`);
  }

  updateConfig(
    id: WorkspaceId,
    patch: Partial<WorkspaceConfig>,
  ): Promise<WorkspaceConfig> {
    return this.http.patch<WorkspaceConfig>(
      `/api/workspaces/${id}/config`,
      patch,
    );
  }
}
