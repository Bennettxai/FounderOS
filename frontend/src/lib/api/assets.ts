// Assets API client - the per-workspace brand/media library (images, logos,
// videos, documents, links shared across projects and campaigns). Goes through
// request<T>() which handles the API base, session cookie, CSRF, and the
// X-Workspace-ID header. Uploads use uploadRequest() for multipart/form-data.
import { request, getApiBaseUrl, getCSRFToken } from "./base";

// Active workspace id, read straight from localStorage like request() does.
function currentWorkspaceId(): string | null {
  if (typeof localStorage === "undefined") return null;
  return localStorage.getItem("businessos_current_workspace_id");
}

export type AssetKind =
  | "image"
  | "logo"
  | "video"
  | "document"
  | "file"
  | "link";

export interface Asset {
  id: string;
  name: string;
  kind: AssetKind;
  url: string;
  mime_type: string;
  size_bytes: number;
  tags: string;
  created_by: string | null;
  created_at: string;
  updated_at: string;
}

export interface AssetLinkInput {
  name: string;
  kind: AssetKind;
  url: string;
  tags: string;
}

export async function getAssets(
  q?: string,
): Promise<{ assets: Asset[]; count: number }> {
  const query = q ? `?q=${encodeURIComponent(q)}` : "";
  return request<{ assets: Asset[]; count: number }>(`/assets${query}`);
}

export async function createAssetLink(input: AssetLinkInput): Promise<Asset> {
  return request<Asset>(`/assets`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateAsset(
  id: string,
  input: { name?: string; tags?: string; kind?: AssetKind },
): Promise<Asset> {
  return request<Asset>(`/assets/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export async function deleteAsset(id: string): Promise<{ message: string }> {
  return request<{ message: string }>(`/assets/${id}`, { method: "DELETE" });
}

// uploadAsset posts a real file as multipart/form-data. We hit the backend
// directly (not request(), which JSON-encodes) but reuse the session cookie,
// CSRF header, and X-Workspace-ID so auth + scoping match every other call.
export async function uploadAsset(
  file: File,
  meta: { name?: string; kind?: AssetKind; tags?: string } = {},
): Promise<Asset> {
  const form = new FormData();
  form.append("file", file);
  if (meta.name) form.append("name", meta.name);
  if (meta.kind) form.append("kind", meta.kind);
  if (meta.tags) form.append("tags", meta.tags);

  const headers: Record<string, string> = {};
  const wsId = currentWorkspaceId();
  if (wsId) headers["X-Workspace-ID"] = wsId;
  const csrf = getCSRFToken();
  if (csrf) headers["X-CSRF-Token"] = csrf;

  // Same base + version prefix as request(), so it goes through the dev proxy
  // (/api/v1/*). We don't set Content-Type — the browser adds the multipart
  // boundary itself.
  const res = await fetch(`${getApiBaseUrl()}/assets/upload`, {
    method: "POST",
    credentials: "include",
    headers,
    body: form,
  });
  if (!res.ok) {
    let msg = `Upload failed (${res.status})`;
    try {
      const j = await res.json();
      if (j?.error) msg = j.error;
    } catch {
      /* ignore */
    }
    throw new Error(msg);
  }
  return res.json();
}

// The src for rendering an asset. Link assets carry an external URL; uploaded
// assets are served from /api/v1/assets/:id/raw. We rebuild that from the
// versioned API base so it flows through the dev proxy (same-origin → the
// session cookie is sent, which is how the raw endpoint authenticates).
export function assetSrc(a: Asset): string {
  if (a.url.startsWith("http://") || a.url.startsWith("https://")) return a.url;
  if (a.kind === "link") return a.url;
  return `${getApiBaseUrl()}/assets/${a.id}/raw`;
}
