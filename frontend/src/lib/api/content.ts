// Content API client - the per-workspace content pipeline (posts, reels,
// newsletters, podcasts, threads, articles) tracked through their lifecycle.
// Goes through request<T>() which handles the API base, session cookie, CSRF,
// and the X-Workspace-ID header.
import { request } from "./base";

export type ContentType =
  | "script"
  | "copywriting"
  | "carousel"
  | "image"
  | "video"
  | "post"
  | "reel"
  | "story"
  | "newsletter"
  | "podcast"
  | "thread"
  | "article"
  | "other";

export type ContentStatus =
  | "idea"
  | "scripting"
  | "to_film"
  | "to_edit"
  | "client_review"
  | "approved"
  | "to_post"
  | "posted"
  | "archive"
  | "draft"
  | "scheduled"
  | "published";

export type ContentPriority = "low" | "normal" | "high" | "urgent";

export interface ContentItem {
  id: string;
  title: string;
  content_type: ContentType;
  status: ContentStatus;
  hook: string;
  body: string;
  caption: string;
  cta: string;
  channel: string;
  link: string;
  category: string;
  theme: string;
  client: string;
  campaign: string;
  owner: string;
  editor: string;
  priority: ContentPriority;
  due_date: string;
  publish_date: string;
  asset_link: string;
  review_link: string;
  revision_notes: string;
  notes: string;
  views: number;
  reach: number;
  likes: number;
  comments: number;
  saves: number;
  shares: number;
  reposts: number;
  follows: number;
  profile_activity: number;
  accounts_engaged: number;
  avg_watch_time_seconds: number;
  retention_rate: number;
  analytics_notes: string;
  created_by: string | null;
  created_at: string;
  updated_at: string;
}

export interface ContentItemInput {
  title: string;
  content_type: ContentType;
  status: ContentStatus;
  hook: string;
  body: string;
  caption: string;
  cta: string;
  channel: string;
  link: string;
  category: string;
  theme: string;
  client: string;
  campaign: string;
  owner: string;
  editor: string;
  priority: ContentPriority;
  due_date: string;
  publish_date: string;
  asset_link: string;
  review_link: string;
  revision_notes: string;
  notes: string;
  views?: number;
  reach?: number;
  likes?: number;
  comments?: number;
  saves?: number;
  shares?: number;
  reposts?: number;
  follows?: number;
  profile_activity?: number;
  accounts_engaged?: number;
  avg_watch_time_seconds?: number;
  retention_rate?: number;
  analytics_notes?: string;
}

function workspaceHeaders(workspaceId?: string | null): Record<string, string> {
  return workspaceId ? { "X-Workspace-ID": workspaceId } : {};
}

export async function getContent(
  q?: string,
  workspaceId?: string | null,
): Promise<{ items: ContentItem[]; count: number }> {
  const query = q ? `?q=${encodeURIComponent(q)}` : "";
  return request<{ items: ContentItem[]; count: number }>(`/content${query}`, {
    skipCache: true,
    headers: workspaceHeaders(workspaceId),
  });
}

export function createContentItem(
  body: ContentItemInput,
  workspaceId?: string | null,
): Promise<ContentItem> {
  return request<ContentItem>(`/content`, {
    method: "POST",
    body,
    headers: workspaceHeaders(workspaceId),
  });
}

export function updateContentItem(
  id: string,
  body: ContentItemInput,
  workspaceId?: string | null,
): Promise<ContentItem> {
  return request<ContentItem>(`/content/${id}`, {
    method: "PUT",
    body,
    headers: workspaceHeaders(workspaceId),
  });
}

export function deleteContentItem(
  id: string,
  workspaceId?: string | null,
): Promise<unknown> {
  return request(`/content/${id}`, {
    method: "DELETE",
    headers: workspaceHeaders(workspaceId),
  });
}
