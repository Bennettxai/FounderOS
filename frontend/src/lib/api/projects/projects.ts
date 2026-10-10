import { getBackendUrl, request } from "../base";
import type {
  Project,
  CreateProjectData,
  ProjectFileLink,
  ProjectNote,
} from "./types";

export async function getProjects(
  status?: string,
  clientId?: string,
  workspaceId?: string,
): Promise<Project[]> {
  const params = new URLSearchParams();
  if (status) params.set("status_filter", status);
  if (clientId) params.set("client_id", clientId);
  const query = params.toString();
  const raw = await request<Record<string, unknown>>(
    `/projects${query ? `?${query}` : ""}`,
    {
      headers: workspaceId ? { "X-Workspace-ID": workspaceId } : {},
      skipCache: true,
    },
  );

  // Backend returns paginated format { data: [...], pagination: {...} }
  if (
    raw &&
    typeof raw === "object" &&
    "data" in raw &&
    Array.isArray(raw.data)
  ) {
    return raw.data as Project[];
  }
  if (Array.isArray(raw)) {
    return raw as unknown as Project[];
  }
  return [];
}

export async function getProject(id: string): Promise<Project> {
  return request<Project>(`/projects/${id}`);
}

export async function createProject(data: CreateProjectData): Promise<Project> {
  return request<Project>("/projects", { method: "POST", body: data });
}

export async function updateProject(
  id: string,
  data: Partial<CreateProjectData>,
): Promise<Project> {
  return request<Project>(`/projects/${id}`, { method: "PUT", body: data });
}

export async function deleteProject(id: string): Promise<void> {
  return request(`/projects/${id}`, { method: "DELETE" }) as unknown as void;
}

export async function addProjectNote(
  projectId: string,
  content: string,
): Promise<ProjectNote> {
  return request<ProjectNote>(`/projects/${projectId}/notes`, {
    method: "POST",
    body: { content },
  });
}

export async function uploadProjectFile(
  projectId: string,
  file: File,
): Promise<ProjectFileLink> {
  const formData = new FormData();
  formData.append("file", file);

  const response = await fetch(
    `${getBackendUrl()}/projects/${projectId}/files`,
    {
      method: "POST",
      credentials: "include",
      body: formData,
    },
  );

  if (!response.ok) {
    const error = await response
      .json()
      .catch(() => ({ error: "File upload failed" }));
    throw new Error(error.error || error.detail || "File upload failed");
  }

  const data = (await response.json()) as Partial<ProjectFileLink>;
  return {
    name: data.name || file.name,
    size: data.size || `${(file.size / 1024 / 1024).toFixed(1)} MB`,
    url: data.url,
  };
}

export async function addProjectFileLink(
  project: Project,
  file: File,
  existingLinks: ProjectFileLink[] = [],
): Promise<Project> {
  const uploaded = await uploadProjectFile(project.id, file);
  return updateProject(project.id, {
    project_metadata: {
      ...(project.project_metadata ?? {}),
      quick_links: [...existingLinks, uploaded],
    },
  });
}

export async function removeProjectFileLink(
  project: Project,
  fileIndex: number,
  existingLinks: ProjectFileLink[] = [],
): Promise<Project> {
  return updateProject(project.id, {
    project_metadata: {
      ...(project.project_metadata ?? {}),
      quick_links: existingLinks.filter((_, index) => index !== fileIndex),
    },
  });
}
