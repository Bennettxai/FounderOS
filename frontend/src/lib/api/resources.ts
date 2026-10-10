// Resources API client — the per-workspace library of links, docs, SOPs, and
// tools the team relies on. Goes through request<T>() which handles the API
// base, session cookie, CSRF, and the X-Workspace-ID header.
import { request } from "./base";

export interface Resource {
  id: string;
  title: string;
  url: string;
  category: string;
  notes: string;
  created_by: string | null;
  created_at: string;
  updated_at: string;
}

export interface ResourceInput {
  title: string;
  url: string;
  category: string;
  notes: string;
}

export async function getResources(
  q?: string,
): Promise<{ resources: Resource[]; count: number }> {
  const query = q ? `?q=${encodeURIComponent(q)}` : "";
  return request<{ resources: Resource[]; count: number }>(
    `/resources${query}`,
  );
}

export async function createResource(input: ResourceInput): Promise<Resource> {
  return request<Resource>(`/resources`, { method: "POST", body: input });
}

export async function updateResource(
  id: string,
  input: ResourceInput,
): Promise<Resource> {
  return request<Resource>(`/resources/${id}`, { method: "PUT", body: input });
}

export async function deleteResource(
  id: string,
): Promise<{ message: string }> {
  return request<{ message: string }>(`/resources/${id}`, {
    method: "DELETE",
  });
}
