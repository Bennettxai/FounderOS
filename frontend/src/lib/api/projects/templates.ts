// Project Templates API client - reusable delivery blueprints (e.g. the Agency
// MIOSA "Growth Systems Audit"). A template carries phases + deliverables; using
// one materializes a real project with those phases snapshotted into metadata.
// Goes through request<T>() which handles API base, session cookie, CSRF, and
// the X-Workspace-ID header (templates are workspace-scoped + global built-ins).
import { request } from "../base";
import type { Project } from "./types";

export interface ProjectTemplatePhase {
  name: string;
  tasks?: string[];
  deliverables?: string[];
}

export interface ProjectTemplate {
  id: string;
  workspace_id: string | null;
  key: string;
  name: string;
  description: string;
  phases: ProjectTemplatePhase[];
  deliverables: string[];
  is_builtin: boolean;
  created_by: string | null;
  created_at: string;
  updated_at: string;
}

export interface UseTemplateInput {
  name?: string;
  client_name?: string;
  due_date?: string | null;
  priority?: "critical" | "high" | "medium" | "low";
}

export async function getProjectTemplates(): Promise<ProjectTemplate[]> {
  const raw = await request<{ templates: ProjectTemplate[]; count: number }>(
    "/projects/templates",
    { skipCache: true },
  );
  return raw?.templates ?? [];
}

export async function createProjectFromTemplate(
  key: string,
  body: UseTemplateInput = {},
): Promise<Project> {
  return request<Project>(
    `/projects/templates/${encodeURIComponent(key)}/use`,
    {
      method: "POST",
      body,
    },
  );
}
