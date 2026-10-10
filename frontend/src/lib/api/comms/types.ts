// Communications module — unified types
// Spans Gmail + Outlook with a single normalized shape so the email tab can
// render either provider through the same components.
//
// Backend mapping reference:
//   Gmail emails  →  internal/integrations/google/gmail_types.go (Email)
//   Outlook mail  →  internal/integrations/microsoft/outlook_mail.go
//   Both store into normalized columns described in
//   internal/database/schema.sql (emails, microsoft_mail_messages).

import type {
  Email,
  EmailAddress,
  EmailAttachment,
  EmailFolder,
} from "$lib/api/gmail/types";

export type EmailProvider = "gmail" | "outlook";

// UnifiedEmail is the same row shape Gmail returns, but with `provider`
// narrowed to the EmailProvider union. Outlook normalizes to this same
// shape on the way out of the unified inbox endpoint.
export type UnifiedEmail = Omit<Email, "provider"> & {
  provider: EmailProvider;
};

// EmailFolder lives in $lib/api/gmail/types — re-exported here so consumers
// of $lib/api/comms get a self-contained type surface.
export type { EmailAddress, EmailAttachment, EmailFolder };

export interface EmailAccount {
  provider: EmailProvider;
  email: string;
  account_id: string;
  connected_at?: string;
  last_sync?: string | null;
  status?: "healthy" | "stale" | "reauth_required" | "error";
  unread_count?: number;
}

export interface UnifiedInboxParams {
  providers?: EmailProvider[];
  folder?: EmailFolder;
  limit?: number;
  offset?: number;
  search?: string;
}

export interface UnifiedInboxResponse {
  emails: UnifiedEmail[];
  total?: number;
  has_more?: boolean;
}

// A client-side grouping of UnifiedEmail rows that share a thread_id.
// Built by commsEmailUtils.groupByThread; never round-trips to the server.
export interface EmailThread {
  id: string;
  subject: string;
  snippet: string;
  latest_date: string;
  message_count: number;
  unread: boolean;
  starred: boolean;
  has_attachments: boolean;
  participants: EmailAddress[];
  provider: EmailProvider;
  messages: UnifiedEmail[];
}

export interface ContactSuggestion {
  email: string;
  name?: string;
  source: "crm" | "microsoft" | "hubspot" | "frequency";
}
