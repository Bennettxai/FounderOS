// Deliverables API client - the per-workspace catalog of client deliverables and
// packaged outputs (documents, decks, packages, scripts) tied to a client and
// project/engagement, tracking what has been produced and sent. Goes through
// request<T>() which handles the API base, session cookie, CSRF, and the
// X-Workspace-ID header, mirroring the offers/projects clients.
import { request } from "./base";

// Where a deliverable sits in its production lifecycle.
export type DeliverableStatus = "draft" | "in_progress" | "delivered";

// The kind of packaged output. Mirrors the deliverable types the agency ships.
export type DeliverableKind =
  "package" | "document" | "deck" | "script" | "report" | "video" | "other";

export interface Deliverable {
  id: string;
  title: string;
  kind: DeliverableKind;
  status: DeliverableStatus;
  // Free-text so a deliverable can be tracked before the client/project record
  // exists. The backend may later promote these to client_id / project_id.
  client: string;
  project: string;
  // A link to the produced artifact (Drive folder, doc, deployed page, zip).
  link: string;
  description: string;
  created_by: string | null;
  created_at: string;
  updated_at: string;
}

export interface DeliverableInput {
  title: string;
  kind: DeliverableKind;
  status: DeliverableStatus;
  client: string;
  project: string;
  link: string;
  description: string;
}

export async function getDeliverables(
  q?: string,
): Promise<{ deliverables: Deliverable[]; count: number }> {
  const query = q ? `?q=${encodeURIComponent(q)}` : "";
  return request<{ deliverables: Deliverable[]; count: number }>(
    `/deliverables${query}`,
    { skipCache: true },
  );
}

export function createDeliverable(
  body: DeliverableInput,
): Promise<Deliverable> {
  return request<Deliverable>(`/deliverables`, { method: "POST", body });
}

export function updateDeliverable(
  id: string,
  body: DeliverableInput,
): Promise<Deliverable> {
  return request<Deliverable>(`/deliverables/${id}`, { method: "PUT", body });
}

export function deleteDeliverable(id: string): Promise<unknown> {
  return request(`/deliverables/${id}`, { method: "DELETE" });
}
