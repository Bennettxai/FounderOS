// Helpers shared across the comms/channels components.
// Pure functions only — no side effects, no Svelte stores. The orchestrator
// (+page.svelte) holds the state.
//
// Mirrors the pattern from commsEmailUtils.ts so cross-tab work stays consistent.
import type {
  ChannelProvider,
  CommsChannel,
  CommsMessage,
  CommsWorkspace,
} from "$lib/api/comms";

export type {
  ChannelProvider,
  CommsChannel,
  CommsMessage,
  CommsWorkspace,
};

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
const WEEKDAY_MONTH_DAY: Intl.DateTimeFormatOptions = {
  weekday: "short",
  month: "short",
  day: "numeric",
};

// Time-only format used inside the message stream: "10:42 AM".
export function formatMessageTime(iso: string | undefined | null): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleTimeString(undefined, HOUR_MINUTE);
}

// Sidebar last-activity badge: "2m ago", "3h ago", "Yesterday", "Mon", "May 1".
export function formatLastActivity(iso: string | undefined | null): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const diffMs = Date.now() - d.getTime();
  const minutes = Math.floor(diffMs / 60_000);
  if (minutes < 1) return "now";
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h`;
  const days = Math.floor(hours / 24);
  if (days === 1) return "Yesterday";
  if (days < 7) {
    return d.toLocaleDateString(undefined, { weekday: "short" });
  }
  if (d.getFullYear() === new Date().getFullYear()) {
    return d.toLocaleDateString(undefined, MONTH_DAY);
  }
  return d.toLocaleDateString(undefined, MONTH_DAY_YEAR);
}

// Date-divider label: "Today", "Yesterday", "Mon · May 12", "May 1, 2025".
export function formatDateDivider(iso: string | undefined | null): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const now = new Date();
  if (sameDay(d, now)) return "Today";
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (sameDay(d, yesterday)) return "Yesterday";
  const sixDaysAgo = new Date(now);
  sixDaysAgo.setDate(now.getDate() - 6);
  if (d >= sixDaysAgo) {
    return d.toLocaleDateString(undefined, WEEKDAY_MONTH_DAY);
  }
  if (d.getFullYear() === now.getFullYear()) {
    return d.toLocaleDateString(undefined, MONTH_DAY);
  }
  return d.toLocaleDateString(undefined, MONTH_DAY_YEAR);
}

// Verbose timestamp for hover tooltips: "Wed, May 1, 2026, 10:42 AM".
export function formatFullTimestamp(iso: string | undefined | null): string {
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

function sameDay(a: Date, b: Date): boolean {
  return (
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  );
}

// ---------- Provider / workspace labels ----------

export function providerLabel(provider: ChannelProvider): string {
  if (provider === "slack") return "Slack";
  if (provider === "whatsapp") return "WhatsApp";
  return "Microsoft Teams";
}

export function workspaceLabel(workspace: CommsWorkspace): string {
  return workspace.workspace_name || providerLabel(workspace.provider);
}

// Channel display name with the right hash/lock prefix decided at render time
// in components — this returns the bare name only.
export function channelLabel(channel: CommsChannel): string {
  return channel.name || "(unnamed)";
}

// DM display name: pull a participant from the channel's name if Slack stores
// it as "alice,bob"; otherwise return the raw name. Keeps the helper provider-
// agnostic — Teams may store DM names differently and the spec accepts either.
export function dmDisplayName(channel: CommsChannel): string {
  const raw = channel.name?.trim();
  if (!raw) return "(direct message)";
  // Slack convention: comma-separated user ids/names for group DMs.
  if (raw.includes(",")) {
    const parts = raw
      .split(",")
      .map((s) => s.trim())
      .filter(Boolean);
    if (parts.length <= 3) return parts.join(", ");
    return `${parts.slice(0, 3).join(", ")} +${parts.length - 3}`;
  }
  return raw;
}

// ---------- Partition channels by type ----------

export interface ChannelPartition {
  channels: CommsChannel[];
  dms: CommsChannel[];
}

export function partitionByType(channels: CommsChannel[]): ChannelPartition {
  const result: ChannelPartition = { channels: [], dms: [] };
  for (const c of channels) {
    if (c.is_archived) continue;
    if (c.is_dm) result.dms.push(c);
    else result.channels.push(c);
  }
  // Sort each section by last_message_at desc, then name asc as a tiebreaker.
  const byActivityDesc = (a: CommsChannel, b: CommsChannel): number => {
    const aT = a.last_message_at ? Date.parse(a.last_message_at) : 0;
    const bT = b.last_message_at ? Date.parse(b.last_message_at) : 0;
    if (aT !== bT) return bT - aT;
    return a.name.localeCompare(b.name);
  };
  result.channels.sort(byActivityDesc);
  result.dms.sort(byActivityDesc);
  return result;
}

// ---------- Filter channels by workspace scope ----------

export type WorkspaceScope = "all" | string; // "all" or a specific workspace_id

export function filterByWorkspace(
  channels: CommsChannel[],
  scope: WorkspaceScope,
): CommsChannel[] {
  if (scope === "all") return channels;
  return channels.filter((c) => c.workspace_id === scope);
}

// ---------- Sidebar search filter ----------

// Matches against channel name, topic, and description. Case-insensitive.
// DMs match against the display name.
export function filterChannelsByQuery(
  channels: CommsChannel[],
  query: string,
): CommsChannel[] {
  const q = query.trim().toLowerCase();
  if (!q) return channels;
  return channels.filter((c) => {
    const haystack = [c.name, c.topic, c.description, dmDisplayName(c)]
      .filter(Boolean)
      .map((s) => s!.toLowerCase());
    return haystack.some((s) => s.includes(q));
  });
}

// ---------- Sender grouping in the message stream ----------

// 5-minute window — consecutive messages from the same sender within this
// window collapse under a single avatar+sender header. Date change always
// breaks a group.
const GROUP_WINDOW_MS = 5 * 60 * 1000;

export interface MessageGroup {
  groupId: string;
  senderId?: string;
  senderName?: string;
  senderAvatar?: string;
  messages: CommsMessage[];
  // The first message's sent_at — used by the group header.
  startedAt: string;
}

// Groups consecutive same-sender messages within a 5-minute window. Thread
// replies (messages with thread_ts that points elsewhere) never group with
// non-thread messages because the thread drawer renders them differently;
// in the main stream we hide bare reply-only messages and surface them as
// a reply summary on the thread root (see ChannelMessages component).
export function groupBySender(messages: CommsMessage[]): MessageGroup[] {
  const groups: MessageGroup[] = [];
  for (const m of messages) {
    const last = groups[groups.length - 1];
    const lastMsg = last ? last.messages[last.messages.length - 1] : undefined;
    const sameSender =
      last !== undefined && last.senderId === m.sender_id && !!m.sender_id;
    const withinWindow =
      lastMsg !== undefined &&
      Math.abs(Date.parse(m.sent_at) - Date.parse(lastMsg.sent_at)) <
        GROUP_WINDOW_MS;
    const sameDate =
      lastMsg !== undefined &&
      sameDay(new Date(m.sent_at), new Date(lastMsg.sent_at));
    if (last && sameSender && withinWindow && sameDate) {
      last.messages.push(m);
      continue;
    }
    groups.push({
      groupId: m.id,
      senderId: m.sender_id,
      senderName: m.sender_name,
      senderAvatar: m.sender_avatar,
      messages: [m],
      startedAt: m.sent_at,
    });
  }
  return groups;
}

// ---------- Date dividers ----------

// Walks a sorted-ascending message list and inserts a date-divider sentinel
// before the first message of each new calendar day. Returns a flat array of
// (divider | group) so the renderer can iterate once.
export interface DateDividerItem {
  kind: "divider";
  id: string;
  label: string;
  iso: string;
}

export interface GroupItem {
  kind: "group";
  group: MessageGroup;
}

export type StreamItem = DateDividerItem | GroupItem;

export function buildStreamItems(messages: CommsMessage[]): StreamItem[] {
  // Filter out bare thread replies — they live in the drawer. Keep thread
  // roots (is_thread_root) and non-threaded messages.
  const inMain = messages.filter(
    (m) => !m.thread_ts || m.is_thread_root || m.thread_ts === m.external_id,
  );
  // Sort ascending by sent_at so the stream reads top-to-bottom = oldest-to-newest.
  const sorted = [...inMain].sort(
    (a, b) => Date.parse(a.sent_at) - Date.parse(b.sent_at),
  );
  const groups = groupBySender(sorted);
  const out: StreamItem[] = [];
  let lastDate: Date | undefined;
  for (const group of groups) {
    const first = group.messages[0];
    const d = new Date(first.sent_at);
    if (!lastDate || !sameDay(d, lastDate)) {
      out.push({
        kind: "divider",
        id: `div-${first.id}`,
        label: formatDateDivider(first.sent_at),
        iso: first.sent_at,
      });
      lastDate = d;
    }
    out.push({ kind: "group", group });
  }
  return out;
}

// ---------- Display helpers ----------

export function senderDisplayName(message: CommsMessage): string {
  return (
    message.sender_name?.trim() || message.sender_id || "Unknown sender"
  );
}

export function initials(name: string): string {
  const trimmed = name.trim();
  if (!trimmed) return "?";
  const parts = trimmed.split(/[\s,]+/).slice(0, 2);
  return parts.map((p) => p.charAt(0).toUpperCase()).join("") || "?";
}

// ---------- Workspace status helpers ----------

export function workspaceStatusColor(
  status: CommsWorkspace["status"] | undefined,
): string {
  if (status === "reauth_required" || status === "error") {
    return "var(--bos-status-error)";
  }
  if (status === "stale") return "var(--bos-status-warning)";
  return "var(--bos-status-success)";
}

// ---------- Misc ----------

export function isOwnMessage(
  message: CommsMessage,
  selfUserId: string | undefined,
): boolean {
  if (!selfUserId) return false;
  return message.sender_id === selfUserId;
}
