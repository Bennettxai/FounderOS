/**
 * Wiki API Types
 * Matches the backend wiki endpoints on port 8801.
 * A "wiki page" is a Page/Context row with wiki_* metadata written into
 * contexts.properties by the curate endpoint.
 */

// -------------------------------------------------------------------------
// Shared
// -------------------------------------------------------------------------

export interface WikiCitation {
  chunk_id: string;
  claim_hash?: string;
}

// -------------------------------------------------------------------------
// GET /api/wiki  (list)
// -------------------------------------------------------------------------

export interface WikiPageSummary {
  slug: string;
  tenant_id: string;
  version: number;
  audience: string;
  updated_at: string;
}

export interface WikiListResponse {
  pages: WikiPageSummary[];
}

// -------------------------------------------------------------------------
// GET /api/wiki/:slug  (fetch latest version)
// -------------------------------------------------------------------------

export interface WikiPage {
  slug: string;
  tenant_id: string;
  audience: string;
  version: number;
  body: string;
  citations: WikiCitation[];
  created_at: string;
  updated_at: string;
}

// -------------------------------------------------------------------------
// POST /api/wiki/curate
// -------------------------------------------------------------------------

export interface CurateRequest {
  tenant_id?: string;
  slug: string;
  audience: string;
  body: string;
  citations?: WikiCitation[];
}

export interface CurateResponse {
  slug: string;
  version: number;
  audience: string;
  updated_at: string;
}

// -------------------------------------------------------------------------
// POST /api/wiki/:slug/render
// -------------------------------------------------------------------------

export type RenderFormat = "plain" | "markdown" | "claude" | "openai";

export interface RenderRequest {
  audience: string;
  format: RenderFormat;
}

export interface RenderResponse {
  slug: string;
  body: string;
  format: RenderFormat;
  citations: WikiCitation[];
}

// -------------------------------------------------------------------------
// POST /api/wiki/:slug/verify
// -------------------------------------------------------------------------

export interface VerifySchema {
  required_sections?: string[];
  min_citations?: number;
  max_bytes?: number;
}

export interface VerifyRequest {
  audience: string;
  schema?: VerifySchema;
}

export type IssueSeverity = "error" | "warning" | "info";

export interface VerifyIssue {
  severity: IssueSeverity;
  code: string;
  message: string;
}

export interface VerifyResponse {
  slug: string;
  ok: boolean;
  issues: VerifyIssue[];
}
