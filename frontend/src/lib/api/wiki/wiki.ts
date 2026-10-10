/**
 * Wiki API client
 * All endpoints live under /api/wiki and require an authenticated session
 * (better-auth cookie). CSRF is handled by the shared request() helper.
 */

import { request } from "../base";
import type {
  WikiListResponse,
  WikiPage,
  CurateRequest,
  CurateResponse,
  RenderRequest,
  RenderResponse,
  VerifyRequest,
  VerifyResponse,
} from "./types";

/**
 * List all wiki-tagged pages for the current tenant.
 * tenant_id defaults to user.id on the backend when omitted.
 */
export async function listWikiPages(
  tenantId?: string,
): Promise<WikiListResponse> {
  const params = new URLSearchParams();
  if (tenantId) params.set("tenant_id", tenantId);
  const qs = params.toString();
  return request<WikiListResponse>(`/wiki${qs ? `?${qs}` : ""}`, {
    skipCache: true,
  });
}

/**
 * Fetch the latest version of a wiki page by slug.
 *
 * @param slug       - Page slug (share_id or slugified name)
 * @param audience   - Audience filter (e.g. "default")
 * @param tenantId   - Optional tenant override
 */
export async function getWikiPage(
  slug: string,
  audience?: string,
  tenantId?: string,
): Promise<WikiPage> {
  const params = new URLSearchParams();
  if (audience) params.set("audience", audience);
  if (tenantId) params.set("tenant_id", tenantId);
  const qs = params.toString();
  return request<WikiPage>(
    `/wiki/${encodeURIComponent(slug)}${qs ? `?${qs}` : ""}`,
    {
      skipCache: true,
    },
  );
}

/**
 * Curate a page — writes body + citations into the backend wiki layer.
 * The backend writes through to contexts.properties.wiki_*.
 */
export async function curateWikiPage(
  data: CurateRequest,
): Promise<CurateResponse> {
  return request<CurateResponse>("/wiki/curate", {
    method: "POST",
    body: data,
  });
}

/**
 * Render a wiki page in the requested format.
 *
 * @param slug    - Page slug
 * @param data    - Audience + format (plain|markdown|claude|openai)
 */
export async function renderWikiPage(
  slug: string,
  data: RenderRequest,
): Promise<RenderResponse> {
  return request<RenderResponse>(`/wiki/${encodeURIComponent(slug)}/render`, {
    method: "POST",
    body: data,
  });
}

/**
 * Verify a wiki page against an optional schema.
 * Returns an ok flag and a list of issues with severity badges.
 *
 * @param slug    - Page slug
 * @param data    - Audience + optional schema constraints
 */
export async function verifyWikiPage(
  slug: string,
  data: VerifyRequest,
): Promise<VerifyResponse> {
  return request<VerifyResponse>(`/wiki/${encodeURIComponent(slug)}/verify`, {
    method: "POST",
    body: data,
  });
}
