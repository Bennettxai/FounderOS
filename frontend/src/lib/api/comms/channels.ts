// Unified channels client.
// Mirrors the inbox.ts pattern: prefer Ghost's /api/comms/channels endpoint
// when available; fall back to the existing provider-specific endpoints
// (Slack today; Teams once Ghost ships) so the channels tab keeps working
// before the unified endpoint lands.
//
// Backend mapping reference:
//   Slack channels → internal/integrations/slack/channels.go (Channel)
//   Slack messages → internal/integrations/slack/messages.go (Message)
//   Teams (Wave 2 backend, parallel tables) → internal/integrations/microsoft/teams_*.go (planned)
//   Generic shape → internal/database/schema.sql (channels, channel_messages)
import { request } from "$lib/api/base";
import {
  getSlackChannels,
  getSlackMessages,
  sendSlackMessage,
  syncSlackChannels,
  getSlackConnectionStatus,
  type SlackChannel,
  type SlackMessage,
} from "$lib/api/slack";

const COMMS_BASE = "/comms";

// ---------- Public types (frontend contract — see comms-channels-spec §1.3) ----------

export type ChannelProvider = "slack" | "teams" | "whatsapp";

export type ChannelType = "channel" | "dm";

export interface CommsWorkspace {
  provider: ChannelProvider;
  workspace_id: string;
  workspace_name: string;
  connected_at?: string;
  status?: "healthy" | "stale" | "reauth_required" | "error";
  connection_status?: "connected" | "disconnected";
  sync_status?: "history_synced" | "not_synced" | "syncing" | "error";
  routing_status?: "unassigned" | "configured";
  read_only?: boolean;
  chat_count?: number;
  message_count?: number;
  last_message_at?: string;
  routed_workspace_id?: string;
  routed_workspace_name?: string;
}

export type ChannelPresence = 'online' | 'away' | 'offline' | 'unknown';

export interface CommsChannel {
  id: string;
  provider: ChannelProvider;
  workspace_id: string;
  workspace_name: string;
  external_id: string;
  name: string;
  description?: string;
  topic?: string;
  is_private: boolean;
  is_archived: boolean;
  is_dm: boolean;
  member_count: number;
  unread_count: number;
  message_count?: number;
  last_message_at?: string;
  read_only?: boolean;
  routed_workspace_id?: string;
  routed_workspace_name?: string;
  routing_scope?: CommunicationRouteScope;
  // Presence is populated for 1:1 DMs only (group DMs and channels remain
  // 'unknown'). Backend wiring lands later — Wave 3 ships the UI.
  presence?: ChannelPresence;
}

export type CommunicationRouteScope = "account" | "conversation";

export interface CommunicationRoute {
  id: string;
  provider: "gmail" | "outlook" | ChannelProvider;
  scope: CommunicationRouteScope;
  external_id: string;
  workspace_id: string;
  workspace_name: string;
  enabled: boolean;
  updated_at: string;
}

export interface SaveCommunicationRouteInput {
  provider: CommunicationRoute["provider"];
  scope: CommunicationRouteScope;
  external_id?: string;
  workspace_id: string;
  backfill_limit?: number;
}

export interface CommsAttachment {
  id: string;
  name: string;
  size_bytes: number;
  mime_type: string;
  url?: string;
  thumbnail_url?: string;
}

export interface CommsReaction {
  emoji: string;
  count: number;
  reacted_by_me: boolean;
}

export interface CommsMessage {
  id: string;
  channel_id: string;
  external_id: string;
  sender_id?: string;
  sender_name?: string;
  sender_avatar?: string;
  content: string;
  content_html?: string;
  attachments: CommsAttachment[];
  reactions: CommsReaction[];
  mentions: string[];
  thread_ts?: string;
  parent_message_id?: string;
  reply_count: number;
  is_thread_root: boolean;
  is_edited: boolean;
  is_deleted: boolean;
  sent_at: string;
  edited_at?: string;
}

export interface ListChannelsParams {
  workspace_id?: string;
  type?: ChannelType | "all";
  providers?: ChannelProvider[];
}

export interface ListChannelsResponse {
  channels: CommsChannel[];
}

export interface ListMessagesParams {
  channel_id: string;
  provider?: ChannelProvider;
  limit?: number;
  before?: string;
}

export interface ListMessagesResponse {
  messages: CommsMessage[];
  has_more: boolean;
}

export interface SendMessageInput {
  channel_id: string;
  provider?: ChannelProvider;
  text: string;
  thread_ts?: string;
}

export interface SendMessageResponse {
  id?: string;
  success?: boolean;
}

// ---------- Probe caches ----------
// Matches inbox.ts pattern. `null` = unprobed, `true` = available, `false` = absent.

let unifiedChannelsAvailable: boolean | null = null;
let unifiedMessagesAvailable: boolean | null = null;
let unifiedSendAvailable: boolean | null = null;
let workspacesEndpointAvailable: boolean | null = null;

export function __resetChannelsProbes(): void {
  unifiedChannelsAvailable = null;
  unifiedMessagesAvailable = null;
  unifiedSendAvailable = null;
  workspacesEndpointAvailable = null;
}

// ---------- Slack → unified normalization ----------

function normalizeSlackChannel(c: SlackChannel): CommsChannel {
  return {
    id: c.id,
    provider: "slack",
    // Slack today is single-workspace per OAuth grant; the backend doesn't
    // surface the workspace id on the Channel response, so we synthesize one
    // until Ghost adds the field.
    workspace_id: "slack",
    workspace_name: "Slack",
    external_id: c.slack_id,
    name: c.name,
    topic: c.topic,
    description: c.purpose,
    is_private: c.is_private,
    is_archived: false,
    is_dm: c.is_dm,
    member_count: c.member_count,
    unread_count: c.unread_count ?? 0,
    last_message_at: c.last_activity,
  };
}

function normalizeSlackMessage(m: SlackMessage): CommsMessage {
  return {
    id: m.id,
    channel_id: m.channel_id,
    external_id: m.slack_ts,
    sender_id: m.sender_id,
    sender_name: m.sender_name,
    sender_avatar: undefined,
    content: m.content,
    content_html: undefined,
    attachments: [],
    // Slack handler does not yet expose the JSONB reactions column on the
    // Message JSON response (see comms-channels-audit §B.4). Empty array is
    // the honest answer until Ghost ships it; the UI hides the reactions row
    // when the array is empty.
    reactions: m.reactions
      ? m.reactions.map((r) => ({
          emoji: r.icon,
          count: r.count,
          reacted_by_me: false,
        }))
      : [],
    mentions: [],
    thread_ts: m.thread_ts,
    parent_message_id: undefined,
    reply_count: m.reply_count,
    is_thread_root: !!m.thread_ts && m.thread_ts === m.slack_ts,
    is_edited: m.is_edited,
    is_deleted: false,
    sent_at: m.sent_at,
    edited_at: undefined,
  };
}

// ---------- Workspaces ----------

export async function getWorkspaces(): Promise<CommsWorkspace[]> {
  if (workspacesEndpointAvailable !== false) {
    try {
      const res = await request<{ workspaces: CommsWorkspace[] }>(
        `${COMMS_BASE}/workspaces`,
      );
      workspacesEndpointAvailable = true;
      return res.workspaces ?? [];
    } catch (err) {
      if (err instanceof Error && /HTTP 404/.test(err.message)) {
        workspacesEndpointAvailable = false;
      } else {
        throw err;
      }
    }
  }
  // Fallback: probe Slack status, synthesize a single workspace row.
  try {
    const status = await getSlackConnectionStatus();
    if (!status.connected) return [];
    return [
      {
        provider: "slack",
        workspace_id: status.workspace_id ?? "slack",
        workspace_name: status.workspace_name ?? "Slack",
        connected_at: status.connected_at,
        status: "healthy",
      },
    ];
  } catch {
    return [];
  }
}

// ---------- Workspace routing ----------

export async function getCommunicationRoutes(): Promise<CommunicationRoute[]> {
  const response = await request<{ routes: CommunicationRoute[] }>(
    `${COMMS_BASE}/routes`,
  );
  return response.routes ?? [];
}

export async function saveCommunicationRoute(
  input: SaveCommunicationRouteInput,
): Promise<{ route: CommunicationRoute; indexed: number }> {
  return request<{ route: CommunicationRoute; indexed: number }>(
    `${COMMS_BASE}/routes`,
    { method: "PUT", body: input },
  );
}

export async function deleteCommunicationRoute(
  provider: CommunicationRoute["provider"],
  scope: CommunicationRouteScope,
  externalId?: string,
): Promise<void> {
  const query = new URLSearchParams({ provider, scope });
  if (externalId) query.set("external_id", externalId);
  await request<void>(`${COMMS_BASE}/routes?${query.toString()}`, {
    method: "DELETE",
  });
}

export async function syncCommunicationRoutes(
  provider: CommunicationRoute["provider"],
): Promise<{ success: boolean; indexed: number }> {
  return request<{ success: boolean; indexed: number }>(
    `${COMMS_BASE}/routes/sync`,
    { method: "POST", body: { provider } },
  );
}

// ---------- Channels list ----------

async function fetchUnifiedChannels(
  params: ListChannelsParams,
): Promise<ListChannelsResponse | null> {
  const search = new URLSearchParams();
  if (params.workspace_id) search.set("workspace_id", params.workspace_id);
  if (params.type && params.type !== "all") search.set("type", params.type);
  if (params.providers?.length) {
    search.set("providers", params.providers.join(","));
  }
  const qs = search.toString();
  try {
    return await request<ListChannelsResponse>(
      `${COMMS_BASE}/channels${qs ? `?${qs}` : ""}`,
    );
  } catch (err) {
    if (err instanceof Error && /HTTP 404/.test(err.message)) {
      unifiedChannelsAvailable = false;
      return null;
    }
    throw err;
  }
}

async function fetchSlackChannelsFallback(
  params: ListChannelsParams,
): Promise<ListChannelsResponse> {
  // Caller filtered to a non-Slack provider — return empty rather than ignore.
  if (params.providers && !params.providers.includes("slack")) {
    return { channels: [] };
  }
  const res = await getSlackChannels();
  let normalized = (res.channels ?? []).map(normalizeSlackChannel);
  if (params.type === "channel") {
    normalized = normalized.filter((c) => !c.is_dm);
  } else if (params.type === "dm") {
    normalized = normalized.filter((c) => c.is_dm);
  }
  return { channels: normalized };
}

export async function getUnifiedChannels(
  params: ListChannelsParams = {},
): Promise<ListChannelsResponse> {
  if (unifiedChannelsAvailable === false) {
    return fetchSlackChannelsFallback(params);
  }
  const unified = await fetchUnifiedChannels(params);
  if (unified) {
    unifiedChannelsAvailable = true;
    return unified;
  }
  return fetchSlackChannelsFallback(params);
}

// ---------- Messages list ----------

async function fetchUnifiedMessages(
  params: ListMessagesParams,
): Promise<ListMessagesResponse | null> {
  const search = new URLSearchParams();
  if (params.provider) search.set("provider", params.provider);
  if (params.limit !== undefined) search.set("limit", String(params.limit));
  if (params.before) search.set("before", params.before);
  const qs = search.toString();
  try {
    return await request<ListMessagesResponse>(
      `${COMMS_BASE}/channels/${params.channel_id}/messages${qs ? `?${qs}` : ""}`,
    );
  } catch (err) {
    if (err instanceof Error && /HTTP 404/.test(err.message)) {
      unifiedMessagesAvailable = false;
      return null;
    }
    throw err;
  }
}

async function fetchSlackMessagesFallback(
  params: ListMessagesParams,
): Promise<ListMessagesResponse> {
  if (params.provider && params.provider !== "slack") {
    return { messages: [], has_more: false };
  }
  const res = await getSlackMessages(params.channel_id, {
    limit: params.limit,
  });
  const normalized = (res.messages ?? []).map(normalizeSlackMessage);
  return { messages: normalized, has_more: false };
}

export async function getUnifiedMessages(
  params: ListMessagesParams,
): Promise<ListMessagesResponse> {
  if (unifiedMessagesAvailable === false) {
    return fetchSlackMessagesFallback(params);
  }
  const unified = await fetchUnifiedMessages(params);
  if (unified) {
    unifiedMessagesAvailable = true;
    return unified;
  }
  return fetchSlackMessagesFallback(params);
}

// ---------- Thread messages ----------
// Threads have no dedicated Slack endpoint; the unified backend will expose one
// (see comms-channels-spec §1.3). Fallback derives thread replies from the
// in-channel page by matching thread_ts.

export async function getThreadMessages(
  channelId: string,
  threadTs: string,
  provider?: ChannelProvider,
): Promise<ListMessagesResponse> {
  if (unifiedMessagesAvailable !== false) {
    try {
      const search = new URLSearchParams();
      if (provider) search.set("provider", provider);
      const qs = search.toString();
      const res = await request<ListMessagesResponse>(
        `${COMMS_BASE}/channels/${channelId}/threads/${encodeURIComponent(
          threadTs,
        )}/messages${qs ? `?${qs}` : ""}`,
      );
      return res;
    } catch (err) {
      if (!(err instanceof Error) || !/HTTP 404/.test(err.message)) {
        throw err;
      }
    }
  }
  // Slack fallback: pull a wide page of channel messages and filter by thread_ts.
  const res = await getSlackMessages(channelId, { limit: 200 });
  const normalized = (res.messages ?? [])
    .map(normalizeSlackMessage)
    .filter(
      (m) => m.thread_ts === threadTs || m.external_id === threadTs,
    )
    // Ascending order — root first, replies follow.
    .sort((a, b) => a.sent_at.localeCompare(b.sent_at));
  return { messages: normalized, has_more: false };
}

// ---------- Send ----------

export async function sendUnifiedMessage(
  input: SendMessageInput,
): Promise<SendMessageResponse> {
  if (unifiedSendAvailable !== false) {
    try {
      const res = await request<SendMessageResponse>(
        `${COMMS_BASE}/channels/${input.channel_id}/messages`,
        {
          method: "POST",
          body: { text: input.text, thread_ts: input.thread_ts },
        },
      );
      unifiedSendAvailable = true;
      return res;
    } catch (err) {
      if (err instanceof Error && /HTTP 404/.test(err.message)) {
        unifiedSendAvailable = false;
      } else {
        throw err;
      }
    }
  }
  // Slack fallback. Note the Slack handler binds `text` (slack/handler.go:226)
  // — sendSlackMessage already sends that field. Thread support depends on
  // whether the upstream client passes thread_ts; today it doesn't, so
  // thread replies in the Slack-only fallback land at the channel root and
  // we surface a console.warn so callers can detect it during dev.
  if (input.thread_ts) {
    // eslint-disable-next-line no-console
    console.warn(
      "channels: thread reply via Slack fallback drops thread_ts (waiting on unified endpoint)",
    );
  }
  const res = await sendSlackMessage(input.channel_id, input.text);
  return { success: res.success };
}

// ---------- Sync ----------

export async function syncWorkspace(
  workspace: CommsWorkspace,
): Promise<{ success: boolean }> {
  if (workspace.provider === "slack") {
    const res = await syncSlackChannels();
    return { success: res.synced_channels >= 0 };
  }
  if (workspace.provider === "whatsapp") {
    const result = await syncCommunicationRoutes("whatsapp");
    return { success: result.success };
  }
  // Teams sync — Ghost's Wave 2 endpoint (per spec); harmless 404 falls back
  // to a no-op until ready.
  try {
    return await request<{ success: boolean }>(
      `/integrations/microsoft/teams/sync`,
      { method: "POST" },
    );
  } catch (err) {
    if (err instanceof Error && /HTTP 404/.test(err.message)) {
      return { success: false };
    }
    throw err;
  }
}

// ---------- Reactions ----------
// The reactions JSONB column exists but Ghost hasn't exposed it on the wire.
// These helpers will become live once the unified endpoint lands; for now they
// throw a flagged error so callers can disable the UI when the contract isn't met.

export class UnifiedReactionsUnavailable extends Error {
  constructor() {
    super(
      "Unified reactions endpoint not available — Ghost wave 2 dependency",
    );
    this.name = "UnifiedReactionsUnavailable";
  }
}

export async function toggleReaction(
  channelId: string,
  messageId: string,
  emoji: string,
  add: boolean,
): Promise<{ reactions: CommsReaction[] }> {
  try {
    return await request<{ reactions: CommsReaction[] }>(
      `${COMMS_BASE}/channels/${channelId}/messages/${messageId}/reactions`,
      {
        method: add ? "POST" : "DELETE",
        body: { emoji },
      },
    );
  } catch (err) {
    if (err instanceof Error && /HTTP 404/.test(err.message)) {
      throw new UnifiedReactionsUnavailable();
    }
    throw err;
  }
}
