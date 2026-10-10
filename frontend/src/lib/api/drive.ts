// Drive API client - the per-workspace file store (files organized into simple
// folder paths, shared across the workspace). Goes through request<T>() which
// handles the API base, session cookie, CSRF, and the X-Workspace-ID header.
// Uploads use a direct multipart/form-data fetch, mirroring the assets client.
import { request, getApiBaseUrl, getCSRFToken } from "./base";

// Active workspace id, read straight from localStorage like request() does.
function currentWorkspaceId(): string | null {
  if (typeof localStorage === "undefined") return null;
  return localStorage.getItem("businessos_current_workspace_id");
}

export interface DriveFile {
  id: string;
  name: string;
  folder: string;
  mime_type: string;
  size_bytes: number;
  url: string;
  created_by: string | null;
  created_at: string;
  updated_at: string;
}

export async function getDriveFiles(opts: {
  folder?: string;
  q?: string;
} = {}): Promise<{ files: DriveFile[]; count: number }> {
  const qs = new URLSearchParams();
  if (opts.folder !== undefined) qs.set("folder", opts.folder);
  if (opts.q) qs.set("q", opts.q);
  const query = qs.toString();
  return request<{ files: DriveFile[]; count: number }>(
    `/drive${query ? `?${query}` : ""}`,
  );
}

export async function getDriveFolders(): Promise<{
  folders: string[];
  count: number;
}> {
  return request<{ folders: string[]; count: number }>(`/drive/folders`);
}

export async function updateDriveFile(
  id: string,
  input: { name?: string; folder?: string },
): Promise<DriveFile> {
  return request<DriveFile>(`/drive/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export async function deleteDriveFile(
  id: string,
): Promise<{ message: string }> {
  return request<{ message: string }>(`/drive/${id}`, { method: "DELETE" });
}

// uploadDriveFile posts a real file as multipart/form-data. We hit the backend
// directly (not request(), which JSON-encodes) but reuse the session cookie,
// CSRF header, and X-Workspace-ID so auth + scoping match every other call.
export async function uploadDriveFile(
  file: File,
  meta: { name?: string; folder?: string } = {},
): Promise<DriveFile> {
  const form = new FormData();
  form.append("file", file);
  if (meta.name) form.append("name", meta.name);
  if (meta.folder) form.append("folder", meta.folder);

  const headers: Record<string, string> = {};
  const wsId = currentWorkspaceId();
  if (wsId) headers["X-Workspace-ID"] = wsId;
  const csrf = getCSRFToken();
  if (csrf) headers["X-CSRF-Token"] = csrf;

  // Same base + version prefix as request(), so it goes through the dev proxy
  // (/api/v1/*). We don't set Content-Type — the browser adds the multipart
  // boundary itself.
  const res = await fetch(`${getApiBaseUrl()}/drive/upload`, {
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

// The download/preview src for a file. Uploaded files are served from
// /api/v1/drive/:id/raw. We rebuild that from the versioned API base so it flows
// through the dev proxy (same-origin → the session cookie is sent, which is how
// the raw endpoint authenticates).
export function driveFileSrc(f: DriveFile): string {
  return `${getApiBaseUrl()}/drive/${f.id}/raw`;
}
