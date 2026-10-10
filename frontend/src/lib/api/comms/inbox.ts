// Unified inbox client.
// Prefers Ghost's /api/comms/inbox endpoint when available; falls back to the
// existing provider-specific endpoints so the email tab keeps working before
// the unified endpoint lands.
import { request } from "$lib/api/base";
import { getEmails as getGmailEmails } from "$lib/api/gmail/gmail";
import type { Email } from "$lib/api/gmail/types";
import type {
  EmailAccount,
  EmailProvider,
  UnifiedEmail,
  UnifiedInboxParams,
  UnifiedInboxResponse,
} from "./types";

const COMMS_BASE = "/comms";

// Cache the result of probing the unified endpoint so we don't pay the 404 on
// every call. `null` = not probed yet, `true` = available, `false` = absent.
let unifiedInboxAvailable: boolean | null = null;

function normalizeGmail(e: Email): UnifiedEmail {
  return { ...e, provider: "gmail" };
}

async function fetchUnifiedInbox(
  params: UnifiedInboxParams,
): Promise<UnifiedInboxResponse | null> {
  const search = new URLSearchParams();
  if (params.providers?.length) search.set("providers", params.providers.join(","));
  if (params.folder) search.set("folder", params.folder);
  if (params.limit !== undefined) search.set("limit", String(params.limit));
  if (params.offset !== undefined) search.set("offset", String(params.offset));
  if (params.search) search.set("q", params.search);
  const qs = search.toString();
  try {
    return await request<UnifiedInboxResponse>(
      `${COMMS_BASE}/inbox${qs ? `?${qs}` : ""}`,
    );
  } catch (err) {
    if (err instanceof Error && /HTTP 404/.test(err.message)) {
      unifiedInboxAvailable = false;
      return null;
    }
    throw err;
  }
}

async function fetchGmailFallback(
  params: UnifiedInboxParams,
): Promise<UnifiedInboxResponse> {
  const emails = await getGmailEmails({
    folder: params.folder,
    limit: params.limit,
    offset: params.offset,
  });
  // Tool-handler may wrap the list as `{emails, count, ...}`; normalize either shape.
  const list = Array.isArray(emails)
    ? emails
    : ((emails as unknown as { emails?: Email[] })?.emails ?? []);
  const normalized = list.map(normalizeGmail);
  // Honor an explicit Outlook-only filter by returning empty — caller can decide
  // whether to surface a "Outlook not connected" empty state.
  if (params.providers && !params.providers.includes("gmail")) {
    return { emails: [], total: 0, has_more: false };
  }
  return { emails: normalized, total: normalized.length, has_more: false };
}

export async function getUnifiedInbox(
  params: UnifiedInboxParams = {},
): Promise<UnifiedInboxResponse> {
  if (unifiedInboxAvailable === false) {
    return fetchGmailFallback(params);
  }
  const unified = await fetchUnifiedInbox(params);
  if (unified) {
    unifiedInboxAvailable = true;
    return unified;
  }
  return fetchGmailFallback(params);
}

export async function getUnifiedEmail(
  provider: EmailProvider,
  id: string,
): Promise<UnifiedEmail> {
  if (provider === "gmail") {
    const email = await request<Email>(`/integrations/google_gmail/emails/${id}`);
    return normalizeGmail(email);
  }
  // Outlook path — Microsoft handler exposes /integrations/microsoft/mail/emails/:id.
  const email = await request<Email>(`/integrations/microsoft/mail/emails/${id}`);
  return { ...email, provider: "outlook" };
}

// Returns the set of email-capable accounts the user has connected. Falls back
// to a Gmail-only inference when Ghost's /comms/accounts endpoint isn't live.
export async function getConnectedAccounts(): Promise<EmailAccount[]> {
  try {
    return await request<EmailAccount[]>(`${COMMS_BASE}/accounts`);
  } catch (err) {
    if (!(err instanceof Error) || !/HTTP 404/.test(err.message)) throw err;
  }
  // Fallback: poke Gmail's status endpoint and synthesize an account row.
  try {
    const status = await request<{
      connected: boolean;
      email?: string;
      account_id?: string;
    }>(`/integrations/google_gmail/status`);
    if (!status.connected) return [];
    return [
      {
        provider: "gmail",
        email: status.email ?? status.account_id ?? "",
        account_id: status.account_id ?? "",
      },
    ];
  } catch {
    return [];
  }
}

export async function syncProvider(
  provider: EmailProvider,
): Promise<{ success: boolean }> {
  if (provider === "gmail") {
    return request<{ success: boolean }>(
      `/integrations/google_gmail/sync`,
      { method: "POST" },
    );
  }
  return request<{ success: boolean }>(
    `/integrations/microsoft/mail/sync`,
    { method: "POST" },
  );
}

// Resets the probe cache. Tests use this; production callers should not need it.
export function __resetUnifiedInboxProbe(): void {
  unifiedInboxAvailable = null;
}

// ---------- Send + drafts ----------

import { raw } from "$lib/api/base";

export interface SendEmailInput {
  provider: EmailProvider;
  to: string[];
  cc?: string[];
  bcc?: string[];
  subject: string;
  body: string;
  is_html?: boolean;
  reply_to?: string;
  attachments?: File[];
}

export interface SendEmailResponse {
  success: boolean;
  message_id?: string;
}

// Sends via multipart when attachments are present; JSON otherwise. Routes
// through the provider-specific endpoint until Ghost ships /api/comms/send.
export async function sendUnifiedEmail(
  input: SendEmailInput,
): Promise<SendEmailResponse> {
  const base =
    input.provider === "gmail"
      ? "/integrations/google_gmail"
      : "/integrations/microsoft/mail";

  if (!input.attachments?.length) {
    const res = await raw.post(`${base}/send`, {
      to: input.to,
      cc: input.cc,
      bcc: input.bcc,
      subject: input.subject,
      body: input.body,
      is_html: input.is_html ?? false,
      reply_to: input.reply_to,
    });
    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      throw new Error(
        data.error || data.detail || `Send failed (HTTP ${res.status})`,
      );
    }
    return res.json().catch(() => ({ success: true }));
  }

  // Multipart path. Backend endpoint convention:
  //   POST {base}/send-multipart  with form fields:
  //     payload: JSON string of {to, cc, bcc, subject, body, is_html, reply_to}
  //     files:   one or more File parts (field name "files[]")
  const form = new FormData();
  form.append(
    "payload",
    JSON.stringify({
      to: input.to,
      cc: input.cc,
      bcc: input.bcc,
      subject: input.subject,
      body: input.body,
      is_html: input.is_html ?? false,
      reply_to: input.reply_to,
    }),
  );
  for (const file of input.attachments) {
    form.append("files[]", file, file.name);
  }
  const res = await raw.postFormData(`${base}/send-multipart`, form);
  if (res.status === 404) {
    throw new Error(
      "Attachments aren't supported yet — the backend multipart endpoint is pending.",
    );
  }
  if (!res.ok) {
    const data = await res.json().catch(() => ({}));
    throw new Error(
      data.error || data.detail || `Send failed (HTTP ${res.status})`,
    );
  }
  return res.json().catch(() => ({ success: true }));
}

export interface DraftPayload {
  id?: string;
  provider: EmailProvider;
  to: string[];
  cc: string[];
  bcc: string[];
  subject: string;
  body: string;
  attachments_meta?: { filename: string; size: number; mime_type: string }[];
}

export interface SavedDraft {
  id: string;
}

let draftsAvailable: boolean | null = null;

// Tries Ghost's /api/integrations/{provider}/drafts endpoint. Returns null when
// the endpoint isn't live so the caller can keep the local-only fallback.
export async function saveDraftRemote(
  draft: DraftPayload,
): Promise<SavedDraft | null> {
  if (draftsAvailable === false) return null;
  const base =
    draft.provider === "gmail"
      ? "/integrations/google_gmail"
      : "/integrations/microsoft/mail";
  const url = draft.id ? `${base}/drafts/${draft.id}` : `${base}/drafts`;
  try {
    const res = await request<SavedDraft>(url, {
      method: draft.id ? "PUT" : "POST",
      body: draft,
    });
    draftsAvailable = true;
    return res;
  } catch (err) {
    if (err instanceof Error && /HTTP 404/.test(err.message)) {
      draftsAvailable = false;
      return null;
    }
    throw err;
  }
}

export function __resetDraftsProbe(): void {
  draftsAvailable = null;
}
