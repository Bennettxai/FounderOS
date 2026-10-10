// Personas API client - the per-workspace buyer/customer personas (ideal
// customer profiles). Goes through request<T>() which handles the API base,
// session cookie, CSRF, and the X-Workspace-ID header.
import { request } from "./base";

// best-fit vs poor-fit ICP, per the offer one-pager's "Who it is for".
export type PersonaFit = "best" | "poor";

export interface Persona {
  id: string;
  name: string;
  segment: string;
  fit: PersonaFit;
  pains: string;
  objections: string;
  language: string;
  notes: string;
  created_by: string | null;
  created_at: string;
  updated_at: string;
}

export interface PersonaInput {
  name: string;
  segment: string;
  fit: PersonaFit;
  pains: string;
  objections: string;
  language: string;
  notes: string;
}

export async function getPersonas(
  q?: string,
): Promise<{ personas: Persona[]; count: number }> {
  const query = q ? `?q=${encodeURIComponent(q)}` : "";
  return request<{ personas: Persona[]; count: number }>(`/personas${query}`, {
    skipCache: true,
  });
}

export function createPersona(body: PersonaInput): Promise<Persona> {
  return request<Persona>(`/personas`, { method: "POST", body });
}

export function updatePersona(
  id: string,
  body: PersonaInput,
): Promise<Persona> {
  return request<Persona>(`/personas/${id}`, { method: "PUT", body });
}

export function deletePersona(id: string): Promise<unknown> {
  return request(`/personas/${id}`, { method: "DELETE" });
}
