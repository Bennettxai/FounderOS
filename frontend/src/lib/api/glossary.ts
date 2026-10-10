// Glossary API client - the per-workspace business glossary (term + definition).
// Goes through request<T>() which handles the API base, session cookie, CSRF,
// and the X-Workspace-ID header.
import { request } from "./base";

export interface GlossaryTerm {
  id: string;
  term: string;
  definition: string;
  category: string | null;
  aliases: string | null;
  created_by: string | null;
  created_at: string;
  updated_at: string;
}

export interface GlossaryTermInput {
  term: string;
  definition: string;
  category?: string | null;
  aliases?: string | null;
}

export async function getGlossary(
  q?: string,
): Promise<{ terms: GlossaryTerm[]; count: number }> {
  const query = q ? `?q=${encodeURIComponent(q)}` : "";
  return request<{ terms: GlossaryTerm[]; count: number }>(
    `/glossary${query}`,
    { skipCache: true },
  );
}

export function createGlossaryTerm(
  body: GlossaryTermInput,
): Promise<GlossaryTerm> {
  return request<GlossaryTerm>(`/glossary`, { method: "POST", body });
}

export function updateGlossaryTerm(
  id: string,
  body: GlossaryTermInput,
): Promise<GlossaryTerm> {
  return request<GlossaryTerm>(`/glossary/${id}`, { method: "PUT", body });
}

export function deleteGlossaryTerm(id: string): Promise<unknown> {
  return request(`/glossary/${id}`, { method: "DELETE" });
}
