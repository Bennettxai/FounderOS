// Helpers shared across the comms/email components.
// Pure functions only — no side effects, no Svelte stores. The orchestrator
// (+page.svelte) holds the state.
import type {
  EmailFolder,
  EmailProvider,
  EmailThread,
  UnifiedEmail,
} from "$lib/api/comms";

export type { EmailFolder, EmailProvider, EmailThread, UnifiedEmail };

// ---------- Date formatting ----------

const HOUR_MINUTE: Intl.DateTimeFormatOptions = {
  hour: "numeric",
  minute: "2-digit",
};
const MONTH_DAY: Intl.DateTimeFormatOptions = {
  month: "short",
  day: "numeric",
};
const MONTH_DAY_YEAR: Intl.DateTimeFormatOptions = {
  month: "short",
  day: "numeric",
  year: "numeric",
};

// Compact format used in list rows: "10:42 AM" today, "May 1" this year, else "May 1, 2025".
export function formatRowDate(iso: string | undefined | null): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const now = new Date();
  if (d.toDateString() === now.toDateString()) {
    return d.toLocaleTimeString(undefined, HOUR_MINUTE);
  }
  if (d.getFullYear() === now.getFullYear()) {
    return d.toLocaleDateString(undefined, MONTH_DAY);
  }
  return d.toLocaleDateString(undefined, MONTH_DAY_YEAR);
}

// Verbose format for the preview pane header: "Wed, May 1, 2026, 10:42 AM".
export function formatPreviewDate(iso: string | undefined | null): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleString(undefined, {
    weekday: "short",
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

// Relative format for sync banners: "2 minutes ago", "3 hours ago", "yesterday".
export function formatRelative(iso: string | undefined | null): string {
  if (!iso) return "never";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "never";
  const diffMs = Date.now() - d.getTime();
  const minutes = Math.floor(diffMs / 60_000);
  if (minutes < 1) return "just now";
  if (minutes < 60) return `${minutes} minute${minutes === 1 ? "" : "s"} ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours} hour${hours === 1 ? "" : "s"} ago`;
  const days = Math.floor(hours / 24);
  if (days === 1) return "yesterday";
  if (days < 7) return `${days} days ago`;
  return d.toLocaleDateString(undefined, MONTH_DAY);
}

// ---------- Subject normalization ----------

// Collapses runs of "Re: Re: Fwd: Re:" prefixes into a single canonical prefix
// so list rows don't bloat. Returns the trimmed subject and the prefix that
// was stripped (caller decides whether to render it as a chip).
export function normalizeSubject(raw: string | undefined | null): {
  subject: string;
  prefix: "Re:" | "Fwd:" | null;
} {
  const s = (raw ?? "").trim();
  if (!s) return { subject: "(no subject)", prefix: null };
  const match = s.match(/^((?:re|fwd?):\s*)+/i);
  if (!match) return { subject: s, prefix: null };
  const stripped = s.slice(match[0].length).trim();
  const lower = match[0].toLowerCase();
  const prefix: "Re:" | "Fwd:" =
    lower.includes("fw") ? "Fwd:" : "Re:";
  return { subject: stripped || s, prefix };
}

// ---------- Threading ----------

// Groups messages sharing a thread_id; messages without a thread_id stand alone.
// Each thread's metadata reflects the most-recent message; participants are
// deduplicated by email.
export function groupByThread(emails: UnifiedEmail[]): EmailThread[] {
  const buckets = new Map<string, UnifiedEmail[]>();
  for (const email of emails) {
    const key = email.thread_id || email.id;
    const bucket = buckets.get(key);
    if (bucket) bucket.push(email);
    else buckets.set(key, [email]);
  }

  const threads: EmailThread[] = [];
  for (const [id, bucket] of buckets) {
    const sorted = [...bucket].sort(
      (a, b) => new Date(b.date).getTime() - new Date(a.date).getTime(),
    );
    const latest = sorted[0];
    const participants = dedupeParticipants(sorted);
    const { subject } = normalizeSubject(latest.subject);
    threads.push({
      id,
      subject,
      snippet: latest.snippet,
      latest_date: latest.date,
      message_count: sorted.length,
      unread: sorted.some((m) => !m.is_read),
      starred: sorted.some((m) => m.is_starred),
      has_attachments: sorted.some(
        (m) => (m.attachments?.length ?? 0) > 0,
      ),
      participants,
      provider: latest.provider,
      messages: sorted,
    });
  }

  threads.sort(
    (a, b) =>
      new Date(b.latest_date).getTime() - new Date(a.latest_date).getTime(),
  );
  return threads;
}

function dedupeParticipants(messages: UnifiedEmail[]) {
  const seen = new Map<string, { email: string; name?: string }>();
  for (const m of messages) {
    const key = m.from_email.toLowerCase();
    if (!seen.has(key)) {
      seen.set(key, { email: m.from_email, name: m.from_name });
    }
  }
  return Array.from(seen.values());
}

// ---------- Folder taxonomy ----------

export interface FolderMeta {
  id: EmailFolder;
  label: string;
}

export const FOLDERS: ReadonlyArray<FolderMeta> = [
  { id: "inbox", label: "Inbox" },
  { id: "sent", label: "Sent" },
  { id: "drafts", label: "Drafts" },
  { id: "starred", label: "Starred" },
  { id: "archive", label: "Archive" },
  { id: "trash", label: "Trash" },
];

// Folders supported per provider. Outlook lacks a first-class Starred folder —
// flagged messages live in Inbox with a flag, so the Starred filter is local-only.
const GMAIL_FOLDERS: ReadonlySet<EmailFolder> = new Set([
  "inbox",
  "sent",
  "drafts",
  "starred",
  "archive",
  "trash",
]);
const OUTLOOK_FOLDERS: ReadonlySet<EmailFolder> = new Set([
  "inbox",
  "sent",
  "drafts",
  "archive",
  "trash",
]);

export function foldersByProvider(provider: EmailProvider): EmailFolder[] {
  const set = provider === "gmail" ? GMAIL_FOLDERS : OUTLOOK_FOLDERS;
  return FOLDERS.filter((f) => set.has(f.id)).map((f) => f.id);
}

export function providerLabel(provider: EmailProvider): string {
  return provider === "gmail" ? "Gmail" : "Outlook";
}

// ---------- Empty-state copy ----------

export interface EmptyCopy {
  title: string;
  description: string;
  // When set, the empty state surface should render a primary action with this label.
  actionLabel?: "Sync now";
}

// Folder-specific copy for the "this folder has zero rows" state.
// Inbox uses an "inbox zero" cue, Trash a flat acknowledgement, etc.
export function emptyFolderCopy(folder: EmailFolder): EmptyCopy {
  switch (folder) {
    case "inbox":
      return {
        title: "Inbox zero",
        description: "Nothing new right now. Sync to check again.",
        actionLabel: "Sync now",
      };
    case "sent":
      return {
        title: "Nothing sent yet",
        description: "Messages you send from BusinessOS show up here.",
      };
    case "drafts":
      return {
        title: "No drafts",
        description: "Start composing — drafts auto-save as you type.",
      };
    case "starred":
      return {
        title: "No starred emails",
        description: "Star a thread to keep it close.",
      };
    case "archive":
      return {
        title: "Nothing archived",
        description: "Archived threads land here.",
      };
    case "trash":
      return {
        title: "Trash is empty",
        description: "Deleted threads stay here for 30 days.",
      };
  }
}

// ---------- Display helpers ----------

// "Sender Name", or "sender@example.com" if name missing, or "Unknown" if both.
export function displayName(email: UnifiedEmail): string {
  return email.from_name?.trim() || email.from_email || "Unknown";
}

export function initials(name: string): string {
  const trimmed = name.trim();
  if (!trimmed) return "?";
  const parts = trimmed.split(/\s+/).slice(0, 2);
  return parts.map((p) => p.charAt(0).toUpperCase()).join("") || "?";
}

// "to alice@…, bob@…" with truncation to 4 names + "(+ N more)".
export function summarizeRecipients(
  to: { email: string; name?: string }[] | undefined,
  cc?: { email: string; name?: string }[],
): string {
  const all = [...(to ?? []), ...(cc ?? [])];
  if (!all.length) return "";
  const display = (r: { email: string; name?: string }) => r.name || r.email;
  if (all.length <= 4) return all.map(display).join(", ");
  const first = all.slice(0, 4).map(display).join(", ");
  return `${first} (+ ${all.length - 4} more)`;
}

// ---------- Local contact frequency cache ----------

// Builds a frequency-weighted suggestion list from the user's currently loaded
// emails. Used as the autocomplete fallback before Ghost's
// /api/comms/contacts/search endpoint lands.
export function buildLocalContactIndex(
  emails: UnifiedEmail[],
): Map<string, { email: string; name?: string; count: number }> {
  const index = new Map<string, { email: string; name?: string; count: number }>();
  const bump = (email: string, name?: string) => {
    if (!email) return;
    const key = email.toLowerCase();
    const existing = index.get(key);
    if (existing) {
      existing.count += 1;
      if (!existing.name && name) existing.name = name;
    } else {
      index.set(key, { email, name, count: 1 });
    }
  };
  for (const m of emails) {
    bump(m.from_email, m.from_name);
    for (const r of m.to_emails ?? []) bump(r.email, r.name);
    for (const r of m.cc_emails ?? []) bump(r.email, r.name);
  }
  return index;
}

export function searchLocalContacts(
  index: Map<string, { email: string; name?: string; count: number }>,
  query: string,
  limit = 8,
): { email: string; name?: string }[] {
  const q = query.trim().toLowerCase();
  if (!q) return [];
  const matches = Array.from(index.values()).filter(
    (c) =>
      c.email.toLowerCase().includes(q) ||
      (c.name?.toLowerCase().includes(q) ?? false),
  );
  matches.sort((a, b) => b.count - a.count);
  return matches.slice(0, limit).map(({ email, name }) => ({ email, name }));
}

// ---------- Validation ----------

// Standards-honest enough for a recipient field. Not RFC-strict; matches what
// the backend SMTP layer accepts in practice.
const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function isValidEmail(value: string): boolean {
  return EMAIL_PATTERN.test(value.trim());
}
