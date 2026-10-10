import { request } from "../base";
import type {
  DSARExportRequest,
  DSARExportResponse,
  ErasurePreviewRequest,
  ErasurePreviewResponse,
  ErasureRequest,
  ErasureResponse,
  LegalHold,
  PlaceLegalHoldRequest,
  ListLegalHoldsResponse,
  RetentionPolicy,
  CreateRetentionPolicyRequest,
  ListRetentionPoliciesResponse,
  RetentionSweepRequest,
  RetentionSweepResponse,
  AuditQueryParams,
  AuditEventsResponse,
} from "./types";

// ============================================================================
// DSAR — Data Subject Access Request
// ============================================================================

/**
 * Export all data held for a principal under a tenant.
 * Returns structured counts + raw data blob.
 */
export async function exportDSAR(
  body: DSARExportRequest,
): Promise<DSARExportResponse> {
  return request<DSARExportResponse>("/compliance/dsar/export", {
    method: "POST",
    body,
    // Compliance exports can be slow on large datasets
    timeout: 60_000,
  });
}

// ============================================================================
// Erasure
// ============================================================================

/**
 * Preview what would be erased for a principal — returns counts + hold blockers.
 * Safe to call without modifying data.
 */
export async function previewErasure(
  body: ErasurePreviewRequest,
): Promise<ErasurePreviewResponse> {
  return request<ErasurePreviewResponse>("/compliance/erasure/preview", {
    method: "POST",
    body,
    timeout: 30_000,
  });
}

/**
 * Permanently erase all data for a principal.
 * Requires `confirm: true` in the request body — enforced at both API and UI level.
 */
export async function eraseData(
  body: ErasureRequest,
): Promise<ErasureResponse> {
  return request<ErasureResponse>("/compliance/erasure/erase", {
    method: "POST",
    body,
    timeout: 60_000,
  });
}

// ============================================================================
// Legal Holds
// ============================================================================

/**
 * List all active legal holds for a tenant.
 */
export async function listLegalHolds(
  tenantId: string,
): Promise<ListLegalHoldsResponse> {
  return request<ListLegalHoldsResponse>(
    `/compliance/holds?tenant_id=${encodeURIComponent(tenantId)}`,
    { skipCache: true },
  );
}

/**
 * Place a new legal hold on a signal.
 */
export async function placeLegalHold(
  body: PlaceLegalHoldRequest,
): Promise<LegalHold> {
  return request<LegalHold>("/compliance/holds", {
    method: "POST",
    body,
  });
}

/**
 * Release (delete) a legal hold by ID.
 */
export async function releaseLegalHold(holdId: string): Promise<void> {
  return request<void>(`/compliance/holds/${encodeURIComponent(holdId)}`, {
    method: "DELETE",
  });
}

// ============================================================================
// Retention Policies
// ============================================================================

/**
 * List all retention policies for a tenant.
 */
export async function listRetentionPolicies(
  tenantId: string,
): Promise<ListRetentionPoliciesResponse> {
  return request<ListRetentionPoliciesResponse>(
    `/compliance/retention/policies?tenant_id=${encodeURIComponent(tenantId)}`,
    { skipCache: true },
  );
}

/**
 * Create a new retention policy.
 */
export async function createRetentionPolicy(
  body: CreateRetentionPolicyRequest,
): Promise<RetentionPolicy> {
  return request<RetentionPolicy>("/compliance/retention/policies", {
    method: "POST",
    body,
  });
}

/**
 * Delete a retention policy by ID.
 */
export async function deleteRetentionPolicy(policyId: string): Promise<void> {
  return request<void>(
    `/compliance/retention/policies/${encodeURIComponent(policyId)}`,
    {
      method: "DELETE",
    },
  );
}

/**
 * Trigger a retention sweep. Defaults to dry_run: true for safety.
 */
export async function runRetentionSweep(
  body: RetentionSweepRequest,
): Promise<RetentionSweepResponse> {
  return request<RetentionSweepResponse>("/compliance/retention/sweep", {
    method: "POST",
    body,
    timeout: 60_000,
  });
}

// ============================================================================
// Audit Log
// ============================================================================

/**
 * Query audit events with optional filters.
 * Max 500 results per request (enforced server-side).
 */
export async function queryAuditEvents(
  params: AuditQueryParams,
): Promise<AuditEventsResponse> {
  const qs = new URLSearchParams();
  qs.set("tenant_id", params.tenant_id);
  if (params.principal) qs.set("principal", params.principal);
  if (params.kind) qs.set("kind", params.kind);
  if (params.since) qs.set("since", params.since);
  if (params.until) qs.set("until", params.until);
  if (params.limit !== undefined) qs.set("limit", String(params.limit));

  return request<AuditEventsResponse>(`/audit/events?${qs.toString()}`, {
    skipCache: true,
  });
}
