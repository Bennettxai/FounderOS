// Knowledge Base client. Reads the workspace markdown tree + file bodies from the
// Go backend (/api/knowledge/*), proxied by Vite in dev. Workspace-scoped by slug.

import type { KBTreeNode, KBFile } from "./types";
import { getCSRFToken } from "$lib/api/base";

async function getJSON<T>(url: string): Promise<T> {
  const res = await fetch(url, { credentials: "include" });
  if (!res.ok) throw new Error(`Knowledge request failed: ${res.status}`);
  return res.json() as Promise<T>;
}

// Write (create or overwrite) a markdown file. Sends CSRF for the mutation.
export async function saveFile(
  workspace: string,
  path: string,
  content: string,
): Promise<{ ok: boolean; path: string; modified?: string }> {
  const res = await fetch("/api/knowledge/file", {
    method: "PUT",
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      "X-CSRF-Token": getCSRFToken() ?? "",
    },
    body: JSON.stringify({ workspace, path, content }),
  });
  if (!res.ok) throw new Error(`Save failed: ${res.status}`);
  return res.json();
}

// Per-document source map: where each doc lives - "engine" (your local 2nd
// brain), "cloud" (the shared synced copy), or "synced" (both). Lets the
// Knowledge module show engine + cloud together, tagged, so you decide what to sync.
export async function fetchSources(workspace: string): Promise<{
  sources: Record<string, string>;
  counts: { engine: number; cloud: number; synced: number };
}> {
  if (!workspace)
    return { sources: {}, counts: { engine: 0, cloud: 0, synced: 0 } };
  try {
    return await getJSON<{
      sources: Record<string, string>;
      counts: { engine: number; cloud: number; synced: number };
    }>(`/api/knowledge/sources?workspace=${encodeURIComponent(workspace)}`);
  } catch {
    return { sources: {}, counts: { engine: 0, cloud: 0, synced: 0 } };
  }
}

// Cloud storage usage for a workspace, in bytes. The backend defaults to a
// 1 GB limit and 0 used when the workspace has never synced.
export interface StorageUsage {
  bytes_used: number;
  bytes_limit: number;
  // Whether the user has opted in to cloud sync for this workspace. Local
  // knowledge is always on + free; cloud sync is the opt-in, paid-later layer.
  activated: boolean;
  // Soft flag: over the free cloud quota. Never blocks (billing not live yet).
  over_limit: boolean;
}

// Result of a cloud sync. Backend now returns post-sync storage usage alongside
// the doc count so the UI can update the usage meter without a second call.
export interface CloudSyncResult {
  workspace: string;
  documents: number;
  synced_by?: string;
  storage?: StorageUsage;
}

const DEFAULT_STORAGE_LIMIT = 1073741824; // 1 GB

// Human-readable byte size (B/KB/MB/GB), for storage meters + messages.
export function formatBytes(n: number): string {
  if (!Number.isFinite(n) || n <= 0) return "0 MB";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let val = n;
  let i = 0;
  while (val >= 1024 && i < units.length - 1) {
    val /= 1024;
    i++;
  }
  const rounded =
    val < 10 && i > 0 ? val.toFixed(1) : Math.round(val).toString();
  return `${rounded} ${units[i]}`;
}

// Cloud storage usage for a workspace. Falls back to the backend defaults
// (0 used, 1 GB limit) when fields are missing.
export async function getStorage(workspace: string): Promise<StorageUsage> {
  const data = await getJSON<{
    workspace: string;
    bytes_used: number;
    bytes_limit: number;
    activated?: boolean;
    over_limit?: boolean;
  }>(`/api/knowledge/storage?workspace=${encodeURIComponent(workspace)}`);
  return {
    bytes_used: data.bytes_used ?? 0,
    bytes_limit: data.bytes_limit ?? DEFAULT_STORAGE_LIMIT,
    activated: data.activated ?? false,
    over_limit: data.over_limit ?? false,
  };
}

// Opt in to cloud sync for a workspace. Until this is called, nothing syncs to
// the cloud (the backend refuses SyncToCloud with a 403 NOT_ACTIVATED). Returns
// once the gate is flipped on server-side.
export async function activateCloudSync(
  workspace: string,
): Promise<{ activated: boolean; workspace: string }> {
  const res = await fetch(
    `/api/knowledge/cloud/activate?workspace=${encodeURIComponent(workspace)}`,
    {
      method: "POST",
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
        "X-CSRF-Token": getCSRFToken() ?? "",
      },
    },
  );
  if (!res.ok) {
    let msg = `Activation failed: ${res.status}`;
    try {
      const e = await res.json();
      if (
        e?.error &&
        typeof e.error === "object" &&
        typeof e.error.message === "string"
      ) {
        msg = e.error.message;
      } else if (typeof e?.error === "string") {
        msg = e.error;
      }
    } catch {
      /* keep default */
    }
    throw new Error(msg);
  }
  return res.json();
}

// Push this workspace's local knowledge up to the shared cloud copy so
// teammates without their own engine can view it. Owner-only on the backend.
export async function syncToCloud(
  workspace: string,
  workspaceId?: string,
): Promise<CloudSyncResult> {
  const res = await fetch("/api/knowledge/sync-to-cloud", {
    method: "POST",
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      "X-CSRF-Token": getCSRFToken() ?? "",
    },
    body: JSON.stringify({ workspace, workspace_id: workspaceId }),
  });
  if (!res.ok) {
    let msg = `Sync failed: ${res.status}`;
    try {
      const e = await res.json();
      // The quota error (413) returns a structured object:
      // { error: { code: "STORAGE_LIMIT", message, bytes_used, bytes_limit } }.
      // Surface a clear, actionable message instead of a generic failure.
      if (e?.error && typeof e.error === "object") {
        if (e.error.code === "STORAGE_LIMIT") {
          msg =
            `Cloud storage full - ${formatBytes(e.error.bytes_used ?? 0)} of ` +
            `${formatBytes(e.error.bytes_limit ?? 0)} used. Free up space or upgrade.`;
        } else if (typeof e.error.message === "string") {
          msg = e.error.message;
        }
      } else if (typeof e?.error === "string") {
        msg = e.error;
      }
    } catch {
      /* keep default */
    }
    throw new Error(msg);
  }
  return res.json();
}

// Slugify a title into a filename-safe slug.
export function slugify(s: string): string {
  return s
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "");
}

export interface KBWorkspace {
  slug: string;
  name: string;
}

export async function fetchWorkspaces(): Promise<KBWorkspace[]> {
  const data = await getJSON<{ workspaces: KBWorkspace[] }>(
    `/api/knowledge/workspaces`,
  );
  return data.workspaces ?? [];
}

export async function fetchTree(workspace: string): Promise<KBTreeNode[]> {
  if (!workspace) return [];
  const data = await getJSON<{ tree: KBTreeNode[] }>(
    `/api/knowledge/tree?workspace=${encodeURIComponent(workspace)}`,
  );
  return data.tree ?? [];
}

export async function fetchFile(
  workspace: string,
  path: string,
): Promise<KBFile> {
  return getJSON<KBFile>(
    `/api/knowledge/file?workspace=${encodeURIComponent(workspace)}&path=${encodeURIComponent(path)}`,
  );
}

// Split YAML frontmatter from a markdown body. Returns the raw frontmatter block
// (without fences) and the remaining markdown.
export function splitFrontmatter(content: string): {
  frontmatter: string;
  body: string;
} {
  const m = content.match(/^---\n([\s\S]*?)\n---\n?([\s\S]*)$/);
  if (m) return { frontmatter: m[1], body: m[2] };
  return { frontmatter: "", body: content };
}

// Tiny frontmatter parser (key: value), enough to show title/status as properties.
export function parseFrontmatter(fm: string): Record<string, string> {
  const out: Record<string, string> = {};
  for (const line of fm.split("\n")) {
    const i = line.indexOf(":");
    if (i > 0) {
      const k = line.slice(0, i).trim();
      const v = line
        .slice(i + 1)
        .trim()
        .replace(/^["']|["']$/g, "");
      if (k) out[k] = v;
    }
  }
  return out;
}
