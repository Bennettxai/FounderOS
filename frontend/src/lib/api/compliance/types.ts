// ============================================================================
// Compliance API Types
// ============================================================================

// ---------------------------------------------------------------------------
// DSAR — Data Subject Access Request
// ---------------------------------------------------------------------------

export interface DSARExportRequest {
  principal_id: string;
  tenant_id: string;
}

export interface DSARExportCounts {
  [resource: string]: number;
}

export interface DSARExportResponse {
  principal_id: string;
  tenant_id: string;
  exported_at: string;
  counts: DSARExportCounts;
  data: Record<string, unknown>;
}

// ---------------------------------------------------------------------------
// Erasure
// ---------------------------------------------------------------------------

export interface ErasurePreviewRequest {
  principal_id: string;
  tenant_id: string;
  force?: boolean;
}

export interface ErasurePreviewResponse {
  principal_id: string;
  tenant_id: string;
  counts: DSARExportCounts;
  blocked_by_holds?: string[];
  can_proceed: boolean;
}

export interface ErasureRequest {
  principal_id: string;
  tenant_id: string;
  force?: boolean;
  confirm: true;
}

export interface ErasureResponse {
  principal_id: string;
  tenant_id: string;
  erased_at: string;
  counts: DSARExportCounts;
}

// ---------------------------------------------------------------------------
// Legal Holds
// ---------------------------------------------------------------------------

export interface LegalHold {
  id: string;
  tenant_id: string;
  signal_id: string;
  reason: string;
  placed_at: string;
  placed_by?: string;
  released_at?: string | null;
}

export interface PlaceLegalHoldRequest {
  tenant_id: string;
  signal_id: string;
  reason: string;
}

export interface ListLegalHoldsResponse {
  holds: LegalHold[];
  total: number;
}

// ---------------------------------------------------------------------------
// Retention Policies
// ---------------------------------------------------------------------------

export type RetentionScopeType = "node" | "genre" | "tenant" | "global";
export type RetentionAction = "archive" | "delete" | "redact";

export interface RetentionPolicy {
  id: string;
  tenant_id: string;
  scope_type: RetentionScopeType;
  scope_value: string;
  ttl_days: number;
  action: RetentionAction;
  created_at: string;
  updated_at?: string;
}

export interface CreateRetentionPolicyRequest {
  tenant_id: string;
  scope_type: RetentionScopeType;
  scope_value: string;
  ttl_days: number;
  action: RetentionAction;
}

export interface ListRetentionPoliciesResponse {
  policies: RetentionPolicy[];
  total: number;
}

export interface RetentionSweepRequest {
  tenant_id: string;
  dry_run: boolean;
}

export interface RetentionSweepResponse {
  dry_run: boolean;
  tenant_id: string;
  swept_at: string;
  counts: DSARExportCounts;
  affected: number;
}

// ---------------------------------------------------------------------------
// Audit Log
// ---------------------------------------------------------------------------

export type AuditEventKind =
  | "signal.created"
  | "signal.updated"
  | "signal.deleted"
  | "signal.read"
  | "principal.created"
  | "principal.updated"
  | "principal.deleted"
  | "hold.placed"
  | "hold.released"
  | "retention.policy.created"
  | "retention.policy.deleted"
  | "retention.sweep.ran"
  | "dsar.exported"
  | "erasure.previewed"
  | "erasure.erased"
  | "auth.login";

export const AUDIT_EVENT_KINDS: AuditEventKind[] = [
  "signal.created",
  "signal.updated",
  "signal.deleted",
  "signal.read",
  "principal.created",
  "principal.updated",
  "principal.deleted",
  "hold.placed",
  "hold.released",
  "retention.policy.created",
  "retention.policy.deleted",
  "retention.sweep.ran",
  "dsar.exported",
  "erasure.previewed",
  "erasure.erased",
  "auth.login",
];

export interface AuditEvent {
  id: string;
  tenant_id: string;
  principal: string;
  kind: AuditEventKind;
  occurred_at: string;
  metadata?: Record<string, unknown>;
}

export interface AuditQueryParams {
  tenant_id: string;
  principal?: string;
  kind?: AuditEventKind;
  since?: string;
  until?: string;
  limit?: number;
}

export interface AuditEventsResponse {
  events: AuditEvent[];
  total: number;
}

// ---------------------------------------------------------------------------
// Generic API error shape
// ---------------------------------------------------------------------------

export interface ComplianceApiError {
  error: string;
  code: string;
}
