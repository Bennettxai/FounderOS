// Gmail API functions
// Uses tool-specific OAuth: /integrations/google_gmail/* (gmail scopes only)
import { apiClient } from "$lib/api";
import { withOAuthRedirect } from "$lib/utils/platform";
import type {
  Email,
  ComposeEmail,
  GmailAccessStatus,
  GmailStats,
  SyncResult,
  GetEmailsParams,
} from "./types";

// ============================================
// Gmail API - Uses tool-specific OAuth (gmail scopes only)
// Routes under /integrations/google_gmail/* for isolated OAuth
// ============================================

const GMAIL_BASE = "/integrations/google_gmail";

/**
 * Check if the user has Gmail access
 * Uses the tool-specific Gmail status endpoint
 */
export async function checkGmailAccess(): Promise<GmailAccessStatus> {
  const res = await apiClient.get(`${GMAIL_BASE}/status`);
  if (!res.ok) {
    // Not connected yet
    return { has_access: false, requires_upgrade: false };
  }
  const data = await res.json();
  return {
    has_access: data.connected,
    requires_upgrade: false, // Tool-specific OAuth always has correct scopes
    email: data.email || data.account_id,
  };
}

/**
 * Request Gmail access (initiates OAuth flow with Gmail scopes only)
 */
export async function requestGmailAccess(): Promise<{ auth_url: string }> {
  const res = await apiClient.get(withOAuthRedirect(`${GMAIL_BASE}/auth`));
  if (!res.ok) {
    throw new Error("Failed to request Gmail access");
  }
  return res.json();
}

/**
 * Get emails from the user's Gmail
 */
export async function getEmails(params?: GetEmailsParams): Promise<Email[]> {
  const searchParams = new URLSearchParams();
  if (params?.folder) searchParams.set("folder", params.folder);
  if (params?.limit) searchParams.set("limit", params.limit.toString());
  if (params?.offset) searchParams.set("offset", params.offset.toString());

  const res = await apiClient.get(
    `${GMAIL_BASE}/emails?${searchParams.toString()}`,
  );
  if (!res.ok) {
    const data = await res.json();
    if (data.requires_upgrade) {
      throw new Error("REQUIRES_UPGRADE");
    }
    throw new Error("Failed to get emails");
  }
  return res.json();
}

/**
 * Get a single email by ID
 */
export async function getEmail(id: string): Promise<Email> {
  const res = await apiClient.get(`${GMAIL_BASE}/emails/${id}`);
  if (!res.ok) {
    throw new Error("Failed to get email");
  }
  return res.json();
}

/**
 * Send an email
 */
export async function sendEmail(
  email: ComposeEmail,
): Promise<{ message: string }> {
  const res = await apiClient.post(`${GMAIL_BASE}/send`, email);
  if (!res.ok) {
    const data = await res.json();
    throw new Error(data.error || "Failed to send email");
  }
  return res.json();
}

/**
 * Mark an email as read
 */
export async function markAsRead(id: string): Promise<{ message: string }> {
  const res = await apiClient.post(`${GMAIL_BASE}/emails/${id}/read`);
  if (!res.ok) {
    throw new Error("Failed to mark email as read");
  }
  return res.json();
}

/**
 * Archive an email
 */
export async function archiveEmail(id: string): Promise<{ message: string }> {
  const res = await apiClient.post(`${GMAIL_BASE}/emails/${id}/archive`);
  if (!res.ok) {
    throw new Error("Failed to archive email");
  }
  return res.json();
}

/**
 * Delete an email (moves to trash)
 */
export async function deleteEmail(id: string): Promise<{ message: string }> {
  const res = await apiClient.delete(`${GMAIL_BASE}/emails/${id}`);
  if (!res.ok) {
    throw new Error("Failed to delete email");
  }
  return res.json();
}

/**
 * Sync emails from Gmail
 */
export async function syncEmails(
  maxResults?: number,
): Promise<{ message: string; result: SyncResult }> {
  const searchParams = new URLSearchParams();
  if (maxResults) searchParams.set("max", maxResults.toString());

  const res = await apiClient.post(
    `${GMAIL_BASE}/sync?${searchParams.toString()}`,
  );
  if (!res.ok) {
    const data = await res.json();
    if (data.requires_upgrade) {
      throw new Error("REQUIRES_UPGRADE");
    }
    throw new Error(data.error || "Failed to sync emails");
  }
  return res.json();
}

/**
 * Get Gmail statistics — total count, unread count, last sync time.
 */
export async function getGmailStats(): Promise<GmailStats> {
  const res = await apiClient.get(`${GMAIL_BASE}/stats`);
  if (!res.ok) {
    return {
      has_access: false,
      total_emails: 0,
      unread_count: 0,
      last_sync: null,
    };
  }
  return res.json();
}
