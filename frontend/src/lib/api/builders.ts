// Builders API client - the per-workspace registry of builder tools: the tools
// for creating forms, flows, automations, apps, and sites. Each builder is a
// catalog entry (name, kind, status, config) so the team and AI agents share one
// source of truth for what can be built. Goes through request<T>() which handles
// the API base, session cookie, CSRF, and the X-Workspace-ID header.
import { request } from "./base";

export type BuilderKind = "form" | "flow" | "automation" | "app" | "site";
export type BuilderStatus = "draft" | "active" | "archived";

export interface Builder {
  id: string;
  name: string;
  kind: BuilderKind;
  description: string;
  config: unknown;
  status: BuilderStatus;
  created_by: string | null;
  created_at: string;
  updated_at: string;
}

export interface BuilderInput {
  name: string;
  kind: BuilderKind;
  description: string;
  status: BuilderStatus;
}

export async function getBuilders(
  q?: string,
  kind?: BuilderKind,
): Promise<{ builders: Builder[]; count: number }> {
  const params = new URLSearchParams();
  if (q) params.set("q", q);
  if (kind) params.set("kind", kind);
  const query = params.toString() ? `?${params.toString()}` : "";
  return request<{ builders: Builder[]; count: number }>(`/builders${query}`);
}

export async function createBuilder(input: BuilderInput): Promise<Builder> {
  return request<Builder>(`/builders`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateBuilder(
  id: string,
  input: BuilderInput,
): Promise<Builder> {
  return request<Builder>(`/builders/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export async function deleteBuilder(id: string): Promise<{ message: string }> {
  return request<{ message: string }>(`/builders/${id}`, { method: "DELETE" });
}
