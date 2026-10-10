// Gmail API Types

export interface EmailAddress {
  email: string;
  name?: string;
}

export interface EmailAttachment {
  id: string;
  filename: string;
  mime_type: string;
  size: number;
}

export interface Email {
  id: string;
  user_id: string;
  provider: string;
  external_id: string;
  thread_id?: string;
  subject: string;
  snippet: string;
  from_email: string;
  from_name?: string;
  to_emails: EmailAddress[];
  cc_emails?: EmailAddress[];
  bcc_emails?: EmailAddress[];
  reply_to?: string;
  body_text?: string;
  body_html?: string;
  attachments?: EmailAttachment[];
  is_read: boolean;
  is_starred: boolean;
  is_important: boolean;
  is_draft: boolean;
  is_sent: boolean;
  is_archived: boolean;
  is_trash: boolean;
  labels: string[];
  date: string;
  received_at?: string;
}

export interface ComposeEmail {
  to: string[];
  cc?: string[];
  bcc?: string[];
  subject: string;
  body: string;
  is_html?: boolean;
  reply_to?: string;
}

export type EmailFolder = 'inbox' | 'sent' | 'drafts' | 'starred' | 'archive' | 'trash';

export interface GmailAccessStatus {
  has_access: boolean;
  requires_upgrade: boolean;
  message?: string;
  email?: string;
}

export interface GmailStats {
  has_access: boolean;
  requires_upgrade?: boolean;
  total_emails?: number;
  unread_count?: number;
  last_sync?: string | null;
}

export interface SyncResult {
  total_records: number;
  synced_records: number;
  failed_records: number;
}

export interface GetEmailsParams {
  folder?: EmailFolder;
  limit?: number;
  offset?: number;
}
